/*
 * Veles collector (build plan Stage 3; spec I3 and §5).
 *
 * Non-moving mark-sweep. The heap is made of 64 KiB spans, each holding
 * objects of one size class, so any address — interior pointers included
 * (D10) — resolves to its span through a page table and then to its object
 * by division. Roots are scanned conservatively (the stack, registers via
 * setjmp, and registered globals); heap objects are scanned precisely using
 * the type descriptor stored in each object's header. Descriptors list the
 * word offsets that may hold pointers; values found there are validated
 * against the page table, so a payload that is only sometimes a pointer
 * (a sealed union) is simply listed as a candidate.
 */
/* glibc declares its extensions (pthread_getattr_np, ...) only when asked. */
#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <setjmp.h>

#if defined(_WIN32)
#include <windows.h>
#elif defined(__APPLE__)
#include <pthread.h>
#else
#include <pthread.h>
#endif

typedef struct veles_desc {
    int64_t size;     /* object size in bytes (element size for arrays) */
    int64_t kind;     /* 0 object, 1 array of elements */
    int64_t nptrs;    /* number of candidate word offsets */
    int64_t offsets[]; /* byte offsets of candidate pointer words */
} veles_desc;

#define SPAN_SHIFT 16
#define SPAN_SIZE ((size_t)1 << SPAN_SHIFT)
#define HEADER 8

typedef struct veles_span {
    char *start;
    size_t bytes;      /* SPAN_SIZE or a multiple for large objects */
    size_t objsize;    /* including header */
    size_t nobjs;
    uint8_t *marks;    /* per object */
    uint8_t *used;     /* per object: allocated */
    size_t free_hint;  /* every slot below it is in use (slots are freed only by a sweep) */
    size_t nused;      /* slots in use, so a full span is skipped without a scan */
    struct veles_span *next; /* in size class list */
    void *owner;       /* the thread allocating from it, or NULL (D66) */
    int cls;
} veles_span;

static const size_t class_sizes[] = {16, 32, 48, 64, 96, 128, 192, 256, 384, 512, 768, 1024, 1536, 2048, 4096, 8192, 16384, 32768};
#define NCLASSES (sizeof class_sizes / sizeof class_sizes[0])

static veles_span *classes[NCLASSES];
/* the first span of each class that may have a free slot: the spans before
 * it filled up since the last sweep, and only a sweep frees a slot */
static veles_span *cursor[NCLASSES];
static veles_span **all_spans;
static size_t nspans, cap_spans;

/* page table: open-addressing hash from page number to span */
static veles_span **pages;
static size_t pages_cap;
static size_t pages_used;

static size_t allocated_since_gc;
static size_t live_bytes;
static size_t gc_threshold = 8u << 20;
static size_t gc_min_threshold = 8u << 20;
static char *stack_base;
static bool gc_enabled = true;
static int64_t gc_count;

typedef struct {
    void *addr;
    veles_desc *desc;
} veles_root;
static veles_root *roots;
static size_t nroots, cap_roots;

void veles_panic(const char *msg, int64_t len);


/* ---- threads and the stopped world (D66) -------------------------------
 * Every OS thread that runs Veles code is registered. A collection stops
 * the world: it raises veles_stop_requested and waits until every other
 * thread is parked at a safepoint (an allocation, a runtime lock, a loop's
 * back edge) or inside a safe region (a blocking system call, an idle
 * wait). Either way the thread has left its registers in its record and
 * its stack below a known top, and the collector scans both. */

typedef struct veles_lock veles_lock;
typedef struct veles_cond veles_cond;
veles_lock *veles_lock_new(void);
void veles_lock_acquire(veles_lock *l);
int64_t veles_lock_try(veles_lock *l);
void veles_lock_release(veles_lock *l);
veles_cond *veles_cond_new(void);
void veles_cond_wait(veles_cond *c, veles_lock *l, int64_t timeout_ms);
void veles_cond_broadcast(veles_cond *c);

#include "veles_tls.h"

