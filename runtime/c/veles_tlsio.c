/*
 * Veles bootstrap runtime — the TLS engine for std/tls (D128).
 *
 * The engine never touches a socket. It is a state machine over three byte
 * buffers: the Veles side feeds it the bytes that arrived from the peer,
 * asks it to decrypt or encrypt, and sends the bytes it takes out. So the
 * same code runs over a net.Conn, over any io.Stream, and in tests over
 * nothing, and every call here is non-blocking and short — the waiting
 * happens in Veles, where a task can be parked.
 *
 * Backends: SChannel on Windows (the system's own TLS and trust store, no
 * library to ship); OpenSSL elsewhere, loaded with dlopen on first use so a
 * program that never speaks TLS neither links nor needs it, and no headers
 * are needed to build.
 *
 * Status codes of the calls that move data:
 *   0 done / data delivered   1 needs more bytes from the peer
 *   2 the peer closed the session cleanly (close_notify)   3 failed
 * A failure's text is read with veles_tls_error. The caller always drains
 * veles_tls_take after a call, whatever the status: a handshake step, a
 * decrypt and a key update can all leave bytes the peer is waiting for.
 *
 * Locks: a session has one; a call holds it for its own duration and never
 * calls into the collector while holding it (an allocation may stop the
 * world, and a thread waiting for the lock would never reach the stop).
 */
#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>

#if defined(_WIN32)
#define SECURITY_WIN32
#define SCHANNEL_USE_BLACKLISTS
#include <windows.h>
#include <wincrypt.h>
#include <security.h>
#include <schannel.h>
#include <ncrypt.h>
#include <sspi.h>
typedef CRITICAL_SECTION tls_lock;
#define LOCK_INIT(l) InitializeCriticalSection(l)
#define LOCK_FREE(l) DeleteCriticalSection(l)
#define LOCK(l) EnterCriticalSection(l)
#define UNLOCK(l) LeaveCriticalSection(l)
#else
#include <pthread.h>
#include <dlfcn.h>
typedef pthread_mutex_t tls_lock;
#define LOCK_INIT(l) pthread_mutex_init(l, NULL)
#define LOCK_FREE(l) pthread_mutex_destroy(l)
#define LOCK(l) pthread_mutex_lock(l)
#define UNLOCK(l) pthread_mutex_unlock(l)
#endif

typedef struct {
    const char *data;
    int64_t len;
} veles_string;

typedef struct {
    char *data;
    int64_t len;
} veles_bytes_view;

void *veles_alloc(int64_t size);

enum { T_OK = 0, T_NEED = 1, T_CLOSED = 2, T_FAIL = 3 };

/* ---------- byte buffers ---------- */

typedef struct {
    uint8_t *p;
    size_t len, cap;
} buf_t;

static bool buf_add(buf_t *b, const void *data, size_t n) {
    if (n == 0) return true;
    if (b->len + n > b->cap) {
        size_t cap = b->cap ? b->cap : 4096;
        while (cap < b->len + n) cap *= 2;
        uint8_t *p = realloc(b->p, cap);
        if (!p) return false;
        b->p = p;
        b->cap = cap;
    }
    memcpy(b->p + b->len, data, n);
    b->len += n;
    return true;
}

static void buf_drop(buf_t *b, size_t n) {
    if (n >= b->len) {
        b->len = 0;
        return;
    }
    memmove(b->p, b->p + n, b->len - n);
    b->len -= n;
}

/* ---------- sessions ---------- */

/* What a server shares between its connections: the certificate, its key and the
 * protocols it speaks. A session holds a reference, so a certificate that is
 * replaced stays alive for the connections already using it. */
typedef struct srvcreds {
    int refs; /* under table_lock */
#if defined(_WIN32)
    CredHandle cred;
    bool have_cred;
    PCCERT_CONTEXT cert;
    NCRYPT_KEY_HANDLE key;
    wchar_t key_name[64];
    uint8_t *alpn_wire; /* SEC_APPLICATION_PROTOCOLS, handed to every handshake call */
    size_t alpn_wire_len;
#else
    void *ctx;
    uint8_t alpn[512]; /* the wire form: length-prefixed names */
    size_t alpn_len;
#endif
} srvcreds;

typedef struct session {
    srvcreds *sc; /* a server's credentials (referenced), or NULL */
    tls_lock lock;
    bool server;
    bool established;
    bool closed; /* the peer's close_notify arrived */
    char err[256];
    char alpn[64];
    buf_t in;    /* ciphertext from the peer, not yet consumed */
    buf_t out;   /* ciphertext for the peer, not yet taken */
    buf_t plain; /* decrypted, not yet delivered */
#if defined(_WIN32)
    CredHandle cred;
    CtxtHandle ctx;
    bool have_cred, have_ctx;
    SecPkgContext_StreamSizes sizes;
    bool sizes_known;
    char *host;          /* UTF-8 */
    wchar_t *whost;
    HCERTSTORE roots;    /* custom trust anchors, or NULL for the system's */
    bool insecure;
    bool verified;
    uint8_t *alpn_wire;  /* SEC_APPLICATION_PROTOCOL_LIST, first call only */
    size_t alpn_wire_len;
#else
    void *ctx, *ssl, *rbio, *wbio;
#endif
} session;

/* Handle tables. A handle is a slot number plus the slot's generation, so one
 * that was freed can never reach the object that took its slot next. */
#define SLOT_BITS 20

typedef struct {
    void **items;
    uint32_t *gens;
    size_t len;
} htable;

static htable sessions, credtab;
static tls_lock table_lock;

#if defined(_WIN32)
static INIT_ONCE table_once = INIT_ONCE_STATIC_INIT;
static BOOL CALLBACK table_init(PINIT_ONCE o, PVOID p, PVOID *c) {
    (void)o; (void)p; (void)c;
    LOCK_INIT(&table_lock);
    return TRUE;
}
static void table_setup(void) { InitOnceExecuteOnce(&table_once, table_init, NULL, NULL); }
#else
static pthread_once_t table_once = PTHREAD_ONCE_INIT;
static void table_init(void) { LOCK_INIT(&table_lock); }
static void table_setup(void) { pthread_once(&table_once, table_init); }
#endif

static void failf(session *s, const char *msg) {
    snprintf(s->err, sizeof s->err, "%s", msg);
}

/* the table lock is held by all of these */
static int64_t ht_add(htable *t, void *item) {
    size_t slot = t->len;
    for (size_t i = 0; i < t->len; i++) {
        if (!t->items[i]) {
            slot = i;
            break;
        }
    }
    if (slot == t->len) {
        if (t->len + 16 > ((size_t)1 << SLOT_BITS) - 1) return 0;
        void **it = realloc(t->items, (t->len + 16) * sizeof *it);
        if (it) t->items = it;
        uint32_t *g = realloc(t->gens, (t->len + 16) * sizeof *g);
        if (g) t->gens = g;
        if (!it || !g) return 0;
        memset(t->items + t->len, 0, 16 * sizeof *it);
        memset(t->gens + t->len, 0, 16 * sizeof *g);
        t->len += 16;
    }
    t->items[slot] = item;
    return ((int64_t)t->gens[slot] << SLOT_BITS) | (int64_t)(slot + 1);
}

/* the item a handle names, or NULL when it is stale */
static void *ht_get(htable *t, int64_t h) {
    int64_t slot = (h & (((int64_t)1 << SLOT_BITS) - 1)) - 1;
    if (h <= 0 || slot < 0 || (size_t)slot >= t->len || !t->items[slot]) return NULL;
    return (uint32_t)(h >> SLOT_BITS) == t->gens[slot] ? t->items[slot] : NULL;
}

/* removes and returns the item a handle names */
static void *ht_take(htable *t, int64_t h) {
    void *item = ht_get(t, h);
    if (!item) return NULL;
    int64_t slot = (h & (((int64_t)1 << SLOT_BITS) - 1)) - 1;
    t->items[slot] = NULL;
    t->gens[slot]++;
    return item;
}

static int64_t table_add(session *s) {
    table_setup();
    LOCK(&table_lock);
    int64_t h = ht_add(&sessions, s);
    UNLOCK(&table_lock);
    return h;
}

static void backend_creds_free(srvcreds *c);

static void creds_unref(srvcreds *c) {
    table_setup();
    LOCK(&table_lock);
    int left = --c->refs;
    UNLOCK(&table_lock);
    if (left == 0) {
        backend_creds_free(c);
        free(c);
    }
}

/* the credentials of a handle with a reference taken, or NULL */
static srvcreds *creds_ref(int64_t h) {
    table_setup();
    LOCK(&table_lock);
    srvcreds *c = ht_get(&credtab, h);
    if (c) c->refs++;
    UNLOCK(&table_lock);
    return c;
}

/* the session of a handle, locked; NULL for a handle that was freed */
static session *session_lock(int64_t h) {
    table_setup();
    LOCK(&table_lock);
    session *s = ht_get(&sessions, h);
    if (s) LOCK(&s->lock);
    UNLOCK(&table_lock);
    return s;
}

static session *session_new(bool server) {
    session *s = calloc(1, sizeof *s);
    if (!s) return NULL;
    LOCK_INIT(&s->lock);
    s->server = server;
    return s;
}

