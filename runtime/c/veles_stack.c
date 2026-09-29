/*
 * Veles runtime — thread stacks and stack overflow (D92).
 *
 * Every thread that runs Veles code gets a big stack, reserved and not
 * committed (the pages appear as the recursion reaches them): 256 MB, or
 * VELES_STACK megabytes. The program's start runs on such a thread too —
 * this file owns the process's `main` and the generated one is called
 * `veles_main` — because the main thread's stack is whatever the OS gave the
 * process (1 MB on Windows, 8 MB on Linux) and cannot be sized from inside.
 *
 * Running out of it is a panic, not a silent death: a fault handler that
 * recognises the guard region prints what any panic prints (`panic: ...`,
 * exit code 101) and, in a debug build, the innermost calls the D81 chain
 * has. It runs on the nearly exhausted stack, so it uses no allocation and
 * no formatted output — a buffer, `write`, and reads of memory that already
 * exists.
 */
#if !defined(_WIN32) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#if defined(_WIN32)
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#else
#include <pthread.h>
#include <signal.h>
#include <unistd.h>
#endif

int32_t veles_main(int32_t argc, char **argv);
int64_t veles_shadow_peek(const char **frames, int64_t max, int64_t *depth, int64_t *kept);
int veles_chain_recorded(void);

#define DEFAULT_STACK_MB 256
#define MAX_STACK_MB 4096
#define SMALLEST_STACK (8u << 20) /* a refused reservation is retried at half down to this, then the program runs on the stack the OS gave it */

static size_t stack_bytes;

/* the size of a Veles thread's stack: VELES_STACK, in megabytes */
size_t veles_stack_size(void) {
    if (stack_bytes) return stack_bytes;
    unsigned long long mb = DEFAULT_STACK_MB;
    const char *env = getenv("VELES_STACK");
    if (env && *env) {
        char *end = NULL;
        unsigned long long v = strtoull(env, &end, 10);
        if (*end == 0 && v >= 1 && v <= MAX_STACK_MB) {
            mb = v;
        } else {
            fprintf(stderr, "veles: VELES_STACK=%s is not a number of megabytes from 1 to %d; using %d\n", env, MAX_STACK_MB, DEFAULT_STACK_MB);
        }
    }
    stack_bytes = (size_t)mb << 20;
    return stack_bytes;
}

/* ---- the report ------------------------------------------------------------ */

static size_t put(char *buf, size_t cap, size_t n, const char *s) {
    while (*s && n + 1 < cap) buf[n++] = *s++;
    return n;
}

static size_t put_num(char *buf, size_t cap, size_t n, unsigned long long v) {
    char digits[24];
    int k = 0;
    do { digits[k++] = (char)('0' + v % 10); v /= 10; } while (v && k < 23);
    while (k && n + 1 < cap) buf[n++] = digits[--k];
    return n;
}

/* "callee\0" of a record "site\0callee\0" */
static const char *callee_of(const char *rec) {
    return rec + strlen(rec) + 1;
}

/* the panic, as text: what the thread's stack was and, in a debug build,
 * the innermost calls of the chain, laid out as a panic's are */
static size_t compose(char *buf, size_t cap, size_t stack) {
    size_t n = 0;
    n = put(buf, cap, n, "panic: stack overflow\n  the stack is ");
    n = put_num(buf, cap, n, stack >> 20);
    n = put(buf, cap, n, " MB; VELES_STACK=<megabytes> sets it\n");
    const char *frames[4];
    int64_t depth = 0, kept = 0;
    int64_t got = veles_shadow_peek(frames, 4, &depth, &kept);
    if (!veles_chain_recorded()) {
        n = put(buf, cap, n, "  (a debug build shows the call chain)\n");
    } else if (got > 0) {
        n = put(buf, cap, n, "  in ");
        n = put(buf, cap, n, callee_of(frames[0]));
        n = put(buf, cap, n, "\n");
        /* the outermost record is the program's first frame and has no site */
        for (int64_t i = 0; i + 1 < got; i++) {
            n = put(buf, cap, n, "  called from ");
            n = put(buf, cap, n, frames[i]);
            n = put(buf, cap, n, " in ");
            n = put(buf, cap, n, callee_of(frames[i + 1]));
            n = put(buf, cap, n, "\n");
        }
        if (depth > kept) {
            n = put(buf, cap, n, "  (");
            n = put_num(buf, cap, n, (unsigned long long)depth);
            n = put(buf, cap, n, " calls deep; the chain keeps the first ");
            n = put_num(buf, cap, n, (unsigned long long)kept);
            n = put(buf, cap, n, ")\n");
        }
    }
    return n;
}

/* ---- Windows ---------------------------------------------------------------- */
#if defined(_WIN32)

static LONG WINAPI on_exception(EXCEPTION_POINTERS *ep) {
    if (ep->ExceptionRecord->ExceptionCode != EXCEPTION_STACK_OVERFLOW) return EXCEPTION_CONTINUE_SEARCH;
    char buf[1024];
    size_t n = compose(buf, sizeof buf, stack_bytes);
    fflush(stdout);
    DWORD written;
    WriteFile(GetStdHandle(STD_ERROR_HANDLE), buf, (DWORD)n, &written, NULL);
    /* a call that needs as little stack as any: `_exit` may run past what is left */
    TerminateProcess(GetCurrentProcess(), 101);
    return EXCEPTION_CONTINUE_SEARCH;
}

