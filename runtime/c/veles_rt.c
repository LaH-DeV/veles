/*
 * Veles bootstrap runtime.
 *
 * Linked into every Veles executable. The collector in veles_gc.c manages
 * every allocation; containers carry element descriptors for it. All
 * entry points take explicit primitive arguments or out-pointers so that
 * no struct crosses the C ABI by value (the compiler passes `string` as a
 * (data, len) pair and receives strings through `veles_string*`).
 */
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <inttypes.h>

typedef struct {
    const char *data;
    int64_t len;
} veles_string;

typedef struct veles_desc veles_desc;

typedef struct {
    char *data;
    int64_t len;
    int64_t cap;
    int64_t elem;
    veles_desc *desc; /* array descriptor of the elements */
} veles_list;

int64_t veles_desc_size(veles_desc *d);

static int g_argc;
static char **g_argv;
void veles_gc_init(void);

void veles_rt_init(int32_t argc, char **argv) {
    g_argc = argc;
    g_argv = argv;
    setvbuf(stdout, NULL, _IOLBF, 0);
    veles_gc_init();
}

/* the process arguments, for std/os (veles_os.c) */
int veles_os_argc_value(void) { return g_argc; }
char **veles_os_argv_value(void) { return g_argv; }

/* ---- memory: the collector in veles_gc.c owns every allocation --------- */

void *veles_alloc(int64_t size);       /* bytes without pointers */
void *veles_alloc_words(int64_t size); /* every word may be a pointer */
void *veles_gc_alloc(veles_desc *desc, int64_t size);

/* ---- panics (D20: unwind to the task scope; bootstrap: exit) ------------ */

int64_t veles_task_panic(const char *msg, int64_t len);

void veles_panic(const char *msg, int64_t len) {
    if (veles_task_panic(msg, len)) return;
    fflush(stdout);
    fputs("panic: ", stderr);
    fwrite(msg, 1, (size_t)len, stderr);
    fputc('\n', stderr);
    fflush(stderr);
    exit(101);
}

void veles_report_error(const char *msg, int64_t len) {
    fflush(stdout);
    fputs("error: main failed with ", stderr);
    fwrite(msg, 1, (size_t)len, stderr);
    fputc('\n', stderr);
    fflush(stderr);
    exit(1);
}

/* ---- console ------------------------------------------------------------ */

void veles_print(const char *s, int64_t len) {
    fwrite(s, 1, (size_t)len, stdout);
}

void veles_eprint(const char *s, int64_t len) {
    fflush(stdout);
    fwrite(s, 1, (size_t)len, stderr);
}

bool veles_read_line(veles_string *out) {
    size_t cap = 128, n = 0;
    char *buf = veles_alloc((int64_t)cap);
    int c;
    bool any = false;
    while ((c = fgetc(stdin)) != EOF) {
        any = true;
        if (c == '\n') break;
        if (n + 1 >= cap) {
            cap *= 2;
            char *nb = veles_alloc((int64_t)cap);
            memcpy(nb, buf, n);
            buf = nb;
        }
        buf[n++] = (char)c;
    }
    if (!any) return false;
    if (n > 0 && buf[n - 1] == '\r') n--;
    out->data = buf;
    out->len = (int64_t)n;
    return true;
}

/* ---- strings (D18: immutable, validated UTF-8, byte-indexed) ------------ */

void veles_string_concat(veles_string *out, const char *a, int64_t alen, const char *b, int64_t blen) {
    char *buf = veles_alloc(alen + blen + 1);
    memcpy(buf, a, (size_t)alen);
    memcpy(buf + alen, b, (size_t)blen);
    out->data = buf;
    out->len = alen + blen;
}

bool veles_string_eq(const char *a, int64_t alen, const char *b, int64_t blen) {
    return alen == blen && (alen == 0 || memcmp(a, b, (size_t)alen) == 0);
}

int32_t veles_string_cmp(const char *a, int64_t alen, const char *b, int64_t blen) {
    int64_t n = alen < blen ? alen : blen;
    int r = n ? memcmp(a, b, (size_t)n) : 0;
    if (r != 0) return r < 0 ? -1 : 1;
    if (alen == blen) return 0;
    return alen < blen ? -1 : 1;
}