typedef struct veles_thread {
    char *stack_base;
    char *stack_top;   /* while safe or parked: the deepest live frame */
    uint64_t regs[VELES_CAPTURE_WORDS]; /* while safe or parked: the callee-saved registers */
    int64_t safe;      /* nesting depth of safe regions (parked counts as one) */
    int32_t in_safe;   /* 1 while safe: what the collector reads (atomic) */
    int64_t foreign;   /* registered by a callback from a thread Veles did not start */
    struct veles_tls *tls; /* its per-thread block: the executor keeps task pointers there */
    struct veles_span *tl[NCLASSES]; /* the spans this thread allocates from */
    size_t tl_bytes;   /* allocated since last added to allocated_since_gc */
    struct veles_thread *next;
} veles_thread;

static veles_lock *world_lock;
static veles_cond *world_cv;
static veles_thread *threads;
#include "veles_tls.h"
#define me (veles_tls_get()->thread) /* this thread's record, NULL until attached */
static veles_lock *heap_lock;
static size_t tl_chunk = 64u << 10; /* a thread's allocation between trigger checks */

/* read by the loop back-edge polls the compiler emits */
volatile int32_t veles_stop_requested;

static void *sys_alloc_aligned(size_t bytes) {
#if defined(_WIN32)
    /* VirtualAlloc returns 64 KiB aligned regions */
    void *p = VirtualAlloc(NULL, bytes, MEM_COMMIT | MEM_RESERVE, PAGE_READWRITE);
    if (p && ((uintptr_t)p & (SPAN_SIZE - 1))) {
        VirtualFree(p, 0, MEM_RELEASE);
        p = VirtualAlloc(NULL, bytes + SPAN_SIZE, MEM_RESERVE, PAGE_READWRITE);
        uintptr_t aligned = ((uintptr_t)p + SPAN_SIZE - 1) & ~(uintptr_t)(SPAN_SIZE - 1);
        VirtualFree(p, 0, MEM_RELEASE);
        p = VirtualAlloc((void *)aligned, bytes, MEM_COMMIT | MEM_RESERVE, PAGE_READWRITE);
    }
    return p;
#else
    void *p = NULL;
    if (posix_memalign(&p, SPAN_SIZE, bytes) != 0) return NULL;
    memset(p, 0, bytes);
    return p;
#endif
}

static void sys_free_aligned(void *p, size_t bytes) {
#if defined(_WIN32)
    (void)bytes;
    VirtualFree(p, 0, MEM_RELEASE);
#else
    (void)bytes;
    free(p);
#endif
}

static void oom(void) {
    fputs("panic: out of memory\n", stderr);
    exit(101);
}

/* ---- page table --------------------------------------------------------- */

static size_t page_hash(uintptr_t page) {
    return (size_t)((page * 0x9E3779B97F4A7C15ULL) >> 20);
}

static void pages_grow(void) {
    size_t old_cap = pages_cap;
    veles_span **old = pages;
    pages_cap = pages_cap ? pages_cap * 2 : 1024;
    pages = calloc(pages_cap, sizeof *pages);
    if (!pages) oom();
    pages_used = 0;
    for (size_t i = 0; i < old_cap; i++) {
        veles_span *s = old[i];
        if (!s) continue;
        for (size_t off = 0; off < s->bytes; off += SPAN_SIZE) {
            uintptr_t page = ((uintptr_t)s->start + off) >> SPAN_SHIFT;
            /* each page may appear once per span page */
            size_t h = page_hash(page) & (pages_cap - 1);
            while (pages[h] && pages[h] != s) h = (h + 1) & (pages_cap - 1);
            if (pages[h] != s) { pages[h] = s; pages_used++; }
        }
    }
    free(old);
}

static void pages_insert(uintptr_t page, veles_span *s) {
    if ((pages_used + 1) * 2 > pages_cap) pages_grow();
    size_t h = page_hash(page) & (pages_cap - 1);
    while (pages[h]) h = (h + 1) & (pages_cap - 1);
    pages[h] = s;
    pages_used++;
}

