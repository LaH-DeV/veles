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
#include <sched.h>
#include <sys/resource.h>
#if defined(__linux__)
#include <sys/syscall.h>
#endif
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

/* the block of a thread that is ending (thread_main, below), unless the
 * collector still scans it: a thread that never detached */
static void tls_release(void) {
#if defined(_WIN32) && defined(__x86_64__)
    veles_tls *t = (veles_tls *)__readgsqword(0x1480 + veles_tls_slot * 8);
    if (!t || t->thread) return;
    TlsSetValue(veles_tls_slot, NULL);
#else
    veles_tls *t = veles_tls_block;
    if (!t || t->thread) return;
    veles_tls_block = NULL;
#endif
    free(t);
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
    /* a few tries before sleeping in the kernel, as the Windows critical
     * section's spin count does: glibc's mutex sleeps at once, and the
     * runtime lock's sections are short (with many threads the futex
     * round trips were most of the executor's time, bench/httphello) */
    for (int i = 0; i < 64; i++) {
        if (pthread_mutex_trylock(&l->m) == 0) return;
#if defined(__x86_64__) || defined(__i386__)
        __builtin_ia32_pause();
#elif defined(__aarch64__)
        __asm__ __volatile__("yield");
#endif
    }
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
#elif defined(__APPLE__)
    pthread_cond_init(&c->cv, NULL);
#else
    /* timed waits count on the monotonic clock: setting the wall clock
     * neither ends one early nor stretches it */
    pthread_condattr_t attr;
    pthread_condattr_init(&attr);
    pthread_condattr_setclock(&attr, CLOCK_MONOTONIC);
    pthread_cond_init(&c->cv, &attr);
    pthread_condattr_destroy(&attr);
#endif
    return c;
}

/* waits until signalled or timeout_ns passes (negative: no limit), to the
 * platform's resolution: a few tens of microseconds on Linux; on Windows
 * whole milliseconds, rounded up, at the timer period veles_timer_period
 * asked for (F9) */
void veles_cond_wait_ns(veles_cond *c, veles_lock *l, int64_t timeout_ns) {
#if defined(_WIN32)
    DWORD ms = INFINITE;
    if (timeout_ns >= 0) {
        int64_t m = (timeout_ns + 999999) / 1000000;
        ms = m > 0x7ffffffe ? 0x7ffffffe : (DWORD)m;
    }
    SleepConditionVariableCS(&c->cv, &l->cs, ms);
#else
    if (timeout_ns < 0) {
        pthread_cond_wait(&c->cv, &l->m);
        return;
    }
    struct timespec ts;
#if defined(__APPLE__)
    clock_gettime(CLOCK_REALTIME, &ts);
#else
    clock_gettime(CLOCK_MONOTONIC, &ts);
#endif
    ts.tv_sec += timeout_ns / 1000000000;
    ts.tv_nsec += timeout_ns % 1000000000;
    if (ts.tv_nsec >= 1000000000) {
        ts.tv_sec++;
        ts.tv_nsec -= 1000000000;
    }
    pthread_cond_timedwait(&c->cv, &l->m, &ts);
#endif
}

/* waits until signalled or timeout_ms passes (negative: no limit) */
void veles_cond_wait(veles_cond *c, veles_lock *l, int64_t timeout_ms) {
    veles_cond_wait_ns(c, l, timeout_ms < 0 ? -1 : timeout_ms * 1000000);
}

/* ---- parking a thread (the executor's idle threads) -------------------------
 * Each idle thread waits on a park of its own and is woken by name: a wake
 * reaches the one thread chosen, takes no lock, and a wake that comes first
 * makes the next wait return at once. A wait may also return early, so
 * the caller always looks again. Timed waits are as fine as the platform
 * allows: a futex's on Linux; on Windows a high-resolution waitable timer
 * (Windows 10 1803 and later, about half a millisecond), else the 1 ms
 * timer period; a condition variable elsewhere. (One condition variable per
 * executor, under the runtime lock, woke whichever waiter the system picked
 * and kept every timeout to whole milliseconds on Windows.) */
#if defined(__linux__)
#include <linux/futex.h>
#endif

typedef struct veles_park {
#if defined(_WIN32)
    HANDLE event; /* auto-reset */
    HANDLE timer; /* high-resolution; NULL where the system has none */
#elif defined(__linux__)
    int32_t word; /* 1: a wake is pending */
#else
    pthread_mutex_t m;
    pthread_cond_t c;
    int pending;
#endif
} veles_park;

