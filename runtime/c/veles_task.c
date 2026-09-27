/*
 * Veles tasks (build plan Stage 4; spec D2, D3, D16, D34, D36, D38).
 *
 * Suspending functions are LLVM switched-resume coroutines whose frames
 * live on the GC heap. This file is the executor that resumes them on a
 * pool of worker threads (D66): run queues per worker, blocking on channels /
 * tasks / timers / scopes / sockets, and fail-fast scopes with cancellation.
 * Every object here is
 * allocated on the GC heap with conservative word scanning, and the
 * executor's own list heads are registered as roots, so suspended frames
 * stay reachable through their tasks.
 */
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <time.h>
#include <setjmp.h>

#if defined(_WIN32)
#include <winsock2.h> /* before windows.h: WSAPoll drives socket waits */
#include <windows.h>
#else
#include <unistd.h>
#include <poll.h>
#endif

typedef struct veles_desc veles_desc;
void *veles_alloc(int64_t size);
void *veles_alloc_words(int64_t size);
void *veles_gc_alloc(veles_desc *desc, int64_t size);
void veles_gc_root(void *addr, veles_desc *desc);
void veles_panic(const char *msg, int64_t len);
void veles_panic_at(const char *msg, int64_t len, const char *loc, int64_t loc_len);
int64_t veles_desc_size(veles_desc *d);

/* the thread layer (veles_sync.c) and the collector's thread protocol
 * (veles_gc.c, D66) */
typedef struct veles_lock veles_lock;
typedef struct veles_cond veles_cond;
veles_lock *veles_lock_new(void);
void veles_lock_release(veles_lock *l);
veles_cond *veles_cond_new(void);
void veles_cond_wait(veles_cond *c, veles_lock *l, int64_t timeout_ms);
void veles_cond_signal(veles_cond *c);
void veles_cond_broadcast(veles_cond *c);
int64_t veles_thread_spawn(void (*fn)(void *), void *arg);
int64_t veles_cpu_count(void);
void veles_lock_enter(veles_lock *l);
void veles_lock_acquire(veles_lock *l);
void veles_enter_safe(void);
void veles_leave_safe(void);
void veles_thread_attach(void);

enum { T_RUNNABLE, T_BLOCKED, T_DONE, T_CANCELLED };

typedef struct veles_scope veles_scope;
typedef struct veles_race veles_race;

/* One entry in a channel's list of blocked receivers or senders. A task in
 * a plain send or recv waits through the node inside it; a race waits
 * through one node per channel arm, so one task can sit in several lists at
 * once without their links getting tangled. A plain waiter's data is the
 * value slot in its frame — what it sends, or where it receives — so the
 * other side copies straight across (channels, below). */
typedef struct veles_waiter {
    struct veles_task *task;
    struct veles_waiter *next;
    veles_race *race; /* the race the node belongs to; NULL for a plain wait */
    int64_t arm;      /* the race arm */
    void *data;       /* a plain waiter's value slot */
} veles_waiter;

typedef struct veles_task {
    void *hdl;              /* coroutine frame; NULL after completion */
    int64_t state;
    struct veles_task *waiter; /* task blocked in await on this one */
    void *result;           /* heap cell holding the return value */
    int64_t failed;         /* result is Err */
    veles_scope *scope;
    int64_t index;          /* the launch site within its scope (a loop launches from one site many times) */
    struct veles_task *next;      /* the shared run queue's link */
    struct veles_task *sibling;   /* scope children list, both ways: a finished */
    struct veles_task *sibling_prev; /* child unlinks itself in O(1) */
    int64_t listed;               /* on its scope's children list */
    int64_t wake_at;        /* timer, ms since start; 0 = none */
    int64_t yielded;        /* sleep(0) put the task at the back of the run queue once */
    struct veles_task *timer_next;
    veles_race *race;
    int64_t panicked;
    const char *panic_msg;
    int64_t panic_len;
    const char *panic_loc;  /* D64: where it panicked; empty inside the runtime */
    int64_t panic_loc_len;
    int64_t io_fd;          /* socket the task waits on (std/net); io_waiting set */
    int64_t io_write;       /* waiting to write rather than read */
    int64_t io_waiting;
    int64_t io_ready;
    struct veles_task *io_next;
    int64_t cancel_requested; /* unwinds at its next suspension point (D20/D43) */
    struct veles_task *awaiting; /* the task this one is blocked in await on */
    struct veles_cleanup *cleanups; /* active `with` closes and scope joins, innermost first */
    int64_t unwinding;           /* a panic is running the cleanups */
    int64_t sched;               /* S_IDLE, S_QUEUED, S_RUNNING or S_WOKEN (D66) */
    void (*entry)(struct veles_task *, void *); /* spawned, not yet started: its ramp */
    struct veles_chan *chan_wait; /* the channel whose waiter list holds chan_node */
    int64_t chan_wait_send;
    int64_t chan_done;            /* the other side completed its blocked send or recv */
    veles_waiter chan_node;       /* its entry there, for a plain send or recv */
    void *entry_args;
    struct veles_local *locals;   /* task-local bindings, innermost first (D72) */
} veles_task;

/* One active cleanup (D43/D49): the close of a `with` binding or the
 * cancellation of a scope's children. Code pushes on entry and pops on
 * every exit it emits itself; a panic runs whatever is still pushed,
 * innermost first, before the task is abandoned. */
typedef struct veles_cleanup {
    void (*fn)(void *env);
    void *env;
    struct veles_cleanup *next;
} veles_cleanup;

struct veles_scope {
    veles_task *owner;
    veles_task *children;  /* linked and unlinked under lock */
    int32_t lock;          /* a spinlock: the owner joins children without the runtime lock */
    int64_t live;          /* children not finished (atomic) */
    int64_t fail_fast;
    veles_task *failed;
};

typedef struct veles_chan {
    int32_t lock;      /* a spinlock: every field below changes under it */
    char *buf;
    int64_t cap, len, head, elem;
    veles_desc *desc;
    int64_t closed;
    int64_t remaining; /* closeAfter: sends left before the channel closes itself; -1 = never */
    veles_waiter *recv_waiters;
    veles_waiter *send_waiters;
} veles_chan;

struct veles_race {
    veles_task *task;
    int64_t winner;
    int64_t ready;
    int64_t closed; /* the winning channel arm was closed */
    /* registered sources */
    struct {
        veles_chan *ch;
        void *out;
        int64_t deadline;
        veles_task *awaited;
        veles_waiter node; /* its entry in ch's receivers */
    } arms[16];
    int64_t narms;
};

static veles_task *timers;
static veles_task *io_waiters;
#include "veles_tls.h"
#define current (veles_tls_get()->task)
static int64_t start_ms;
static veles_task *gq_head, *gq_tail; /* the shared queue (run queues, below) */
static bool roots_registered;

/* ---- the runtime lock (D66) -----------------------------------------------
 * Tasks run on a pool of worker threads. Timers, socket waiters,
 * channels, races and a task's wait state change only with the runtime
 * lock held. The hot path of a task's life does not take it: the run
 * queues (a spinlock each), a task's scheduling word (compare-and-swap),
 * a scope's children (a spinlock) and a successful finish (atomics, then
 * the lock only to wake someone) — see "run queues" below. The lock is
 * recursive, since one entry point may call another, and taking it is a
 * safepoint: a thread that must wait for it lets a collection run. Veles
 * code runs without it. */
static veles_lock *rt_lock;
static veles_cond *work_cv;     /* a task became runnable, or the root finished */
static veles_cond *spare_cv;    /* a run queue was handed off (blocking calls), or the root finished */
#define rt_depth (veles_tls_get()->rt_depth)
static veles_task *root_task;   /* the task veles_run drives; NULL between runs */
static int64_t active_workers;  /* workers inside a task (atomic) */
static int64_t idle_workers;    /* waiting on work_cv; changed under the runtime lock */
static int64_t waking;          /* a worker was signalled and has not come back yet */
static void rt_enter(void);
static void rt_exit(void);

/* A task was queued: wake one idle worker, unless one is already on its
 * way. Waking a worker per queued task would have them all fight over
 * the queues; instead the woken worker, once it has a task, wakes the next
 * if more are waiting — the number of awake workers follows the work (Go's
 * spinning threads). idle_workers only changes under the runtime lock, and
 * a sleeper holds it from counting itself idle, through a last look at
 * the queues, into the wait: a signal sent under the lock to a positive
 * count reaches a waiting worker. */
static void wake_worker(void) {
    if (__atomic_load_n(&idle_workers, __ATOMIC_SEQ_CST) == 0 || __atomic_load_n(&waking, __ATOMIC_RELAXED)) return;
    rt_enter();
    if (idle_workers > 0 && !waking) {
        waking = 1;
        veles_cond_signal(work_cv);
    }
    rt_exit();
}
static int64_t poller_busy;     /* a worker is blocked in poll() */
static int64_t workers_started;

static void rt_enter(void) {
    veles_lock_enter(rt_lock);
    rt_depth++;
}

static void rt_exit(void) {
    rt_depth--;
    veles_lock_release(rt_lock);
}

/* after a panic's longjmp: give back whatever the unwound frames held */
static void rt_unwind_to(int64_t depth) {
    while (rt_depth > depth) rt_exit();
}

static void register_roots(void) {
    if (roots_registered) return;
    roots_registered = true;
    veles_gc_root(&gq_head, NULL);
    veles_gc_root(&gq_tail, NULL);
    veles_gc_root(&timers, NULL);
    veles_gc_root(&io_waiters, NULL);
    veles_gc_root(&root_task, NULL);
}

/* the runtime lock exists before the first task does */
void veles_task_init(void) {
    if (rt_lock) return;
    rt_lock = veles_lock_new();
    work_cv = veles_cond_new();
    spare_cv = veles_cond_new();
    register_roots();
}

static int64_t now_ms(void) {
#if defined(_WIN32)
    return (int64_t)GetTickCount64();
#else
    struct timespec ts;
    clock_gettime(CLOCK_MONOTONIC, &ts);
    return ts.tv_sec * 1000 + ts.tv_nsec / 1000000;
#endif
}

