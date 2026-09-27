/* Veles runtime — the operating system's threads, locks and condition
 * variables behind the multi-threaded executor (D66) and `Mutex<T>`.
 *
 * Locks are recursive: the executor's runtime lock is taken by every entry
 * point, and one entry point may call another. A condition variable is
 * only ever waited on with its lock held exactly once. */

#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

#if defined(_WIN32)
#include <windows.h>
#include <intrin.h>
#else
#include <pthread.h>
#include <errno.h>
#include <time.h>
#include <unistd.h>
#endif

#include "veles_tls.h"

/* ---- the per-thread block (veles_tls.h) ---------------------------------- */

#if defined(_WIN32) && defined(__x86_64__)
uint32_t veles_tls_slot;

/* a TLS index below 64, whose value the thread environment block holds
 * inline (higher ones live in a side table __readgsqword cannot reach) */
static void tls_init(void) {
    DWORD taken[64];
    int n = 0;
    for (;;) {
        DWORD i = TlsAlloc();
        if (i == TLS_OUT_OF_INDEXES || n == 64) {
            fputs("veles: no thread-local slot below 64 is free\n", stderr);
            exit(101);
        }
        if (i < 64) {
            veles_tls_slot = (uint32_t)i;
            break;
        }
        taken[n++] = i;
    }
    while (n > 0) TlsFree(taken[--n]);
}
#else
__thread veles_tls *veles_tls_block;

static void tls_init(void) { }
#endif

static void *must(void *p);

#if defined(VELES_CAPTURE_STORED)
/* the registers VELES_CAPTURE_ASM put on the stack (AArch64) */
void veles_capture_store(const uint64_t *regs) {
    veles_tls *t = veles_tls_get();
    for (int i = 0; i < VELES_CAPTURE_STORED; i++) t->capture[i] = regs[i];
}
#endif

veles_tls *veles_tls_first(void) {
    /* calloc aligns to 16, as the jmp_buf inside needs */
    veles_tls *t = must(calloc(1, sizeof *t));
#if defined(_WIN32) && defined(__x86_64__)
    TlsSetValue(veles_tls_slot, t);
#else
    veles_tls_block = t;
#endif
    return t;
}

typedef struct veles_lock {
#if defined(_WIN32)
    CRITICAL_SECTION cs;
#else
    pthread_mutex_t m;
#endif
} veles_lock;

typedef struct veles_cond {
#if defined(_WIN32)
    CONDITION_VARIABLE cv;
#else
    pthread_cond_t cv;
#endif
} veles_cond;

static void *must(void *p) {
    if (!p) {
        fputs("panic: out of memory\n", stderr);
        exit(101);
    }
    return p;
}

veles_lock *veles_lock_new(void) {
    veles_lock *l = must(calloc(1, sizeof *l));
#if defined(_WIN32)
    /* the runtime lock guards short sections taken very often: spin a
     * little before sleeping in the kernel */
    InitializeCriticalSectionAndSpinCount(&l->cs, 2000);
#else
    pthread_mutexattr_t a;
    pthread_mutexattr_init(&a);
    pthread_mutexattr_settype(&a, PTHREAD_MUTEX_RECURSIVE);
    pthread_mutex_init(&l->m, &a);
    pthread_mutexattr_destroy(&a);
#endif
    return l;
}

void veles_lock_acquire(veles_lock *l) {
#if defined(_WIN32)
    EnterCriticalSection(&l->cs);
#else
    pthread_mutex_lock(&l->m);
#endif
}

int64_t veles_lock_try(veles_lock *l) {
#if defined(_WIN32)
    return TryEnterCriticalSection(&l->cs) ? 1 : 0;
#else
    return pthread_mutex_trylock(&l->m) == 0;
#endif
}

void veles_lock_release(veles_lock *l) {
#if defined(_WIN32)
    LeaveCriticalSection(&l->cs);
#else
    pthread_mutex_unlock(&l->m);
#endif
}

veles_cond *veles_cond_new(void) {
    veles_cond *c = must(calloc(1, sizeof *c));
#if defined(_WIN32)
    InitializeConditionVariable(&c->cv);
#else
    pthread_cond_init(&c->cv, NULL);
#endif
    return c;
}

/* waits until signalled or timeout_ms passes (negative: no limit) */
void veles_cond_wait(veles_cond *c, veles_lock *l, int64_t timeout_ms) {
#if defined(_WIN32)
    SleepConditionVariableCS(&c->cv, &l->cs, timeout_ms < 0 ? INFINITE : (DWORD)timeout_ms);
#else
    if (timeout_ms < 0) {
        pthread_cond_wait(&c->cv, &l->m);
        return;
    }
    struct timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    ts.tv_sec += timeout_ms / 1000;
    ts.tv_nsec += (timeout_ms % 1000) * 1000000;
    if (ts.tv_nsec >= 1000000000) {
        ts.tv_sec++;
        ts.tv_nsec -= 1000000000;
    }
    pthread_cond_timedwait(&c->cv, &l->m, &ts);
#endif
}