#if defined(_WIN32) && !defined(CREATE_WAITABLE_TIMER_HIGH_RESOLUTION)
#define CREATE_WAITABLE_TIMER_HIGH_RESOLUTION 0x00000002
#endif

veles_park *veles_park_new(void) {
    veles_park *p = must(calloc(1, sizeof *p));
#if defined(_WIN32)
    p->event = CreateEventW(NULL, FALSE, FALSE, NULL);
    if (!p->event) must(NULL);
    p->timer = CreateWaitableTimerExW(NULL, NULL, CREATE_WAITABLE_TIMER_HIGH_RESOLUTION, TIMER_ALL_ACCESS);
#elif defined(__linux__)
    (void)0;
#else
    pthread_mutex_init(&p->m, NULL);
    pthread_cond_init(&p->c, NULL);
#endif
    return p;
}

/* waits until woken or timeout_ns passes (negative: no limit) */
void veles_park_wait(veles_park *p, int64_t timeout_ns) {
#if defined(_WIN32)
    if (timeout_ns < 0) {
        WaitForSingleObject(p->event, INFINITE);
        return;
    }
    if (p->timer) {
        LARGE_INTEGER due;
        due.QuadPart = -(timeout_ns / 100 > 0 ? timeout_ns / 100 : 1); /* relative, in 100 ns */
        if (SetWaitableTimer(p->timer, &due, 0, NULL, NULL, FALSE)) {
            HANDLE both[2] = {p->event, p->timer};
            WaitForMultipleObjects(2, both, FALSE, INFINITE);
            return;
        }
    }
    int64_t ms = (timeout_ns + 999999) / 1000000;
    WaitForSingleObject(p->event, ms > 0x7ffffffe ? 0x7ffffffe : (DWORD)ms);
#elif defined(__linux__)
    if (__atomic_exchange_n(&p->word, 0, __ATOMIC_ACQUIRE)) return;
    struct timespec ts, *tp = NULL;
    if (timeout_ns >= 0) {
        ts.tv_sec = timeout_ns / 1000000000;
        ts.tv_nsec = timeout_ns % 1000000000;
        tp = &ts;
    }
    syscall(SYS_futex, &p->word, FUTEX_WAIT_PRIVATE, 0, tp, NULL, 0);
    __atomic_store_n(&p->word, 0, __ATOMIC_RELEASE);
#else
    pthread_mutex_lock(&p->m);
    if (!p->pending) {
        if (timeout_ns < 0) {
            pthread_cond_wait(&p->c, &p->m);
        } else {
            struct timespec ts;
            clock_gettime(CLOCK_REALTIME, &ts);
            ts.tv_sec += timeout_ns / 1000000000;
            ts.tv_nsec += timeout_ns % 1000000000;
            if (ts.tv_nsec >= 1000000000) {
                ts.tv_sec++;
                ts.tv_nsec -= 1000000000;
            }
            pthread_cond_timedwait(&p->c, &p->m, &ts);
        }
    }
    p->pending = 0;
    pthread_mutex_unlock(&p->m);
#endif
}

void veles_park_wake(veles_park *p) {
#if defined(_WIN32)
    SetEvent(p->event);
#elif defined(__linux__)
    if (__atomic_exchange_n(&p->word, 1, __ATOMIC_RELEASE) == 0) syscall(SYS_futex, &p->word, FUTEX_WAKE_PRIVATE, 1, NULL, NULL, 0);
#else
    pthread_mutex_lock(&p->m);
    p->pending = 1;
    pthread_cond_signal(&p->c);
    pthread_mutex_unlock(&p->m);
#endif
}

/* Windows wakes a waiting thread on its timer tick, 15.6 ms unless a
 * process asks for less; the executor's timers want milliseconds, so the
 * runtime asks for 1 ms once (per process since Windows 10 2004). winmm is
 * loaded for it, not linked. Elsewhere waits are precise already. */