/* ---- run queues (D66 stage 2) ------------------------------------------------
 * Each worker has a queue of its own: a ring it pushes to and takes from,
 * and that idle workers steal half of. A task woken outside any worker, a
 * ring that overflows and a yield go to one shared queue. None of this
 * takes the runtime lock: each queue has a spinlock held for a handful of
 * instructions, and a task's scheduling state is one word changed by
 * compare-and-swap —
 *   S_IDLE     parked or new: no queue holds it, no worker runs it
 *   S_QUEUED   in a queue or a worker's runnext slot
 *   S_RUNNING  a worker is inside its frame
 *   S_WOKEN    woken while running: queued again by that worker once the
 *              frame has parked, since no other worker may resume a frame
 *              that has not suspended yet
 * Everything else about a task still changes under the runtime lock. */

enum { S_IDLE, S_QUEUED, S_RUNNING, S_WOKEN };

#define RING 256
#define MAX_WORKERS 256
#define RUNNEXT_TURNS 32 /* runnext turns in a row before the queues get one */

/* a run queue's owner (blocking calls, below) */
enum { B_RUNNING, B_BLOCKED, B_HANDED_OFF };

typedef struct veles_worker {
    int32_t lock;
    int64_t bstate;          /* B_*: whether its thread is inside a blocking call */
    int64_t bseq;            /* blocking calls entered, so the monitor tells a long one from many */
    int64_t seen;            /* the monitor's: bseq when it last saw the thread blocked */
    uint32_t head, tail;     /* head: the next to take; tail: the next free slot */
    uint32_t seed;           /* where stealing starts looking */
    int64_t ticks;           /* tasks taken, for the shared queue's turn */
    veles_task *runnext;     /* woken by the task running here: runs next here */
    int64_t runnext_streak;
    veles_task *ring[RING];
} veles_worker;

static veles_worker *workers[MAX_WORKERS]; /* each one a GC root */
static int64_t nworkers;
static int32_t gq_lock;
static int64_t gq_len;

static inline void cpu_pause(void) {
#if defined(__x86_64__) || defined(__i386__)
    __builtin_ia32_pause();
#elif defined(__aarch64__)
    __asm__ __volatile__("yield");
#endif
}

static void spin_lock(int32_t *l) {
    while (__atomic_exchange_n(l, 1, __ATOMIC_ACQUIRE)) {
        while (__atomic_load_n(l, __ATOMIC_RELAXED)) cpu_pause();
    }
}

static void spin_unlock(int32_t *l) {
    __atomic_store_n(l, 0, __ATOMIC_RELEASE);
}

static void push_global_chain(veles_task *first, veles_task *last, int64_t n) {
    last->next = NULL;
    spin_lock(&gq_lock);
    if (gq_tail) gq_tail->next = first; else gq_head = first;
    gq_tail = last;
    __atomic_add_fetch(&gq_len, n, __ATOMIC_SEQ_CST);
    spin_unlock(&gq_lock);
}

static void push_global(veles_task *t) {
    push_global_chain(t, t, 1);
}

static veles_task *pop_global(void) {
    if (!__atomic_load_n(&gq_len, __ATOMIC_SEQ_CST)) return NULL;
    spin_lock(&gq_lock);
    veles_task *t = gq_head;
    if (t) {
        gq_head = t->next;
        if (!gq_head) gq_tail = NULL;
        t->next = NULL;
        __atomic_sub_fetch(&gq_len, 1, __ATOMIC_SEQ_CST);
    }
    spin_unlock(&gq_lock);
    return t;
}

static int ring_empty(veles_worker *w) {
    return __atomic_load_n(&w->head, __ATOMIC_ACQUIRE) == __atomic_load_n(&w->tail, __ATOMIC_ACQUIRE);
}

/* to the back of w's own ring (only w's thread pushes to it); a full ring
 * moves half of itself, and t, to the shared queue */
static void push_local(veles_worker *w, veles_task *t) {
    spin_lock(&w->lock);
    if (w->tail - w->head < RING) {
        w->ring[w->tail % RING] = t;
        __atomic_store_n(&w->tail, w->tail + 1, __ATOMIC_RELEASE);
        spin_unlock(&w->lock);
        return;
    }
    veles_task *first = NULL, *last = NULL;
    for (uint32_t i = 0; i < RING / 2; i++) {
        veles_task *x = w->ring[w->head % RING];
        w->ring[w->head % RING] = NULL;
        w->head++;
        x->next = NULL;
        if (last) last->next = x; else first = x;
        last = x;
    }
    spin_unlock(&w->lock);
    last->next = t;
    push_global_chain(first, t, RING / 2 + 1);
}

static veles_task *pop_local(veles_worker *w) {
    if (ring_empty(w)) return NULL;
    veles_task *t = NULL;
    spin_lock(&w->lock);
    if (w->head != w->tail) {
        t = w->ring[w->head % RING];
        w->ring[w->head % RING] = NULL;
        __atomic_store_n(&w->head, w->head + 1, __ATOMIC_RELEASE);
    }
    spin_unlock(&w->lock);
    return t;
}

/* takes half of another worker's ring: one task to run, the rest into
 * w's ring, which is empty when w steals */
static veles_task *steal(veles_worker *w) {
    int64_t n = __atomic_load_n(&nworkers, __ATOMIC_ACQUIRE);
    if (n < 2) return NULL;
    w->seed = w->seed * 1103515245u + 12345u;
    for (int64_t i = 0; i < n; i++) {
        veles_worker *v = workers[(w->seed + (uint32_t)i) % (uint32_t)n];
        if (!v || v == w || ring_empty(v)) continue;
        veles_task *got[RING / 2];
        uint32_t k = 0;
        spin_lock(&v->lock);
        uint32_t take = (v->tail - v->head + 1) / 2;
        for (; k < take; k++) {
            got[k] = v->ring[v->head % RING];
            v->ring[v->head % RING] = NULL;
            v->head++;
        }
        spin_unlock(&v->lock);
        if (k == 0) continue;
        if (k > 1) {
            spin_lock(&w->lock);
            for (uint32_t j = 1; j < k; j++) {
                w->ring[w->tail % RING] = got[j];
                w->tail++;
            }
            spin_unlock(&w->lock);
        }
        return got[0];
    }
    return NULL;
}

static void wake_worker(void);
static void note_timer(int64_t at);
static void finish_unstarted(veles_task *t);

/* puts a task that has just become S_QUEUED where a worker will find it.
 * Woken by the task running on this worker — a send to a waiting
 * receiver, a child finishing for its owner, a spawn — it runs here next,
 * while what the two share is still in this core's cache (Go's runnext);
 * the task it displaces goes to the ring. A yield goes to the back of the
 * shared queue, behind everything already waiting. */
static void place(veles_task *t) {
    veles_tls *tls = veles_tls_get();
    veles_worker *w = tls->worker;
    /* inside a callback from C the queue may be changing hands (blocking
     * calls, below): the shared queue */
    if (!w || t->yielded || tls->callback_depth > 0) {
        push_global(t);
        wake_worker();
        return;
    }
    if (tls->in_resume && tls->task && tls->task != t) {
        veles_task *old = w->runnext;
        w->runnext = t;
        if (!old) return;
        t = old;
    }
    push_local(w, t);
    wake_worker();
}

/* makes a task runnable: queues it, or, while a worker is inside it, has
 * that worker queue it again once the frame parks. Called with the runtime
 * lock held (the task's state is read). */