static void set_out_string(veles_string *out, const void *data, size_t len) {
    char *buf = veles_alloc((int64_t)len + 1);
    if (len) memcpy(buf, data, len);
    buf[len] = 0;
    out->data = buf;
    out->len = (int64_t)len;
}

/* ---------- the backends ---------- */

static int backend_handshake(session *s);
static int backend_read(session *s, int64_t max, buf_t *got);
static void backend_write(session *s, const uint8_t *p, size_t n, bool *ok);
static void backend_shutdown(session *s);
static void backend_free(session *s);

#if defined(_WIN32)

/* ===== SChannel ===== */

static void win_fail(session *s, const char *what, SECURITY_STATUS st) {
    snprintf(s->err, sizeof s->err, "%s (0x%08lx)", what, (unsigned long)st);
}

typedef struct {
    char msg[256];
    NCryptBufferDesc *name; /* the stored key's name, or NULL for an ephemeral key */
} session_err;

static wchar_t *widen(const char *u8) {
    int n = MultiByteToWideChar(CP_UTF8, 0, u8, -1, NULL, 0);
    if (n <= 0) return NULL;
    wchar_t *w = malloc((size_t)n * sizeof *w);
    if (w) MultiByteToWideChar(CP_UTF8, 0, u8, -1, w, n);
    return w;
}

/* custom trust anchors from PEM text: every CERTIFICATE block */
static HCERTSTORE roots_from_pem(const char *pem, size_t len) {
    HCERTSTORE store = CertOpenStore(CERT_STORE_PROV_MEMORY, 0, 0, 0, NULL);
    if (!store) return NULL;
    const char *p = pem, *end = pem + len;
    int count = 0;
    while (p < end) {
        const char *b = strstr(p, "-----BEGIN CERTIFICATE-----");
        if (!b || b >= end) break;
        const char *e = strstr(b, "-----END CERTIFICATE-----");
        if (!e || e >= end) break;
        e += strlen("-----END CERTIFICATE-----");
        DWORD der_len = 0;
        if (CryptStringToBinaryA(b, (DWORD)(e - b), CRYPT_STRING_BASE64HEADER, NULL, &der_len, NULL, NULL)) {
            BYTE *der = malloc(der_len);
            if (der && CryptStringToBinaryA(b, (DWORD)(e - b), CRYPT_STRING_BASE64HEADER, der, &der_len, NULL, NULL)) {
                if (CertAddEncodedCertificateToStore(store, X509_ASN_ENCODING, der, der_len, CERT_STORE_ADD_ALWAYS, NULL)) count++;
            }
            free(der);
        }
        p = e;
    }
    if (count == 0) {
        CertCloseStore(store, 0);
        return NULL;
    }
    return store;
}

static bool win_verify_peer(session *s) {
    PCCERT_CONTEXT cert = NULL;
    SECURITY_STATUS st = QueryContextAttributes(&s->ctx, SECPKG_ATTR_REMOTE_CERT_CONTEXT, &cert);
    if (st != SEC_E_OK || !cert) {
        failf(s, "the server sent no certificate");
        return false;
    }
    bool ok = false;
    HCERTCHAINENGINE engine = NULL;
    PCCERT_CHAIN_CONTEXT chain = NULL;
    if (s->roots) {
        CERT_CHAIN_ENGINE_CONFIG cfg;
        memset(&cfg, 0, sizeof cfg);
        cfg.cbSize = sizeof cfg;
        cfg.hExclusiveRoot = s->roots;
        if (!CertCreateCertificateChainEngine(&cfg, &engine)) {
            failf(s, "cannot build the certificate chain engine");
            goto done;
        }
    }
    static LPSTR usages[] = {(LPSTR)szOID_PKIX_KP_SERVER_AUTH};
    CERT_CHAIN_PARA para;
    memset(&para, 0, sizeof para);
    para.cbSize = sizeof para;
    para.RequestedUsage.dwType = USAGE_MATCH_TYPE_OR;
    para.RequestedUsage.Usage.cUsageIdentifier = 1;
    para.RequestedUsage.Usage.rgpszUsageIdentifier = usages;
    if (!CertGetCertificateChain(engine, cert, NULL, cert->hCertStore, &para, 0, NULL, &chain)) {
        failf(s, "cannot build the certificate chain");
        goto done;
    }
    SSL_EXTRA_CERT_CHAIN_POLICY_PARA ex;
    memset(&ex, 0, sizeof ex);
    ex.cbSize = sizeof ex;
    ex.dwAuthType = AUTHTYPE_SERVER;
    ex.pwszServerName = s->whost;
    CERT_CHAIN_POLICY_PARA pp;
    memset(&pp, 0, sizeof pp);
    pp.cbSize = sizeof pp;
    pp.pvExtraPolicyPara = &ex;
    CERT_CHAIN_POLICY_STATUS status;
    memset(&status, 0, sizeof status);
    status.cbSize = sizeof status;
    if (!CertVerifyCertificateChainPolicy(CERT_CHAIN_POLICY_SSL, chain, &pp, &status)) {
        failf(s, "the certificate could not be checked");
        goto done;
    }
    if (status.dwError != 0) {
        const char *why;
        switch ((unsigned long)status.dwError) {
        case CERT_E_UNTRUSTEDROOT: why = "the certificate is not signed by a trusted authority"; break;
        case CERT_E_CN_NO_MATCH: why = "the certificate is not valid for this host name"; break;
        case CERT_E_EXPIRED: why = "the certificate has expired"; break;
        case CERT_E_REVOKED: why = "the certificate was revoked"; break;
        case CERT_E_UNTRUSTEDTESTROOT: why = "the certificate chains to a test root"; break;
        case CERT_E_CHAINING: why = "the certificate chain is incomplete"; break;
        case TRUST_E_CERT_SIGNATURE: why = "the certificate's signature does not verify"; break;
        default: why = "the certificate was rejected"; break;
        }
        snprintf(s->err, sizeof s->err, "certificate verification failed: %s (0x%08lx)", why, (unsigned long)status.dwError);
        goto done;
    }
    ok = true;
done:
    if (chain) CertFreeCertificateChain(chain);
    if (engine) CertFreeCertificateChainEngine(engine);
    CertFreeCertificateContext(cert);
    return ok;
}

static void win_query_alpn(session *s) {
    SecPkgContext_ApplicationProtocol ap;
    memset(&ap, 0, sizeof ap);
    if (QueryContextAttributes(&s->ctx, SECPKG_ATTR_APPLICATION_PROTOCOL, &ap) == SEC_E_OK &&
        ap.ProtoNegoStatus == SecApplicationProtocolNegotiationStatus_Success && ap.ProtocolIdSize < sizeof s->alpn) {
        memcpy(s->alpn, ap.ProtocolId, ap.ProtocolIdSize);
        s->alpn[ap.ProtocolIdSize] = 0;
    }
}

static CredHandle *credp(session *s) {
    return s->sc ? &s->sc->cred : &s->cred;
}

