/*
 * Veles bootstrap runtime — operating system bindings for std/os, std/fs
 * and std/path.
 *
 * The same conventions as veles_rt.c: strings arrive as (data, len) pairs
 * and leave through a veles_string out-pointer; results that can fail
 * return 0 or an errno value, which the Veles side turns into an IoError.
 * Nothing here is reachable except through `extern "C"` declarations in
 * the standard library.
 */
#include <errno.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <dirent.h>

#if defined(_WIN32)
#include <windows.h>
#include <direct.h>
#include <io.h>
#else
#include <unistd.h>
#include <sys/wait.h>
#endif

typedef struct {
    const char *data;
    int64_t len;
} veles_string;

void *veles_alloc(int64_t size);
void veles_panic(const char *msg, int64_t len);

/* cstr copies a Veles string into a NUL-terminated buffer the C library
 * can take; Veles strings are not NUL-terminated. */
static char *cstr(const char *s, int64_t len) __attribute__((unused));
static char *cstr(const char *s, int64_t len) {
    char *buf = veles_alloc(len + 1);
    memcpy(buf, s, (size_t)len);
    buf[len] = 0;
    return buf;
}

static void set_string(veles_string *out, const char *s, int64_t len) {
    char *buf = veles_alloc(len + 1);
    memcpy(buf, s, (size_t)len);
    buf[len] = 0;
    out->data = buf;
    out->len = len;
}

#if defined(_WIN32)
/* Windows: paths, arguments, environment and directory names go through
 * the UTF-16 API so that non-ASCII text survives (D18: every Veles string
 * is UTF-8; the "ANSI" functions would use the legacy code page). */
static wchar_t *wstr(const char *s, int64_t len) {
    int n = MultiByteToWideChar(CP_UTF8, 0, s, (int)len, NULL, 0);
    wchar_t *w = veles_alloc(((int64_t)n + 1) * (int64_t)sizeof(wchar_t));
    MultiByteToWideChar(CP_UTF8, 0, s, (int)len, w, n);
    w[n] = 0;
    return w;
}

static void set_wstring(veles_string *out, const wchar_t *w) {
    int n = WideCharToMultiByte(CP_UTF8, 0, w, -1, NULL, 0, NULL, NULL); /* counts the NUL */
    if (n <= 0) { set_string(out, "", 0); return; }
    char *buf = veles_alloc(n);
    WideCharToMultiByte(CP_UTF8, 0, w, -1, buf, n, NULL, NULL);
    out->data = buf;
    out->len = n - 1;
}

static FILE *open_file(const char *path, int64_t plen, const char *mode) {
    wchar_t wmode[8];
    int i = 0;
    for (; mode[i] && i < 7; i++) wmode[i] = (wchar_t)mode[i];
    wmode[i] = 0;
    return _wfopen(wstr(path, plen), wmode);
}
#else
static FILE *open_file(const char *path, int64_t plen, const char *mode) {
    return fopen(cstr(path, plen), mode);
}
#endif

/* growing byte buffer for reads */
typedef struct {
    char *data;
    int64_t len, cap;
} buf_t;

static void buf_push(buf_t *b, const char *p, int64_t n) {
    if (b->len + n > b->cap) {
        int64_t ncap = b->cap ? b->cap * 2 : 4096;
        while (ncap < b->len + n) ncap *= 2;
        char *nd = veles_alloc(ncap);
        if (b->len) memcpy(nd, b->data, (size_t)b->len);
        b->data = nd;
        b->cap = ncap;
    }
    memcpy(b->data + b->len, p, (size_t)n);
    b->len += n;
}

static int read_stream(FILE *f, buf_t *b) {
    char chunk[8192];
    size_t n;
    while ((n = fread(chunk, 1, sizeof chunk, f)) > 0) buf_push(b, chunk, (int64_t)n);
    return ferror(f) ? (errno ? errno : EIO) : 0;
}

/* ---- errors ------------------------------------------------------------ */

void veles_os_strerror(int64_t code, veles_string *out) {
#if defined(_WIN32)
    if (code >= 10000) {
        /* a Winsock code (std/net): the system's text, without its trailing
         * newline and full stop */
        wchar_t *msg = NULL;
        DWORD n = FormatMessageW(FORMAT_MESSAGE_ALLOCATE_BUFFER | FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS,
                                 NULL, (DWORD)code, 0, (LPWSTR)&msg, 0, NULL);
        if (n > 0 && msg) {
            while (n > 0 && (msg[n - 1] == L'\r' || msg[n - 1] == L'\n' || msg[n - 1] == L' ' || msg[n - 1] == L'.')) msg[--n] = 0;
            set_wstring(out, msg);
            LocalFree(msg);
            return;
        }
    }
#endif
    const char *s = strerror((int)code);
    set_string(out, s, (int64_t)strlen(s));
}