static veles_span *span_of(const void *p) {
    if (!pages_cap) return NULL;
    uintptr_t page = (uintptr_t)p >> SPAN_SHIFT;
    size_t h = page_hash(page) & (pages_cap - 1);
    while (pages[h]) {
        veles_span *s = pages[h];
        if ((const char *)p >= s->start && (const char *)p < s->start + s->bytes) return s;
        h = (h + 1) & (pages_cap - 1);
    }
    return NULL;
}

/* ---- spans ---------------------------------------------------------------- */

static veles_span *new_span(int cls, size_t objsize, size_t bytes) {
    veles_span *s = calloc(1, sizeof *s);
    if (!s) oom();
    s->start = sys_alloc_aligned(bytes);
    if (!s->start) oom();
    s->bytes = bytes;
    s->objsize = objsize;
    s->nobjs = bytes / objsize;
    s->marks = calloc(s->nobjs, 1);
    s->used = calloc(s->nobjs, 1);
    s->cls = cls;
    if (nspans == cap_spans) {
        cap_spans = cap_spans ? cap_spans * 2 : 64;
        all_spans = realloc(all_spans, cap_spans * sizeof *all_spans);
        if (!all_spans) oom();
    }
    all_spans[nspans++] = s;
    for (size_t off = 0; off < bytes; off += SPAN_SIZE) {
        pages_insert(((uintptr_t)s->start + off) >> SPAN_SHIFT, s);
    }
    return s;
}

static int class_for(size_t bytes) {
    for (size_t i = 0; i < NCLASSES; i++) {
        if (bytes <= class_sizes[i]) return (int)i;
    }
    return -1;
}

void veles_gc_collect(void);

/* ---- allocation ----------------------------------------------------------- */

static void *alloc_in_span(veles_span *s, veles_desc *desc) {
    /* slots are freed only by a sweep, which resets free_hint, so the hint
     * only moves forward between collections and the scan is amortised
     * O(1); a full span answers at once */
    if (s->nused == s->nobjs) return NULL;
    for (size_t i = s->free_hint; i < s->nobjs; i++) {
        if (!s->used[i]) {
            s->used[i] = 1;
            s->nused++;
            s->free_hint = i + 1;
            char *obj = s->start + i * s->objsize;
            memset(obj, 0, s->objsize);
            *(veles_desc **)obj = desc;
            return obj + HEADER;
        }
    }
    return NULL;
}

static void heap_acquire(void);
static void collect_now(void);
static void *alloc_slow(veles_desc *desc, size_t need, int cls);

/* Each thread allocates from spans it owns, one per size class, without a
 * lock (D66): nobody else takes a slot from an owned span, and a
 * collection — the only other writer — runs with every thread stopped and
 * takes the ownership away. What a thread allocated is added to the
 * global count in chunks, so the collection trigger stays close. */
void *veles_gc_alloc(veles_desc *desc, int64_t size) {
    if (size < 0) size = 0;
    size_t need = (size_t)size + HEADER;
    int cls = class_for(need);
    if (me && cls >= 0 && !veles_stop_requested) {
        veles_span *s = me->tl[cls];
        if (s && me->tl_bytes + need <= tl_chunk) {
            void *p = alloc_in_span(s, desc);
            if (p) {
                me->tl_bytes += need;
                return p;
            }
        }
    }
    return alloc_slow(desc, need, cls);
}

/* a span of the class with room that no thread owns, or a new one */
static veles_span *claim_span(int cls) {
    for (veles_span *s = cursor[cls] ? cursor[cls] : classes[cls]; s; s = s->next) {
        if (!s->owner && s->nused < s->nobjs) {
            cursor[cls] = s;
            return s;
        }
    }
    veles_span *s = new_span(cls, class_sizes[cls], SPAN_SIZE);
    s->next = classes[cls];
    classes[cls] = s;
    cursor[cls] = s;
    return s;
}