static int backend_handshake(session *s) {
    for (;;) {
        if ((s->have_ctx || s->server) && s->in.len == 0) return T_NEED;
        SecBuffer inb[3], outb[1];
        SecBufferDesc indesc, outdesc;
        int nin = 0;
        bool alpn_only = !s->server && !s->have_ctx && s->alpn_wire;
        if (alpn_only) {
            inb[nin].pvBuffer = s->alpn_wire;
            inb[nin].cbBuffer = (ULONG)s->alpn_wire_len;
            inb[nin].BufferType = SECBUFFER_APPLICATION_PROTOCOLS;
            nin++;
        } else {
            inb[nin].pvBuffer = s->in.p;
            inb[nin].cbBuffer = (ULONG)s->in.len;
            inb[nin].BufferType = SECBUFFER_TOKEN;
            nin++;
            inb[nin].pvBuffer = NULL;
            inb[nin].cbBuffer = 0;
            inb[nin].BufferType = SECBUFFER_EMPTY;
            nin++;
            if (s->server && s->sc->alpn_wire) {
                inb[nin].pvBuffer = s->sc->alpn_wire;
                inb[nin].cbBuffer = (ULONG)s->sc->alpn_wire_len;
                inb[nin].BufferType = SECBUFFER_APPLICATION_PROTOCOLS;
                nin++;
            }
        }
        indesc.ulVersion = SECBUFFER_VERSION;
        indesc.cBuffers = nin;
        indesc.pBuffers = inb;
        outb[0].pvBuffer = NULL;
        outb[0].cbBuffer = 0;
        outb[0].BufferType = SECBUFFER_TOKEN;
        outdesc.ulVersion = SECBUFFER_VERSION;
        outdesc.cBuffers = 1;
        outdesc.pBuffers = outb;
        ULONG flags = ISC_REQ_SEQUENCE_DETECT | ISC_REQ_REPLAY_DETECT | ISC_REQ_CONFIDENTIALITY |
                      ISC_REQ_ALLOCATE_MEMORY | ISC_REQ_STREAM | ISC_REQ_MANUAL_CRED_VALIDATION;
        ULONG got = 0;
        SECURITY_STATUS st;
        if (s->server) {
            flags = ASC_REQ_SEQUENCE_DETECT | ASC_REQ_REPLAY_DETECT | ASC_REQ_CONFIDENTIALITY | ASC_REQ_ALLOCATE_MEMORY |
                    ASC_REQ_STREAM;
            st = AcceptSecurityContext(credp(s), s->have_ctx ? &s->ctx : NULL, &indesc, flags, 0, &s->ctx, &outdesc, &got,
                                       NULL);
            if (st == SEC_I_CONTINUE_NEEDED || st == SEC_E_OK) s->have_ctx = true;
        } else if (!s->have_ctx) {
            st = InitializeSecurityContextA(credp(s), NULL, s->host, flags, 0, 0, s->alpn_wire ? &indesc : NULL, 0,
                                            &s->ctx, &outdesc, &got, NULL);
            if (st == SEC_I_CONTINUE_NEEDED || st == SEC_E_OK) s->have_ctx = true;
        } else {
            st = InitializeSecurityContextA(credp(s), &s->ctx, s->host, flags, 0, 0, &indesc, 0, NULL, &outdesc, &got,
                                            NULL);
        }
        if (outb[0].pvBuffer) {
            if (outb[0].cbBuffer) buf_add(&s->out, outb[0].pvBuffer, outb[0].cbBuffer);
            FreeContextBuffer(outb[0].pvBuffer);
        }
        if (st == SEC_E_INCOMPLETE_MESSAGE) return T_NEED;
        if (st == SEC_E_OK || st == SEC_I_CONTINUE_NEEDED) {
            if (alpn_only) {
                /* the first call carried only the ALPN list: nothing of the peer's was consumed */
                free(s->alpn_wire);
                s->alpn_wire = NULL;
                if (st == SEC_I_CONTINUE_NEEDED) return T_NEED;
            } else {
                size_t extra = (inb[1].BufferType == SECBUFFER_EXTRA) ? inb[1].cbBuffer : 0;
                buf_drop(&s->in, s->in.len - extra);
            }
            if (st == SEC_E_OK) {
                if (QueryContextAttributes(&s->ctx, SECPKG_ATTR_STREAM_SIZES, &s->sizes) != SEC_E_OK) {
                    failf(s, "cannot read the TLS record sizes");
                    return T_FAIL;
                }
                s->sizes_known = true;
                if (!s->verified) {
                    if (!s->server && !s->insecure && !win_verify_peer(s)) return T_FAIL;
                    s->verified = true;
                    win_query_alpn(s);
                }
                s->established = true;
                return T_OK;
            }
            continue;
        }
        if (st == SEC_I_INCOMPLETE_CREDENTIALS) {
            failf(s, "the server asked for a client certificate");
            return T_FAIL;
        }
        win_fail(s, "TLS handshake failed", st);
        return T_FAIL;
    }
}

static int backend_read(session *s, int64_t max, buf_t *got) {
    for (;;) {
        if (s->plain.len > 0) {
            size_t n = s->plain.len < (size_t)max ? s->plain.len : (size_t)max;
            buf_add(got, s->plain.p, n);
            buf_drop(&s->plain, n);
            return T_OK;
        }
        if (s->closed) return T_CLOSED;
        if (s->in.len == 0) return T_NEED;
        SecBuffer b[4];
        b[0].pvBuffer = s->in.p;
        b[0].cbBuffer = (ULONG)s->in.len;
        b[0].BufferType = SECBUFFER_DATA;
        for (int i = 1; i < 4; i++) {
            b[i].pvBuffer = NULL;
            b[i].cbBuffer = 0;
            b[i].BufferType = SECBUFFER_EMPTY;
        }
        SecBufferDesc desc = {SECBUFFER_VERSION, 4, b};
        SECURITY_STATUS st = DecryptMessage(&s->ctx, &desc, 0, NULL);
        if (st == SEC_E_INCOMPLETE_MESSAGE) return T_NEED;
        if (st == SEC_E_OK || st == SEC_I_CONTEXT_EXPIRED || st == SEC_I_RENEGOTIATE) {
            size_t extra = 0;
            for (int i = 1; i < 4; i++) {
                if (b[i].BufferType == SECBUFFER_DATA) buf_add(&s->plain, b[i].pvBuffer, b[i].cbBuffer);
                else if (b[i].BufferType == SECBUFFER_EXTRA) extra = b[i].cbBuffer;
            }
            buf_drop(&s->in, s->in.len - extra);
            if (st == SEC_I_CONTEXT_EXPIRED) {
                s->closed = true;
            } else if (st == SEC_I_RENEGOTIATE) {
                /* handshake data after the handshake: TLS 1.3 tickets and key updates */
                int r = backend_handshake(s);
                if (r == T_FAIL) return T_FAIL;
            }
            continue;
        }
        win_fail(s, "TLS decryption failed", st);
        return T_FAIL;
    }
}

static void backend_write(session *s, const uint8_t *p, size_t n, bool *ok) {
    *ok = true;
    while (n > 0) {
        size_t chunk = n < s->sizes.cbMaximumMessage ? n : s->sizes.cbMaximumMessage;
        size_t total = s->sizes.cbHeader + chunk + s->sizes.cbTrailer;
        uint8_t *rec = malloc(total);
        if (!rec) {
            failf(s, "out of memory");
            *ok = false;
            return;
        }
        memcpy(rec + s->sizes.cbHeader, p, chunk);
        SecBuffer b[4];
        b[0].pvBuffer = rec;
        b[0].cbBuffer = s->sizes.cbHeader;
        b[0].BufferType = SECBUFFER_STREAM_HEADER;
        b[1].pvBuffer = rec + s->sizes.cbHeader;
        b[1].cbBuffer = (ULONG)chunk;
        b[1].BufferType = SECBUFFER_DATA;
        b[2].pvBuffer = rec + s->sizes.cbHeader + chunk;
        b[2].cbBuffer = s->sizes.cbTrailer;
        b[2].BufferType = SECBUFFER_STREAM_TRAILER;
        b[3].pvBuffer = NULL;
        b[3].cbBuffer = 0;
        b[3].BufferType = SECBUFFER_EMPTY;
        SecBufferDesc desc = {SECBUFFER_VERSION, 4, b};
        SECURITY_STATUS st = EncryptMessage(&s->ctx, 0, &desc, 0);
        if (st != SEC_E_OK) {
            free(rec);
            win_fail(s, "TLS encryption failed", st);
            *ok = false;
            return;
        }
        buf_add(&s->out, rec, (size_t)b[0].cbBuffer + b[1].cbBuffer + b[2].cbBuffer);
        free(rec);
        p += chunk;
        n -= chunk;
    }
}

static void backend_shutdown(session *s) {
    if (!s->have_ctx || !s->established) return;
    DWORD token = SCHANNEL_SHUTDOWN;
    SecBuffer tb = {sizeof token, SECBUFFER_TOKEN, &token};
    SecBufferDesc td = {SECBUFFER_VERSION, 1, &tb};
    if (ApplyControlToken(&s->ctx, &td) != SEC_E_OK) return;
    SecBuffer outb = {0, SECBUFFER_TOKEN, NULL};
    SecBufferDesc od = {SECBUFFER_VERSION, 1, &outb};
    ULONG got = 0;
    if (s->server) {
        ULONG flags = ASC_REQ_SEQUENCE_DETECT | ASC_REQ_REPLAY_DETECT | ASC_REQ_CONFIDENTIALITY | ASC_REQ_ALLOCATE_MEMORY |
                      ASC_REQ_STREAM;
        AcceptSecurityContext(credp(s), &s->ctx, NULL, flags, 0, &s->ctx, &od, &got, NULL);
    } else {
        ULONG flags = ISC_REQ_SEQUENCE_DETECT | ISC_REQ_REPLAY_DETECT | ISC_REQ_CONFIDENTIALITY | ISC_REQ_ALLOCATE_MEMORY |
                      ISC_REQ_STREAM | ISC_REQ_MANUAL_CRED_VALIDATION;
        InitializeSecurityContextA(credp(s), &s->ctx, s->host, flags, 0, 0, NULL, 0, NULL, &od, &got, NULL);
    }
    if (outb.pvBuffer) {
        if (outb.cbBuffer) buf_add(&s->out, outb.pvBuffer, outb.cbBuffer);
        FreeContextBuffer(outb.pvBuffer);
    }
}

static void backend_free(session *s) {
    if (s->have_ctx) DeleteSecurityContext(&s->ctx);
    if (s->have_cred) FreeCredentialsHandle(&s->cred); /* a client's own */
    if (s->roots) CertCloseStore(s->roots, 0);
    free(s->host);
    free(s->whost);
    free(s->alpn_wire);
}

/* ---- reading a private key (PKCS#1, SEC1 and PKCS#8, PEM) ---- */

typedef struct {
    const uint8_t *p, *end;
} der_t;

/* one element: its tag, and its content as a der_t; false when malformed */
static bool der_next(der_t *d, uint8_t *tag, der_t *content) {
    if (d->p >= d->end) return false;
    *tag = *d->p++;
    if (d->p >= d->end) return false;
    size_t len = *d->p++;
    if (len & 0x80) {
        size_t n = len & 0x7f;
        if (n == 0 || n > 4 || (size_t)(d->end - d->p) < n) return false;
        len = 0;
        while (n--) len = (len << 8) | *d->p++;
    }
    if ((size_t)(d->end - d->p) < len) return false;
    content->p = d->p;
    content->end = d->p + len;
    d->p += len;
    return true;
}