/* ---- process ----------------------------------------------------------- */

extern int veles_os_argc_value(void);
extern char **veles_os_argv_value(void);

#if defined(_WIN32)
static wchar_t **wide_argv(int *argc) {
    static wchar_t **cached;
    static int cached_argc;
    if (!cached) {
        cached = CommandLineToArgvW(GetCommandLineW(), &cached_argc);
        if (!cached) cached_argc = 0;
    }
    *argc = cached_argc;
    return cached;
}

int64_t veles_os_argc(void) {
    int argc;
    wide_argv(&argc);
    return argc ? argc : veles_os_argc_value();
}

void veles_os_arg(int64_t i, veles_string *out) {
    int argc;
    wchar_t **argv = wide_argv(&argc);
    if (!argv || argc == 0) {
        char **a = veles_os_argv_value();
        argc = veles_os_argc_value();
        if (i < 0 || i >= argc) veles_panic("os.arg: index out of range", 26);
        set_string(out, a[i], (int64_t)strlen(a[i]));
        return;
    }
    if (i < 0 || i >= argc) veles_panic("os.arg: index out of range", 26);
    set_wstring(out, argv[i]);
}
#else
int64_t veles_os_argc(void) { return veles_os_argc_value(); }

void veles_os_arg(int64_t i, veles_string *out) {
    char **argv = veles_os_argv_value();
    int argc = veles_os_argc_value();
    if (i < 0 || i >= argc) {
        veles_panic("os.arg: index out of range", 26);
    }
    set_string(out, argv[i], (int64_t)strlen(argv[i]));
}
#endif

bool veles_os_getenv(const char *name, int64_t nlen, veles_string *out) {
#if defined(_WIN32)
    const wchar_t *v = _wgetenv(wstr(name, nlen));
    if (!v) return false;
    set_wstring(out, v);
    return true;
#else
    const char *v = getenv(cstr(name, nlen));
    if (!v) return false;
    set_string(out, v, (int64_t)strlen(v));
    return true;
#endif
}

void veles_os_exit(int64_t code) {
    fflush(stdout);
    fflush(stderr);
    exit((int)code);
}

/* veles_os_run runs a shell command line, capturing its standard output;
 * standard error is inherited. Returns the exit status, or -1 with errno
 * set when the process could not be started. */
int64_t veles_os_run(const char *cmd, int64_t clen, veles_string *out, int64_t *err) {
    fflush(stdout);
#if defined(_WIN32)
    FILE *p = _wpopen(wstr(cmd, clen), L"r");
#else
    FILE *p = popen(cstr(cmd, clen), "r");
#endif
    if (!p) {
        *err = errno ? errno : EIO;
        return -1;
    }
    buf_t b = {0};
    int rerr = read_stream(p, &b);
#if defined(_WIN32)
    int status = _pclose(p);
#else
    int status = pclose(p);
#endif
    set_string(out, b.data ? b.data : "", b.len);
    *err = rerr;
#if defined(_WIN32)
    return status;
#else
    if (status == -1) {
        *err = errno;
        return -1;
    }
    return WIFEXITED(status) ? WEXITSTATUS(status) : 128 + WTERMSIG(status);
#endif
}

/* ---- files ------------------------------------------------------------- */

bool veles_utf8_valid(const char *s, int64_t len);

static int64_t read_raw(const char *path, int64_t plen, veles_string *out) {
    FILE *f = open_file(path, plen, "rb");
    if (!f) return errno;
    buf_t b = {0};
    int err = read_stream(f, &b);
    fclose(f);
    if (err) return err;
    set_string(out, b.data ? b.data : "", b.len);
    return 0;
}

/* veles_fs_read_file reads a text file; bytes that are not UTF-8 are an
 * error (EILSEQ), since a Veles string is always well-formed (D18). */
int64_t veles_fs_read_file(const char *path, int64_t plen, veles_string *out) {
    int64_t err = read_raw(path, plen, out);
    if (err) return err;
    if (!veles_utf8_valid(out->data, out->len)) return EILSEQ;
    return 0;
}