void veles_cond_signal(veles_cond *c) {
#if defined(_WIN32)
    WakeConditionVariable(&c->cv);
#else
    pthread_cond_signal(&c->cv);
#endif
}

void veles_cond_broadcast(veles_cond *c) {
#if defined(_WIN32)
    WakeAllConditionVariable(&c->cv);
#else
    pthread_cond_broadcast(&c->cv);
#endif
}

typedef struct {
    void (*fn)(void *);
    void *arg;
} thread_start;

#if defined(_WIN32)
static DWORD WINAPI thread_main(LPVOID p) {
    thread_start s = *(thread_start *)p;
    free(p);
    s.fn(s.arg);
    return 0;
}
#else
static void *thread_main(void *p) {
    thread_start s = *(thread_start *)p;
    free(p);
    s.fn(s.arg);
    return NULL;
}
#endif

/* starts a detached OS thread running fn(arg); 0 on success */
int64_t veles_thread_spawn(void (*fn)(void *), void *arg) {
    thread_start *s = must(malloc(sizeof *s));
    s->fn = fn;
    s->arg = arg;
#if defined(_WIN32)
    HANDLE h = CreateThread(NULL, 0, thread_main, s, 0, NULL);
    if (!h) {
        free(s);
        return -1;
    }
    CloseHandle(h);
    return 0;
#else
    pthread_t t;
    if (pthread_create(&t, NULL, thread_main, s) != 0) {
        free(s);
        return -1;
    }
    pthread_detach(t);
    return 0;
#endif
}

int64_t veles_cpu_count(void) {
#if defined(_WIN32)
    SYSTEM_INFO si;
    GetSystemInfo(&si);
    return (int64_t)si.dwNumberOfProcessors;
#else
    long n = sysconf(_SC_NPROCESSORS_ONLN);
    return n > 0 ? (int64_t)n : 1;
#endif
}

/* ---- Mutex<T> (D66) ---------------------------------------------------------
 * The lock of a Mutex (and an Atomic) is one word in the Veles heap:
 * 0 when free, otherwise the holding thread's tag shifted left once, with
 * bit 0 set when a thread may be waiting. Taking a free lock and giving it
 * back are one atomic operation each. A thread that finds it held spins
 * briefly, then parks on one of a few condition variables picked by the
 * word's address, in a safe region: the collector need not wait for it.
 * The tag makes taking a lock the thread already holds — `withLock`
 * inside its own `withLock` — a panic rather than a silent hang; nothing
 * inside `withLock` suspends, so the holder is always this thread. */

void veles_blocking_enter(void);
void veles_blocking_leave(void);
void veles_panic(const char *msg, int64_t len);

#define STRIPES 64
static veles_lock *stripe_locks[STRIPES];
static veles_cond *stripe_conds[STRIPES];
static int64_t next_tag;

void veles_sync_init(void) {
    if (stripe_locks[0]) return;
    tls_init();
    for (int i = 0; i < STRIPES; i++) {
        stripe_locks[i] = veles_lock_new();
        stripe_conds[i] = veles_cond_new();
    }
}

static int64_t tag(void) {
    veles_tls *t = veles_tls_get();
    if (!t->lock_tag) t->lock_tag = __atomic_add_fetch(&next_tag, 1, __ATOMIC_RELAXED);
    return t->lock_tag << 1;
}

static inline void cpu_relax(void) {
#if defined(__x86_64__) || defined(__i386__)
    __builtin_ia32_pause();
#elif defined(__aarch64__)
    __asm__ __volatile__("yield");
#endif
}

static int try_take(int64_t *w, int64_t me) {
    int64_t free_word = 0;
    return __atomic_compare_exchange_n(w, &free_word, me, 0, __ATOMIC_ACQUIRE, __ATOMIC_RELAXED);
}

void veles_mutex_lock(int64_t *w) {
    int64_t me = tag();
    if (try_take(w, me)) return;
    if ((__atomic_load_n(w, __ATOMIC_RELAXED) & ~(int64_t)1) == me) {
        veles_panic("a Mutex was locked again inside its own withLock", 48);
    }
    for (int i = 0; i < 64; i++) {
        cpu_relax();
        if (__atomic_load_n(w, __ATOMIC_RELAXED) == 0 && try_take(w, me)) return;
    }
    size_t s = ((uintptr_t)w >> 4) % STRIPES;
    veles_blocking_enter();
    veles_lock_acquire(stripe_locks[s]);
    for (;;) {
        int64_t old = __atomic_load_n(w, __ATOMIC_RELAXED);
        if (old == 0) {
            /* taken with the waiters bit set: others may still be parked */
            if (__atomic_compare_exchange_n(w, &old, me | 1, 0, __ATOMIC_ACQUIRE, __ATOMIC_RELAXED)) break;
            continue;
        }
        if (!(old & 1) && !__atomic_compare_exchange_n(w, &old, old | 1, 0, __ATOMIC_RELAXED, __ATOMIC_RELAXED)) continue;
        veles_cond_wait(stripe_conds[s], stripe_locks[s], -1);
    }
    veles_lock_release(stripe_locks[s]);
    veles_blocking_leave();
}