static bool pem_block(const char *pem, size_t len, const char *label, uint8_t **der, DWORD *der_len) {
    char begin[64], endl[64];
    snprintf(begin, sizeof begin, "-----BEGIN %s-----", label);
    snprintf(endl, sizeof endl, "-----END %s-----", label);
    char *text = malloc(len + 1);
    if (!text) return false;
    memcpy(text, pem, len);
    text[len] = 0;
    char *b = strstr(text, begin);
    char *e = b ? strstr(b, endl) : NULL;
    bool ok = false;
    if (b && e) {
        e += strlen(endl);
        if (CryptStringToBinaryA(b, (DWORD)(e - b), CRYPT_STRING_BASE64HEADER, NULL, der_len, NULL, NULL)) {
            *der = malloc(*der_len);
            if (*der && CryptStringToBinaryA(b, (DWORD)(e - b), CRYPT_STRING_BASE64HEADER, *der, der_len, NULL, NULL)) ok = true;
            else {
                free(*der);
                *der = NULL;
            }
        }
    }
    free(text);
    return ok;
}

/* the curve of an EC key: its OID, as the DER content bytes */
static const uint8_t oid_p256[] = {0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07};
static const uint8_t oid_p384[] = {0x2b, 0x81, 0x04, 0x00, 0x22};
static const uint8_t oid_p521[] = {0x2b, 0x81, 0x04, 0x00, 0x23};
static const uint8_t oid_ec[] = {0x2a, 0x86, 0x48, 0xce, 0x3d, 0x02, 0x01};
static const uint8_t oid_rsa[] = {0x2a, 0x86, 0x48, 0x86, 0xf7, 0x0d, 0x01, 0x01, 0x01};

static bool oid_is(der_t d, const uint8_t *oid, size_t n) {
    return (size_t)(d.end - d.p) == n && memcmp(d.p, oid, n) == 0;
}

/* SEC1 ECPrivateKey -> an NCrypt key; curve_oid may be empty when the key carries its own */
static bool import_ec(session_err *e, der_t seq, der_t curve, NCRYPT_PROV_HANDLE prov, NCRYPT_KEY_HANDLE *key) {
    uint8_t tag;
    der_t d = seq, v;
    if (!der_next(&d, &tag, &v) || tag != 0x02) return false; /* version */
    der_t priv, pub = {0, 0};
    if (!der_next(&d, &tag, &priv) || tag != 0x04) return false;
    while (der_next(&d, &tag, &v)) {
        der_t inner = v, c;
        if (tag == 0xa0) {
            if (der_next(&inner, &tag, &c) && tag == 0x06) curve = c;
        } else if (tag == 0xa1) {
            if (der_next(&inner, &tag, &c) && tag == 0x03 && c.end - c.p > 1) {
                pub.p = c.p + 1; /* skip the unused-bits byte */
                pub.end = c.end;
            }
        }
    }
    ULONG magic, bytes;
    if (oid_is(curve, oid_p256, sizeof oid_p256)) { magic = 0x32534345; bytes = 32; }
    else if (oid_is(curve, oid_p384, sizeof oid_p384)) { magic = 0x34534345; bytes = 48; }
    else if (oid_is(curve, oid_p521, sizeof oid_p521)) { magic = 0x36534345; bytes = 66; }
    else {
        snprintf(e->msg, sizeof e->msg, "the key is on an elliptic curve SChannel does not support (P-256, P-384 and P-521 are)");
        return false;
    }
    size_t plen = (size_t)(priv.end - priv.p);
    if (plen > bytes || (size_t)(pub.end - pub.p) != 1 + 2 * (size_t)bytes || pub.p[0] != 4) {
        snprintf(e->msg, sizeof e->msg, "the EC key does not carry its public point (regenerate it with openssl ecparam -genkey)");
        return false;
    }
    size_t total = sizeof(BCRYPT_ECCKEY_BLOB) + 3 * (size_t)bytes;
    uint8_t *blob = calloc(1, total);
    if (!blob) return false;
    BCRYPT_ECCKEY_BLOB *hdr = (BCRYPT_ECCKEY_BLOB *)blob;
    hdr->dwMagic = magic;
    hdr->cbKey = bytes;
    memcpy(blob + sizeof *hdr, pub.p + 1, 2 * (size_t)bytes);
    memcpy(blob + sizeof *hdr + 2 * (size_t)bytes + (bytes - plen), priv.p, plen);
    SECURITY_STATUS st = NCryptImportKey(prov, 0, BCRYPT_ECCPRIVATE_BLOB, e->name, key, blob, (DWORD)total, 0);
    SecureZeroMemory(blob, total);
    free(blob);
    if (st != ERROR_SUCCESS) {
        snprintf(e->msg, sizeof e->msg, "cannot import the EC key (0x%08lx)", (unsigned long)st);
        return false;
    }
    return true;
}

static bool import_rsa(session_err *e, const uint8_t *der, DWORD len, NCRYPT_PROV_HANDLE prov, NCRYPT_KEY_HANDLE *key) {
    DWORD blob_len = 0;
    if (!CryptDecodeObjectEx(X509_ASN_ENCODING, CNG_RSA_PRIVATE_KEY_BLOB, der, len, 0, NULL, NULL, &blob_len) || !blob_len) {
        snprintf(e->msg, sizeof e->msg, "the RSA key is malformed");
        return false;
    }
    uint8_t *blob = malloc(blob_len);
    if (!blob) return false;
    if (!CryptDecodeObjectEx(X509_ASN_ENCODING, CNG_RSA_PRIVATE_KEY_BLOB, der, len, 0, NULL, blob, &blob_len)) {
        free(blob);
        snprintf(e->msg, sizeof e->msg, "the RSA key is malformed");
        return false;
    }
    SECURITY_STATUS st = NCryptImportKey(prov, 0, BCRYPT_RSAPRIVATE_BLOB, e->name, key, blob, blob_len, 0);
    SecureZeroMemory(blob, blob_len);
    free(blob);
    if (st != ERROR_SUCCESS) {
        snprintf(e->msg, sizeof e->msg, "cannot import the RSA key (0x%08lx)", (unsigned long)st);
        return false;
    }
    return true;
}

static bool load_key(session_err *e, const char *pem, size_t len, NCRYPT_KEY_HANDLE *key) {
    NCRYPT_PROV_HANDLE prov = 0;
    if (NCryptOpenStorageProvider(&prov, MS_KEY_STORAGE_PROVIDER, 0) != ERROR_SUCCESS) {
        snprintf(e->msg, sizeof e->msg, "cannot open the system's key storage");
        return false;
    }
    uint8_t *der = NULL;
    DWORD dlen = 0;
    bool ok = false;
    uint8_t tag;
    if (pem_block(pem, len, "RSA PRIVATE KEY", &der, &dlen)) {
        ok = import_rsa(e, der, dlen, prov, key);
    } else if (pem_block(pem, len, "EC PRIVATE KEY", &der, &dlen)) {
        der_t d = {der, der + dlen}, seq, none = {0, 0};
        ok = der_next(&d, &tag, &seq) && tag == 0x30 && import_ec(e, seq, none, prov, key);
        if (!ok && !e->msg[0]) snprintf(e->msg, sizeof e->msg, "the EC key is malformed");
    } else if (pem_block(pem, len, "PRIVATE KEY", &der, &dlen)) {
        der_t d = {der, der + dlen}, info, v;
        if (der_next(&d, &tag, &info) && tag == 0x30) {
            if (der_next(&info, &tag, &v) && tag == 0x02) { /* version */
                der_t alg, algid, params = {0, 0}, priv;
                if (der_next(&info, &tag, &alg) && tag == 0x30 && der_next(&info, &tag, &priv) && tag == 0x04 &&
                    der_next(&alg, &tag, &algid) && tag == 0x06) {
                    if (oid_is(algid, oid_rsa, sizeof oid_rsa)) {
                        ok = import_rsa(e, priv.p, (DWORD)(priv.end - priv.p), prov, key);
                    } else if (oid_is(algid, oid_ec, sizeof oid_ec)) {
                        if (der_next(&alg, &tag, &params) && tag != 0x06) params.p = params.end = NULL;
                        der_t inner = priv, seq;
                        ok = der_next(&inner, &tag, &seq) && tag == 0x30 && import_ec(e, seq, params, prov, key);
                        if (!ok && !e->msg[0]) snprintf(e->msg, sizeof e->msg, "the EC key is malformed");
                    } else {
                        snprintf(e->msg, sizeof e->msg, "the key's algorithm is not supported (RSA and ECDSA P-256/384/521 are)");
                    }
                }
            }
        }
        if (!ok && !e->msg[0]) snprintf(e->msg, sizeof e->msg, "the private key is malformed");
    } else {
        snprintf(e->msg, sizeof e->msg, "the key file holds no private key in PEM form (an encrypted key is not supported)");
    }
    if (der) {
        SecureZeroMemory(der, dlen);
        free(der);
    }
    NCryptFreeObject(prov);
    return ok;
}

/* A key stored for SChannel outlives a process that crashed. The first server
 * credentials of a process delete the ones left by processes that are gone. */
