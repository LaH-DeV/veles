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
#include <direct.h>
#include <io.h>
#define veles_popen _popen
#define veles_pclose _pclose
#define veles_mkdir(p) _mkdir(p)
#else
#include <unistd.h>
#include <sys/wait.h>
#define veles_popen popen
#define veles_pclose pclose
#define veles_mkdir(p) mkdir(p, 0777)
#endif

typedef struct {
    const char *data;
    int64_t len;
} veles_string;

void *veles_alloc(int64_t size);
void veles_panic(const char *msg, int64_t len);

/* cstr copies a Veles string into a NUL-terminated buffer the C library
 * can take; Veles strings are not NUL-terminated. */
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
    const char *s = strerror((int)code);
    set_string(out, s, (int64_t)strlen(s));
}

/* ---- process ----------------------------------------------------------- */

extern int veles_os_argc_value(void);
extern char **veles_os_argv_value(void);

int64_t veles_os_argc(void) { return veles_os_argc_value(); }

void veles_os_arg(int64_t i, veles_string *out) {
    char **argv = veles_os_argv_value();
    int argc = veles_os_argc_value();
    if (i < 0 || i >= argc) {
        veles_panic("os.arg: index out of range", 26);
    }
    set_string(out, argv[i], (int64_t)strlen(argv[i]));
}

bool veles_os_getenv(const char *name, int64_t nlen, veles_string *out) {
    const char *v = getenv(cstr(name, nlen));
    if (!v) return false;
    set_string(out, v, (int64_t)strlen(v));
    return true;
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
    FILE *p = veles_popen(cstr(cmd, clen), "r");
    if (!p) {
        *err = errno ? errno : EIO;
        return -1;
    }
    buf_t b = {0};
    int rerr = read_stream(p, &b);
    int status = veles_pclose(p);
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

int64_t veles_fs_read_file(const char *path, int64_t plen, veles_string *out) {
    FILE *f = fopen(cstr(path, plen), "rb");
    if (!f) return errno;
    buf_t b = {0};
    int err = read_stream(f, &b);
    fclose(f);
    if (err) return err;
    set_string(out, b.data ? b.data : "", b.len);
    return 0;
}

static int64_t write_file(const char *path, int64_t plen, const char *text, int64_t tlen, const char *mode) {
    FILE *f = fopen(cstr(path, plen), mode);
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
    struct stat st;
    if (stat(cstr(path, plen), &st) != 0) return 0;
    return S_ISDIR(st.st_mode) ? 2 : 1;
}

/* veles_fs_list_dir writes the entry names, one per line, sorted by the
 * Veles side. */
int64_t veles_fs_list_dir(const char *path, int64_t plen, veles_string *out) {
    DIR *d = opendir(cstr(path, plen));
    if (!d) return errno;
    buf_t b = {0};
    struct dirent *e;
    while ((e = readdir(d)) != NULL) {
        if (strcmp(e->d_name, ".") == 0 || strcmp(e->d_name, "..") == 0) continue;
        if (b.len) buf_push(&b, "\n", 1);
        buf_push(&b, e->d_name, (int64_t)strlen(e->d_name));
    }
    closedir(d);
    set_string(out, b.data ? b.data : "", b.len);
    return 0;
}

int64_t veles_fs_mkdir(const char *path, int64_t plen) {
    if (veles_mkdir(cstr(path, plen)) != 0 && errno != EEXIST) return errno;
    return 0;
}

int64_t veles_fs_remove(const char *path, int64_t plen) {
    char *p = cstr(path, plen);
    struct stat st;
    if (stat(p, &st) != 0) return errno;
    if (S_ISDIR(st.st_mode)) {
        if (rmdir(p) != 0) return errno;
        return 0;
    }
    if (remove(p) != 0) return errno;
    return 0;
}

int64_t veles_fs_rename(const char *from, int64_t flen, const char *to, int64_t tlen) {
    if (rename(cstr(from, flen), cstr(to, tlen)) != 0) return errno;
    return 0;
}

int64_t veles_fs_cwd(veles_string *out) {
    char buf[4096];
#if defined(_WIN32)
    if (!_getcwd(buf, sizeof buf)) return errno;
#else
    if (!getcwd(buf, sizeof buf)) return errno;
#endif
    set_string(out, buf, (int64_t)strlen(buf));
    return 0;
}