static void enqueue(veles_task *t) {
    if (__atomic_load_n(&t->state, __ATOMIC_SEQ_CST) >= T_DONE) return;
    for (;;) {
        int64_t s = __atomic_load_n(&t->sched, __ATOMIC_SEQ_CST);
        if (s == S_QUEUED || s == S_WOKEN) return;
        if (s == S_RUNNING) {
            if (__atomic_compare_exchange_n(&t->sched, &s, S_WOKEN, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) return;
            continue;
        }
        if (__atomic_compare_exchange_n(&t->sched, &s, S_QUEUED, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) break;
    }
    /* only a blocked task becomes runnable: a wake that comes after the
     * task finished (channels wake once their lock is released) leaves
     * the state alone, and run_task drops the finished task */
    int64_t blocked = T_BLOCKED;
    __atomic_compare_exchange_n(&t->state, &blocked, T_RUNNABLE, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST);
    place(t);
}

/* hands this worker's runnext to the shared queue: the thread is about to
 * block, and the task must not wait for it */
static void flush_runnext(void) {
    veles_worker *w = veles_tls_get()->worker;
    if (!w || !w->runnext) return;
    veles_task *t = w->runnext;
    w->runnext = NULL;
    push_global(t);
    wake_worker();
}

/* the next task for worker w, or NULL: now and then the shared queue
 * first, so a busy ring cannot starve it; then runnext, unless it has had
 * RUNNEXT_TURNS turns in a row; the ring; the shared queue; another
 * worker's ring */
static veles_task *find_task(veles_worker *w) {
    veles_task *t;
    if (++w->ticks % 61 == 0 && (t = pop_global())) return t;
    if (w->runnext && w->runnext_streak < RUNNEXT_TURNS) {
        t = w->runnext;
        w->runnext = NULL;
        w->runnext_streak++;
        return t;
    }
    w->runnext_streak = 0;
    if ((t = pop_local(w))) return t;
    if ((t = pop_global())) return t;
    if ((t = steal(w))) return t;
    if ((t = w->runnext)) w->runnext = NULL;
    return t;
}

/* any task another worker could take: the shared queue or a ring. A
 * runnext slot does not count — only its worker runs it, and a worker
 * with one is active — or idle workers would spin instead of sleeping. */
static int anything_queued(void) {
    if (__atomic_load_n(&gq_len, __ATOMIC_SEQ_CST)) return 1;
    int64_t n = __atomic_load_n(&nworkers, __ATOMIC_ACQUIRE);
    for (int64_t i = 0; i < n; i++) {
        veles_worker *w = workers[i];
        if (w && !ring_empty(w)) return 1;
    }
    return 0;
}

/* a task in some worker's runnext slot. It is not work for an idle worker
 * (only its own worker runs it), but it is work: a worker that left its
 * loop for the timers, holding one, is waiting for the lock a sleeper
 * holds while it decides whether the program is deadlocked. */
static int any_runnext(void) {
    int64_t n = __atomic_load_n(&nworkers, __ATOMIC_ACQUIRE);
    for (int64_t i = 0; i < n; i++) {
        veles_worker *w = workers[i];
        if (w && __atomic_load_n(&w->runnext, __ATOMIC_ACQUIRE)) return 1;
    }
    return 0;
}

static void wake(veles_task *t) {
    if (!t) return;
    int64_t s = __atomic_load_n(&t->sched, __ATOMIC_SEQ_CST);
    int64_t st = __atomic_load_n(&t->state, __ATOMIC_SEQ_CST);
    if (st == T_BLOCKED || ((s == S_RUNNING || s == S_WOKEN) && st == T_RUNNABLE)) enqueue(t);
}

/* the frame of t has parked (or finished): a wake that came while it ran
 * queues it now. Needs no lock: only this worker leaves S_RUNNING/S_WOKEN. */
static void after_run(veles_task *t) {
    int64_t s = S_RUNNING;
    if (__atomic_compare_exchange_n(&t->sched, &s, S_IDLE, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) return;
    /* S_WOKEN */
    if (t->state == T_DONE || t->state == T_CANCELLED) {
        __atomic_store_n(&t->sched, S_IDLE, __ATOMIC_SEQ_CST);
        return;
    }
    __atomic_store_n(&t->sched, S_QUEUED, __ATOMIC_SEQ_CST);
    place(t);
}

/* ---- blocking calls (D66) ----------------------------------------------------
 * Around a call that may block its thread for long — foreign code, a read
 * from the terminal, waiting for a child process, a contended Mutex — the
 * thread enters a safe region, so a collection does not wait for the call
 * to return, and marks its run queue blocked, so tasks do not either: a
 * monitor thread looks every millisecond, and a queue whose thread has
 * been inside one call for a whole look while work waits — tasks queued,
 * a timer due, sockets to poll — and no worker is idle is handed to a
 * spare thread (Go's sysmon hands off a P the same way). The thread that
 * blocked finds on its way back that it lost the queue; the task it runs
 * is on its stack, so it finishes that task's step, down to the next
 * suspension, and then waits as a spare for a queue of its own. At most
 * VELES_THREADS threads run tasks at a time; a program that blocks in C
 * uses more threads, never fewer cores. */
static int timers_due(void);
static void worker_main(void *arg);
static int64_t monitor_asleep;
static void wake_monitor(void);

static veles_worker *free_workers[MAX_WORKERS]; /* handed off, waiting for a thread */
static int64_t nfree;              /* under the runtime lock */
static int64_t spares_waiting;     /* threads waiting on spare_cv (runtime lock) */

#if defined(VELES_CAPTURE_ASM)
void veles_enter_safe_c(void);
#define enter_safe_here veles_enter_safe_c
#else
#define enter_safe_here veles_enter_safe
#endif

/* The outermost blocking call of a thread with a run queue marks it: the
 * task that would run next goes to the shared queue first, since the queue
 * may change hands before the call returns. Inside a callback from C the
 * outer call's mark stands (and place() queues nothing here). */
static void block_worker(void) {
    veles_tls *tls = veles_tls_get();
    if (tls->blocking++ > 0 || tls->callback_depth > 0) return;
    veles_worker *w = tls->worker;
    if (!w) return;
    flush_runnext();
    __atomic_store_n(&w->bseq, w->bseq + 1, __ATOMIC_RELAXED);
    __atomic_store_n(&w->bstate, B_BLOCKED, __ATOMIC_SEQ_CST);
    if (__atomic_load_n(&monitor_asleep, __ATOMIC_SEQ_CST)) wake_monitor();
}

/* back from the call: the queue is still this thread's, unless the
 * monitor handed it off meanwhile — one compare-and-swap decides */
static void unblock_worker(void) {
    veles_tls *tls = veles_tls_get();
    if (--tls->blocking > 0 || tls->callback_depth > 0) return;
    veles_worker *w = tls->worker;
    if (!w) return;
    int64_t s = B_BLOCKED;
    if (!__atomic_compare_exchange_n(&w->bstate, &s, B_RUNNING, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) {
        tls->worker = NULL;
    }
}

#if defined(VELES_CAPTURE_ASM)
/* recorded where the blocking call is made, as veles_enter_safe is
 * (veles_gc.c): this frame returns before the call blocks */
void veles_blocking_enter_c(void) {
    block_worker();
    enter_safe_here();
}

__attribute__((naked)) void veles_blocking_enter(void) {
    __asm__ volatile(VELES_CAPTURE_ASM VELES_ASM_TAIL(veles_blocking_enter_c));
}
#else
void veles_blocking_enter(void) {
    block_worker();
    veles_enter_safe();
}
#endif

void veles_blocking_leave(void) {
    veles_leave_safe();
    unblock_worker();
}

/* a run queue for a thread that has none (runtime lock held): one the
 * monitor handed off */
static veles_worker *take_free_worker(void) {
    if (nfree == 0) return NULL;
    veles_worker *w = free_workers[--nfree];
    free_workers[nfree] = NULL;
    w->runnext_streak = 0;
    __atomic_store_n(&w->bstate, B_RUNNING, __ATOMIC_SEQ_CST);
    veles_tls_get()->worker = w;
    return w;
}

/* whether a queue stuck behind a blocking call keeps work waiting */
static int work_waiting(void) {
    if (__atomic_load_n(&idle_workers, __ATOMIC_SEQ_CST) > 0) return 0; /* an idle worker takes it */
    return anything_queued() || timers_due() || __atomic_load_n(&io_waiters, __ATOMIC_RELAXED) != NULL;
}

static void hand_off(veles_worker *w) {
    int spawn = 0;
    rt_enter();
    free_workers[nfree++] = w;
    if (spares_waiting > 0) veles_cond_signal(spare_cv); else spawn = 1;
    rt_exit();
    /* with no thread to be had, the queue waits in free_workers for its
     * owner, which takes it back once its task parks */
    if (spawn) veles_thread_spawn(worker_main, NULL);
}

static int any_blocked(void) {
    int64_t n = __atomic_load_n(&nworkers, __ATOMIC_ACQUIRE);
    for (int64_t i = 0; i < n; i++) {
        veles_worker *w = workers[i];
        if (w && __atomic_load_n(&w->bstate, __ATOMIC_SEQ_CST) == B_BLOCKED) return 1;
    }
    return 0;
}

/* The monitor looks every millisecond while threads block, and after a
 * tenth of a second with none it sleeps until block_worker wakes it: a
 * program that never blocks in a call costs it no wakeups. Asleep is set
 * and the queues looked at with monitor_lock held, and a blocker that
 * sees asleep takes that lock to signal, so the signal cannot fall
 * between the look and the wait. */
static veles_lock *monitor_lock;
static veles_cond *monitor_cv;

static void wake_monitor(void) {
    veles_lock_acquire(monitor_lock);
    veles_cond_signal(monitor_cv);
    veles_lock_release(monitor_lock);
}

static void monitor_main(void *arg) {
    (void)arg;
    int64_t quiet = 0; /* looks in a row that found no thread blocked */
    for (;;) {
        veles_lock_acquire(monitor_lock);
        if (quiet > 100) {
            __atomic_store_n(&monitor_asleep, 1, __ATOMIC_SEQ_CST);
            if (!any_blocked()) veles_cond_wait(monitor_cv, monitor_lock, -1);
            __atomic_store_n(&monitor_asleep, 0, __ATOMIC_SEQ_CST);
            quiet = 0;
        } else {
            veles_cond_wait(monitor_cv, monitor_lock, 1);
        }
        veles_lock_release(monitor_lock);
        int any = 0;
        int64_t n = __atomic_load_n(&nworkers, __ATOMIC_ACQUIRE);
        for (int64_t i = 0; i < n; i++) {
            veles_worker *w = workers[i];
            if (!w || __atomic_load_n(&w->bstate, __ATOMIC_SEQ_CST) != B_BLOCKED) continue;
            any = 1;
            int64_t seq = __atomic_load_n(&w->bseq, __ATOMIC_RELAXED);
            if (seq != w->seen) {
                w->seen = seq; /* a call that began since the last look */
                continue;
            }
            if (!work_waiting()) continue;
            int64_t s = B_BLOCKED;
            if (__atomic_compare_exchange_n(&w->bstate, &s, B_HANDED_OFF, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) hand_off(w);
        }
        quiet = any ? 0 : quiet + 1;
    }
}

/* ---- tasks ----------------------------------------------------------------- */

veles_task *veles_task_new(void) {
    veles_task_init();
    veles_task *t = veles_alloc_words(sizeof *t);
    t->state = T_RUNNABLE;
    /* a task started here — by async, or for a suspending call — sees the
     * task-local values bound where it started (D72) */
    veles_tls *tls = veles_tls_get();
    t->locals = tls->task ? tls->task->locals : tls->locals;
    return t;
}

/* ---- task-local values (D72) ------------------------------------------------
 * A task holds its bindings as a list, innermost first, that nothing ever
 * changes in place: withValue puts a node in front and later puts the old
 * head back, and a new task starts from its creator's head. So a child
 * shares its parent's nodes without a lock, keeps the values bound when
 * it started, and never sees a rebinding its parent makes afterwards.
 * Code outside any task (a callback on a thread Veles did not start) binds
 * on its thread instead. */
typedef struct veles_local {
    int64_t key;               /* the TaskLocal's identity */
    void *cell;                /* the bound value, a GC cell */
    struct veles_local *next;
} veles_local;

static int64_t local_keys;

static veles_local **current_locals(void) {
    veles_tls *tls = veles_tls_get();
    return tls->task ? &tls->task->locals : &tls->locals;
}

int64_t veles_local_key(void) {
    return __atomic_add_fetch(&local_keys, 1, __ATOMIC_RELAXED);
}

/* the cell bound to key in the current task, or NULL */
void *veles_local_find(int64_t key) {
    for (veles_local *l = *current_locals(); l; l = l->next) {
        if (l->key == key) return l->cell;
    }
    return NULL;
}

/* binds key to cell in front of the current bindings; returns the head
 * to put back when the binding ends */
void *veles_local_bind(int64_t key, void *cell) {
    veles_local *n = veles_alloc_words(sizeof *n);
    n->key = key;
    n->cell = cell;
    veles_local **head = current_locals();
    n->next = *head;
    *head = n;
    return n->next;
}

void veles_local_restore(void *head) {
    *current_locals() = head;
}

veles_task *veles_task_current(void) {
    return current;
}

void veles_task_set_current(veles_task *t) {
    current = t;
}

/* veles_task_started records the coroutine handle once the ramp returned. */
static void veles_task_started_impl(veles_task *t, void *hdl) {
    if (t->state != T_DONE && t->state != T_CANCELLED) t->hdl = hdl;
}

static void scope_child_finished(veles_task *t);
static void remove_timer(veles_task *t);

/* called by the coroutine body before its final suspend */
static void veles_task_finish_impl(veles_task *t, const void *result, int64_t size, int64_t failed) {
    if (t->state == T_CANCELLED) return;
    t->result = veles_alloc_words(size > 0 ? size : 8);
    if (size > 0 && result) memcpy(t->result, result, (size_t)size);
    t->failed = failed;
    /* released: an await that sees T_DONE without the lock sees the result */
    __atomic_store_n(&t->state, T_DONE, __ATOMIC_RELEASE);
    t->hdl = NULL;
    wake(t->waiter);
    t->waiter = NULL;
    scope_child_finished(t);
}

/* the cancelled task has run its cleanups (D43) and is done */
static void veles_task_finish_cancelled_impl(veles_task *t) {
    t->hdl = NULL;
    if (t->state == T_CANCELLED) return;
    t->state = T_CANCELLED;
    wake(t->waiter);
    t->waiter = NULL;
    scope_child_finished(t);
}

/* await: true when the target is done, otherwise blocks the caller */
static int64_t veles_task_await_impl(veles_task *self, veles_task *target) {
    if (target->state == T_DONE) {
        self->awaiting = NULL;
        return 1;
    }
    if (target->state == T_CANCELLED) {
        self->awaiting = NULL;
        veles_panic("awaited task was cancelled", 26);
    }
    /* registered first, then the state read again: a task finishing
     * without the lock publishes T_DONE and then reads its waiter, so one
     * of the two always sees the other */
    __atomic_store_n(&target->waiter, self, __ATOMIC_SEQ_CST);
    if (__atomic_load_n(&target->state, __ATOMIC_SEQ_CST) == T_DONE) {
        target->waiter = NULL;
        self->awaiting = NULL;
        return 1;
    }
    self->awaiting = target;
    self->state = T_BLOCKED;
    return 0;
}

void *veles_task_result(veles_task *t) {
    return t->result;
}

int64_t veles_task_failed(veles_task *t) {
    return t->failed;
}

/* ---- scopes (D34/D36) ------------------------------------------------------ */

veles_scope *veles_scope_begin(veles_task *owner, int64_t fail_fast) {
    veles_scope *s = veles_alloc_words(sizeof *s);
    s->owner = owner;
    s->fail_fast = fail_fast;
    return s;
}

/* async: the task of a scope child. It joins the scope when it is spawned
 * (veles_task_spawn), after its arguments are evaluated — an argument that
 * throws leaves no child behind for the scope to wait for — and taking no
 * lock here leaves one runtime-lock round trip per launch. */
veles_task *veles_task_launch(veles_scope *s, int64_t site) {
    veles_task *t = veles_task_new();
    t->scope = s;
    t->index = site;
    return t;
}

static void spin_lock(int32_t *l);
static void spin_unlock(int32_t *l);

/* links a launched task into its scope's children */
static void join_scope(veles_task *t) {
    veles_scope *s = t->scope;
    if (!s || t->listed) return;
    __atomic_add_fetch(&s->live, 1, __ATOMIC_SEQ_CST);
    spin_lock(&s->lock);
    t->sibling = s->children;
    if (s->children) s->children->sibling_prev = t;
    s->children = t;
    t->listed = 1;
    spin_unlock(&s->lock);
}

static void remove_io_waiter(veles_task *t);
static void leave_channel_waits(veles_task *t);

/* Cancellation is a request: the task is woken and unwinds at the
 * suspension point it was parked on (running its `with` cleanups, D43),
 * then reports itself finished. A task with no frame — it never
 * suspended, or already returned — is finished on the spot. */
static void cancel_task(veles_task *t) {
    if (t->state == T_DONE || t->state == T_CANCELLED || t->cancel_requested) return;
    t->cancel_requested = 1;
    remove_timer(t);
    remove_io_waiter(t);
    leave_channel_waits(t);
    int64_t sched = __atomic_load_n(&t->sched, __ATOMIC_SEQ_CST);
    if (sched != S_IDLE) {
        /* a worker is inside it, or is about to be: it sees the request at
         * its next suspension point (a spawned task that has not started
         * is finished unstarted by the worker that takes it) */
        if (sched == S_RUNNING) enqueue(t);
        return;
    }
    veles_task *callee = t->awaiting;
    if (callee && !callee->scope && callee->state != T_DONE && callee->state != T_CANCELLED) {
        /* blocked in a suspending call (the callee runs as a task of its
         * own): cancel from the inside out, so the innermost frame's
         * cleanups run first; the callee's finish wakes this task, which
         * then sees the request at its own suspension point */
        cancel_task(callee);
        return;
    }
    if (t->hdl) {
        t->state = T_BLOCKED;
        wake(t);
        return;
    }
    finish_unstarted(t);
}

/* `task.cancel()`: ask a task to stop; it unwinds at its next suspension
 * point, and its scope still waits for it */
static void veles_task_cancel_impl(veles_task *t) {
    cancel_task(t);
}

/* cancel every child still running; a child with no frame finishes on the
 * spot and unlinks itself, so the next link is read first */
static void cancel_children(veles_scope *s, veles_task *except) {
    /* a child that finishes unlinks itself without the runtime lock, so
     * the list is copied under the scope's lock first (into the heap: the
     * copy must keep the tasks alive, and cancelling may reach a safepoint) */
    int64_t n = 0;
    spin_lock(&s->lock);
    for (veles_task *c = s->children; c; c = c->sibling) n++;
    spin_unlock(&s->lock);
    if (n == 0) return;
    veles_task **snap = veles_alloc_words(n * (int64_t)sizeof *snap);
    int64_t k = 0;
    spin_lock(&s->lock);
    for (veles_task *c = s->children; c && k < n; c = c->sibling) snap[k++] = c;
    spin_unlock(&s->lock);
    for (int64_t i = 0; i < k; i++) {
        if (snap[i] != except) cancel_task(snap[i]);
    }
}

/* a child leaves its scope: one fewer to wait for, and off the list;
 * returns how many are left. Takes no runtime lock. */
static int64_t scope_leave(veles_task *t) {
    veles_scope *s = t->scope;
    int64_t left = __atomic_sub_fetch(&s->live, 1, __ATOMIC_SEQ_CST);
    /* a finished child leaves the scope's list: a server's accept loop
     * launches a task per connection for as long as it runs, and the list
     * would otherwise keep every one of them (and its frame) alive */
    spin_lock(&s->lock);
    if (t->listed) {
        if (t->sibling_prev) t->sibling_prev->sibling = t->sibling; else s->children = t->sibling;
        if (t->sibling) t->sibling->sibling_prev = t->sibling_prev;
        t->sibling = t->sibling_prev = NULL;
        t->listed = 0;
    }
    spin_unlock(&s->lock);
    return left;
}

static void scope_child_finished(veles_task *t) {
    veles_scope *s = t->scope;
    if (!s) return;
    int64_t left = scope_leave(t);
    if (t->failed && s->fail_fast && !s->failed) {
        s->failed = t;
        cancel_children(s, t);
        /* the owner may be blocked in the scope body (a recv that will now
         * never complete): wake it so its next suspension point sees the
         * failure and abandons the body */
        wake(s->owner);
    }
    if (left <= 0) wake(s->owner);
}

/* a task abandoning a scope body forgets whatever it was waiting on. It
   is unlinked from every list, so a later send or close cannot hand a
   value to (or wake) a wait that is over */
static void veles_task_leave_waits_impl(veles_task *t) {
    leave_channel_waits(t);
    remove_timer(t);
    remove_io_waiter(t);
}

/* the body left the scope early (return, throw, cancellation): the
 * children still running are cancelled; the owner then waits for them
 * as usual, so nothing outlives the block (D34) */
static void veles_scope_cancel_impl(veles_scope *s) {
    cancel_children(s, NULL);
}

/* wait for every child: true when done, otherwise blocks the owner */
static int64_t veles_scope_wait_impl(veles_task *owner, veles_scope *s) {
    if (__atomic_load_n(&s->live, __ATOMIC_SEQ_CST) <= 0) return 1;
    owner->state = T_BLOCKED;
    return 0;
}

static veles_task *veles_scope_failed_impl(veles_scope *s) {
    return s->failed;
}

static int64_t veles_scope_failed_index_impl(veles_scope *s) {
    return s->failed ? s->failed->index : -1;
}

/* ---- channels (D16) --------------------------------------------------------
 * Each channel has a lock of its own — a spinlock held for a handful of
 * instructions — so tasks on different channels never contend (D66); the
 * runtime lock is not taken. A blocked sender or receiver waits with its
 * value slot in its node, and the side that arrives second copies across:
 * a sender into a waiting receiver's slot, a receiver out of a waiting
 * sender's, marking the waiter's operation done (chan_done) before waking
 * it — its retry then returns at once. So a channel without capacity is a
 * rendezvous: a send completes when a receiver has the value, and blocked
 * senders of a full channel keep their order (a receiver that frees a slot
 * fills it from the first of them). An operation marked done has happened:
 * a cancellation that arrives after it is seen at the task's next
 * suspension point.
 *
 * Tasks are woken only after the channel's lock is released (wakes, below):
 * waking may take the runtime lock to rouse an idle worker, and code
 * holding the runtime lock takes channel locks (a race registering its
 * arms, a cancellation unlinking a waiter) — never the other way round. */

/* the tasks a channel operation woke, woken once its lock is released */
typedef struct wakes {
    veles_task *few[8];
    veles_task **more;
    int64_t n, cap;
} wakes;

static void wakes_add(wakes *k, veles_task *t) {
    if (!t) return;
    if (k->n < 8) {
        k->few[k->n++] = t;
        return;
    }
    int64_t i = k->n - 8;
    if (i >= k->cap) {
        k->cap = k->cap ? k->cap * 2 : 16;
        veles_task **m = realloc(k->more, (size_t)k->cap * sizeof *m);
        if (!m) veles_panic("out of memory", 13);
        k->more = m;
    }
    k->more[i] = t;
    k->n++;
}

static void wakes_run(wakes *k) {
    for (int64_t i = 0; i < k->n; i++) wake(i < 8 ? k->few[i] : k->more[i - 8]);
    free(k->more);
}

veles_chan *veles_chan_new(veles_desc *desc, int64_t cap) {
    veles_chan *c = veles_alloc_words(sizeof *c);
    c->desc = desc;
    c->elem = veles_desc_size(desc);
    if (cap < 0) cap = 0;
    c->cap = cap;
    c->remaining = -1;
    c->buf = veles_gc_alloc(desc, c->elem * cap + 1);
    return c;
}

static veles_waiter *pop_waiter(veles_waiter **list) {
    veles_waiter *w = *list;
    if (w) {
        *list = w->next;
        w->next = NULL;
        if (!w->race) w->task->chan_wait = NULL;
    }
    return w;
}

/* the task of the first blocked sender, taken off the list (senders only
 * ever wait plainly) */
static veles_task *pop_sender(veles_chan *c) {
    veles_waiter *w = pop_waiter(&c->send_waiters);
    return w ? w->task : NULL;
}

/* the first blocked sender that is not being cancelled */
static veles_waiter *pop_live_sender(veles_chan *c) {
    veles_waiter *w;
    while ((w = pop_waiter(&c->send_waiters))) {
        veles_task *t = w->task;
        if (__atomic_load_n(&t->state, __ATOMIC_ACQUIRE) != T_CANCELLED &&
            !__atomic_load_n(&t->cancel_requested, __ATOMIC_ACQUIRE)) return w;
    }
    return NULL;
}

/* a blocked send or recv of t was completed by the other side */
static void chan_complete(veles_task *t, wakes *k) {
    __atomic_store_n(&t->chan_done, 1, __ATOMIC_SEQ_CST);
    wakes_add(k, t);
}

static void push_waiter(veles_waiter **list, veles_waiter *w) {
    w->next = NULL;
    while (*list) list = &(*list)->next;
    *list = w;
}

static void remove_waiter(veles_waiter **list, veles_waiter *w) {
    while (*list) {
        if (*list == w) {
            *list = w->next;
            w->next = NULL;
            continue;
        }
        list = &(*list)->next;
    }
}

/* A race's winner is claimed by compare-and-swap — a value or a close on
 * one of its channels (under that channel's lock), its timer, or the race
 * itself finding an arm ready — so exactly one arm completes it. The
 * claimer fills the arm's slot and only then publishes ready. */
static bool race_claim(veles_race *r, int64_t arm) {
    int64_t none = -1;
    return __atomic_compare_exchange_n(&r->winner, &none, arm, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST);
}

static void race_ready(veles_race *r, wakes *k) {
    __atomic_store_n(&r->ready, 1, __ATOMIC_RELEASE);
    wakes_add(k, r->task);
}

/* parks self in c's receivers (or senders) through its own node (c's lock
 * held); a task woken for another reason and blocking again is moved to
 * the back, not added twice */
static void chan_block(veles_task *self, veles_chan *c, int64_t send, void *data) {
    veles_waiter **list = send ? &c->send_waiters : &c->recv_waiters;
    veles_waiter *w = &self->chan_node;
    if (self->chan_wait == c) remove_waiter(self->chan_wait_send ? &c->send_waiters : &c->recv_waiters, w);
    w->task = self;
    w->race = NULL;
    w->arm = 0;
    w->data = data;
    push_waiter(list, w);
    self->chan_wait = c;
    self->chan_wait_send = send;
    self->state = T_BLOCKED;
}

/* unlinks t's plain wait, on whichever channel it is (no channel lock
 * held; the channel's own is taken) */
static void leave_chan_wait(veles_task *t) {
    veles_chan *c = t->chan_wait;
    if (!c) return;
    spin_lock(&c->lock);
    if (t->chan_wait == c) {
        remove_waiter(t->chan_wait_send ? &c->send_waiters : &c->recv_waiters, &t->chan_node);
        t->chan_wait = NULL;
    }
    spin_unlock(&c->lock);
}

static void chan_push(veles_chan *c, const void *item) {
    int64_t slot = (c->head + c->len) % c->cap;
    memcpy(c->buf + slot * c->elem, item, (size_t)c->elem);
    c->len++;
}

static void chan_pop(veles_chan *c, void *out) {
    memcpy(out, c->buf + c->head * c->elem, (size_t)c->elem);
    c->head = (c->head + 1) % c->cap;
    c->len--;
}

/* deliver to a waiting receiver if any; returns true if delivered */
static bool chan_hand_off(veles_chan *c, const void *item, wakes *k) {
    while (c->recv_waiters) {
        veles_waiter *w = pop_waiter(&c->recv_waiters);
        veles_task *r = w->task;
        if (__atomic_load_n(&r->state, __ATOMIC_ACQUIRE) == T_CANCELLED ||
            __atomic_load_n(&r->cancel_requested, __ATOMIC_ACQUIRE)) continue;
        if (w->race) {
            if (!race_claim(w->race, w->arm)) continue; /* already won elsewhere */
            memcpy(w->race->arms[w->arm].out, item, (size_t)c->elem);
            race_ready(w->race, k);
            return true;
        }
        /* plain receiver: straight into its slot */
        memcpy(w->data, item, (size_t)c->elem);
        chan_complete(r, k);
        return true;
    }
    return false;
}

static void chan_close_locked(veles_chan *c, wakes *k);

/* a value entered the channel's keeping: closeAfter counts it */
static void chan_sent(veles_chan *c, wakes *k) {
    if (c->remaining > 0 && --c->remaining == 0) chan_close_locked(c, k);
}

/* whether a value can be taken from c without blocking */
static bool chan_takeable(veles_chan *c) {
    if (c->len > 0) return true;
    for (veles_waiter *w = c->send_waiters; w; w = w->next) {
        veles_task *t = w->task;
        if (__atomic_load_n(&t->state, __ATOMIC_ACQUIRE) != T_CANCELLED &&
            !__atomic_load_n(&t->cancel_requested, __ATOMIC_ACQUIRE)) return true;
    }
    return false;
}

/* a value from c into out, without blocking: from the buffer, whose freed
 * slot takes the first blocked sender's value, or from a blocked sender
 * directly (a rendezvous) */
static bool chan_take(veles_chan *c, void *out, wakes *k) {
    veles_waiter *s;
    if (c->len > 0) {
        chan_pop(c, out);
        if ((s = pop_live_sender(c))) {
            chan_push(c, s->data);
            chan_complete(s->task, k);
            chan_sent(c, k);
        }
        return true;
    }
    if ((s = pop_live_sender(c))) {
        memcpy(out, s->data, (size_t)c->elem);
        chan_complete(s->task, k);
        chan_sent(c, k);
        return true;
    }
    return false;
}

/* a send or recv this task blocked in was completed meanwhile */
static bool chan_took_place(veles_task *self) {
    if (!__atomic_load_n(&self->chan_done, __ATOMIC_SEQ_CST)) return false;
    __atomic_store_n(&self->chan_done, 0, __ATOMIC_SEQ_CST);
    return true;
}

/* send: 1 when a waiting receiver took the value or it was buffered; 0
 * blocks the sender, and a receiver then takes the value from its slot;
 * -1 for a closed channel (the caller panics, with the lock released) */
static int64_t chan_send_locked(veles_task *self, veles_chan *c, const void *item, wakes *k) {
    if (chan_took_place(self)) return 1;
    if (c->closed) return -1;
    if (!chan_hand_off(c, item, k)) {
        if (c->len >= c->cap) {
            chan_block(self, c, 1, (void *)item);
            return 0;
        }
        chan_push(c, item);
    }
    chan_sent(c, k);
    return 1;
}

/* recv: 1 value received, 2 closed and empty, 0 blocked */
static int64_t chan_recv_locked(veles_task *self, veles_chan *c, void *out, wakes *k) {
    if (chan_took_place(self)) return 1;
    if (chan_take(c, out, k)) return 1;
    if (c->closed) return 2;
    chan_block(self, c, 0, out);
    return 0;
}

static void chan_close_locked(veles_chan *c, wakes *k) {
    c->closed = 1;
    veles_waiter *w;
    while ((w = pop_waiter(&c->recv_waiters))) {
        veles_race *r = w->race;
        if (r) {
            /* a closed channel completes the race arm with "closed" */
            if (race_claim(r, w->arm)) {
                r->closed = 1;
                race_ready(r, k);
            }
            continue;
        }
        wakes_add(k, w->task);
    }
    /* a sender blocked on a full channel is woken too: its retry finds the
     * channel closed and panics, as any send on a closed channel does */
    veles_task *t;
    while ((t = pop_sender(c))) wakes_add(k, t);
}

/* trySend: 1 when a waiting receiver took the value or it was buffered, 0
   when neither could (a full buffer; a rendezvous channel with no receiver
   waiting), -1 for a closed channel; never blocks. */
static int64_t chan_try_send_locked(veles_chan *c, const void *item, wakes *k) {
    if (c->closed) return -1;
    if (!chan_hand_off(c, item, k)) {
        if (c->len >= c->cap) return 0;
        chan_push(c, item);
    }
    chan_sent(c, k);
    return 1;
}

/* closeAfter(n): the channel closes itself once n more values have been sent,
   so several producers can end it without coordinating. */
static void chan_close_after_locked(veles_chan *c, int64_t n, wakes *k) {
    if (n <= 0) {
        chan_close_locked(c, k);
        return;
    }
    c->remaining = n;
}

static bool race_claim(veles_race *r, int64_t arm);

/* ---- timers ------------------------------------------------------------------ */

static void remove_timer(veles_task *t) {
    veles_task **pp = &timers;
    while (*pp) {
        if (*pp == t) {
            *pp = t->timer_next;
            t->timer_next = NULL;
            continue;
        }
        pp = &(*pp)->timer_next;
    }
    t->wake_at = 0;
}

static void add_timer(veles_task *t, int64_t ms) {
    t->wake_at = now_ms() + ms;
    t->timer_next = timers;
    timers = t;
    note_timer(t->wake_at);
}

/* sleep: true once the deadline passed; first call arms it and blocks.
 * A task can be woken before its deadline for another reason - a scope
 * child finishing wakes the scope's owner - and it then comes back here:
 * it must block again, still on the timer list. Returning 0 while leaving
 * the task T_RUNNABLE lost it: fire_timers' wake() only wakes a blocked
 * task, so the sleeper was never resumed and the executor reported a
 * deadlock (a producer finishing while main slept on a timer). */
static int64_t veles_task_sleep_impl(veles_task *self, int64_t ms) {
    if (self->wake_at != 0) {
        if (now_ms() >= self->wake_at) {
            self->wake_at = 0;
            return 1;
        }
        self->state = T_BLOCKED;
        return 0;
    }
    if (ms <= 0) {
        /* a yield: to the back of the run queue once, so every other
         * runnable task gets a turn before this one continues - the
         * polling loop around a tryRecv depends on it */
        if (self->yielded) {
            self->yielded = 0;
            return 1;
        }
        self->yielded = 1;
        enqueue(self);
        return 0;
    }
    remove_timer(self);
    add_timer(self, ms);
    self->state = T_BLOCKED;
    return 0;
}

static void fire_timers(void) {
    int64_t now = now_ms();
    veles_task **pp = &timers;
    while (*pp) {
        veles_task *t = *pp;
        if (t->wake_at != 0 && now >= t->wake_at) {
            *pp = t->timer_next;
            t->timer_next = NULL;
            if (t->race) {
                /* the timer arm claims the race, unless another arm has */
                veles_race *race = t->race;
                t->wake_at = 0;
                for (int64_t i = 0; i < race->narms; i++) {
                    if (race->arms[i].deadline && race->arms[i].deadline <= now) {
                        if (race_claim(race, i)) wake(t);
                        break;
                    }
                }
            } else {
                wake(t);
            }
        } else {
            pp = &t->timer_next;
        }
    }
}

/* ---- socket waits (std/net) ------------------------------------------------ */

/* Sockets are non-blocking; when a call would block, the task parks here
 * until the executor's poll reports the descriptor ready (readable,
 * writable, or in error - the retried call then reports what happened). */

static void remove_io_waiter(veles_task *t) {
    veles_task **pp = &io_waiters;
    while (*pp) {
        if (*pp == t) {
            *pp = t->io_next;
            t->io_next = NULL;
            continue;
        }
        pp = &(*pp)->io_next;
    }
    t->io_waiting = 0;
    t->io_ready = 0;
}

/* wait for fd: true once the poll saw it ready; the first call parks the
 * task, a wake for any other reason parks it again */
static int64_t veles_task_wait_io_impl(veles_task *self, int64_t fd, int64_t write) {
    if (self->io_waiting) {
        if (self->io_ready) {
            self->io_waiting = 0;
            self->io_ready = 0;
            return 1;
        }
        self->state = T_BLOCKED;
        return 0;
    }
    self->io_fd = fd;
    self->io_write = write;
    self->io_waiting = 1;
    self->io_ready = 0;
    self->io_next = io_waiters;
    io_waiters = self;
    self->state = T_BLOCKED;
    return 0;
}

static int64_t io_waiter_count(void) {
    int64_t n = 0;
    for (veles_task *t = io_waiters; t; t = t->io_next) n++;
    return n;
}

/* poll every parked descriptor, waiting at most timeout_ms (-1: forever),
 * and wake the tasks whose descriptors are ready. Called with the runtime
 * lock held once; the wait itself happens without it, in a safe region, so
 * other workers keep running tasks and the collector may run. */
static void poll_io(int64_t timeout_ms) {
    int64_t n = io_waiter_count();
    if (n == 0 || poller_busy) return;
#if defined(_WIN32)
    WSAPOLLFD *fds = malloc(sizeof(WSAPOLLFD) * (size_t)n);
#else
    struct pollfd *fds = malloc(sizeof(struct pollfd) * (size_t)n);
#endif
    veles_task **tasks = malloc(sizeof(veles_task *) * (size_t)n);
    int64_t i = 0;
    for (veles_task *t = io_waiters; t; t = t->io_next, i++) {
        tasks[i] = t;
#if defined(_WIN32)
        fds[i].fd = (SOCKET)t->io_fd;
#else
        fds[i].fd = (int)t->io_fd;
#endif
        fds[i].events = t->io_write ? POLLOUT : POLLIN;
        fds[i].revents = 0;
    }
    if (timeout_ms > 0x7fffffff) timeout_ms = 0x7fffffff;
    int r;
    if (timeout_ms == 0) {
#if defined(_WIN32)
        r = WSAPoll(fds, (ULONG)n, 0);
#else
        r = poll(fds, (nfds_t)n, 0);
#endif
    } else {
        poller_busy = 1;
        int64_t depth = rt_depth;
        rt_unwind_to(0);
        veles_enter_safe();
#if defined(_WIN32)
        r = WSAPoll(fds, (ULONG)n, (INT)timeout_ms);
#else
        r = poll(fds, (nfds_t)n, (int)timeout_ms);
#endif
        veles_leave_safe();
        while (rt_depth < depth) rt_enter();
        poller_busy = 0;
    }
    if (r > 0) {
        for (i = 0; i < n; i++) {
            if (fds[i].revents == 0) continue;
            veles_task *t = tasks[i];
            /* while the lock was released the task may have been cancelled
             * or have moved on; only a task still waiting on this socket
             * is woken */
            if (!t->io_waiting || t->io_ready || t->io_fd != (int64_t)fds[i].fd) continue;
            remove_io_waiter(t);
            t->io_waiting = 1; /* stays "waiting" so the retry sees ready */
            t->io_ready = 1;
            wake(t);
        }
    }
    free(fds);
    free(tasks);
}

/* ---- race (D38) ------------------------------------------------------------
 * A race waits on several things at once: it registers a node on each
 * channel arm (under that channel's lock), itself as the waiter of each
 * awaited task, and one timer for the earliest deadline. Whatever becomes
 * ready first claims the winner by compare-and-swap (race_claim) and fills
 * the arm, and only the claimer does. The race's own entry point runs
 * with the runtime lock held (timers, awaited tasks) and takes each
 * channel's lock in turn — the lock order everywhere. */

veles_race *veles_race_new(veles_task *t) {
    veles_race *r = veles_alloc_words(sizeof *r);
    r->task = t;
    r->winner = -1;
    return r;
}

void veles_race_recv(veles_race *r, veles_chan *c, void *out) {
    int64_t i = r->narms++;
    r->arms[i].ch = c;
    r->arms[i].out = out;
}

void veles_race_sleep(veles_race *r, int64_t ms) {
    int64_t i = r->narms++;
    r->arms[i].deadline = now_ms() + ms;
    if (ms <= 0) r->arms[i].deadline = 1;
}

void veles_race_await(veles_race *r, veles_task *t) {
    int64_t i = r->narms++;
    r->arms[i].awaited = t;
}

/* takes self's nodes off every list the race registered them on (runtime
 * lock held; each channel's own is taken) */
static void race_detach(veles_task *self, veles_race *r) {
    for (int64_t i = 0; i < r->narms; i++) {
        veles_chan *c = r->arms[i].ch;
        if (c) {
            spin_lock(&c->lock);
            remove_waiter(&c->recv_waiters, &r->arms[i].node);
            spin_unlock(&c->lock);
        }
        if (r->arms[i].awaited && r->arms[i].awaited->waiter == self) r->arms[i].awaited->waiter = NULL;
    }
    remove_timer(self);
    if (self->race == r) self->race = NULL;
}

/* unlinks t from whatever channel lists it is on: the arms of a race it
 * waits in, or the one channel it is blocked sending to or receiving from
 * (runtime lock held) */
static void leave_channel_waits(veles_task *t) {
    if (t->race) race_detach(t, t->race);
    leave_chan_wait(t);
}

/* channel arm i, under its lock: a value or a close, claimed for the race
 * — or nothing ready, and the node registered when register_node */
static bool race_try_chan(veles_race *r, int64_t i, int register_node, wakes *k) {
    veles_chan *c = r->arms[i].ch;
    bool won = false;
    spin_lock(&c->lock);
    if (chan_takeable(c) || c->closed) {
        if (race_claim(r, i)) {
            if (!chan_take(c, r->arms[i].out, k)) r->closed = 1;
            won = true;
        }
    } else if (register_node) {
        veles_waiter *w = &r->arms[i].node;
        w->task = r->task;
        w->race = r;
        w->arm = i;
        w->data = NULL;
        push_waiter(&c->recv_waiters, w);
    }
    spin_unlock(&c->lock);
    return won;
}

/* returns the winning arm, or -1 to block */
static int64_t veles_race_wait_impl(veles_task *self, veles_race *r) {
    wakes k = {0};
    int64_t won = -1;
    /* woken for another reason while registered: start over from nothing,
     * so no node is ever on a list twice. Whatever claimed the race first
     * did all of it under a lock this takes, so a claim seen here is
     * complete. */
    if (self->race == r) race_detach(self, r);
    if (__atomic_load_n(&r->winner, __ATOMIC_SEQ_CST) >= 0) {
        race_detach(self, r);
        return __atomic_load_n(&r->winner, __ATOMIC_SEQ_CST);
    }
    self->race = r;
    /* ready now, or registered everywhere */
    int64_t now = now_ms();
    int64_t earliest = 0;
    for (int64_t i = 0; i < r->narms && won < 0; i++) {
        if (r->arms[i].ch) {
            if (race_try_chan(r, i, 1, &k)) won = i;
        } else if (r->arms[i].deadline) {
            if (now >= r->arms[i].deadline) {
                if (race_claim(r, i)) won = i;
            } else if (!earliest || r->arms[i].deadline < earliest) {
                earliest = r->arms[i].deadline;
            }
        } else if (r->arms[i].awaited) {
            veles_task *a = r->arms[i].awaited;
            a->waiter = self;
            /* an awaited task that finishes without the lock publishes
             * T_DONE and then reads its waiter (the await explains it) */
            if (__atomic_load_n(&a->state, __ATOMIC_SEQ_CST) == T_DONE && race_claim(r, i)) won = i;
        }
        if (won < 0 && __atomic_load_n(&r->winner, __ATOMIC_SEQ_CST) >= 0) break; /* claimed on an arm already registered */
    }
    if (won >= 0) {
        race_detach(self, r);
        wakes_run(&k);
        return won;
    }
    if (earliest) {
        remove_timer(self);
        self->wake_at = earliest;
        self->timer_next = timers;
        timers = self;
        note_timer(earliest);
    }
    /* blocked — also when an arm registered above was claimed meanwhile:
     * its claimer wakes this task once it has filled the arm */
    self->state = T_BLOCKED;
    wakes_run(&k);
    return -1;
}

/* ---- executor ---------------------------------------------------------------- */

#define panic_return (veles_tls_get()->panic_return)
#define in_resume (veles_tls_get()->in_resume)
static void run_entry(veles_task *t, void (*entry)(veles_task *, void *), void *args);

/* ---- cleanups (D43/D49) ---------------------------------------------------- */

/* c is the entry itself, in the frame of the function that entered the
 * cleanup; it stays there until the matching pop */
void veles_cleanup_push(veles_cleanup *c, void (*fn)(void *), void *env) {
    if (!current) return;
    c->fn = fn;
    c->env = env;
    c->next = current->cleanups;
    current->cleanups = c;
}

void veles_cleanup_pop(veles_cleanup *c) {
    if (!current || current->cleanups != c) return;
    current->cleanups = c->next;
}

/* veles_task_panic is called by veles_panic while a task runs: the task's
 * active cleanups run (a `with` closes its resource, a scope cancels its
 * children), then the task fails with the message and control returns to
 * the executor (D20: a panic unwinds to the enclosing task scope). A
 * panic inside a cleanup continues the unwinding with the first message. */
int64_t veles_task_panic(const char *msg, int64_t len, const char *loc, int64_t loc_len) {
    if (!in_resume || !current) return 0;
    veles_task *t = current;
    /* the cleanups are Veles code: they run without the runtime lock the
     * panicking runtime call may have held */
    rt_unwind_to(0);
    if (!t->unwinding) {
        t->unwinding = 1;
        char *copy = veles_alloc(len + 1);
        memcpy(copy, msg, (size_t)len);
        t->panic_msg = copy;
        t->panic_len = len;
        char *where = veles_alloc(loc_len + 1);
        if (loc_len > 0) memcpy(where, loc, (size_t)loc_len);
        t->panic_loc = where;
        t->panic_loc_len = loc_len;
    }
    while (t->cleanups) {
        veles_cleanup *c = t->cleanups;
        t->cleanups = c->next;
        c->fn(c->env);
    }
    void *result = veles_alloc_words(8);
    rt_enter();
    t->panicked = 1;
    t->failed = 1;
    t->result = result;
    __atomic_store_n(&t->state, T_DONE, __ATOMIC_RELEASE);
    t->hdl = NULL;
    wake(t->waiter);
    t->waiter = NULL;
    scope_child_finished(t);
    rt_exit();
    longjmp(panic_return, 1);
    return 1;
}

int64_t veles_task_panicked(veles_task *t) {
    return t->panicked;
}

const char *veles_task_panic_msg(veles_task *t, int64_t *len) {
    *len = t->panic_len;
    return t->panic_msg;
}

const char *veles_task_panic_loc(veles_task *t, int64_t *len) {
    *len = t->panic_loc_len;
    return t->panic_loc ? t->panic_loc : "";
}

static void resume(veles_task *t) {
    void *hdl = t->hdl;
    if (!hdl) return;
    current = t;
    void (*fn)(void *) = *(void (**)(void *))hdl;
    in_resume = 1;
    if (setjmp(panic_return) == 0) {
        fn(hdl);
    } else {
        rt_unwind_to(0);
    }
    in_resume = 0;
    current = NULL;
}

static int64_t nearest_timer(void) {
    int64_t nearest = 0;
    for (veles_task *x = timers; x; x = x->timer_next) {
        if (x->wake_at && (!nearest || x->wake_at < nearest)) nearest = x->wake_at;
    }
    return nearest;
}

static int root_finished(void) {
    return !root_task || root_task->state == T_DONE || root_task->state == T_CANCELLED;
}

static void scope_child_finished(veles_task *t);

/* A spawned task cancelled before it ran: it never starts, and finishes
 * as cancelled (its scope stops waiting for it, an await on it panics). */
static void finish_unstarted(veles_task *t) {
    t->entry = NULL;
    t->entry_args = NULL;
    t->state = T_CANCELLED;
    wake(t->waiter);
    t->waiter = NULL;
    scope_child_finished(t);
}

static int64_t worker_target; /* VELES_THREADS: how many run queues there are */

/* the calling thread's run queue (runtime lock held): the one it has, one
 * handed off by the monitor, or a new one while there are fewer than
 * VELES_THREADS — else NULL, and the thread waits as a spare */
static veles_worker *join_workers(void) {
    veles_tls *tls = veles_tls_get();
    if (tls->worker) return tls->worker;
    veles_worker *free = take_free_worker();
    if (free) return free;
    if (nworkers >= worker_target) return NULL;
    veles_worker *w = veles_alloc_words(sizeof *w);
    w->seed = (uint32_t)nworkers * 2654435761u + 1;
    workers[nworkers] = w;
    veles_gc_root(&workers[nworkers], NULL);
    __atomic_store_n(&nworkers, nworkers + 1, __ATOMIC_RELEASE);
    tls->worker = w;
    return w;
}

/* runs t, which this worker took from a queue (S_QUEUED), until its frame
 * parks or finishes */
static void run_task(veles_task *t) {
    __atomic_store_n(&t->sched, S_RUNNING, __ATOMIC_SEQ_CST);
    if (t->state == T_DONE || t->state == T_CANCELLED) {
        after_run(t);
        return;
    }
    void (*entry)(veles_task *, void *) = t->entry;
    void *args = t->entry_args;
    if (entry && __atomic_load_n(&t->cancel_requested, __ATOMIC_SEQ_CST)) {
        rt_enter();
        finish_unstarted(t);
        rt_exit();
        after_run(t);
        return;
    }
    t->entry = NULL;
    t->entry_args = NULL;
    t->state = T_RUNNABLE;
    if (entry) run_entry(t, entry, args); else resume(t);
    after_run(t);
}

/* the earliest timer deadline, readable without the lock (0: none): a
 * worker running tasks goes back for the timers once it has passed */
static int64_t timer_due;

static void note_timer(int64_t at) {
    int64_t due = __atomic_load_n(&timer_due, __ATOMIC_RELAXED);
    if (!due || at < due) __atomic_store_n(&timer_due, at, __ATOMIC_RELAXED);
}

static int timers_due(void) {
    int64_t due = __atomic_load_n(&timer_due, __ATOMIC_RELAXED);
    return due && now_ms() >= due;
}

/* One thread's loop, entered and left with the runtime lock held once. The
 * lock is for the housekeeping — timers, sockets, going to sleep; tasks
 * are taken and run without it. It runs until the root has finished; stay
 * (a pool thread) keeps it waiting for the next root instead of returning.
 * A thread without a run queue — there are VELES_THREADS, and its own was
 * handed off during a blocking call — waits for one. */
static void work(int stay) {
    veles_tls *tls = veles_tls_get();
    for (;;) {
        veles_worker *w = join_workers();
        if (root_finished()) {
            if (!stay) return;
            veles_enter_safe();
            veles_cond_wait(w ? work_cv : spare_cv, rt_lock, -1);
            veles_leave_safe();
            continue;
        }
        if (!w) {
            spares_waiting++;
            veles_enter_safe();
            veles_cond_wait(spare_cv, rt_lock, 50);
            veles_leave_safe();
            spares_waiting--;
            continue;
        }
        fire_timers();
        __atomic_store_n(&timer_due, nearest_timer(), __ATOMIC_RELAXED);
        if (io_waiters && anything_queued()) poll_io(0); /* runnable tasks must not starve the sockets */
        int64_t ran = 0;
        /* active from the first look at the queues to the last task run:
         * a worker holding a task it took is not idle, and no sleeper may
         * call the program deadlocked meanwhile */
        __atomic_add_fetch(&active_workers, 1, __ATOMIC_SEQ_CST);
        rt_exit();
        for (;;) {
            veles_task *t = find_task(w);
            if (!t) break;
            if (anything_queued()) wake_worker(); /* more waiting: a hand for them */
            run_task(t);
            ran++;
            if (tls->worker != w) break; /* handed off while t blocked */
            if (root_task && (root_task->state == T_DONE || root_task->state == T_CANCELLED)) break;
            if (timers_due() || (io_waiters && ran % 64 == 0)) break;
        }
        __atomic_sub_fetch(&active_workers, 1, __ATOMIC_SEQ_CST);
        rt_enter();
        if (ran) {
            if (root_finished()) {
                veles_cond_broadcast(work_cv);
                veles_cond_broadcast(spare_cv);
            }
            continue;
        }
        /* nothing runnable: wait for a task, the nearest timer or a socket */
        int64_t nearest = nearest_timer();
        int64_t wait = nearest ? nearest - now_ms() : -1;
        if (nearest && wait < 0) wait = 0;
        if (io_waiters && !poller_busy) {
            /* another worker may register a socket or wake a task while
             * this one sits in poll(): keep the wait short while any run */
            if (__atomic_load_n(&active_workers, __ATOMIC_SEQ_CST) > 0 && (wait < 0 || wait > 10)) wait = 10;
            poll_io(wait < 0 ? 1000 : (wait == 0 ? 1 : wait));
            continue;
        }
        if (wait == 0) continue;
        idle_workers++;
        if (anything_queued()) {
            idle_workers--;
            continue;
        }
        if (root_finished()) {
            idle_workers--;
            continue; /* finished while this worker looked: nothing is stuck */
        }
        if (!nearest && !io_waiters && !poller_busy && __atomic_load_n(&active_workers, __ATOMIC_SEQ_CST) == 0 && !any_runnext()) {
            veles_panic("deadlock: every task is blocked", 31);
        }
        if (wait < 0 || wait > 50) wait = 50; /* recheck: a timer set elsewhere, a deadlock */
        veles_enter_safe();
        veles_cond_wait(work_cv, rt_lock, wait);
        veles_leave_safe();
        idle_workers--;
        waking = 0;
    }
}

/* a pool thread, or a spare started for a queue handed off */
static void worker_main(void *arg) {
    (void)arg;
    veles_thread_attach();
    rt_enter();
    work(1);
    rt_exit();
}

/* how many threads run tasks: VELES_THREADS, or one per core */
static int64_t worker_count(void) {
    const char *env = getenv("VELES_THREADS");
    int64_t n = env ? strtoll(env, NULL, 10) : veles_cpu_count();
    if (n < 1) n = 1;
    if (n > 256) n = 256;
    return n;
}

/* veles_run drives the executor until the root task completes. The calling
 * thread is one of the workers; the others are started on the first run
 * and wait between runs (veles test runs one root per test). */
void veles_run(veles_task *root) {
    veles_task_init();
    rt_enter();
    start_ms = now_ms();
    root_task = root;
    if (!workers_started) {
        workers_started = 1;
        worker_target = worker_count();
        for (int64_t i = 1; i < worker_target; i++) veles_thread_spawn(worker_main, NULL);
        monitor_lock = veles_lock_new();
        monitor_cv = veles_cond_new();
        veles_thread_spawn(monitor_main, NULL);
    }
    veles_cond_broadcast(work_cv);
    work(0);
    root_task = NULL;
    rt_exit();
}

int64_t veles_race_closed(veles_race *r) {
    return r->closed;
}

/* re-raise a child's panic in the current context (scope re-raises, D52) */
void veles_task_repanic(veles_task *t) {
    veles_panic_at(t->panic_msg, t->panic_len, t->panic_loc, t->panic_loc_len);
}

/* veles_task_start runs a coroutine ramp through an entry thunk inside the
 * executor's panic guard, so a panic before the first suspension is
 * captured like any other. Entries nest (a ramp may launch children), so
 * the guard state is saved and restored. */
/* runs a task's ramp on this thread until it first parks (t->sched is
 * set, so no other worker resumes it meanwhile). Ramps nest — a ramp may
 * start children — so the guard state is saved and restored. */
static void run_entry(veles_task *t, void (*entry)(veles_task *, void *), void *args) {
    veles_task *saved_current = current;
    int64_t saved_in = in_resume;
    int64_t saved_depth = rt_depth;
    jmp_buf saved;
    memcpy(&saved, &panic_return, sizeof saved);
    current = t;
    in_resume = 1;
    if (setjmp(panic_return) == 0) {
        entry(t, args);
    } else {
        rt_unwind_to(saved_depth);
    }
    memcpy(&panic_return, &saved, sizeof saved);
    current = saved_current;
    in_resume = saved_in;
}

void veles_task_start(veles_task *t, void (*entry)(veles_task *, void *), void *args) {
    veles_task_init();
    __atomic_store_n(&t->sched, S_RUNNING, __ATOMIC_SEQ_CST);
    run_entry(t, entry, args);
    after_run(t);
}

/* `async f(x)` (D66): the task goes on a run queue with its ramp and
 * arguments — this worker's runnext, so it starts here once the owner
 * parks unless an idle worker steals it first — and whichever worker takes
 * it runs the ramp: the children of a scope start on other cores while
 * the body goes on. A task cancelled before any worker took it never
 * starts. */
void veles_task_spawn(veles_task *t, void (*entry)(veles_task *, void *), void *args) {
    /* no runtime lock: the task is new, and only its owner knows it until
     * it joins the scope. It is S_QUEUED before it is visible there, so a
     * cancel that finds it only sets the request, and the worker that
     * takes it finishes it unstarted (run_task). */
    t->entry = entry;
    t->entry_args = args;
    t->state = T_RUNNABLE;
    __atomic_store_n(&t->sched, S_QUEUED, __ATOMIC_SEQ_CST);
    join_scope(t);
    place(t);
}

/* ---- the entry points the compiled code calls ------------------------------
 * Most take the runtime lock around the implementation above (D66); the
 * ones on every task's path — started, finish, cancelled, await of a
 * finished task — say why they need not. */

/* no lock: the worker running t is the only one to touch its frame
 * handle while t is S_RUNNING (cancel_task reads it only when idle) */
void veles_task_started(veles_task *t, void *hdl) {
    veles_task_started_impl(t, hdl);
}

/* A task returning. The common case — a result, not an error — takes the
 * runtime lock only to wake someone: a task awaiting this one, or the
 * scope's owner when this was its last child. An error may fail the
 * scope and cancel siblings, which is done under the lock. */
void veles_task_finish(veles_task *t, const void *result, int64_t size, int64_t failed) {
    if (failed || __atomic_load_n(&t->state, __ATOMIC_SEQ_CST) == T_CANCELLED) {
        rt_enter();
        veles_task_finish_impl(t, result, size, failed);
        rt_exit();
        return;
    }
    void *cell = veles_alloc_words(size > 0 ? size : 8);
    if (size > 0 && result) memcpy(cell, result, (size_t)size);
    t->result = cell;
    t->failed = 0;
    __atomic_store_n(&t->state, T_DONE, __ATOMIC_SEQ_CST);
    t->hdl = NULL;
    int wake_waiter = __atomic_load_n(&t->waiter, __ATOMIC_SEQ_CST) != NULL;
    int64_t left = t->scope ? scope_leave(t) : 1;
    if (wake_waiter || left <= 0) {
        rt_enter();
        if (t->waiter) {
            wake(t->waiter);
            t->waiter = NULL;
        }
        if (left <= 0) wake(t->scope->owner);
        rt_exit();
    }
}

/* checked at every suspension point, so it takes no lock: both flags only
 * ever go from 0 to 1, and a request that lands just after the check is
 * seen at the next one, as it would be with the lock */
int64_t veles_task_cancelled(veles_task *t) {
    /* a send or recv the other side completed has happened: the retry
     * returns it, and the request is seen at the next suspension point */
    if (__atomic_load_n(&t->chan_done, __ATOMIC_ACQUIRE)) return 0;
    return __atomic_load_n(&t->cancel_requested, __ATOMIC_ACQUIRE) ||
           __atomic_load_n(&t->state, __ATOMIC_ACQUIRE) == T_CANCELLED;
}

void veles_task_finish_cancelled(veles_task *t) {
    rt_enter();
    veles_task_finish_cancelled_impl(t);
    rt_exit();
}

int64_t veles_task_await(veles_task *self, veles_task *target) {
    /* a task already done needs no lock: its result was published with
     * the state (awaiting is only read while self is parked) */
    if (__atomic_load_n(&target->state, __ATOMIC_ACQUIRE) == T_DONE) {
        self->awaiting = NULL;
        return 1;
    }
    rt_enter();
    int64_t result_ = veles_task_await_impl(self, target);
    rt_exit();
    return result_;
}


void veles_task_cancel(veles_task *t) {
    rt_enter();
    veles_task_cancel_impl(t);
    rt_exit();
}

void veles_task_leave_waits(veles_task *t) {
    rt_enter();
    veles_task_leave_waits_impl(t);
    rt_exit();
}

void veles_scope_cancel(veles_scope *s) {
    rt_enter();
    veles_scope_cancel_impl(s);
    rt_exit();
}

int64_t veles_scope_wait(veles_task *owner, veles_scope *s) {
    rt_enter();
    int64_t result_ = veles_scope_wait_impl(owner, s);
    rt_exit();
    return result_;
}

veles_task *veles_scope_failed(veles_scope *s) {
    rt_enter();
    veles_task * result_ = veles_scope_failed_impl(s);
    rt_exit();
    return result_;
}

int64_t veles_scope_failed_index(veles_scope *s) {
    rt_enter();
    int64_t result_ = veles_scope_failed_index_impl(s);
    rt_exit();
    return result_;
}

/* The channel operations take the channel's own lock, never the runtime
 * lock, and wake the tasks they completed once it is released. A task
 * still listed on another channel — woken for another reason, then sent
 * here — is unlinked there first, so no two channel locks are ever held. */
static void closed_send(void) {
    veles_panic("send on a closed channel", 24);
}

int64_t veles_chan_send(veles_task *self, veles_chan *c, const void *item) {
    wakes k = {0};
    if (self->chan_wait && self->chan_wait != c) leave_chan_wait(self);
    spin_lock(&c->lock);
    int64_t result_ = chan_send_locked(self, c, item, &k);
    spin_unlock(&c->lock);
    wakes_run(&k);
    if (result_ < 0) closed_send();
    return result_;
}

int64_t veles_chan_recv(veles_task *self, veles_chan *c, void *out) {
    wakes k = {0};
    if (self->chan_wait && self->chan_wait != c) leave_chan_wait(self);
    spin_lock(&c->lock);
    int64_t result_ = chan_recv_locked(self, c, out, &k);
    spin_unlock(&c->lock);
    wakes_run(&k);
    return result_;
}

void veles_chan_close(veles_chan *c) {
    wakes k = {0};
    spin_lock(&c->lock);
    chan_close_locked(c, &k);
    spin_unlock(&c->lock);
    wakes_run(&k);
}

int64_t veles_chan_len(veles_chan *c) {
    spin_lock(&c->lock);
    int64_t result_ = c->len;
    spin_unlock(&c->lock);
    return result_;
}

int64_t veles_chan_try_send(veles_chan *c, const void *item) {
    wakes k = {0};
    spin_lock(&c->lock);
    int64_t result_ = chan_try_send_locked(c, item, &k);
    spin_unlock(&c->lock);
    wakes_run(&k);
    if (result_ < 0) closed_send();
    return result_;
}

int64_t veles_chan_try_recv(veles_chan *c, void *out) {
    wakes k = {0};
    spin_lock(&c->lock);
    int64_t result_ = chan_take(c, out, &k) ? 1 : 0;
    spin_unlock(&c->lock);
    wakes_run(&k);
    return result_;
}

void veles_chan_close_after(veles_chan *c, int64_t n) {
    wakes k = {0};
    spin_lock(&c->lock);
    chan_close_after_locked(c, n, &k);
    spin_unlock(&c->lock);
    wakes_run(&k);
}

int64_t veles_task_sleep(veles_task *self, int64_t ms) {
    rt_enter();
    int64_t result_ = veles_task_sleep_impl(self, ms);
    rt_exit();
    return result_;
}

int64_t veles_task_wait_io(veles_task *self, int64_t fd, int64_t write) {
    rt_enter();
    int64_t result_ = veles_task_wait_io_impl(self, fd, write);
    rt_exit();
    return result_;
}

int64_t veles_race_wait(veles_task *self, veles_race *r) {
    rt_enter();
    int64_t result_ = veles_race_wait_impl(self, r);
    rt_exit();
    return result_;
}
