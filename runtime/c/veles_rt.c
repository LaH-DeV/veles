/*
 * Veles bootstrap runtime.
 *
 * Linked into every Veles executable. The collector in veles_gc.c manages
 * every allocation; containers carry element descriptors for it. All
 * entry points take explicit primitive arguments or out-pointers so that
 * no struct crosses the C ABI by value (the compiler passes `string` as a
 * (data, len) pair and receives strings through `veles_string*`).
 */
/* glibc declares its extensions (pthread_getattr_np, ...) only when asked. */
#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <stdbool.h>
#include <inttypes.h>
#include <time.h>
#if defined(_WIN32)
#include <windows.h>
#include <fcntl.h> /* _O_BINARY */
#include <io.h>    /* _setmode */
#else
#include <sys/stat.h> /* fstat: is stdout a file */
#include <unistd.h>   /* isatty */
#endif

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
    int64_t mods;      /* D102: changes of length or order so far; a loop over
                        * the list compares it on every step */
    int64_t flags;     /* LIST_EXPOSED / LIST_FROZEN (D188) */
} veles_list;

/* D188: a pointer into the storage was made (EXPOSED); freeze() turned the
 * collection immutable (FROZEN): any change through a name that still sees it
 * as mutable panics. The same two bits on a map's header. */
#define LIST_EXPOSED 1
#define LIST_FROZEN 2

int64_t veles_desc_size(veles_desc *d);

static int g_argc;
static char **g_argv;
void veles_gc_init(void);
void veles_task_init(void);
void veles_ffi_init(void);
void veles_sync_init(void);

/* stdout goes somewhere someone may be watching — a terminal, a pipe into
 * `docker logs` or the journal — rather than to a regular file: each line
 * is written out as it ends (veles_print). To a file, lines are buffered:
 * a million of them take a tenth of the time. */
static int flush_lines = 1;

void veles_hash_init(void); /* the maps' hash key, below */

void veles_rt_init(int32_t argc, char **argv) {
    veles_sync_init(); /* the per-thread block's slot first: everything else uses it */
    g_argc = argc;
    g_argv = argv;
#if defined(_WIN32)
    flush_lines = GetFileType(GetStdHandle(STD_OUTPUT_HANDLE)) != FILE_TYPE_DISK;
#else
    struct stat st;
    flush_lines = !(fstat(1, &st) == 0 && S_ISREG(st.st_mode));
#endif
    setvbuf(stdout, NULL, flush_lines ? _IOLBF : _IOFBF, 0);
#if defined(_WIN32)
    /* The standard streams carry the bytes the program wrote, as on every
     * other platform and as files already do ("rb"/"wb"): text mode turned
     * each '\n' into "\r\n" and ended stdin at the first 0x1A byte.
     * readLine still drops a line's trailing '\r'. */
    _setmode(_fileno(stdin), _O_BINARY);
    _setmode(_fileno(stdout), _O_BINARY);
    _setmode(_fileno(stderr), _O_BINARY);
    /* the console shows UTF-8 as such (D18); pipes and files are bytes anyway */
    SetConsoleOutputCP(CP_UTF8);
    SetConsoleCP(CP_UTF8);
#endif
    veles_gc_init();
    veles_hash_init(); /* its key comes from the heap-allocating veles_random_bytes */
    veles_task_init(); /* the executor's lock, before any channel or task */
    veles_ffi_init();
}

/* the process arguments, for std/os (veles_os.c) */
int veles_os_argc_value(void) { return g_argc; }
char **veles_os_argv_value(void) { return g_argv; }

/* ---- memory: the collector in veles_gc.c owns every allocation --------- */

void *veles_alloc(int64_t size);       /* bytes without pointers */
void *veles_alloc_words(int64_t size); /* every word may be a pointer */
void *veles_gc_alloc(veles_desc *desc, int64_t size);

/* ---- panics (D20: unwind to the task scope; bootstrap: exit) ------------ */

int64_t veles_task_panic(const char *msg, int64_t len, const char *loc, int64_t loc_len);

/* veles_panic_at fails with a message and the source location the compiler
 * wrote at the panicking site (D64: `file:line:col`, relative to the package
 * root; empty for a panic raised inside the runtime itself). */
int64_t veles_ffi_in_callback(void);
void veles_panic_print_report(const char *loc, int64_t loc_len, int64_t indent);
int veles_chain_recorded(void);

void veles_panic_at(const char *msg, int64_t len, const char *loc, int64_t loc_len) {
    /* inside an `extern "C" fun` there are C frames below: unwinding to
     * the task boundary would jump over them (D69), so the process ends */
    int in_c = veles_ffi_in_callback() > 0;
    if (!in_c && veles_task_panic(msg, len, loc, loc_len)) return;
    fflush(stdout);
    fputs("panic: ", stderr);
    fwrite(msg, 1, (size_t)len, stderr);
    veles_panic_print_report(loc, loc_len, 2);
    fputc('\n', stderr);
    if (!veles_chain_recorded())
        fputs("  (a debug build shows the call chain)\n", stderr);
    if (in_c)
        fputs("  (in a function called from C, which cannot be unwound: the process ends)\n", stderr);
    fflush(stderr);
    exit(101);
}

