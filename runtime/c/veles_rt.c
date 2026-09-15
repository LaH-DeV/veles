/*
 * Veles bootstrap runtime.
 *
 * Linked into every Veles executable. Everything here follows the build
 * plan's Stage 1 cut: no garbage collector yet — allocations leak. All
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

typedef struct {
    char *data;
    int64_t len;
    int64_t cap;
    int64_t elem;
} veles_list;

static int g_argc;
static char **g_argv;

void veles_rt_init(int32_t argc, char **argv) {
    g_argc = argc;
    g_argv = argv;
    setvbuf(stdout, NULL, _IOLBF, 0);
}

/* ---- memory (D1 deferred: leak everything) ------------------------------ */

void *veles_alloc(int64_t size) {
    if (size <= 0) size = 1;
    void *p = calloc(1, (size_t)size);
    if (!p) {
        fputs("panic: out of memory\n", stderr);
        exit(101);
    }
    return p;
}

/* ---- panics (D20: unwind to the task scope; bootstrap: exit) ------------ */

void veles_panic(const char *msg, int64_t len) {
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

void veles_f64_to_string(veles_string *out, double v) {
    char buf[64];
    int n = snprintf(buf, sizeof buf, "%.17g", v);
    /* shortest round-trip: try shorter precisions first */
    for (int prec = 1; prec <= 17; prec++) {
        char tmp[64];
        int m = snprintf(tmp, sizeof tmp, "%.*g", prec, v);
        if (strtod(tmp, NULL) == v) {
            memcpy(buf, tmp, (size_t)m + 1);
            n = m;
            break;
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

void veles_bool_to_string(veles_string *out, bool v) {
    out->data = v ? "true" : "false";
    out->len = v ? 4 : 5;
}

/* ---- lists (D25: reference types; element type erased to a size) -------- */

veles_list *veles_list_new(int64_t elem, int64_t cap) {
    veles_list *l = veles_alloc(sizeof *l);
    if (cap < 4) cap = 4;
    l->elem = elem;
    l->cap = cap;
    l->len = 0;
    l->data = veles_alloc(elem * cap);
    return l;
}

int64_t veles_list_len(veles_list *l) {
    return l->len;
}

void veles_list_push(veles_list *l, const void *item) {
    if (l->len == l->cap) {
        int64_t ncap = l->cap * 2;
        char *nd = veles_alloc(l->elem * ncap);
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
    veles_list *c = veles_list_new(l->elem, l->cap);
    memcpy(c->data, l->data, (size_t)(l->elem * l->len));
    c->len = l->len;
    return c;
}

void veles_list_clear(veles_list *l) {
    l->len = 0;
}
