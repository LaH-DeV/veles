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
/* glibc declares its extensions (pthread_getattr_np, ...) only when asked. */
#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif
#include <ctype.h>
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
#include <bcrypt.h>
#include <direct.h>
#include <io.h>
#else
#include <spawn.h>
#include <unistd.h>
#include <sys/wait.h>
#include <fcntl.h>
#include <poll.h>
#include <signal.h>
#if defined(__linux__)
#include <sys/random.h>
#endif
#endif

typedef struct {
    const char *data;
    int64_t len;
} veles_string;

void *veles_alloc(int64_t size);
void veles_blocking_enter(void);
void veles_blocking_leave(void);
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

/* A growing byte buffer for reads. It lives outside the Veles heap, so a
 * read can fill it inside a safe region (D66) — blocked on a pipe, a slow
 * disk or the terminal without holding up a collection — and buf_finish
 * copies the result into a string afterwards. */
typedef struct {
    char *data;
    int64_t len, cap;
} buf_t;

static void buf_push(buf_t *b, const char *p, int64_t n) {
    if (b->len + n > b->cap) {
        int64_t ncap = b->cap ? b->cap * 2 : 4096;
        while (ncap < b->len + n) ncap *= 2;
        char *nd = realloc(b->data, (size_t)ncap);
        if (!nd) veles_panic("out of memory", 13);
        b->data = nd;
        b->cap = ncap;
    }
    memcpy(b->data + b->len, p, (size_t)n);
    b->len += n;
}

static void buf_finish(veles_string *out, buf_t *b) {
    set_string(out, b->data ? b->data : "", b->len);
    free(b->data);
    b->data = NULL;
    b->len = b->cap = 0;
}