void veles_panic(const char *msg, int64_t len) {
    veles_panic_at(msg, len, NULL, 0);
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

/* while `veles test` runs a test, what it prints is kept for the report
 * (veles_sync.c) and shown only if the test fails */
int veles_test_capture(const char *s, int64_t len, int newline);

/* Standard output is written out at the end of every line, terminal or
 * not (as Rust's is): a service's log line reaches `docker logs` or the
 * journal when it is printed, not when a buffer fills or the process ends.
 * C's line buffering would do it on Unix, but Windows' runtime treats it
 * as full buffering, so the flush is explicit. */
void veles_print(const char *s, int64_t len) {
    if (veles_test_capture(s, len, 0)) return;
    fwrite(s, 1, (size_t)len, stdout);
    if (flush_lines && len > 0 && memchr(s, '\n', (size_t)len)) fflush(stdout);
}

void veles_eprint(const char *s, int64_t len) {
    if (veles_test_capture(s, len, 0)) return;
    fflush(stdout);
    fwrite(s, 1, (size_t)len, stderr);
}

/* A line goes out whole: tasks on other threads printing at the same time
 * (D66) never land in the middle of it. */
#if defined(_WIN32)
#define lock_stream(f) _lock_file(f)
#define unlock_stream(f) _unlock_file(f)
#else
#define lock_stream(f) flockfile(f)
#define unlock_stream(f) funlockfile(f)
#endif

static void write_line(FILE *f, const char *s, int64_t len) {
    lock_stream(f);
    fwrite(s, 1, (size_t)len, f);
    fputc('\n', f);
    if (f != stdout || flush_lines) fflush(f); /* a line is out when it is printed (see veles_print) */
    unlock_stream(f);
}

void veles_println(const char *s, int64_t len) {
    if (veles_test_capture(s, len, 1)) return;
    write_line(stdout, s, len);
}

void veles_eprintln(const char *s, int64_t len) {
    if (veles_test_capture(s, len, 1)) return;
    fflush(stdout);
    write_line(stderr, s, len);
}

/* std/log picks text for a person and JSON for everything else (D91) */
bool veles_stderr_is_terminal(void) {
#if defined(_WIN32)
    return _isatty(_fileno(stderr)) != 0;
#else
    return isatty(2) != 0;
#endif
}

void veles_blocking_enter(void);
void veles_blocking_leave(void);

/* Waiting for the terminal can take forever: the line is read into memory
 * outside the Veles heap, in a safe region (D66), then copied. */
bool veles_read_line(veles_string *out) {
    size_t cap = 128, n = 0;
    char *buf = malloc(cap);
    if (!buf) veles_panic("out of memory", 13);
    int c;
    bool any = false;
    veles_blocking_enter();
    while ((c = fgetc(stdin)) != EOF) {
        any = true;
        if (c == '\n') break;
        if (n + 1 >= cap) {
            cap *= 2;
            char *nb = realloc(buf, cap);
            if (!nb) {
                veles_blocking_leave();
                veles_panic("out of memory", 13);
            }
            buf = nb;
        }
        buf[n++] = (char)c;
    }
    veles_blocking_leave();
    if (!any) {
        free(buf);
        return false;
    }
    if (n > 0 && buf[n - 1] == '\r') n--;
    char *data = veles_alloc((int64_t)n + 1);
    memcpy(data, buf, n);
    data[n] = 0;
    free(buf);
    out->data = data;
    out->len = (int64_t)n;
    return true;
}

/* ---- strings (D18: immutable, validated UTF-8, byte-indexed) ------------ */

void veles_string_concat(veles_string *out, const char *a, int64_t alen, const char *b, int64_t blen) {
    if (alen + blen == 0) { out->data = ""; out->len = 0; return; }
    char *buf = veles_alloc(alen + blen + 1);
    memcpy(buf, a, (size_t)alen);
    memcpy(buf + alen, b, (size_t)blen);
    out->data = buf;
    out->len = alen + blen;
}

/* The same for pieces that are all heap, constant or slice strings (not a
 * number formatted into a caller's temporary buffer): strings are immutable,
 * so joining with an empty one is the other string itself, with no copy. */
void veles_string_concat_shared(veles_string *out, const char *a, int64_t alen, const char *b, int64_t blen) {
    if (blen == 0) { out->data = a; out->len = alen; return; }
    if (alen == 0) { out->data = b; out->len = blen; return; }
    veles_string_concat(out, a, alen, b, blen);
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

/* D19: the result is already a string?, so every way a slice can fail
 * answers null: bounds outside the text or reversed, and a cut through a
 * code point. A parser probing `s.substring(pos, pos + 4)` near the end
 * wants that, not a panic. (Changed 2026-09-18; it used to panic.) */
bool veles_string_substring(veles_string *out, const char *s, int64_t len, int64_t lo, int64_t hi) {
    if (lo < 0 || hi > len || lo > hi) return false;
    if (!is_boundary(s, len, lo) || !is_boundary(s, len, hi)) return false;
    out->data = s + lo;
    out->len = hi - lo;
    return true;
}

veles_list *veles_list_new(veles_desc *desc, int64_t cap);
void veles_list_push(veles_list *l, const void *item);

/* veles_string_ascii_case: s with its ASCII letters in upper (or lower)
 * case, in one copy; s itself when nothing changes (strings are immutable).
 * Other bytes are kept, so UTF-8 stays UTF-8. */
void veles_string_ascii_case(veles_string *out, const char *s, int64_t len, bool upper) {
    int64_t first = 0;
    for (; first < len; first++) {
        unsigned char c = (unsigned char)s[first];
        if (upper ? (c >= 'a' && c <= 'z') : (c >= 'A' && c <= 'Z')) break;
    }
    if (first == len) { out->data = s; out->len = len; return; }
    char *buf = veles_alloc(len + 1);
    memcpy(buf, s, (size_t)first);
    for (int64_t i = first; i < len; i++) {
        unsigned char c = (unsigned char)s[i];
        if (upper && c >= 'a' && c <= 'z') c -= 32;
        else if (!upper && c >= 'A' && c <= 'Z') c += 32;
        buf[i] = (char)c;
    }
    out->data = buf;
    out->len = len;
}

/* veles_string_replace: s with every occurrence of a non-empty old as repl,
 * in one pass and one copy; s itself when old does not occur. */
void veles_string_replace(veles_string *out, const char *s, int64_t len, const char *old, int64_t olen, const char *repl, int64_t rlen) {
    int64_t count = 0;
    for (int64_t i = 0; olen > 0 && i + olen <= len;) {
        if (memcmp(s + i, old, (size_t)olen) == 0) { count++; i += olen; } else i++;
    }
    if (count == 0) { out->data = s; out->len = len; return; }
    int64_t n = len + count * (rlen - olen);
    char *buf = veles_alloc(n + 1);
    int64_t w = 0, i = 0;
    while (i < len) {
        if (i + olen <= len && memcmp(s + i, old, (size_t)olen) == 0) {
            memcpy(buf + w, repl, (size_t)rlen);
            w += rlen;
            i += olen;
        } else {
            buf[w++] = s[i++];
        }
    }
    out->data = buf;
    out->len = n;
}

/* veles_string_split: the parts of s between occurrences of a non-empty
 * sep, as strings that share s's bytes (D10: the collector follows an
 * interior pointer). Two passes — count, then fill a list of exactly that
 * size — and memchr to find a one-byte separator. */
veles_list *veles_string_split(veles_desc *desc, const char *s, int64_t len, const char *sep, int64_t slen) {
    int64_t n = 1;
    for (int64_t i = 0; i + slen <= len;) {
        const char *hit = slen == 1 ? memchr(s + i, sep[0], (size_t)(len - i)) : NULL;
        if (slen == 1) {
            if (!hit) break;
            n++;
            i = (hit - s) + 1;
            continue;
        }
        if (memcmp(s + i, sep, (size_t)slen) == 0) {
            n++;
            i += slen;
        } else {
            i++;
        }
    }
    veles_list *l = veles_list_new(desc, n);
    veles_string part;
    int64_t start = 0;
    for (int64_t i = 0; i + slen <= len;) {
        int64_t at;
        if (slen == 1) {
            const char *hit = memchr(s + i, sep[0], (size_t)(len - i));
            if (!hit) break;
            at = hit - s;
        } else if (memcmp(s + i, sep, (size_t)slen) == 0) {
            at = i;
        } else {
            i++;
            continue;
        }
        part.data = s + start;
        part.len = at - start;
        veles_list_push(l, &part);
        start = at + slen;
        i = start;
    }
    part.data = s + start;
    part.len = len - start;
    veles_list_push(l, &part);
    return l;
}

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

/* Integer formatting writes the digits into a caller's buffer of at least
 * VELES_INT_DIGITS bytes and returns their count. Interpolation formats into
 * a stack buffer and copies once into the finished string; snprintf cost
 * ~100 ns a number in the Windows CRT, most of `"item-$i"`. */
#define VELES_INT_DIGITS 20

static const char digit_pairs[201] =
    "00010203040506070809101112131415161718192021222324252627282930313233343536373839"
    "40414243444546474849505152535455565758596061626364656667686970717273747576777879"
    "8081828384858687888990919293949596979899";

int64_t veles_u64_format(char *buf, uint64_t v) {
    char tmp[VELES_INT_DIGITS];
    int i = VELES_INT_DIGITS;
    while (v >= 100) {
        unsigned r = (unsigned)(v % 100);
        v /= 100;
        tmp[--i] = digit_pairs[2 * r + 1];
        tmp[--i] = digit_pairs[2 * r];
    }
    if (v >= 10) {
        tmp[--i] = digit_pairs[2 * v + 1];
        tmp[--i] = digit_pairs[2 * v];
    } else {
        tmp[--i] = (char)('0' + v);
    }
    int64_t n = VELES_INT_DIGITS - i;
    memcpy(buf, tmp + i, (size_t)n);
    return n;
}

/* one more byte than the digits for the sign */
int64_t veles_i64_format(char *buf, int64_t v) {
    if (v < 0) {
        buf[0] = '-';
        return 1 + veles_u64_format(buf + 1, 0 - (uint64_t)v);
    }
    return veles_u64_format(buf, (uint64_t)v);
}

void veles_i64_to_string(veles_string *out, int64_t v) {
    char buf[VELES_INT_DIGITS + 1];
    from_buf(out, buf, (int)veles_i64_format(buf, v));
}

void veles_u64_to_string(veles_string *out, uint64_t v) {
    char buf[VELES_INT_DIGITS];
    from_buf(out, buf, (int)veles_u64_format(buf, v));
}

void veles_string_concat_n(veles_string *out, const veles_string *parts, int64_t n);

/* concat_n for pieces that are all heap, constant or slice strings (not a
 * number formatted into a caller's temporary buffer): a single non-empty
 * piece is the result itself */
void veles_string_concat_n_shared(veles_string *out, const veles_string *parts, int64_t n) {
    int64_t nonEmpty = 0, last = 0;
    for (int64_t i = 0; i < n; i++) {
        if (parts[i].len) { nonEmpty++; last = i; }
    }
    if (nonEmpty == 1) { *out = parts[last]; return; }
    veles_string_concat_n(out, parts, n);
}

/* concatenates n strings with one allocation (interpolation, D18) */
void veles_string_concat_n(veles_string *out, const veles_string *parts, int64_t n) {
    int64_t len = 0;
    for (int64_t i = 0; i < n; i++) len += parts[i].len;
    /* a piece may live in a caller's temporary buffer (a formatted number),
     * so a single piece is copied too; only the empty result is shared */
    if (len == 0) { out->data = ""; out->len = 0; return; }
    char *buf = veles_alloc(len + 1);
    char *p = buf;
    for (int64_t i = 0; i < n; i++) {
        if (parts[i].len) memcpy(p, parts[i].data, (size_t)parts[i].len);
        p += parts[i].len;
    }
    out->data = buf;
    out->len = len;
}

/* f64_short_decimal: the common case of float_to_string without a
 * snprintf/strtod search (which costs ~1 us a value on the Windows CRT).
 * For |v| in [1e-4, 1e15) — the range printed without an exponent — find
 * the fewest decimals d such that k = round(v * 10^d) gives back v as
 * k / 10^d. With |k| < 2^53 and 10^d exact (d <= 22), that division is
 * correctly rounded, so equality proves "k with d decimals" reads back as
 * v: the text round-trips, and the first d that works is the shortest in
 * this notation. Returns the length written into buf (at least 64 bytes),
 * or 0 when v is outside the fast path and the search must run. */
static int f64_short_decimal(char *buf, double v) {
    static const double pow10[] = {1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8,
                                   1e9, 1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17};
    double a = v < 0 ? -v : v;
    if (!(a >= 1e-4 && a < 1e15)) return 0; /* also NaN, the infinities and zero */
    for (int d = 0; d <= 17; d++) {
        double scaled = a * pow10[d];
        if (scaled >= 9007199254740992.0) return 0; /* k would not be exact */
        /* round the exact a * 10^d, not its rounded product, so the digits
           are the correctly rounded ones printf would give: fma yields the
           product's rounding error exactly, and scaled - floor(scaled) is
           exact below 2^53 */
        double err = fma(a, pow10[d], -scaled);
        double fl = floor(scaled);
        double frac = (scaled - fl) + err;
        double k = frac > 0.5 ? fl + 1 : (frac < 0.5 ? fl : (fmod(fl, 2) == 0 ? fl : fl + 1));
        if (k / pow10[d] == a) {
            uint64_t ik = (uint64_t)k;
            char digits[24];
            int nd = 0;
            do {
                digits[nd++] = (char)('0' + ik % 10);
                ik /= 10;
            } while (ik);
            while (nd <= d) digits[nd++] = '0'; /* at least one digit before the point */
            int n = 0;
            if (v < 0) buf[n++] = '-';
            for (int i = nd - 1; i >= d; i--) buf[n++] = digits[i];
            buf[n++] = '.';
            if (d == 0) {
                buf[n++] = '0';
            } else {
                for (int i = d - 1; i >= 0; i--) buf[n++] = digits[i];
            }
            buf[n] = 0;
            return n;
        }
    }
    return 0;
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
    if (!is_f32) {
        int m = f64_short_decimal(buf, v);
        if (m > 0) {
            from_buf(out, buf, m);
            return;
        }
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
    /* C pads the exponent to two digits (1e-07); print it plain (1e-7) */
    e = strchr(buf, 'e');
    if (e && (e[1] == '+' || e[1] == '-')) {
        char *d = e + 2;
        while (d[0] == '0' && d[1] >= '0' && d[1] <= '9') {
            memmove(d, d + 1, strlen(d + 1) + 1);
            n--;
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

/* veles_parse_f64: the double nearest a decimal number, correctly rounded -
 * strtod is, in the UCRT and in glibc and the BSDs alike, and it is what
 * float_to_string checks its shortest form against, so printing and parsing
 * are exact inverses. The caller (string.toF64) has checked the grammar;
 * out-of-range text gives an infinity or zero, as the value it names. */
double veles_parse_f64(const char *s, int64_t len) {
    /* Clinger's fast path: at most 15 significant digits (so the digits are
       an exact integer m < 2^53) and a power of ten within 10^22 (exact as a
       double) — then m * 10^e or m / 10^-e is one correctly rounded
       operation, the same double strtod gives, for a fraction of its cost */
    {
        static const double pow10[] = {1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11,
                                       1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19, 1e20, 1e21, 1e22};
        int64_t i = 0;
        bool neg = false;
        if (i < len && (s[i] == '-' || s[i] == '+')) neg = s[i++] == '-';
        uint64_t m = 0;
        int sig = 0, frac = 0;
        bool point = false, ok = i < len;
        for (; i < len; i++) {
            char c = s[i];
            if (c == '.' && !point) {
                point = true;
                continue;
            }
            if (c < '0' || c > '9') break;
            if (m == 0 && c == '0') {
                if (point) frac++;
                continue; /* a leading zero is not significant */
            }
            if (++sig > 15) {
                ok = false;
                break;
            }
            m = m * 10 + (uint64_t)(c - '0');
            if (point) frac++;
        }
        int64_t e = 0;
        if (ok && i < len && (s[i] == 'e' || s[i] == 'E')) {
            i++;
            bool eneg = false;
            if (i < len && (s[i] == '-' || s[i] == '+')) eneg = s[i++] == '-';
            for (; i < len && e < 10000; i++) {
                if (s[i] < '0' || s[i] > '9') break;
                e = e * 10 + (s[i] - '0');
            }
            if (eneg) e = -e;
        }
        if (ok && i == len) {
            int64_t p = e - frac;
            double v;
            if (m == 0) {
                v = 0.0;
            } else if (p >= 0 && p <= 22) {
                v = (double)m * pow10[p];
            } else if (p < 0 && p >= -22) {
                v = (double)m / pow10[-p];
            } else {
                goto slow;
            }
            return neg ? -v : v;
        }
    }
slow:;
    char small[128];
    char *buf = len < (int64_t)sizeof small ? small : veles_alloc(len + 1);
    memcpy(buf, s, (size_t)len);
    buf[len] = 0;
    return strtod(buf, NULL);
}
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
    if (len > 0) memcpy(l->data, s, (size_t)len);
    l->len = len;
    return l;
}

/* veles_utf8_valid checks the encoding: shortest forms only, no surrogates,
 * nothing above U+10FFFF (D18: every string is valid UTF-8). */
static bool veles_utf8_valid(const unsigned char *p, int64_t len) {
    int64_t i = 0;
    while (i < len) {
        /* text is mostly ASCII: eight bytes with no high bit at a time */
        while (i + 8 <= len) {
            uint64_t w;
            memcpy(&w, p + i, 8);
            if (w & 0x8080808080808080ull) break;
            i += 8;
        }
        if (i >= len) break;
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
    if (bytes->len > 0) memcpy(buf, bytes->data, (size_t)bytes->len);
    out->data = buf;
    out->len = bytes->len;
    return true;
}

/* veles_bytes_decode_utf8_range: bytes[from:to] as text in one copy (std
 * only: a parser taking a string out of its input); false when the range
 * is out of bounds or not valid UTF-8. */
bool veles_bytes_decode_utf8_range(veles_string *out, veles_list *bytes, int64_t from, int64_t to) {
    if (from < 0 || to < from || to > bytes->len) return false;
    const unsigned char *p = (const unsigned char *)bytes->data + from;
    if (!veles_utf8_valid(p, to - from)) return false;
    char *buf = veles_alloc(to - from + 1);
    if (to > from) memcpy(buf, p, (size_t)(to - from));
    out->data = buf;
    out->len = to - from;
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
    if (cap < 0) cap = 0;
    if (cap > 0 && cap < 4) cap = 4;
    l->elem = veles_desc_size(desc);
    l->desc = desc;
    l->cap = cap;
    l->len = 0;
    /* an empty list owns no storage until the first push (cap 0, data NULL) */
    l->data = cap > 0 ? veles_gc_alloc(desc, l->elem * cap) : NULL;
    return l;
}

int64_t veles_list_len(veles_list *l) {
    return l->len;
}

/* ---- freeze (D188) ----------------------------------------------------- */

static void frozen_panic(const char *what) {
    char msg[96];
    int n = snprintf(msg, sizeof msg, "%s was frozen by freeze() and cannot be changed; copy it with toMutable()", what);
    veles_panic(msg, n);
}

/* the guard of every change to a list through a name that sees it as mutable */
static inline void list_writable(veles_list *l) {
    if (l->flags & LIST_FROZEN) frozen_panic("list");
}

/* veles_list_move moves count elements from index `from` to index `to`, the
 * ranges overlapping as memmove allows: std's insert and removeAt, which
 * check that both lie inside the list. A change of order (D102). */
void veles_list_move(veles_list *l, int64_t from, int64_t to, int64_t count) {
    list_writable(l);
    if (count <= 0 || from == to) return;
    memmove(l->data + to * l->elem, l->data + from * l->elem, (size_t)(count * l->elem));
    l->mods++;
}

/* a write through an element place (set, an element assignment) */
void veles_list_write_check(veles_list *l) {
    list_writable(l);
}

/* a pointer into the list's storage is being made: remember it, so freeze()
 * copies the storage instead of sharing it with the pointer. A write-capable
 * pointer into a frozen list is refused like any other change. */
void veles_list_expose(veles_list *l) {
    list_writable(l);
    l->flags |= LIST_EXPOSED;
}

/* The list becomes immutable in place: the same object, now read-only through
 * every name. Storage nobody holds a pointer into is shared as it is; storage
 * a pointer was made into is copied first, so that pointer can never change
 * the frozen list (it writes into the abandoned buffer, as after a growth). */
veles_list *veles_list_freeze(veles_list *l) {
    list_writable(l);
    if (l->flags & LIST_EXPOSED) {
        char *nd = l->len > 0 ? veles_gc_alloc(l->desc, l->elem * l->len) : NULL;
        if (l->len > 0) memcpy(nd, l->data, (size_t)(l->elem * l->len));
        l->data = nd;
    }
    l->cap = l->len; /* no room left: the inline push falls to the runtime, which panics */
    l->flags |= LIST_FROZEN;
    l->flags &= ~(int64_t)LIST_EXPOSED;
    return l;
}

/* room for at least n elements in all (D83): pushing up to n does not grow
 * the storage again; never shrinks */
void veles_list_reserve(veles_list *l, int64_t n) {
    list_writable(l);
    if (n <= l->cap) return;
    char *nd = veles_gc_alloc(l->desc, l->elem * n);
    if (l->len > 0) memcpy(nd, l->data, (size_t)(l->elem * l->len));
    l->data = nd;
    l->cap = n;
}

void veles_list_push(veles_list *l, const void *item) {
    list_writable(l);
    if (l->len == l->cap) {
        int64_t ncap = l->cap ? l->cap * 2 : 4;
        char *nd = veles_gc_alloc(l->desc, l->elem * ncap);
        if (l->len > 0) memcpy(nd, l->data, (size_t)(l->elem * l->len));
        l->data = nd;
        l->cap = ncap;
    }
    if (l->elem > 0) memcpy(l->data + l->elem * l->len, item, (size_t)l->elem);
    l->len++;
    l->mods++;
}

/* ---- List.sorted() on integers and strings ----------------------------
 * Two equal integers (or byte-equal strings) cannot be told apart, so
 * stability is not observable and the natural order needs no comparator
 * call: the prelude's merge sort pays an indirect call per comparison.
 * The same algorithm here: insertion-sorted runs of 32, then bottom-up
 * merges through a scratch buffer. No GC allocation, so no collection
 * can run while the elements are in the scratch buffer. */

#define SORT_RUN 32

#define DEFINE_SORT(NAME, T, LESS)                                            \
static void NAME(T *a, int64_t n) {                                           \
    for (int64_t s = 0; s < n; s += SORT_RUN) {                               \
        int64_t e = s + SORT_RUN < n ? s + SORT_RUN : n;                      \
        for (int64_t i = s + 1; i < e; i++) {                                 \
            T x = a[i];                                                       \
            int64_t j = i;                                                    \
            while (j > s && LESS(x, a[j - 1])) { a[j] = a[j - 1]; j--; }      \
            a[j] = x;                                                         \
        }                                                                     \
    }                                                                         \
    if (n <= SORT_RUN) return;                                                \
    T *buf = malloc(sizeof(T) * (size_t)n);                                   \
    if (!buf) { fputs("panic: out of memory\n", stderr); exit(101); }         \
    T *src = a, *dst = buf;                                                   \
    for (int64_t w = SORT_RUN; w < n; w *= 2) {                               \
        for (int64_t lo = 0; lo < n; lo += 2 * w) {                           \
            int64_t mid = lo + w < n ? lo + w : n;                            \
            int64_t hi = lo + 2 * w < n ? lo + 2 * w : n;                     \
            int64_t i = lo, j = mid, k = lo;                                  \
            while (i < mid && j < hi) dst[k++] = LESS(src[j], src[i]) ? src[j++] : src[i++]; \
            while (i < mid) dst[k++] = src[i++];                              \
            while (j < hi) dst[k++] = src[j++];                               \
        }                                                                     \
        T *t = src; src = dst; dst = t;                                       \
    }                                                                         \
    if (src != a) memcpy(a, src, sizeof(T) * (size_t)n);                      \
    free(buf);                                                                \
}

#define NUM_LESS(x, y) ((x) < (y))
#define STR_LESS(x, y) (veles_string_cmp((x).data, (x).len, (y).data, (y).len) < 0)
DEFINE_SORT(sort_i8, int8_t, NUM_LESS)
DEFINE_SORT(sort_i16, int16_t, NUM_LESS)
DEFINE_SORT(sort_i32, int32_t, NUM_LESS)
DEFINE_SORT(sort_i64, int64_t, NUM_LESS)
DEFINE_SORT(sort_u8, uint8_t, NUM_LESS)
DEFINE_SORT(sort_u16, uint16_t, NUM_LESS)
DEFINE_SORT(sort_u32, uint32_t, NUM_LESS)
DEFINE_SORT(sort_u64, uint64_t, NUM_LESS)
DEFINE_SORT(sort_str, veles_string, STR_LESS)

/* kind: 1–4 signed 8/16/32/64 bits, 5–8 unsigned, 9 string */
void veles_list_sort_native(veles_list *l, int32_t kind) {
    void *d = l->data;
    switch (kind) {
    case 1: sort_i8(d, l->len); break;
    case 2: sort_i16(d, l->len); break;
    case 3: sort_i32(d, l->len); break;
    case 4: sort_i64(d, l->len); break;
    case 5: sort_u8(d, l->len); break;
    case 6: sort_u16(d, l->len); break;
    case 7: sort_u32(d, l->len); break;
    case 8: sort_u64(d, l->len); break;
    case 9: sort_str(d, l->len); break;
    }
}

/* the strings of a list joined with sep, one allocation (List.join) */
void veles_string_join(veles_string *out, veles_list *l, const char *sep, int64_t seplen) {
    const veles_string *parts = (const veles_string *)l->data;
    int64_t len = l->len > 0 ? seplen * (l->len - 1) : 0;
    for (int64_t i = 0; i < l->len; i++) len += parts[i].len;
    char *buf = veles_alloc(len + 1);
    char *p = buf;
    for (int64_t i = 0; i < l->len; i++) {
        if (i > 0 && seplen) {
            memcpy(p, sep, (size_t)seplen);
            p += seplen;
        }
        if (parts[i].len) memcpy(p, parts[i].data, (size_t)parts[i].len);
        p += parts[i].len;
    }
    out->data = buf;
    out->len = len;
}

/* In-place text edits on a list of u8, a StringBuilder's (D207); ASCII
 * whitespace and case, as the string methods. */
static inline bool ascii_space(unsigned char c) { return c == ' ' || (c >= 9 && c <= 13); }

void veles_bytes_trim(veles_list *l) {
    list_writable(l);
    int64_t i = 0, j = l->len;
    while (i < j && ascii_space((unsigned char)l->data[i])) i++;
    while (j > i && ascii_space((unsigned char)l->data[j - 1])) j--;
    if (i == 0 && j == l->len) return;
    if (i > 0 && j > i) memmove(l->data, l->data + i, (size_t)(j - i));
    l->len = j - i;
    l->mods++;
}

void veles_bytes_ascii_case(veles_list *l, bool upper) {
    list_writable(l);
    unsigned char lo = upper ? 'a' : 'A', hi = upper ? 'z' : 'Z';
    for (int64_t i = 0; i < l->len; i++) {
        unsigned char c = (unsigned char)l->data[i];
        if (c >= lo && c <= hi) l->data[i] = (char)(c ^ 0x20);
    }
}

/* every occurrence of a non-empty old as repl: in place when repl is no
 * longer than old, else into one new buffer of the final size */
void veles_bytes_replace(veles_list *l, const char *old, int64_t olen, const char *repl, int64_t rlen) {
    list_writable(l);
    char *s = l->data;
    int64_t len = l->len, count = 0;
    for (int64_t i = 0; olen > 0 && i + olen <= len;) {
        if (memcmp(s + i, old, (size_t)olen) == 0) { count++; i += olen; } else i++;
    }
    if (count == 0) return;
    int64_t n = len + count * (rlen - olen);
    char *out = s;
    if (rlen > olen) {
        int64_t ncap = n < 16 ? 16 : n;
        out = veles_gc_alloc(l->desc, ncap);
        l->cap = ncap;
    }
    int64_t w = 0, i = 0;
    while (i < len) {
        if (i + olen <= len && memcmp(s + i, old, (size_t)olen) == 0) {
            memcpy(out + w, repl, (size_t)rlen);
            w += rlen;
            i += olen;
        } else {
            out[w++] = s[i++];
        }
    }
    l->data = out;
    l->len = n;
    if (n != len) l->mods++;
}

/* n bytes at index at, which std checked is in 0..len on a character edge */
void veles_bytes_insert(veles_list *l, int64_t at, const char *p, int64_t n) {
    list_writable(l);
    if (n == 0) return;
    if (l->len + n > l->cap) {
        int64_t ncap = l->cap * 2;
        if (ncap < l->len + n) ncap = l->len + n;
        if (ncap < 16) ncap = 16;
        char *nd = veles_gc_alloc(l->desc, ncap);
        if (l->len > 0) memcpy(nd, l->data, (size_t)l->len);
        l->data = nd;
        l->cap = ncap;
    }
    memmove(l->data + at + n, l->data + at, (size_t)(l->len - at));
    memcpy(l->data + at, p, (size_t)n);
    l->len += n;
    l->mods++;
}

/* the bytes as text without a copy (`edit` on a string, D207): the string
 * shares the storage and the list is frozen, so nothing can change it again;
 * storage a pointer was made into is copied first, as by freeze. False
 * when the bytes are not UTF-8. */
bool veles_bytes_take_text(veles_string *out, veles_list *l) {
    list_writable(l);
    if (!veles_utf8_valid((const unsigned char *)l->data, l->len)) return false;
    if (l->flags & LIST_EXPOSED) {
        char *buf = veles_alloc(l->len + 1);
        if (l->len > 0) memcpy(buf, l->data, (size_t)l->len);
        out->data = buf;
    } else {
        out->data = l->data;
    }
    out->len = l->len;
    l->cap = l->len;
    l->flags |= LIST_FROZEN;
    l->flags &= ~(int64_t)LIST_EXPOSED;
    return true;
}

/* appends n bytes to a list of u8 (StringBuilder.append) */
void veles_list_append_bytes(veles_list *l, const char *p, int64_t n) {
    list_writable(l);
    if (l->len + n > l->cap) {
        int64_t ncap = l->cap * 2;
        if (ncap < l->len + n) ncap = l->len + n;
        if (ncap < 16) ncap = 16; /* a builder's first piece: room for a few more */
        char *nd = veles_gc_alloc(l->desc, ncap);
        if (l->len > 0) memcpy(nd, l->data, (size_t)l->len);
        l->data = nd;
        l->cap = ncap;
    }
    if (n) memcpy(l->data + l->len, p, (size_t)n);
    l->len += n;
    l->mods++;
}

void *veles_list_ref(veles_list *l, int64_t i) {
    if (i < 0 || i >= l->len) {
        char msg[80];
        int n = snprintf(msg, sizeof msg, "index %" PRId64 " out of bounds for list of length %" PRId64, i, l->len);
        veles_panic(msg, n);
    }
    return l->data + l->elem * i;
}

/* veles_list_index_panic is the out-of-range branch of an inlined element
 * access: the same message as veles_list_ref, with the site's location. */
void veles_list_index_panic(veles_list *l, int64_t i, const char *loc, int64_t loc_len) {
    char msg[80];
    int n = snprintf(msg, sizeof msg, "index %" PRId64 " out of bounds for list of length %" PRId64, i, l->len);
    veles_panic_at(msg, n, loc, loc_len);
}

/* veles_array_index_panic is the out-of-range branch of an inline array's
 * element access (D121): the list's message, for an array. */
void veles_array_index_panic(int64_t i, int64_t len, const char *loc, int64_t loc_len) {
    char msg[80];
    int n = snprintf(msg, sizeof msg, "index %" PRId64 " out of bounds for array of length %" PRId64, i, len);
    veles_panic_at(msg, n, loc, loc_len);
}

/* ---- Secret (D112) -----------------------------------------------------
 * A Secret's bytes are a List<u8> whose storage has the wipe descriptor:
 * the collector zeroes it when it is freed, and close() zeroes it at once.
 * Nothing grows the list, so its bytes are never copied elsewhere. */
extern veles_desc veles_wipe_u8_desc;
void veles_wipe(void *p, size_t n);

veles_list *veles_secret_bytes(const char *p, int64_t len) {
    veles_list *l = veles_list_new(&veles_wipe_u8_desc, len);
    if (len > 0) memcpy(l->data, p, (size_t)len);
    l->len = len;
    return l;
}

void veles_secret_wipe(veles_list *l) {
    if (l->cap > 0) veles_wipe(l->data, (size_t)(l->cap * l->elem));
}

/* expose(): an ordinary copy, as text (the bytes came from a string, so
 * they are valid UTF-8) or as a List<u8> of the given element descriptor */
void veles_secret_text(veles_string *out, veles_list *l) {
    char *buf = veles_alloc(l->len + 1);
    if (l->len > 0) memcpy(buf, l->data, (size_t)l->len);
    out->data = buf;
    out->len = l->len;
}

veles_list *veles_secret_copy(veles_desc *desc, veles_list *l) {
    veles_list *c = veles_list_new(desc, l->len);
    if (l->len > 0) memcpy(c->data, l->data, (size_t)l->len);
    c->len = l->len;
    return c;
}

/* veles_unchecked_index_panic is D114's debug assertion for atUnchecked,
 * setUnchecked (kind 0, a list) and byteAtUnchecked (kind 1, a string). */
void veles_unchecked_index_panic(int64_t i, int64_t len, int64_t kind, const char *loc, int64_t loc_len) {
    char msg[96];
    int n = snprintf(msg, sizeof msg, "unchecked index %" PRId64 " out of bounds for %s of length %" PRId64,
                     i, kind ? "string" : "list", len);
    veles_panic_at(msg, n, loc, loc_len);
}

bool veles_list_pop(veles_list *l, void *out) {
    list_writable(l);
    if (l->len == 0) return false;
    l->len--;
    l->mods++;
    if (l->elem > 0) memcpy(out, l->data + l->elem * l->len, (size_t)l->elem);
    return true;
}

veles_list *veles_list_copy(veles_list *l) {
    veles_list *c = veles_list_new(l->desc, l->cap);
    if (l->len > 0) memcpy(c->data, l->data, (size_t)(l->elem * l->len));
    c->len = l->len;
    return c;
}

/* The elements in from..<to as a new list, both ends clamped to the list;
 * `..rest` in a list pattern (D62). */
veles_list *veles_list_slice(veles_list *l, int64_t from, int64_t to) {
    if (from < 0) from = 0;
    if (to > l->len) to = l->len;
    if (from > to) from = to;
    int64_t n = to - from;
    veles_list *c = veles_list_new(l->desc, n);
    if (n > 0) memcpy(c->data, l->data + l->elem * from, (size_t)(l->elem * n));
    c->len = n;
    return c;
}

/* The same elements as a window onto l's storage, not a copy (D196): for a
 * receiver that is an immutable List, which nothing can write through, so the
 * sharing cannot be seen. The new header points into the middle of l's buffer
 * (the collector follows an interior pointer, D10), which it keeps alive; an
 * empty window holds on to nothing. */
veles_list *veles_list_view(veles_list *l, int64_t from, int64_t to) {
    if (from < 0) from = 0;
    if (to > l->len) to = l->len;
    if (from > to) from = to;
    int64_t n = to - from;
    veles_list *c = veles_list_new(l->desc, 0);
    if (n > 0) {
        c->data = l->data + l->elem * from;
        c->len = n;
        c->cap = n;
    }
    return c;
}

/* appends every element of src to dst (MutableList.addAll); src may be dst */
/* `race { }` (D169): a race whose sources are the tasks of a list, one per
 * element in list order, so the winner's index is its place in the list
 * (veles_task.c owns the race; a handle is a task pointer) */
void *veles_race_new(void *t, int64_t narms);
void veles_race_await(void *r, void *t);

void *veles_race_list(void *t, veles_list *tasks) {
    void *r = veles_race_new(t, tasks->len);
    void **handles = (void **)tasks->data;
    for (int64_t i = 0; i < tasks->len; i++) veles_race_await(r, handles[i]);
    return r;
}

void veles_list_append_list(veles_list *dst, veles_list *src) {
    list_writable(dst);
    int64_t n = src->len;
    if (n == 0) return;
    if (dst->len + n > dst->cap) {
        int64_t ncap = dst->cap * 2;
        if (ncap < dst->len + n) ncap = dst->len + n;
        char *nd = veles_gc_alloc(dst->desc, dst->elem * ncap);
        if (dst->len > 0) memcpy(nd, dst->data, (size_t)(dst->elem * dst->len));
        dst->data = nd;
        dst->cap = ncap;
    }
    memmove(dst->data + dst->elem * dst->len, src->data, (size_t)(src->elem * n));
    dst->len += n;
    dst->mods++;
}

void veles_list_clear(veles_list *l) {
    list_writable(l);
    l->len = 0;
    l->mods++;
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
    int64_t mods; /* D102: entries added or removed so far */
    int64_t flags; /* LIST_EXPOSED / LIST_FROZEN (D188) */
} veles_map;

/* Hashing is keyed with a secret drawn from the operating system at start-up,
 * so keys cannot be chosen ahead of time to collide: a JSON object, a form or
 * a header block of colliding names would make every insert probe all the
 * others, quadratic time from one request (hash flooding, "HashDoS"). Go's
 * runtime and Rust's HashMap seed the same way. Nothing a program sees
 * depends on the hashes — maps keep insertion order (D25) and `hash()` of a
 * built-in type cannot be called — so only timing changes from run to run.
 * The function is Go's memhash fallback (wyhash's structure): a 64×64→128
 * multiply folded to 64 bits per 16 input bytes. */
static uint64_t hash_key[4];

int64_t veles_random_bytes(int64_t n, veles_string *out); /* veles_os.c */

#define HASH_M1 0xa0761d6478bd642fULL
#define HASH_M2 0xe7037ed1a0b428dbULL
#define HASH_M3 0x8ebc6af09c88c6e3ULL
#define HASH_M4 0x589965cc75374cc3ULL
#define HASH_M5 0x1d8e4e27c47d124fULL

static inline uint64_t hash_mum(uint64_t a, uint64_t b) {
    __uint128_t r = (__uint128_t)a * b;
    return (uint64_t)r ^ (uint64_t)(r >> 64);
}

static inline uint64_t hash_r8(const unsigned char *p) {
    uint64_t v;
    memcpy(&v, p, 8);
    return v;
}

static inline uint64_t hash_r4(const unsigned char *p) {
    uint32_t v;
    memcpy(&v, p, 4);
    return v;
}

/* The key, from the system's secure generator. Without one (it has never been
 * seen to fail) the clock and addresses ASLR placed stand in: weaker, but a
 * program that cannot hash cannot run at all. */
void veles_hash_init(void) {
    veles_string s;
    if (veles_random_bytes((int64_t)sizeof hash_key, &s) == 0) {
        memcpy(hash_key, s.data, sizeof hash_key);
        return;
    }
    uint64_t x = (uint64_t)time(NULL) ^ (uint64_t)(uintptr_t)&s ^ ((uint64_t)(uintptr_t)hash_key << 17) ^ (uint64_t)clock();
    for (int i = 0; i < 4; i++) {
        x += 0x9e3779b97f4a7c15ULL; /* splitmix64 */
        uint64_t z = x;
        z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9ULL;
        z = (z ^ (z >> 27)) * 0x94d049bb133111ebULL;
        hash_key[i] = z ^ (z >> 31);
    }
}

int64_t veles_hash_bytes(const char *src, int64_t len) {
    const unsigned char *p = (const unsigned char *)src;
    uint64_t s = (uint64_t)len, a, b;
    uint64_t seed = hash_key[0] ^ HASH_M1;
    if (s == 0) return (int64_t)seed;
    if (s < 4) {
        a = (uint64_t)p[0] | (uint64_t)p[s >> 1] << 8 | (uint64_t)p[s - 1] << 16;
        b = 0;
    } else if (s == 4) {
        a = b = hash_r4(p);
    } else if (s < 8) {
        a = hash_r4(p);
        b = hash_r4(p + s - 4);
    } else if (s == 8) {
        a = b = hash_r8(p);
    } else if (s <= 16) {
        a = hash_r8(p);
        b = hash_r8(p + s - 8);
    } else {
        uint64_t l = s;
        if (l > 48) {
            uint64_t seed1 = seed, seed2 = seed;
            for (; l > 48; l -= 48) {
                seed = hash_mum(hash_r8(p) ^ hash_key[1] ^ HASH_M2, hash_r8(p + 8) ^ seed);
                seed1 = hash_mum(hash_r8(p + 16) ^ hash_key[2] ^ HASH_M3, hash_r8(p + 24) ^ seed1);
                seed2 = hash_mum(hash_r8(p + 32) ^ hash_key[3] ^ HASH_M4, hash_r8(p + 40) ^ seed2);
                p += 48;
            }
            seed ^= seed1 ^ seed2;
        }
        for (; l > 16; l -= 16) {
            seed = hash_mum(hash_r8(p) ^ hash_key[1] ^ HASH_M2, hash_r8(p + 8) ^ seed);
            p += 16;
        }
        /* the last 16 bytes, overlapping what the loops read: s > 16 */
        a = hash_r8(p + l - 16);
        b = hash_r8(p + l - 8);
    }
    return (int64_t)hash_mum(HASH_M5 ^ s, hash_mum(a ^ hash_key[1] ^ HASH_M2, b ^ seed));
}

/* Combines a running hash with the next part's (tuples, structs, lists,
 * variants). Keyed as well: an unkeyed combine can be solved for a second
 * part that cancels the first, so (a, b) keys would collide at will. */
int64_t veles_hash_mix(int64_t a, int64_t b) {
    return (int64_t)hash_mum((uint64_t)a ^ hash_key[2] ^ HASH_M3, (uint64_t)b ^ hash_key[3] ^ HASH_M4);
}

/* A hash as a map keeps it. An integer key's hash is the integer itself, and
 * a user `hash()` is whatever it returns, so it is scrambled with the key too
 * (Go's memhash64): keys that differ only above the index's bits — multiples
 * of 1024, say — would otherwise all start probing at one slot. The public
 * entry points (find, insert, remove, const_hash) scramble the hash they are
 * given, once; the metadata holds the scrambled value, so a rebuild or a copy
 * reuses it and everything below takes it as it is. */
static inline uint64_t map_slot(int64_t hash) {
    uint64_t h = (uint64_t)hash;
    return hash_mum(HASH_M5 ^ 8, hash_mum(h ^ hash_key[1] ^ HASH_M2, h ^ hash_key[0] ^ HASH_M1));
}

/* Test hooks (driver TestHashFlooding). The longest probe run that n integer
 * keys i * stride make in an index sized as a map sizes it: under the
 * unkeyed slot (hash & mask) a stride of 2^20 put every key on one slot. */
int64_t veles_hash_test_probes(int64_t n, int64_t stride) {
    int64_t icap = 16;
    while (n * 2 >= icap) icap *= 2;
    bool *used = calloc((size_t)icap, 1);
    uint64_t mask = (uint64_t)icap - 1;
    int64_t worst = 0;
    for (int64_t k = 0; k < n; k++) {
        uint64_t i = map_slot((int64_t)((uint64_t)k * (uint64_t)stride)) & mask;
        int64_t run = 0;
        while (used[i]) i = (i + 1) & mask, run++;
        used[i] = true;
        if (run > worst) worst = run;
    }
    free(used);
    return worst;
}

/* the hash of one string, which differs from run to run */
int64_t veles_hash_test_string(void) {
    return veles_hash_bytes("veles", 5);
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

/* A constant Map or Set (D113) is laid out by the compiler with its entries,
 * but its hashes depend on this run's key: veles_init_const_maps stores each
 * entry's hash, then this builds the index over them. */
void veles_map_const_hash(veles_map *m, int64_t e, int64_t hash) {
    m->meta[e].hash = (int64_t)map_slot(hash);
}

void veles_map_const_index(veles_map *m) {
    for (int64_t e = 0; e < m->used; e++) map_index_insert(m, m->meta[e].hash, e);
}

veles_map *veles_map_new(veles_desc *keyDesc, veles_desc *valDesc) {
    veles_map *m = veles_alloc_words(sizeof *m);
    m->keyDesc = keyDesc;
    m->valDesc = valDesc;
    m->keySize = veles_desc_size(keyDesc);
    m->valSize = valDesc ? veles_desc_size(valDesc) : 0;
    /* an empty map owns no storage (cap 0, icap 0, every array NULL) until
     * the first insert: map_start allocates it */
    return m;
}

/* A map of at most MAP_SMALL entries has no hash index (icap 0): a lookup
 * scans the stored hashes, which is as fast as probing for that few, and the
 * first insert allocates three arrays instead of four. Once the entry arrays
 * are larger than that the index is built and kept (icap > 0). */
#define MAP_SMALL 8

/* a set has no values: its value array is this, never written (valSize 0) */
static char map_no_vals[16];

/* the storage of a map that has none yet: room for MAP_SMALL entries */
static void map_start(veles_map *m) {
    m->cap = MAP_SMALL;
    m->keys = veles_gc_alloc(m->keyDesc, m->keySize * m->cap + 1);
    m->vals = m->valDesc ? veles_gc_alloc(m->valDesc, m->valSize * m->cap + 1) : map_no_vals;
    m->meta = veles_alloc((int64_t)sizeof(veles_meta) * m->cap);
}

/* an index with at least half its slots free for the entries so far plus one */
static void map_build_index(veles_map *m) {
    int64_t icap = 16;
    while ((m->used + 1) * 2 > icap) icap *= 2;
    map_rebuild(m, icap);
}

int64_t veles_map_len(veles_map *m) {
    return m->len;
}

/* ---- freeze (D188), as for lists ---------------------------------------- */

static inline void map_writable(veles_map *m) {
    if (m->flags & LIST_FROZEN) frozen_panic(m->valDesc ? "map" : "set");
}

/* a pointer into the entries is being made (m.ref) */
void veles_map_expose(veles_map *m) {
    map_writable(m);
    m->flags |= LIST_EXPOSED;
}

/* The map becomes immutable in place. Entry arrays a pointer was made into are
 * copied first (the index is rebuilt over the copy), so that pointer can never
 * change the frozen map. */
veles_map *veles_map_freeze(veles_map *m) {
    map_writable(m);
    if ((m->flags & LIST_EXPOSED) && m->cap > 0) {
        char *nk = veles_gc_alloc(m->keyDesc, m->keySize * m->cap + 1);
        char *nv = m->valDesc ? veles_gc_alloc(m->valDesc, m->valSize * m->cap + 1) : map_no_vals;
        veles_meta *nm = veles_alloc((int64_t)sizeof(veles_meta) * m->cap);
        memcpy(nk, m->keys, (size_t)(m->keySize * m->used));
        if (m->valSize) memcpy(nv, m->vals, (size_t)(m->valSize * m->used));
        memcpy(nm, m->meta, sizeof(veles_meta) * (size_t)m->used);
        m->keys = nk;
        m->vals = nv;
        m->meta = nm;
        if (m->icap > 0) map_rebuild(m, m->icap);
    }
    m->flags |= LIST_FROZEN;
    m->flags &= ~(int64_t)LIST_EXPOSED;
    return m;
}

int64_t veles_map_mods(veles_map *m) {
    return m->mods;
}

/* hash is scrambled (map_slot) */
static int64_t map_find(veles_map *m, int64_t hash, const void *key, veles_eq_fn eq) {
    if (m->icap == 0) { /* small: scan the entries */
        for (int64_t e = 0; e < m->used; e++) {
            if (m->meta[e].live && m->meta[e].hash == hash && (m->keySize == 0 || eq(key, m->keys + e * m->keySize))) return e;
        }
        return -1;
    }
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

int64_t veles_map_find(veles_map *m, int64_t hash, const void *key, veles_eq_fn eq) {
    return map_find(m, (int64_t)map_slot(hash), key, eq);
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
    if (m->icap > 0) map_rebuild(m, m->icap);
}

/* append an entry known to be absent; returns its index */
static int64_t map_append(veles_map *m, int64_t hash, const void *key, const void *val) {
    if (m->cap == 0) map_start(m);
    if (m->used == m->cap) {
        if (m->used > 2 * m->len + 8) {
            map_compact(m);
        } else {
            int64_t ncap = m->cap * 2;
            char *nk = veles_gc_alloc(m->keyDesc, m->keySize * ncap + 1);
            char *nv = m->valDesc ? veles_gc_alloc(m->valDesc, m->valSize * ncap + 1) : map_no_vals;
            veles_meta *nm = veles_alloc((int64_t)sizeof(veles_meta) * ncap);
            memcpy(nk, m->keys, (size_t)(m->keySize * m->used));
            if (m->valSize) memcpy(nv, m->vals, (size_t)(m->valSize * m->used));
            memcpy(nm, m->meta, sizeof(veles_meta) * (size_t)m->used);
            m->keys = nk;
            m->vals = nv;
            m->meta = nm;
            m->cap = ncap;
        }
    }
    if (m->icap == 0) {
        if (m->cap > MAP_SMALL) map_build_index(m); /* grown past small: index every entry */
    } else if (m->used * 2 >= m->icap) {
        map_rebuild(m, m->icap * 2);
    }
    int64_t e = m->used++;
    if (m->keySize) memcpy(m->keys + e * m->keySize, key, (size_t)m->keySize);
    if (m->valSize) memcpy(m->vals + e * m->valSize, val, (size_t)m->valSize);
    m->meta[e].hash = hash;
    m->meta[e].live = true;
    m->len++;
    m->mods++;
    if (m->icap > 0) map_index_insert(m, hash, e);
    return e;
}

/* insert or overwrite; returns the entry index */
int64_t veles_map_insert(veles_map *m, int64_t hash, const void *key, const void *val, veles_eq_fn eq) {
    map_writable(m);
    hash = (int64_t)map_slot(hash);
    int64_t e = map_find(m, hash, key, eq);
    if (e >= 0) {
        if (m->valSize) memcpy(m->vals + e * m->valSize, val, (size_t)m->valSize);
        return e;
    }
    return map_append(m, hash, key, val);
}

bool veles_map_remove(veles_map *m, int64_t hash, const void *key, veles_eq_fn eq) {
    map_writable(m);
    hash = (int64_t)map_slot(hash);
    if (m->icap == 0) { /* small: no index to update */
        int64_t e = map_find(m, hash, key, eq);
        if (e < 0) return false;
        m->meta[e].live = false;
        m->len--;
        m->mods++;
        return true;
    }
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
                m->mods++;
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
    map_writable(m);
    m->len = 0;
    m->mods++;
    m->used = 0;
    if (m->icap > 0) map_rebuild(m, m->icap);
}

/* D105: room for n live entries in all, so that inserting until there are
 * n neither grows the arrays nor rebuilds the index (map_append's two
 * conditions). Never shrinks; the contents and their order stay. */
void veles_map_reserve(veles_map *m, int64_t n) {
    map_writable(m);
    if (n <= 0) return;
    if (m->cap == 0) map_start(m);
    if (m->used > m->len) map_compact(m); /* dead entries would count against n */
    if (n > m->cap) {
        char *nk = veles_gc_alloc(m->keyDesc, m->keySize * n + 1);
        char *nv = m->valDesc ? veles_gc_alloc(m->valDesc, m->valSize * n + 1) : map_no_vals;
        veles_meta *nm = veles_alloc((int64_t)sizeof(veles_meta) * n);
        memcpy(nk, m->keys, (size_t)(m->keySize * m->used));
        if (m->valSize) memcpy(nv, m->vals, (size_t)(m->valSize * m->used));
        memcpy(nm, m->meta, sizeof(veles_meta) * (size_t)m->used);
        m->keys = nk;
        m->vals = nv;
        m->meta = nm;
        m->cap = n;
    }
    /* an append rebuilds once used * 2 reaches icap: keep (n - 1) * 2 below
     * it. A map that stays small needs no index at all. */
    if (m->icap == 0 && m->cap <= MAP_SMALL) return;
    int64_t icap = m->icap ? m->icap : 16;
    while ((n - 1) * 2 >= icap) icap *= 2;
    if (icap != m->icap) map_rebuild(m, icap);
}

veles_map *veles_map_copy(veles_map *m) {
    veles_map *c = veles_map_new(m->keyDesc, m->valDesc);
    for (int64_t e = 0; e < m->used; e++) {
        if (m->meta[e].live) {
            map_append(c, m->meta[e].hash, m->keys + e * m->keySize, m->vals + e * m->valSize);
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
        int n = snprintf(msg, sizeof msg, "negative exponent %" PRId64 " in integer pow", exp);
        if (n >= (int)sizeof msg) n = (int)sizeof msg - 1;
        veles_panic_at(msg, n, where, where_len);
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
        int n = snprintf(msg, sizeof msg, "integer overflow in pow");
        if (n >= (int)sizeof msg) n = (int)sizeof msg - 1;
        veles_panic_at(msg, n, where, where_len);
    }
    return result;
}