static void *alloc_slow(veles_desc *desc, size_t need, int cls) {
    heap_acquire();
    if (me) {
        allocated_since_gc += me->tl_bytes;
        me->tl_bytes = 0;
    }
    if (gc_enabled && allocated_since_gc > gc_threshold) {
        veles_lock_release(heap_lock);
        collect_now();
        heap_acquire();
    }
    allocated_since_gc += need;
    void *p;
    if (cls < 0) {
        size_t bytes = (need + SPAN_SIZE - 1) & ~(SPAN_SIZE - 1);
        veles_span *s = new_span(-1, bytes, bytes);
        s->nobjs = 1;
        p = alloc_in_span(s, desc);
    } else {
        veles_span *s = me ? me->tl[cls] : NULL;
        p = s ? alloc_in_span(s, desc) : NULL;
        if (!p) {
            if (s) s->owner = NULL; /* full: give it back */
            s = claim_span(cls);
            if (me) {
                s->owner = me;
                me->tl[cls] = s;
            }
            p = alloc_in_span(s, desc);
        }
    }
    veles_lock_release(heap_lock);
    if (!p) oom();
    return p;
}

/* descriptors for runtime-internal allocations */
static veles_desc desc_noscan = {0, 0, 0};
static veles_desc desc_words = {8, 1, 1, {0}}; /* array of pointer-sized candidates */

void *veles_alloc(int64_t size) {
    /* raw bytes with no pointers (strings, metadata) */
    return veles_gc_alloc(&desc_noscan, size);
}

void *veles_alloc_words(int64_t size) {
    /* every word may be a pointer (runtime containers) */
    return veles_gc_alloc(&desc_words, size);
}

void veles_gc_root(void *addr, veles_desc *desc) {
    heap_acquire();
    if (nroots == cap_roots) {
        cap_roots = cap_roots ? cap_roots * 2 : 32;
        roots = realloc(roots, cap_roots * sizeof *roots);
        if (!roots) oom();
    }
    roots[nroots].addr = addr;
    roots[nroots].desc = desc;
    nroots++;
    veles_lock_release(heap_lock);
}

/* ---- marking -------------------------------------------------------------- */

static char **work;
static size_t nwork, cap_work;

static void push_work(char *obj) {
    if (nwork == cap_work) {
        cap_work = cap_work ? cap_work * 2 : 4096;
        work = realloc(work, cap_work * sizeof *work);
        if (!work) oom();
    }
    work[nwork++] = obj;
}

/* mark_candidate resolves an arbitrary word to a heap object and marks it */
static void mark_candidate(uintptr_t word) {
    if (word < 4096) return;
    veles_span *s = span_of((void *)word);
    if (!s) return;
    size_t off = (size_t)(word - (uintptr_t)s->start);
    size_t idx = off / s->objsize;
    if (idx >= s->nobjs || !s->used[idx] || s->marks[idx]) return;
    s->marks[idx] = 1;
    push_work(s->start + idx * s->objsize);
}

/* Under `veles build --sanitize` (AddressSanitizer): the collector reads
 * whole stacks, the redzones between a C frame's locals included — those
 * reads are the point, so the scan is not instrumented. And a C local must
 * live on the real stack, where the scan finds it: in the sanitizer's
 * use-after-return mode it lives in a "fake frame" on the side, and a heap
 * object held only there would be swept while in use. */
#if defined(__has_feature)
#if __has_feature(address_sanitizer)
#define VELES_ASAN 1
#endif
#endif
#if defined(__SANITIZE_ADDRESS__) && !defined(VELES_ASAN)
#define VELES_ASAN 1
#endif
#if defined(VELES_ASAN)
#define NO_ASAN __attribute__((no_sanitize("address")))
const char *__asan_default_options(void) { return "detect_stack_use_after_return=0"; }
#else
#define NO_ASAN
#endif

NO_ASAN static void scan_range(const char *lo, const char *hi) {
    lo = (const char *)(((uintptr_t)lo + 7) & ~(uintptr_t)7);
    for (const char *p = lo; p + 8 <= hi; p += 8) {
        mark_candidate(*(const uintptr_t *)p);
    }
}