static void sweep_stale_keys(void) {
    static volatile LONG swept;
    if (InterlockedExchange(&swept, 1)) return;
    NCRYPT_PROV_HANDLE prov = 0;
    if (NCryptOpenStorageProvider(&prov, MS_KEY_STORAGE_PROVIDER, 0) != ERROR_SUCCESS) return;
    wchar_t stale[32][64];
    int count = 0;
    NCryptKeyName *kn = NULL;
    PVOID state = NULL;
    while (count < 32 && NCryptEnumKeys(prov, NULL, &kn, &state, 0) == ERROR_SUCCESS) {
        unsigned long pid = 0;
        if (wcsncmp(kn->pszName, L"veles-tls-", 10) == 0 && swscanf(kn->pszName + 10, L"%lu-", &pid) == 1 &&
            pid != GetCurrentProcessId() && wcslen(kn->pszName) < 64) {
            HANDLE p = OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, FALSE, pid);
            if (p) {
                CloseHandle(p);
            } else if (GetLastError() == ERROR_INVALID_PARAMETER) { /* no such process */
                wcscpy(stale[count++], kn->pszName);
            }
        }
        NCryptFreeBuffer(kn);
    }
    if (state) NCryptFreeBuffer(state);
    for (int i = 0; i < count; i++) {
        NCRYPT_KEY_HANDLE k;
        if (NCryptOpenKey(prov, &k, stale[i], 0, 0) == ERROR_SUCCESS) NCryptDeleteKey(k, 0);
    }
    NCryptFreeObject(prov);
}

/* a signature made with the private key must verify with the certificate's public key */
static bool key_matches(srvcreds *c) {
    BCRYPT_KEY_HANDLE pub = NULL;
    if (!CryptImportPublicKeyInfoEx2(X509_ASN_ENCODING, &c->cert->pCertInfo->SubjectPublicKeyInfo, 0, NULL, &pub)) return false;
    bool rsa = c->cert->pCertInfo->SubjectPublicKeyInfo.Algorithm.pszObjId &&
               strcmp(c->cert->pCertInfo->SubjectPublicKeyInfo.Algorithm.pszObjId, szOID_RSA_RSA) == 0;
    BCRYPT_PKCS1_PADDING_INFO pad = {BCRYPT_SHA256_ALGORITHM};
    uint8_t hash[32];
    memset(hash, 0x5a, sizeof hash);
    DWORD flags = rsa ? BCRYPT_PAD_PKCS1 : 0;
    DWORD siglen = 0;
    bool ok = false;
    if (NCryptSignHash(c->key, rsa ? &pad : NULL, hash, sizeof hash, NULL, 0, &siglen, flags) == ERROR_SUCCESS) {
        uint8_t *sig = malloc(siglen);
        if (sig && NCryptSignHash(c->key, rsa ? &pad : NULL, hash, sizeof hash, sig, siglen, &siglen, flags) == ERROR_SUCCESS) {
            ok = BCryptVerifySignature(pub, rsa ? &pad : NULL, hash, sizeof hash, sig, siglen, flags) == 0;
        }
        free(sig);
    }
    BCryptDestroyKey(pub);
    return ok;
}

static bool backend_server_creds(srvcreds *c, const char *cert, size_t cert_len, const char *key, size_t key_len,
                                 const char *alpn, size_t alpn_len, char *err, size_t errlen) {
    session_err e;
    e.msg[0] = 0;
    sweep_stale_keys();
    /* SChannel finds the key through the certificate's provider info, so it is stored under a
     * name of its own — and deleted again when the credentials are freed */
    static volatile LONG counter;
    swprintf(c->key_name, 64, L"veles-tls-%lu-%ld", (unsigned long)GetCurrentProcessId(), (long)InterlockedIncrement(&counter));
    NCryptBuffer nb = {(ULONG)((wcslen(c->key_name) + 1) * sizeof(wchar_t)), NCRYPTBUFFER_PKCS_KEY_NAME, c->key_name};
    NCryptBufferDesc nd = {NCRYPTBUFFER_VERSION, 1, &nb};
    e.name = &nd;
    uint8_t *der = NULL;
    DWORD dlen = 0;
    if (!pem_block(cert, cert_len, "CERTIFICATE", &der, &dlen)) {
        snprintf(err, errlen, "the certificate file holds no certificate in PEM form");
        return false;
    }
    c->cert = CertCreateCertificateContext(X509_ASN_ENCODING, der, dlen);
    free(der);
    if (!c->cert) {
        snprintf(err, errlen, "the certificate is malformed");
        return false;
    }
    if (!load_key(&e, key, key_len, &c->key)) {
        snprintf(err, errlen, "%s", e.msg);
        return false;
    }
    if (!key_matches(c)) {
        snprintf(err, errlen, "the key does not match the certificate");
        return false;
    }
    CRYPT_KEY_PROV_INFO kpi;
    memset(&kpi, 0, sizeof kpi);
    kpi.pwszContainerName = c->key_name;
    kpi.pwszProvName = (LPWSTR)MS_KEY_STORAGE_PROVIDER;
    if (!CertSetCertificateContextProperty(c->cert, CERT_KEY_PROV_INFO_PROP_ID, 0, &kpi)) {
        snprintf(err, errlen, "cannot attach the key to the certificate");
        return false;
    }
    if (alpn_len > 0) {
        size_t wire = 0;
        for (size_t i = 0, start = 0; i <= alpn_len; i++) {
            if (i == alpn_len || alpn[i] == ',') {
                wire += 1 + (i - start);
                start = i + 1;
            }
        }
        c->alpn_wire_len = 10 + wire;
        c->alpn_wire = calloc(1, c->alpn_wire_len);
        if (!c->alpn_wire) return false;
        uint32_t *w = (uint32_t *)c->alpn_wire;
        w[0] = (uint32_t)(6 + wire);
        w[1] = 2; /* SecApplicationProtocolNegotiationExt_ALPN */
        *(uint16_t *)(c->alpn_wire + 8) = (uint16_t)wire;
        uint8_t *q = c->alpn_wire + 10;
        for (size_t i = 0, start = 0; i <= alpn_len; i++) {
            if (i == alpn_len || alpn[i] == ',') {
                *q++ = (uint8_t)(i - start);
                memcpy(q, alpn + start, i - start);
                q += i - start;
                start = i + 1;
            }
        }
    }
    SCH_CREDENTIALS cred;
    memset(&cred, 0, sizeof cred);
    cred.dwVersion = SCH_CREDENTIALS_VERSION;
    cred.cCreds = 1;
    cred.paCred = &c->cert;
    cred.dwFlags = SCH_CRED_NO_SYSTEM_MAPPER | SCH_USE_STRONG_CRYPTO;
    TLS_PARAMETERS params;
    memset(&params, 0, sizeof params);
    params.grbitDisabledProtocols = SP_PROT_SSL3 | SP_PROT_TLS1_0 | SP_PROT_TLS1_1;
    cred.cTlsParameters = 1;
    cred.pTlsParameters = &params;
    SECURITY_STATUS st = AcquireCredentialsHandleA(NULL, (LPSTR)UNISP_NAME_A, SECPKG_CRED_INBOUND, NULL, &cred, NULL, NULL,
                                                   &c->cred, NULL);
    if (st != SEC_E_OK) {
        snprintf(err, errlen, "cannot start TLS with this certificate (0x%08lx)", (unsigned long)st);
        return false;
    }
    c->have_cred = true;
    return true;
}

static void backend_creds_free(srvcreds *c) {
    if (c->have_cred) FreeCredentialsHandle(&c->cred);
    if (c->cert) CertFreeCertificateContext(c->cert);
    if (c->key) NCryptDeleteKey(c->key, 0); /* the stored key, and the handle */
    free(c->alpn_wire);
}

static bool backend_server_session(session *s, srvcreds *c) {
    (void)c;
    s->verified = true;
    return true;
}

/* a TLS client: SChannel with TLS 1.2 and 1.3 enabled (1.3 where the system has it) */
static bool backend_client(session *s, const char *host, const char *roots, size_t roots_len, const char *alpn,
                           size_t alpn_len) {
    s->host = _strdup(host);
    s->whost = widen(host);
    if (!s->host || !s->whost) {
        failf(s, "out of memory");
        return false;
    }
    if (roots_len > 0) {
        s->roots = roots_from_pem(roots, roots_len);
        if (!s->roots) {
            failf(s, "the root certificates hold no certificate in PEM form");
            return false;
        }
    }
    if (alpn_len > 0) {
        /* SEC_APPLICATION_PROTOCOL_LIST: ext type, list length, then 1-byte-length-prefixed names */
        size_t wire = 0;
        for (size_t i = 0, start = 0; i <= alpn_len; i++) {
            if (i == alpn_len || alpn[i] == ',') {
                wire += 1 + (i - start);
                start = i + 1;
            }
        }
        size_t total = sizeof(uint32_t) + sizeof(uint16_t) + wire; /* inclusion + list size + list */
        size_t full = sizeof(uint32_t) + total;
        uint8_t *a = calloc(1, full);
        if (!a) {
            failf(s, "out of memory");
            return false;
        }
        uint32_t *w = (uint32_t *)a;
        w[0] = (uint32_t)(sizeof(uint32_t) + sizeof(uint16_t) + wire); /* ProtocolListsSize */
        w[1] = 2;                                                     /* SecApplicationProtocolNegotiationExt_ALPN */
        *(uint16_t *)(a + 8) = (uint16_t)wire;
        uint8_t *q = a + 10;
        for (size_t i = 0, start = 0; i <= alpn_len; i++) {
            if (i == alpn_len || alpn[i] == ',') {
                *q++ = (uint8_t)(i - start);
                memcpy(q, alpn + start, i - start);
                q += i - start;
                start = i + 1;
            }
        }
        s->alpn_wire = a;
        s->alpn_wire_len = full;
    }
    SCH_CREDENTIALS cred;
    memset(&cred, 0, sizeof cred);
    cred.dwVersion = SCH_CREDENTIALS_VERSION;
    cred.dwFlags = SCH_CRED_NO_DEFAULT_CREDS | SCH_CRED_MANUAL_CRED_VALIDATION | SCH_USE_STRONG_CRYPTO;
    /* TLS 1.2 is the minimum: SSL 3 and TLS 1.0/1.1 are off whatever the system's defaults */
    TLS_PARAMETERS params;
    memset(&params, 0, sizeof params);
    params.grbitDisabledProtocols = SP_PROT_SSL3 | SP_PROT_TLS1_0 | SP_PROT_TLS1_1;
    cred.cTlsParameters = 1;
    cred.pTlsParameters = &params;
    SECURITY_STATUS st = AcquireCredentialsHandleA(NULL, (LPSTR)UNISP_NAME_A, SECPKG_CRED_OUTBOUND, NULL, &cred, NULL,
                                                   NULL, &s->cred, NULL);
    if (st != SEC_E_OK) {
        win_fail(s, "cannot start TLS", st);
        return false;
    }
    s->have_cred = true;
    return true;
}