void veles_timer_period(void) {
#if defined(_WIN32)
    HMODULE winmm = LoadLibraryW(L"winmm.dll");
    if (!winmm) return;
    typedef UINT (WINAPI *begin_fn)(UINT);
    begin_fn begin = (begin_fn)(void *)GetProcAddress(winmm, "timeBeginPeriod");
    if (begin) begin(1);
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

void veles_stack_thread_init(void);
void veles_stack_thread_done(void);
size_t veles_stack_size(void);

#if defined(_WIN32)
static DWORD WINAPI thread_main(LPVOID p) {
    thread_start s = *(thread_start *)p;
    free(p);
    veles_stack_thread_init();
    s.fn(s.arg);
    veles_stack_thread_done();
    tls_release();
    return 0;
}
#else
static void *thread_main(void *p) {
    thread_start s = *(thread_start *)p;
    free(p);
    veles_stack_thread_init();
    s.fn(s.arg);
    veles_stack_thread_done();
    tls_release();
    return NULL;
}
#endif

/* one attempt at a detached OS thread with a stack of `size` bytes (0: the
 * system's default); 0 on success */
static int64_t spawn_once(thread_start *s, size_t size) {
#if defined(_WIN32)
    HANDLE h = CreateThread(NULL, size, thread_main, s, size ? STACK_SIZE_PARAM_IS_A_RESERVATION : 0, NULL);
    if (!h) return -1;
    CloseHandle(h);
    return 0;
#else
    pthread_attr_t attr;
    pthread_t t;
    if (pthread_attr_init(&attr) != 0) return -1;
    if (size && pthread_attr_setstacksize(&attr, size) != 0) {
        pthread_attr_destroy(&attr);
        return -1;
    }
    int rc = pthread_create(&t, &attr, thread_main, s);
    pthread_attr_destroy(&attr);
    if (rc != 0) return -1;
    pthread_detach(t);
    return 0;
#endif
}

/* starts a detached OS thread running fn(arg) on a stack of `size` bytes; a
 * reservation the system refuses is retried at half, down to 8 MB, and then
 * at the default. 0 on success */
static int64_t spawn_sized(void (*fn)(void *), void *arg, size_t size) {
    for (;;) {
        thread_start *s = must(malloc(sizeof *s));
        s->fn = fn;
        s->arg = arg;
        if (spawn_once(s, size) == 0) return 0;
        free(s);
        if (size == 0) return -1;
        size = size / 2 >= (8u << 20) ? size / 2 : 0;
    }
}

/* a thread that runs Veles code: the big stack (veles_stack.c) */
int64_t veles_thread_spawn(void (*fn)(void *), void *arg) {
    return spawn_sized(fn, arg, veles_stack_size());
}

/* a thread that runs Veles code on a stack of `size` bytes (0: the big
 * stack a worker has) — `Thread.start(stackSize:)`, D143 */
int64_t veles_thread_spawn_sized(void (*fn)(void *), void *arg, int64_t size) {
    return spawn_sized(fn, arg, size > 0 ? (size_t)size : veles_stack_size());
}

/* a helper thread that runs only runtime code and needs no deep stack */
int64_t veles_thread_spawn_small(void (*fn)(void *), void *arg) {
    return spawn_sized(fn, arg, 0);
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

/* ---- thread settings (D143) --------------------------------------------------
 * A thread of an executor, or one `Thread.start` makes, sets its own name
 * (seen in debuggers and profilers), priority and CPUs as it starts. What
 * the OS refuses is reported, never ignored: 0, or -1 with the reason in
 * err. Priorities: 0 Low, 1 Normal (left as the thread was made), 2 High,
 * 3 Realtime. */
static const char *priority_names[] = {"Low", "Normal", "High", "Realtime"};

/* CPUs the machine has, online or not: the numbers an affinity may name */
static int64_t cpus_configured(void) {
#if defined(_WIN32)
    DWORD n = GetActiveProcessorCount(ALL_PROCESSOR_GROUPS); /* every processor group's */
    return n > 0 ? (int64_t)n : veles_cpu_count();
#else
    long n = sysconf(_SC_NPROCESSORS_CONF);
    return n > 0 ? (int64_t)n : veles_cpu_count();
#endif
}

#if defined(_WIN32)
typedef HRESULT (WINAPI *set_description_fn)(HANDLE, PCWSTR);

/* a CPU's processor group and its number there: the CPUs are numbered from
 * 0 through every group in turn (Windows keeps at most 64 to a group) */
static int cpu_group_of(int32_t cpu, WORD *group, BYTE *number) {
    WORD groups = GetActiveProcessorGroupCount();
    int32_t base = 0;
    for (WORD g = 0; g < groups; g++) {
        int32_t n = (int32_t)GetActiveProcessorCount(g);
        if (cpu < base + n) {
            *group = g;
            *number = (BYTE)(cpu - base);
            return 1;
        }
        base += n;
    }
    return 0;
}
#endif

int64_t veles_thread_configure(const char *name, int64_t priority, const int32_t *cpus, int64_t ncpus, char *err, int64_t cap) {
    if (priority < 0 || priority > 3) priority = 1;
    int64_t have = cpus_configured();
    for (int64_t i = 0; i < ncpus; i++) {
        if (cpus[i] >= have) {
            snprintf(err, (size_t)cap, "there is no CPU %d: this machine has %lld (0 to %lld)", cpus[i], (long long)have, (long long)have - 1);
            return -1;
        }
    }
#if defined(_WIN32)
    if (name && *name) {
        /* Windows 10 1607 and later; looked up, so older systems run unnamed */
        set_description_fn set = (set_description_fn)(void *)GetProcAddress(GetModuleHandleW(L"kernel32.dll"), "SetThreadDescription");
        if (set) {
            wchar_t wide[64];
            int n = MultiByteToWideChar(CP_UTF8, 0, name, -1, wide, 64);
            if (n > 0) set(GetCurrentThread(), wide);
        }
    }
    if (priority != 1) {
        static const int levels[] = {THREAD_PRIORITY_BELOW_NORMAL, THREAD_PRIORITY_NORMAL, THREAD_PRIORITY_HIGHEST, THREAD_PRIORITY_TIME_CRITICAL};
        if (!SetThreadPriority(GetCurrentThread(), levels[priority])) {
            snprintf(err, (size_t)cap, "the system refused priority %s (error %lu)", priority_names[priority], (unsigned long)GetLastError());
            return -1;
        }
    }
    if (ncpus > 0) {
        /* a thread runs in one processor group: the CPUs listed must share
         * it (Windows 11's CPU sets reach across groups, but only as a
         * preference the scheduler may set aside) */
        GROUP_AFFINITY ga;
        memset(&ga, 0, sizeof ga);
        for (int64_t i = 0; i < ncpus; i++) {
            WORD group;
            BYTE number;
            if (!cpu_group_of(cpus[i], &group, &number)) {
                snprintf(err, (size_t)cap, "there is no CPU %d: this machine has %lld (0 to %lld)", cpus[i], (long long)have, (long long)have - 1);
                return -1;
            }
            if (i > 0 && group != ga.Group) {
                snprintf(err, (size_t)cap, "CPUs %d and %d are in different processor groups (%u and %u): a thread on Windows runs in one", cpus[0], cpus[i], (unsigned)ga.Group, (unsigned)group);
                return -1;
            }
            ga.Group = group;
            ga.Mask |= (KAFFINITY)1 << number;
        }
        if (!SetThreadGroupAffinity(GetCurrentThread(), &ga, NULL)) {
            snprintf(err, (size_t)cap, "the system refused the CPUs asked for (error %lu)", (unsigned long)GetLastError());
            return -1;
        }
    }
    return 0;
#else
#if defined(__linux__)
    if (name && *name) {
        char short_name[16]; /* the kernel keeps 15 bytes */
        snprintf(short_name, sizeof short_name, "%s", name);
        pthread_setname_np(pthread_self(), short_name);
    }
    if (priority == 0 || priority == 2) {
        /* a thread's nice value: Linux sets it per thread */
        if (setpriority(PRIO_PROCESS, (id_t)syscall(SYS_gettid), priority == 0 ? 10 : -10) != 0) {
            snprintf(err, (size_t)cap, "the system refused priority %s: %s%s", priority_names[priority], strerror(errno),
                     errno == EACCES || errno == EPERM ? " (raising a thread's priority needs CAP_SYS_NICE)" : "");
            return -1;
        }
    } else if (priority == 3) {
        struct sched_param sp = {0};
        sp.sched_priority = sched_get_priority_min(SCHED_FIFO) + 1;
        int rc = pthread_setschedparam(pthread_self(), SCHED_FIFO, &sp);
        if (rc != 0) {
            snprintf(err, (size_t)cap, "the system refused priority Realtime: %s%s", strerror(rc),
                     rc == EPERM ? " (real-time scheduling needs CAP_SYS_NICE or an RLIMIT_RTPRIO)" : "");
            return -1;
        }
    }
    if (ncpus > 0) {
        cpu_set_t set;
        CPU_ZERO(&set);
        for (int64_t i = 0; i < ncpus; i++) {
            if (cpus[i] >= CPU_SETSIZE) {
                snprintf(err, (size_t)cap, "CPU %d is past the %d an affinity can name", cpus[i], CPU_SETSIZE);
                return -1;
            }
            CPU_SET(cpus[i], &set);
        }
        if (sched_setaffinity(0, sizeof set, &set) != 0) {
            snprintf(err, (size_t)cap, "the system refused the CPUs asked for: %s", strerror(errno));
            return -1;
        }
    }
    return 0;
#else
    /* macOS (plan A7): a name only */
#if defined(__APPLE__)
    if (name && *name) pthread_setname_np(name);
#endif
    if (priority != 1) {
        snprintf(err, (size_t)cap, "priority %s cannot be set on this system yet", priority_names[priority]);
        return -1;
    }
    if (ncpus > 0) {
        snprintf(err, (size_t)cap, "this system cannot keep a thread to chosen CPUs");
        return -1;
    }
    return 0;
#endif
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

/* a held Mutex shields its holder: a loop's back edge takes no
 * cancellation and no yield inside a lock region (D145) */
static void mutex_lock(int64_t *w);

void veles_mutex_lock(int64_t *w) {
    mutex_lock(w);
    veles_tls_get()->shield++;
}

static void mutex_lock(int64_t *w) {
    int64_t me = tag();
    if (try_take(w, me)) return;
    if ((__atomic_load_n(w, __ATOMIC_RELAXED) & ~(int64_t)1) == me) {
        veles_panic("a Mutex was locked again while this task holds it", 49);
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
    veles_tls *tls = veles_tls_get();
    if (tls->shield > 0) tls->shield--;
    int64_t old = __atomic_exchange_n(w, 0, __ATOMIC_RELEASE);
    if (old & 1) {
        size_t s = ((uintptr_t)w >> 4) % STRIPES;
        veles_lock_acquire(stripe_locks[s]);
        veles_cond_broadcast(stripe_conds[s]);
        veles_lock_release(stripe_locks[s]);
    }
}

/* ---- RwLock<T> (D146) -------------------------------------------------------
 * Two words in the Veles heap: the state — bit 0 a writer holds it, bit 1
 * a thread may be parked, and the readers counted from bit 2 — and the
 * number of writers waiting. A reader enters only while no writer holds
 * or waits, so a stream of readers cannot starve a writer. Waiting is as a
 * Mutex's: a short spin, then a park on the stripe's condition variable in
 * a safe region. A held RwLock shields its holder, as a Mutex does (D145).
 * The thread remembers the RwLocks it holds: taking one again — a read
 * inside a read or a write, a write inside either — panics rather than
 * hanging (nothing inside a lock region suspends, so the holder is this
 * thread). */

#define RW_WRITER 1
#define RW_PARKED 2
#define RW_READER 4

static void rw_check_free(veles_tls *tls, int64_t *rw) {
    int64_t n = tls->rw_held_n < 8 ? tls->rw_held_n : 8;
    for (int64_t i = 0; i < n; i++) {
        if (tls->rw_held[i] == rw) {
            veles_panic("an RwLock was locked again while this task holds it", 51);
        }
    }
}

static void rw_note(veles_tls *tls, int64_t *rw) {
    if (tls->rw_held_n < 8) tls->rw_held[tls->rw_held_n] = rw;
    tls->rw_held_n++;
    tls->shield++;
}

static void rw_forget(veles_tls *tls, int64_t *rw) {
    if (tls->shield > 0) tls->shield--;
    if (tls->rw_held_n == 0) return;
    int64_t n = tls->rw_held_n < 8 ? tls->rw_held_n : 8;
    for (int64_t i = n - 1; i >= 0; i--) {
        if (tls->rw_held[i] == rw) {
            for (int64_t j = i; j + 1 < n; j++) tls->rw_held[j] = tls->rw_held[j + 1];
            break;
        }
    }
    tls->rw_held_n--;
}

/* may a reader enter, seeing state s? */
static int rw_readable(int64_t *rw, int64_t s) {
    return !(s & RW_WRITER) && __atomic_load_n(&rw[1], __ATOMIC_RELAXED) == 0;
}

/* parks until try_enter succeeds: under the stripe's lock, set the parked
 * bit before sleeping, so a release that changes the state after the
 * check finds it and wakes the stripe */
static void rw_park(int64_t *rw, int (*try_enter)(int64_t *rw, int64_t s)) {
    size_t st = ((uintptr_t)rw >> 4) % STRIPES;
    veles_blocking_enter();
    veles_lock_acquire(stripe_locks[st]);
    for (;;) {
        int64_t s = __atomic_load_n(&rw[0], __ATOMIC_RELAXED);
        if (try_enter(rw, s)) break;
        if (!(s & RW_PARKED) && !__atomic_compare_exchange_n(&rw[0], &s, s | RW_PARKED, 0, __ATOMIC_RELAXED, __ATOMIC_RELAXED)) continue;
        veles_cond_wait(stripe_conds[st], stripe_locks[st], -1);
    }
    veles_lock_release(stripe_locks[st]);
    veles_blocking_leave();
}

static void rw_wake(int64_t *rw) {
    size_t st = ((uintptr_t)rw >> 4) % STRIPES;
    veles_lock_acquire(stripe_locks[st]);
    veles_cond_broadcast(stripe_conds[st]);
    veles_lock_release(stripe_locks[st]);
}

static int rw_try_read(int64_t *rw, int64_t s) {
    return rw_readable(rw, s) && __atomic_compare_exchange_n(&rw[0], &s, s + RW_READER, 0, __ATOMIC_ACQUIRE, __ATOMIC_RELAXED);
}

/* a writer enters when nobody holds it; the parked bit stays, for the
 * others still asleep */
static int rw_try_write(int64_t *rw, int64_t s) {
    return (s & ~(int64_t)RW_PARKED) == 0 && __atomic_compare_exchange_n(&rw[0], &s, s | RW_WRITER, 0, __ATOMIC_ACQUIRE, __ATOMIC_RELAXED);
}

void veles_rw_read_lock(int64_t *rw) {
    veles_tls *tls = veles_tls_get();
    rw_check_free(tls, rw);
    for (int i = 0; i < 64; i++) {
        if (rw_try_read(rw, __atomic_load_n(&rw[0], __ATOMIC_RELAXED))) {
            rw_note(tls, rw);
            return;
        }
        cpu_relax();
    }
    rw_park(rw, rw_try_read);
    rw_note(tls, rw);
}

void veles_rw_read_unlock(int64_t *rw) {
    rw_forget(veles_tls_get(), rw);
    int64_t s = __atomic_sub_fetch(&rw[0], RW_READER, __ATOMIC_RELEASE);
    /* the last reader out wakes a waiting writer (and the readers parked
     * behind it, who look again) */
    if (s == RW_PARKED && __atomic_compare_exchange_n(&rw[0], &s, 0, 0, __ATOMIC_RELAXED, __ATOMIC_RELAXED)) rw_wake(rw);
}

void veles_rw_write_lock(int64_t *rw) {
    veles_tls *tls = veles_tls_get();
    rw_check_free(tls, rw);
    int64_t s = 0;
    if (__atomic_compare_exchange_n(&rw[0], &s, RW_WRITER, 0, __ATOMIC_ACQUIRE, __ATOMIC_RELAXED)) {
        rw_note(tls, rw);
        return;
    }
    /* waiting: no new reader enters from now on */
    __atomic_add_fetch(&rw[1], 1, __ATOMIC_SEQ_CST);
    int entered = 0;
    for (int i = 0; i < 64 && !entered; i++) {
        cpu_relax();
        entered = rw_try_write(rw, __atomic_load_n(&rw[0], __ATOMIC_RELAXED));
    }
    if (!entered) rw_park(rw, rw_try_write);
    __atomic_sub_fetch(&rw[1], 1, __ATOMIC_SEQ_CST);
    rw_note(tls, rw);
}

void veles_rw_write_unlock(int64_t *rw) {
    rw_forget(veles_tls_get(), rw);
    int64_t old = __atomic_exchange_n(&rw[0], 0, __ATOMIC_RELEASE);
    if (old & RW_PARKED) rw_wake(rw);
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
    if (veles_thread_spawn_small(watch_main, NULL) != 0) {
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