static void scan_object(char *obj, size_t objsize) {
    veles_desc *d = *(veles_desc **)obj;
    char *body = obj + HEADER;
    if (!d || d->nptrs == 0) return;
    if (d->kind == 1) {
        if (d->size <= 0) return;
        size_t count = (objsize - HEADER) / (size_t)d->size;
        for (size_t i = 0; i < count; i++) {
            char *el = body + i * d->size;
            for (int64_t k = 0; k < d->nptrs; k++) {
                mark_candidate(*(uintptr_t *)(el + d->offsets[k]));
            }
        }
        return;
    }
    for (int64_t k = 0; k < d->nptrs; k++) {
        if ((size_t)d->offsets[k] + 8 <= objsize - HEADER) {
            mark_candidate(*(uintptr_t *)(body + d->offsets[k]));
        }
    }
}

static void drain(void) {
    while (nwork > 0) {
        char *obj = work[--nwork];
        veles_span *s = span_of(obj);
        scan_object(obj, s->objsize);
    }
}

void veles_capture_regs(void);

/* the collecting thread's own roots: its registers and stack as they are
 * here, recorded the way a safe thread's are (veles_capture_regs); this
 * frame is live while it scans */
static void scan_stack(void) {
#if defined(VELES_CAPTURE_ASM)
    veles_tls *b = veles_tls_get();
    veles_capture_regs();
    scan_range((const char *)b->capture, (const char *)(b->capture + VELES_CAPTURE_WORDS));
    char *sp = (char *)(uintptr_t)b->capture[VELES_CAPTURE_SP];
#else
    jmp_buf regs;
    setjmp(regs); /* spill callee-saved registers onto the stack */
    char *sp = (char *)&regs;
    scan_range((const char *)regs, (const char *)regs + sizeof regs);
#endif
    char *base = me ? me->stack_base : stack_base;
    if (base && sp < base) scan_range(sp, base);
}

static char *find_stack_base(void) {
#if defined(_WIN32)
    NT_TIB *tib = (NT_TIB *)NtCurrentTeb();
    return (char *)tib->StackBase;
#elif defined(__APPLE__)
    return (char *)pthread_get_stackaddr_np(pthread_self());
#else
    pthread_attr_t attr;
    void *addr = NULL;
    size_t size = 0;
    if (pthread_getattr_np(pthread_self(), &attr) == 0) {
        pthread_attr_getstack(&attr, &addr, &size);
        pthread_attr_destroy(&attr);
        return (char *)addr + size;
    }
    return NULL;
#endif
}

/* ---- the thread protocol -------------------------------------------------- */

/* ---- recording a safe thread's roots ----------------------------------------
 * A safe or parked thread is scanned from what it left behind: its
 * callee-saved registers and its stack from the waiting frame up. Both
 * must be the waiting frame's own. A C function recording them — setjmp
 * in a helper — first saves the caller's registers it is about to use in
 * its own frame, and that frame is dead once it returns: whatever the
 * thread calls while it waits (a lock, a condition variable, a foreign
 * function) overwrites it before the collector looks, and a pointer the
 * caller kept in such a register is lost (seen as a task just created by
 * `async` being swept while its owner waited out a collection). So
 * veles_enter_safe and veles_blocking_enter begin in assembly that records
 * the registers and stack pointer exactly as the caller left them
 * (VELES_CAPTURE_ASM, veles_tls.h); the caller's frame stays live for as
 * long as the thread is safe. */

#if defined(VELES_CAPTURE_ASM)
__attribute__((naked)) void veles_capture_regs(void) {
    __asm__ volatile(VELES_CAPTURE_ASM "ret\n\t");
}

void veles_enter_safe_c(void);

__attribute__((naked)) void veles_enter_safe(void) {
    __asm__ volatile(VELES_CAPTURE_ASM VELES_ASM_TAIL(veles_enter_safe_c));
}