static int64_t write_file(const char *path, int64_t plen, const char *text, int64_t tlen, const char *mode) {
    FILE *f = open_file(path, plen, mode);
    if (!f) return errno;
    if (tlen > 0 && fwrite(text, 1, (size_t)tlen, f) != (size_t)tlen) {
        int err = errno ? errno : EIO;
        fclose(f);
        return err;
    }
    if (fclose(f) != 0) return errno ? errno : EIO;
    return 0;
}

int64_t veles_fs_write_file(const char *path, int64_t plen, const char *text, int64_t tlen) {
    return write_file(path, plen, text, tlen, "wb");
}

int64_t veles_fs_append_file(const char *path, int64_t plen, const char *text, int64_t tlen) {
    return write_file(path, plen, text, tlen, "ab");
}

/* veles_fs_stat: 0 = missing, 1 = file, 2 = directory */
int64_t veles_fs_stat(const char *path, int64_t plen) {
#if defined(_WIN32)
    struct _stat64 st;
    if (_wstat64(wstr(path, plen), &st) != 0) return 0;
#else
    struct stat st;
    if (stat(cstr(path, plen), &st) != 0) return 0;
#endif
    return S_ISDIR(st.st_mode) ? 2 : 1;
}

/* veles_fs_list_dir writes the entry names, one per line, sorted by the
 * Veles side. */
int64_t veles_fs_list_dir(const char *path, int64_t plen, veles_string *out) {
    buf_t b = {0};
#if defined(_WIN32)
    _WDIR *d = _wopendir(wstr(path, plen));
    if (!d) return errno;
    struct _wdirent *e;
    while ((e = _wreaddir(d)) != NULL) {
        if (wcscmp(e->d_name, L".") == 0 || wcscmp(e->d_name, L"..") == 0) continue;
        veles_string name;
        set_wstring(&name, e->d_name);
        if (b.len) buf_push(&b, "\n", 1);
        buf_push(&b, name.data, name.len);
    }
    _wclosedir(d);
#else
    DIR *d = opendir(cstr(path, plen));
    if (!d) return errno;
    struct dirent *e;
    while ((e = readdir(d)) != NULL) {
        if (strcmp(e->d_name, ".") == 0 || strcmp(e->d_name, "..") == 0) continue;
        if (b.len) buf_push(&b, "\n", 1);
        buf_push(&b, e->d_name, (int64_t)strlen(e->d_name));
    }
    closedir(d);
#endif
    set_string(out, b.data ? b.data : "", b.len);
    return 0;
}

int64_t veles_fs_mkdir(const char *path, int64_t plen) {
#if defined(_WIN32)
    if (_wmkdir(wstr(path, plen)) != 0 && errno != EEXIST) return errno;
#else
    if (mkdir(cstr(path, plen), 0777) != 0 && errno != EEXIST) return errno;
#endif
    return 0;
}

int64_t veles_fs_remove(const char *path, int64_t plen) {
#if defined(_WIN32)
    wchar_t *p = wstr(path, plen);
    struct _stat64 st;
    if (_wstat64(p, &st) != 0) return errno;
    if (S_ISDIR(st.st_mode)) {
        if (_wrmdir(p) != 0) return errno;
        return 0;
    }
    if (_wremove(p) != 0) return errno;
    return 0;
#else
    char *p = cstr(path, plen);
    struct stat st;
    if (stat(p, &st) != 0) return errno;
    if (S_ISDIR(st.st_mode)) {
        if (rmdir(p) != 0) return errno;
        return 0;
    }
    if (remove(p) != 0) return errno;
    return 0;
#endif
}

int64_t veles_fs_rename(const char *from, int64_t flen, const char *to, int64_t tlen) {
#if defined(_WIN32)
    if (_wrename(wstr(from, flen), wstr(to, tlen)) != 0) return errno;
#else
    if (rename(cstr(from, flen), cstr(to, tlen)) != 0) return errno;
#endif
    return 0;
}

int64_t veles_fs_cwd(veles_string *out) {
#if defined(_WIN32)
    wchar_t buf[4096];
    if (!_wgetcwd(buf, 4096)) return errno;
    set_wstring(out, buf);
#else
    char buf[4096];
    if (!getcwd(buf, sizeof buf)) return errno;
    set_string(out, buf, (int64_t)strlen(buf));
#endif
    return 0;
}

/* ---- bytes ------------------------------------------------------------- */

/* A List<u8> from the Veles side: only data and len are read here. */
typedef struct {
    char *data;
    int64_t len;
} veles_bytes_view;

/* veles_fs_read_bytes reads a file as raw bytes into a string-shaped
 * buffer; the Veles side turns it into a List<u8> with .bytes(). */