static int read_stream(FILE *f, buf_t *b) {
    char chunk[8192];
    size_t n;
    veles_blocking_enter();
    while ((n = fread(chunk, 1, sizeof chunk, f)) > 0) buf_push(b, chunk, (int64_t)n);
    int err = ferror(f) ? (errno ? errno : EIO) : 0;
    veles_blocking_leave();
    return err;
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


/* ---- process identity -------------------------------------------------- */

int64_t veles_os_pid(void) {
#if defined(_WIN32)
    return (int64_t)GetCurrentProcessId();
#else
    return (int64_t)getpid();
#endif
}

/* ---- shutdown signals (D68) ----------------------------------------------
 * Nothing is intercepted until veles_signal_watch() is first called, so a
 * program that never asks keeps the default (the signal ends it). Each
 * watch arms the handler for one signal, recorded for veles_signal_take();
 * a second signal before the next watch has the default effect, so a
 * shutdown that hangs can still be killed. Values: 2 = interrupt,
 * 15 = terminate. */

#if defined(_WIN32)
static volatile LONG signal_pending;
static volatile LONG signal_fired; /* the armed signal has come */
static volatile LONG signal_watching;

static BOOL WINAPI on_console_ctrl(DWORD kind) {
    if (InterlockedCompareExchange(&signal_fired, 1, 0) != 0)
        return FALSE; /* the second one: default handling ends the process */
    LONG sig = kind == CTRL_C_EVENT ? 2 : 15;
    InterlockedExchange(&signal_pending, sig);
    if (kind == CTRL_CLOSE_EVENT || kind == CTRL_LOGOFF_EVENT || kind == CTRL_SHUTDOWN_EVENT) {
        /* Windows ends the process as soon as this handler returns for these
         * three; waiting here is the shutdown's time budget (about five
         * seconds for a closed console). A normal exit ends this thread. */
        Sleep(INFINITE);
    }
    return TRUE;
}

void veles_signal_watch(void) {
    if (InterlockedCompareExchange(&signal_watching, 1, 0) == 0)
        SetConsoleCtrlHandler(on_console_ctrl, TRUE);
    InterlockedExchange(&signal_fired, 0); /* armed again */
}

int64_t veles_signal_take(void) {
    return (int64_t)InterlockedExchange(&signal_pending, 0);
}

/* as if `sig` came from outside: recorded when watched, otherwise the
 * default — the process ends */
void veles_signal_raise(int64_t sig) {
    if (signal_watching && InterlockedCompareExchange(&signal_fired, 1, 0) == 0) {
        InterlockedExchange(&signal_pending, (LONG)sig);
        return;
    }
    ExitProcess(sig == 2 ? 0xC000013A /* STATUS_CONTROL_C_EXIT */ : 1);
}
#else
#include <signal.h>

static volatile sig_atomic_t signal_pending;

static void on_signal(int sig) {
    signal_pending = sig == SIGINT ? 2 : 15;
    /* disarm the other one too, as on Windows: whichever comes next has
     * the default effect (sigaction is async-signal-safe) */
    struct sigaction dfl;
    memset(&dfl, 0, sizeof dfl);
    dfl.sa_handler = SIG_DFL;
    sigaction(sig == SIGINT ? SIGTERM : SIGINT, &dfl, NULL);
}

void veles_signal_watch(void) {
    /* installed on every watch: SA_RESETHAND took it down after the last one */
    struct sigaction sa;
    memset(&sa, 0, sizeof sa);
    sa.sa_handler = on_signal;
    sigemptyset(&sa.sa_mask);
    /* one-shot: the handler is reset to the default once it has run, and
     * a slow system call is restarted rather than failing with EINTR */
    sa.sa_flags = SA_RESETHAND | SA_RESTART;
    sigaction(SIGINT, &sa, NULL);
    sigaction(SIGTERM, &sa, NULL);
}

int64_t veles_signal_take(void) {
    int64_t sig = signal_pending;
    signal_pending = 0;
    return sig;
}

void veles_signal_raise(int64_t sig) {
    raise(sig == 2 ? SIGINT : SIGTERM);
}
#endif

/* the host's name, as the network knows it; 0 or an error code */
int64_t veles_os_hostname(veles_string *out) {
#if defined(_WIN32)
    wchar_t buf[256];
    DWORD n = sizeof buf / sizeof buf[0];
    if (!GetComputerNameExW(ComputerNameDnsHostname, buf, &n)) return EIO;
    set_wstring(out, buf);
#else
    char buf[256];
    if (gethostname(buf, sizeof buf) != 0) return errno;
    buf[sizeof buf - 1] = 0;
    set_string(out, buf, (int64_t)strlen(buf));
#endif
    return 0;
}

/* the directory for temporary files, without a trailing separator (a root
 * keeps its own: "C:\", "/"). TMPDIR, then /tmp, on POSIX; GetTempPathW,
 * which reads TMP and TEMP, on Windows. */
void veles_os_temp_dir(veles_string *out) {
#if defined(_WIN32)
    wchar_t buf[MAX_PATH + 2];
    DWORD n = GetTempPathW(MAX_PATH + 2, buf);
    if (n == 0 || n > MAX_PATH + 1) {
        set_string(out, "C:\\Windows\\Temp", 15);
        return;
    }
    while (n > 3 && (buf[n - 1] == L'\\' || buf[n - 1] == L'/')) buf[--n] = 0;
    set_wstring(out, buf);
#else
    const char *t = getenv("TMPDIR");
    if (!t || !*t) t = "/tmp";
    size_t len = strlen(t);
    while (len > 1 && t[len - 1] == '/') len--;
    set_string(out, t, (int64_t)len);
#endif
}

/* what happens to the child's standard error (os.Stderr, D82) */
#define RUN_CAPTURE 0 /* into its own text, errout */
#define RUN_INHERIT 1 /* to this process's standard error */
#define RUN_MERGE 2   /* into the output, in order */

/* veles_os_run starts a program with arguments — `argz` is the program
 * and each argument, separated by NUL bytes — writes `input` to its
 * standard input and closes it (no input: it reads the end at once), and
 * captures its standard output, and its standard error as `mode` says.
 * There is NO SHELL: an argument reaches the program as one argument,
 * whatever it holds, so `;`, `|`, `$(...)`, `%VAR%` and quotes are text.
 * Returns the exit status, or -1 with an errno value in *err when the
 * program could not be started. */
#if defined(_WIN32)
/* One argument on a Windows command line, quoted by the rules the C
 * runtime's parser (and CommandLineToArgvW) reads back: backslashes are
 * literal except before a quote, where they are doubled, and a quote is
 * written \". */
static void append_arg(buf_t *b, const char *s, int64_t n) {
    bool plain = n > 0;
    for (int64_t i = 0; i < n && plain; i++) {
        plain = s[i] != ' ' && s[i] != '\t' && s[i] != '\n' && s[i] != '\v' && s[i] != '"';
    }
    if (plain) {
        buf_push(b, s, n);
        return;
    }
    buf_push(b, "\"", 1);
    int64_t slashes = 0;
    for (int64_t i = 0; i < n; i++) {
        if (s[i] == '\\') {
            slashes++;
            continue;
        }
        for (int64_t k = 0; k < (s[i] == '"' ? 2 * slashes + 1 : slashes); k++) buf_push(b, "\\", 1);
        slashes = 0;
        buf_push(b, s + i, 1);
    }
    for (int64_t k = 0; k < 2 * slashes; k++) buf_push(b, "\\", 1); /* before the closing quote */
    buf_push(b, "\"", 1);
}

static int64_t errno_of_win32(DWORD e) {
    switch (e) {
    case ERROR_FILE_NOT_FOUND:
    case ERROR_PATH_NOT_FOUND:
    case ERROR_INVALID_NAME:
        return ENOENT;
    case ERROR_ACCESS_DENIED:
        return EACCES;
    case ERROR_BAD_EXE_FORMAT:
        return ENOEXEC;
    case ERROR_NOT_ENOUGH_MEMORY:
    case ERROR_OUTOFMEMORY:
        return ENOMEM;
    default:
        return EIO;
    }
}

typedef struct {
    HANDLE h;
    const char *data;
    int64_t len;
} run_writer;

/* writes the child's input and closes the pipe, so it sees the end */
static DWORD WINAPI write_input(LPVOID arg) {
    run_writer *w = arg;
    int64_t off = 0;
    while (off < w->len) {
        DWORD n = 0, want = (DWORD)(w->len - off > 65536 ? 65536 : w->len - off);
        if (!WriteFile(w->h, w->data + off, want, &n, NULL)) break; /* the child stopped reading */
        off += n;
    }
    CloseHandle(w->h);
    return 0;
}

typedef struct {
    HANDLE h;
    buf_t b;
} run_reader;

static DWORD WINAPI read_all(LPVOID arg) {
    run_reader *r = arg;
    char chunk[8192];
    DWORD n;
    while (ReadFile(r->h, chunk, sizeof chunk, &n, NULL) && n > 0) buf_push(&r->b, chunk, (int64_t)n);
    CloseHandle(r->h);
    return 0;
}

int64_t veles_os_run(const char *argz, int64_t alen, const char *input, int64_t inlen, int64_t mode, veles_string *out, veles_string *errout, int64_t *err) {
    /* CreateProcess hands a .bat or .cmd file to cmd.exe, which parses the
     * arguments again by rules no quoting here survives: refused (-2) */
    int64_t plen = (int64_t)strnlen(argz, (size_t)alen);
    if (plen >= 4 && argz[plen - 4] == '.' &&
        ((tolower(argz[plen - 3]) == 'b' && tolower(argz[plen - 2]) == 'a' && tolower(argz[plen - 1]) == 't') ||
         (tolower(argz[plen - 3]) == 'c' && tolower(argz[plen - 2]) == 'm' && tolower(argz[plen - 1]) == 'd'))) {
        *err = EINVAL;
        return -2;
    }
    fflush(stdout);
    buf_t line = {0};
    for (int64_t i = 0, start = 0; i <= alen; i++) {
        if (i == alen || argz[i] == 0) {
            if (start > 0) buf_push(&line, " ", 1);
            append_arg(&line, argz + start, i - start);
            start = i + 1;
        }
    }
    wchar_t *cmd = wstr(line.data, line.len);
    free(line.data);

    /* three pipes: the input the child reads, its output, and — when it is
     * captured apart — its errors; the parent's ends are not inherited */
    SECURITY_ATTRIBUTES sa = {sizeof sa, NULL, TRUE};
    HANDLE out_r, out_w, in_r, in_w, err_r = NULL, err_w = NULL;
    if (!CreatePipe(&out_r, &out_w, &sa, 0)) {
        *err = errno_of_win32(GetLastError());
        return -1;
    }
    if (!CreatePipe(&in_r, &in_w, &sa, 0)) {
        *err = errno_of_win32(GetLastError());
        CloseHandle(out_r);
        CloseHandle(out_w);
        return -1;
    }
    if (mode == RUN_CAPTURE && !CreatePipe(&err_r, &err_w, &sa, 0)) {
        *err = errno_of_win32(GetLastError());
        CloseHandle(out_r);
        CloseHandle(out_w);
        CloseHandle(in_r);
        CloseHandle(in_w);
        return -1;
    }
    SetHandleInformation(out_r, HANDLE_FLAG_INHERIT, 0);
    SetHandleInformation(in_w, HANDLE_FLAG_INHERIT, 0);
    if (err_r) SetHandleInformation(err_r, HANDLE_FLAG_INHERIT, 0);
    STARTUPINFOW si = {0};
    si.cb = sizeof si;
    si.dwFlags = STARTF_USESTDHANDLES;
    si.hStdInput = in_r;
    si.hStdOutput = out_w;
    si.hStdError = mode == RUN_MERGE ? out_w : mode == RUN_CAPTURE ? err_w : GetStdHandle(STD_ERROR_HANDLE);
    PROCESS_INFORMATION pi;
    /* the child runs as long as it likes: this thread waits for it in a
     * safe region, out of the collector's way */
    veles_blocking_enter();
    BOOL started = CreateProcessW(NULL, cmd, NULL, NULL, TRUE, 0, NULL, NULL, &si, &pi);
    DWORD startErr = GetLastError();
    /* else the reads below never see the end */
    CloseHandle(out_w);
    CloseHandle(in_r);
    if (err_w) CloseHandle(err_w);
    if (!started) {
        CloseHandle(out_r);
        CloseHandle(in_w);
        if (err_r) CloseHandle(err_r);
        veles_blocking_leave();
        *err = errno_of_win32(startErr);
        return -1;
    }
    CloseHandle(pi.hThread);
    /* the input goes in, and the errors come out, at the same time as the
     * output is read: a child that fills one pipe while this thread waits
     * on another would otherwise wait for ever, and so would this */
    run_writer wr = {in_w, input, inlen};
    HANDLE tw = CreateThread(NULL, 0, write_input, &wr, 0, NULL);
    if (!tw) CloseHandle(in_w); /* no input then: the child sees its end */
    run_reader er = {err_r, {0}};
    HANDLE te = err_r ? CreateThread(NULL, 0, read_all, &er, 0, NULL) : NULL;
    buf_t b = {0};
    char chunk[8192];
    DWORD n;
    while (ReadFile(out_r, chunk, sizeof chunk, &n, NULL) && n > 0) buf_push(&b, chunk, (int64_t)n);
    CloseHandle(out_r);
    if (err_r && !te) read_all(&er); /* no thread for it: read it now */
    if (tw) {
        WaitForSingleObject(tw, INFINITE);
        CloseHandle(tw);
    }
    if (te) {
        WaitForSingleObject(te, INFINITE);
        CloseHandle(te);
    }
    WaitForSingleObject(pi.hProcess, INFINITE);
    DWORD code = 0;
    GetExitCodeProcess(pi.hProcess, &code);
    CloseHandle(pi.hProcess);
    veles_blocking_leave();
    buf_finish(out, &b);
    buf_finish(errout, &er.b);
    *err = 0;
    return (int64_t)code;
}
#else
extern char **environ;

/* reads what is there on fd into b; closes it and returns -1 at its end */
static int drain_fd(int fd, buf_t *b) {
    char chunk[8192];
    ssize_t n = read(fd, chunk, sizeof chunk);
    if (n > 0) {
        buf_push(b, chunk, (int64_t)n);
        return fd;
    }
    if (n < 0 && (errno == EINTR || errno == EAGAIN)) return fd;
    close(fd);
    return -1;
}

int64_t veles_os_run(const char *argz, int64_t alen, const char *input, int64_t inlen, int64_t mode, veles_string *out, veles_string *errout, int64_t *err) {
    fflush(stdout);
    int64_t argc = 1;
    for (int64_t i = 0; i < alen; i++) argc += argz[i] == 0;
    char **argv = calloc((size_t)argc + 1, sizeof *argv);
    char *copy = malloc((size_t)alen + 1);
    if (!argv || !copy) veles_panic("out of memory", 13);
    memcpy(copy, argz, (size_t)alen);
    copy[alen] = 0;
    for (int64_t i = 0, k = 0, start = 0; i <= alen; i++) {
        if (i == alen || copy[i] == 0) {
            argv[k++] = copy + start;
            start = i + 1;
        }
    }
    /* three pipes: the input the child reads, its output, and — when it is
     * captured apart — its errors */
    int outp[2], inp[2], errp[2] = {-1, -1};
    if (pipe(outp) != 0) {
        *err = errno;
        free(argv);
        free(copy);
        return -1;
    }
    if (pipe(inp) != 0 || (mode == RUN_CAPTURE && pipe(errp) != 0)) {
        *err = errno;
        close(outp[0]);
        close(outp[1]);
        if (inp[0] >= 0) { close(inp[0]); close(inp[1]); }
        free(argv);
        free(copy);
        return -1;
    }
    /* a child that stops reading its input is an EPIPE here, not a signal
     * that ends this process */
    signal(SIGPIPE, SIG_IGN);
    posix_spawn_file_actions_t fa;
    posix_spawn_file_actions_init(&fa);
    posix_spawn_file_actions_adddup2(&fa, inp[0], 0);
    posix_spawn_file_actions_adddup2(&fa, outp[1], 1);
    if (mode == RUN_MERGE) posix_spawn_file_actions_adddup2(&fa, outp[1], 2);
    if (mode == RUN_CAPTURE) posix_spawn_file_actions_adddup2(&fa, errp[1], 2);
    int ends[] = {inp[0], inp[1], outp[0], outp[1], errp[0], errp[1]};
    for (int i = 0; i < 6; i++)
        if (ends[i] > 2) posix_spawn_file_actions_addclose(&fa, ends[i]);
    veles_blocking_enter();
    pid_t pid;
    int rc = posix_spawnp(&pid, argv[0], &fa, NULL, argv, environ);
    posix_spawn_file_actions_destroy(&fa);
    /* else the reads below never see the end */
    close(inp[0]);
    close(outp[1]);
    if (errp[1] >= 0) close(errp[1]);
    free(argv);
    free(copy);
    if (rc != 0) {
        close(inp[1]);
        close(outp[0]);
        if (errp[0] >= 0) close(errp[0]);
        veles_blocking_leave();
        *err = rc;
        return -1;
    }
    /* the input goes in and both outputs come out at once: a child that
     * fills one pipe while this waits on another would wait for ever, and
     * so would this */
    int in_fd = inp[1], out_fd = outp[0], err_fd = errp[0];
    if (inlen == 0) {
        close(in_fd);
        in_fd = -1;
    } else {
        fcntl(in_fd, F_SETFL, fcntl(in_fd, F_GETFL) | O_NONBLOCK);
    }
    buf_t b = {0}, eb = {0};
    int64_t off = 0;
    while (in_fd >= 0 || out_fd >= 0 || err_fd >= 0) {
        struct pollfd p[3];
        int n = 0, at_in = -1, at_out = -1, at_err = -1;
        if (in_fd >= 0) { at_in = n; p[n].fd = in_fd; p[n++].events = POLLOUT; }
        if (out_fd >= 0) { at_out = n; p[n].fd = out_fd; p[n++].events = POLLIN; }
        if (err_fd >= 0) { at_err = n; p[n].fd = err_fd; p[n++].events = POLLIN; }
        if (poll(p, (nfds_t)n, -1) < 0) {
            if (errno == EINTR) continue;
            break;
        }
        if (at_in >= 0 && p[at_in].revents) {
            int64_t want = inlen - off > 65536 ? 65536 : inlen - off;
            ssize_t w = write(in_fd, input + off, (size_t)want);
            if (w > 0) off += w;
            if ((w < 0 && errno != EAGAIN && errno != EINTR) || off >= inlen) {
                close(in_fd); /* written, or the child stopped reading */
                in_fd = -1;
            }
        }
        if (at_out >= 0 && p[at_out].revents) out_fd = drain_fd(out_fd, &b);
        if (at_err >= 0 && p[at_err].revents) err_fd = drain_fd(err_fd, &eb);
    }
    if (in_fd >= 0) close(in_fd);
    if (out_fd >= 0) close(out_fd);
    if (err_fd >= 0) close(err_fd);
    int rerr = 0;
    int status;
    while (waitpid(pid, &status, 0) < 0) {
        if (errno != EINTR) {
            status = -1;
            break;
        }
    }
    veles_blocking_leave();
    buf_finish(out, &b);
    buf_finish(errout, &eb);
    *err = rerr;
    if (status == -1) {
        *err = ECHILD;
        return -1;
    }
    return WIFEXITED(status) ? WEXITSTATUS(status) : 128 + WTERMSIG(status);
}
#endif

/* ---- files ------------------------------------------------------------- */

bool veles_utf8_valid(const char *s, int64_t len);

static int64_t read_raw(const char *path, int64_t plen, veles_string *out) {
    FILE *f = open_file(path, plen, "rb");
    if (!f) return errno;
    buf_t b = {0};
    int err = read_stream(f, &b);
    fclose(f);
    if (err) {
        free(b.data);
        return err;
    }
    buf_finish(out, &b);
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

/* veles_fs_lstat: like veles_fs_stat, but a symbolic link is 3 and is not
 * followed. On Windows a junction counts as a link too; other reparse
 * points (OneDrive placeholders, deduplicated files) are ordinary entries,
 * so the reparse tag is read rather than the attribute alone. */
int64_t veles_fs_lstat(const char *path, int64_t plen) {
#if defined(_WIN32)
    wchar_t *w = wstr(path, plen);
    DWORD a = GetFileAttributesW(w);
    if (a == INVALID_FILE_ATTRIBUTES) return 0;
    if (a & FILE_ATTRIBUTE_REPARSE_POINT) {
        WIN32_FIND_DATAW fd;
        HANDLE h = FindFirstFileW(w, &fd);
        if (h != INVALID_HANDLE_VALUE) {
            FindClose(h);
            if (fd.dwReserved0 == IO_REPARSE_TAG_SYMLINK || fd.dwReserved0 == IO_REPARSE_TAG_MOUNT_POINT) return 3;
        }
    }
    return (a & FILE_ATTRIBUTE_DIRECTORY) ? 2 : 1;
#else
    struct stat st;
    if (lstat(cstr(path, plen), &st) != 0) return 0;
    if (S_ISLNK(st.st_mode)) return 3;
    return S_ISDIR(st.st_mode) ? 2 : 1;
#endif
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
    buf_finish(out, &b);
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
    if (err) {
        free(b.data);
        return err;
    }
    buf_finish(out, &b);
    return 0;
}

/* ---- time -------------------------------------------------------------- */

#include <time.h>

/* microseconds since the Unix epoch, UTC. Microseconds, not milliseconds:
 * a Veles Timestamp is an i64 of them, which round-trips RFC 3339's six
 * fractional digits and a PostgreSQL timestamptz without loss and still
 * spans +/-292,000 years. */
int64_t veles_time_now_us(void) {
#if defined(_WIN32)
    FILETIME ft;
#if defined(_WIN32_WINNT) && _WIN32_WINNT >= 0x0602
    GetSystemTimePreciseAsFileTime(&ft); /* sub-microsecond */
#else
    GetSystemTimeAsFileTime(&ft);        /* ~15 ms tick on older targets */
#endif
    uint64_t t = ((uint64_t)ft.dwHighDateTime << 32) | ft.dwLowDateTime;
    /* FILETIME counts 100 ns ticks from 1601-01-01 */
    return (int64_t)(t / 10) - 11644473600000000LL;
#else
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    return ts.tv_sec * 1000000LL + ts.tv_nsec / 1000;
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

/* Days since 1970-01-01 from a proleptic Gregorian date (Howard Hinnant's
 * days_from_civil). std/time has the same function in Veles; this copy
 * exists only to difference the two struct tm below. */
static int64_t days_from_civil(int64_t y, int64_t m, int64_t d) {
    y -= m <= 2;
    int64_t era = (y >= 0 ? y : y - 399) / 400;
    int64_t yoe = y - era * 400;                                   /* 0..399 */
    int64_t doy = (153 * (m + (m > 2 ? -3 : 9)) + 2) / 5 + d - 1;  /* 0..365 */
    int64_t doe = yoe * 365 + yoe / 4 - yoe / 100 + doy;           /* 0..146096 */
    return era * 146097 + doe - 719468;
}

/* The host time zone's offset east of UTC, in minutes, at the instant
 * `secs` (Unix seconds). Taken as the difference between that instant's
 * local and UTC calendar fields, so it needs no tz internals and follows
 * daylight saving.
 *
 * VELES_TIME_OFFSET_UNKNOWN (not zero — zero is a real offset, and a UTC
 * host must not be indistinguishable from a failure) when the platform
 * will not convert that instant: the Microsoft CRT refuses a negative
 * time_t and anything past the year 3000, where glibc is happy. std/time
 * answers that by asking again for the same date in a year the host can
 * do, which is the only approximation available without a tz database. */
#define VELES_TIME_OFFSET_UNKNOWN 100000

int64_t veles_time_local_offset_minutes(int64_t secs) {
    time_t t = (time_t)secs;
    struct tm lt, gt;
#if defined(_WIN32)
    if (localtime_s(&lt, &t) != 0 || gmtime_s(&gt, &t) != 0) return VELES_TIME_OFFSET_UNKNOWN;
#else
    if (!localtime_r(&t, &lt) || !gmtime_r(&t, &gt)) return VELES_TIME_OFFSET_UNKNOWN;
#endif
    int64_t l = days_from_civil(lt.tm_year + 1900, lt.tm_mon + 1, lt.tm_mday) * 86400
              + lt.tm_hour * 3600 + lt.tm_min * 60 + lt.tm_sec;
    int64_t g = days_from_civil(gt.tm_year + 1900, gt.tm_mon + 1, gt.tm_mday) * 86400
              + gt.tm_hour * 3600 + gt.tm_min * 60 + gt.tm_sec;
    return (l - g) / 60;
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

/* ---- randomness -------------------------------------------------------- */

/* veles_random_bytes fills out with n bytes from the operating system's
 * cryptographically secure generator: BCryptGenRandom on Windows,
 * getrandom(2) on Linux with a /dev/urandom fallback for kernels that
 * lack it, arc4random_buf on the BSDs and macOS. Returns 0, or an errno
 * value when the system will not produce randomness — a condition
 * std/crypto refuses to paper over. */
int64_t veles_random_bytes(int64_t n, veles_string *out) {
    if (n < 0) return EINVAL;
    char *buf = veles_alloc(n + 1);
    buf[n] = 0;
#if defined(_WIN32)
    if (n > 0) {
        NTSTATUS st = BCryptGenRandom(NULL, (PUCHAR)buf, (ULONG)n, BCRYPT_USE_SYSTEM_PREFERRED_RNG);
        if (st != 0) return EIO;
    }
#elif defined(__linux__)
    int64_t got = 0;
    while (got < n) {
        ssize_t r = getrandom(buf + got, (size_t)(n - got), 0);
        if (r < 0) {
            if (errno == EINTR) continue;
            break;
        }
        got += r;
    }
    if (got < n) {
        /* getrandom is missing (pre-3.17) or refused: /dev/urandom is the
         * same pool, reached the older way */
        FILE *f = fopen("/dev/urandom", "rb");
        if (!f) return errno ? errno : EIO;
        size_t r = fread(buf + got, 1, (size_t)(n - got), f);
        fclose(f);
        if (got + (int64_t)r < n) return EIO;
    }
#else
    if (n > 0) arc4random_buf(buf, (size_t)n);
#endif
    out->data = buf;
    out->len = n;
    return 0;
}