/* the record VELES_CAPTURE_ASM made at entry becomes the thread's */
static void save_state(void) {
    veles_tls *b = veles_tls_get();
    for (int i = 0; i < VELES_CAPTURE_WORDS; i++) me->regs[i] = b->capture[i];
    me->stack_top = (char *)(uintptr_t)b->capture[VELES_CAPTURE_SP];
}
#else
/* no capture for this architecture: setjmp in a helper, which misses a
 * register the helper reuses (see above) */
__attribute__((noinline)) static void save_state(void) {
    char here;
    jmp_buf regs;
    setjmp(regs);
    memcpy(me->regs, &regs, sizeof me->regs < sizeof regs ? sizeof me->regs : sizeof regs);
    me->stack_top = &here;
}
#define veles_enter_safe_c veles_enter_safe
#endif

/* A thread about to block where it touches no Veles memory — a system
 * call, a foreign function, an idle wait, a contended lock. The collector
 * may run meanwhile. Entering and leaving take no lock unless a collection
 * is starting: each side writes its own flag, then reads the other's (the
 * thread in_safe then veles_stop_requested; the collector the reverse),
 * all sequentially consistent, so at least one of them sees the other and
 * a thread never runs Veles code while the collector counts it safe. */
void veles_enter_safe_c(void) {
    if (!me) return;
    if (me->safe++ > 0) return;
    save_state();
    __atomic_store_n(&me->in_safe, 1, __ATOMIC_SEQ_CST);
    if (__atomic_load_n(&veles_stop_requested, __ATOMIC_SEQ_CST)) {
        /* a collector may be waiting for this thread */
        veles_lock_acquire(world_lock);
        veles_cond_broadcast(world_cv);
        veles_lock_release(world_lock);
    }
}

/* back from blocking: waits out a collection that is starting or running */
void veles_leave_safe(void) {
    if (!me) return;
    if (--me->safe > 0) return;
    for (;;) {
        __atomic_store_n(&me->in_safe, 0, __ATOMIC_SEQ_CST);
        if (!__atomic_load_n(&veles_stop_requested, __ATOMIC_SEQ_CST)) return;
        __atomic_store_n(&me->in_safe, 1, __ATOMIC_SEQ_CST);
        veles_lock_acquire(world_lock);
        veles_cond_broadcast(world_cv); /* it may have seen the 0 */
        while (__atomic_load_n(&veles_stop_requested, __ATOMIC_SEQ_CST)) veles_cond_wait(world_cv, world_lock, -1);
        veles_lock_release(world_lock);
    }
}

/* A call from C into Veles (D69) leaves the safe region of the extern call
 * that reached C, and returns to it after; a thread Veles did not start is
 * registered on its first callback and counts as safe whenever it is back
 * in C. What enter returns, leave restores. */
void veles_thread_attach(void);

/* The safe region a callback interrupts was recorded where the Veles code
 * called C, and its frames are still there under C's: that record is kept
 * aside for the callback and put back after it, since recording again on
 * the way back would describe the C frames that are about to return. */
typedef struct outer_record {
    int64_t safe;
    uint64_t regs[VELES_CAPTURE_WORDS];
    char *stack_top;
} outer_record;

int64_t veles_callback_enter(void) {
    if (!me) {
        veles_thread_attach();
        me->foreign = 1;
        return 0;
    }
    if (me->safe == 0) return 0;
    outer_record *o = malloc(sizeof *o);
    if (!o) oom();
    o->safe = me->safe;
    memcpy(o->regs, me->regs, sizeof o->regs);
    o->stack_top = me->stack_top;
    me->safe = 1;
    veles_leave_safe();
    return (int64_t)(uintptr_t)o;
}

/* back into C: safe again, described by the record from before the
 * callback — or, for a thread Veles did not start, by nothing: no Veles
 * frame is left on it */
static void enter_safe_recorded(int64_t safe) {
    me->safe = safe;
    __atomic_store_n(&me->in_safe, 1, __ATOMIC_SEQ_CST);
    if (__atomic_load_n(&veles_stop_requested, __ATOMIC_SEQ_CST)) {
        veles_lock_acquire(world_lock);
        veles_cond_broadcast(world_cv);
        veles_lock_release(world_lock);
    }
}