#else

/* ===== OpenSSL, loaded at run time ===== */

#define SSL_ERROR_SSL 1
#define SSL_ERROR_WANT_READ 2
#define SSL_ERROR_WANT_WRITE 3
#define SSL_ERROR_SYSCALL 5
#define SSL_ERROR_ZERO_RETURN 6

static struct {
    bool tried, ok;
    void *(*TLS_client_method)(void);
    void *(*TLS_server_method)(void);
    void *(*SSL_CTX_new)(void *);
    void (*SSL_CTX_free)(void *);
    void *(*SSL_new)(void *);
    void (*SSL_free)(void *);
    void *(*BIO_new)(void *);
    void *(*BIO_s_mem)(void);
    int (*BIO_read)(void *, void *, int);
    int (*BIO_write)(void *, const void *, int);
    long (*BIO_ctrl)(void *, int, long, void *);
    void *(*BIO_new_mem_buf)(const void *, int);
    void (*BIO_free_all)(void *);
    void (*SSL_set_bio)(void *, void *, void *);
    void (*SSL_set_connect_state)(void *);
    void (*SSL_set_accept_state)(void *);
    int (*SSL_do_handshake)(void *);
    int (*SSL_read)(void *, void *, int);
    int (*SSL_write)(void *, const void *, int);
    int (*SSL_get_error)(const void *, int);
    int (*SSL_shutdown)(void *);
    long (*SSL_ctrl)(void *, int, long, void *);
    long (*SSL_CTX_ctrl)(void *, int, long, void *);
    void (*SSL_CTX_set_verify)(void *, int, void *);
    int (*SSL_CTX_set_default_verify_paths)(void *);
    int (*SSL_set1_host)(void *, const char *);
    void *(*SSL_get0_param)(void *);
    int (*X509_VERIFY_PARAM_set1_ip_asc)(void *, const char *);
    int (*SSL_CTX_set_alpn_protos)(void *, const unsigned char *, unsigned);
    void (*SSL_get0_alpn_selected)(const void *, const unsigned char **, unsigned *);
    void *(*SSL_CTX_get_cert_store)(void *);
    int (*X509_STORE_add_cert)(void *, void *);
    void *(*PEM_read_bio_X509)(void *, void **, void *, void *);
    void (*X509_free)(void *);
    long (*SSL_get_verify_result)(const void *);
    const char *(*X509_verify_cert_error_string)(long);
    unsigned long (*ERR_get_error)(void);
    void (*ERR_error_string_n)(unsigned long, char *, size_t);
    void (*ERR_clear_error)(void);
    int (*SSL_CTX_use_certificate)(void *, void *);
    int (*SSL_CTX_use_PrivateKey)(void *, void *);
    int (*SSL_CTX_check_private_key)(const void *);
    void *(*PEM_read_bio_PrivateKey)(void *, void **, void *, void *);
    void (*EVP_PKEY_free)(void *);
    void (*SSL_CTX_set_alpn_select_cb)(void *, int (*)(void *, const unsigned char **, unsigned char *, const unsigned char *, unsigned, void *), void *);
    int (*SSL_select_next_proto)(unsigned char **, unsigned char *, const unsigned char *, unsigned, const unsigned char *, unsigned);
    void *(*SSL_CTX_get0_param)(void *);
} ossl;

static void *sym(void *a, void *b, const char *name) {
    void *p = a ? dlsym(a, name) : NULL;
    if (!p && b) p = dlsym(b, name);
    return p;
}

static bool ossl_load(void) {
    if (ossl.tried) return ossl.ok;
    ossl.tried = true;
    static const char *ssl_names[] = {"libssl.so.3", "libssl.so.1.1", "libssl.so", "libssl.3.dylib",
                                      "/opt/homebrew/lib/libssl.3.dylib", "/usr/local/lib/libssl.3.dylib",
                                      "libssl.dylib", NULL};
    static const char *crypto_names[] = {"libcrypto.so.3", "libcrypto.so.1.1", "libcrypto.so", "libcrypto.3.dylib",
                                         "/opt/homebrew/lib/libcrypto.3.dylib", "/usr/local/lib/libcrypto.3.dylib",
                                         "libcrypto.dylib", NULL};
    void *ssl = NULL, *crypto = NULL;
    for (int i = 0; ssl_names[i] && !ssl; i++) ssl = dlopen(ssl_names[i], RTLD_NOW | RTLD_GLOBAL);
    for (int i = 0; crypto_names[i] && !crypto; i++) crypto = dlopen(crypto_names[i], RTLD_NOW | RTLD_GLOBAL);
    if (!ssl) return false;
#define L(name) *(void **)&ossl.name = sym(ssl, crypto, #name); if (!ossl.name) return false
    L(TLS_client_method); L(TLS_server_method); L(SSL_CTX_new); L(SSL_CTX_free); L(SSL_new); L(SSL_free);
    L(BIO_new); L(BIO_s_mem); L(BIO_read); L(BIO_write); L(BIO_ctrl); L(BIO_new_mem_buf); L(BIO_free_all);
    L(SSL_set_bio); L(SSL_set_connect_state); L(SSL_set_accept_state); L(SSL_do_handshake); L(SSL_read); L(SSL_write);
    L(SSL_get_error); L(SSL_shutdown); L(SSL_ctrl); L(SSL_CTX_ctrl); L(SSL_CTX_set_verify);
    L(SSL_CTX_set_default_verify_paths); L(SSL_set1_host); L(SSL_get0_param); L(X509_VERIFY_PARAM_set1_ip_asc);
    L(SSL_CTX_set_alpn_protos); L(SSL_get0_alpn_selected); L(SSL_CTX_get_cert_store); L(X509_STORE_add_cert);
    L(PEM_read_bio_X509); L(X509_free); L(SSL_get_verify_result); L(X509_verify_cert_error_string);
    L(ERR_get_error); L(ERR_error_string_n); L(ERR_clear_error); L(SSL_CTX_use_certificate);
    L(SSL_CTX_use_PrivateKey); L(SSL_CTX_check_private_key); L(PEM_read_bio_PrivateKey); L(EVP_PKEY_free);
    L(SSL_CTX_set_alpn_select_cb); L(SSL_select_next_proto);
#undef L
    ossl.ok = true;
    return true;
}

static void ossl_text(char *dst, size_t n, const char *what) {
    unsigned long e = ossl.ERR_get_error();
    char text[160] = "";
    if (e) ossl.ERR_error_string_n(e, text, sizeof text);
    ossl.ERR_clear_error();
    snprintf(dst, n, "%s%s%s", what, text[0] ? ": " : "", text);
}

static void ossl_error(session *s, const char *what) {
    unsigned long e = ossl.ERR_get_error();
    char text[160] = "";
    if (e) ossl.ERR_error_string_n(e, text, sizeof text);
    ossl.ERR_clear_error();
    if (s->ssl && !s->server) {
        long v = ossl.SSL_get_verify_result(s->ssl);
        if (v != 0) {
            snprintf(s->err, sizeof s->err, "certificate verification failed: %s", ossl.X509_verify_cert_error_string(v));
            return;
        }
    }
    snprintf(s->err, sizeof s->err, "%s%s%s", what, text[0] ? ": " : "", text);
}

/* moves what OpenSSL queued for the peer into s->out */
static void ossl_drain(session *s) {
    uint8_t tmp[4096];
    while (ossl.BIO_ctrl(s->wbio, 10 /* BIO_CTRL_PENDING */, 0, NULL) > 0) {
        int n = ossl.BIO_read(s->wbio, tmp, sizeof tmp);
        if (n <= 0) break;
        buf_add(&s->out, tmp, (size_t)n);
    }
}

/* hands the received bytes to OpenSSL's read side */
static void ossl_feed(session *s) {
    if (s->in.len > 0) {
        int n = ossl.BIO_write(s->rbio, s->in.p, (int)s->in.len);
        if (n > 0) buf_drop(&s->in, (size_t)n);
    }
}

