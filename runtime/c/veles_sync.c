/* Veles runtime — the operating system's threads, locks and condition
 * variables behind the multi-threaded executor (D66) and `Mutex<T>`.
 *
 * Locks are recursive: the executor's runtime lock is taken by every entry
 * point, and one entry point may call another. A condition variable is
 * only ever waited on with its lock held exactly once. */

/* glibc declares its extensions (pthread_getattr_np, ...) only when asked. */
#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif
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

/* ---- `veles test`: one record per running test (D78, D80) -----------------
 * Tests run at once, on any threads, so what a test records is its own: a
 * record is bound as a task-local value of the test's task (under a key
 * veles_local_key never hands out), and every task the test starts
 * inherits it. `expect` and the rest append failure lines to it, `io`
 * output goes to it while it runs, and the runner takes it when the test
 * is done and prints the report in declaration order. */

int64_t veles_time_monotonic_ns(void);
void *veles_alloc_words(int64_t size);
void *veles_alloc(int64_t size);
void *veles_local_find(int64_t key);
void *veles_local_bind(int64_t key, void *cell);
void veles_local_restore(void *head);

#define TEST_REC_KEY (-2)

typedef struct {
    char *data;
    int64_t len;
} veles_test_string;

typedef struct {
    char *data;
    int64_t len, cap;
} text_buf;

typedef struct veles_test_rec {
    veles_lock *lock;
    text_buf fails;  /* the recorded failure lines */
    text_buf output; /* what the test printed, kept for its report */
    int64_t count, stopped;
    int64_t taken;   /* the runner has the report: nothing more is kept */
    const char *name;
    int64_t name_len;
    int64_t deadline; /* monotonic ns while it runs under --timeout; else 0 */
    struct veles_test_rec *next;
} veles_test_rec;