void veles_callback_leave(int64_t saved) {
    if (!me) return;
    if (saved) {
        outer_record *o = (outer_record *)(uintptr_t)saved;
        memcpy(me->regs, o->regs, sizeof me->regs);
        me->stack_top = o->stack_top;
        enter_safe_recorded(o->safe);
        free(o);
        return;
    }
    if (me->foreign) {
        memset(me->regs, 0, sizeof me->regs);
        me->stack_top = me->stack_base;
        enter_safe_recorded(1);
    }
}

/* a safepoint that found a collection requested: stop here until it ends */
void veles_gc_park(void) {
    if (!me || me->safe > 0) return;
    veles_enter_safe();
    veles_leave_safe();
}

void veles_safepoint(void) {
    if (veles_stop_requested) veles_gc_park();
}

/* takes a lock another thread may hold across a safepoint: waiting for it
 * is safe, and the world must not be stopping once it is ours */
void veles_lock_enter(veles_lock *l) {
    for (;;) {
        if (veles_stop_requested) veles_gc_park();
        if (veles_lock_try(l)) {
            if (!veles_stop_requested || !me || me->safe > 0) return;
            veles_lock_release(l);
            continue;
        }
        veles_enter_safe();
        veles_lock_acquire(l);
        veles_lock_release(l);
        veles_leave_safe();
    }
}

static void heap_acquire(void) {
    veles_lock_enter(heap_lock);
}

/* registers the calling OS thread; a worker calls it before any Veles code */
void veles_thread_attach(void) {
    if (me) return;
    veles_thread *t = calloc(1, sizeof *t);
    if (!t) oom();
    t->stack_base = find_stack_base();
    t->tls = veles_tls_get();
    veles_lock_acquire(world_lock);
    while (veles_stop_requested) veles_cond_wait(world_cv, world_lock, -1);
    t->next = threads;
    threads = t;
    me = t;
    veles_lock_release(world_lock);
}

/* every thread but this one is in a safe region or parked */
static int others_safe(void) {
    for (veles_thread *t = threads; t; t = t->next) {
        if (t != me && !__atomic_load_n(&t->in_safe, __ATOMIC_SEQ_CST)) return 0;
    }
    return 1;
}

/* stops every other thread; 0 when another thread's collection ran
 * instead (this one waited it out, parked) */
static int stop_the_world(void) {
    veles_lock_acquire(world_lock);
    if (veles_stop_requested) {
        veles_lock_release(world_lock);
        veles_gc_park();
        return 0;
    }
    __atomic_store_n(&veles_stop_requested, 1, __ATOMIC_SEQ_CST);
    while (!others_safe()) veles_cond_wait(world_cv, world_lock, -1);
    veles_lock_release(world_lock);
    return 1;
}

static void start_the_world(void) {
    veles_lock_acquire(world_lock);
    __atomic_store_n(&veles_stop_requested, 0, __ATOMIC_SEQ_CST);
    veles_cond_broadcast(world_cv);
    veles_lock_release(world_lock);
}

static void scan_threads(void) {
    for (veles_thread *t = threads; t; t = t->next) {
        scan_range((const char *)t->tls, (const char *)(t->tls + 1));
        if (t == me) continue;
        scan_range((const char *)&t->regs, (const char *)&t->regs + sizeof t->regs);
        if (t->stack_top && t->stack_base && t->stack_top < t->stack_base) scan_range(t->stack_top, t->stack_base);
    }
}

static void collect_locked(void);

static void collect_now(void) {
    if (me && me->safe > 0) return; /* a safe thread must not touch the heap */
    if (!stop_the_world()) return;
    veles_lock_acquire(heap_lock);
    if (allocated_since_gc > gc_threshold) collect_locked();
    veles_lock_release(heap_lock);
    start_the_world();
}