static void install_handler(void) {
    AddVectoredExceptionHandler(1, on_exception);
}

/* what a thread does once, first: leave room for the handler to run when
 * the guard page is hit */
void veles_stack_thread_init(void) {
    ULONG room = 64 * 1024;
    SetThreadStackGuarantee(&room);
}

void veles_stack_thread_done(void) {}

static DWORD WINAPI main_thread(LPVOID p);

typedef struct {
    int32_t argc;
    char **argv;
    int32_t code;
} main_args;

static DWORD WINAPI main_thread(LPVOID p) {
    main_args *a = p;
    veles_stack_thread_init();
    a->code = veles_main(a->argc, a->argv);
    return 0;
}

static int run_on_thread(size_t size, main_args *a) {
    HANDLE h = CreateThread(NULL, size, main_thread, a, STACK_SIZE_PARAM_IS_A_RESERVATION, NULL);
    if (!h) return 0;
    WaitForSingleObject(h, INFINITE);
    CloseHandle(h);
    return 1;
}

/* ---- POSIX ------------------------------------------------------------------ */
#else

#define ALT_STACK_SIZE (64 * 1024)

static __thread char *stack_lo;    /* the lowest usable stack address of this thread */
static __thread size_t stack_size; /* and how big the stack is */
static __thread void *alt_memory;

static void on_fault(int sig, siginfo_t *si, void *ctx) {
    (void)ctx;
    char *addr = (char *)si->si_addr;
    /* the guard region lies just under the stack; a frame that jumps it lands
     * a little further down. Anything else is an ordinary crash. */
    if (stack_lo && addr >= stack_lo - (1 << 20) && addr < stack_lo + (64 << 10)) {
        char buf[1024];
        size_t n = compose(buf, sizeof buf, stack_size);
        fflush(stdout);
        ssize_t w = write(2, buf, n);
        (void)w;
        _exit(101);
    }
    struct sigaction dfl;
    memset(&dfl, 0, sizeof dfl);
    dfl.sa_handler = SIG_DFL;
    sigaction(sig, &dfl, NULL);
}

static void install_handler(void) {
    struct sigaction sa;
    memset(&sa, 0, sizeof sa);
    sa.sa_sigaction = on_fault;
    sa.sa_flags = SA_SIGINFO | SA_ONSTACK;
    sigemptyset(&sa.sa_mask);
    sigaction(SIGSEGV, &sa, NULL);
    sigaction(SIGBUS, &sa, NULL);
}

/* what a thread does once, first: learn where its stack is and give the
 * handler a stack of its own to run on */
void veles_stack_thread_init(void) {
#if defined(__APPLE__)
    stack_size = pthread_get_stacksize_np(pthread_self());
    stack_lo = (char *)pthread_get_stackaddr_np(pthread_self()) - stack_size;
#else
    pthread_attr_t attr;
    void *addr = NULL;
    size_t size = 0;
    if (pthread_getattr_np(pthread_self(), &attr) == 0) {
        pthread_attr_getstack(&attr, &addr, &size);
        pthread_attr_destroy(&attr);
        stack_lo = addr;
        stack_size = size;
    }
#endif
    alt_memory = malloc(ALT_STACK_SIZE);
    if (alt_memory) {
        stack_t ss;
        ss.ss_sp = alt_memory;
        ss.ss_size = ALT_STACK_SIZE;
        ss.ss_flags = 0;
        sigaltstack(&ss, NULL);
    }
}

void veles_stack_thread_done(void) {
    if (!alt_memory) return;
    stack_t ss;
    memset(&ss, 0, sizeof ss);
    ss.ss_flags = SS_DISABLE;
    sigaltstack(&ss, NULL);
    free(alt_memory);
    alt_memory = NULL;
    stack_lo = NULL;
}

typedef struct {
    int32_t argc;
    char **argv;
    int32_t code;
} main_args;

static void *main_thread(void *p) {
    main_args *a = p;
    veles_stack_thread_init();
    a->code = veles_main(a->argc, a->argv);
    return NULL;
}

static int run_on_thread(size_t size, main_args *a) {
    pthread_attr_t attr;
    pthread_t t;
    if (pthread_attr_init(&attr) != 0) return 0;
    if (pthread_attr_setstacksize(&attr, size) != 0 || pthread_create(&t, &attr, main_thread, a) != 0) {
        pthread_attr_destroy(&attr);
        return 0;
    }
    pthread_attr_destroy(&attr);
    pthread_join(t, NULL);
    return 1;
}

#endif

/* the process's entry: the program runs on a thread with the big stack; if
 * the system will not reserve one, on a smaller one, and last of all on the
 * stack the process started with */
int main(int argc, char **argv) {
    size_t size = veles_stack_size();
    install_handler();
    main_args a = {argc, argv, 0};
    for (;;) {
        if (run_on_thread(size, &a)) return a.code;
        if (size <= SMALLEST_STACK) break;
        size = size / 2 >= SMALLEST_STACK ? size / 2 : SMALLEST_STACK;
    }
    return veles_main(argc, argv);
}
