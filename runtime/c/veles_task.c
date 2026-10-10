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
/* glibc declares its extensions (pthread_getattr_np, ...) only when asked. */
#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stddef.h>
#include <stdbool.h>
#include <time.h>
#include <setjmp.h>

#if defined(_WIN32)
#include <windows.h>
#else
#include <unistd.h>
#include <sched.h>
#endif

#include "veles_tls.h"
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
void veles_cond_wait_ns(veles_cond *c, veles_lock *l, int64_t timeout_ns);
void veles_timer_period(void);
void veles_cond_signal(veles_cond *c);
void veles_cond_broadcast(veles_cond *c);
typedef struct veles_park veles_park;
veles_park *veles_park_new(void);
void veles_park_wait(veles_park *p, int64_t timeout_ns);
void veles_park_wake(veles_park *p);
int64_t veles_thread_spawn(void (*fn)(void *), void *arg);
int64_t veles_thread_spawn_small(void (*fn)(void *), void *arg);
int64_t veles_cpu_count(void);
void veles_lock_enter_kept(veles_lock *l);
void veles_lock_acquire(veles_lock *l);
void veles_enter_safe(void);
void veles_leave_safe(void);
void veles_thread_attach(void);

/* T_ENDING: unwound by a panic, or by a cancellation seen in code that
 * does not suspend (D145), whose scopes still have children unwinding: it
 * has no frame and runs no more, and it ends — T_DONE or T_CANCELLED —
 * when the last of them has (end_unwound, below) */
enum { T_RUNNABLE, T_BLOCKED, T_DONE, T_CANCELLED, T_ENDING };

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
    struct veles_waiter *next, *prev;
    struct waiter_list *in; /* the list it is on; NULL when on none */
    veles_race *race; /* the race the node belongs to; NULL for a plain wait */
    int64_t arm;      /* the race arm */
    void *data;       /* a plain waiter's value slot */
} veles_waiter;

/* a channel's blocked receivers or senders, oldest first: appending and
 * unlinking are O(1), so n tasks park on one channel in O(n) */
typedef struct waiter_list {
    veles_waiter *head, *tail;
} waiter_list;

/* what a panic report shows under its message besides the location (D81, D78) */
typedef struct panic_trace {
    const char *in;         /* the function the panic is in (constant); NULL in a release build */
    const char *chain;      /* "site in caller" lines, newline-joined, innermost first */
    int64_t chain_len;
    const char *sites;      /* test helpers' call sites in effect, newline-joined, innermost first */
    int64_t sites_len;
} panic_trace;

/* what a panicked task carries, allocated when it panics: most tasks never
 * do, so a task holds one pointer instead of all of it (review B4) */
typedef struct task_panic {
    const char *msg;
    int64_t len;
    const char *loc;        /* D64: where it panicked; empty inside the runtime */
    int64_t loc_len;
    panic_trace trace;      /* the call chain and the test helpers' call sites at the panic */
    int64_t drain;          /* while unwinding without a frame: scopes whose children are still unwinding, + 1 */
    int64_t cancelled;      /* that unwinding is a cancellation (D145), not a panic: msg is NULL */
} task_panic;

/* the call chain of a debug build (D81), allocated by its first call: a
 * release build makes none */
typedef struct task_debug {
    veles_shadow shadow;              /* the calls in progress */
    struct veles_task *chain_parent;  /* the task whose suspending call this one runs */
} task_debug;

/* Every parked task is one of these, so it is kept small (review B4): the
 * rare parts hang off pointers, and the flags are narrow. A flag read or
 * written with __atomic_* is int32_t; the others are bytes, each written
 * only by the thread that runs the task or under the runtime lock. */
typedef struct veles_task {
    void *hdl;              /* coroutine frame; NULL after completion */
    struct veles_task *waiter; /* task blocked in await on this one */
    union {
        void *entry_args;   /* spawned, not yet started: its arguments (cleared as it starts) */
        void *result;       /* finished: heap cell holding the return value */
    };
    struct frame_arena *arena; /* where its calls' frames live (review F3) */
    veles_scope *scope;
    struct veles_task *next;      /* the shared run queue's link */
    struct veles_task *sibling;   /* scope children list, both ways: a finished */
    struct veles_task *sibling_prev; /* child unlinks itself in O(1) */
    int64_t wake_at;        /* timer, monotonic ns; 0 = none */
    veles_race *race;
    task_panic *panic;      /* set when it panics */
    task_debug *debug;      /* a debug build's call chain */
    int64_t io_fd;          /* socket the task waits on (std/net); io_waiting set */
    struct veles_task *io_next;   /* the other tasks parked on io_fd the same way */
    struct veles_task *io_prev;
    struct veles_cleanup *cleanups; /* active `with` closes and scope joins, innermost first */
    void (*entry)(struct veles_task *, void *); /* spawned, not yet started: its ramp */
    struct veles_chan *chan_wait; /* the channel whose waiter list holds chan_node */
    veles_waiter chan_node;       /* its entry there, for a plain send or recv */
    struct veles_local *locals;   /* task-local bindings, innermost first (D72) */
    int32_t state;
    int32_t sched;                /* S_IDLE, S_QUEUED, S_RUNNING or S_WOKEN (D66) */
    int16_t index;                /* the launch site within its scope (a loop launches from one site many times) */
    uint8_t shield;               /* inside a close() that may suspend: no cancellation is delivered, and a parked task is not woken for one (D47, D147) */
    uint8_t spare;
    int32_t timer_slot;           /* its place in the timer heap + 1; 0 = none */
    int32_t depth;                /* the frame hdl is: 0 the task's own, n a call n deep (review F3) */
    int32_t abandon_depth;        /* frames deeper than this unwind: a failed child abandons the scope body at it */
    uint8_t failed;               /* result is Err */
    uint8_t cancel_requested;     /* unwinds at its next suspension point (D20/D43) */
    uint8_t chan_done;            /* the other side completed its blocked send or recv */
    uint8_t value_off;            /* where the Ok payload sits in the result (unwrap): after the tag word */
    uint8_t unwrap;               /* a fail-fast child (D141): await gives the Ok payload */
    uint8_t listed;               /* on its scope's children list */
    uint8_t yielded;              /* sleep(0) put the task at the back of the run queue once */
    uint8_t panicked;
    uint8_t test_task;            /* a test the runner queued (D80): its end starts the next */
    uint8_t io_write;             /* waiting to write rather than read */
    uint8_t io_waiting;
    uint8_t io_ready;
    uint8_t unwinding;            /* a panic is running the cleanups */
    uint8_t chan_wait_send;
    uint8_t popped;               /* the frame resumed last returned to its caller, which resumes next */
    uint8_t attn;                 /* counted in veles_attention while it runs (D145) */
} veles_task;

/* One active cleanup (D43/D49): the close of a `with` binding or the
 * cancellation of a scope's children. Code pushes on entry and pops on
 * every exit it emits itself; a panic runs whatever is still pushed,
 * innermost first, before the task is abandoned. The entry and what env
 * points to live in the function's frame or on the stack; `move` copies
 * env's contents to the heap, for a cleanup that must outlive them — one
 * a close() that suspends takes on to run after it (D147). */
typedef struct veles_cleanup {
    void (*fn)(void *env);
    void *env;
    struct veles_cleanup *next;
    void *(*move)(void *env);
} veles_cleanup;

struct veles_scope {
    veles_task *owner;
    struct veles_exec *exec; /* where its children run: `scope(on: e)`, or the owner's executor (D143) */
    veles_task *children;  /* linked and unlinked under lock */
    int32_t lock;          /* a spinlock: the owner joins children without the runtime lock */
    int32_t depth;         /* the depth of the owner's frame that runs the body */
    int32_t draining;      /* the owner unwound without its frame and waits, T_ENDING, for these children */
    int32_t closing;       /* its one child runs a close() that suspends for the owner, which unwinds without its frame (D147) */
    int64_t live;          /* children not finished (atomic) */
    int64_t fail_fast;
    veles_task *failed;
    int64_t over;          /* the owner has joined every child: the scope has ended */
};

typedef struct veles_chan {
    int32_t lock;      /* a spinlock: every field below changes under it */
    char *buf;
    int64_t cap, len, head, elem;
    veles_desc *desc;
    int64_t closed;
    int64_t remaining; /* closeAfter: sends left before the channel closes itself; -1 = never */
    waiter_list recv_waiters;
    waiter_list send_waiters;
} veles_chan;

struct veles_race {
    veles_task *task;
    int64_t winner;
    int64_t ready;
    int64_t closed; /* the winning channel arm was closed */
    int64_t narms;
    /* registered sources: as many as the race has arms (veles_race_new),
     * so a two-arm race (every withTimeout) is small and any count fits */
    struct {
        veles_chan *ch;
        int64_t send;   /* a send arm (D108): `in` is the value it offers */
        void *in;
        void *out;
        int64_t deadline;
        veles_task *awaited;
        veles_waiter node; /* its entry in ch's receivers (senders, for a send arm) */
    } arms[];
};

/* one shard of the timers (timers, below); one per cache line */
typedef struct timer_shard {
    int32_t lock;
    veles_task **heap;      /* on the GC heap: it keeps the sleeping tasks alive (a root) */
    int64_t len, cap;
    int64_t first;          /* the earliest deadline, 0 = none (lock; read without it) */
} __attribute__((aligned(64))) timer_shard;

#define TIMER_SHARDS 64
static timer_shard timer_shards[TIMER_SHARDS];

/* a `time.ticker` (D110; see fire_tickers) */
typedef struct veles_ticker {
    veles_chan *ch;
    int64_t period, next_at;
    struct veles_ticker *next;
} veles_ticker;

static veles_ticker *tickers;
/* the tasks parked on sockets, by descriptor (socket waits, below) */
typedef struct io_entry {
    int64_t key;            /* the descriptor + 1; 0 = an empty slot */
    veles_task *readers, *writers;
} io_entry;

#define IO_SHARDS 64

/* the descriptors whose hash falls in it, under a spinlock of its own;
 * one per cache line */
typedef struct io_shard {
    int32_t lock;
    io_entry *table;        /* on the GC heap, word-scanned: it keeps the parked tasks alive (a root) */
    int64_t cap, used;
    int64_t *closing;       /* descriptors being closed (malloc) */
    int64_t nclosing, closing_cap;
} __attribute__((aligned(64))) io_shard;

static io_shard io_shards[IO_SHARDS];
static int64_t io_count;    /* tasks parked on a socket (atomic) */
#define current (veles_tls_get()->task)
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
static veles_cond *spare_cv;    /* a run queue was handed off (blocking calls), or the root finished */
#define rt_depth (veles_tls_get()->rt_depth)
static veles_task *root_task;   /* the task veles_run drives; NULL between runs */
static int64_t active_workers;  /* workers inside a task, of every executor (atomic) */

/* ---- executors (D143) --------------------------------------------------------
 * An executor is a set of worker threads with run queues of their own: the
 * default pool, which veles_run drives, a pool made by `Executor(...)`, one
 * thread (`Executor.thread`), and the blocking pool `blocking(f)` uses. A
 * task runs only on its executor's threads — the executor of the scope it
 * was launched in (`scope(on: e)`, or else the launching task's) — so every
 * wake puts it back in that executor's queues. Timers and sockets stay one
 * set for the program, served by the default pool's threads: a pool's
 * threads only run tasks, and one that arms a timer or waits on a socket
 * has the default pool take a look (kick_housekeeper). */
typedef struct veles_worker veles_worker;
#define MAX_WORKERS 256

enum { X_DEFAULT, X_POOL, X_THREAD, X_BLOCKING };

typedef struct veles_exec {
    int32_t gq_lock;           /* the shared queue: a woken task from elsewhere, a yield, an overflow */
    int32_t kind;              /* X_* */
    int64_t gq_len;
    veles_task *gq_head, *gq_tail;
    int32_t idle_lock;         /* the idle threads (a spinlock) */
    struct idle_node *idle_top;
    int64_t idle_workers;      /* threads parked idle here (atomic) */
    int64_t waking;            /* a thread was woken for work and has not come back yet (atomic) */
    int64_t spinning;          /* workers looking for work before they sleep (atomic; Go's nmspinning) */
    int64_t nworkers;          /* run queues (atomic): grows to target, for the default pool as threads join */
    int64_t target;            /* how many threads it has; the blocking pool's most */
    int64_t threads;           /* threads still in its loop (runtime lock) */
    int64_t stopping;          /* closed: its threads leave once out of work */
    veles_cond *stopped_cv;    /* the last thread left */
    int32_t priority;          /* what its threads are started with (veles_thread_configure) */
    int32_t ncpus;
    int32_t *cpus;
    char name[32];
    struct veles_exec *next;   /* every live executor, for the monitor */
    veles_worker *workers[];   /* target of them; each one reachable through here */
} veles_exec;

static veles_exec *default_exec; /* the default pool; GC root */
static veles_exec *execs;        /* the others still open (runtime lock); GC root */
static veles_exec *blocking_exec;

/* the executor a task runs on: its scope's, or the default pool's for a
 * task outside any scope (the root, a test) */
static inline veles_exec *exec_of(veles_task *t) {
    return t->scope && t->scope->exec ? t->scope->exec : default_exec;
}

/* the reactor (veles_poll.c) */
typedef struct veles_poll_event {
    int64_t fd;
    int32_t read;
    int32_t write;
} veles_poll_event;
void veles_poll_init(void);
int64_t veles_poll_arm(int64_t fd, int64_t read, int64_t write);
int64_t veles_poll_wait(int64_t timeout_ns, veles_poll_event *out, int64_t max);
void veles_poll_wake(void);
static void rt_enter(void);
static void rt_exit(void);

static int64_t poller_busy;     /* a thread is waiting in the reactor, or claimed to (atomic) */
static int64_t poller_blocked;  /* ... and may be blocked there: work must interrupt it (atomic) */
static int64_t poll_until;      /* the deadline it waits until, monotonic ns (atomic) */

static void grow_blocking(veles_exec *e);
static void spin_lock(int32_t *l);
static void spin_unlock(int32_t *l);

/* ---- idle threads (F11) ---------------------------------------------------------
 * A thread out of work parks on a park of its own (veles_sync.c) and is
 * listed among its executor's idle threads; a wake takes one off the list
 * and wakes that one by name — no lock but the list's spinlock, held for a
 * few instructions. (Each executor once had one condition variable under
 * the runtime lock: every wake took the lock, the system picked the waiter,
 * and on Windows every timeout was whole milliseconds.) A thread lists
 * itself before its last look at the queues, and a waker queues before it
 * looks at the list (both ordered by sequentially consistent fences), so
 * one of the two always sees the other. */
typedef struct idle_node {
    veles_park *park;
    struct idle_node *next, *prev;
    int32_t listed;          /* on an executor's list (its idle_lock) */
} idle_node;

/* Nodes are never freed: a waker may still hold one it took off a list
 * when its thread ends (a closed pool's, an idle blocking thread's). An
 * ending thread leaves its node here for the next thread to start; a late
 * wake then only makes that thread look again. */
static idle_node *spare_nodes;
static int32_t spare_nodes_lock;

/* this thread's node, made (or reused) on its first idle wait */
static idle_node *my_idle(void) {
    veles_tls *tls = veles_tls_get();
    idle_node *n = tls->idle;
    if (!n) {
        spin_lock(&spare_nodes_lock);
        n = spare_nodes;
        if (n) spare_nodes = n->next;
        spin_unlock(&spare_nodes_lock);
        if (n) {
            n->next = n->prev = NULL;
        } else {
            n = calloc(1, sizeof *n);
            if (!n) veles_panic("out of memory", 13);
            n->park = veles_park_new();
        }
        tls->idle = n;
    }
    return n;
}

/* the calling thread ends: its node, off every list, goes to the spares */
static void idle_node_release(void) {
    veles_tls *tls = veles_tls_get();
    idle_node *n = tls->idle;
    if (!n) return;
    tls->idle = NULL;
    spin_lock(&spare_nodes_lock);
    n->next = spare_nodes;
    spare_nodes = n;
    spin_unlock(&spare_nodes_lock);
}

static void idle_push(veles_exec *e, idle_node *n) {
    spin_lock(&e->idle_lock);
    n->prev = NULL;
    n->next = e->idle_top;
    if (e->idle_top) e->idle_top->prev = n;
    e->idle_top = n;
    n->listed = 1;
    __atomic_add_fetch(&e->idle_workers, 1, __ATOMIC_SEQ_CST);
    spin_unlock(&e->idle_lock);
    __atomic_thread_fence(__ATOMIC_SEQ_CST);
}

static void idle_unlink(veles_exec *e, idle_node *n) {
    if (n->prev) n->prev->next = n->next; else e->idle_top = n->next;
    if (n->next) n->next->prev = n->prev;
    n->next = n->prev = NULL;
    n->listed = 0;
    __atomic_sub_fetch(&e->idle_workers, 1, __ATOMIC_SEQ_CST);
}

/* one idle thread off the list, the most recent (its cache is the warmest),
 * or NULL */
static idle_node *idle_pop(veles_exec *e) {
    if (!__atomic_load_n(&e->idle_workers, __ATOMIC_SEQ_CST)) return NULL;
    spin_lock(&e->idle_lock);
    idle_node *n = e->idle_top;
    if (n) idle_unlink(e, n);
    spin_unlock(&e->idle_lock);
    return n;
}

/* the thread takes itself off the list after its wait: 1 when it was still
 * on it (it woke by itself), 0 when a waker took it off to wake it */
static int idle_leave(veles_exec *e, idle_node *n) {
    spin_lock(&e->idle_lock);
    int listed = n->listed;
    if (listed) idle_unlink(e, n);
    spin_unlock(&e->idle_lock);
    return listed;
}

/* every thread idle on e now: the root finished or started, the executor
 * stops. The list is taken whole, then woken: popping until it was empty
 * never ended while the woken threads, finding nothing, listed themselves
 * again (a test run's next root then never started, 2026-10-10) */
static void wake_all(veles_exec *e) {
    idle_node *few[256], **taken = few;
    int64_t cap = 256, k = 0;
    spin_lock(&e->idle_lock);
    while (e->idle_top) {
        if (k == cap) {
            /* more than 256 idle (blocking threads count): the rest in a
             * bigger array, allocated without the spinlock */
            spin_unlock(&e->idle_lock);
            idle_node **more = malloc((size_t)cap * 2 * sizeof *more);
            if (!more) veles_panic("out of memory", 13);
            memcpy(more, taken, (size_t)k * sizeof *more);
            if (taken != few) free(taken);
            taken = more;
            cap *= 2;
            spin_lock(&e->idle_lock);
            continue;
        }
        idle_node *n = e->idle_top;
        idle_unlink(e, n);
        taken[k++] = n;
    }
    spin_unlock(&e->idle_lock);
    /* the links are not read after the lock: a woken thread may be listed
     * again by then */
    for (int64_t i = 0; i < k; i++) veles_park_wake(taken[i]->park);
    if (taken != few) free(taken);
}

/* A task was queued on e: wake one of its idle threads, unless one is
 * already on its way. Waking a thread per queued task would have them all
 * fight over the queues; instead the woken thread, once it has a task,
 * wakes the next if more are waiting — the number of awake threads follows
 * the work (Go's spinning threads). */