void veles_gc_init(void) {
    world_lock = veles_lock_new();
    world_cv = veles_cond_new();
    heap_lock = veles_lock_new();
    veles_thread_attach();
    stack_base = find_stack_base();
    const char *env = getenv("VELES_GC_THRESHOLD");
    if (env) gc_min_threshold = gc_threshold = (size_t)strtoull(env, NULL, 10);
    tl_chunk = gc_min_threshold / 8;
    if (tl_chunk < 256) tl_chunk = 256;
    if (tl_chunk > (256u << 10)) tl_chunk = 256u << 10;
    if (getenv("VELES_GC_OFF")) gc_enabled = false;
}

/* ---- sweeping ------------------------------------------------------------- */

/* VELES_GC_POISON=1 fills every swept object with 0xCD: a use after a
 * wrong free then crashes at the use instead of reading a new object */
static int gc_poison = -1;
static void sweep(void) {
    if (gc_poison < 0) gc_poison = getenv("VELES_GC_POISON") != NULL;
    live_bytes = 0;
    memset(cursor, 0, sizeof cursor); /* every span may have room again */
    size_t w = 0;
    for (size_t i = 0; i < nspans; i++) {
        veles_span *s = all_spans[i];
        size_t live = 0;
        for (size_t k = 0; k < s->nobjs; k++) {
            if (s->used[k] && !s->marks[k]) {
                s->used[k] = 0;
                if (gc_poison) memset(s->start + k * s->objsize, 0xCD, s->objsize);
            } else if (s->used[k]) {
                live++;
            }
            s->marks[k] = 0;
        }
        s->free_hint = 0;
        s->nused = live;
        live_bytes += live * s->objsize;
        if (live == 0 && s->cls < 0) {
            /* release large-object spans */
            sys_free_aligned(s->start, s->bytes);
            free(s->marks);
            free(s->used);
            free(s);
            continue;
        }
        all_spans[w++] = s;
    }
    if (w != nspans) {
        nspans = w;
        /* rebuild the page table without the released spans */
        memset(pages, 0, pages_cap * sizeof *pages);
        pages_used = 0;
        for (size_t i = 0; i < nspans; i++) {
            veles_span *sp = all_spans[i];
            for (size_t off = 0; off < sp->bytes; off += SPAN_SIZE) {
                pages_insert(((uintptr_t)sp->start + off) >> SPAN_SHIFT, sp);
            }
        }
    }
}

/* an explicit collection (tests, the gc example) */
void veles_gc_collect(void) {
    if (!stop_the_world()) return;
    veles_lock_acquire(heap_lock);
    collect_locked();
    veles_lock_release(heap_lock);
    start_the_world();
}

static void collect_locked(void) {
    gc_count++;
    for (veles_thread *t = threads; t; t = t->next) {
        allocated_since_gc += t->tl_bytes;
        t->tl_bytes = 0;
    }
    for (size_t i = 0; i < nroots; i++) {
        veles_desc *d = roots[i].desc;
        char *addr = roots[i].addr;
        if (!d) {
            mark_candidate(*(uintptr_t *)addr);
            continue;
        }
        for (int64_t k = 0; k < d->nptrs; k++) {
            mark_candidate(*(uintptr_t *)(addr + d->offsets[k]));
        }
    }
    scan_stack();
    scan_threads();
    drain();
    sweep();
    /* every thread is stopped: the spans are nobody's until claimed again */
    for (veles_thread *t = threads; t; t = t->next) memset(t->tl, 0, sizeof t->tl);
    for (size_t i = 0; i < nspans; i++) all_spans[i]->owner = NULL;
    allocated_since_gc = 0;
    size_t next = live_bytes * 2;
    if (next < gc_min_threshold) next = gc_min_threshold;
    gc_threshold = next;
    if (getenv("VELES_GC_TRACE")) {
        fprintf(stderr, "[gc #%lld] live %zu bytes, %zu spans\n", (long long)gc_count, live_bytes, nspans);
    }
}

int64_t veles_gc_collections(void) {
    return gc_count;
}

int64_t veles_gc_live_bytes(void) {
    return (int64_t)live_bytes;
}

int64_t veles_desc_size(veles_desc *d) {
    return d ? d->size : 0;
}