bool veles_string_starts_with(const char *a, int64_t alen, const char *b, int64_t blen) {
    return blen <= alen && memcmp(a, b, (size_t)blen) == 0;
}

bool veles_string_ends_with(const char *a, int64_t alen, const char *b, int64_t blen) {
    return blen <= alen && memcmp(a + (alen - blen), b, (size_t)blen) == 0;
}

bool veles_string_contains(const char *a, int64_t alen, const char *b, int64_t blen) {
    if (blen == 0) return true;
    for (int64_t i = 0; i + blen <= alen; i++) {
        if (memcmp(a + i, b, (size_t)blen) == 0) return true;
    }
    return false;
}

static bool is_boundary(const char *s, int64_t len, int64_t i) {
    if (i == 0 || i == len) return true;
    return ((unsigned char)s[i] & 0xC0) != 0x80;
}

/* D19: a slice that splits a code point returns null rather than panicking;
 * an out-of-range slice is a panic like any bounds violation (D20). */
bool veles_string_substring(veles_string *out, const char *s, int64_t len, int64_t lo, int64_t hi) {
    if (lo < 0 || hi > len || lo > hi) {
        char msg[96];
        int n = snprintf(msg, sizeof msg, "substring bounds %" PRId64 "..%" PRId64 " out of range for length %" PRId64, lo, hi, len);
        veles_panic(msg, n);
    }
    if (!is_boundary(s, len, lo) || !is_boundary(s, len, hi)) return false;
    out->data = s + lo;
    out->len = hi - lo;
    return true;
}

veles_list *veles_list_new(veles_desc *desc, int64_t cap);
void veles_list_push(veles_list *l, const void *item);

/* D18: len() is bytes. These give the Unicode scalar view on demand. */
int64_t veles_string_char_count(const char *s, int64_t len) {
    int64_t n = 0;
    for (int64_t i = 0; i < len; i++) {
        if (((unsigned char)s[i] & 0xC0) != 0x80) n++;
    }
    return n;
}

veles_list *veles_string_chars(veles_desc *desc, const char *s, int64_t len) {
    veles_list *l = veles_list_new(desc, veles_string_char_count(s, len));
    int64_t i = 0;
    while (i < len) {
        int64_t j = i + 1;
        while (j < len && ((unsigned char)s[j] & 0xC0) == 0x80) j++;
        veles_string ch = { (char *)s + i, j - i };
        veles_list_push(l, &ch);
        i = j;
    }
    return l;
}

bool veles_string_to_int(const char *s, int64_t len, int64_t *out) {
    if (len == 0 || len > 20) return false;
    int64_t i = 0;
    bool neg = false;
    if (s[0] == '-' || s[0] == '+') {
        neg = s[0] == '-';
        i = 1;
        if (len == 1) return false;
    }
    uint64_t v = 0;
    for (; i < len; i++) {
        if (s[i] < '0' || s[i] > '9') return false;
        uint64_t d = (uint64_t)(s[i] - '0');
        if (v > (UINT64_MAX - d) / 10) return false;
        v = v * 10 + d;
    }
    if (neg) {
        if (v > (uint64_t)INT64_MAX + 1) return false;
        *out = (int64_t)(0 - v);
    } else {
        if (v > (uint64_t)INT64_MAX) return false;
        *out = (int64_t)v;
    }
    return true;
}

static void from_buf(veles_string *out, const char *buf, int n) {
    char *p = veles_alloc(n + 1);
    memcpy(p, buf, (size_t)n);
    out->data = p;
    out->len = n;
}

void veles_i64_to_string(veles_string *out, int64_t v) {
    char buf[32];
    int n = snprintf(buf, sizeof buf, "%" PRId64, v);
    from_buf(out, buf, n);
}

void veles_u64_to_string(veles_string *out, uint64_t v) {
    char buf[32];
    int n = snprintf(buf, sizeof buf, "%" PRIu64, v);
    from_buf(out, buf, n);
}