static void wake_worker(veles_exec *e) {
    /* the caller's queueing is ordered before the flags are read, as a
     * spinner that gives up, an idle thread listing itself and the poller
     * order theirs before they look at the queues again */
    __atomic_thread_fence(__ATOMIC_SEQ_CST);
    /* a thread is already looking: it finds the task, and wakes the next
     * if more are waiting */
    if (__atomic_load_n(&e->waking, __ATOMIC_RELAXED) || __atomic_load_n(&e->spinning, __ATOMIC_SEQ_CST)) return;
    if (__atomic_load_n(&e->idle_workers, __ATOMIC_SEQ_CST) == 0) {
        /* the one idle thread may be the one waiting for sockets (Go's
         * netpollBreak): one system call, however many wakes */
        if (e == default_exec && __atomic_load_n(&poller_blocked, __ATOMIC_SEQ_CST)) veles_poll_wake();
        /* every blocking thread busy: one more, up to the bound */
        if (e->kind == X_BLOCKING) grow_blocking(e);
        return;
    }
    int64_t none = 0;
    if (!__atomic_compare_exchange_n(&e->waking, &none, 1, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) return;
    idle_node *n = idle_pop(e);
    if (n) {
        veles_park_wake(n->park);
        return;
    }
    __atomic_store_n(&e->waking, 0, __ATOMIC_SEQ_CST);
    if (e == default_exec && __atomic_load_n(&poller_blocked, __ATOMIC_SEQ_CST)) veles_poll_wake();
}

/* The default pool's timekeeper: of its idle threads, the one parked until
 * the nearest timer (the others park without a limit), unless a thread
 * waits in the reactor, which waits until the nearest timer itself. A
 * timer armed sooner than the one it waits for wakes it (timer_armed). */
static idle_node *keeper;       /* atomic */
static int64_t keeper_at;       /* its deadline (atomic) */

/* a thread of another executor waits on a socket, which the default pool
 * serves: one of its idle threads takes a look */
static void kick_housekeeper(void) {
    if (__atomic_load_n(&poller_blocked, __ATOMIC_SEQ_CST)) {
        veles_poll_wake();
        return;
    }
    wake_worker(default_exec);
}

int64_t veles_time_monotonic_ns(void);

/* a timer was armed for at: whoever waits until a later deadline for the
 * default pool — the thread in the reactor, the timekeeper — looks again;
 * with neither, an idle thread becomes the timekeeper. A thread of the
 * default pool that is busy looks at the timers between its tasks. */
static void timer_armed(int64_t at) {
    if (__atomic_load_n(&poller_blocked, __ATOMIC_SEQ_CST)) {
        if (at < __atomic_load_n(&poll_until, __ATOMIC_SEQ_CST)) veles_poll_wake();
        return;
    }
    idle_node *k = __atomic_load_n(&keeper, __ATOMIC_SEQ_CST);
    if (k) {
        if (at < __atomic_load_n(&keeper_at, __ATOMIC_SEQ_CST)) veles_park_wake(k->park);
        return;
    }
    if (__atomic_load_n(&default_exec->idle_workers, __ATOMIC_SEQ_CST)) {
        idle_node *n = idle_pop(default_exec);
        if (n) veles_park_wake(n->park);
    }
}
static int64_t workers_started;

static void rt_enter(void) {
    veles_lock_enter_kept(rt_lock);
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

/* an executor with room for slots run queues (word-scanned: its queue and
 * workers keep their tasks alive) */
static veles_exec *exec_alloc(int32_t kind, int64_t slots) {
    veles_exec *e = veles_alloc_words((int64_t)(sizeof *e + (size_t)slots * sizeof e->workers[0]));
    e->kind = kind;
    e->target = slots;
    e->stopped_cv = veles_cond_new();
    return e;
}

static void register_roots(void) {
    if (roots_registered) return;
    roots_registered = true;
    veles_gc_root(&default_exec, NULL);
    veles_gc_root(&execs, NULL);
    veles_gc_root(&blocking_exec, NULL);
    for (int i = 0; i < TIMER_SHARDS; i++) veles_gc_root(&timer_shards[i].heap, NULL);
    veles_gc_root(&tickers, NULL);
    for (int i = 0; i < IO_SHARDS; i++) veles_gc_root(&io_shards[i].table, NULL);
    veles_gc_root(&root_task, NULL);
}

/* the runtime lock exists before the first task does */
void veles_task_init(void) {
    if (rt_lock) return;
    rt_lock = veles_lock_new();
    spare_cv = veles_cond_new();
    veles_timer_period();
    veles_poll_init();
    register_roots();
    default_exec = exec_alloc(X_DEFAULT, MAX_WORKERS);
    memcpy(default_exec->name, "veles", 6);
}

/* The executor's clock: the monotonic nanoseconds Stopwatch reads too, so
 * a deadline (a sleep, a race's timeout, a ticker) is never before the time
 * asked for. It was GetTickCount64 on Windows — 15.6 ms steps — truncated
 * to milliseconds: `sleep(10 ms)` returned after anything from 0 to 10 ms
 * (2026-10-07, driver TestTimerHeapUnderThreads). */
int64_t veles_time_monotonic_ns(void);
#define NS_PER_MS 1000000

static int64_t now_ns(void) {
    return veles_time_monotonic_ns();
}

/* how long a wait for deadline at is, in nanoseconds (a condition variable
 * or the reactor waits that long, to the platform's resolution): 0 once it
 * passed */
static int64_t ns_until(int64_t at) {
    int64_t d = at - now_ns();
    return d <= 0 ? 0 : d;
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
#define RUNNEXT_TURNS 32 /* runnext turns in a row before the queues get one */

/* a run queue's owner (blocking calls, below) */
enum { B_RUNNING, B_BLOCKED, B_HANDED_OFF };

struct veles_worker {
    veles_exec *exec;        /* the executor whose tasks it runs */
    int32_t lock;
    int64_t bstate;          /* B_*: whether its thread is inside a blocking call */
    int64_t bseq;            /* blocking calls entered, so the monitor tells a long one from many */
    int64_t seen;            /* the monitor's: bseq when it last saw the thread blocked */
    uint32_t head, tail;     /* head: the next to take; tail: the next free slot */
    uint32_t seed;           /* where stealing starts looking */
    int64_t ticks;           /* tasks taken, for the shared queue's turn */
    veles_task *runnext;     /* woken by the task running here: runs next here */
    int64_t runnext_streak;
    uint32_t run_seq;        /* odd while a task runs here, one more each start and end (D145) */
    uint32_t seen_run;       /* the monitor's: run_seq at its last look */
    int32_t seen_looks;      /* the monitor's: looks in a row that saw the same task running */
    int32_t preempt;         /* the task running here has run long while others wait: its next back edge in a suspending function yields */
    veles_worker *rn_victim; /* steal: the worker whose runnext it saw last, in which run, which task */
    uint32_t rn_seq;
    veles_task *rn_task;
    veles_task *ring[RING];
};

/* the calling thread runs the default pool's tasks, or no executor's */
static inline int on_default_pool(void) {
    veles_worker *w = veles_tls_get()->worker;
    return !w || w->exec == default_exec;
}

static inline void cpu_pause(void) {
#if defined(__x86_64__) || defined(__i386__)
    __builtin_ia32_pause();
#elif defined(__aarch64__)
    __asm__ __volatile__("yield");
#endif
}

/* Every section under these locks is short, so a waiter spins first; past
 * that the holder has most likely lost its core (preempted by the OS), and
 * spinning on would only burn this one until it gets it back — so the
 * waiter gives its time slice up instead. The thread stays in Veles code,
 * as when spinning: a collection waits for it the same way. */
static void spin_lock(int32_t *l) {
    int spins = 0;
    while (__atomic_exchange_n(l, 1, __ATOMIC_ACQUIRE)) {
        while (__atomic_load_n(l, __ATOMIC_RELAXED)) {
            if (++spins < 128) {
                cpu_pause();
                continue;
            }
#if defined(_WIN32)
            SwitchToThread();
#else
            sched_yield();
#endif
        }
    }
}

static void spin_unlock(int32_t *l) {
    __atomic_store_n(l, 0, __ATOMIC_RELEASE);
}

static void push_global_chain(veles_exec *e, veles_task *first, veles_task *last, int64_t n) {
    last->next = NULL;
    spin_lock(&e->gq_lock);
    if (e->gq_tail) e->gq_tail->next = first; else e->gq_head = first;
    e->gq_tail = last;
    __atomic_add_fetch(&e->gq_len, n, __ATOMIC_SEQ_CST);
    spin_unlock(&e->gq_lock);
}

static void push_global(veles_exec *e, veles_task *t) {
    push_global_chain(e, t, t, 1);
}

static veles_task *pop_global(veles_exec *e) {
    if (!__atomic_load_n(&e->gq_len, __ATOMIC_SEQ_CST)) return NULL;
    spin_lock(&e->gq_lock);
    veles_task *t = e->gq_head;
    if (t) {
        e->gq_head = t->next;
        if (!e->gq_head) e->gq_tail = NULL;
        t->next = NULL;
        __atomic_sub_fetch(&e->gq_len, 1, __ATOMIC_SEQ_CST);
    }
    spin_unlock(&e->gq_lock);
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
    push_global_chain(w->exec, first, t, RING / 2 + 1);
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
    veles_exec *e = w->exec;
    int64_t n = __atomic_load_n(&e->nworkers, __ATOMIC_ACQUIRE);
    if (n < 2) return NULL;
    w->seed = w->seed * 1103515245u + 12345u;
    for (int64_t i = 0; i < n; i++) {
        veles_worker *v = e->workers[(w->seed + (uint32_t)i) % (uint32_t)n];
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
    /* no ring had work: a task woken to run next on a worker still running
     * the task that woke it (Go steals runnext the same way). Its owner
     * usually parks soon and runs it there, warm in its cache, so it is
     * taken only when this stealer's next look (a spin round later, a few
     * microseconds) finds the same task there and the owner still in the
     * same run — a launched child no longer waits for its parent to stop
     * computing (F9), and a ping-pong of two tasks is left alone */
    for (int64_t i = 0; i < n; i++) {
        veles_worker *v = e->workers[(w->seed + (uint32_t)i) % (uint32_t)n];
        if (!v || v == w) continue;
        veles_task *rn = __atomic_load_n(&v->runnext, __ATOMIC_ACQUIRE);
        if (!rn) continue;
        uint32_t seq = __atomic_load_n(&v->run_seq, __ATOMIC_ACQUIRE);
        if (!(seq & 1)) continue; /* between runs: the owner takes it next */
        if (w->rn_victim != v || w->rn_seq != seq || w->rn_task != rn) {
            w->rn_victim = v; /* first sighting: look again later */
            w->rn_seq = seq;
            w->rn_task = rn;
            continue;
        }
        w->rn_victim = NULL;
        w->rn_task = NULL;
        if (__atomic_compare_exchange_n(&v->runnext, &rn, NULL, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) return rn;
    }
    return NULL;
}

static void wake_worker(veles_exec *e);
static void note_timer(int64_t at);
static void finish_unstarted(veles_task *t);

/* puts a task that has just become S_QUEUED where a worker of its executor
 * will find it. Woken by the task running on this worker — a send to a
 * waiting receiver, a child finishing for its owner, a spawn — it runs here
 * next, while what the two share is still in this core's cache (Go's
 * runnext); the task it displaces goes to the ring. A yield goes to the
 * back of the shared queue, behind everything already waiting; so does a
 * task of another executor (D143). */
static void place(veles_task *t) {
    veles_tls *tls = veles_tls_get();
    veles_worker *w = tls->worker;
    veles_exec *e = exec_of(t);
    /* inside a callback from C the queue may be changing hands (blocking
     * calls, below): the shared queue */
    if (!w || w->exec != e || t->yielded || tls->callback_depth > 0) {
        push_global(e, t);
        wake_worker(e);
        return;
    }
    if (tls->in_resume && tls->task && tls->task != t) {
        veles_task *old = __atomic_exchange_n(&w->runnext, t, __ATOMIC_SEQ_CST);
        if (!old) {
            /* an idle worker takes it if this one goes on running (steal) */
            wake_worker(e);
            return;
        }
        t = old;
    }
    push_local(w, t);
    wake_worker(e);
}

/* makes a task runnable: queues it, or, while a worker is inside it, has
 * that worker queue it again once the frame parks. Called with the runtime
 * lock held (the task's state is read). */
static void enqueue(veles_task *t) {
    if (__atomic_load_n(&t->state, __ATOMIC_SEQ_CST) >= T_DONE) return;
    for (;;) {
        int32_t s = __atomic_load_n(&t->sched, __ATOMIC_SEQ_CST);
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
    int32_t blocked = T_BLOCKED;
    __atomic_compare_exchange_n(&t->state, &blocked, T_RUNNABLE, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST);
    place(t);
}

/* hands this worker's runnext to the shared queue: the thread is about to
 * block, and the task must not wait for it */
static void flush_runnext(void) {
    veles_worker *w = veles_tls_get()->worker;
    if (!w || !w->runnext) return;
    veles_task *t = __atomic_exchange_n(&w->runnext, NULL, __ATOMIC_SEQ_CST);
    if (!t) return;
    push_global(w->exec, t);
    wake_worker(w->exec);
}

/* the next task for worker w, or NULL: now and then the shared queue
 * first, so a busy ring cannot starve it; then runnext, unless it has had
 * RUNNEXT_TURNS turns in a row; the ring; the shared queue; another
 * worker's ring */
static veles_task *find_task(veles_worker *w) {
    veles_task *t;
    veles_exec *e = w->exec;
    if (++w->ticks % 61 == 0 && (t = pop_global(e))) return t;
    /* runnext changes by exchange: the monitor may take it (D145) */
    if (w->runnext && w->runnext_streak < RUNNEXT_TURNS && (t = __atomic_exchange_n(&w->runnext, NULL, __ATOMIC_SEQ_CST))) {
        w->runnext_streak++;
        return t;
    }
    w->runnext_streak = 0;
    if ((t = pop_local(w))) return t;
    if ((t = pop_global(e))) return t;
    if ((t = steal(w))) return t;
    if (w->runnext) t = __atomic_exchange_n(&w->runnext, NULL, __ATOMIC_SEQ_CST);
    return t;
}

/* any task another worker of e could take: the shared queue or a ring. A
 * runnext slot does not count — only its worker runs it, and a worker
 * with one is active — or idle workers would spin instead of sleeping. */
static int anything_queued(veles_exec *e) {
    if (__atomic_load_n(&e->gq_len, __ATOMIC_SEQ_CST)) return 1;
    int64_t n = __atomic_load_n(&e->nworkers, __ATOMIC_ACQUIRE);
    for (int64_t i = 0; i < n; i++) {
        veles_worker *w = e->workers[i];
        if (w && !ring_empty(w)) return 1;
    }
    return 0;
}

/* a task in some worker's runnext slot. It is not work for an idle worker
 * (only its own worker runs it), but it is work: a worker that left its
 * loop for the timers, holding one, is waiting for the lock a sleeper
 * holds while it decides whether the program is deadlocked. */
static int any_runnext(veles_exec *e) {
    int64_t n = __atomic_load_n(&e->nworkers, __ATOMIC_ACQUIRE);
    for (int64_t i = 0; i < n; i++) {
        veles_worker *w = e->workers[i];
        if (w && __atomic_load_n(&w->runnext, __ATOMIC_ACQUIRE)) return 1;
    }
    return 0;
}

/* work queued on any executor (runtime lock held: the list of executors
 * is its) — what an idle default thread must see before it calls the
 * program deadlocked */
static int work_anywhere(void) {
    if (anything_queued(default_exec) || any_runnext(default_exec)) return 1;
    for (veles_exec *e = execs; e; e = e->next)
        if (anything_queued(e) || any_runnext(e)) return 1;
    veles_exec *b = blocking_exec;
    return b && (anything_queued(b) || any_runnext(b));
}

static void wake(veles_task *t) {
    if (!t) return;
    int64_t s = __atomic_load_n(&t->sched, __ATOMIC_SEQ_CST);
    int64_t st = __atomic_load_n(&t->state, __ATOMIC_SEQ_CST);
    if (st == T_BLOCKED || ((s == S_RUNNING || s == S_WOKEN) && st == T_RUNNABLE)) enqueue(t);
}

/* D145: a running task that is cancelled (or whose scope body a failed
 * child abandons) is counted in veles_attention, so the next back edge of
 * the loop it is in takes a look; once it stops running, its next
 * suspension point looks anyway, and the count goes */

static void attention_on(veles_task *t) {
    if (__atomic_exchange_n(&t->attn, 1, __ATOMIC_SEQ_CST) == 0)
        __atomic_add_fetch(&veles_attention, 1, __ATOMIC_SEQ_CST);
}

static void attention_off(veles_task *t) {
    if (__atomic_exchange_n(&t->attn, 0, __ATOMIC_SEQ_CST) == 1)
        __atomic_sub_fetch(&veles_attention, 1, __ATOMIC_SEQ_CST);
}

/* the frame of t has parked (or finished): a wake that came while it ran
 * queues it now. Needs no lock: only this worker leaves S_RUNNING/S_WOKEN. */
static void after_run(veles_task *t) {
    attention_off(t);
    int32_t s = S_RUNNING;
    if (__atomic_compare_exchange_n(&t->sched, &s, S_IDLE, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) return;
    /* S_WOKEN */
    if (t->state == T_DONE || t->state == T_CANCELLED || t->state == T_ENDING) {
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

/* The outermost blocking call of a thread with a run queue of the default
 * pool marks it: the task that would run next goes to the shared queue
 * first, since the queue may change hands before the call returns. Inside a
 * callback from C the outer call's mark stands (and place() queues nothing
 * here). Another executor's queue is never handed off — its threads are
 * the ones it was made with, and `Executor.thread`'s tasks run on one
 * thread (D143) — so its thread is only safe for the collector meanwhile. */
static void block_worker(void) {
    veles_tls *tls = veles_tls_get();
    if (tls->blocking++ > 0 || tls->callback_depth > 0) return;
    veles_worker *w = tls->worker;
    if (!w || w->exec != default_exec) return;
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
    if (!w || w->exec != default_exec) return;
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
    veles_exec *e = default_exec;
    if (__atomic_load_n(&e->idle_workers, __ATOMIC_SEQ_CST) > 0) return 0; /* an idle worker takes it */
    /* parked sockets need a thread only while none waits in the reactor */
    return anything_queued(e) || timers_due() ||
           (__atomic_load_n(&io_count, __ATOMIC_RELAXED) > 0 && !__atomic_load_n(&poller_busy, __ATOMIC_RELAXED));
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
    veles_exec *e = default_exec;
    int64_t n = __atomic_load_n(&e->nworkers, __ATOMIC_ACQUIRE);
    for (int64_t i = 0; i < n; i++) {
        veles_worker *w = e->workers[i];
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

/* the monitor's look at the run queues of e for a task that runs long
 * (D145); 1 when one of them runs a task */
static int look_at_runs(veles_exec *e) {
    int busy = 0;
    int64_t nw = __atomic_load_n(&e->nworkers, __ATOMIC_ACQUIRE);
    for (int64_t i = 0; i < nw; i++) {
        veles_worker *w = e->workers[i];
        if (!w) continue;
        uint32_t seq = __atomic_load_n(&w->run_seq, __ATOMIC_RELAXED);
        if (!(seq & 1) || seq != w->seen_run) {
            w->seen_run = seq;
            w->seen_looks = 0;
            busy |= (int)(seq & 1);
            continue;
        }
        busy = 1;
        ++w->seen_looks;
        /* the same task, running since the last look (at least 5 ms):
         * the task it woke to run next here — a child it launched, say —
         * goes where an idle worker takes it, instead of waiting for a
         * loop to end */
        veles_task *rn = __atomic_load_n(&w->runnext, __ATOMIC_ACQUIRE) ? __atomic_exchange_n(&w->runnext, NULL, __ATOMIC_SEQ_CST) : NULL;
        if (rn) {
            push_global(e, rn);
            wake_worker(e);
        }
        /* since two looks ago, about 10 ms: if others wait for the
         * thread, its next back edge in a suspending function gives it
         * up */
        if (w->seen_looks >= 2 && (anything_queued(e) || (e == default_exec && timers_due())) &&
            __atomic_exchange_n(&w->preempt, 1, __ATOMIC_SEQ_CST) == 0)
            __atomic_add_fetch(&veles_attention, 1, __ATOMIC_SEQ_CST);
    }
    return busy;
}

static void fire_timers(void);
static void refresh_timer_due(void);
static void poll_io_quick(void);
/* the earliest timer deadline, readable without a lock (0: none): a worker
 * running tasks goes back for the timers once it has passed */
static int64_t timer_due;
static int64_t last_poll; /* when a thread last looked at the sockets (monotonic ns) */

/* The monitor moves tasks between queues and fires timers, so it is one of
 * the threads a collection stops: it registers with the collector, and is
 * in a safe region whenever it waits. (Moving a task it held only in a
 * local while a collection marked could lose it.) */
static void monitor_main(void *arg) {
    (void)arg;
    veles_thread_attach();
    int64_t quiet = 0; /* looks in a row that found no thread blocked and none running */
    int blocked_seen = 0;
    for (;;) {
        veles_enter_safe();
        veles_lock_acquire(monitor_lock);
        if (quiet > 100) {
            __atomic_store_n(&monitor_asleep, 1, __ATOMIC_SEQ_CST);
            if (!any_blocked()) veles_cond_wait(monitor_cv, monitor_lock, -1);
            __atomic_store_n(&monitor_asleep, 0, __ATOMIC_SEQ_CST);
            quiet = 0;
        } else {
            /* blocked calls are looked at every millisecond; tasks that
             * run long (D145) every five */
            veles_cond_wait(monitor_cv, monitor_lock, blocked_seen ? 1 : 5);
        }
        veles_lock_release(monitor_lock);
        veles_leave_safe();
        /* timers and sockets no thread of the default pool came back for:
         * each runs a long loop that cannot give its thread up, and the
         * tasks of the other executors — or the default pool's own, once a
         * thread frees up — need not wait for those to end (D143); Go's
         * sysmon does the same */
        int64_t due = __atomic_load_n(&timer_due, __ATOMIC_RELAXED);
        if (due && now_ns() - due >= NS_PER_MS) {
            fire_timers();
            refresh_timer_due();
        }
        if (__atomic_load_n(&io_count, __ATOMIC_RELAXED) && now_ns() - __atomic_load_n(&last_poll, __ATOMIC_RELAXED) >= 10 * NS_PER_MS)
            poll_io_quick();
        int any = 0;
        int busy = look_at_runs(default_exec);
        /* the other executors come and go: their list is the runtime
         * lock's, and a closed one may be collected once off it */
        if (__atomic_load_n(&execs, __ATOMIC_ACQUIRE)) {
            rt_enter();
            for (veles_exec *e = execs; e; e = e->next) busy |= look_at_runs(e);
            rt_exit();
        }
        veles_exec *d = default_exec;
        int64_t n = __atomic_load_n(&d->nworkers, __ATOMIC_ACQUIRE);
        for (int64_t i = 0; i < n; i++) {
            veles_worker *w = d->workers[i];
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
        blocked_seen = any;
        quiet = any || busy ? 0 : quiet + 1;
    }
}

/* ---- tasks ----------------------------------------------------------------- */

veles_task *veles_task_new(void) {
    veles_task_init();
    veles_task *t = veles_alloc_words(sizeof *t);
    t->state = T_RUNNABLE;
    t->abandon_depth = INT32_MAX;
    /* a task started here — by async, or by the runtime — sees the
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

/* Test helpers (D78): while test code calls a `test fun`, the call site is
 * bound under a key veles_local_key never hands out, so a failure recorded
 * inside the helper — or in a task it started — can say where the test
 * called it from. The site text is a constant of the program. */
#define TEST_SITE_KEY (-1)

typedef struct {
    const char *where;
    int64_t len;
} test_site;

int64_t veles_test_enter(const char *where, int64_t len) {
    test_site *s = veles_alloc_words(sizeof *s);
    s->where = where;
    s->len = len;
    return (int64_t)(uintptr_t)veles_local_bind(TEST_SITE_KEY, s);
}

void veles_test_leave(int64_t prev) {
    veles_local_restore((void *)(uintptr_t)prev);
}

/* calls each for every helper call site in effect, innermost first */
void veles_test_sites(void (*each)(void *ctx, const char *where, int64_t len), void *ctx) {
    for (veles_local *l = *current_locals(); l; l = l->next) {
        if (l->key == TEST_SITE_KEY) {
            test_site *s = l->cell;
            each(ctx, s->where, s->len);
        }
    }
}

/* ---- the test runner's queue (D80) -----------------------------------------
 * The runner queues every test as a task; at most `test_jobs` run at once,
 * and a test that ends starts the next one waiting. A test's task is known
 * by the record bound in its locals (veles_sync.c), which the watchdog is
 * told about as the test starts and ends. */
#define TEST_REC_KEY (-2)

/* ---- the receiving scope (D111) ---------------------------------------------
 * While a `with` computes a value that holds tasks, the running task carries
 * the with's scope as a binding under this key; an `async` field argument
 * of the value's constructor launches into it, and a task started for a
 * suspending call inherits it with the other bindings. */
#define RECEIVING_KEY (-3)

void *veles_receiving_bind(veles_scope *s) {
    return veles_local_bind(RECEIVING_KEY, s);
}

void veles_receiving_restore(void *head) {
    veles_local_restore(head);
}

veles_scope *veles_receiving(void) {
    veles_scope *s = veles_local_find(RECEIVING_KEY);
    if (!s) {
        const char *m = "internal error: a held task has no receiving scope (D111)";
        veles_panic(m, (int64_t)strlen(m));
    }
    return s;
}

typedef struct test_pending {
    veles_task *t;
    void (*entry)(veles_task *, void *);
    void *args;
    struct test_pending *next;
} test_pending;

static veles_lock *test_q_lock;
static test_pending *test_q_head, *test_q_tail;
static int64_t test_jobs, test_running;

static int64_t worker_count(void);
void veles_test_watch_rec(void *rec, int64_t running);
void veles_task_spawn(veles_task *t, void (*entry)(veles_task *, void *), void *args);

static void *test_rec_of(veles_task *t) {
    for (veles_local *l = t->locals; l; l = l->next)
        if (l->key == TEST_REC_KEY) return l->cell;
    return NULL;
}

void veles_test_jobs(int64_t jobs) {
    test_q_lock = veles_lock_new();
    test_jobs = jobs > 0 ? jobs : worker_count();
    /* a waiting test is known only to the queue until it starts */
    veles_gc_root(&test_q_head, NULL);
    veles_gc_root(&test_q_tail, NULL);
}

static void test_start(veles_task *t, void (*entry)(veles_task *, void *), void *args) {
    veles_test_watch_rec(test_rec_of(t), 1);
    veles_task_spawn(t, entry, args);
}

void veles_test_queue(veles_task *t, void (*entry)(veles_task *, void *), void *args) {
    t->test_task = 1;
    veles_lock_acquire(test_q_lock);
    if (test_running < test_jobs) {
        test_running++;
        veles_lock_release(test_q_lock);
        test_start(t, entry, args);
        return;
    }
    test_pending *p = veles_alloc_words(sizeof *p);
    p->t = t;
    p->entry = entry;
    p->args = args;
    if (test_q_tail) test_q_tail->next = p; else test_q_head = p;
    test_q_tail = p;
    veles_lock_release(test_q_lock);
}

/* a test's task is done (returned, failed or panicked): its slot goes to
 * the next test waiting */
static void test_task_done(veles_task *t) {
    veles_test_watch_rec(test_rec_of(t), 0);
    veles_lock_acquire(test_q_lock);
    test_pending *p = test_q_head;
    if (p) {
        test_q_head = p->next;
        if (!test_q_head) test_q_tail = NULL;
    } else {
        test_running--;
    }
    veles_lock_release(test_q_lock);
    if (p) test_start(p->t, p->entry, p->args);
}

veles_task *veles_task_current(void) {
    return current;
}

void veles_task_set_current(veles_task *t) {
    current = t;
}

/* veles_task_started records the coroutine handle once the ramp returned. */
/* the task's ramp has returned: nothing to record, since a frame that
 * suspends parks the task on itself (veles_frame_park), which may be a
 * call's frame deeper than the ramp's own */
static void veles_task_started_impl(veles_task *t, void *hdl) {
    (void)t;
    (void)hdl;
}

static void scope_child_finished(veles_task *t);
static void scope_empty(veles_scope *s);
static void end_drained(veles_task *t);
static void scope_child_failed(veles_task *t);
static void remove_timer(veles_task *t);

/* called by the coroutine body before its final suspend */
static void veles_task_finish_impl(veles_task *t, const void *result, int64_t size, int64_t failed) {
    if (t->state == T_CANCELLED) return;
    t->result = veles_alloc_words(size > 0 ? size : 8);
    if (size > 0 && result) memcpy(t->result, result, (size_t)size);
    t->failed = failed;
    /* the scope fails before the result is published, so a task woken by
     * it sees the failure at its next check (D141) */
    scope_child_failed(t);
    /* released: an await that sees T_DONE without the lock sees the result */
    __atomic_store_n(&t->state, T_DONE, __ATOMIC_SEQ_CST);
    t->hdl = NULL;
    wake(__atomic_exchange_n(&t->waiter, NULL, __ATOMIC_SEQ_CST));
    scope_child_finished(t);
}

/* the cancelled task has run its cleanups (D43) and is done */
static void veles_task_finish_cancelled_impl(veles_task *t) {
    t->hdl = NULL;
    if (t->state == T_CANCELLED) return;
    __atomic_store_n(&t->state, T_CANCELLED, __ATOMIC_SEQ_CST);
    wake(__atomic_exchange_n(&t->waiter, NULL, __ATOMIC_SEQ_CST));
    scope_child_finished(t);
}

/* `await` on a fail-fast child that failed (D141): its error is its
 * scope's, so there is no value to give. The awaiting code is in that
 * scope — a handle does not leave it — and is about to be abandoned (the
 * body, at the check after this suspension point) or cancelled (a sibling,
 * or a call the body is inside): yield until that happens. A handle a
 * callee kept past the end of its scope is the one way to get here after
 * it; that is a panic, as awaiting a cancelled task is. */
static int64_t await_failed(veles_task *self, veles_task *target) {
    if (target->scope && __atomic_load_n(&target->scope->over, __ATOMIC_ACQUIRE))
        veles_panic("awaited task failed; its error left with its scope", 50);
    rt_enter();
    enqueue(self);
    rt_exit();
    return 0;
}

static int64_t done_and_failed(veles_task *t) {
    return t->unwrap && t->failed && !t->panicked;
}

/* await: true when the target is done, otherwise blocks the caller */
static int64_t veles_task_await_impl(veles_task *self, veles_task *target) {
    for (;;) {
        int64_t st = __atomic_load_n(&target->state, __ATOMIC_SEQ_CST);
        if (st == T_DONE) {
            if (done_and_failed(target)) return await_failed(self, target);
            return 1;
        }
        if (st == T_CANCELLED) {
            veles_panic("awaited task was cancelled", 26);
        }
        /* registered first, then the state read again: a task finishing
         * publishes its state and then takes its waiter, so one of the two
         * always sees the other */
        __atomic_store_n(&target->waiter, self, __ATOMIC_SEQ_CST);
        int32_t now = __atomic_load_n(&target->state, __ATOMIC_SEQ_CST);
        if (now == T_DONE || now == T_CANCELLED) { /* not T_ENDING: it is still unwinding */
            veles_task *me = self;
            __atomic_compare_exchange_n(&target->waiter, &me, NULL, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST);
            continue;
        }
        __atomic_store_n(&self->state, T_BLOCKED, __ATOMIC_SEQ_CST);
        return 0;
    }
}

void *veles_task_result(veles_task *t) {
    return t->result;
}

/* what `await` gives: the result, or for a fail-fast child (D141) the Ok
 * payload inside it */
void *veles_task_value(veles_task *t) {
    return (char *)t->result + (t->unwrap ? t->value_off : 0);
}

/* a launch in a fail-fast block of a function that throws (D141): the
 * payload of its Result sits at off */
void veles_task_set_unwrap(veles_task *t, int64_t off) {
    t->unwrap = 1;
    t->value_off = off;
}

int64_t veles_task_failed(veles_task *t) {
    return t->failed;
}

/* ---- scopes (D34/D36) ------------------------------------------------------ */

veles_scope *veles_scope_begin(veles_task *owner, int64_t fail_fast, int64_t depth) {
    veles_scope *s = veles_alloc_words(sizeof *s);
    s->owner = owner;
    s->exec = owner ? exec_of(owner) : default_exec; /* children run where the owner does (D143) */
    s->fail_fast = fail_fast;
    s->depth = (int32_t)depth;
    return s;
}

/* async: the task of a scope child. It joins the scope when it is spawned
 * (veles_task_spawn), after its arguments are evaluated — an argument that
 * throws leaves no child behind for the scope to wait for — and taking no
 * lock here leaves one runtime-lock round trip per launch. */
veles_task *veles_task_launch(veles_scope *s, int64_t site) {
    veles_task *t = veles_task_new();
    t->scope = s;
    t->index = (int16_t)site; /* a launch site in the source: far fewer than 2^15 per scope */
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
    if (t->state >= T_DONE || t->cancel_requested) return;
    t->cancel_requested = 1;
    /* inside a close() that suspends (D147): it goes on waiting for what
     * it waits for, and sees the request at the first point after it */
    if (__atomic_load_n(&t->shield, __ATOMIC_ACQUIRE)) return;
    /* parked: claimed first (S_IDLE → S_QUEUED), so no wake runs it while
     * its waits are taken apart here; a task that is running, or queued to
     * run, waits and leaves its waits on its own thread — races and sleeps
     * do so without the runtime lock (F9) — and sees the request at its
     * next suspension point (veles_task_cancelled), or at a loop's back
     * edge (D145), so the loops are told to look; a spawned task that has
     * not started is finished unstarted by the worker that takes it */
    int32_t sched = S_IDLE;
    if (!__atomic_compare_exchange_n(&t->sched, &sched, S_QUEUED, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) {
        if (sched == S_RUNNING) {
            attention_on(t);
            enqueue(t);
        }
        return;
    }
    remove_timer(t);
    remove_io_waiter(t);
    leave_channel_waits(t);
    /* the innermost frame resumes first and sees the request, so the
     * deepest call unwinds first and each frame returns to its caller,
     * which unwinds in turn (D35 v0.29) */
    if (t->hdl) {
        int32_t blocked = T_BLOCKED;
        __atomic_compare_exchange_n(&t->state, &blocked, T_RUNNABLE, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST);
        place(t);
        return;
    }
    __atomic_store_n(&t->sched, S_IDLE, __ATOMIC_SEQ_CST);
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

/* a failed child fails its fail-fast scope: its siblings are cancelled.
 * Called before the child publishes T_DONE (runtime lock held), and again,
 * harmlessly, as it leaves. */
static void scope_child_failed(veles_task *t) {
    veles_scope *s = t->scope;
    if (!s || !t->failed || !s->fail_fast || s->failed) return;
    __atomic_store_n(&s->failed, t, __ATOMIC_RELEASE);
    cancel_children(s, t);
    /* the body may be inside suspending calls: their frames unwind first,
     * as if cancelled, back to the frame running the body (runtime lock
     * held, as for any change of the owner's waits) */
    if (s->depth < __atomic_load_n(&s->owner->abandon_depth, __ATOMIC_ACQUIRE))
        __atomic_store_n(&s->owner->abandon_depth, s->depth, __ATOMIC_RELEASE);
    if (__atomic_load_n(&s->owner->sched, __ATOMIC_SEQ_CST) == S_RUNNING) attention_on(s->owner);
    /* the owner may be blocked in the scope body (a recv that will now
     * never complete): wake it so its next suspension point sees the
     * failure and abandons the body — unless it waits inside a close()
     * that suspends, which runs to its end first (D147) */
    if (!__atomic_load_n(&s->owner->shield, __ATOMIC_ACQUIRE)) wake(s->owner);
}

/* a scope's last child has finished: its owner sees that in its join —
 * or, unwound without its frame and waiting for it (T_ENDING), ends */
static void scope_empty(veles_scope *s) {
    if (__atomic_load_n(&s->draining, __ATOMIC_ACQUIRE) && __atomic_exchange_n(&s->draining, 0, __ATOMIC_SEQ_CST)) {
        end_drained(s->owner);
        return;
    }
    wake(s->owner);
}

static void scope_child_finished(veles_task *t) {
    veles_scope *s = t->scope;
    if (!s) return;
    int64_t left = scope_leave(t);
    scope_child_failed(t);
    if (left <= 0) scope_empty(s);
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
 * as usual, so nothing outlives the block (D34). An owner unwinding with
 * no frame to wait in (a panic; a cancellation in code that does not
 * suspend, D145) waits as T_ENDING instead: the scope drains, and the
 * last child to finish ends it. */
static void veles_scope_cancel_impl(veles_scope *s) {
    cancel_children(s, NULL);
    veles_task *o = s->owner;
    if (o->unwinding && o->panic) {
        __atomic_store_n(&s->draining, 1, __ATOMIC_SEQ_CST);
        __atomic_add_fetch(&o->panic->drain, 1, __ATOMIC_SEQ_CST);
        if (__atomic_load_n(&s->live, __ATOMIC_SEQ_CST) <= 0 && __atomic_exchange_n(&s->draining, 0, __ATOMIC_SEQ_CST))
            __atomic_sub_fetch(&o->panic->drain, 1, __ATOMIC_SEQ_CST);
    }
}

/* a child failed, and the body is abandoned at a suspension point (D34):
 * the children still running are cancelled. A suspending call the body
 * was inside has already unwound — its frames saw the abandonment as a
 * cancellation, innermost first, and returned — so nothing it opened is
 * still open when the scope rethrows (D3). */
static void veles_scope_abandon_impl(veles_scope *s, veles_task *owner) {
    cancel_children(s, NULL);
    /* the calls the body was inside have unwound to its frame already
     * (veles_task_cancelled): the frame takes the abandonment from here */
    if (__atomic_load_n(&owner->abandon_depth, __ATOMIC_ACQUIRE) >= s->depth)
        __atomic_store_n(&owner->abandon_depth, INT32_MAX, __ATOMIC_RELEASE);
}

/* wait for every child: true when done, otherwise blocks the owner */
static int64_t veles_scope_wait_impl(veles_task *owner, veles_scope *s) {
    if (__atomic_load_n(&s->live, __ATOMIC_SEQ_CST) <= 0) {
        __atomic_store_n(&s->over, 1, __ATOMIC_RELEASE);
        return 1;
    }
    __atomic_store_n(&owner->state, T_BLOCKED, __ATOMIC_SEQ_CST);
    return 0;
}

static veles_task *veles_scope_failed_impl(veles_scope *s) {
    return __atomic_load_n(&s->failed, __ATOMIC_ACQUIRE);
}

static int64_t veles_scope_failed_index_impl(veles_scope *s) {
    veles_task *f = __atomic_load_n(&s->failed, __ATOMIC_ACQUIRE);
    return f ? f->index : -1;
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

static void unlink_waiter(waiter_list *list, veles_waiter *w) {
    if (w->prev) w->prev->next = w->next; else list->head = w->next;
    if (w->next) w->next->prev = w->prev; else list->tail = w->prev;
    w->next = w->prev = NULL;
    w->in = NULL;
}

static veles_waiter *pop_waiter(waiter_list *list) {
    veles_waiter *w = list->head;
    if (w) {
        unlink_waiter(list, w);
        if (!w->race) w->task->chan_wait = NULL;
    }
    return w;
}

static bool race_claim(veles_race *r, int64_t arm);
static void race_ready(veles_race *r, wakes *k);

static bool being_cancelled(veles_task *t) {
    return __atomic_load_n(&t->state, __ATOMIC_ACQUIRE) == T_CANCELLED ||
           __atomic_load_n(&t->cancel_requested, __ATOMIC_ACQUIRE);
}

/* the first blocked sender that is not being cancelled, taken off the
 * list; a race's send arm (D108) only once this has claimed its race, so
 * the value is taken exactly when the arm wins */
static veles_waiter *pop_live_sender(veles_chan *c) {
    veles_waiter *w;
    while ((w = pop_waiter(&c->send_waiters))) {
        if (being_cancelled(w->task)) continue;
        if (w->race && !race_claim(w->race, w->arm)) continue; /* won elsewhere: not sent */
        return w;
    }
    return NULL;
}

/* a blocked send or recv of t was completed by the other side */
static void chan_complete(veles_task *t, wakes *k) {
    __atomic_store_n(&t->chan_done, 1, __ATOMIC_SEQ_CST);
    wakes_add(k, t);
}

/* appends w; a node still on a list (a waiter woken for another reason
 * and blocking again) leaves that one first, so it is never on two */
static void push_waiter(waiter_list *list, veles_waiter *w) {
    if (w->in) unlink_waiter(w->in, w);
    w->next = NULL;
    w->prev = list->tail;
    if (list->tail) list->tail->next = w; else list->head = w;
    list->tail = w;
    w->in = list;
}

/* takes w off list if it is there (a race's arm that never registered,
 * or one already taken, is on none) */
static void remove_waiter(waiter_list *list, veles_waiter *w) {
    if (w->in == list) unlink_waiter(list, w);
}

/* A race's winner is claimed by compare-and-swap — a value or a close on
 * one of its channels (under that channel's lock), its timer, or the race
 * itself finding an arm ready — so exactly one arm completes it. The
 * claimer fills the arm's slot and only then publishes ready.
 *
 * RACE_BUSY: the race's own task is pairing one of its arms with another
 * race's arm on the same channel (a send arm and a receive arm, D108),
 * which takes both claims or neither (race_pair). For those few
 * instructions a claimer waits instead of failing — a failed claim drops
 * the waiter, and the pairing may yet let go. */
#define RACE_BUSY (-2)

static bool race_claim(veles_race *r, int64_t arm) {
    for (;;) {
        int64_t none = -1;
        if (__atomic_compare_exchange_n(&r->winner, &none, arm, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) return true;
        if (none != RACE_BUSY) return false;
        cpu_pause();
    }
}

enum { PAIR_WON, PAIR_SELF_LOST, PAIR_PEER_LOST };

/* claims arm i of self's race r and arm j of the peer race p together
 * (the channel's lock held). Only a race's own task pairs it, inside
 * veles_race_wait, which holds the runtime lock — so p is never busy
 * itself, and two pairings never wait on each other. */
static int race_pair(veles_race *r, int64_t i, veles_race *p, int64_t j) {
    int64_t none = -1;
    if (!__atomic_compare_exchange_n(&r->winner, &none, RACE_BUSY, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) return PAIR_SELF_LOST;
    if (race_claim(p, j)) {
        __atomic_store_n(&r->winner, i, __ATOMIC_SEQ_CST);
        return PAIR_WON;
    }
    __atomic_store_n(&r->winner, -1, __ATOMIC_SEQ_CST);
    return PAIR_PEER_LOST;
}

static void race_ready(veles_race *r, wakes *k) {
    __atomic_store_n(&r->ready, 1, __ATOMIC_RELEASE);
    wakes_add(k, r->task);
}

/* parks self in c's receivers (or senders) through its own node (c's lock
 * held); a task woken for another reason and blocking again is moved to
 * the back, not added twice */
static void chan_block(veles_task *self, veles_chan *c, int64_t send, void *data) {
    waiter_list *list = send ? &c->send_waiters : &c->recv_waiters;
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
    while (c->recv_waiters.head) {
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

/* a blocked sender's value was taken: its send, or its race's send arm,
 * is complete */
static void sender_done(veles_waiter *s, wakes *k) {
    if (s->race) race_ready(s->race, k);
    else chan_complete(s->task, k);
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
            sender_done(s, k);
            chan_sent(c, k);
        }
        return true;
    }
    if ((s = pop_live_sender(c))) {
        memcpy(out, s->data, (size_t)c->elem);
        sender_done(s, k);
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
     * channel closed and panics, as any send on a closed channel does; a
     * race's send arm wins with "closed", and the race panics (D108) */
    while ((w = pop_waiter(&c->send_waiters))) {
        veles_race *r = w->race;
        if (r) {
            if (race_claim(r, w->arm)) {
                r->closed = 1;
                race_ready(r, k);
            }
            continue;
        }
        wakes_add(k, w->task);
    }
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

/* ---- timers ------------------------------------------------------------------
 * The tasks with a deadline (sleep, a race's timeout arm) are kept in
 * binary min-heaps by wake_at: arming and disarming are O(log n), the
 * nearest is the root. A server holds a timer per connection, and the list
 * this replaced was scanned whole by every worker's every pass
 * (bench/httphello). Each task's timer lives in the shard its address
 * picks, under that shard's spinlock: a request arms and disarms about a
 * dozen (every withTimeout), and one heap under one lock was found taken
 * three times in four at 32 threads (2026-10-10). A task knows its slot in
 * its shard's heap (timer_slot, index + 1; 0 = none). The heaps are on the
 * GC heap and rooted: they keep the sleeping tasks alive.
 *
 * A shard's heap, and a task's wake_at and timer_slot, change under the
 * shard's lock; nothing that allocates or takes another lock runs under it
 * — a collection must not wait for a thread spinning on it — so a heap
 * grows outside, and fire_timers wakes what it took out once the lock is
 * given back. */
static timer_shard *timer_shard_of(veles_task *t) {
    uint64_t h = ((uint64_t)(uintptr_t)t >> 4) * 0x9E3779B97F4A7C15ull;
    return &timer_shards[h >> 58];
}

static void heap_put(veles_task **heap, int64_t i, veles_task *t) {
    heap[i] = t;
    t->timer_slot = (int32_t)(i + 1);
}

static void heap_up(timer_shard *sh, int64_t i) {
    veles_task **heap = sh->heap;
    veles_task *t = heap[i];
    while (i > 0) {
        int64_t p = (i - 1) / 2;
        if (heap[p]->wake_at <= t->wake_at) break;
        heap_put(heap, i, heap[p]);
        i = p;
    }
    heap_put(heap, i, t);
}

static void heap_down(timer_shard *sh, int64_t i) {
    veles_task **heap = sh->heap;
    veles_task *t = heap[i];
    for (;;) {
        int64_t c = 2 * i + 1;
        if (c >= sh->len) break;
        if (c + 1 < sh->len && heap[c + 1]->wake_at < heap[c]->wake_at) c++;
        if (heap[c]->wake_at >= t->wake_at) break;
        heap_put(heap, i, heap[c]);
        i = c;
    }
    heap_put(heap, i, t);
}

/* the shard's earliest deadline, for those who read it without the lock */
static void shard_first(timer_shard *sh) {
    __atomic_store_n(&sh->first, sh->len > 0 ? sh->heap[0]->wake_at : 0, __ATOMIC_RELEASE);
}

/* takes t out of its shard's heap (its lock held); its wake_at stays
 * (sleep reads it) */
static void heap_remove(timer_shard *sh, veles_task *t) {
    int64_t i = t->timer_slot - 1;
    if (i < 0) return;
    t->timer_slot = 0;
    veles_task *last = sh->heap[--sh->len];
    sh->heap[sh->len] = NULL;
    if (last != t) {
        heap_put(sh->heap, i, last);
        heap_up(sh, i);
        heap_down(sh, last->timer_slot - 1);
    }
}

static void remove_timer(veles_task *t) {
    if (!t->timer_slot && !t->wake_at) return; /* no timer: no lock */
    timer_shard *sh = timer_shard_of(t);
    spin_lock(&sh->lock);
    heap_remove(sh, t);
    t->wake_at = 0;
    shard_first(sh);
    spin_unlock(&sh->lock);
}

/* arms t's timer for the absolute time at (monotonic nanoseconds) */
static void timer_at(veles_task *t, int64_t at) {
    timer_shard *sh = timer_shard_of(t);
    spin_lock(&sh->lock);
    heap_remove(sh, t);
    while (sh->len == sh->cap) {
        /* the bigger array is allocated without the lock (see above) */
        int64_t cap = sh->cap ? sh->cap * 2 : 16;
        spin_unlock(&sh->lock);
        veles_task **grown = veles_alloc_words(cap * (int64_t)sizeof *grown);
        spin_lock(&sh->lock);
        if (sh->cap >= cap) continue; /* another thread grew it meanwhile */
        if (sh->len) memcpy(grown, sh->heap, (size_t)sh->len * sizeof *grown);
        sh->heap = grown;
        sh->cap = cap;
    }
    t->wake_at = at;
    sh->heap[sh->len] = t;
    heap_up(sh, sh->len++);
    int first = sh->heap[0] == t;
    shard_first(sh);
    spin_unlock(&sh->lock);
    note_timer(at);
    /* the default pool fires timers: a thread of it waiting until a later
     * deadline looks again */
    if (first) timer_armed(at);
}

static void add_timer(veles_task *t, int64_t ns) {
    timer_at(t, now_ns() + ns);
}

/* sleep: true once the deadline passed; first call arms it and blocks.
 * A task can be woken before its deadline for another reason - a scope
 * child finishing wakes the scope's owner - and it then comes back here:
 * it must block again, its timer still armed. Returning 0 while leaving
 * the task T_RUNNABLE lost it: fire_timers' wake() only wakes a blocked
 * task, so the sleeper was never resumed and the executor reported a
 * deadlock (a producer finishing while main slept on a timer). */
static int64_t veles_task_sleep_impl(veles_task *self, int64_t ns) {
    if (__atomic_load_n(&self->wake_at, __ATOMIC_ACQUIRE) != 0) {
        if (now_ns() >= self->wake_at) {
            remove_timer(self);
            return 1;
        }
        self->state = T_BLOCKED;
        return 0;
    }
    if (ns <= 0) {
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
    add_timer(self, ns);
    self->state = T_BLOCKED;
    return 0;
}

/* ---- tickers (D110) ----------------------------------------------------------
 * `time.ticker(every)`: no task. Each period the timer pass offers the
 * time (a Timestamp: microseconds since the epoch) to the ticker's channel
 * as trySend does, so a reader that falls behind misses ticks rather than
 * queueing them. A ticker that fell behind by several periods ticks once
 * and starts counting again from now. The list is ticker_lock's, a
 * spinlock under which nothing allocates or wakes: the ticks are sent once
 * it is given back. */
int64_t veles_time_now_us(void);

static int32_t ticker_lock;

veles_ticker *veles_ticker_start(veles_chan *c, int64_t period_ns) {
    veles_task_init();
    veles_ticker *t = veles_alloc_words(sizeof *t);
    t->ch = c;
    t->period = period_ns < 1 ? 1 : period_ns;
    spin_lock(&ticker_lock);
    t->next_at = now_ns() + t->period;
    t->next = tickers;
    __atomic_store_n(&tickers, t, __ATOMIC_RELEASE);
    spin_unlock(&ticker_lock);
    note_timer(t->next_at);
    timer_armed(t->next_at); /* the default pool fires it */
    return t;
}

void veles_ticker_stop(veles_ticker *t) {
    spin_lock(&ticker_lock);
    for (veles_ticker **pp = &tickers; *pp; pp = &(*pp)->next) {
        if (*pp == t) {
            *pp = t->next;
            break;
        }
    }
    t->next = NULL;
    spin_unlock(&ticker_lock);
}

#define TICK_BATCH 16

static void fire_tickers(int64_t now) {
    if (!__atomic_load_n(&tickers, __ATOMIC_ACQUIRE)) return;
    for (;;) {
        veles_chan *due[TICK_BATCH];
        int n = 0, more = 0;
        spin_lock(&ticker_lock);
        for (veles_ticker *t = tickers; t; t = t->next) {
            if (now < t->next_at) continue;
            if (n == TICK_BATCH) {
                more = 1;
                break;
            }
            due[n++] = t->ch;
            t->next_at += t->period;
            if (t->next_at <= now) t->next_at = now + t->period;
        }
        spin_unlock(&ticker_lock);
        for (int i = 0; i < n; i++) {
            int64_t at = veles_time_now_us();
            wakes k = {0};
            spin_lock(&due[i]->lock);
            if (!due[i]->closed) chan_try_send_locked(due[i], &at, &k);
            spin_unlock(&due[i]->lock);
            wakes_run(&k);
        }
        if (!more) return;
    }
}

/* the nearest of the tickers' next ticks, 0 for none */
static int64_t nearest_tick(void) {
    if (!__atomic_load_n(&tickers, __ATOMIC_ACQUIRE)) return 0;
    int64_t nearest = 0;
    spin_lock(&ticker_lock);
    for (veles_ticker *t = tickers; t; t = t->next) {
        if (!nearest || t->next_at < nearest) nearest = t->next_at;
    }
    spin_unlock(&ticker_lock);
    return nearest;
}

#define FIRE_BATCH 64

/* the tasks whose deadline passed come out of each shard's heap under its
 * lock and are woken after it, a batch at a time. A race's timer arm claims
 * the race, unless another arm has: the task may be leaving that race on its
 * own thread meanwhile (it no longer holds the runtime lock), and a claim
 * then fails, or wakes it once too often, which every wait tolerates. */
static void fire_shard(timer_shard *sh, int64_t now) {
    for (;;) {
        veles_task *due[FIRE_BATCH];
        int n = 0;
        spin_lock(&sh->lock);
        while (n < FIRE_BATCH && sh->len > 0 && sh->heap[0]->wake_at <= now) {
            veles_task *t = sh->heap[0];
            heap_remove(sh, t);
            if (__atomic_load_n(&t->race, __ATOMIC_ACQUIRE)) t->wake_at = 0;
            due[n++] = t;
        }
        int more = n == FIRE_BATCH;
        shard_first(sh);
        spin_unlock(&sh->lock);
        for (int j = 0; j < n; j++) {
            veles_task *t = due[j];
            veles_race *race = __atomic_load_n(&t->race, __ATOMIC_ACQUIRE);
            if (!race) {
                wake(t);
                continue;
            }
            for (int64_t i = 0; i < race->narms; i++) {
                if (race->arms[i].deadline && race->arms[i].deadline <= now) {
                    if (race_claim(race, i)) wake(t);
                    break;
                }
            }
        }
        if (!more) return;
    }
}

static void fire_timers(void) {
    int64_t now = now_ns();
    fire_tickers(now);
    for (int i = 0; i < TIMER_SHARDS; i++) {
        int64_t first = __atomic_load_n(&timer_shards[i].first, __ATOMIC_ACQUIRE);
        if (first && first <= now) fire_shard(&timer_shards[i], now);
    }
}

/* ---- socket waits (std/net) ------------------------------------------------ */

/* Sockets are non-blocking; when a call would block, the task parks here
 * until the reactor (veles_poll.c) reports the descriptor ready (readable,
 * writable, or in error - the retried call then reports what happened).
 * The parked tasks are kept by descriptor, readers and writers apart, and
 * the descriptor is armed for the union of what they wait for. The table is
 * split in shards by the descriptor's hash, each under a spinlock of its
 * own: socket waits once took the runtime lock, four times an HTTP request,
 * and at 32 threads most of those found it taken (bench/httphello,
 * 2026-10-10). Under a shard's lock nothing allocates on the collected heap
 * (its table grows outside) and nobody is woken: the tasks to wake are
 * collected and woken once it is given back, since waking may take the
 * runtime lock, which the reactor's thread holds while it dispatches. */

static uint64_t io_hash(int64_t key) {
    return ((uint64_t)key * 0x9E3779B97F4A7C15ull) >> 17;
}

static io_shard *io_shard_of(int64_t fd) {
    return &io_shards[io_hash(fd + 1) & (IO_SHARDS - 1)];
}

/* the slot a key starts probing from in a table of cap slots (the hash's
 * low bits picked the shard) */
static int64_t io_slot(int64_t key, int64_t cap) {
    return (int64_t)((io_hash(key) >> 6) & (uint64_t)(cap - 1));
}

static io_entry *io_find(io_shard *sh, int64_t fd) {
    if (!sh->cap) return NULL;
    int64_t key = fd + 1;
    for (int64_t i = io_slot(key, sh->cap);; i = (i + 1) & (sh->cap - 1)) {
        if (sh->table[i].key == key) return &sh->table[i];
        if (!sh->table[i].key) return NULL;
    }
}

static io_entry *io_insert_key(io_entry *table, int64_t cap, int64_t key) {
    int64_t i = io_slot(key, cap);
    while (table[i].key && table[i].key != key) i = (i + 1) & (cap - 1);
    table[i].key = key;
    return &table[i];
}

/* takes sh's lock, with room in its table for one more descriptor: a
 * bigger table is allocated with the lock given back */
static void io_lock_room(io_shard *sh) {
    spin_lock(&sh->lock);
    while ((sh->used + 1) * 2 > sh->cap) {
        int64_t cap = sh->cap ? sh->cap * 2 : 16;
        spin_unlock(&sh->lock);
        io_entry *grown = veles_alloc_words(cap * (int64_t)sizeof *grown);
        spin_lock(&sh->lock);
        if (sh->cap >= cap) continue; /* another thread grew it meanwhile */
        for (int64_t i = 0; i < sh->cap; i++) {
            if (!sh->table[i].key) continue;
            io_entry *n = io_insert_key(grown, cap, sh->table[i].key);
            n->readers = sh->table[i].readers;
            n->writers = sh->table[i].writers;
        }
        sh->table = grown;
        sh->cap = cap;
    }
}

static io_entry *io_find_or_add(io_shard *sh, int64_t fd) {
    io_entry *e = io_find(sh, fd);
    if (e) return e;
    sh->used++;
    return io_insert_key(sh->table, sh->cap, fd + 1);
}

/* an entry with no task left leaves the table: the rest of its probe run
 * moves up, so a lookup never stops at a hole */
static void io_drop(io_shard *sh, io_entry *e) {
    int64_t i = e - sh->table;
    sh->table[i] = (io_entry){0};
    sh->used--;
    for (int64_t j = (i + 1) & (sh->cap - 1); sh->table[j].key; j = (j + 1) & (sh->cap - 1)) {
        io_entry moved = sh->table[j];
        sh->table[j] = (io_entry){0};
        io_entry *n = io_insert_key(sh->table, sh->cap, moved.key);
        n->readers = moved.readers;
        n->writers = moved.writers;
    }
}

static void io_unlink(io_entry *e, veles_task *t) {
    veles_task **head = t->io_write ? &e->writers : &e->readers;
    if (t->io_prev) t->io_prev->io_next = t->io_next; else *head = t->io_next;
    if (t->io_next) t->io_next->io_prev = t->io_prev;
    t->io_next = t->io_prev = NULL;
    __atomic_sub_fetch(&io_count, 1, __ATOMIC_RELAXED);
}

/* every task of a list that has left the table is ready: its retry runs
 * (shard lock held; they are woken from k once it is given back) */
static void io_ready_list(veles_task *t, wakes *k) {
    while (t) {
        veles_task *next = t->io_next;
        t->io_next = t->io_prev = NULL;
        __atomic_sub_fetch(&io_count, 1, __ATOMIC_RELAXED);
        __atomic_store_n(&t->io_ready, 1, __ATOMIC_SEQ_CST); /* io_waiting stays set, so the retry sees ready */
        wakes_add(k, t);
        t = next;
    }
}

/* arms e's descriptor for the tasks still on it; one the reactor refuses
 * is ready (its retried call says what is wrong) */
static void io_rearm(io_shard *sh, io_entry *e, wakes *k) {
    if (!e->readers && !e->writers) {
        io_drop(sh, e);
        return;
    }
    if (veles_poll_arm(e->key - 1, e->readers != NULL, e->writers != NULL) == 0) return;
    veles_task *r = e->readers, *w = e->writers;
    io_drop(sh, e);
    io_ready_list(r, k);
    io_ready_list(w, k);
}

/* the task stops waiting for its socket (cancelled, or leaving a scope) */
static void remove_io_waiter(veles_task *t) {
    if (!__atomic_load_n(&t->io_waiting, __ATOMIC_ACQUIRE)) return;
    io_shard *sh = io_shard_of(t->io_fd);
    spin_lock(&sh->lock);
    if (!__atomic_load_n(&t->io_ready, __ATOMIC_ACQUIRE)) {
        io_entry *e = io_find(sh, t->io_fd);
        if (e) {
            io_unlink(e, t);
            /* the descriptor may stay armed for this task: a later report
             * finds nobody, or wakes the others once too often - harmless,
             * every wait is a retry loop */
            if (!e->readers && !e->writers) io_drop(sh, e);
        }
    }
    __atomic_store_n(&t->io_waiting, 0, __ATOMIC_RELEASE);
    __atomic_store_n(&t->io_ready, 0, __ATOMIC_RELEASE);
    spin_unlock(&sh->lock);
}

static int is_closing(io_shard *sh, int64_t fd) {
    for (int64_t i = 0; i < sh->nclosing; i++) {
        if (sh->closing[i] == fd) return 1;
    }
    return 0;
}

/* wait for fd: true once the reactor saw it ready; the first call parks
 * the task, a wake for any other reason parks it again. A closed socket's
 * -1, a descriptor being closed, or one the reactor cannot watch is
 * "ready" at once: the retry fails or blocks (a regular file). */
static int64_t veles_task_wait_io_impl(veles_task *self, int64_t fd, int64_t write) {
    if (__atomic_load_n(&self->io_waiting, __ATOMIC_ACQUIRE)) {
        if (__atomic_load_n(&self->io_ready, __ATOMIC_ACQUIRE)) {
            __atomic_store_n(&self->io_waiting, 0, __ATOMIC_RELEASE);
            __atomic_store_n(&self->io_ready, 0, __ATOMIC_RELEASE);
            return 1;
        }
        /* woken for another reason: the reactor's wake, if it comes
         * meanwhile, finds the task running and queues it again */
        self->state = T_BLOCKED;
        return 0;
    }
    if (fd < 0) return 1;
    io_shard *sh = io_shard_of(fd);
    io_lock_room(sh);
    if (is_closing(sh, fd)) {
        spin_unlock(&sh->lock);
        return 1;
    }
    io_entry *e = io_find_or_add(sh, fd);
    veles_task **head = write ? &e->writers : &e->readers;
    int was_armed = *head != NULL;
    self->io_fd = fd;
    self->io_write = write;
    __atomic_store_n(&self->io_ready, 0, __ATOMIC_RELAXED);
    __atomic_store_n(&self->io_waiting, 1, __ATOMIC_RELEASE);
    self->io_prev = NULL;
    self->io_next = *head;
    if (*head) (*head)->io_prev = self;
    *head = self;
    __atomic_add_fetch(&io_count, 1, __ATOMIC_RELAXED);
    /* a second task the same way changes nothing the reactor knows */
    if (!was_armed && veles_poll_arm(fd, e->readers != NULL, e->writers != NULL) != 0) {
        io_unlink(e, self);
        if (!e->readers && !e->writers) io_drop(sh, e);
        __atomic_store_n(&self->io_waiting, 0, __ATOMIC_RELEASE);
        spin_unlock(&sh->lock);
        return 1;
    }
    self->state = T_BLOCKED;
    spin_unlock(&sh->lock);
    /* the default pool waits in the reactor: with none of its threads
     * there, one goes (D143) */
    if (!__atomic_load_n(&poller_busy, __ATOMIC_SEQ_CST) && !on_default_pool()) kick_housekeeper();
    return 0;
}

/* wakes the tasks of each descriptor the reactor reported, and arms it
 * again for the ones waiting the other way */
static void io_dispatch(veles_poll_event *ev, int64_t n) {
    wakes k = {0};
    for (int64_t i = 0; i < n; i++) {
        io_shard *sh = io_shard_of(ev[i].fd);
        spin_lock(&sh->lock);
        io_entry *e = io_find(sh, ev[i].fd);
        if (e) { /* none: everyone stopped waiting meanwhile */
            veles_task *r = NULL, *w = NULL;
            if (ev[i].read) {
                r = e->readers;
                e->readers = NULL;
            }
            if (ev[i].write) {
                w = e->writers;
                e->writers = NULL;
            }
            io_rearm(sh, e, &k);
            io_ready_list(r, &k);
            io_ready_list(w, &k);
        }
        spin_unlock(&sh->lock);
    }
    wakes_run(&k);
}

#define POLL_BATCH 128

static int32_t poll_looking; /* a thread takes a quick look (atomic) */

/* A quick look at the sockets by a thread with tasks to run, so they do not
 * starve the sockets: none while a thread waits in the reactor (it wakes
 * their tasks), and one thread at a time. It takes no runtime lock; a
 * look made while a thread starts waiting in the reactor is harmless, as
 * every backend lets two threads wait at once (a descriptor reported to
 * both is armed one-shot: the second finds nobody, or wakes the tasks once
 * too often, which every wait tolerates). */
static void poll_io_quick(void) {
    if (!__atomic_load_n(&io_count, __ATOMIC_RELAXED) || __atomic_load_n(&poller_busy, __ATOMIC_RELAXED)) return;
    if (__atomic_exchange_n(&poll_looking, 1, __ATOMIC_ACQUIRE)) return;
    veles_poll_event ev[POLL_BATCH];
    int64_t n = veles_poll_wait(0, ev, POLL_BATCH);
    __atomic_store_n(&last_poll, now_ns(), __ATOMIC_RELAXED);
    __atomic_store_n(&poll_looking, 0, __ATOMIC_RELEASE);
    io_dispatch(ev, n);
}

/* An idle thread of the default pool that claimed the reactor (poller_busy
 * 0 → 1) waits there up to timeout_ns (-1: until interrupted) for socket
 * events, and wakes their tasks; it gives the claim back after. The wait
 * is a safe region, so the collector may run, and is interrupted by
 * veles_poll_wake when work arrives that no idle thread can take
 * (wake_worker), a timer is armed sooner than it waits for (timer_armed),
 * or the root finishes. */
static void poll_io(int64_t timeout_ns) {
    veles_poll_event ev[POLL_BATCH];
    __atomic_store_n(&poll_until, timeout_ns < 0 ? INT64_MAX : now_ns() + timeout_ns, __ATOMIC_SEQ_CST);
    __atomic_store_n(&poller_blocked, 1, __ATOMIC_SEQ_CST);
    /* work queued before the flag was seen: do not sleep through it */
    if (anything_queued(default_exec)) timeout_ns = 0;
    veles_enter_safe();
    int64_t n = veles_poll_wait(timeout_ns, ev, POLL_BATCH);
    __atomic_store_n(&last_poll, now_ns(), __ATOMIC_RELAXED);
    veles_leave_safe();
    __atomic_store_n(&poller_blocked, 0, __ATOMIC_SEQ_CST);
    io_dispatch(ev, n);
}

/* ---- race (D38) ------------------------------------------------------------
 * A race waits on several things at once: it registers a node on each
 * channel arm (under that channel's lock), itself as the waiter of each
 * awaited task, and one timer for the earliest deadline. Whatever becomes
 * ready first claims the winner by compare-and-swap (race_claim) and fills
 * the arm, and only the claimer does. The race's own entry point runs
 * with the runtime lock held (timers, awaited tasks) and takes each
 * channel's lock in turn — the lock order everywhere. */

/* a race of narms arms (the compiler counts them) */
veles_race *veles_race_new(veles_task *t, int64_t narms) {
    veles_race *r = veles_alloc_words((int64_t)(sizeof *r + (size_t)narms * sizeof r->arms[0]));
    r->task = t;
    r->winner = -1;
    return r;
}

void veles_race_recv(veles_race *r, veles_chan *c, void *out) {
    int64_t i = r->narms++;
    r->arms[i].ch = c;
    r->arms[i].out = out;
}

/* a send arm (D108): in holds the value, evaluated once when the race
 * started; it enters the channel only if this arm wins */
void veles_race_send(veles_race *r, veles_chan *c, void *in) {
    int64_t i = r->narms++;
    r->arms[i].ch = c;
    r->arms[i].send = 1;
    r->arms[i].in = in;
}

void veles_race_sleep(veles_race *r, int64_t ns) {
    int64_t i = r->narms++;
    r->arms[i].deadline = now_ns() + ns;
    if (ns <= 0) r->arms[i].deadline = 1;
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
            remove_waiter(r->arms[i].send ? &c->send_waiters : &c->recv_waiters, &r->arms[i].node);
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

/* receive arm i (its channel's lock held): a buffered value, a waiting
 * sender's value or the close, claimed for the race. A waiting sender that
 * is another race's send arm is taken only with both races claimed. */
static bool race_try_recv(veles_race *r, int64_t i, wakes *k) {
    veles_chan *c = r->arms[i].ch;
    void *out = r->arms[i].out;
    if (c->len > 0) {
        if (!race_claim(r, i)) return false;
        chan_take(c, out, k);
        return true;
    }
    for (veles_waiter *s = c->send_waiters.head; s; s = s->next) {
        if (being_cancelled(s->task) || s->race == r) continue;
        if (!s->race) {
            if (!race_claim(r, i)) return false;
            remove_waiter(&c->send_waiters, s);
            s->task->chan_wait = NULL;
            memcpy(out, s->data, (size_t)c->elem);
            chan_complete(s->task, k);
            chan_sent(c, k);
            return true;
        }
        int got = race_pair(r, i, s->race, s->arm);
        if (got == PAIR_SELF_LOST) return false;
        if (got == PAIR_WON) {
            remove_waiter(&c->send_waiters, s);
            memcpy(out, s->data, (size_t)c->elem);
            race_ready(s->race, k);
            chan_sent(c, k);
            return true;
        }
        /* that race won elsewhere: its task takes the node off */
    }
    if (c->closed && race_claim(r, i)) {
        r->closed = 1;
        return true;
    }
    return false;
}

/* send arm i (its channel's lock held): a waiting receiver or buffer room
 * takes the value, or the channel is closed — claimed for the race */
static bool race_try_send(veles_race *r, int64_t i, wakes *k) {
    veles_chan *c = r->arms[i].ch;
    void *in = r->arms[i].in;
    if (c->closed) {
        if (!race_claim(r, i)) return false;
        r->closed = 1;
        return true;
    }
    for (veles_waiter *w = c->recv_waiters.head; w; w = w->next) {
        if (being_cancelled(w->task) || w->race == r) continue;
        if (!w->race) {
            if (!race_claim(r, i)) return false;
            remove_waiter(&c->recv_waiters, w);
            w->task->chan_wait = NULL;
            memcpy(w->data, in, (size_t)c->elem);
            chan_complete(w->task, k);
            chan_sent(c, k);
            return true;
        }
        int got = race_pair(r, i, w->race, w->arm);
        if (got == PAIR_SELF_LOST) return false;
        if (got == PAIR_WON) {
            remove_waiter(&c->recv_waiters, w);
            memcpy(w->race->arms[w->arm].out, in, (size_t)c->elem);
            race_ready(w->race, k);
            chan_sent(c, k);
            return true;
        }
    }
    if (c->len < c->cap) {
        if (!race_claim(r, i)) return false;
        chan_push(c, in);
        chan_sent(c, k);
        return true;
    }
    return false;
}

/* channel arm i, under its lock: ready and claimed for the race — or
 * nothing ready, and the node registered when register_node */
static bool race_try_chan(veles_race *r, int64_t i, int register_node, wakes *k) {
    veles_chan *c = r->arms[i].ch;
    spin_lock(&c->lock);
    bool won = r->arms[i].send ? race_try_send(r, i, k) : race_try_recv(r, i, k);
    if (!won && register_node) {
        veles_waiter *w = &r->arms[i].node;
        w->task = r->task;
        w->race = r;
        w->arm = i;
        w->data = r->arms[i].send ? r->arms[i].in : NULL;
        push_waiter(r->arms[i].send ? &c->send_waiters : &c->recv_waiters, w);
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
    int64_t now = now_ns();
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
            /* a fail-fast child that failed is never ready: its error is
             * the scope's, which abandons this race (D141) */
            if (__atomic_load_n(&a->state, __ATOMIC_SEQ_CST) == T_DONE && !done_and_failed(a) && race_claim(r, i)) won = i;
        }
        if (won < 0 && __atomic_load_n(&r->winner, __ATOMIC_SEQ_CST) >= 0) break; /* claimed on an arm already registered */
    }
    if (won >= 0) {
        race_detach(self, r);
        wakes_run(&k);
        return won;
    }
    if (earliest) {
        timer_at(self, earliest);
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
void veles_cleanup_push(veles_cleanup *c, void (*fn)(void *), void *env, void *(*move)(void *)) {
    if (!current) return;
    c->fn = fn;
    c->env = env;
    c->move = move;
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
/* ---- the call chain (D81) ----------------------------------------------
 * A debug build brackets every call of a Veles function with
 * veles_call_push(record) and veles_call_pop, the record being the constant
 * "site\0callee\0" the compiler wrote for that call. The chain is the
 * task's own — a suspended task keeps it, whichever thread resumes it — and
 * outside any task this thread's. A panic copies it as text; the release
 * build makes no calls and keeps nothing. */
#define SHADOW_MAX 4096 /* deeper than this the chain is not kept */

static int chain_recorded; /* a debug build ran: there is a chain to show */

/* t's debug record, made on first use when create (the task is the only
 * one to touch it: its own thread, or the creator before it starts) */
static task_debug *task_debug_of(veles_task *t, int create) {
    if (!t->debug && create) t->debug = veles_alloc_words(sizeof(task_debug));
    return t->debug;
}

/* the chain of t, or of this thread outside any task; NULL for a task
 * that has made no call a debug build records */
static veles_shadow *shadow_of(veles_task *t, int create) {
    if (!t) return &veles_tls_get()->shadow;
    task_debug *d = task_debug_of(t, create);
    return d ? &d->shadow : NULL;
}

static veles_shadow *shadow_now(void) {
    return shadow_of(current, 0);
}

static void shadow_push(veles_shadow *s, int collected, const char *rec) {
    if (s->depth < SHADOW_MAX) {
        if (s->depth == s->cap) {
            int64_t cap = s->cap ? s->cap * 2 : 16;
            /* a task's chain lives in collected memory, a thread's outside it */
            const char **at = collected ? veles_alloc_words(cap * (int64_t)sizeof *at) : malloc((size_t)cap * sizeof *at);
            if (!at) return;
            if (s->depth) memcpy(at, s->at, (size_t)s->depth * sizeof *at);
            if (!collected) free(s->at);
            s->at = at;
            s->cap = cap;
        }
        s->at[s->depth] = rec;
    }
    s->depth++;
}

void veles_call_push(const char *rec) {
    veles_task *t = current;
    shadow_push(shadow_of(t, 1), t != NULL, rec);
}

/* the task started for a suspending call continues the chain of the one that awaits it */
void veles_call_link(veles_task *t) {
    task_debug_of(t, 1)->chain_parent = current;
}

void veles_call_pop(void) {
    veles_shadow *s = shadow_now();
    if (s && s->depth > 0) s->depth--;
}

/* a task's or the program's first frame: the function it runs, with no site */
void veles_call_base(veles_task *t, const char *rec) {
    if (!chain_recorded) chain_recorded = 1;
    if (t && t->debug && t->debug->chain_parent) return; /* a suspending call: the caller's frame names it */
    veles_shadow *s = shadow_of(t, 1);
    s->depth = 0;
    shadow_push(s, t != NULL, rec);
}

static const char *frame_name(const char *rec) {
    return rec + strlen(rec) + 1;
}

/* the panic's function and the "site in caller" lines above it: the
 * frames of t (or of this thread, outside any task), then those of the
 * task that awaits it through a suspending call, and so on */
static void trace_capture(panic_trace *tr, veles_task *t, int collected) {
    tr->in = NULL;
    tr->chain = NULL;
    tr->chain_len = 0;
    int64_t n = 0, cap = 64, size = 0;
    const char **fr = malloc((size_t)cap * sizeof *fr);
    if (!fr) return;
    veles_task *owner = t;
    veles_shadow *s = shadow_of(t, 0);
    for (;;) {
        if (s && s->depth > SHADOW_MAX) { n = 0; break; }
        for (int64_t i = s ? s->depth - 1 : -1; i >= 0; i--) {
            if (n == cap) {
                cap *= 2;
                const char **grown = realloc(fr, (size_t)cap * sizeof *fr);
                if (!grown) { free(fr); return; }
                fr = grown;
            }
            fr[n++] = s->at[i];
        }
        owner = owner && owner->debug ? owner->debug->chain_parent : NULL;
        if (!owner) break;
        s = shadow_of(owner, 0);
    }
    if (n > 0) {
        tr->in = frame_name(fr[0]);
        for (int64_t i = 0; i + 1 < n; i++)
            size += (int64_t)strlen(fr[i]) + 4 + (int64_t)strlen(frame_name(fr[i + 1])) + 1;
    }
    char *out = size > 0 ? (collected ? veles_alloc(size) : malloc((size_t)size)) : NULL;
    if (out) {
        int64_t at = 0;
        for (int64_t i = 0; i + 1 < n; i++) {
            const char *site = fr[i], *caller = frame_name(fr[i + 1]);
            if (at) out[at++] = '\n';
            memcpy(out + at, site, strlen(site));
            at += (int64_t)strlen(site);
            memcpy(out + at, " in ", 4);
            at += 4;
            memcpy(out + at, caller, strlen(caller));
            at += (int64_t)strlen(caller);
        }
        tr->chain = out;
        tr->chain_len = at;
    }
    free(fr);
}

/* a helper's call site is in the chain already when a "site in caller" line
 * starts with it */
static int in_chain(const panic_trace *tr, const char *site, int64_t len) {
    for (int64_t i = 0; i + len + 4 <= tr->chain_len; i++) {
        if (i > 0 && tr->chain[i - 1] != '\n') continue;
        if (memcmp(tr->chain + i, site, (size_t)len) == 0 && memcmp(tr->chain + i + len, " in ", 4) == 0) return 1;
    }
    return 0;
}

/* The lines a report shows under the panic's message, each led by a newline
 * and indent spaces: "at LOC in FN", a "called from" line per call of the
 * chain, then the test helpers' call sites the chain does not have. */
static const char *trace_report(const panic_trace *tr, const char *loc, int64_t loc_len, int64_t indent, int collected, int64_t *len) {
    int64_t in_len = tr->in ? (int64_t)strlen(tr->in) : 0;
    int64_t lines = 1 + tr->chain_len / 4 + tr->sites_len / 4 + 2;
    int64_t size = loc_len + in_len + tr->chain_len + tr->sites_len + lines * (1 + indent + 16) + 16;
    char *out = collected ? veles_alloc(size) : malloc((size_t)size);
    if (!out) { *len = 0; return ""; }
    int64_t at = 0;
#define LINE(label) do { out[at++] = '\n'; memset(out + at, ' ', (size_t)indent); at += indent; \
    memcpy(out + at, label, strlen(label)); at += (int64_t)strlen(label); } while (0)
#define PUT(p, n) do { memcpy(out + at, (p), (size_t)(n)); at += (n); } while (0)
    /* a `@caller_location` function reported the site it was called from (D88):
     * that site is a line of the chain, below the frames of the marked
     * functions themselves, so the report says "at SITE in CALLER" and goes on
     * with the lines under it */
    int64_t first = 0, skip = 0;
    int at_site = 0;
    if (loc_len > 0) {
        for (int64_t start = 0; start < tr->chain_len; ) {
            int64_t end = start;
            while (end < tr->chain_len && tr->chain[end] != '\n') end++;
            if (end - start >= loc_len + 4 && memcmp(tr->chain + start, loc, (size_t)loc_len) == 0 &&
                memcmp(tr->chain + start + loc_len, " in ", 4) == 0) {
                at_site = 1;
                first = start;
                skip = end < tr->chain_len ? end + 1 : end;
                break;
            }
            start = end + 1;
        }
    }
    if (at_site) {
        int64_t end = first;
        while (end < tr->chain_len && tr->chain[end] != '\n') end++;
        LINE("at ");
        PUT(tr->chain + first, end - first);
    } else if (loc_len > 0 || tr->in) {
        LINE(loc_len > 0 ? "at " : "in ");
        if (loc_len > 0) {
            PUT(loc, loc_len);
            if (tr->in) PUT(" in ", 4);
        }
        if (tr->in) PUT(tr->in, in_len);
    }
    for (int64_t i = skip, from = skip; i <= tr->chain_len && tr->chain_len > skip; i++) {
        if (i < tr->chain_len && tr->chain[i] != '\n') continue;
        LINE("called from ");
        PUT(tr->chain + from, i - from);
        from = i + 1;
    }
    for (int64_t i = 0, from = 0; i <= tr->sites_len && tr->sites_len > 0; i++) {
        if (i < tr->sites_len && tr->sites[i] != '\n') continue;
        if (!in_chain(tr, tr->sites + from, i - from)) {
            LINE("called from ");
            PUT(tr->sites + from, i - from);
        }
        from = i + 1;
    }
#undef LINE
#undef PUT
    if (at == 0 && !collected) {
        free(out);
        *len = 0;
        return "";
    }
    *len = at;
    return out;
}

/* ---- unwinding a task from where it is (D20, D49, D145) ---------------
 * A panic, and a cancellation seen where the code cannot suspend — a
 * loop's back edge in a function that does not suspend, `checkCancelled`
 * — unwind the whole task from wherever it is: every cleanup registered
 * (the `with` closes, the cancellation of each scope's children) runs,
 * innermost first, and the thread jumps back to the executor. A scope
 * whose children are still unwinding keeps the task from ending (D3): it
 * is T_ENDING, with no frame, until the last of them has finished. */

/* the end of an unwound task (runtime lock held): cancelled, or failed
 * with its panic */
static void end_unwound(veles_task *t) {
    t->hdl = NULL;
    if (t->panic->cancelled) {
        veles_task_finish_cancelled_impl(t);
        return;
    }
    t->panicked = 1;
    t->failed = 1;
    scope_child_failed(t); /* before the result is published (D141) */
    __atomic_store_n(&t->state, T_DONE, __ATOMIC_SEQ_CST);
    wake(__atomic_exchange_n(&t->waiter, NULL, __ATOMIC_SEQ_CST));
    scope_child_finished(t);
}

/* one of the things an unwound task waits for — its own cleanups, then
 * each draining scope — is over; the last ends it */
static void end_drained(veles_task *t) {
    if (__atomic_sub_fetch(&t->panic->drain, 1, __ATOMIC_SEQ_CST) != 0) return;
    rt_enter();
    end_unwound(t);
    rt_exit();
    if (t->test_task) test_task_done(t);
}

/* runs t's cleanups, innermost first; it ends now, or once its draining
 * scopes have. A cleanup that panics continues the unwinding itself. */
static void unwind_cleanups(veles_task *t) {
    while (t->cleanups) {
        veles_cleanup *c = t->cleanups;
        t->cleanups = c->next;
        c->fn(c->env);
    }
    rt_enter();
    t->hdl = NULL;
    __atomic_store_n(&t->state, T_ENDING, __ATOMIC_SEQ_CST);
    rt_exit();
    end_drained(t);
}

/* a cancellation seen in code that cannot suspend (D145) */
static void unwind_cancelled(veles_task *t) {
    rt_unwind_to(0);
    t->unwinding = 1;
    task_panic *pn = veles_alloc_words(sizeof *pn);
    pn->drain = 1;
    pn->cancelled = 1;
    t->panic = pn;
    unwind_cleanups(t);
    longjmp(panic_return, 1);
}

int64_t veles_task_panic(const char *msg, int64_t len, const char *loc, int64_t loc_len) {
    if (!in_resume || !current) return 0;
    veles_task *t = current;
    /* the cleanups are Veles code: they run without the runtime lock the
     * panicking runtime call may have held */
    rt_unwind_to(0);
    /* the first panic is the one reported; a panic in a cleanup run for a
     * cancellation makes the task end with that panic */
    if (!t->unwinding || (t->panic && t->panic->cancelled)) {
        t->unwinding = 1;
        char *copy = veles_alloc(len + 1);
        memcpy(copy, msg, (size_t)len);
        task_panic *pn = t->panic;
        if (!pn) {
            pn = veles_alloc_words(sizeof *pn);
            pn->drain = 1;
            t->panic = pn;
        }
        pn->cancelled = 0;
        t->result = veles_alloc_words(8);
        pn->msg = copy;
        pn->len = len;
        char *where = veles_alloc(loc_len + 1);
        if (loc_len > 0) memcpy(where, loc, (size_t)loc_len);
        pn->loc = where;
        pn->loc_len = loc_len;
        /* the call chain: the panic's own (debug build), or the one a scope
         * re-raises from the child that panicked (D81) */
        veles_tls *tls = veles_tls_get();
        veles_task *from = tls->carry;
        tls->carry = NULL;
        if (from) {
            pn->trace.in = from->panic->trace.in;
            pn->trace.chain = from->panic->trace.chain;
            pn->trace.chain_len = from->panic->trace.chain_len;
        } else {
            trace_capture(&pn->trace, t, 1);
        }
        /* inside a test helper: where the test called it (D78) — read now,
         * while the bindings are the panicking code's */
        int64_t n = 0;
        for (veles_local *l = *current_locals(); l; l = l->next)
            if (l->key == TEST_SITE_KEY) n += ((test_site *)l->cell)->len + 1;
        if (n > 0) {
            char *sites = veles_alloc(n);
            int64_t at = 0;
            for (veles_local *l = *current_locals(); l; l = l->next) {
                if (l->key != TEST_SITE_KEY) continue;
                test_site *s = l->cell;
                if (at > 0) sites[at++] = '\n';
                memcpy(sites + at, s->where, (size_t)s->len);
                at += s->len;
            }
            pn->trace.sites = sites;
            pn->trace.sites_len = at;
        }
    }
    unwind_cleanups(t);
    longjmp(panic_return, 1);
    return 1;
}

extern volatile int32_t veles_stop_requested;
void veles_gc_park(void);
int64_t veles_task_cancelled(veles_task *t, int64_t depth);

/* A loop's back edge in a suspending function found veles_attention set
 * (D145): 0 goes on (a scope body checks whether a child failed, as after a
 * suspension point), 1 yields (the frame suspends with the task at the back
 * of the run queue), 2 unwinds (the frame takes its cancellation path), 3
 * goes on without a look — in a lock region or a close() (shield), or
 * while unwinding. */
int64_t veles_backedge(veles_task *t, int64_t depth) {
    if (veles_stop_requested) veles_gc_park();
    veles_tls *tls = veles_tls_get();
    if (tls->shield > 0 || tls->callback_depth > 0 || t->unwinding || t->shield) return 3;
    if (veles_task_cancelled(t, depth)) return 2;
    veles_worker *w = tls->worker;
    if (w && __atomic_exchange_n(&w->preempt, 0, __ATOMIC_SEQ_CST)) {
        __atomic_sub_fetch(&veles_attention, 1, __ATOMIC_SEQ_CST);
        return 1;
    }
    return 0;
}

/* the cancellation check of code that cannot suspend: a loop's back edge
 * in a plain function, and `checkCancelled()` (D145) */
static void check_cancelled_here(void) {
    veles_tls *tls = veles_tls_get();
    if (!in_resume || !current || tls->shield > 0 || tls->callback_depth > 0) return;
    veles_task *t = current;
    if (t->unwinding || t->shield) return;
    /* a plain function cannot give its thread up: a request to is dropped,
     * and the monitor asks again if it still runs */
    veles_worker *w = tls->worker;
    if (w && __atomic_load_n(&w->preempt, __ATOMIC_RELAXED) && __atomic_exchange_n(&w->preempt, 0, __ATOMIC_SEQ_CST))
        __atomic_sub_fetch(&veles_attention, 1, __ATOMIC_SEQ_CST);
    if (__atomic_load_n(&t->cancel_requested, __ATOMIC_ACQUIRE) ||
        __atomic_load_n(&t->state, __ATOMIC_ACQUIRE) == T_CANCELLED)
        unwind_cancelled(t);
}

void veles_backedge_plain(void) {
    if (veles_stop_requested) veles_gc_park();
    check_cancelled_here();
}

void veles_check_cancelled(void) {
    check_cancelled_here();
}

/* around a `with`'s close (codegen runClose): shielded, as a lock region is */
void veles_shield_enter(void) {
    veles_tls_get()->shield++;
}

void veles_shield_leave(void) {
    veles_tls *tls = veles_tls_get();
    if (tls->shield > 0) tls->shield--;
}

/* around a close() that suspends (D147): the shield is the task's, since
 * the close may resume on another thread. A cancellation requested inside
 * is seen at the first suspension point after it. */
void veles_task_shield_enter(veles_task *t) {
    __atomic_add_fetch(&t->shield, 1, __ATOMIC_SEQ_CST);
}

void veles_task_shield_leave(veles_task *t) {
    if (__atomic_load_n(&t->shield, __ATOMIC_ACQUIRE)) __atomic_sub_fetch(&t->shield, 1, __ATOMIC_SEQ_CST);
}

/* A `with` whose close() suspends, reached while the running task unwinds
 * without its frame (a panic, or a cancellation in code that cannot
 * suspend, D145): the close cannot run on this frame, so it runs as a task
 * of its own — `closer`, which the codegen thunk then spawns with the
 * resource — and that task takes the cleanups still to run, which it runs
 * when the close is over, so everything still closes innermost first. The
 * unwinding task waits for it as for a draining scope (T_ENDING); a closer
 * that meets another such close hands on to the next, and the unwinding
 * task waits for each (D147). */
void veles_cleanup_continue(veles_task *closer) {
    veles_task *t = current;
    veles_task *root = t;
    if (t->scope && t->scope->closing) root = t->scope->owner;
    veles_scope *s = veles_scope_begin(root, 0, 0);
    s->closing = 1;
    s->draining = 1;
    __atomic_add_fetch(&root->panic->drain, 1, __ATOMIC_SEQ_CST);
    closer->scope = s;
    closer->locals = t->locals; /* a test's record, the task-locals of the code that opened it */
    /* the entries and what they close are in frames and on a stack this
     * unwinding is about to leave: the closer gets copies in the heap */
    veles_cleanup **tail = &closer->cleanups;
    for (veles_cleanup *c = t->cleanups; c; c = c->next) {
        veles_cleanup *m = veles_alloc_words(sizeof *m);
        m->fn = c->fn;
        m->move = c->move;
        m->env = c->move ? c->move(c->env) : c->env;
        *tail = m;
        tail = &m->next;
    }
    t->cleanups = NULL;
}

/* a closer's close() is over: the cleanups it took from the unwinding task
 * run now, innermost first (one may hand on to another closer) */
static void run_inherited_cleanups(veles_task *t) {
    while (t->cleanups) {
        veles_cleanup *c = t->cleanups;
        t->cleanups = c->next;
        c->fn(c->env);
    }
}

int64_t veles_task_panicked(veles_task *t) {
    return t->panicked;
}

/* The message of the panic the running task is unwinding by, for a close()
 * that records it (Lazy, D146): 1 and the text in *out, or 0 when it is not
 * unwinding by a panic (it returned normally, or a cancellation unwinds it). */
typedef struct {
    const char *data;
    int64_t len;
} panic_text;

int64_t veles_panic_current(panic_text *out) {
    if (!in_resume || !current) return 0;
    task_panic *pn = current->panic;
    if (!current->unwinding || !pn || pn->cancelled || !pn->msg) return 0;
    out->data = pn->msg;
    out->len = pn->len;
    return 1;
}

const char *veles_task_panic_msg(veles_task *t, int64_t *len) {
    *len = t->panic ? t->panic->len : 0;
    return t->panic ? t->panic->msg : "";
}

const char *veles_task_panic_loc(veles_task *t, int64_t *len) {
    *len = t->panic ? t->panic->loc_len : 0;
    return t->panic && t->panic->loc ? t->panic->loc : "";
}

/* the lines under a panicked task's message (see trace_report) */
const char *veles_task_panic_report(veles_task *t, int64_t indent, int64_t *len) {
    if (!t->panic) {
        *len = 0;
        return "";
    }
    return trace_report(&t->panic->trace, t->panic->loc, t->panic->loc_len, indent, 1, len);
}

/* the same for a panic that ends the process, written to standard error:
 * this thread's own chain, or the one a scope re-raises */
int veles_chain_recorded(void) {
    return chain_recorded;
}

/* For the stack-overflow report (veles_stack.c), which runs in a fault
 * handler: reads, allocates nothing. Stores the innermost `max` records of
 * the running task's (or thread's) chain that were kept, innermost first;
 * `depth` is how many calls are in progress and `kept` how many of them
 * the chain holds (the first SHADOW_MAX). Returns how many it stored. */
int64_t veles_shadow_peek(const char **frames, int64_t max, int64_t *depth, int64_t *kept) {
    veles_shadow *s = shadow_now();
    if (!s) {
        *depth = *kept = 0;
        return 0;
    }
    int64_t held = s->depth < SHADOW_MAX ? s->depth : SHADOW_MAX;
    *depth = s->depth;
    *kept = held;
    int64_t n = 0;
    while (n < max && n < held) {
        frames[n] = s->at[held - 1 - n];
        n++;
    }
    return n;
}

void veles_panic_print_report(const char *loc, int64_t loc_len, int64_t indent) {
    veles_tls *tls = veles_tls_get();
    panic_trace tr = {0};
    int carried = tls->carry != NULL;
    if (carried) {
        tr.in = tls->carry->panic->trace.in;
        tr.chain = tls->carry->panic->trace.chain;
        tr.chain_len = tls->carry->panic->trace.chain_len;
        tls->carry = NULL;
    } else {
        trace_capture(&tr, current, 0);
    }
    int64_t len = 0;
    const char *report = trace_report(&tr, loc, loc_len, indent, 0, &len);
    if (len > 0) {
        fwrite(report, 1, (size_t)len, stderr);
        free((void *)report);
    }
    if (!carried && tr.chain) free((void *)tr.chain);
}

/* resumes t's innermost frame; while the frame resumed returns to its
 * caller (a suspending call finishing, or unwinding), the caller resumes
 * next, until a frame parks or the task's own frame finishes */
static void resume(veles_task *t) {
    if (!t->hdl) return;
    current = t;
    in_resume = 1;
    if (setjmp(panic_return) == 0) {
        do {
            void *hdl = t->hdl;
            t->popped = 0;
            void (*fn)(void *) = *(void (**)(void *))hdl;
            fn(hdl);
        } while (t->popped && t->hdl);
    } else {
        rt_unwind_to(0);
    }
    in_resume = 0;
    current = NULL;
}

static int64_t nearest_timer(void) {
    int64_t nearest = 0;
    for (int i = 0; i < TIMER_SHARDS; i++) {
        int64_t first = __atomic_load_n(&timer_shards[i].first, __ATOMIC_ACQUIRE);
        if (first && (!nearest || first < nearest)) nearest = first;
    }
    int64_t tick = nearest_tick();
    if (tick && (!nearest || tick < nearest)) nearest = tick;
    return nearest;
}

/* the root has finished (read once: veles_run clears it as it returns) */
static int root_done(void) {
    veles_task *r = __atomic_load_n(&root_task, __ATOMIC_ACQUIRE);
    if (!r) return 0;
    int32_t st = __atomic_load_n(&r->state, __ATOMIC_ACQUIRE);
    return st == T_DONE || st == T_CANCELLED;
}

/* no root to run (between runs), or it has finished */
static int root_finished(void) {
    return !__atomic_load_n(&root_task, __ATOMIC_ACQUIRE) || root_done();
}

static void scope_child_finished(veles_task *t);

/* A spawned task cancelled before it ran: it never starts, and finishes
 * as cancelled (its scope stops waiting for it, an await on it panics). */
static void finish_unstarted(veles_task *t) {
    t->entry = NULL;
    t->entry_args = NULL;
    __atomic_store_n(&t->state, T_CANCELLED, __ATOMIC_SEQ_CST);
    wake(__atomic_exchange_n(&t->waiter, NULL, __ATOMIC_SEQ_CST));
    scope_child_finished(t);
}

/* a run queue of e at slot i (runtime lock held) */
static veles_worker *new_worker(veles_exec *e, int64_t i) {
    veles_worker *w = veles_alloc_words(sizeof *w);
    w->exec = e;
    w->seed = (uint32_t)i * 2654435761u + 1;
    e->workers[i] = w;
    if (i + 1 > e->nworkers) __atomic_store_n(&e->nworkers, i + 1, __ATOMIC_RELEASE);
    return w;
}

/* the calling thread's run queue of the default pool (runtime lock held):
 * the one it has, one handed off by the monitor, or a new one while there
 * are fewer than VELES_THREADS — else NULL, and the thread waits as a
 * spare */
static veles_worker *join_workers(void) {
    veles_tls *tls = veles_tls_get();
    if (tls->worker) return tls->worker;
    veles_worker *free = take_free_worker();
    if (free) return free;
    veles_exec *e = default_exec;
    if (e->nworkers >= e->target) return NULL;
    veles_worker *w = new_worker(e, e->nworkers);
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

static void note_timer(int64_t at) {
    int64_t due = __atomic_load_n(&timer_due, __ATOMIC_RELAXED);
    while ((!due || at < due) && !__atomic_compare_exchange_n(&timer_due, &due, at, 1, __ATOMIC_RELAXED, __ATOMIC_RELAXED)) {}
}

static int timers_due(void) {
    int64_t due = __atomic_load_n(&timer_due, __ATOMIC_RELAXED);
    return due && now_ns() >= due;
}

/* after the due timers fired: the hint becomes the nearest deadline — unless
 * a timer armed meanwhile noted an earlier one, which stays */
static void refresh_timer_due(void) {
    int64_t old = __atomic_load_n(&timer_due, __ATOMIC_RELAXED);
    int64_t nearest = nearest_timer();
    __atomic_compare_exchange_n(&timer_due, &old, nearest, 0, __ATOMIC_RELAXED, __ATOMIC_RELAXED);
}

/* A worker out of work looks again for a little while (tens of
 * microseconds) before it goes to sleep, as Go's spinning threads do: a
 * server's tasks park and wake many thousand times a second, and putting a
 * thread to sleep on a condition variable and waking it costs more than
 * the task it was woken for — with every core a worker, more threads made
 * bench/httphello slower (2026-10-07). At most half the busy workers spin,
 * wake_worker wakes no sleeper while one does, and a spinner that finds a
 * task and was the last wakes the next if more are waiting. A default
 * thread stops early for what only its outer loop does: timers due,
 * sockets no thread waits for, the root finished. */
#define SPIN_ROUNDS 24

static veles_task *spin_for_task(veles_worker *w) {
    veles_exec *e = w->exec;
    int64_t n = __atomic_load_n(&e->nworkers, __ATOMIC_ACQUIRE);
    if (n < 2) return NULL;
    int64_t busy = n - __atomic_load_n(&e->idle_workers, __ATOMIC_RELAXED);
    if (2 * __atomic_load_n(&e->spinning, __ATOMIC_SEQ_CST) >= busy) return NULL;
    __atomic_add_fetch(&e->spinning, 1, __ATOMIC_SEQ_CST);
    int housekeeping = e == default_exec;
    veles_task *t = NULL;
    for (int round = 0; round < SPIN_ROUNDS && !t; round++) {
        for (int i = 0; i < 50; i++) cpu_pause();
        if (housekeeping) {
            if (timers_due()) break;
            if (__atomic_load_n(&io_count, __ATOMIC_RELAXED) && !__atomic_load_n(&poller_busy, __ATOMIC_RELAXED)) break;
            if (root_done()) break;
        } else if (__atomic_load_n(&e->stopping, __ATOMIC_RELAXED)) {
            break;
        }
        t = find_task(w);
    }
    int64_t left = __atomic_sub_fetch(&e->spinning, 1, __ATOMIC_SEQ_CST);
    if (t && left == 0 && anything_queued(e)) wake_worker(e);
    return t;
}

/* runs the task t a worker took, marking the run for the monitor (D145) */
static void run_counted(veles_worker *w, veles_task *t) {
    /* the monitor tells a task that runs long by run_seq staying the same
     * odd number; a request to yield was for the task before this one */
    if (__atomic_load_n(&monitor_asleep, __ATOMIC_RELAXED)) wake_monitor();
    __atomic_store_n(&w->run_seq, w->run_seq + 1, __ATOMIC_RELAXED);
    if (__atomic_exchange_n(&w->preempt, 0, __ATOMIC_SEQ_CST))
        __atomic_sub_fetch(&veles_attention, 1, __ATOMIC_SEQ_CST);
    run_task(t);
    __atomic_store_n(&w->run_seq, w->run_seq + 1, __ATOMIC_RELAXED);
}

/* The tasks a default thread runs between two looks at the timers and the
 * sockets: until none is left, the root finishes, a timer is due, or 64
 * have run while sockets wait for a look; 0 when it found none. */
static int64_t run_batch(veles_worker *w) {
    veles_tls *tls = veles_tls_get();
    veles_exec *e = w->exec;
    int64_t ran = 0;
    for (;;) {
        veles_task *t = find_task(w);
        if (!t) t = spin_for_task(w);
        if (!t) break;
        if (anything_queued(e)) wake_worker(e); /* more waiting: a hand for them */
        run_counted(w, t);
        ran++;
        if (tls->worker != w) break; /* handed off while t blocked */
        if (root_done()) break;
        if (timers_due()) break;
        if (ran % 64 == 0 && __atomic_load_n(&io_count, __ATOMIC_RELAXED) && !__atomic_load_n(&poller_busy, __ATOMIC_RELAXED)) break;
    }
    return ran;
}

/* An idle thread waits at most this long (ns) before it looks again: a
 * backstop — a wake it should have had costs a second, not a hang */
#define IDLE_BACKSTOP (1000 * NS_PER_MS)

/* the root has finished: every thread of the default pool hears it */
static void root_ended(void) {
    wake_all(default_exec);
    rt_enter();
    veles_cond_broadcast(spare_cv);
    rt_exit();
    if (__atomic_load_n(&poller_blocked, __ATOMIC_SEQ_CST)) veles_poll_wake();
}

/* every task is blocked, and nothing — a timer, a socket, a thread running
 * a task anywhere — can wake one: looked at again under the runtime lock,
 * so two idle threads do not both decide it */
static void check_deadlock(void) {
    rt_enter();
    if (!nearest_timer() && !__atomic_load_n(&io_count, __ATOMIC_SEQ_CST) && !__atomic_load_n(&poller_busy, __ATOMIC_SEQ_CST) &&
        __atomic_load_n(&active_workers, __ATOMIC_SEQ_CST) == 0 && !work_anywhere() && !root_finished())
        veles_panic("deadlock: every task is blocked", 31);
    rt_exit();
}

/* A thread of the default pool with nothing to run waits for a task, the
 * nearest timer or a socket: in the reactor, if no thread waits there; or
 * parked — until the nearest timer if no other idle thread keeps time,
 * else until it is woken. */
static void idle_default(veles_exec *e) {
    int64_t nearest = nearest_timer();
    int64_t wait = nearest ? ns_until(nearest) : -1;
    if (wait == 0) return;
    if (__atomic_load_n(&io_count, __ATOMIC_SEQ_CST)) {
        int64_t free_ = 0;
        if (__atomic_compare_exchange_n(&poller_busy, &free_, 1, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) {
            poll_io(wait < 0 || wait > IDLE_BACKSTOP ? IDLE_BACKSTOP : wait);
            __atomic_store_n(&poller_busy, 0, __ATOMIC_SEQ_CST);
            return;
        }
    }
    idle_node *me = my_idle();
    idle_push(e, me);
    /* a last look, now that a waker would find this thread listed */
    if (anything_queued(e) || root_finished() ||
        (__atomic_load_n(&io_count, __ATOMIC_SEQ_CST) && !__atomic_load_n(&poller_busy, __ATOMIC_SEQ_CST))) {
        if (!idle_leave(e, me)) __atomic_store_n(&e->waking, 0, __ATOMIC_SEQ_CST);
        return;
    }
    nearest = nearest_timer(); /* one armed meanwhile found no keeper */
    wait = nearest ? ns_until(nearest) : -1;
    /* every executor's threads count: a task waiting for one placed on a
     * pool is not stuck while that one runs or is queued (D143) */
    if (!nearest && !__atomic_load_n(&io_count, __ATOMIC_SEQ_CST) && !__atomic_load_n(&poller_busy, __ATOMIC_SEQ_CST) &&
        __atomic_load_n(&active_workers, __ATOMIC_SEQ_CST) == 0 && !work_anywhere())
        check_deadlock();
    int keeping = 0;
    if (nearest) {
        idle_node *none = NULL;
        if (__atomic_compare_exchange_n(&keeper, &none, me, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) {
            __atomic_store_n(&keeper_at, nearest, __ATOMIC_SEQ_CST);
            keeping = 1;
            wait = ns_until(nearest); /* a timer armed since the look above is in it, or woke the keeper */
        } else {
            wait = -1; /* another thread keeps time */
        }
    }
    if (wait != 0) {
        veles_enter_safe();
        veles_park_wait(me->park, wait < 0 || wait > IDLE_BACKSTOP ? IDLE_BACKSTOP : wait);
        veles_leave_safe();
    }
    if (keeping) {
        idle_node *mine = me;
        __atomic_compare_exchange_n(&keeper, &mine, NULL, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST);
    }
    /* taken off the list by a waker: back now, so another may be woken */
    if (!idle_leave(e, me)) __atomic_store_n(&e->waking, 0, __ATOMIC_SEQ_CST);
}

/* One thread's loop in the default pool. Tasks are taken and run without
 * the runtime lock, and so are the timers, the sockets and going idle; the
 * lock is for the queues handed off around blocking calls. (Taken between
 * every batch, and by every idle wait and every wake, it was the last lock
 * most threads waited for at 32 threads, 2026-10-10.) It runs until the
 * root has finished; stay (a pool thread) keeps it waiting for the next
 * root instead of returning. A thread without a run queue — there are
 * VELES_THREADS, and its own was handed off during a blocking call — waits
 * for one. */
static void work(int stay) {
    veles_tls *tls = veles_tls_get();
    veles_exec *e = default_exec;
    for (;;) {
        veles_worker *w = tls->worker;
        if (!w) {
            rt_enter();
            w = join_workers();
            if (!w) {
                int leave = root_finished() && !stay;
                if (!leave) {
                    spares_waiting++;
                    veles_enter_safe();
                    veles_cond_wait(spare_cv, rt_lock, 50);
                    veles_leave_safe();
                    spares_waiting--;
                }
                rt_exit();
                if (leave) return;
                continue;
            }
            rt_exit();
        }
        if (root_finished()) {
            if (!stay) return;
            /* between roots (veles test runs one per test): parked until
             * veles_run wakes every thread */
            idle_node *me = my_idle();
            idle_push(e, me);
            if (root_finished()) {
                veles_enter_safe();
                veles_park_wait(me->park, IDLE_BACKSTOP);
                veles_leave_safe();
            }
            if (!idle_leave(e, me)) __atomic_store_n(&e->waking, 0, __ATOMIC_SEQ_CST);
            continue;
        }
        /* active from the first look at the queues to the last task run:
         * a worker holding a task it took is not idle, and no sleeper may
         * call the program deadlocked meanwhile */
        __atomic_add_fetch(&active_workers, 1, __ATOMIC_SEQ_CST);
        int64_t ran;
        for (;;) {
            fire_timers();
            refresh_timer_due();
            /* runnable tasks must not starve the sockets: a quick look,
             * unless a thread is waiting in the reactor already */
            if (anything_queued(e)) poll_io_quick();
            ran = run_batch(w);
            if (!ran || tls->worker != w || root_finished()) break;
        }
        __atomic_sub_fetch(&active_workers, 1, __ATOMIC_SEQ_CST);
        if (ran) {
            if (root_finished()) root_ended();
            continue;
        }
        if (tls->worker == w) idle_default(e);
    }
}

/* a pool thread, or a spare started for a queue handed off */
static void worker_main(void *arg) {
    (void)arg;
    veles_thread_attach();
    work(1);
}

/* ---- the threads of the other executors (D143) -------------------------------
 * A thread of a pool, of `Executor.thread` or of the blocking pool runs
 * only its executor's tasks: no timers and no sockets (the default pool
 * serves those), and no hand-off of its queue around a blocking call. It
 * sleeps when out of work and leaves when the executor is closed; a thread
 * of the blocking pool also leaves after BLOCKING_KEEP_MS idle, and a new
 * one starts when a call finds every one busy, up to the pool's bound. */
#define BLOCKING_KEEP_MS 10000
#define BLOCKING_MAX 128

int64_t veles_thread_configure(const char *name, int64_t priority, const int32_t *cpus, int64_t ncpus, char *err, int64_t cap);
void veles_thread_detach(void);

typedef struct exec_start {
    veles_exec *e;
    int64_t slot;
    int64_t status;     /* 0 starting, 1 running, -1 refused (err says why) */
    int64_t detached;   /* nobody waits for the status: the thread frees this */
    char err[160];
} exec_start;

/* the loop of a thread of e, run without the runtime lock: returns when
 * the executor stops, or a blocking thread has idled long. fresh: a
 * blocking thread just started for a call (grow_blocking), which lets the
 * next be started once it is here. */
static void exec_loop(veles_worker *w, int fresh) {
    veles_exec *e = w->exec;
    if (fresh) __atomic_store_n(&e->waking, 0, __ATOMIC_SEQ_CST);
    for (;;) {
        __atomic_add_fetch(&active_workers, 1, __ATOMIC_SEQ_CST);
        for (;;) {
            veles_task *t = find_task(w);
            if (!t) t = spin_for_task(w);
            if (!t) break;
            if (anything_queued(e)) wake_worker(e); /* more waiting: a hand for them */
            run_counted(w, t);
        }
        __atomic_sub_fetch(&active_workers, 1, __ATOMIC_SEQ_CST);
        idle_node *me = my_idle();
        idle_push(e, me);
        if (anything_queued(e) || __atomic_load_n(&w->runnext, __ATOMIC_SEQ_CST) || __atomic_load_n(&e->stopping, __ATOMIC_SEQ_CST)) {
            if (!idle_leave(e, me)) __atomic_store_n(&e->waking, 0, __ATOMIC_SEQ_CST);
            if (__atomic_load_n(&e->stopping, __ATOMIC_SEQ_CST) && !anything_queued(e) && !w->runnext) return;
            continue;
        }
        int64_t since = now_ns();
        veles_enter_safe();
        veles_park_wait(me->park, e->kind == X_BLOCKING ? (int64_t)BLOCKING_KEEP_MS * NS_PER_MS : -1);
        veles_leave_safe();
        if (!idle_leave(e, me)) __atomic_store_n(&e->waking, 0, __ATOMIC_SEQ_CST);
        if (e->kind == X_BLOCKING && !anything_queued(e) && now_ns() - since >= (int64_t)BLOCKING_KEEP_MS * NS_PER_MS)
            return;
    }
}

static void exec_main(void *arg) {
    exec_start *s = arg;
    veles_exec *e = s->e;
    veles_thread_attach();
    char name[48];
    if (e->kind == X_THREAD) snprintf(name, sizeof name, "%s", e->name);
    else snprintf(name, sizeof name, "%s-%d", e->name, (int)s->slot);
    char err[160];
    int64_t rc = veles_thread_configure(name, e->priority, e->cpus, e->ncpus, err, sizeof err);
    rt_enter();
    if (rc != 0) {
        e->threads--;
        memcpy(s->err, err, sizeof err);
        __atomic_store_n(&s->status, -1, __ATOMIC_SEQ_CST);
        veles_cond_broadcast(e->stopped_cv);
        rt_exit();
        veles_thread_detach();
        return;
    }
    veles_worker *w = e->workers[s->slot];
    if (!w) w = new_worker(e, s->slot);
    w->bstate = 1; /* the slot is taken (bstate is otherwise unused here: no hand-off) */
    veles_tls_get()->worker = w;
    int64_t detached = s->detached;
    __atomic_store_n(&s->status, 1, __ATOMIC_SEQ_CST);
    veles_cond_broadcast(e->stopped_cv);
    if (detached) free(s);
    rt_exit();
    exec_loop(w, (int)detached);
    rt_enter();
    w->bstate = 0;
    veles_tls_get()->worker = NULL;
    e->threads--;
    veles_cond_broadcast(e->stopped_cv);
    rt_exit();
    idle_node_release();
    veles_thread_detach();
}

/* one more thread for the blocking pool, every one being busy: up to its
 * bound, one at a time — the one starting wakes the next if more calls
 * wait */
static void grow_blocking(veles_exec *e) {
    rt_enter();
    int64_t none = 0;
    if (e->stopping || __atomic_load_n(&e->idle_workers, __ATOMIC_SEQ_CST) > 0 || e->threads >= e->target ||
        !__atomic_compare_exchange_n(&e->waking, &none, 1, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) {
        rt_exit();
        return;
    }
    int64_t slot = -1;
    for (int64_t i = 0; i < e->target; i++) {
        if (!e->workers[i] || !e->workers[i]->bstate) {
            slot = i;
            break;
        }
    }
    exec_start *s = slot < 0 ? NULL : calloc(1, sizeof *s);
    if (!s) {
        __atomic_store_n(&e->waking, 0, __ATOMIC_SEQ_CST);
        rt_exit();
        return;
    }
    s->e = e;
    s->slot = slot;
    s->detached = 1;
    if (!e->workers[slot]) new_worker(e, slot);
    e->workers[slot]->bstate = 1;
    e->threads++;
    /* waking stays set until the thread is in its loop: no second one meanwhile */
    if (veles_thread_spawn(exec_main, s) != 0) {
        e->threads--;
        e->workers[slot]->bstate = 0;
        __atomic_store_n(&e->waking, 0, __ATOMIC_SEQ_CST);
        free(s);
    }
    rt_exit();
}

/* how many threads run tasks: VELES_THREADS, the program's `[runtime]
 * threads` (veles.toml, set before the first run, D143), or one per core */
static int64_t threads_configured;

void veles_runtime_threads(int64_t n) {
    threads_configured = n;
}

static int64_t worker_count(void) {
    const char *env = getenv("VELES_THREADS");
    int64_t n = env ? strtoll(env, NULL, 10) : threads_configured > 0 ? threads_configured : veles_cpu_count();
    if (n < 1) n = 1;
    if (n > MAX_WORKERS) n = MAX_WORKERS;
    return n;
}

/* veles_run drives the executor until the root task completes. The calling
 * thread is one of the workers; the others are started on the first run
 * and wait between runs (veles test runs one root per test). */
void veles_run(veles_task *root) {
    veles_task_init();
    rt_enter();
    int64_t st = __atomic_load_n(&root->state, __ATOMIC_SEQ_CST);
    if (st == T_DONE || st == T_CANCELLED) { /* a test that finished while an earlier one was awaited */
        rt_exit();
        return;
    }
    root_task = root;
    if (!workers_started) {
        workers_started = 1;
        default_exec->target = worker_count();
        for (int64_t i = 1; i < default_exec->target; i++) veles_thread_spawn(worker_main, NULL);
        monitor_lock = veles_lock_new();
        monitor_cv = veles_cond_new();
        veles_thread_spawn_small(monitor_main, NULL);
    }
    rt_exit();
    wake_all(default_exec); /* the threads parked between roots */
    work(0);
    rt_enter();
    root_task = NULL;
    rt_exit();
}

/* ---- Executor (D143) ------------------------------------------------------- */

/* how many threads the default pool has */
int64_t veles_exec_default_threads(void) {
    veles_task_init();
    return workers_started ? default_exec->target : worker_count();
}

static void exec_link(veles_exec *e) {
    e->next = execs;
    __atomic_store_n(&execs, e, __ATOMIC_RELEASE);
}

static void exec_unlink(veles_exec *e) {
    for (veles_exec **pp = &execs; *pp; pp = &(*pp)->next) {
        if (*pp == e) {
            *pp = e->next;
            break;
        }
    }
    e->next = NULL;
}

/* waits, the runtime lock held, until e's threads have left */
static void exec_join(veles_exec *e) {
    __atomic_store_n(&e->stopping, 1, __ATOMIC_SEQ_CST);
    wake_all(e);
    while (e->threads > 0) {
        veles_enter_safe();
        veles_cond_wait(e->stopped_cv, rt_lock, -1);
        veles_leave_safe();
    }
}

/* An executor of `threads` threads (`Executor(...)`; single: the one of
 * `Executor.thread`), named for the OS (name-0, name-1, … in a pool), each
 * started with priority (0 Low, 1 Normal, 2 High, 3 Realtime) and the CPUs
 * listed (none: any). NULL when a thread could not be started or the OS
 * refused its priority or CPUs: err then says why, and no thread is left
 * running. */
static veles_exec *exec_create(int64_t threads, int64_t single, const char *name, int64_t name_len,
                           int64_t priority, const int64_t *cpus, int64_t ncpus, char *err, int64_t cap) {
    veles_task_init();
    if (single || threads < 1) threads = 1;
    if (threads > MAX_WORKERS) threads = MAX_WORKERS;
    veles_exec *e = exec_alloc(single ? X_THREAD : X_POOL, threads);
    e->target = threads;
    if (name_len > (int64_t)sizeof e->name - 1) name_len = (int64_t)sizeof e->name - 1;
    if (name_len > 0) memcpy(e->name, name, (size_t)name_len);
    e->priority = (int32_t)priority;
    if (ncpus > 0) {
        e->cpus = veles_alloc((int64_t)ncpus * (int64_t)sizeof(int32_t));
        for (int64_t i = 0; i < ncpus; i++) {
            if (cpus[i] < 0 || cpus[i] > INT32_MAX) {
                snprintf(err, (size_t)cap, "there is no CPU %lld: CPUs are numbered from 0", (long long)cpus[i]);
                return NULL;
            }
            e->cpus[i] = (int32_t)cpus[i];
        }
        e->ncpus = (int32_t)ncpus;
    }
    exec_start *starts = calloc((size_t)threads, sizeof *starts);
    if (!starts) {
        snprintf(err, (size_t)cap, "out of memory");
        return NULL;
    }
    rt_enter();
    int failed = 0;
    for (int64_t i = 0; i < threads && !failed; i++) {
        starts[i].e = e;
        starts[i].slot = i;
        e->threads++;
        if (veles_thread_spawn(exec_main, &starts[i]) != 0) {
            e->threads--;
            snprintf(err, (size_t)cap, "the system could not start thread %lld of %lld", (long long)i + 1, (long long)threads);
            failed = 1;
            break;
        }
        /* each thread reports whether the OS took its settings before the
         * next starts: a refusal leaves no thread behind */
        while (__atomic_load_n(&starts[i].status, __ATOMIC_SEQ_CST) == 0) {
            veles_enter_safe();
            veles_cond_wait(e->stopped_cv, rt_lock, -1);
            veles_leave_safe();
        }
        if (starts[i].status < 0) {
            snprintf(err, (size_t)cap, "%s", starts[i].err);
            failed = 1;
        }
    }
    if (failed) {
        exec_join(e);
        rt_exit();
        free(starts);
        return NULL;
    }
    exec_link(e);
    rt_exit();
    free(starts);
    return e;
}

typedef struct {
    const char *data;
    int64_t len;
} exec_text;

static void set_error(exec_text *out, const char *msg) {
    int64_t n = (int64_t)strlen(msg);
    char *buf = veles_alloc(n + 1);
    memcpy(buf, msg, (size_t)n + 1);
    out->data = buf;
    out->len = n;
}

/* `Executor.pool(...)` and `Executor.thread(...)`: the executor, or NULL
 * with the reason in *err (a ThreadError) */
veles_exec *veles_exec_new(int64_t threads, int64_t single, const char *name, int64_t name_len,
                           int64_t priority, const int64_t *cpus, int64_t ncpus, exec_text *err) {
    char why[200] = {0};
    veles_exec *e = exec_create(threads, single, name, name_len, priority, cpus, ncpus, why, sizeof why);
    if (!e) set_error(err, why);
    return e;
}

/* ---- Thread (D143) -----------------------------------------------------------
 * `Thread.start(...)` runs a plain function on an OS thread of its own. The
 * thread runs it as a task outside any executor — current, so a panic in
 * it unwinds as one does, running its cleanups, and is kept for the join —
 * and the `with` holding the Thread joins it when its block ends, blocking
 * the joining thread (a blocking call: the default pool hands its queue
 * on). The record is reachable from os_threads until it is joined, which
 * keeps the function alive while only C holds it. */
typedef struct os_thread {
    veles_task *task;           /* where a panic in the body is kept; its locals are the starter's (D72) */
    void *body;                 /* the function's closure pair: { code, env } — a `fun()` is called as code(env) */
    int64_t status;             /* 0 starting, 1 running, -1 refused, 2 finished */
    int64_t joined;
    int32_t priority;
    int32_t ncpus;
    int32_t *cpus;
    char name[32];
    char err[200];
    struct os_thread *next;     /* os_threads: not joined yet */
} os_thread;

static os_thread *os_threads; /* GC root (the runtime lock's) */
static veles_cond *os_thread_cv;

int64_t veles_thread_spawn_sized(void (*fn)(void *), void *arg, int64_t size);
void veles_task_repanic(veles_task *t);

static void os_thread_main(void *arg) {
    os_thread *r = arg;
    veles_thread_attach();
    char err[200];
    int64_t rc = veles_thread_configure(r->name, r->priority, r->cpus, r->ncpus, err, sizeof err);
    rt_enter();
    if (rc != 0) {
        memcpy(r->err, err, sizeof err);
        __atomic_store_n(&r->status, -1, __ATOMIC_SEQ_CST);
        veles_cond_broadcast(os_thread_cv);
        rt_exit();
        veles_thread_detach();
        return;
    }
    __atomic_store_n(&r->status, 1, __ATOMIC_SEQ_CST);
    veles_cond_broadcast(os_thread_cv);
    rt_exit();
    veles_task *t = r->task;
    void **pair = r->body;
    void (*code)(void *) = (void (*)(void *))pair[0];
    void *env = pair[1];
    current = t;
    in_resume = 1;
    /* running, it may yet wake a task: no idle thread calls the program
     * deadlocked meanwhile */
    __atomic_add_fetch(&active_workers, 1, __ATOMIC_SEQ_CST);
    if (setjmp(panic_return) == 0) {
        code(env);
    } else {
        rt_unwind_to(0);
        veles_tls_get()->shield = 0;
    }
    __atomic_sub_fetch(&active_workers, 1, __ATOMIC_SEQ_CST);
    in_resume = 0;
    current = NULL;
    rt_enter();
    __atomic_store_n(&r->status, 2, __ATOMIC_SEQ_CST);
    veles_cond_broadcast(os_thread_cv);
    rt_exit();
    veles_thread_detach();
}

/* starts the thread, or returns NULL with why in *err: the system could
 * not start one, or refused its priority or CPUs */
os_thread *veles_thread_start(const char *name, int64_t name_len, int64_t priority, const int64_t *cpus, int64_t ncpus,
                              int64_t stack_size, void *body, exec_text *err) {
    veles_task_init();
    os_thread *r = veles_alloc_words(sizeof *r);
    r->task = veles_task_new();
    r->body = body;
    r->priority = (int32_t)priority;
    if (name_len > (int64_t)sizeof r->name - 1) name_len = (int64_t)sizeof r->name - 1;
    if (name_len > 0) memcpy(r->name, name, (size_t)name_len);
    if (ncpus > 0) {
        r->cpus = veles_alloc(ncpus * (int64_t)sizeof(int32_t));
        for (int64_t i = 0; i < ncpus; i++) {
            if (cpus[i] < 0 || cpus[i] > INT32_MAX) {
                char why[96];
                snprintf(why, sizeof why, "there is no CPU %lld: CPUs are numbered from 0", (long long)cpus[i]);
                set_error(err, why);
                return NULL;
            }
            r->cpus[i] = (int32_t)cpus[i];
        }
        r->ncpus = (int32_t)ncpus;
    }
    rt_enter();
    if (!os_thread_cv) {
        os_thread_cv = veles_cond_new();
        veles_gc_root(&os_threads, NULL);
    }
    r->next = os_threads;
    os_threads = r;
    int failed = veles_thread_spawn_sized(os_thread_main, r, stack_size) != 0;
    if (failed) {
        snprintf(r->err, sizeof r->err, "the system could not start a thread%s", stack_size > 0 ? " with that stack size" : "");
    } else {
        while (__atomic_load_n(&r->status, __ATOMIC_SEQ_CST) == 0) {
            veles_enter_safe();
            veles_cond_wait(os_thread_cv, rt_lock, -1);
            veles_leave_safe();
        }
        failed = r->status < 0;
    }
    if (failed) {
        for (os_thread **pp = &os_threads; *pp; pp = &(*pp)->next) {
            if (*pp == r) {
                *pp = r->next;
                break;
            }
        }
    }
    rt_exit();
    if (failed) {
        set_error(err, r->err);
        return NULL;
    }
    return r;
}

/* the `with` holding the Thread ends: waits for the thread to finish — a
 * blocking call — and re-raises a panic of its body here. Joining again
 * (a copy) does nothing. */
void veles_thread_join(os_thread *r) {
    veles_blocking_enter();
    rt_enter();
    while (__atomic_load_n(&r->status, __ATOMIC_SEQ_CST) != 2)
        veles_cond_wait(os_thread_cv, rt_lock, -1);
    int first = !r->joined;
    r->joined = 1;
    for (os_thread **pp = &os_threads; *pp; pp = &(*pp)->next) {
        if (*pp == r) {
            *pp = r->next;
            break;
        }
    }
    rt_exit();
    veles_blocking_leave();
    veles_task *t = r->task;
    if (first && t->panic && !t->panic->cancelled && t->panic->msg) veles_task_repanic(t);
}

/* `close()` of an Executor: its tasks have all been joined by the scopes
 * that placed them (D143); its threads leave, and the call returns once
 * they have. Closing it again — through a copy — does nothing. */
void veles_exec_close(veles_exec *e) {
    veles_worker *w = veles_tls_get()->worker;
    if (w && w->exec == e) veles_panic("an executor cannot be closed by one of its own tasks", 52);
    rt_enter();
    if (!e->stopping) {
        exec_join(e);
        exec_unlink(e);
    }
    rt_exit();
}

int64_t veles_exec_closed(veles_exec *e) {
    return __atomic_load_n(&e->stopping, __ATOMIC_ACQUIRE) != 0;
}

/* the executor the running task is on; the default pool outside a task */
veles_exec *veles_exec_current(void) {
    veles_task_init();
    veles_task *t = current;
    return t ? exec_of(t) : default_exec;
}

/* `scope(on: e)`, `gather(on: e)`: the children launched in the scope s
 * run on e. A closed executor runs nothing: the placement panics. */
void veles_scope_on(veles_scope *s, veles_exec *e) {
    if (__atomic_load_n(&e->stopping, __ATOMIC_ACQUIRE)) veles_panic("the executor was closed: nothing more runs on it", 48);
    s->exec = e;
}

/* the blocking pool (`blocking(f)`), made when the program starts (a
 * prelude global); its threads start as calls need them. It is not on the
 * list of executors: the monitor has nothing to look at in it — its
 * calls are plain code, which cannot yield — and leaving the list empty
 * keeps the monitor off the runtime lock in a program with no executor
 * of its own. */
veles_exec *veles_exec_blocking(void) {
    veles_task_init();
    veles_exec *e = __atomic_load_n(&blocking_exec, __ATOMIC_ACQUIRE);
    if (e) return e;
    rt_enter();
    if (!blocking_exec) {
        e = exec_alloc(X_BLOCKING, BLOCKING_MAX);
        e->target = BLOCKING_MAX;
        e->priority = 1;
        memcpy(e->name, "blocking", 9);
        __atomic_store_n(&blocking_exec, e, __ATOMIC_RELEASE);
    }
    rt_exit();
    return blocking_exec;
}

int64_t veles_race_closed(veles_race *r) {
    return r->closed;
}

/* re-raise a child's panic in the current context (scope re-raises, D52) */
void veles_task_repanic(veles_task *t) {
    veles_tls_get()->carry = t;
    veles_panic_at(t->panic->msg, t->panic->len, t->panic->loc, t->panic->loc_len);
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
    int64_t saved_shield = veles_tls_get()->shield;
    jmp_buf saved;
    memcpy(&saved, &panic_return, sizeof saved);
    current = t;
    in_resume = 1;
    if (setjmp(panic_return) == 0) {
        entry(t, args);
    } else {
        rt_unwind_to(saved_depth);
        veles_tls_get()->shield = saved_shield; /* a lock region or close() the jump left (D145) */
    }
    memcpy(&panic_return, &saved, sizeof saved);
    current = saved_current;
    in_resume = saved_in;
}

/* the root task (main): queued, not run here, so that veles_run has the
 * workers and the monitor going before any of it runs — run inline, its
 * code up to its first suspension ran with no other thread to start the
 * children it launched (F9: a child launched before main first waited
 * never started while main computed) */
void veles_task_start(veles_task *t, void (*entry)(veles_task *, void *), void *args) {
    veles_task_init();
    t->entry = entry;
    t->entry_args = args;
    t->state = T_RUNNABLE;
    __atomic_store_n(&t->sched, S_QUEUED, __ATOMIC_SEQ_CST);
    push_global(default_exec, t);
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

/* A task returning. The common case — a result, not an error — takes no
 * lock: the result is published with T_DONE, then the waiter is read (an
 * await publishes itself as the waiter, then reads the state: one of the
 * two sees the other), and waking is safe without the runtime lock, as the
 * channels' is. A wake the other side also saw for itself is a spurious
 * one, which every wait tolerates. The scope's owner is woken when this was
 * its last child. An error may fail the scope and cancel siblings, which
 * is done under the lock. (The lock here, in await and in the scope wait
 * was most of the executor's contention: bench/httphello, 2026-10-07.) */
static void arena_done(veles_task *t);

void veles_task_finish(veles_task *t, const void *result, int64_t size, int64_t failed) {
    /* only a closer has cleanups left when its frame returns (D147) */
    if (t->cleanups) run_inherited_cleanups(t);
    arena_done(t);
    if (failed || __atomic_load_n(&t->state, __ATOMIC_SEQ_CST) == T_CANCELLED) {
        rt_enter();
        veles_task_finish_impl(t, result, size, failed);
        rt_exit();
        if (t->test_task) test_task_done(t);
        return;
    }
    void *cell = veles_alloc_words(size > 0 ? size : 8);
    if (size > 0 && result) memcpy(cell, result, (size_t)size);
    t->result = cell;
    t->failed = 0;
    __atomic_store_n(&t->state, T_DONE, __ATOMIC_SEQ_CST);
    t->hdl = NULL;
    veles_task *waiter = __atomic_exchange_n(&t->waiter, NULL, __ATOMIC_SEQ_CST);
    int64_t left = t->scope ? scope_leave(t) : 1;
    if (waiter) wake(waiter);
    if (left <= 0) scope_empty(t->scope);
    if (t->test_task) test_task_done(t);
}

void veles_task_finish_cancelled(veles_task *t);
void veles_task_leave_waits(veles_task *t);

/* checked at every suspension point by the frame at depth, so it takes no
 * lock: both flags only ever go from 0 to 1, and a request that lands just
 * after the check is seen at the next one, as it would be with the lock.
 * A frame deeper than a scope body a failed child abandons is cancelled
 * too: it leaves whatever it waited on and unwinds, back to that body. */
int64_t veles_task_cancelled(veles_task *t, int64_t depth) {
    /* a send or recv the other side completed has happened: the retry
     * returns it, and the request is seen at the next suspension point */
    if (__atomic_load_n(&t->chan_done, __ATOMIC_ACQUIRE)) return 0;
    /* inside a close() that suspends: shielded (D47, D147) */
    if (__atomic_load_n(&t->shield, __ATOMIC_ACQUIRE)) return 0;
    if (__atomic_load_n(&t->cancel_requested, __ATOMIC_ACQUIRE) ||
        __atomic_load_n(&t->state, __ATOMIC_ACQUIRE) == T_CANCELLED) {
        /* cancelled while it ran or was queued: what it registered since
         * is left here, before it unwinds (cancel_task leaves a parked
         * task's waits itself) */
        if (t->race || t->chan_wait || t->timer_slot || t->io_waiting) veles_task_leave_waits(t);
        return 1;
    }
    if (depth > __atomic_load_n(&t->abandon_depth, __ATOMIC_ACQUIRE)) {
        veles_task_leave_waits(t);
        return 1;
    }
    return 0;
}

/* ---- calls of suspending functions (review F3) -------------------------
 * A suspending call runs the callee's frame in the caller's task: the
 * caller passes a link in its own frame (its handle, the callee's depth,
 * a done flag, then room for the result). A callee that finishes without
 * waiting has written the result and set done before its ramp returns, and
 * the caller goes straight on: no task, no queue. One that waits parks the
 * task on its own frame; when it finishes it hands the task back to its
 * caller, which the executor resumes next (resume, below). A task's own
 * frame has the root link, whose parent is NULL. */
typedef struct frame_link {
    void *parent;     /* the caller's frame; NULL for a task's own */
    int64_t depth;
    int64_t done;
} frame_link;

/* The frames of a task's calls are strictly last in, first out — a call
 * finishes before its caller does — so they come from a bump arena of the
 * task's instead of one collected object each: a chunk of collected
 * memory (scanned word by word like a frame, and kept alive by the task
 * and by the frames' handles into it). A task's own frame is an ordinary
 * object: it lives as long as the task, however long that parks. */
typedef struct frame_arena {
    struct frame_arena *prev;  /* the chunk before this one filled up */
    int64_t top, cap;          /* bytes in use; bytes in data */
    char data[];
} frame_arena;

/* a chunk is an object of the collector's 2048-byte class with its header
 * word: 2072 bytes, as it once was, took a 4096-byte slot */
#define ARENA_CHUNK (2048 - 8 - (int64_t)sizeof(frame_arena))

/* A task that parks in its own frame has no call in progress, so its chunk
 * goes back to the thread, which hands it to the next call that needs one
 * — a task's next call after each wait took a new chunk, and those were
 * two thirds of what a server allocated (bench/httphello, 2026-10-10). The
 * thread block keeps the few it holds alive; a chunk is zeroed as it is
 * given back, so stale frames in it keep nothing alive. */
static void arena_give_back(frame_arena *a) {
    veles_tls *tls = veles_tls_get();
    if (a->cap != ARENA_CHUNK || tls->narenas == (int64_t)(sizeof tls->arenas / sizeof tls->arenas[0])) return;
    memset(a->data, 0, (size_t)a->cap);
    a->top = 0;
    a->prev = NULL;
    tls->arenas[tls->narenas++] = a;
}

/* the chunks of t above keep (newest first) hold no frame any more: back
 * to the thread they go, and keep is t's chunk again */
static void arena_unwind_to(veles_task *t, frame_arena *keep) {
    frame_arena *a = t->arena;
    while (a && a != keep) {
        frame_arena *prev = a->prev;
        arena_give_back(a);
        a = prev;
    }
    t->arena = keep;
}

/* t has no call in progress — it parks in its own frame, or ends (most
 * tasks end without parking there) — so its chunks go back */
static void arena_done(veles_task *t) {
    if (t->arena) arena_unwind_to(t, NULL);
}

static frame_arena *arena_take(void) {
    veles_tls *tls = veles_tls_get();
    if (tls->narenas == 0) return NULL;
    frame_arena *a = tls->arenas[--tls->narenas];
    tls->arenas[tls->narenas] = NULL;
    return a;
}

/* the code the compiler emits takes the common paths of veles_frame_alloc,
 * veles_frame_free and veles_frame_back itself (codegen/llvm/coro.go,
 * frameHelpers), reading and writing these fields: they stay put */
_Static_assert(offsetof(veles_task, hdl) == 0, "codegen/llvm/coro.go frameHelpers");
_Static_assert(offsetof(veles_task, arena) == 24, "codegen/llvm/coro.go frameHelpers");
_Static_assert(offsetof(veles_task, depth) == 224 && sizeof(((veles_task *)0)->depth) == 4, "codegen/llvm/coro.go frameHelpers");
_Static_assert(offsetof(veles_task, popped) == 246, "codegen/llvm/coro.go frameHelpers");
_Static_assert(sizeof(veles_task) == 248, "review B4: a parked task stays small");
_Static_assert(offsetof(frame_arena, top) == 8 && offsetof(frame_arena, cap) == 16 && offsetof(frame_arena, data) == 24, "codegen/llvm/coro.go frameHelpers");

/* storage for the frame of a call (link has a parent) or of a task (none) */
void *veles_frame_alloc(veles_task *t, frame_link *l, int64_t size) {
    if (!l->parent) return veles_alloc_words(size);
    size = (size + 15) & ~(int64_t)15;
    frame_arena *a = t->arena;
    if (!a || a->top + size > a->cap) {
        frame_arena *n = size <= ARENA_CHUNK ? arena_take() : NULL;
        if (!n) {
            int64_t cap = size > ARENA_CHUNK / 2 ? size * 2 : ARENA_CHUNK;
            n = veles_alloc_words((int64_t)sizeof *n + cap);
            n->cap = cap;
        }
        n->prev = a;
        t->arena = a = n;
    }
    void *p = a->data + a->top;
    a->top += size;
    return p;
}

/* a call's frame is over: it and everything above it in the arena go
 * (a frame its caller never freed — the caller unwound for a cancellation
 * — is above it). A frame of a chunk before the current one empties the
 * newer chunks, which go back to the thread. */
void veles_frame_free(veles_task *t, void *hdl) {
    char *p = hdl;
    frame_arena *a = t->arena;
    while (a && !(p >= a->data && p < a->data + a->cap)) a = a->prev;
    if (!a) return; /* not an arena frame */
    a->top = p - a->data;
    if (a != t->arena) arena_unwind_to(t, a); /* the call chain had grown into newer chunks */
}

/* the frame at depth suspends: the task resumes there. A task parked in
 * its own frame has no call in progress, so it keeps no arena — an idle
 * task stays small — and its chunk goes back to this thread. */
void veles_frame_park(veles_task *t, void *hdl, int64_t depth) {
    t->hdl = hdl;
    t->depth = (int32_t)depth;
    t->popped = 0;
    if (depth == 0) arena_done(t);
}

/* a call's frame has finished (its result written, or unwound by a
 * cancellation): the task goes back to the caller's frame */
static void frame_return(veles_task *t, frame_link *l) {
    l->done = 1;
    t->hdl = l->parent;
    t->depth = (int32_t)(l->depth - 1);
    t->popped = 1;
}

/* a call's frame has written its result into the link and returns */
void veles_frame_back(veles_task *t, frame_link *l) {
    frame_return(t, l);
}

/* the frame returns: a task's own frame finishes the task; a call's
 * copies its result (size bytes at value) into the link at out */
void veles_frame_return(veles_task *t, frame_link *l, const void *value, int64_t size, int64_t failed, void *out) {
    if (!l->parent) {
        veles_task_finish(t, value, size, failed);
        return;
    }
    if (size > 0 && value) memmove(out, value, (size_t)size);
    frame_return(t, l);
}

/* the frame has unwound for a cancellation: a task's own frame finishes
 * the task as cancelled; a call's returns, marked unwound (done 2), and its
 * caller unwinds next — also when the caller goes on at once, which it
 * does when the call unwound at a loop's back edge without having
 * suspended: the link holds no result for it to read (codegen callFrame) */
void veles_frame_unwound(veles_task *t, frame_link *l) {
    if (!l->parent) {
        veles_task_finish_cancelled(t);
        return;
    }
    frame_return(t, l);
    l->done = 2;
}

void veles_task_finish_cancelled(veles_task *t) {
    arena_done(t);
    rt_enter();
    veles_task_finish_cancelled_impl(t);
    rt_exit();
}

/* no lock: the waiter handshake with veles_task_finish needs none (see
 * there) */
int64_t veles_task_await(veles_task *self, veles_task *target) {
    if (__atomic_load_n(&target->state, __ATOMIC_ACQUIRE) == T_DONE && !done_and_failed(target)) {
        return 1;
    }
    return veles_task_await_impl(self, target);
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
    /* nothing left to cancel — the common way a scope is left early, a
     * withTimeout returning its value — takes no lock: only the owner,
     * which is here, launches into the scope, and live only falls (F9) */
    if (__atomic_load_n(&s->live, __ATOMIC_SEQ_CST) <= 0 && !s->owner->unwinding) return;
    rt_enter();
    veles_scope_cancel_impl(s);
    rt_exit();
}

void veles_scope_abandon(veles_scope *s, veles_task *owner) {
    rt_enter();
    veles_scope_abandon_impl(s, owner);
    rt_exit();
}

/* no lock: the live count is atomic, and the last child decrements it
 * before it wakes the owner, which reads it before it parks — the await
 * handshake again */
int64_t veles_scope_wait(veles_task *owner, veles_scope *s) {
    return veles_scope_wait_impl(owner, s);
}

/* no lock: failed is set once, under the lock, and read as a published
 * pointer */
veles_task *veles_scope_failed(veles_scope *s) {
    return veles_scope_failed_impl(s);
}

int64_t veles_scope_failed_index(veles_scope *s) {
    return veles_scope_failed_index_impl(s);
}

/* The channel operations take the channel's own lock, never the runtime
 * lock, and wake the tasks they completed once it is released. A task
 * still listed on another channel — woken for another reason, then sent
 * here — is unlinked there first, so no two channel locks are ever held. */
static void closed_send(void) {
    veles_panic("send on a closed channel", 24);
}

/* a send arm won (D108): its value is in the channel, or the channel was
 * closed, which panics as `send` does */
void veles_race_sent(veles_race *r) {
    if (r->closed) closed_send();
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

/* no runtime lock: the timer is its shard's (F9) */
int64_t veles_task_sleep(veles_task *self, int64_t ns) {
    return veles_task_sleep_impl(self, ns);
}

/* no runtime lock: the descriptor's shard has a lock of its own */
int64_t veles_task_wait_io(veles_task *self, int64_t fd, int64_t write) {
    return veles_task_wait_io_impl(self, fd, write);
}

/* ---- closing sockets (std/net's Socket) --------------------------------------
 * std/net closes a socket's descriptor only when no operation holds it any
 * more (Go's fdMutex), and the operations holding it must stop waiting:
 * veles_io_closing wakes every task parked on the descriptor, as if the
 * poll had reported it ready, and until veles_io_closed no task parks on
 * it again — the wait returns at once. Each retries, finds the socket
 * closing, fails and lets go; the last to let go closes the descriptor.
 * Both under the descriptor's shard lock, which a task registering its
 * wait holds too: an operation that checked "not closing" just before the
 * close cannot then park unseen. */
void veles_io_closing(int64_t fd) {
    veles_task_init();
    io_shard *sh = io_shard_of(fd);
    wakes k = {0};
    spin_lock(&sh->lock);
    if (!is_closing(sh, fd)) {
        if (sh->nclosing == sh->closing_cap) {
            int64_t cap = sh->closing_cap ? sh->closing_cap * 2 : 4;
            int64_t *grown = realloc(sh->closing, (size_t)cap * sizeof *grown);
            if (!grown) {
                spin_unlock(&sh->lock);
                veles_panic("out of memory", 13);
            }
            sh->closing = grown;
            sh->closing_cap = cap;
        }
        sh->closing[sh->nclosing++] = fd;
    }
    io_entry *e = io_find(sh, fd);
    if (e) {
        veles_task *r = e->readers, *w = e->writers;
        io_drop(sh, e);
        io_ready_list(r, &k);
        io_ready_list(w, &k);
    }
    spin_unlock(&sh->lock);
    wakes_run(&k);
}

/* the descriptor is closed: its number may belong to a new socket now */
void veles_io_closed(int64_t fd) {
    io_shard *sh = io_shard_of(fd);
    spin_lock(&sh->lock);
    for (int64_t i = 0; i < sh->nclosing; i++) {
        if (sh->closing[i] == fd) {
            sh->closing[i] = sh->closing[--sh->nclosing];
            break;
        }
    }
    spin_unlock(&sh->lock);
}

/* no runtime lock (F9): the race's channels have locks of their own, its
 * timer is its shard's, an awaited task is joined by the waiter handshake
 * (veles_task_await), and only this task's own thread registers or leaves
 * its waits while it runs (cancel_task takes apart only a parked task's) */
int64_t veles_race_wait(veles_task *self, veles_race *r) {
    return veles_race_wait_impl(self, r);
}