void veles_mutex_unlock(int64_t *w) {
    int64_t old = __atomic_exchange_n(w, 0, __ATOMIC_RELEASE);
    if (old & 1) {
        size_t s = ((uintptr_t)w >> 4) % STRIPES;
        veles_lock_acquire(stripe_locks[s]);
        veles_cond_broadcast(stripe_conds[s]);
        veles_lock_release(stripe_locks[s]);
    }
}

/* ---- `veles test --timeout`: a watchdog over the running test ----------
 * The runner arms it before each test with the test's name and disarms it
 * after the last. A test still running at its deadline cannot be stopped
 * (a task busy in a loop never yields), so the watchdog reports it and ends
 * the process; the tests after it do not run, and the report says so. */

int64_t veles_time_monotonic_ns(void);

static veles_lock *watch_lock;
static veles_cond *watch_cond;
static int64_t watch_deadline; /* monotonic ns; 0 while disarmed */
static int64_t watch_ms;
static const char *watch_name;
static int64_t watch_name_len;
static int64_t watch_left; /* tests after the running one */

static void print_captured(void);

static void watch_main(void *arg) {
    (void)arg;
    veles_lock_acquire(watch_lock);
    for (;;) {
        if (watch_deadline == 0) {
            veles_cond_wait(watch_cond, watch_lock, -1);
            continue;
        }
        int64_t now = veles_time_monotonic_ns();
        if (now < watch_deadline) {
            veles_cond_wait(watch_cond, watch_lock, (watch_deadline - now) / 1000000 + 1);
            continue;
        }
        if (watch_ms % 1000 == 0) {
            printf("FAILED: timed out after %llds\n", (long long)(watch_ms / 1000));
        } else {
            printf("FAILED: timed out after %lldms\n", (long long)watch_ms);
        }
        print_captured(); /* what a hanging test printed is often why */
        printf("\ntimed out: %.*s", (int)watch_name_len, watch_name);
        if (watch_left > 0) {
            printf("; %lld test%s after it did not run", (long long)watch_left, watch_left == 1 ? "" : "s");
        }
        printf("\n");
        fflush(stdout);
        fflush(stderr);
        _Exit(1);
    }
}

void veles_test_watch(int64_t timeout_ms, const char *name, int64_t len, int64_t left) {
    if (!watch_lock) {
        if (timeout_ms <= 0) return;
        watch_lock = veles_lock_new();
        watch_cond = veles_cond_new();
        if (veles_thread_spawn(watch_main, NULL) != 0) {
            fputs("veles test: cannot start the timeout watchdog\n", stderr);
            exit(101);
        }
    }
    fflush(stdout); /* the `test name ... ` line is out before a timeout report */
    veles_lock_acquire(watch_lock);
    watch_ms = timeout_ms;
    watch_name = name;
    watch_name_len = len;
    watch_left = left;
    watch_deadline = timeout_ms > 0 ? veles_time_monotonic_ns() + timeout_ms * 1000000 : 0;
    veles_cond_signal(watch_cond);
    veles_lock_release(watch_lock);
}

/* ---- test failures (D78) ---------------------------------------------------
 * `expect`, `require`, `fail` and the rest record a failure here instead of
 * ending the test (a soft `expect` lets the test go on), each line prefixed
 * with the call's location. The runner calls veles_test_begin before a test
 * and veles_test_take after it, and prints what was recorded. A test may
 * record from tasks on any thread, hence the lock. */

typedef struct {
    char *data;
    int64_t len;
} veles_test_string;

void *veles_alloc(int64_t size);

typedef struct {
    char *data;
    int64_t len, cap;
} text_buf;

static veles_lock *fail_lock;
static text_buf fails;   /* the recorded failure lines */
static text_buf output;  /* what the test printed, kept for its report */
static int64_t fail_count, fail_stopped;
static int capturing;    /* a test is running: io output goes to `output` */

static void buf_append(text_buf *b, const char *s, int64_t n) {
    if (b->len + n > b->cap) {
        int64_t cap = b->cap ? b->cap * 2 : 256;
        while (cap < b->len + n) cap *= 2;
        char *grown = realloc(b->data, (size_t)cap);
        if (!grown) return; /* out of memory: the count still says it failed */
        b->data = grown;
        b->cap = cap;
    }
    memcpy(b->data + b->len, s, (size_t)n);
    b->len += n;
}

