/*
 * Veles tasks (build plan Stage 4; spec D2, D3, D16, D34, D36, D38).
 *
 * Suspending functions are LLVM switched-resume coroutines whose frames
 * live on the GC heap. This file is the single-threaded executor that
 * resumes them: a run queue, blocking on channels / tasks / timers /
 * scopes, and fail-fast scopes with cancellation. Every object here is
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

enum { T_RUNNABLE, T_BLOCKED, T_DONE, T_CANCELLED };

typedef struct veles_scope veles_scope;
typedef struct veles_race veles_race;

typedef struct veles_task {
    void *hdl;              /* coroutine frame; NULL after completion */
    int64_t state;
    struct veles_task *waiter; /* task blocked in await on this one */
    void *result;           /* heap cell holding the return value */
    int64_t failed;         /* result is Err */
    veles_scope *scope;
    int64_t index;          /* launch index within its scope */
    struct veles_task *next;      /* run queue / wait list link */
    struct veles_task *sibling;   /* scope children list */
    int64_t queued;
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
    veles_task *children;
    int64_t live;
    int64_t fail_fast;
    veles_task *failed;
    int64_t launches;
};

typedef struct veles_chan {
    char *buf;
    int64_t cap, len, head, elem;
    veles_desc *desc;
    int64_t closed;
    int64_t remaining; /* closeAfter: sends left before the channel closes itself; -1 = never */
    veles_task *recv_waiters;
    veles_task *send_waiters;
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
    } arms[16];
    int64_t narms;
};

static veles_task *run_head, *run_tail;
static veles_task *timers;
static veles_task *io_waiters;
static veles_task *current;
static int64_t start_ms;
static bool roots_registered;