static int backend_handshake(session *s) {
    ossl_feed(s);
    int rc = ossl.SSL_do_handshake(s->ssl);
    ossl_drain(s);
    if (rc == 1) {
        s->established = true;
        const unsigned char *a = NULL;
        unsigned alen = 0;
        ossl.SSL_get0_alpn_selected(s->ssl, &a, &alen);
        if (a && alen < sizeof s->alpn) {
            memcpy(s->alpn, a, alen);
            s->alpn[alen] = 0;
        }
        return T_OK;
    }
    int err = ossl.SSL_get_error(s->ssl, rc);
    if (err == SSL_ERROR_WANT_READ || err == SSL_ERROR_WANT_WRITE) return T_NEED;
    ossl_error(s, "TLS handshake failed");
    return T_FAIL;
}

static int backend_read(session *s, int64_t max, buf_t *got) {
    ossl_feed(s);
    if (max > 65536) max = 65536;
    uint8_t *tmp = malloc((size_t)max);
    if (!tmp) {
        failf(s, "out of memory");
        return T_FAIL;
    }
    int rc = ossl.SSL_read(s->ssl, tmp, (int)max);
    ossl_drain(s);
    if (rc > 0) {
        buf_add(got, tmp, (size_t)rc);
        free(tmp);
        return T_OK;
    }
    free(tmp);
    int err = ossl.SSL_get_error(s->ssl, rc);
    if (err == SSL_ERROR_WANT_READ || err == SSL_ERROR_WANT_WRITE) return T_NEED;
    if (err == SSL_ERROR_ZERO_RETURN) {
        s->closed = true;
        return T_CLOSED;
    }
    ossl_error(s, "TLS read failed");
    return T_FAIL;
}

static void backend_write(session *s, const uint8_t *p, size_t n, bool *ok) {
    *ok = true;
    while (n > 0) {
        int chunk = n > (1 << 20) ? (1 << 20) : (int)n;
        int rc = ossl.SSL_write(s->ssl, p, chunk);
        ossl_drain(s);
        if (rc <= 0) {
            ossl_error(s, "TLS write failed");
            *ok = false;
            return;
        }
        p += rc;
        n -= (size_t)rc;
    }
}

static void backend_shutdown(session *s) {
    if (!s->established) return;
    ossl.SSL_shutdown(s->ssl);
    ossl_drain(s);
}

static void backend_free(session *s) {
    if (s->ssl) ossl.SSL_free(s->ssl); /* frees both BIOs */
    if (s->ctx) ossl.SSL_CTX_free(s->ctx); /* a client's own; a server's belongs to its credentials */
}

static bool is_ip_literal(const char *h) {
    bool colon = false, digits_dots = true;
    for (const char *c = h; *c; c++) {
        if (*c == ':') colon = true;
        else if (!((*c >= '0' && *c <= '9') || *c == '.')) digits_dots = false;
    }
    return colon || (digits_dots && strchr(h, '.'));
}

static bool backend_context(session *s, bool server) {
    if (!ossl_load()) {
        failf(s, "TLS needs OpenSSL (libssl.so.3), which is not installed");
        return false;
    }
    s->ctx = ossl.SSL_CTX_new(server ? ossl.TLS_server_method() : ossl.TLS_client_method());
    if (!s->ctx) {
        ossl_error(s, "cannot create the TLS context");
        return false;
    }
    ossl.SSL_CTX_ctrl(s->ctx, 123 /* SSL_CTRL_SET_MIN_PROTO_VERSION */, 0x0303 /* TLS 1.2 */, NULL);
    return true;
}

static bool backend_session(session *s) {
    s->ssl = ossl.SSL_new(s->ctx);
    s->rbio = ossl.BIO_new(ossl.BIO_s_mem());
    s->wbio = ossl.BIO_new(ossl.BIO_s_mem());
    if (!s->ssl || !s->rbio || !s->wbio) {
        ossl_error(s, "cannot create the TLS session");
        return false;
    }
    ossl.SSL_set_bio(s->ssl, s->rbio, s->wbio);
    return true;
}

static bool alpn_wire(const char *csv, size_t len, uint8_t *out, size_t *outlen) {
    size_t q = 0;
    for (size_t i = 0, start = 0; i <= len; i++) {
        if (i == len || csv[i] == ',') {
            size_t n = i - start;
            if (n == 0 || n > 255 || q + 1 + n > 512) return false;
            out[q++] = (uint8_t)n;
            memcpy(out + q, csv + start, n);
            q += n;
            start = i + 1;
        }
    }
    *outlen = q;
    return true;
}

static bool backend_client(session *s, const char *host, int insecure, const char *roots, size_t roots_len,
                           const char *alpn, size_t alpn_len) {
    if (!backend_context(s, false)) return false;
    if (insecure) {
        ossl.SSL_CTX_set_verify(s->ctx, 0, NULL);
    } else {
        ossl.SSL_CTX_set_verify(s->ctx, 1 /* SSL_VERIFY_PEER */, NULL);
        if (roots_len > 0) {
            void *store = ossl.SSL_CTX_get_cert_store(s->ctx);
            void *bio = ossl.BIO_new_mem_buf(roots, (int)roots_len);
            int count = 0;
            while (bio) {
                void *x = ossl.PEM_read_bio_X509(bio, NULL, NULL, NULL);
                if (!x) break;
                ossl.X509_STORE_add_cert(store, x);
                ossl.X509_free(x);
                count++;
            }
            if (bio) ossl.BIO_free_all(bio);
            ossl.ERR_clear_error();
            if (count == 0) {
                failf(s, "the root certificates hold no certificate in PEM form");
                return false;
            }
        } else if (!ossl.SSL_CTX_set_default_verify_paths(s->ctx)) {
            failf(s, "cannot load the system's trusted certificates");
            return false;
        }
    }
    if (alpn_len > 0) {
        uint8_t wire[512];
        size_t wl = 0;
        if (!alpn_wire(alpn, alpn_len, wire, &wl)) {
            failf(s, "invalid ALPN protocol list");
            return false;
        }
        ossl.SSL_CTX_set_alpn_protos(s->ctx, wire, (unsigned)wl);
    }
    if (!backend_session(s)) return false;
    if (!is_ip_literal(host)) ossl.SSL_ctrl(s->ssl, 55 /* SET_TLSEXT_HOSTNAME */, 0, (void *)host);
    if (!insecure) {
        if (is_ip_literal(host)) ossl.X509_VERIFY_PARAM_set1_ip_asc(ossl.SSL_get0_param(s->ssl), host);
        else ossl.SSL_set1_host(s->ssl, host);
    }
    ossl.SSL_set_connect_state(s->ssl);
    return true;
}

static int alpn_select(void *ssl, const unsigned char **out, unsigned char *outlen, const unsigned char *in,
                       unsigned inlen, void *arg) {
    (void)ssl;
    srvcreds *c = arg;
    unsigned char *sel = NULL;
    unsigned char sl = 0;
    if (ossl.SSL_select_next_proto(&sel, &sl, c->alpn, (unsigned)c->alpn_len, in, inlen) == 1 /* NEGOTIATED */) {
        *out = sel;
        *outlen = sl;
        return 0;
    }
    return 3; /* SSL_TLSEXT_ERR_NOACK: no common protocol; the connection goes on without one */
}

static bool backend_server_creds(srvcreds *c, const char *cert, size_t cert_len, const char *key, size_t key_len,
                                 const char *alpn, size_t alpn_len, char *err, size_t errlen) {
    if (!ossl_load()) {
        snprintf(err, errlen, "TLS needs OpenSSL (libssl.so.3), which is not installed");
        return false;
    }
    c->ctx = ossl.SSL_CTX_new(ossl.TLS_server_method());
    if (!c->ctx) {
        ossl_text(err, errlen, "cannot create the TLS context");
        return false;
    }
    ossl.SSL_CTX_ctrl(c->ctx, 123 /* SSL_CTRL_SET_MIN_PROTO_VERSION */, 0x0303 /* TLS 1.2 */, NULL);
    void *bio = ossl.BIO_new_mem_buf(cert, (int)cert_len);
    void *x = bio ? ossl.PEM_read_bio_X509(bio, NULL, NULL, NULL) : NULL;
    if (!x || ossl.SSL_CTX_use_certificate(c->ctx, x) != 1) {
        if (x) ossl.X509_free(x);
        if (bio) ossl.BIO_free_all(bio);
        ossl_text(err, errlen, "the certificate file holds no certificate in PEM form");
        return false;
    }
    ossl.X509_free(x);
    for (;;) { /* the rest is the chain the client needs */
        x = ossl.PEM_read_bio_X509(bio, NULL, NULL, NULL);
        if (!x) break;
        if (ossl.SSL_CTX_ctrl(c->ctx, 14 /* SSL_CTRL_EXTRA_CHAIN_CERT */, 0, x) != 1) ossl.X509_free(x);
    }
    ossl.BIO_free_all(bio);
    ossl.ERR_clear_error();
    bio = ossl.BIO_new_mem_buf(key, (int)key_len);
    void *pkey = bio ? ossl.PEM_read_bio_PrivateKey(bio, NULL, NULL, NULL) : NULL;
    if (bio) ossl.BIO_free_all(bio);
    if (!pkey) {
        ossl_text(err, errlen, "the key holds no private key in PEM form (an encrypted key is not supported)");
        return false;
    }
    int used = ossl.SSL_CTX_use_PrivateKey(c->ctx, pkey);
    ossl.EVP_PKEY_free(pkey);
    if (used != 1 || ossl.SSL_CTX_check_private_key(c->ctx) != 1) {
        ossl_text(err, errlen, "the key does not match the certificate");
        return false;
    }
    if (alpn_len > 0) {
        if (!alpn_wire(alpn, alpn_len, c->alpn, &c->alpn_len)) {
            snprintf(err, errlen, "invalid ALPN protocol list");
            return false;
        }
        ossl.SSL_CTX_set_alpn_select_cb(c->ctx, alpn_select, c);
    }
    return true;
}