static void float_to_string(veles_string *out, double v, int is_f32) {
    char buf[64];
    /* one spelling on every platform (the MSVC runtime prints -nan(ind)) */
    if (v != v) {
        from_buf(out, "NaN", 3);
        return;
    }
    if (v != 0 && v * 2 == v) { /* only the infinities satisfy this */
        if (v > 0) from_buf(out, "inf", 3); else from_buf(out, "-inf", 4);
        return;
    }
    int n = snprintf(buf, sizeof buf, "%.17g", v);
    /* shortest representation that round-trips at the value's own precision */
    int max = is_f32 ? 9 : 17;
    for (int prec = 1; prec <= max; prec++) {
        char tmp[64];
        int m = snprintf(tmp, sizeof tmp, "%.*g", prec, v);
        int same = is_f32 ? ((float)strtod(tmp, NULL) == (float)v) : (strtod(tmp, NULL) == v);
        if (same) {
            memcpy(buf, tmp, (size_t)m + 1);
            n = m;
            break;
        }
    }
    /* %g switches to exponent form early (1.5e+03); prefer plain digits for
       magnitudes people read as ordinary numbers */
    char *e = strchr(buf, 'e');
    if (e && v == v && v - v == 0) {
        int exp = atoi(e + 1);
        if (exp >= -4 && exp < 15) {
            char tmp[64];
            int decimals = (int)(e - buf) - 2; /* digits after the point in mantissa */
            if (decimals < 0) decimals = 0;
            int frac = decimals - exp;
            if (frac < 0) frac = 0;
            int m = snprintf(tmp, sizeof tmp, "%.*f", frac, v);
            if (m > 0 && m < 60) {
                memcpy(buf, tmp, (size_t)m + 1);
                n = m;
            }
        }
    }
    /* make sure it reads as a float */
    bool has_point = false;
    for (int i = 0; i < n; i++) {
        if (buf[i] == '.' || buf[i] == 'e' || buf[i] == 'n' || buf[i] == 'i') has_point = true;
    }
    if (!has_point && n < 60) {
        buf[n++] = '.';
        buf[n++] = '0';
        buf[n] = 0;
    }
    from_buf(out, buf, n);
}

void veles_f64_to_string(veles_string *out, double v) { float_to_string(out, v, 0); }
void veles_f32_to_string(veles_string *out, float v) { float_to_string(out, (double)v, 1); }

void veles_bool_to_string(veles_string *out, bool v) {
    out->data = v ? "true" : "false";
    out->len = v ? 4 : 5;
}

/* ---- bytes: the primitives the prelude builds the string methods on ---- */

uint8_t veles_string_byte_at(const char *s, int64_t len, int64_t i) {
    if (i < 0 || i >= len) {
        char msg[80];
        int n = snprintf(msg, sizeof msg, "index %" PRId64 " out of bounds for string of length %" PRId64, i, len);
        veles_panic(msg, n);
    }
    return (uint8_t)s[i];
}

veles_list *veles_string_bytes(veles_desc *desc, const char *s, int64_t len) {
    veles_list *l = veles_list_new(desc, len);
    memcpy(l->data, s, (size_t)len);
    l->len = len;
    return l;
}

/* veles_utf8_valid checks the encoding: shortest forms only, no surrogates,
 * nothing above U+10FFFF (D18: every string is valid UTF-8). */
static bool veles_utf8_valid(const unsigned char *p, int64_t len) {
    int64_t i = 0;
    while (i < len) {
        unsigned char c = p[i];
        if (c < 0x80) { i++; continue; }
        int n;
        uint32_t cp;
        if ((c & 0xE0) == 0xC0) { n = 1; cp = c & 0x1F; if (c < 0xC2) return false; }
        else if ((c & 0xF0) == 0xE0) { n = 2; cp = c & 0x0F; }
        else if ((c & 0xF8) == 0xF0) { n = 3; cp = c & 0x07; if (c > 0xF4) return false; }
        else return false;
        for (int k = 1; k <= n; k++) {
            if (i + k >= len || (p[i + k] & 0xC0) != 0x80) return false;
            cp = (cp << 6) | (p[i + k] & 0x3F);
        }
        if (n == 2 && (cp < 0x800 || (cp >= 0xD800 && cp <= 0xDFFF))) return false;
        if (n == 3 && (cp < 0x10000 || cp > 0x10FFFF)) return false;
        i += n + 1;
    }
    return true;
}

bool veles_bytes_decode_utf8(veles_string *out, veles_list *bytes) {
    if (!veles_utf8_valid((const unsigned char *)bytes->data, bytes->len)) return false;
    char *buf = veles_alloc(bytes->len + 1);
    memcpy(buf, bytes->data, (size_t)bytes->len);
    out->data = buf;
    out->len = bytes->len;
    return true;
}