static void register_roots(void) {
    if (roots_registered) return;
    roots_registered = true;
    veles_gc_root(&run_head, NULL);
    veles_gc_root(&run_tail, NULL);
    veles_gc_root(&timers, NULL);
    veles_gc_root(&io_waiters, NULL);
    veles_gc_root(&current, NULL);
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

static void sleep_ms(int64_t ms) {
    if (ms <= 0) return;
#if defined(_WIN32)
    Sleep((DWORD)ms);
#else
    usleep((useconds_t)(ms * 1000));
#endif
}

/* ---- run queue ------------------------------------------------------------ */

static void enqueue(veles_task *t) {
    if (t->queued || t->state == T_DONE || t->state == T_CANCELLED) return;
    t->state = T_RUNNABLE;
    t->queued = 1;
    t->next = NULL;
    if (run_tail) run_tail->next = t; else run_head = t;
    run_tail = t;
}

static veles_task *dequeue(void) {
    veles_task *t = run_head;
    if (!t) return NULL;
    run_head = t->next;
    if (!run_head) run_tail = NULL;
    t->next = NULL;
    t->queued = 0;
    return t;
}

static void wake(veles_task *t) {
    if (t && t->state == T_BLOCKED) enqueue(t);
}

/* ---- tasks ----------------------------------------------------------------- */

veles_task *veles_task_new(void) {
    register_roots();
    veles_task *t = veles_alloc_words(sizeof *t);
    t->state = T_RUNNABLE;
    return t;
}

veles_task *veles_task_current(void) {
    return current;
}

void veles_task_set_current(veles_task *t) {
    current = t;
}

/* veles_task_started records the coroutine handle once the ramp returned. */
void veles_task_started(veles_task *t, void *hdl) {
    if (t->state != T_DONE && t->state != T_CANCELLED) t->hdl = hdl;
}

static void scope_child_finished(veles_task *t);
static void remove_timer(veles_task *t);

/* called by the coroutine body before its final suspend */
void veles_task_finish(veles_task *t, const void *result, int64_t size, int64_t failed) {
    if (t->state == T_CANCELLED) return;
    t->result = veles_alloc_words(size > 0 ? size : 8);
    if (size > 0 && result) memcpy(t->result, result, (size_t)size);
    t->failed = failed;
    t->state = T_DONE;
    t->hdl = NULL;
    wake(t->waiter);
    t->waiter = NULL;
    scope_child_finished(t);
}

/* a resumed task checks this to honour cancellation (D20: delivered at a
 * suspension point) */
int64_t veles_task_cancelled(veles_task *t) {
    return t->cancel_requested || t->state == T_CANCELLED;
}

/* the cancelled task has run its cleanups (D43) and is done */
void veles_task_finish_cancelled(veles_task *t) {
    t->hdl = NULL;
    if (t->state == T_CANCELLED) return;
    t->state = T_CANCELLED;
    wake(t->waiter);
    t->waiter = NULL;
    scope_child_finished(t);
}

/* await: true when the target is done, otherwise blocks the caller */
int64_t veles_task_await(veles_task *self, veles_task *target) {
    if (target->state == T_DONE) {
        self->awaiting = NULL;
        return 1;
    }
    if (target->state == T_CANCELLED) {
        self->awaiting = NULL;
        veles_panic("awaited task was cancelled", 26);
    }
    target->waiter = self;
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

veles_task *veles_task_launch(veles_scope *s) {
    veles_task *t = veles_task_new();
    t->scope = s;
    t->index = s->launches++;
    t->sibling = s->children;
    s->children = t;
    s->live++;
    return t;
}

static void remove_io_waiter(veles_task *t);

/* Cancellation is a request: the task is woken and unwinds at the
 * suspension point it was parked on (running its `with` cleanups, D43),
 * then reports itself finished. A task with no frame — it never
 * suspended, or already returned — is finished on the spot. */
static void cancel_task(veles_task *t) {
    if (t->state == T_DONE || t->state == T_CANCELLED || t->cancel_requested) return;
    t->cancel_requested = 1;
    remove_timer(t);
    remove_io_waiter(t);
    t->race = NULL;
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
    t->state = T_CANCELLED;
    scope_child_finished(t);
}

/* `task.cancel()`: ask a task to stop; it unwinds at its next suspension
 * point, and its scope still waits for it */
void veles_task_cancel(veles_task *t) {
    cancel_task(t);
}

/* cancel every child still running; a child with no frame finishes on the
 * spot and unlinks itself, so the next link is read first */
static void cancel_children(veles_scope *s, veles_task *except) {
    veles_task *c = s->children;
    while (c) {
        veles_task *next = c->sibling;
        if (c != except) cancel_task(c);
        c = next;
    }
}

static void scope_child_finished(veles_task *t) {
    veles_scope *s = t->scope;
    if (!s) return;
    s->live--;
    /* a finished child leaves the scope's list: a server's accept loop
     * launches a task per connection for as long as it runs, and the list
     * would otherwise keep every one of them (and its frame) alive */
    for (veles_task **pp = &s->children; *pp; pp = &(*pp)->sibling) {
        if (*pp == t) {
            *pp = t->sibling;
            t->sibling = NULL;
            break;
        }
    }
    if (t->failed && s->fail_fast && !s->failed) {
        s->failed = t;
        cancel_children(s, t);
        /* the owner may be blocked in the scope body (a recv that will now
         * never complete): wake it so its next suspension point sees the
         * failure and abandons the body */
        wake(s->owner);
    }
    if (s->live <= 0) wake(s->owner);
}

/* a task abandoning a scope body forgets whatever it was waiting on; a
   stale entry in a channel's waiter list only causes a harmless wake */
void veles_task_leave_waits(veles_task *t) {
    t->race = NULL;
    remove_timer(t);
    remove_io_waiter(t);
}

/* the body left the scope early (return, throw, cancellation): the
 * children still running are cancelled; the owner then waits for them
 * as usual, so nothing outlives the block (D34) */
void veles_scope_cancel(veles_scope *s) {
    cancel_children(s, NULL);
}

/* wait for every child: true when done, otherwise blocks the owner */
int64_t veles_scope_wait(veles_task *owner, veles_scope *s) {
    if (s->live <= 0) return 1;
    owner->state = T_BLOCKED;
    return 0;
}

veles_task *veles_scope_failed(veles_scope *s) {
    return s->failed;
}

int64_t veles_scope_failed_index(veles_scope *s) {
    return s->failed ? s->failed->index : -1;
}

/* ---- channels (D16) -------------------------------------------------------- */

veles_chan *veles_chan_new(veles_desc *desc, int64_t cap) {
    veles_chan *c = veles_alloc_words(sizeof *c);
    c->desc = desc;
    c->elem = veles_desc_size(desc);
    if (cap < 1) cap = 1; /* rendezvous channels behave as capacity 1 in the single-threaded executor */
    c->cap = cap;
    c->remaining = -1;
    c->buf = veles_gc_alloc(desc, c->elem * cap + 1);
    return c;
}

static veles_task *pop_waiter(veles_task **list) {
    veles_task *t = *list;
    if (t) {
        *list = t->next;
        t->next = NULL;
    }
    return t;
}

static void push_waiter(veles_task **list, veles_task *t) {
    t->next = NULL;
    if (!*list) {
        *list = t;
        return;
    }
    veles_task *p = *list;
    while (p->next) p = p->next;
    p->next = t;
}

static void race_deliver(veles_task *t, veles_chan *c, const void *item);

static void remove_waiter(veles_task **list, veles_task *t) {
    while (*list) {
        if (*list == t) {
            *list = t->next;
            t->next = NULL;
            continue;
        }
        list = &(*list)->next;
    }
}

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
static bool chan_hand_off(veles_chan *c, const void *item) {
    while (c->recv_waiters) {
        veles_task *r = pop_waiter(&c->recv_waiters);
        if (r->state == T_CANCELLED || r->cancel_requested) continue;
        if (r->race) {
            if (r->race->winner >= 0) continue; /* already won elsewhere */
            race_deliver(r, c, item);
            return true;
        }
        /* plain receiver: buffer the value and wake it */
        chan_push(c, item);
        wake(r);
        return true;
    }
    return false;
}

void veles_chan_close(veles_chan *c);

/* send: true when delivered or buffered; false blocks the sender */
int64_t veles_chan_send(veles_task *self, veles_chan *c, const void *item) {
    if (c->closed) veles_panic("send on a closed channel", 24);
    if (c->len < c->cap) {
        if (!chan_hand_off(c, item)) chan_push(c, item);
        if (c->remaining > 0 && --c->remaining == 0) veles_chan_close(c);
        return 1;
    }
    push_waiter(&c->send_waiters, self);
    self->state = T_BLOCKED;
    return 0;
}

/* recv: 1 value received, 2 closed and empty, 0 blocked */
int64_t veles_chan_recv(veles_task *self, veles_chan *c, void *out) {
    if (c->len > 0) {
        chan_pop(c, out);
        wake(pop_waiter(&c->send_waiters));
        return 1;
    }
    if (c->closed) return 2;
    push_waiter(&c->recv_waiters, self);
    self->state = T_BLOCKED;
    return 0;
}

void veles_chan_close(veles_chan *c) {
    c->closed = 1;
    veles_task *t;
    while ((t = pop_waiter(&c->recv_waiters))) {
        if (t->race) {
            if (t->race->winner < 0) {
                /* a closed channel completes the race arm with "closed" */
                t->race->winner = 0;
                t->race->closed = 1;
                for (int64_t i = 0; i < t->race->narms; i++) {
                    if (t->race->arms[i].ch == c) t->race->winner = i;
                }
                t->race->ready = 1;
                t->race = NULL;
                wake(t);
            }
            continue;
        }
        wake(t);
    }
}

int64_t veles_chan_len(veles_chan *c) {
    return c->len;
}

/* trySend: 1 when delivered or buffered, 0 when the buffer is full; never
   blocks. A closed channel panics, as send does: sending into it is a bug
   in the program, not a condition to poll for. */
int64_t veles_chan_try_send(veles_chan *c, const void *item) {
    if (c->closed) veles_panic("send on a closed channel", 24);
    if (c->len >= c->cap) return 0;
    if (!chan_hand_off(c, item)) chan_push(c, item);
    if (c->remaining > 0 && --c->remaining == 0) veles_chan_close(c);
    return 1;
}

/* tryRecv: 1 with a value in *out, 0 when nothing is buffered (empty, or
   closed and drained); never blocks. */
int64_t veles_chan_try_recv(veles_chan *c, void *out) {
    if (c->len == 0) return 0;
    chan_pop(c, out);
    wake(pop_waiter(&c->send_waiters));
    return 1;
}

/* closeAfter(n): the channel closes itself once n more values have been sent,
   so several producers can end it without coordinating. */
void veles_chan_close_after(veles_chan *c, int64_t n) {
    if (n <= 0) {
        veles_chan_close(c);
        return;
    }
    c->remaining = n;
}

/* ---- timers ------------------------------------------------------------------ */

static void add_timer(veles_task *t, int64_t ms) {
    t->wake_at = now_ms() + ms;
    t->timer_next = timers;
    timers = t;
}

/* sleep: true once the deadline passed; first call arms it and blocks.
 * A task can be woken before its deadline for another reason - a scope
 * child finishing wakes the scope's owner - and it then comes back here:
 * it must block again, still on the timer list. Returning 0 while leaving
 * the task T_RUNNABLE lost it: fire_timers' wake() only wakes a blocked
 * task, so the sleeper was never resumed and the executor reported a
 * deadlock (a producer finishing while main slept on a timer). */
int64_t veles_task_sleep(veles_task *self, int64_t ms) {
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
                if (t->race->winner < 0) {
                    for (int64_t i = 0; i < t->race->narms; i++) {
                        if (t->race->arms[i].deadline && t->race->arms[i].deadline <= now) {
                            t->race->winner = i;
                            break;
                        }
                    }
                    t->race->ready = 1;
                    t->race = NULL;
                    t->wake_at = 0;
                    wake(t);
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
int64_t veles_task_wait_io(veles_task *self, int64_t fd, int64_t write) {
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
 * and wake the tasks whose descriptors are ready */
static void poll_io(int64_t timeout_ms) {
    int64_t n = io_waiter_count();
    if (n == 0) return;
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
#if defined(_WIN32)
    int r = WSAPoll(fds, (ULONG)n, (INT)timeout_ms);
#else
    int r = poll(fds, (nfds_t)n, (int)timeout_ms);
#endif
    if (r > 0) {
        for (i = 0; i < n; i++) {
            if (fds[i].revents == 0) continue;
            veles_task *t = tasks[i];
            remove_io_waiter(t);
            t->io_waiting = 1; /* stays "waiting" so the retry sees ready */
            t->io_ready = 1;
            wake(t);
        }
    }
    free(fds);
    free(tasks);
}

/* ---- race (D38) ------------------------------------------------------------ */

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

static void race_deliver(veles_task *t, veles_chan *c, const void *item) {
    veles_race *r = t->race;
    for (int64_t i = 0; i < r->narms; i++) {
        if (r->arms[i].ch == c) {
            memcpy(r->arms[i].out, item, (size_t)c->elem);
            r->winner = i;
            break;
        }
    }
    r->ready = 1;
    t->race = NULL;
    wake(t);
}

/* returns the winning arm, or -1 to block; -2 for a closed channel arm */
static void race_detach(veles_task *self, veles_race *r) {
    for (int64_t i = 0; i < r->narms; i++) {
        if (r->arms[i].ch) remove_waiter(&r->arms[i].ch->recv_waiters, self);
        if (r->arms[i].awaited && r->arms[i].awaited->waiter == self) r->arms[i].awaited->waiter = NULL;
    }
    remove_timer(self);
    self->race = NULL;
}

int64_t veles_race_wait(veles_task *self, veles_race *r) {
    if (r->ready) {
        race_detach(self, r);
        return r->winner;
    }
    /* immediate checks */
    int64_t now = now_ms();
    for (int64_t i = 0; i < r->narms; i++) {
        if (r->arms[i].ch) {
            veles_chan *c = r->arms[i].ch;
            if (c->len > 0) {
                chan_pop(c, r->arms[i].out);
                wake(pop_waiter(&c->send_waiters));
                r->ready = 1;
                r->winner = i;
                return i;
            }
            if (c->closed) {
                r->ready = 1;
                r->closed = 1;
                r->winner = i;
                return i;
            }
        } else if (r->arms[i].deadline) {
            if (now >= r->arms[i].deadline) {
                r->ready = 1;
                r->winner = i;
                return i;
            }
        } else if (r->arms[i].awaited && r->arms[i].awaited->state == T_DONE) {
            r->ready = 1;
            r->winner = i;
            return i;
        }
    }
    /* register everywhere and block */
    self->race = r;
    int64_t earliest = 0;
    for (int64_t i = 0; i < r->narms; i++) {
        if (r->arms[i].ch) {
            push_waiter(&r->arms[i].ch->recv_waiters, self);
        } else if (r->arms[i].deadline) {
            if (!earliest || r->arms[i].deadline < earliest) earliest = r->arms[i].deadline;
        } else if (r->arms[i].awaited) {
            r->arms[i].awaited->waiter = self;
        }
    }
    if (earliest) {
        remove_timer(self);
        self->wake_at = earliest;
        self->timer_next = timers;
        timers = self;
    }
    self->state = T_BLOCKED;
    return -1;
}

/* ---- executor ---------------------------------------------------------------- */

static jmp_buf panic_return;
static int in_resume;

/* ---- cleanups (D43/D49) ---------------------------------------------------- */

void veles_cleanup_push(void (*fn)(void *), void *env) {
    if (!current) return;
    veles_cleanup *c = veles_alloc_words(sizeof *c);
    c->fn = fn;
    c->env = env;
    c->next = current->cleanups;
    current->cleanups = c;
}

void veles_cleanup_pop(void) {
    if (!current || !current->cleanups) return;
    current->cleanups = current->cleanups->next;
}

/* veles_task_panic is called by veles_panic while a task runs: the task's
 * active cleanups run (a `with` closes its resource, a scope cancels its
 * children), then the task fails with the message and control returns to
 * the executor (D20: a panic unwinds to the enclosing task scope). A
 * panic inside a cleanup continues the unwinding with the first message. */
int64_t veles_task_panic(const char *msg, int64_t len, const char *loc, int64_t loc_len) {
    if (!in_resume || !current) return 0;
    veles_task *t = current;
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
    t->panicked = 1;
    t->failed = 1;
    t->state = T_DONE;
    t->hdl = NULL;
    t->result = veles_alloc_words(8);
    wake(t->waiter);
    t->waiter = NULL;
    scope_child_finished(t);
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
    }
    in_resume = 0;
    current = NULL;
}

/* veles_run drives the executor until the root task completes */
void veles_run(veles_task *root) {
    register_roots();
    start_ms = now_ms();
    while (root->state != T_DONE && root->state != T_CANCELLED) {
        fire_timers();
        if (io_waiters) poll_io(0); /* runnable tasks must not starve the sockets */
        veles_task *t = dequeue();
        if (!t) {
            /* nothing runnable: wait for the nearest timer or a socket */
            int64_t nearest = 0;
            for (veles_task *x = timers; x; x = x->timer_next) {
                if (x->wake_at && (!nearest || x->wake_at < nearest)) nearest = x->wake_at;
            }
            if (io_waiters) {
                int64_t wait = nearest ? nearest - now_ms() : -1;
                if (nearest && wait < 0) wait = 0;
                poll_io(wait);
                continue;
            }
            if (!nearest) {
                veles_panic("deadlock: every task is blocked", 31);
            }
            sleep_ms(nearest - now_ms());
            continue;
        }
        if (t->state == T_CANCELLED || t->state == T_DONE) continue;
        t->state = T_RUNNABLE;
        resume(t);
    }
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
void veles_task_start(veles_task *t, void (*entry)(veles_task *, void *), void *args) {
    register_roots();
    veles_task *saved_current = current;
    int saved_in = in_resume;
    jmp_buf saved;
    memcpy(&saved, &panic_return, sizeof saved);
    current = t;
    in_resume = 1;
    if (setjmp(panic_return) == 0) {
        entry(t, args);
    }
    memcpy(&panic_return, &saved, sizeof saved);
    current = saved_current;
    in_resume = saved_in;
}