static void backend_creds_free(srvcreds *c) {
    if (c->ctx) ossl.SSL_CTX_free(c->ctx);
}

static bool backend_server_session(session *s, srvcreds *c) {
    s->ctx = NULL;
    s->ssl = ossl.SSL_new(c->ctx);
    s->rbio = ossl.BIO_new(ossl.BIO_s_mem());
    s->wbio = ossl.BIO_new(ossl.BIO_s_mem());
    if (!s->ssl || !s->rbio || !s->wbio) {
        ossl_error(s, "cannot create the TLS session");
        return false;
    }
    ossl.SSL_set_bio(s->ssl, s->rbio, s->wbio);
    ossl.SSL_set_accept_state(s->ssl);
    return true;
}

#endif /* backends */

/* ---------- the calls of std/tls ---------- */

static char *cstr(const char *p, int64_t n) {
    char *c = malloc((size_t)n + 1);
    if (!c) return NULL;
    memcpy(c, p, (size_t)n);
    c[n] = 0;
    return c;
}

static void free_session(session *s) {
    backend_free(s);
    if (s->sc) creds_unref(s->sc);
    free(s->in.p);
    free(s->out.p);
    free(s->plain.p);
    LOCK_FREE(&s->lock);
    free(s);
}

/* 0 with a handle, or 3 with the reason in *err; alpn is a comma-separated list */
int64_t veles_tls_client(const char *host, int64_t hlen, const char *roots, int64_t rlen, const char *alpn,
                         int64_t alen, int64_t insecure, int64_t *handle, veles_string *err) {
    session *s = session_new(false);
    char *h = cstr(host, hlen);
    if (!s || !h) {
        free(h);
        if (s) free_session(s);
        set_out_string(err, "out of memory", 13);
        return T_FAIL;
    }
    bool ok;
#if defined(_WIN32)
    s->insecure = insecure != 0;
    ok = backend_client(s, h, roots, (size_t)rlen, alpn, (size_t)alen);
#else
    ok = backend_client(s, h, (int)insecure, roots, (size_t)rlen, alpn, (size_t)alen);
#endif
    free(h);
    if (!ok) {
        char msg[256];
        snprintf(msg, sizeof msg, "%s", s->err);
        free_session(s);
        set_out_string(err, msg, strlen(msg));
        return T_FAIL;
    }
    *handle = table_add(s);
    if (!*handle) {
        free_session(s);
        set_out_string(err, "out of memory", 13);
        return T_FAIL;
    }
    return T_OK;
}

/* ciphertext from the peer */
void veles_tls_feed(int64_t h, veles_bytes_view *bytes) {
    session *s = session_lock(h);
    if (!s) return;
    buf_add(&s->in, bytes->data, (size_t)bytes->len);
    UNLOCK(&s->lock);
}

/* ciphertext the peer is waiting for; empty when there is none */
void veles_tls_take(int64_t h, veles_string *out) {
    session *s = session_lock(h);
    if (!s) {
        set_out_string(out, "", 0);
        return;
    }
    size_t n = s->out.len;
    uint8_t *copy = n ? malloc(n) : NULL;
    if (copy) {
        memcpy(copy, s->out.p, n);
        s->out.len = 0;
    } else {
        n = 0;
    }
    UNLOCK(&s->lock);
    set_out_string(out, copy, n);
    free(copy);
}

/* 1 when ciphertext waits to be taken */
int64_t veles_tls_pending(int64_t h) {
    session *s = session_lock(h);
    if (!s) return 0;
    int64_t n = s->out.len > 0;
    UNLOCK(&s->lock);
    return n;
}

int64_t veles_tls_handshake(int64_t h) {
    session *s = session_lock(h);
    if (!s) return T_FAIL;
    int r = s->established ? T_OK : backend_handshake(s);
    UNLOCK(&s->lock);
    return r;
}

/* plaintext, at most max bytes: 0 with data in out, 1 needs input, 2 closed, 3 failed */
int64_t veles_tls_read(int64_t h, int64_t max, veles_string *out) {
    session *s = session_lock(h);
    if (!s) {
        set_out_string(out, "", 0);
        return T_FAIL;
    }
    if (max < 1) max = 1;
    if (max > 1 << 20) max = 1 << 20;
    buf_t got = {0};
    int r = backend_read(s, max, &got);
    UNLOCK(&s->lock);
    set_out_string(out, got.p, got.len);
    free(got.p);
    return r;
}

/* encrypts bytes[offset..] into the output queue; 0, or 3 */
int64_t veles_tls_write(int64_t h, veles_bytes_view *bytes, int64_t offset) {
    session *s = session_lock(h);
    if (!s) return T_FAIL;
    bool ok = true;
    if (offset < bytes->len) backend_write(s, (const uint8_t *)bytes->data + offset, (size_t)(bytes->len - offset), &ok);
    UNLOCK(&s->lock);
    return ok ? T_OK : T_FAIL;
}

/* queues close_notify */
void veles_tls_close_notify(int64_t h) {
    session *s = session_lock(h);
    if (!s) return;
    backend_shutdown(s);
    UNLOCK(&s->lock);
}

void veles_tls_error(int64_t h, veles_string *out) {
    session *s = session_lock(h);
    if (!s) {
        set_out_string(out, "TLS session closed", 18);
        return;
    }
    char msg[256];
    snprintf(msg, sizeof msg, "%s", s->err[0] ? s->err : "TLS error");
    UNLOCK(&s->lock);
    set_out_string(out, msg, strlen(msg));
}

/* the protocol agreed with ALPN, or empty */
void veles_tls_alpn(int64_t h, veles_string *out) {
    session *s = session_lock(h);
    char msg[64] = "";
    if (s) {
        snprintf(msg, sizeof msg, "%s", s->alpn);
        UNLOCK(&s->lock);
    }
    set_out_string(out, msg, strlen(msg));
}

void veles_tls_free(int64_t h) {
    table_setup();
    LOCK(&table_lock);
    session *s = ht_take(&sessions, h);
    UNLOCK(&table_lock);
    if (!s) return;
    LOCK(&s->lock); /* waits for a call that already holds it */
    UNLOCK(&s->lock);
    free_session(s);
}

/* the certificate chain and the private key, both PEM text, for the connections a
 * server accepts; 0 with a handle, or 3 with the reason in *err */
int64_t veles_tls_server_creds(const char *cert, int64_t clen, const char *key, int64_t klen, const char *alpn,
                               int64_t alen, int64_t *handle, veles_string *err) {
    srvcreds *c = calloc(1, sizeof *c);
    char msg[256] = "out of memory";
    if (!c) {
        set_out_string(err, msg, strlen(msg));
        return T_FAIL;
    }
    if (!backend_server_creds(c, cert, (size_t)clen, key, (size_t)klen, alpn, (size_t)alen, msg, sizeof msg)) {
        backend_creds_free(c);
        free(c);
        set_out_string(err, msg, strlen(msg));
        return T_FAIL;
    }
    c->refs = 1;
    table_setup();
    LOCK(&table_lock);
    *handle = ht_add(&credtab, c);
    UNLOCK(&table_lock);
    if (!*handle) {
        backend_creds_free(c);
        free(c);
        set_out_string(err, "out of memory", 13);
        return T_FAIL;
    }
    return T_OK;
}

/* drops the caller's hold; connections already accepted keep theirs */
void veles_tls_server_creds_free(int64_t h) {
    table_setup();
    LOCK(&table_lock);
    srvcreds *c = ht_take(&credtab, h);
    UNLOCK(&table_lock);
    if (c) creds_unref(c);
}

/* a session for a client that connected; 0 with a handle, or 3 */
int64_t veles_tls_accept(int64_t creds, int64_t *handle, veles_string *err) {
    srvcreds *c = creds_ref(creds);
    if (!c) {
        set_out_string(err, "the certificate was closed", 26);
        return T_FAIL;
    }
    session *s = session_new(true);
    if (!s) {
        creds_unref(c);
        set_out_string(err, "out of memory", 13);
        return T_FAIL;
    }
    s->sc = c;
    if (!backend_server_session(s, c)) {
        char msg[256];
        snprintf(msg, sizeof msg, "%s", s->err);
        free_session(s);
        set_out_string(err, msg, strlen(msg));
        return T_FAIL;
    }
    *handle = table_add(s);
    if (!*handle) {
        free_session(s);
        set_out_string(err, "out of memory", 13);
        return T_FAIL;
    }
    return T_OK;
}