/* veles_string_find returns the byte index of the first occurrence of part
 * at or after from, or -1. */
int64_t veles_string_find(const char *s, int64_t len, const char *part, int64_t plen, int64_t from) {
    if (from < 0) from = 0;
    if (plen == 0) return from <= len ? from : -1;
    for (int64_t i = from; i + plen <= len; i++) {
        if (s[i] == part[0] && memcmp(s + i, part, (size_t)plen) == 0) return i;
    }
    return -1;
}

/* ---- lists (D25: reference types; element type erased to a size) -------- */

veles_list *veles_list_new(veles_desc *desc, int64_t cap) {
    veles_list *l = veles_alloc_words(sizeof *l);
    if (cap < 4) cap = 4;
    l->elem = veles_desc_size(desc);
    l->desc = desc;
    l->cap = cap;
    l->len = 0;
    l->data = veles_gc_alloc(desc, l->elem * cap);
    return l;
}

int64_t veles_list_len(veles_list *l) {
    return l->len;
}

void veles_list_push(veles_list *l, const void *item) {
    if (l->len == l->cap) {
        int64_t ncap = l->cap * 2;
        char *nd = veles_gc_alloc(l->desc, l->elem * ncap);
        memcpy(nd, l->data, (size_t)(l->elem * l->len));
        l->data = nd;
        l->cap = ncap;
    }
    if (l->elem > 0) memcpy(l->data + l->elem * l->len, item, (size_t)l->elem);
    l->len++;
}

void *veles_list_ref(veles_list *l, int64_t i) {
    if (i < 0 || i >= l->len) {
        char msg[80];
        int n = snprintf(msg, sizeof msg, "index %" PRId64 " out of bounds for list of length %" PRId64, i, l->len);
        veles_panic(msg, n);
    }
    return l->data + l->elem * i;
}

bool veles_list_pop(veles_list *l, void *out) {
    if (l->len == 0) return false;
    l->len--;
    if (l->elem > 0) memcpy(out, l->data + l->elem * l->len, (size_t)l->elem);
    return true;
}

veles_list *veles_list_copy(veles_list *l) {
    veles_list *c = veles_list_new(l->desc, l->cap);
    memcpy(c->data, l->data, (size_t)(l->elem * l->len));
    c->len = l->len;
    return c;
}

void veles_list_clear(veles_list *l) {
    l->len = 0;
}

/* ---- maps and sets (D25: insertion-ordered; a set is a map with no values) */

typedef bool (*veles_eq_fn)(const void *, const void *);

typedef struct {
    int64_t hash;
    bool live;
} veles_meta;

typedef struct {
    char *keys;
    char *vals;
    veles_meta *meta;
    veles_desc *keyDesc;
    veles_desc *valDesc;
    int64_t *index; /* open addressing: 0 empty, -1 tombstone, else entry+1 */
    int64_t icap;
    int64_t len;  /* live entries */
    int64_t used; /* entries appended, including dead ones */
    int64_t cap;
    int64_t keySize;
    int64_t valSize;
} veles_map;

int64_t veles_hash_bytes(const char *p, int64_t len) {
    uint64_t h = 1469598103934665603ULL;
    for (int64_t i = 0; i < len; i++) {
        h ^= (unsigned char)p[i];
        h *= 1099511628211ULL;
    }
    return (int64_t)h;
}

int64_t veles_hash_mix(int64_t a, int64_t b) {
    uint64_t h = (uint64_t)a * 0x9E3779B97F4A7C15ULL;
    h ^= (uint64_t)b + 0x7F4A7C15ULL + (h << 6) + (h >> 2);
    return (int64_t)h;
}

static void map_index_insert(veles_map *m, int64_t hash, int64_t entry) {
    uint64_t mask = (uint64_t)m->icap - 1;
    uint64_t i = (uint64_t)hash & mask;
    while (m->index[i] > 0) i = (i + 1) & mask;
    m->index[i] = entry + 1;
}

static void map_rebuild(veles_map *m, int64_t icap) {
    m->icap = icap;
    m->index = veles_alloc(icap * (int64_t)sizeof(int64_t));
    for (int64_t e = 0; e < m->used; e++) {
        if (m->meta[e].live) map_index_insert(m, m->meta[e].hash, e);
    }
}