static void buf_append(text_buf *b, const char *s, int64_t n) {
    if (n <= 0) return; /* an empty print: data may still be NULL, and NULL + 0 is undefined in C */
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

/* appends `b` to `to`, each line prefixed with `prefix` */
static void append_indented(text_buf *to, const char *prefix, const text_buf *b) {
    int64_t plen = (int64_t)strlen(prefix);
    for (int64_t i = 0; i < b->len; i++) {
        if (i == 0 || b->data[i - 1] == '\n') buf_append(to, prefix, plen);
        buf_append(to, b->data + i, 1);
    }
    if (b->len > 0 && b->data[b->len - 1] != '\n') buf_append(to, "\n", 1);
}

static veles_test_rec *current_rec(void) {
    return veles_local_find(TEST_REC_KEY);
}

/* every record, for the watchdog; they live until the process ends */
static veles_lock *recs_lock;
static veles_test_rec *recs;
static int64_t tests_total, tests_done;

/* a new record for the test named `name`; the runner binds it around the
 * creation of the test's task (veles_test_bind / veles_test_unbind) */
void *veles_test_new(const char *name, int64_t len) {
    veles_test_rec *r = veles_alloc_words(sizeof *r);
    r->lock = veles_lock_new();
    r->name = name;
    r->name_len = len;
    if (!recs_lock) recs_lock = veles_lock_new();
    veles_lock_acquire(recs_lock);
    r->next = recs;
    recs = r;
    veles_lock_release(recs_lock);
    return r;
}

void *veles_test_bind(void *rec) {
    return veles_local_bind(TEST_REC_KEY, rec);
}

void veles_test_unbind(void *prev) {
    veles_local_restore(prev);
}

/* io.print and friends inside a test: the text is kept, and shown under the
 * test's failure (or dropped when it passes), so a passing run reads as its
 * verdicts alone. Returns 0 outside a test. Standard output and error share
 * the one buffer, in the order they were written. */
int veles_test_capture(const char *s, int64_t len, int newline) {
    veles_test_rec *r = current_rec();
    if (!r) return 0;
    veles_lock_acquire(r->lock);
    if (r->taken) { /* the report is out: nothing would show it */
        veles_lock_release(r->lock);
        return 0;
    }
    buf_append(&r->output, s, len);
    if (newline) buf_append(&r->output, "\n", 1);
    veles_lock_release(r->lock);
    return 1;
}

void veles_test_sites(void (*each)(void *ctx, const char *where, int64_t len), void *ctx);

static void append_site(void *ctx, const char *where, int64_t len) {
    veles_test_rec *r = ctx;
    buf_append(&r->fails, "\n      called from ", 19);
    buf_append(&r->fails, where, len);
}

void veles_test_fail(const char *msg, int64_t len, const char *where, int64_t wlen, int64_t stop) {
    veles_test_rec *r = current_rec();
    if (!r) {
        /* test code running outside `veles test`: nothing collects it */
        fprintf(stderr, "%.*s: %.*s\n", (int)wlen, where, (int)len, msg);
        return;
    }
    veles_lock_acquire(r->lock);
    buf_append(&r->fails, "  ", 2);
    buf_append(&r->fails, where, wlen);
    buf_append(&r->fails, ": ", 2);
    buf_append(&r->fails, msg, len);
    veles_test_sites(append_site, r); /* inside a helper: where the test called it */
    buf_append(&r->fails, "\n", 1);
    r->count++;
    if (stop) r->stopped = 1;
    veles_lock_release(r->lock);
}

/* The failures a test recorded, as one string, and how many; *stopped is
 * set when one ended the test (the panic that did it is not a failure of
 * its own). Every line is indented by `indent` more spaces: the test's depth
 * in its suites (D78). */
int64_t veles_test_take(void *rec, veles_test_string *out, int64_t *stopped, int64_t indent) {
    veles_test_rec *r = rec;
    veles_lock_acquire(r->lock);
    r->taken = 1;
    /* the failure lines, then what the test printed under `output:` —
     * the caller shows it only for a test that failed */
    text_buf all = {0};
    buf_append(&all, r->fails.data, r->fails.len);
    if (r->output.len > 0) {
        buf_append(&all, "  output:\n", 10);
        append_indented(&all, "    ", &r->output);
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
    free(r->fails.data);
    free(r->output.data);
    r->fails = (text_buf){0};
    r->output = (text_buf){0};
    out->data = text;
    out->len = w;
    *stopped = r->stopped;
    int64_t n = r->count;
    veles_lock_release(r->lock);
    return n;
}

/* ---- `veles test --timeout`: a watchdog over the running tests ---------
 * A test is armed when it starts and disarmed when it ends (the runtime's
 * test queue calls in). A test still running at its deadline cannot be
 * stopped — a task busy in a loop never yields — so the watchdog reports
 * it, with what it printed, and ends the process. */

static veles_lock *watch_lock;
static veles_cond *watch_cond;
static int64_t watch_ms;

static void watch_main(void *arg) {
    (void)arg;
    veles_lock_acquire(watch_lock);
    for (;;) {
        int64_t soonest = 0;
        veles_test_rec *late = NULL;
        int64_t now = veles_time_monotonic_ns();
        veles_lock_acquire(recs_lock);
        for (veles_test_rec *r = recs; r; r = r->next) {
            int64_t d = __atomic_load_n(&r->deadline, __ATOMIC_ACQUIRE);
            if (d == 0) continue;
            if (d <= now) late = r;
            if (soonest == 0 || d < soonest) soonest = d;
        }
        veles_lock_release(recs_lock);
        if (!late) {
            veles_cond_wait(watch_cond, watch_lock, soonest == 0 ? -1 : (soonest - now) / 1000000 + 1);
            continue;
        }
        fflush(stdout);
        printf("test %.*s ... ", (int)late->name_len, late->name);
        if (watch_ms % 1000 == 0) {
            printf("FAILED: timed out after %llds\n", (long long)(watch_ms / 1000));
        } else {
            printf("FAILED: timed out after %lldms\n", (long long)watch_ms);
        }
        veles_lock_acquire(late->lock); /* what a hanging test printed is often why */
        if (late->output.len > 0) {
            text_buf shown = {0};
            buf_append(&shown, "  output:\n", 10);
            append_indented(&shown, "    ", &late->output);
            fwrite(shown.data, 1, (size_t)shown.len, stdout);
            free(shown.data);
        }
        veles_lock_release(late->lock);
        int64_t left = tests_total - __atomic_load_n(&tests_done, __ATOMIC_SEQ_CST) - 1;
        printf("\ntimed out: %.*s", (int)late->name_len, late->name);
        if (left > 0) {
            printf("; %lld other test%s did not finish", (long long)left, left == 1 ? "" : "s");
        }
        printf(" (tests run at once; `--jobs 1` runs them one at a time)\n");
        fflush(stdout);
        fflush(stderr);
        _Exit(1);
    }
}

void veles_test_jobs(int64_t jobs);

/* The runner's first call: how many tests may run at once (0: one per
 * worker thread), each one's bound (0: none), and how many there are. */
void veles_gc_root(void *addr, void *desc);

void veles_test_setup(int64_t jobs, int64_t timeout_ms, int64_t total) {
    if (!recs_lock) recs_lock = veles_lock_new();
    veles_gc_root(&recs, NULL); /* the records, until the process ends */
    tests_total = total;
    veles_test_jobs(jobs);
    if (timeout_ms <= 0) return;
    watch_ms = timeout_ms;
    watch_lock = veles_lock_new();
    watch_cond = veles_cond_new();
    if (veles_thread_spawn(watch_main, NULL) != 0) {
        fputs("veles test: cannot start the timeout watchdog\n", stderr);
        exit(101);
    }
}

/* the test whose record `rec` is starts / has ended: arm / disarm it */
void veles_test_watch_rec(void *rec, int64_t running) {
    veles_test_rec *r = rec;
    if (!running) __atomic_add_fetch(&tests_done, 1, __ATOMIC_SEQ_CST);
    if (!r || watch_ms <= 0) return;
    __atomic_store_n(&r->deadline, running ? veles_time_monotonic_ns() + watch_ms * 1000000 : 0, __ATOMIC_RELEASE);
    veles_lock_acquire(watch_lock);
    veles_cond_signal(watch_cond);
    veles_lock_release(watch_lock);
}