static void fail_append(const char *s, int64_t n) {
    buf_append(&fails, s, n);
}

void veles_test_begin(void) {
    if (!fail_lock) fail_lock = veles_lock_new();
    veles_lock_acquire(fail_lock);
    fails.len = output.len = fail_count = fail_stopped = 0;
    __atomic_store_n(&capturing, 1, __ATOMIC_RELEASE);
    veles_lock_release(fail_lock);
}

/* io.print and friends while a test runs: the text is kept, and shown
 * under the test's failure (or dropped when it passes), so a passing run
 * reads as its verdicts alone. Returns 0 when no test is running. Standard
 * output and error share the one buffer, in the order they were written. */
int veles_test_capture(const char *s, int64_t len, int newline) {
    if (!__atomic_load_n(&capturing, __ATOMIC_ACQUIRE)) return 0;
    veles_lock_acquire(fail_lock);
    if (!capturing) { /* the test ended while this task waited for the lock */
        veles_lock_release(fail_lock);
        return 0;
    }
    buf_append(&output, s, len);
    if (newline) buf_append(&output, "\n", 1);
    veles_lock_release(fail_lock);
    return 1;
}

/* appends `b` to `to`, each line prefixed with `prefix` */
static void append_indented(text_buf *to, const char *prefix, const text_buf *b) {
    int64_t plen = (int64_t)strlen(prefix);
    for (int64_t i = 0; i < b->len; i++) {
        if (i == 0 || b->data[i - 1] == '\n') buf_append(to, prefix, plen);
        buf_append(to, b->data + i, 1);
    }
    if (b->len > 0 && b->data[b->len - 1] != '\n') buf_append(to, "\n", 1);
}

/* for the watchdog: the timed-out test's output, as the report shows it */
static void print_captured(void) {
    if (!fail_lock) return;
    veles_lock_acquire(fail_lock);
    __atomic_store_n(&capturing, 0, __ATOMIC_RELEASE);
    if (output.len > 0) {
        text_buf shown = {0};
        buf_append(&shown, "  output:\n", 10);
        append_indented(&shown, "    ", &output);
        fwrite(shown.data, 1, (size_t)shown.len, stdout);
        free(shown.data);
    }
    veles_lock_release(fail_lock);
}

void veles_test_sites(void (*each)(const char *where, int64_t len));

static void append_site(const char *where, int64_t len) {
    fail_append("\n      called from ", 19);
    fail_append(where, len);
}

void veles_test_fail(const char *msg, int64_t len, const char *where, int64_t wlen, int64_t stop) {
    if (!fail_lock) {
        /* test code running outside `veles test`: nothing collects it */
        fprintf(stderr, "%.*s: %.*s\n", (int)wlen, where, (int)len, msg);
        return;
    }
    veles_lock_acquire(fail_lock);
    fail_append("  ", 2);
    fail_append(where, wlen);
    fail_append(": ", 2);
    fail_append(msg, len);
    veles_test_sites(append_site); /* inside a helper: where the test called it */
    fail_append("\n", 1);
    fail_count++;
    if (stop) fail_stopped = 1;
    veles_lock_release(fail_lock);
}

/* The failures a test recorded, as one string, and how many; *stopped is
 * set when one ended the test (the panic that did it is not a failure of
 * its own). Every line is indented by `indent` more spaces: the test's depth
 * in its suites (D78). */
int64_t veles_test_take(veles_test_string *out, int64_t *stopped, int64_t indent) {
    veles_lock_acquire(fail_lock);
    __atomic_store_n(&capturing, 0, __ATOMIC_RELEASE);
    /* the failure lines, then what the test printed under `output:` —
     * the caller shows it only for a test that failed */
    text_buf all = {0};
    buf_append(&all, fails.data, fails.len);
    if (output.len > 0) {
        buf_append(&all, "  output:\n", 10);
        append_indented(&all, "    ", &output);
    }
    int64_t lines = 0;
    for (int64_t i = 0; i < all.len; i++) lines += all.data[i] == '\n';
    char *text = veles_alloc(all.len + lines * indent + 1);
    int64_t w = 0;
    for (int64_t i = 0; i < all.len; i++) {
        if (i == 0 || all.data[i - 1] == '\n') {
            memset(text + w, ' ', (size_t)indent);
            w += indent;
        }
        text[w++] = all.data[i];
    }
    text[w] = 0;
    free(all.data);
    out->data = text;
    out->len = w;
    *stopped = fail_stopped;
    int64_t n = fail_count;
    fails.len = output.len = fail_count = fail_stopped = 0;
    veles_lock_release(fail_lock);
    return n;
}