veles_map *veles_map_new(veles_desc *keyDesc, veles_desc *valDesc) {
    veles_map *m = veles_alloc_words(sizeof *m);
    m->keyDesc = keyDesc;
    m->valDesc = valDesc;
    m->keySize = veles_desc_size(keyDesc);
    m->valSize = valDesc ? veles_desc_size(valDesc) : 0;
    m->cap = 8;
    m->keys = veles_gc_alloc(keyDesc, m->keySize * m->cap + 1);
    m->vals = valDesc ? veles_gc_alloc(valDesc, m->valSize * m->cap + 1) : veles_alloc(1);
    m->meta = veles_alloc((int64_t)sizeof(veles_meta) * m->cap);
    map_rebuild(m, 16);
    return m;
}

int64_t veles_map_len(veles_map *m) {
    return m->len;
}

int64_t veles_map_find(veles_map *m, int64_t hash, const void *key, veles_eq_fn eq) {
    uint64_t mask = (uint64_t)m->icap - 1;
    uint64_t i = (uint64_t)hash & mask;
    for (;;) {
        int64_t slot = m->index[i];
        if (slot == 0) return -1;
        if (slot > 0) {
            int64_t e = slot - 1;
            if (m->meta[e].hash == hash && (m->keySize == 0 || eq(key, m->keys + e * m->keySize))) return e;
        }
        i = (i + 1) & mask;
    }
}

static void map_compact(veles_map *m) {
    int64_t w = 0;
    for (int64_t e = 0; e < m->used; e++) {
        if (!m->meta[e].live) continue;
        if (w != e) {
            memcpy(m->keys + w * m->keySize, m->keys + e * m->keySize, (size_t)m->keySize);
            memcpy(m->vals + w * m->valSize, m->vals + e * m->valSize, (size_t)m->valSize);
            m->meta[w] = m->meta[e];
        }
        w++;
    }
    m->used = w;
    map_rebuild(m, m->icap);
}

/* insert or overwrite; returns the entry index */
int64_t veles_map_insert(veles_map *m, int64_t hash, const void *key, const void *val, veles_eq_fn eq) {
    int64_t e = veles_map_find(m, hash, key, eq);
    if (e >= 0) {
        if (m->valSize) memcpy(m->vals + e * m->valSize, val, (size_t)m->valSize);
        return e;
    }
    if (m->used == m->cap) {
        if (m->used > 2 * m->len + 8) {
            map_compact(m);
        } else {
            int64_t ncap = m->cap * 2;
            char *nk = veles_gc_alloc(m->keyDesc, m->keySize * ncap + 1);
            char *nv = m->valDesc ? veles_gc_alloc(m->valDesc, m->valSize * ncap + 1) : veles_alloc(1);
            veles_meta *nm = veles_alloc((int64_t)sizeof(veles_meta) * ncap);
            memcpy(nk, m->keys, (size_t)(m->keySize * m->used));
            memcpy(nv, m->vals, (size_t)(m->valSize * m->used));
            memcpy(nm, m->meta, sizeof(veles_meta) * (size_t)m->used);
            m->keys = nk;
            m->vals = nv;
            m->meta = nm;
            m->cap = ncap;
        }
    }
    if (m->used * 2 >= m->icap) map_rebuild(m, m->icap * 2);
    e = m->used++;
    if (m->keySize) memcpy(m->keys + e * m->keySize, key, (size_t)m->keySize);
    if (m->valSize) memcpy(m->vals + e * m->valSize, val, (size_t)m->valSize);
    m->meta[e].hash = hash;
    m->meta[e].live = true;
    m->len++;
    map_index_insert(m, hash, e);
    return e;
}

bool veles_map_remove(veles_map *m, int64_t hash, const void *key, veles_eq_fn eq) {
    uint64_t mask = (uint64_t)m->icap - 1;
    uint64_t i = (uint64_t)hash & mask;
    for (;;) {
        int64_t slot = m->index[i];
        if (slot == 0) return false;
        if (slot > 0) {
            int64_t e = slot - 1;
            if (m->meta[e].hash == hash && (m->keySize == 0 || eq(key, m->keys + e * m->keySize))) {
                m->meta[e].live = false;
                m->index[i] = -1;
                m->len--;
                return true;
            }
        }
        i = (i + 1) & mask;
    }
}