int64_t veles_fs_read_bytes(const char *path, int64_t plen, veles_string *out) {
    return read_raw(path, plen, out);
}

/* veles_utf8_valid reports whether the bytes are well-formed UTF-8, so that
 * readFile can refuse a file that is not text (EILSEQ). */
bool veles_utf8_valid(const char *s, int64_t len) {
    const unsigned char *p = (const unsigned char *)s;
    int64_t i = 0;
    while (i < len) {
        unsigned char c = p[i];
        int n; /* continuation bytes */
        if (c < 0x80) { i++; continue; }
        if (c >= 0xC2 && c <= 0xDF) n = 1;
        else if (c >= 0xE0 && c <= 0xEF) n = 2;
        else if (c >= 0xF0 && c <= 0xF4) n = 3;
        else return false;
        if (i + n >= len) return false;
        for (int k = 1; k <= n; k++) {
            if ((p[i + k] & 0xC0) != 0x80) return false;
        }
        if (c == 0xE0 && p[i + 1] < 0xA0) return false;  /* overlong */
        if (c == 0xED && p[i + 1] >= 0xA0) return false; /* surrogate */
        if (c == 0xF0 && p[i + 1] < 0x90) return false;  /* overlong */
        if (c == 0xF4 && p[i + 1] >= 0x90) return false; /* beyond U+10FFFF */
        i += n + 1;
    }
    return true;
}

int64_t veles_fs_write_bytes(const char *path, int64_t plen, veles_bytes_view *bytes, bool append) {
    return write_file(path, plen, bytes->data, bytes->len, append ? "ab" : "wb");
}

/* veles_read_all reads standard input to its end. */
int64_t veles_read_all(veles_string *out) {
    buf_t b = {0};
    int err = read_stream(stdin, &b);
    if (err) return err;
    set_string(out, b.data ? b.data : "", b.len);
    return 0;
}

/* ---- time -------------------------------------------------------------- */

#include <time.h>

/* milliseconds since the Unix epoch, UTC */
int64_t veles_time_now_ms(void) {
#if defined(_WIN32)
    FILETIME ft;
    GetSystemTimeAsFileTime(&ft);
    uint64_t t = ((uint64_t)ft.dwHighDateTime << 32) | ft.dwLowDateTime;
    return (int64_t)(t / 10000) - 11644473600000LL;
#else
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return ts.tv_sec * 1000 + ts.tv_nsec / 1000000;
#endif
}

/* nanoseconds on a monotonic clock, for measuring */
int64_t veles_time_monotonic_ns(void) {
#if defined(_WIN32)
    LARGE_INTEGER f, c;
    QueryPerformanceFrequency(&f);
    QueryPerformanceCounter(&c);
    return (int64_t)((double)c.QuadPart * 1e9 / (double)f.QuadPart);
#else
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000000000LL + ts.tv_nsec;
#endif
}

/* veles_time_civil writes the calendar fields of a Unix-millisecond time
 * (UTC, or local when local is set) as
 * "year month day hour minute second weekday yearday" for the Veles side to
 * split; weekday is 0 for Sunday. */
void veles_time_civil(int64_t ms, bool local, veles_string *out) {
    time_t secs = (time_t)(ms / 1000);
    if (ms < 0 && ms % 1000 != 0) secs -= 1;
    struct tm tmv;
#if defined(_WIN32)
    if (local) localtime_s(&tmv, &secs); else gmtime_s(&tmv, &secs);
#else
    if (local) localtime_r(&secs, &tmv); else gmtime_r(&secs, &tmv);
#endif
    char buf[128];
    int n = snprintf(buf, sizeof buf, "%d %d %d %d %d %d %d %d", tmv.tm_year + 1900, tmv.tm_mon + 1, tmv.tm_mday,
                     tmv.tm_hour, tmv.tm_min, tmv.tm_sec, tmv.tm_wday, tmv.tm_yday + 1);
    set_string(out, buf, n);
}

/* ---- number formatting ------------------------------------------------- */

/* veles_f64_to_fixed renders v with exactly `digits` decimals (0..30). */
void veles_f64_to_fixed(double v, int64_t digits, veles_string *out) {
    if (digits < 0) digits = 0;
    if (digits > 30) digits = 30;
    char buf[400];
    int n = snprintf(buf, sizeof buf, "%.*f", (int)digits, v);
    if (n < 0) n = 0;
    if (n > (int)sizeof buf - 1) n = (int)sizeof buf - 1;
    set_string(out, buf, n);
}