void *veles_map_key_at(veles_map *m, int64_t e) {
    return m->keys + e * m->keySize;
}

void *veles_map_val_at(veles_map *m, int64_t e) {
    return m->vals + e * m->valSize;
}

int64_t veles_map_used(veles_map *m) {
    return m->used;
}

bool veles_map_live(veles_map *m, int64_t e) {
    return m->meta[e].live;
}

void veles_map_clear(veles_map *m) {
    m->len = 0;
    m->used = 0;
    map_rebuild(m, m->icap);
}

veles_map *veles_map_copy(veles_map *m) {
    veles_map *c = veles_map_new(m->keyDesc, m->valDesc);
    for (int64_t e = 0; e < m->used; e++) {
        if (m->meta[e].live) {
            veles_map_insert(c, m->meta[e].hash, m->keys + e * m->keySize, m->vals + e * m->valSize, NULL);
        }
    }
    return c;
}

veles_list *veles_map_keys(veles_map *m) {
    veles_list *l = veles_list_new(m->keyDesc, m->len);
    for (int64_t e = 0; e < m->used; e++) {
        if (m->meta[e].live) veles_list_push(l, m->keys + e * m->keySize);
    }
    return l;
}

veles_list *veles_map_values(veles_map *m) {
    veles_list *l = veles_list_new(m->valDesc, m->len);
    for (int64_t e = 0; e < m->used; e++) {
        if (m->meta[e].live) veles_list_push(l, m->vals + e * m->valSize);
    }
    return l;
}

/* entries as (K, V) tuples laid out with the value at valOffset */
veles_list *veles_map_entries(veles_map *m, veles_desc *tupleDesc, int64_t valOffset, int64_t tupleSize) {
    veles_list *l = veles_list_new(tupleDesc, m->len);
    char *tmp = veles_alloc(tupleSize);
    for (int64_t e = 0; e < m->used; e++) {
        if (!m->meta[e].live) continue;
        memset(tmp, 0, (size_t)tupleSize);
        memcpy(tmp, m->keys + e * m->keySize, (size_t)m->keySize);
        memcpy(tmp + valOffset, m->vals + e * m->valSize, (size_t)m->valSize);
        veles_list_push(l, tmp);
    }
    return l;
}

/* Integer exponentiation by squaring for every width: the operands arrive
 * extended to 64 bits, `bits`/`is_signed` describe the Veles type so that
 * overflow is detected at that type's range (D21: overflow panics). */
int64_t veles_int_pow(int64_t base, int64_t exp, int64_t bits, bool is_signed, const char *where, int64_t where_len) {
    char msg[512];
    if (exp < 0) {
        int n = snprintf(msg, sizeof msg, "negative exponent %" PRId64 " in integer pow at %.*s", exp, (int)where_len, where);
        if (n >= (int)sizeof msg) n = (int)sizeof msg - 1;
        veles_panic(msg, n);
    }
    uint64_t lo = 0, hi = 0;
    if (is_signed) {
        hi = (bits == 64) ? (uint64_t)INT64_MAX : ((uint64_t)1 << (bits - 1)) - 1;
        lo = ~hi; /* two's complement minimum, as an unsigned bit pattern */
    } else {
        hi = (bits == 64) ? UINT64_MAX : ((uint64_t)1 << bits) - 1;
    }
    /* compute in 128-bit-free style: check each multiply against the range */
    int64_t result = 1, b = base;
    bool overflow = false;
    while (exp > 0) {
        if (exp & 1) {
            if (__builtin_mul_overflow(result, b, &result)) { overflow = true; break; }
        }
        exp >>= 1;
        if (exp > 0 && __builtin_mul_overflow(b, b, &b)) { overflow = true; break; }
    }
    if (!overflow) {
        if (is_signed) {
            overflow = result > (int64_t)hi || result < (int64_t)lo;
        } else {
            overflow = (uint64_t)result > hi || result < 0;
        }
    }
    if (overflow) {
        int n = snprintf(msg, sizeof msg, "integer overflow in pow at %.*s", (int)where_len, where);
        if (n >= (int)sizeof msg) n = (int)sizeof msg - 1;
        veles_panic(msg, n);
    }
    return result;
}
