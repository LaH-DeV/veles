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
#if !defined(_WIN32)
#include <sched.h>
#include <sys/mman.h>
#endif

#if defined(_WIN32)
#include <windows.h>
#elif defined(__APPLE__)
#include <pthread.h>
#else
#include <pthread.h>
#endif

typedef struct veles_desc {
    int64_t size;     /* object size in bytes (element size for arrays) */
    int64_t kind;     /* DESC_ARRAY: an array of elements; DESC_WIPE: zeroed when freed */
    int64_t nptrs;    /* number of candidate word offsets */
    int64_t offsets[]; /* byte offsets of candidate pointer words */
} veles_desc;

#define DESC_ARRAY 1
/* D112: the bytes of a Secret live in objects of a descriptor with this bit,
 * and the sweep zeroes such an object when it frees it — no finalizer, no
 * ordering; the memory holds no copy of the secret once it is garbage. */
#define DESC_WIPE 2
/* D120: a type declared `@align(n)` above 8 has n in the kind above bit 8.
 * Its objects come from a size class that is a multiple of n (spans start
 * aligned), so the object is n-aligned, and the body starts n bytes in
 * rather than HEADER: the bytes between the descriptor word and the body
 * are unused. Lists of such elements get their storage from it too. */
#define DESC_ALIGN_SHIFT 8

static size_t body_offset(const veles_desc *d) {
    size_t a = d ? (size_t)(d->kind >> DESC_ALIGN_SHIFT) : 0;
    return a > 8 ? a : 8;
}

/* the byte buffer of a Secret (veles_secret_bytes) */
veles_desc veles_wipe_u8_desc = {1, DESC_ARRAY | DESC_WIPE, 0};

/* zeroes n bytes in a way the C compiler cannot drop as a dead store */
static void *(*volatile wipe_memset)(void *, int, size_t) = memset;
void veles_wipe(void *p, size_t n) {
    wipe_memset(p, 0, n);
}

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
    struct veles_arena *arena; /* where it was carved from, and its first piece there */
    size_t piece;
    int cls;
    int32_t sweeping;  /* a thread sweeps it now (atomic: claimed by exchange) */
    int64_t swept;     /* the collection whose sweep it has had (atomic): below sweep_epoch, its free slots are not known */
    int64_t rescanned; /* a large object's: the rescan of written pages that took it last (F18 c) */
    int32_t hot;       /* an allocator took it since the last collection's end: its pages are not reset (F18 c) */
    uint8_t marked_any; /* something in it is marked (atomic): a rescan of a page skips a span with nothing (F18 c) */
} veles_span;

/* No class is more than a quarter bigger than the one below it from 64
 * bytes on, so an object wastes at most a fifth of its slot: with half
 * steps (128 then 192 then 256) a parked task's frame and many a list's
 * storage gave a third away (bench/idle, 2026-10-10). */
static const size_t class_sizes[] = {16, 32, 48, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 448, 512, 640, 768,
                                     896, 1024, 1280, 1536, 1792, 2048, 2560, 3072, 4096, 5120, 6144, 8192, 10240, 12288,
                                     16384, 20480, 24576, 32768};
#define NCLASSES (sizeof class_sizes / sizeof class_sizes[0])

/* the class of each size up to SMALL_MAX bytes, by 16-byte step: found in
 * one look instead of a walk along the classes */
#define SMALL_MAX 1024
static uint8_t small_class[SMALL_MAX / 16 + 1];

static veles_span *classes[NCLASSES];
/* the first span of each class that may have a free slot: the spans before
 * it filled up since the last sweep, and only a sweep frees a slot */
static veles_span *cursor[NCLASSES];
static veles_span **all_spans;
static size_t nspans, cap_spans;

/* Arenas (plan F18 b): the spans are carved from 64 MiB regions reserved
 * at 64 MiB boundaries (a larger object gets a region of its own, as many
 * boundaries long). On Windows a region is reserved for write watch, so a
 * collection can ask which of its pages were written. An address finds its
 * arena through a two-level table indexed by its high bits, and its span
 * through the arena's slots, one per 64 KiB; neither table is ever moved or
 * freed, so a marker may read them while an allocator adds to them (each
 * entry is published once it is ready). */
#define ARENA_SHIFT 26
#define ARENA_SIZE ((size_t)1 << ARENA_SHIFT)
#define ARENA_L2_BITS 11
#define ARENA_L1_BITS 11 /* 26 + 11 + 11 = 48 bits of address */

typedef struct veles_arena {
    char *base;
    size_t bytes;           /* ARENA_SIZE, or a multiple for an object larger than that */
    size_t nslots;          /* 64 KiB pieces */
    veles_span **slots;     /* the span covering each piece (atomic) */
    uint8_t *taken;         /* a piece belongs to a span, live or released (heap_lock) */
    size_t free_hint;       /* every piece below it is taken */
    struct veles_arena *next;
} veles_arena;

static veles_arena **arena_l1[(size_t)1 << ARENA_L1_BITS];
static veles_arena *arenas;  /* every arena (heap_lock) */

static size_t allocated_since_gc;
static size_t live_bytes;

/* Lazy sweeping (plan F18 a): a collection marks with the world stopped
 * and leaves the sweep for afterwards. Its spans are listed (sweep_list);
 * a span is swept when an allocator claims it, an allocation that takes
 * the slow path sweeps a few more, the helper threads sweep the rest in the
 * background, and the next collection finishes whatever is left before it
 * marks. A span knows it is swept by its `swept` reaching sweep_epoch. */
static int64_t sweep_epoch;
static veles_span **sweep_list;   /* the spans of the last collection, to sweep */
static int64_t sweep_list_n;
static int64_t sweep_list_next;   /* the next one to take (atomic) */
static int64_t sweep_pending;     /* listed and not yet swept (atomic) */

/* Mostly concurrent marking (F18 c): a collection marks while the program
 * runs, between two short pauses; gc_phase says one is under way. */
enum { GC_IDLE, GC_STARTING, GC_MARKING };
static int32_t gc_phase;          /* atomic */
static size_t gc_threshold = 8u << 20;
static size_t gc_min_threshold = 8u << 20;
/* when a collection starts: the threshold — the goal — for one that marks
 * in a pause; earlier for a concurrent one, by what the last one allocated
 * while it marked, so it ends near the goal instead of past it (Go's
 * pacer; allocating past the goal takes fresh memory, while the garbage
 * waits for the marking to end) */
static size_t gc_trigger = 8u << 20;
static size_t mark_alloc_last;    /* allocated while the last concurrent collection marked */
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
void veles_cond_signal(veles_cond *c);

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
/* what a loop's back edge polls (D66, D145): nonzero while something wants
 * a running loop's attention — a collection waiting for this thread
 * (above), or, counted by veles_task.c, a cancellation or a failed child's
 * abandonment of a task that is running, or a task that has run long
 * while others wait. One load and an untaken branch otherwise. It has a
 * cache line of its own: every loop of every thread reads it, and a
 * variable written often beside it would make each of those reads a miss.
 * The code reads the first word (veles_attention, veles_tls.h). */
__attribute__((aligned(64))) volatile int32_t veles_attention_line[16];

static void oom(void) {
    fputs("panic: out of memory\n", stderr);
    exit(101);
}

/* ---- arenas --------------------------------------------------------------- */

/* reserves bytes of address space at an ARENA_SIZE boundary, nothing
 * committed (Windows: watched for writes, F18) */
static char *os_reserve(size_t bytes) {
#if defined(_WIN32)
    for (int tries = 0; tries < 16; tries++) {
        /* a larger reservation shows where an aligned one fits; another
         * thread may take that place before the second call, so retry */
        char *p = VirtualAlloc(NULL, bytes + ARENA_SIZE, MEM_RESERVE, PAGE_NOACCESS);
        if (!p) return NULL;
        uintptr_t a = ((uintptr_t)p + ARENA_SIZE - 1) & ~(uintptr_t)(ARENA_SIZE - 1);
        VirtualFree(p, 0, MEM_RELEASE);
        char *q = VirtualAlloc((void *)a, bytes, MEM_RESERVE | MEM_WRITE_WATCH, PAGE_READWRITE);
        if (q) return q;
    }
    return NULL;
#else
    char *p = mmap(NULL, bytes + ARENA_SIZE, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS | MAP_NORESERVE, -1, 0);
    if (p == MAP_FAILED) return NULL;
    uintptr_t a = ((uintptr_t)p + ARENA_SIZE - 1) & ~(uintptr_t)(ARENA_SIZE - 1);
    size_t head = a - (uintptr_t)p, tail = ARENA_SIZE - head;
    if (head) munmap(p, head);
    if (tail) munmap((char *)a + bytes, tail);
    return (char *)a;
#endif
}

/* makes [p, p + n) usable; it reads as zeros */
static int os_commit(char *p, size_t n) {
#if defined(_WIN32)
    return VirtualAlloc(p, n, MEM_COMMIT, PAGE_READWRITE) != NULL;
#else
    (void)p;
    (void)n;
    return 1; /* mapped readable and writable; the kernel gives a page at its first touch */
#endif
}

/* gives the memory of [p, p + n) back; the range stays reserved */
static void os_decommit(char *p, size_t n) {
#if defined(_WIN32)
    VirtualFree(p, n, MEM_DECOMMIT);
#else
    madvise(p, n, MADV_DONTNEED);
#endif
}

/* the lowest and highest address any arena has: most words a conservative
 * scan reads are not pointers, and most of those fall outside, which saves
 * the tables a look */
static uintptr_t heap_lo = UINTPTR_MAX, heap_hi;

/* Under `--sanitize` LeakSanitizer looks for pointers in the memory it
 * knows: the C heap, stacks, globals — not a mapping of ours. The
 * collected objects hold the C heap's blocks too (an executor's condition
 * variable), so each arena is one of its roots. */
#if defined(__has_feature)
#if __has_feature(address_sanitizer)
#define ARENA_LSAN 1
#endif
#endif
#if defined(__SANITIZE_ADDRESS__)
#define ARENA_LSAN 1
#endif
#if defined(ARENA_LSAN)
void __lsan_register_root_region(const void *p, size_t size);
#endif

/* a new arena of bytes (a multiple of ARENA_SIZE), entered in the table at
 * each of its boundaries (heap_lock) */
static veles_arena *arena_new(size_t bytes) {
    char *base = os_reserve(bytes);
    if (!base) return NULL;
#if defined(ARENA_LSAN)
    __lsan_register_root_region(base, bytes);
#endif
    veles_arena *ar = calloc(1, sizeof *ar);
    if (!ar) oom();
    ar->base = base;
    ar->bytes = bytes;
    ar->nslots = bytes >> SPAN_SHIFT;
    ar->slots = calloc(ar->nslots, sizeof *ar->slots);
    ar->taken = calloc(ar->nslots, 1);
    if (!ar->slots || !ar->taken) oom();
    for (size_t off = 0; off < bytes; off += ARENA_SIZE) {
        uintptr_t a = (uintptr_t)base + off;
        size_t i1 = (a >> (ARENA_SHIFT + ARENA_L2_BITS)) & (((size_t)1 << ARENA_L1_BITS) - 1);
        veles_arena **l2 = __atomic_load_n(&arena_l1[i1], __ATOMIC_ACQUIRE);
        if (!l2) {
            l2 = calloc((size_t)1 << ARENA_L2_BITS, sizeof *l2);
            if (!l2) oom();
            __atomic_store_n(&arena_l1[i1], l2, __ATOMIC_RELEASE);
        }
        __atomic_store_n(&l2[(a >> ARENA_SHIFT) & (((size_t)1 << ARENA_L2_BITS) - 1)], ar, __ATOMIC_RELEASE);
    }
    if ((uintptr_t)base < heap_lo) __atomic_store_n(&heap_lo, (uintptr_t)base, __ATOMIC_RELEASE);
    if ((uintptr_t)base + bytes > heap_hi) __atomic_store_n(&heap_hi, (uintptr_t)base + bytes, __ATOMIC_RELEASE);
    ar->next = arenas;
    arenas = ar;
    return ar;
}

/* k free pieces in a row of an arena, taken (heap_lock): the first run that
 * fits, else in a new arena */
static int arena_take(size_t k, veles_arena **out, size_t *at) {
    for (veles_arena *ar = arenas; ar; ar = ar->next) {
        size_t run = 0;
        for (size_t i = ar->free_hint; i < ar->nslots; i++) {
            if (ar->taken[i]) {
                run = 0;
                continue;
            }
            if (++run == k) {
                size_t first = i + 1 - k;
                memset(ar->taken + first, 1, k);
                if (first == ar->free_hint) {
                    while (ar->free_hint < ar->nslots && ar->taken[ar->free_hint]) ar->free_hint++;
                }
                *out = ar;
                *at = first;
                return 1;
            }
        }
    }
    size_t bytes = ((k << SPAN_SHIFT) + ARENA_SIZE - 1) & ~(ARENA_SIZE - 1);
    veles_arena *ar = arena_new(bytes);
    if (!ar) return 0;
    memset(ar->taken, 1, k);
    ar->free_hint = k;
    *out = ar;
    *at = 0;
    return 1;
}

/* the span holding p, or NULL; a marker may call it while an allocator
 * adds arenas and spans (the entries are published ready, and nothing it
 * reads is freed while the program runs) */
static veles_span *span_of(const void *p) {
    uintptr_t a = (uintptr_t)p;
    if (a < __atomic_load_n(&heap_lo, __ATOMIC_RELAXED) || a >= __atomic_load_n(&heap_hi, __ATOMIC_RELAXED)) return NULL;
    veles_arena **l2 = __atomic_load_n(&arena_l1[(a >> (ARENA_SHIFT + ARENA_L2_BITS)) & (((size_t)1 << ARENA_L1_BITS) - 1)], __ATOMIC_ACQUIRE);
    if (!l2) return NULL;
    veles_arena *ar = __atomic_load_n(&l2[(a >> ARENA_SHIFT) & (((size_t)1 << ARENA_L2_BITS) - 1)], __ATOMIC_ACQUIRE);
    if (!ar || a < (uintptr_t)ar->base) return NULL;
    size_t i = (a - (uintptr_t)ar->base) >> SPAN_SHIFT;
    if (i >= ar->nslots) return NULL;
    veles_span *s = __atomic_load_n(&ar->slots[i], __ATOMIC_ACQUIRE);
    /* a released span keeps its slots until the next collection, start NULL */
    if (!s || !s->start || a < (uintptr_t)s->start || a >= (uintptr_t)s->start + s->bytes) return NULL;
    return s;
}

/* ---- spans ---------------------------------------------------------------- */

/* a span of bytes (a multiple of SPAN_SIZE) for objects of objsize, from an
 * arena (heap_lock); its memory reads as zeros */
static veles_span *new_span(int cls, size_t objsize, size_t bytes) {
    veles_arena *ar;
    size_t at;
    if (!arena_take(bytes >> SPAN_SHIFT, &ar, &at)) oom();
    char *start = ar->base + (at << SPAN_SHIFT);
    if (!os_commit(start, bytes)) oom();
    veles_span *s = calloc(1, sizeof *s);
    if (!s) oom();
    s->start = start;
    s->bytes = bytes;
    s->objsize = objsize;
    s->nobjs = bytes / objsize;
    s->marks = calloc(s->nobjs, 1);
    s->used = calloc(s->nobjs, 1);
    if (!s->marks || !s->used) oom();
    s->cls = cls;
    s->arena = ar;
    s->piece = at;
    s->swept = sweep_epoch; /* nothing in it to sweep */
    s->hot = 1;             /* made for an allocator */
    if (nspans == cap_spans) {
        cap_spans = cap_spans ? cap_spans * 2 : 64;
        all_spans = realloc(all_spans, cap_spans * sizeof *all_spans);
        if (!all_spans) oom();
    }
    all_spans[nspans++] = s;
    for (size_t i = 0; i < bytes >> SPAN_SHIFT; i++) __atomic_store_n(&ar->slots[at + i], s, __ATOMIC_RELEASE);
    return s;
}

static void small_classes_init(void) {
    size_t c = 0;
    for (size_t i = 0; i <= SMALL_MAX / 16; i++) {
        while (class_sizes[c] < i * 16) c++;
        small_class[i] = (uint8_t)c;
    }
}

static int class_for(size_t bytes) {
    if (bytes <= SMALL_MAX) return small_class[(bytes + 15) / 16];
    for (size_t i = small_class[SMALL_MAX / 16]; i < NCLASSES; i++) {
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
            s->nused++;
            s->free_hint = i + 1;
            char *obj = s->start + i * s->objsize;
            memset(obj, 0, s->objsize);
            *(veles_desc **)obj = desc;
            /* Published last, so a marker that finds the slot used finds it
             * whole (the helpers may mark while the program runs, F18 c).
             * A new object is not marked: if it is live at the last pause,
             * that pause reaches it from the roots or from a page written
             * since — and garbage made while the helpers marked is free
             * at the next sweep instead of a cycle later. */
            __atomic_store_n(&s->used[i], 1, __ATOMIC_RELEASE);
            return obj + body_offset(desc);
        }
    }
    return NULL;
}

static void heap_acquire(void);
static void collect_now(void);
static void *alloc_slow(veles_desc *desc, size_t need, int cls);
static int sweep_one(veles_span *s);
static void sweep_some(int64_t n);
static void sweep_kick(void);
static void wait_cycle(void);

/* Each thread allocates from spans it owns, one per size class, without a
 * lock (D66): nobody else takes a slot from an owned span, and a
 * collection — the only other writer — runs with every thread stopped and
 * takes the ownership away. What a thread allocated is added to the
 * global count in chunks, so the collection trigger stays close. */
void *veles_gc_alloc(veles_desc *desc, int64_t size) {
    if (size < 0) size = 0;
    size_t off = body_offset(desc);
    size_t need = (size_t)size + off;
    int cls = class_for(need);
    if (off > HEADER) {
        /* over-aligned (D120): a class whose size the alignment divides */
        while (cls >= 0 && class_sizes[cls] % off != 0) {
            cls = (size_t)cls + 1 < NCLASSES ? cls + 1 : -1;
        }
    }
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
/* The cursor moves past the spans that are full or another thread's (one
 * is given back only once it is full), and stops at one a helper is
 * sweeping right now, which may have room after: a claim never walks the
 * same full spans again until a collection frees slots in them. (New spans
 * went to the head of the list, after which a claim walked every full span
 * behind them — while a concurrent collection marked, that was every span
 * of the class, at every claim.) */
static veles_span *classes_tail[NCLASSES];

static veles_span *claim_span(int cls) {
    veles_span *first_busy = NULL;
    for (veles_span *s = cursor[cls] ? cursor[cls] : classes[cls]; s; s = s->next) {
        if (s->owner) continue;
        /* swept first: until then its free slots are not known (F18 a) */
        if (!sweep_one(s)) {
            if (!first_busy) first_busy = s;
            continue;
        }
        if (s->nused < s->nobjs) {
            cursor[cls] = first_busy ? first_busy : s;
            s->hot = 1; /* an allocator writes it from now on (F18 c) */
            return s;
        }
    }
    veles_span *s = new_span(cls, class_sizes[cls], SPAN_SIZE);
    if (classes_tail[cls]) classes_tail[cls]->next = s;
    else classes[cls] = s;
    classes_tail[cls] = s;
    cursor[cls] = first_busy ? first_busy : s;
    return s;
}

/* spans an allocation on the slow path sweeps on its way (Go's
 * proportional sweep): without helpers the sweep is done by the time the
 * next collection comes, and with them it is shared */
#define SWEEP_ASSIST 2

static void *alloc_slow(veles_desc *desc, size_t need, int cls) {
    if (__atomic_load_n(&sweep_pending, __ATOMIC_RELAXED) > 0) sweep_some(SWEEP_ASSIST);
    heap_acquire();
    if (me) {
        allocated_since_gc += me->tl_bytes;
        me->tl_bytes = 0;
    }
    if (gc_enabled && allocated_since_gc > gc_trigger) {
        if (__atomic_load_n(&gc_phase, __ATOMIC_ACQUIRE) != GC_IDLE) {
            /* a cycle marks already: past half again the goal, this
             * thread waits for it rather than grow the heap without bound */
            if (allocated_since_gc > gc_threshold + gc_threshold / 2) {
                veles_lock_release(heap_lock);
                wait_cycle();
                heap_acquire();
            }
        } else {
            veles_lock_release(heap_lock);
            collect_now();
            heap_acquire();
        }
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

/* ---- marking --------------------------------------------------------------
 * The world is stopped while the collector marks, and the threads that ran
 * tasks wait for it. Once the heap is big enough to be worth it, helper
 * threads mark alongside the collecting thread (markers, below), each
 * draining a stack of its own. The helpers sleep until there is work to
 * share: a marker holding more than it needs while helpers sleep hands a
 * batch to the shared list and wakes one — a helper spinning for work slowed
 * the collecting thread down more than it helped. An object is claimed by
 * the marker whose atomic exchange sets its mark byte, and only that one
 * scans it. Marking is over when no marker is working and nothing is
 * shared. (Marking took a third of bench/httphello at 8 threads,
 * 2026-10-10; its graph is narrow, so there it stays on one thread.) */

typedef struct mark_stack {
    char **items;
    size_t n, cap;
    size_t marked; /* bytes of the objects this marker marked: what is live */
} mark_stack;

#define MARK_BATCH 256

typedef struct mark_batch {
    struct mark_batch *next;
    size_t n;
    char *items[MARK_BATCH];
} mark_batch;

static veles_lock *help_lock;      /* the shared list and the markers' state */
static veles_cond *help_cv;        /* helpers: work was shared, or a phase began */
static veles_cond *help_done_cv;   /* the collecting thread: work for it, or the phase is over */
static mark_batch *shared_batches; /* help_lock */
static int64_t shared_count;       /* batches on the list (help_lock; read without it as a hint) */
static int64_t marking;            /* markers working (help_lock; read without it as a hint) */
static int64_t markers_now = 1;    /* markers this collection has: the collecting thread and its helpers */
static int mark_open;              /* the mark phase is on (help_lock) */
static int collector_waiting;      /* help_lock */

static void push_work(mark_stack *ms, char *obj) {
    if (ms->n == ms->cap) {
        ms->cap = ms->cap ? ms->cap * 2 : 4096;
        ms->items = realloc(ms->items, ms->cap * sizeof *ms->items);
        if (!ms->items) oom();
    }
    ms->items[ms->n++] = obj;
}

/* resolves an arbitrary word to a heap object and, if this marker is the
 * first to mark it, queues it for scanning */
static void mark_candidate(mark_stack *ms, uintptr_t word) {
    if (word < 4096) return;
    veles_span *s = span_of((void *)word);
    if (!s) return;
    size_t off = (size_t)(word - (uintptr_t)s->start);
    size_t idx = off / s->objsize;
    /* acquire: a slot an allocator publishes while the helpers mark is
     * whole once it reads as used (alloc_in_span) */
    if (idx >= s->nobjs || !__atomic_load_n(&s->used[idx], __ATOMIC_ACQUIRE)) return;
    if (__atomic_load_n(&s->marks[idx], __ATOMIC_RELAXED) || __atomic_exchange_n(&s->marks[idx], 1, __ATOMIC_RELAXED)) return;
    ms->marked += s->objsize;
    if (!__atomic_load_n(&s->marked_any, __ATOMIC_RELAXED)) __atomic_store_n(&s->marked_any, 1, __ATOMIC_RELAXED);
    push_work(ms, s->start + idx * s->objsize);
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

NO_ASAN static void scan_range(mark_stack *ms, const char *lo, const char *hi) {
    lo = (const char *)(((uintptr_t)lo + 7) & ~(uintptr_t)7);
    for (const char *p = lo; p + 8 <= hi; p += 8) {
        mark_candidate(ms, *(const uintptr_t *)p);
    }
}

static void scan_object(mark_stack *ms, char *obj, size_t objsize) {
    veles_desc *d = *(veles_desc **)obj;
    if (!d || d->nptrs == 0) return;
    size_t off = body_offset(d);
    char *body = obj + off;
    if (d->kind & DESC_ARRAY) {
        if (d->size <= 0) return;
        size_t count = (objsize - off) / (size_t)d->size;
        for (size_t i = 0; i < count; i++) {
            char *el = body + i * d->size;
            for (int64_t k = 0; k < d->nptrs; k++) {
                mark_candidate(ms, *(uintptr_t *)(el + d->offsets[k]));
            }
        }
        return;
    }
    for (int64_t k = 0; k < d->nptrs; k++) {
        if ((size_t)d->offsets[k] + 8 <= objsize - off) {
            mark_candidate(ms, *(uintptr_t *)(body + d->offsets[k]));
        }
    }
}

/* hands a batch of this marker's work to a sleeping helper: half its stack,
 * up to MARK_BATCH, from the bottom — the objects found first, under which
 * the most is left to find (a tree's stack holds a few subtrees at a time,
 * the biggest at the bottom) */
static void share_batch(mark_stack *ms) {
    mark_batch *b = malloc(sizeof *b);
    if (!b) return; /* this marker keeps the work */
    size_t k = ms->n / 2 < MARK_BATCH ? ms->n / 2 : MARK_BATCH;
    memcpy(b->items, ms->items, k * sizeof *b->items);
    memmove(ms->items, ms->items + k, (ms->n - k) * sizeof *ms->items);
    ms->n -= k;
    b->n = k;
    veles_lock_acquire(help_lock);
    b->next = shared_batches;
    shared_batches = b;
    shared_count++;
    veles_cond_signal(help_cv);
    if (collector_waiting) veles_cond_signal(help_done_cv);
    veles_lock_release(help_lock);
}

/* a shared batch into this marker's stack (help_lock held); 0 when none */
static int take_batch(mark_stack *ms) {
    mark_batch *b = shared_batches;
    if (!b) return 0;
    shared_batches = b->next;
    shared_count--;
    for (size_t i = 0; i < b->n; i++) push_work(ms, b->items[i]);
    free(b);
    return 1;
}

/* scans everything on ms and what that reaches, sharing a batch while
 * markers sleep with nothing shared for them */
#define SHARE_FROM 8

static void mark_stack_drain(mark_stack *ms) {
    while (ms->n > 0) {
        if (ms->n >= SHARE_FROM && markers_now > 1 &&
            __atomic_load_n(&marking, __ATOMIC_RELAXED) + __atomic_load_n(&shared_count, __ATOMIC_RELAXED) < markers_now)
            share_batch(ms);
        char *obj = ms->items[--ms->n];
        veles_span *s = span_of(obj);
        scan_object(ms, obj, s->objsize);
    }
}

/* the collecting thread's mark phase, from the roots it found: done when no
 * marker works and nothing is shared */
static void mark_all(mark_stack *ms) {
    if (markers_now == 1) {
        mark_stack_drain(ms);
        return;
    }
    veles_lock_acquire(help_lock);
    marking = 1;
    mark_open = 1;
    veles_lock_release(help_lock);
    for (;;) {
        mark_stack_drain(ms);
        veles_lock_acquire(help_lock);
        marking--;
        while (!shared_batches && marking > 0) {
            collector_waiting = 1;
            veles_cond_wait(help_done_cv, help_lock, -1);
            collector_waiting = 0;
        }
        if (!take_batch(ms)) {
            mark_open = 0;
            veles_lock_release(help_lock);
            return;
        }
        marking++;
        veles_lock_release(help_lock);
    }
}

void veles_capture_regs(void);

/* the collecting thread's own roots: its registers and stack as they are
 * here, recorded the way a safe thread's are (veles_capture_regs); this
 * frame is live while it scans */
static void scan_stack(mark_stack *ms) {
#if defined(VELES_CAPTURE_ASM)
    veles_tls *b = veles_tls_get();
    veles_capture_regs();
    scan_range(ms, (const char *)b->capture, (const char *)(b->capture + VELES_CAPTURE_WORDS));
    char *sp = (char *)(uintptr_t)b->capture[VELES_CAPTURE_SP];
#else
    jmp_buf regs;
    setjmp(regs); /* spill callee-saved registers onto the stack */
    char *sp = (char *)&regs;
    scan_range(ms, (const char *)regs, (const char *)regs + sizeof regs);
#endif
    char *base = me ? me->stack_base : stack_base;
    if (base && sp < base) scan_range(ms, sp, base);
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

/* veles_lock_enter for a lock the collector never takes (the executor's
 * runtime lock): a waiter keeps the lock it gets and waits out a collection
 * holding it, as it leaves the safe region — every thread waiting for it is
 * in a safe region too, so the collection is not held up. Giving the lock
 * back to try again, as veles_lock_enter must (the collector takes the heap
 * lock), hands a contended lock over twice per entry: under many threads
 * that convoy was most of the executor's time (bench/httphello,
 * 2026-10-07). */
void veles_lock_enter_kept(veles_lock *l) {
    if (veles_stop_requested) veles_gc_park();
    if (veles_lock_try(l)) {
        if (!veles_stop_requested || !me || me->safe > 0) return;
        veles_lock_release(l);
    }
    veles_enter_safe();
    veles_lock_acquire(l);
    veles_leave_safe();
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

/* the calling thread is about to end (a closed executor's, D143): its
 * spans go back to the heap and the collector forgets it. It waits out a
 * collection in progress, safe, so as not to leave the list while the
 * collector reads it; holding world_lock, no other can start. */
void veles_thread_detach(void) {
    veles_thread *t = me;
    if (!t) return;
    heap_acquire();
    allocated_since_gc += t->tl_bytes;
    t->tl_bytes = 0;
    for (size_t c = 0; c < NCLASSES; c++) {
        if (t->tl[c]) t->tl[c]->owner = NULL;
        t->tl[c] = NULL;
    }
    veles_lock_release(heap_lock);
    veles_enter_safe();
    veles_lock_acquire(world_lock);
    while (veles_stop_requested) veles_cond_wait(world_cv, world_lock, -1);
    for (veles_thread **pp = &threads; *pp; pp = &(*pp)->next) {
        if (*pp == t) {
            *pp = t->next;
            break;
        }
    }
    veles_lock_release(world_lock);
    veles_tls_get()->thread = NULL;
    free(t);
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
    __atomic_add_fetch(&veles_attention, 1, __ATOMIC_SEQ_CST);
    while (!others_safe()) veles_cond_wait(world_cv, world_lock, -1);
    veles_lock_release(world_lock);
    return 1;
}

static void start_the_world(void) {
    veles_lock_acquire(world_lock);
    __atomic_store_n(&veles_stop_requested, 0, __ATOMIC_SEQ_CST);
    __atomic_sub_fetch(&veles_attention, 1, __ATOMIC_SEQ_CST);
    veles_cond_broadcast(world_cv);
    veles_lock_release(world_lock);
}

/* The hang watchdog's (veles_task.c), from a thread the collector does not
 * know: stops every thread, waiting at most ms for them to reach a safe
 * point — a hung program's are parked, but one spinning in a plain loop
 * never comes — and says whether they did. When they did not, the request
 * is withdrawn and the caller reads the heap as it is. */
int veles_gc_try_stop_world(int64_t ms) {
    int64_t waited = 0;
    veles_lock_acquire(world_lock);
    while (veles_stop_requested && waited < ms) { /* a collection: let it finish */
        veles_cond_wait(world_cv, world_lock, 10);
        waited += 10;
    }
    if (veles_stop_requested) {
        veles_lock_release(world_lock);
        return 0;
    }
    __atomic_store_n(&veles_stop_requested, 1, __ATOMIC_SEQ_CST);
    __atomic_add_fetch(&veles_attention, 1, __ATOMIC_SEQ_CST);
    while (!others_safe() && waited < ms) {
        veles_cond_wait(world_cv, world_lock, 10);
        waited += 10;
    }
    int stopped = others_safe();
    veles_lock_release(world_lock);
    if (!stopped) start_the_world();
    return stopped;
}

void veles_gc_start_world(void) {
    start_the_world();
}

static void scan_threads(mark_stack *ms) {
    for (veles_thread *t = threads; t; t = t->next) {
        scan_range(ms, (const char *)t->tls, (const char *)(t->tls + 1));
        if (t == me) continue;
        scan_range(ms, (const char *)&t->regs, (const char *)&t->regs + sizeof t->regs);
        if (t->stack_top && t->stack_base && t->stack_top < t->stack_base) scan_range(ms, t->stack_top, t->stack_base);
    }
}

static void collect_locked(int full);

int64_t veles_time_monotonic_ns(void);
static void pause_done(int64_t began, int64_t stopped);
static int concurrent_ok(void);
static void cycle_begin(void);
static void cycle_signal(void);
static void cycle_prepare(void);
static void cycle_init(void);

/* a collection this thread's allocation set off: concurrent when it can be
 * (F18 c) — this pause only begins it — else marked and left to sweep */
static void collect_now(void) {
    if (me && me->safe > 0) return; /* a safe thread must not touch the heap */
    if (concurrent_ok()) {
        /* the coordinator starts the cycle — finishes the last sweep,
         * resets the page tracking, stops the world for the roots — while
         * this thread goes on allocating (it waits only past half again
         * the goal) */
        int32_t idle = GC_IDLE;
        if (__atomic_compare_exchange_n(&gc_phase, &idle, GC_STARTING, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) cycle_signal();
        return;
    }
    int64_t began = veles_time_monotonic_ns();
    if (!stop_the_world()) return;
    int64_t stopped = veles_time_monotonic_ns();
    veles_lock_acquire(heap_lock);
    int ran = allocated_since_gc > gc_trigger && __atomic_load_n(&gc_phase, __ATOMIC_ACQUIRE) == GC_IDLE;
    if (ran) collect_locked(0);
    veles_lock_release(heap_lock);
    start_the_world();
    if (ran) {
        pause_done(began, stopped);
        sweep_kick(); /* the helpers sweep while the program runs */
    }
}

void veles_gc_init(void) {
    small_classes_init(); /* before the first allocation, which needs heap_lock made here too */
    world_lock = veles_lock_new();
    world_cv = veles_cond_new();
    heap_lock = veles_lock_new();
    cycle_init();
    veles_thread_attach();
    stack_base = find_stack_base();
    const char *env = getenv("VELES_GC_THRESHOLD");
    if (env) gc_min_threshold = gc_threshold = gc_trigger = (size_t)strtoull(env, NULL, 10);
    tl_chunk = gc_min_threshold / 8;
    if (tl_chunk < 256) tl_chunk = 256;
    if (tl_chunk > (256u << 10)) tl_chunk = 256u << 10;
    if (getenv("VELES_GC_OFF")) gc_enabled = false;
}

/* ---- sweeping ------------------------------------------------------------- */

/* VELES_GC_POISON=1 fills every swept object with 0xCD: a use after a
 * wrong free then crashes at the use instead of reading a new object */
static int gc_poison = -1;

/* sweeps the span s: frees every unmarked object and clears the marks; an
 * empty large-object span gives its memory back at once and is left with
 * no start, for sweep_compact to drop from the list */
static void sweep_span(veles_span *s) {
    size_t live = 0;
    for (size_t k = 0; k < s->nobjs; k++) {
        if (s->used[k] && !s->marks[k]) {
            s->used[k] = 0;
            char *obj = s->start + k * s->objsize;
            veles_desc *d = *(veles_desc **)obj;
            if (d && (d->kind & DESC_WIPE)) veles_wipe(obj + HEADER, s->objsize - HEADER);
            if (gc_poison) memset(obj, 0xCD, s->objsize);
        } else if (s->used[k]) {
            live++;
        }
        s->marks[k] = 0;
    }
    s->free_hint = 0;
    s->marked_any = 0;
    s->nused = live;
    if (live == 0 && s->cls < 0) {
        /* a large object's span gives its memory back now; its place in
         * the arena is freed at the next collection (sweep_compact) */
        char *start = s->start;
        __atomic_store_n(&s->start, NULL, __ATOMIC_RELEASE);
        os_decommit(start, s->bytes);
        free(s->marks);
        free(s->used);
        s->marks = s->used = NULL;
    }
}

/* sweeps s unless it is swept: 1 once it is, 0 while another thread
 * sweeps it (an allocator claiming it, a helper) */
static int sweep_one(veles_span *s) {
    if (__atomic_load_n(&s->swept, __ATOMIC_ACQUIRE) == sweep_epoch) return 1;
    if (__atomic_exchange_n(&s->sweeping, 1, __ATOMIC_ACQ_REL)) return 0;
    if (__atomic_load_n(&s->swept, __ATOMIC_ACQUIRE) != sweep_epoch) {
        sweep_span(s);
        __atomic_store_n(&s->swept, sweep_epoch, __ATOMIC_RELEASE);
        __atomic_sub_fetch(&sweep_pending, 1, __ATOMIC_SEQ_CST);
    }
    __atomic_store_n(&s->sweeping, 0, __ATOMIC_RELEASE);
    return 1;
}

/* threads inside sweep_some (atomic): a collection waits for them to
 * leave before it marks, so none sweeps a span by the new marks */
static int64_t sweepers;

/* sweeps up to n listed spans (n < 0: every one left) */
static void sweep_some(int64_t n) {
    __atomic_add_fetch(&sweepers, 1, __ATOMIC_SEQ_CST);
    while (n != 0) {
        /* an index is taken only below the length published with the
         * list, so none is lost while the list is replaced (sweep_begin) */
        int64_t len = __atomic_load_n(&sweep_list_n, __ATOMIC_ACQUIRE);
        int64_t i = __atomic_load_n(&sweep_list_next, __ATOMIC_ACQUIRE);
        if (i >= len) break;
        if (!__atomic_compare_exchange_n(&sweep_list_next, &i, i + 1, 0, __ATOMIC_SEQ_CST, __ATOMIC_SEQ_CST)) continue;
        sweep_one(sweep_list[i]); /* one being swept elsewhere is that thread's */
        if (n > 0) n--;
    }
    __atomic_sub_fetch(&sweepers, 1, __ATOMIC_SEQ_CST);
}

/* the collection's spans become the ones to sweep (world stopped): marks
 * are final, and every span is unswept until it is claimed or a sweeper
 * reaches it. The list is closed while it is replaced — a helper may be
 * on its way into sweep_some — and its length published last. */
static void sweep_begin(void) {
    static int64_t cap;
    __atomic_store_n(&sweep_list_n, 0, __ATOMIC_SEQ_CST);
    while (__atomic_load_n(&sweepers, __ATOMIC_SEQ_CST) > 0) {
#if defined(_WIN32)
        SwitchToThread();
#else
        sched_yield();
#endif
    }
    if (cap < (int64_t)nspans) {
        cap = (int64_t)nspans * 2;
        free(sweep_list);
        sweep_list = malloc((size_t)cap * sizeof *sweep_list);
        if (!sweep_list) oom();
    }
    memcpy(sweep_list, all_spans, nspans * sizeof *sweep_list);
    __atomic_store_n(&sweep_list_next, 0, __ATOMIC_SEQ_CST);
    __atomic_store_n(&sweep_pending, (int64_t)nspans, __ATOMIC_SEQ_CST);
    __atomic_add_fetch(&sweep_epoch, 1, __ATOMIC_SEQ_CST);
    memset(cursor, 0, sizeof cursor); /* every span may have room again */
    __atomic_store_n(&sweep_list_n, (int64_t)nspans, __ATOMIC_RELEASE);
}


/* every listed span swept (world stopped, heap_lock held): what is left,
 * then a wait for the helpers still sweeping one; the released spans
 * leave the list */
static void sweep_end(void) {
    sweep_some(-1);
    __atomic_store_n(&sweep_list_next, sweep_list_n, __ATOMIC_SEQ_CST); /* closed */
    while (__atomic_load_n(&sweepers, __ATOMIC_SEQ_CST) > 0 || __atomic_load_n(&sweep_pending, __ATOMIC_SEQ_CST) > 0) {
#if defined(_WIN32)
        SwitchToThread();
#else
        sched_yield();
#endif
    }
}

/* the released spans leave the list and the page table (world stopped) */
static void sweep_compact(void) {
    size_t w = 0;
    for (size_t i = 0; i < nspans; i++) {
        veles_span *s = all_spans[i];
        if (s->start) {
            all_spans[w++] = s;
            continue;
        }
        /* its memory went back when it was swept; now its pieces of the
         * arena are free for a new span */
        veles_arena *ar = s->arena;
        size_t k = s->bytes >> SPAN_SHIFT;
        for (size_t j = 0; j < k; j++) __atomic_store_n(&ar->slots[s->piece + j], NULL, __ATOMIC_RELEASE);
        memset(ar->taken + s->piece, 0, k);
        if (s->piece < ar->free_hint) ar->free_hint = s->piece;
        free(s);
    }
    nspans = w;
}

/* ---- markers ---------------------------------------------------------------
 * Helper threads that mark and sweep with the collecting thread while the
 * world is stopped. They run no Veles code and are not among the threads a
 * collection stops. They start with the first collection after which more
 * than HELP_FROM bytes were live, one per core up to MAX_MARKERS markers in
 * all, the collecting thread one of them; VELES_GC_MARKERS sets the count
 * (1: the collecting thread alone), and with it set every collection
 * shares, however small the heap. After a collection they sweep in the
 * background, taking spans from the list one at a time (sweep_some). */
#define MAX_MARKERS 8
#define HELP_FROM ((size_t)1 << 20)

int64_t veles_cpu_count(void);
int64_t veles_thread_spawn_small(void (*fn)(void *), void *arg);
int64_t veles_thread_spawn(void (*fn)(void *), void *arg);

static int markers_max = -1;   /* markers a collection may use */
static int markers_forced;     /* VELES_GC_MARKERS was set */
static int helpers_started;
static int64_t sweep_round;    /* one more per sweep the helpers are asked to join (help_lock) */
static size_t helpers_marked;  /* the bytes the helpers marked this collection (help_lock) */

/* the helpers sweep what is listed, while the program runs */
static void sweep_kick(void) {
    if (!helpers_started) return;
    veles_lock_acquire(help_lock);
    sweep_round++;
    veles_cond_broadcast(help_cv);
    veles_lock_release(help_lock);
}

static void helper_main(void *arg) {
    (void)arg;
    mark_stack ms = {0};
    int64_t swept = 0;
    veles_lock_acquire(help_lock);
    for (;;) {
        if (mark_open && take_batch(&ms)) {
            marking++;
            for (;;) {
                veles_lock_release(help_lock);
                mark_stack_drain(&ms);
                veles_lock_acquire(help_lock);
                if (!take_batch(&ms)) break;
            }
            helpers_marked += ms.marked;
            ms.marked = 0;
            marking--;
            if (marking == 0 && collector_waiting) veles_cond_signal(help_done_cv);
            continue;
        }
        if (swept != sweep_round) {
            swept = sweep_round;
            veles_lock_release(help_lock);
            sweep_some(-1);
            veles_lock_acquire(help_lock);
            continue;
        }
        veles_cond_wait(help_cv, help_lock, -1);
    }
}

/* how many markers this collection has, helpers started as needed */
static int markers_for(void) {
    if (markers_max < 0) {
        const char *env = getenv("VELES_GC_MARKERS");
        int64_t n = env ? strtoll(env, NULL, 10) : veles_cpu_count();
        markers_forced = env != NULL;
        if (n < 1) n = 1;
        if (n > MAX_MARKERS) n = MAX_MARKERS;
        markers_max = (int)n;
    }
    if (markers_max == 1 || (!markers_forced && live_bytes < HELP_FROM)) return 1;
    if (!help_lock) {
        help_lock = veles_lock_new();
        help_cv = veles_cond_new();
        help_done_cv = veles_cond_new();
    }
    while (helpers_started < markers_max - 1) {
        if (veles_thread_spawn_small(helper_main, NULL) != 0) break;
        helpers_started++;
    }
    return 1 + helpers_started;
}

/* ---- D112 test hook -------------------------------------------------------
 * Whether a sweep zeroes a freed object of a wiping descriptor and leaves
 * an ordinary one as it was: 1 yes, 0 no, -1 when a probe object was not
 * freed (the test cannot tell). The probes' addresses are kept masked, so
 * the conservative scan cannot find them on this stack and keep them. */
#define PROBE_MASK ((uintptr_t)0xA5A5A5A5A5A5A5A5u)
static veles_desc plain_u8_desc = {1, DESC_ARRAY, 0};

#if defined(__GNUC__) || defined(__clang__)
__attribute__((noinline))
#endif
static uintptr_t wipe_probe(veles_desc *d) {
    char *p = veles_gc_alloc(d, 48);
    memset(p, 0x5A, 48);
    return (uintptr_t)p ^ PROBE_MASK;
}

static int probe_freed(const char *p) {
    veles_span *s = span_of(p);
    if (!s) return 0;
    size_t k = (size_t)(p - HEADER - s->start) / s->objsize;
    return !s->used[k];
}

void veles_gc_collect(void);

int64_t veles_gc_test_wipe(void) {
    uintptr_t wiped = wipe_probe(&veles_wipe_u8_desc);
    uintptr_t plain = wipe_probe(&plain_u8_desc);
    veles_gc_collect();
    const char *w = (const char *)(wiped ^ PROBE_MASK);
    const char *p = (const char *)(plain ^ PROBE_MASK);
    if (!probe_freed(w) || !probe_freed(p)) return -1;
    for (int i = 0; i < 48; i++) {
        if (w[i] != 0) return 0;          /* the secret's bytes survived */
        if (p[i] != 0x5A) return 0;       /* an ordinary object was touched */
    }
    return 1;
}

/* an explicit collection (tests, the gc example) */
void veles_gc_collect(void) {
    for (;;) {
        /* a concurrent cycle runs to its end first: this one is full */
        if (__atomic_load_n(&gc_phase, __ATOMIC_ACQUIRE) != GC_IDLE) wait_cycle();
        int64_t began = veles_time_monotonic_ns();
        if (!stop_the_world()) continue; /* another thread's ran meanwhile: this one still runs */
        int64_t stopped = veles_time_monotonic_ns();
        veles_lock_acquire(heap_lock);
        if (__atomic_load_n(&gc_phase, __ATOMIC_ACQUIRE) != GC_IDLE) {
            veles_lock_release(heap_lock);
            start_the_world();
            continue;
        }
        collect_locked(1);
        veles_lock_release(heap_lock);
        start_the_world();
        pause_done(began, stopped);
        return;
    }
}

static mark_stack collector_marks; /* the collecting thread's, kept between collections */

/* what the pauses cost: the longest and all of them (from the request to
 * stop to the world started again), and the parts of the last one, for
 * VELES_GC_TRACE and bench/gcpause */
static int64_t pause_max_ns, pause_total_ns, mark_ns, sweep_ns;
static int gc_trace = -1;

static void pause_done(int64_t began, int64_t stopped) {
    int64_t total = veles_time_monotonic_ns() - began;
    pause_total_ns += total;
    if (total > pause_max_ns) pause_max_ns = total;
    if (gc_trace)
        fprintf(stderr, "[gc #%lld] live %zu bytes, %zu spans; pause %.3f ms: stop %.3f, mark %.3f, sweep %.3f\n", (long long)gc_count, live_bytes,
                nspans, total / 1e6, (stopped - began) / 1e6, mark_ns / 1e6, sweep_ns / 1e6);
}

int64_t veles_gc_pause_max_ns(void) {
    return pause_max_ns;
}

int64_t veles_gc_pause_total_ns(void) {
    return pause_total_ns;
}

/* from now on: a benchmark measures the pauses of its steady state */
void veles_gc_pause_reset(void) {
    pause_max_ns = 0;
    pause_total_ns = 0;
}

/* the next collection's goal — twice what is live — and its trigger */
static void set_goal(void) {
    size_t next = live_bytes * 2;
    if (next < gc_min_threshold) next = gc_min_threshold;
    gc_threshold = next;
    size_t early = mark_alloc_last + mark_alloc_last / 5;
    if (early > next / 2) early = next / 2;
    gc_trigger = next - early;
}

/* the registered globals */
static void mark_roots(mark_stack *ms) {
    for (size_t i = 0; i < nroots; i++) {
        veles_desc *d = roots[i].desc;
        char *addr = roots[i].addr;
        if (!d) {
            mark_candidate(ms, *(uintptr_t *)addr);
            continue;
        }
        for (int64_t k = 0; k < d->nptrs; k++) {
            mark_candidate(ms, *(uintptr_t *)(addr + d->offsets[k]));
        }
    }
}

/* a collection, the world stopped and heap_lock held. The last one's sweep
 * is finished first — a span's marks are about to mean this one — then
 * the marking; the sweep is left to the allocators and the helpers while
 * the program runs (F18 a), unless full: veles_gc_collect sweeps every
 * span before it returns, so what it frees is free (tests, bench/idle,
 * D112's wipe). */
static void collect_locked(int full) {
    gc_count++;
    if (gc_poison < 0) gc_poison = getenv("VELES_GC_POISON") != NULL;
    for (veles_thread *t = threads; t; t = t->next) {
        allocated_since_gc += t->tl_bytes;
        t->tl_bytes = 0;
    }
    int64_t ts = veles_time_monotonic_ns();
    sweep_kick();
    sweep_end();
    sweep_compact();
    mark_stack *ms = &collector_marks;
    ms->marked = 0;
    if (help_lock) {
        veles_lock_acquire(help_lock);
        helpers_marked = 0;
        veles_lock_release(help_lock);
    }
    int64_t t0 = veles_time_monotonic_ns();
    mark_roots(ms);
    scan_stack(ms);
    scan_threads(ms);
    markers_now = markers_for();
    mark_all(ms);
    live_bytes = ms->marked;
    if (help_lock) {
        veles_lock_acquire(help_lock);
        live_bytes += helpers_marked;
        veles_lock_release(help_lock);
    }
    int64_t t1 = veles_time_monotonic_ns();
    sweep_begin();
    if (full) {
        sweep_kick();
        sweep_end();
        sweep_compact();
    }
    mark_ns = t1 - t0;
    sweep_ns = (t0 - ts) + (veles_time_monotonic_ns() - t1);
    /* every thread is stopped: the spans are nobody's until claimed again */
    for (veles_thread *t = threads; t; t = t->next) memset(t->tl, 0, sizeof t->tl);
    for (size_t i = 0; i < nspans; i++) all_spans[i]->owner = NULL, all_spans[i]->hot = 0;
    allocated_since_gc = 0;
    set_goal();
    if (gc_trace < 0) gc_trace = getenv("VELES_GC_TRACE") != NULL;
}

/* ---- mostly concurrent marking (F18 c) ---------------------------------------
 * Boehm's mostly-parallel collection, with the OS telling which pages were
 * written. The thread that starts a collection finishes the last sweep and
 * resets the page tracking; a short pause then takes the roots (globals,
 * every thread's stack and registers), and the world runs again while a
 * coordinator thread and the helpers mark from them. Nothing the program
 * does adds marking work — what it writes is on a written page, what it
 * allocates is unmarked until something marked is found to reach it — so
 * "no work left" stays so. Rounds while the program runs rescan the marked
 * objects on pages written meanwhile (Windows). A last short pause rescans
 * the pages written since, the globals and every stack again, and finishes
 * marking: an object live then is reached from the roots by a path whose
 * first unmarked object follows a marked one that was either scanned after
 * its last write or written since — on a page read as written, rescanned.
 * Coroutine frames, the runtime's own C writes and LLVM's spills are all
 * just writes to pages: nothing in the generated code changes.
 *
 * What makes it cheap: a reset page faults at its next write, and an
 * allocator writes every page it fills. So only pages that hold marked
 * objects are reset — a span an allocator took since the last collection
 * (hot) is never, and keeps reading as written; a rescan skips a span with
 * nothing marked (marked_any), which is what an allocator's garbage is.
 * Resetting every page cost scratchpad gcpause (134 MB live, small lists
 * made and dropped) a third of its speed; reset this way it runs faster
 * than with the pauses (690 ms against 805), its pauses 0.6 ms (19 before).
 *
 * The written pages: Windows' write watch on the arenas (MEM_WRITE_WATCH).
 * Linux's soft-dirty bits only when asked (=1) and a self-test at the first
 * collection shows they work (stock kernels have them; WSL2's does not):
 * they reset for the whole process at once. Elsewhere, and on Linux by
 * default, a collection marks in one pause as before.
 * VELES_GC_CONCURRENT=0 keeps that everywhere; =1 marks concurrently however
 * small the heap; =all takes every page as written — slow, for testing the
 * protocol where the OS cannot tell. */
enum { DIRTY_NONE, DIRTY_WATCH, DIRTY_SOFT, DIRTY_ALL };
static int dirty_kind = -1;
static int concurrent_forced;
static size_t dirty_page = 4096;
#define CONCURRENT_FROM ((size_t)4 << 20) /* live bytes below which one pause is cheaper than two */
#define PRECLEAN_ROUNDS 4
#define PRECLEAN_ENOUGH 256 /* pages with something marked: few enough to leave to the last pause */

#if defined(__linux__)
#include <fcntl.h>
#include <unistd.h>
static int pagemap_fd = -1;

static int clear_refs(void) {
    int fd = open("/proc/self/clear_refs", O_WRONLY);
    if (fd < 0) return 0;
    int ok = write(fd, "4", 1) == 1;
    close(fd);
    return ok;
}

static int soft_dirty(const char *p) {
    uint64_t e = 0;
    if (pread(pagemap_fd, &e, sizeof e, (off_t)(((uintptr_t)p / dirty_page) * sizeof e)) != (ssize_t)sizeof e) return 1;
    return (e >> 55) & 1;
}

/* whether this kernel keeps soft-dirty bits: a page written after the
 * reset reads dirty, one not written reads clean */
static int soft_dirty_works(void) {
    pagemap_fd = open("/proc/self/pagemap", O_RDONLY);
    if (pagemap_fd < 0) return 0;
    long ps = sysconf(_SC_PAGESIZE);
    if (ps > 0) dirty_page = (size_t)ps;
    char *p = mmap(NULL, 2 * dirty_page, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (p == MAP_FAILED) return 0;
    volatile char *v = p;
    v[0] = 1;
    v[dirty_page] = 1;
    int ok = clear_refs();
    v[0] = 2;
    ok = ok && soft_dirty(p) && !soft_dirty(p + dirty_page);
    munmap(p, 2 * dirty_page);
    return ok;
}
#endif

static void dirty_init(void) {
    const char *env = getenv("VELES_GC_CONCURRENT");
    concurrent_forced = env && strcmp(env, "1") == 0;
    if (env && strcmp(env, "0") == 0) {
        dirty_kind = DIRTY_NONE;
    } else if (env && strcmp(env, "all") == 0) {
        dirty_kind = DIRTY_ALL;
        concurrent_forced = 1;
    } else {
#if defined(_WIN32)
        dirty_kind = DIRTY_WATCH;
#elif defined(__linux__)
        /* soft-dirty bits are reset for the whole process at once: every
         * page the program writes after it — an allocator writes every page
         * it fills — takes a fault, so a program that allocates much would
         * run a third slower (as with write watch reset everywhere,
         * measured on Windows 2026-10-13). Asked for (=1) only: it suits a
         * large heap the program reads more than it makes. */
        dirty_kind = concurrent_forced && soft_dirty_works() ? DIRTY_SOFT : DIRTY_NONE;
#else
        dirty_kind = DIRTY_NONE;
#endif
    }
}

/* from now on every write to the arenas is seen */
static void dirty_reset(void) {
#if defined(_WIN32)
    if (dirty_kind == DIRTY_WATCH) {
        for (veles_arena *ar = arenas; ar; ar = ar->next) ResetWriteWatch(ar->base, ar->bytes);
    }
#elif defined(__linux__)
    if (dirty_kind == DIRTY_SOFT && !clear_refs()) dirty_kind = DIRTY_ALL; /* lost the bits: take every page */
#endif
}

/* the marked objects that may lie on page p, scanned again: in an array
 * only the elements on it, anything else whole (a large one once a
 * rescan, rescanned) */
static int64_t rescan_round;

static void rescan_page(mark_stack *ms, char *p) {
    veles_span *s = span_of(p);
    /* nothing marked in it: nothing to rescan — most pages an allocator
     * fills between two collections hold garbage only */
    if (!s || !__atomic_load_n(&s->marked_any, __ATOMIC_RELAXED)) return;
    char *end = p + dirty_page;
    size_t lo = p > s->start ? (size_t)(p - s->start) / s->objsize : 0;
    size_t hi = (size_t)(end - 1 - s->start) / s->objsize;
    if (hi >= s->nobjs) hi = s->nobjs - 1;
    for (size_t i = lo; i <= hi; i++) {
        if (!__atomic_load_n(&s->used[i], __ATOMIC_ACQUIRE) || !__atomic_load_n(&s->marks[i], __ATOMIC_RELAXED)) continue;
        char *obj = s->start + i * s->objsize;
        veles_desc *d = *(veles_desc **)obj;
        if (!d || d->nptrs == 0) continue;
        if (d->kind & DESC_ARRAY) {
            if (d->size <= 0) continue;
            char *body = obj + body_offset(d);
            size_t count = (s->objsize - body_offset(d)) / (size_t)d->size;
            size_t first = p > body ? (size_t)(p - body) / (size_t)d->size : 0;
            size_t last = end > body ? ((size_t)(end - body) + (size_t)d->size - 1) / (size_t)d->size : 0;
            if (last > count) last = count;
            for (size_t e = first; e < last; e++) {
                char *el = body + e * (size_t)d->size;
                for (int64_t k = 0; k < d->nptrs; k++) mark_candidate(ms, *(uintptr_t *)(el + d->offsets[k]));
            }
            continue;
        }
        if (s->cls < 0) {
            if (__atomic_exchange_n(&s->rescanned, rescan_round, __ATOMIC_RELAXED) == rescan_round) continue;
        }
        scan_object(ms, obj, s->objsize);
    }
}

/* every marked object with pointers in the arenas (DIRTY_ALL) */
static void rescan_everything(mark_stack *ms) {
    for (veles_arena *ar = __atomic_load_n(&arenas, __ATOMIC_ACQUIRE); ar; ar = ar->next) {
        for (size_t i = 0; i < ar->nslots; i++) {
            veles_span *s = __atomic_load_n(&ar->slots[i], __ATOMIC_ACQUIRE);
            if (!s || !s->start || s->start != ar->base + (i << SPAN_SHIFT)) continue; /* a span once, at its first piece */
            for (size_t k = 0; k < s->nobjs; k++) {
                if (__atomic_load_n(&s->used[k], __ATOMIC_ACQUIRE) && __atomic_load_n(&s->marks[k], __ATOMIC_RELAXED))
                    scan_object(ms, s->start + k * s->objsize, s->objsize);
            }
        }
    }
}

/* the marked objects on pages written since the last reset, scanned again;
 * reset: the pages are clean again (Windows reads and resets at once).
 * Returns how many pages were written. */
static int64_t dirty_rescan(mark_stack *ms, int reset) {
    rescan_round++;
    int64_t pages = 0;
    if (dirty_kind == DIRTY_ALL) {
        rescan_everything(ms);
        return 1 << 30;
    }
#if defined(_WIN32)
    static void **buf;
    static size_t cap;
    for (veles_arena *ar = __atomic_load_n(&arenas, __ATOMIC_ACQUIRE); ar; ar = ar->next) {
        size_t need = ar->bytes / 4096;
        if (cap < need) {
            free(buf);
            buf = malloc(need * sizeof *buf);
            if (!buf) oom();
            cap = need;
        }
        ULONG_PTR count = need;
        DWORD gran = 0;
        if (GetWriteWatch(reset ? WRITE_WATCH_FLAG_RESET : 0, ar->base, ar->bytes, buf, &count, &gran) != 0) {
            /* the OS would not say: every page of it */
            for (size_t off = 0; off < ar->bytes; off += dirty_page) rescan_page(ms, ar->base + off);
            pages += (int64_t)(ar->bytes / dirty_page);
            continue;
        }
        if (gran) dirty_page = gran;
        for (ULONG_PTR i = 0; i < count; i++) rescan_page(ms, buf[i]);
        pages += (int64_t)count;
    }
#elif defined(__linux__)
    (void)reset; /* soft-dirty bits are reset with the next collection's */
    static uint64_t entries[4096];
    for (veles_arena *ar = __atomic_load_n(&arenas, __ATOMIC_ACQUIRE); ar; ar = ar->next) {
        size_t n = ar->bytes / dirty_page;
        for (size_t at = 0; at < n; at += 4096) {
            size_t k = n - at < 4096 ? n - at : 4096;
            off_t off = (off_t)(((uintptr_t)ar->base / dirty_page + at) * sizeof(uint64_t));
            ssize_t got = pread(pagemap_fd, entries, k * sizeof(uint64_t), off);
            for (size_t j = 0; j < k; j++) {
                if (got < (ssize_t)((j + 1) * sizeof(uint64_t)) || ((entries[j] >> 55) & 1)) {
                    rescan_page(ms, ar->base + (at + j) * dirty_page);
                    pages++;
                }
            }
        }
    }
#else
    (void)ms;
    (void)reset;
#endif
    return pages;
}

/* A round while the program runs (Windows): the written pages read, not
 * reset; of them, the ones in spans holding something marked are reset —
 * first, so a write that comes while they are rescanned is seen — and
 * rescanned. What a page holds that is marked is then known as of the
 * reset, and the last pause rescans only the pages written since. Pages
 * whose span holds nothing marked — an allocator's garbage — are neither
 * reset nor rescanned: resetting a page costs the program a fault at its
 * next write, and an allocator writes every page it fills (resetting every
 * written page each round made scratchpad gcpause 35 % slower,
 * 2026-10-13). Returns the pages rescanned. */
static int64_t preclean_round(mark_stack *ms) {
    int64_t pages = 0;
#if defined(_WIN32)
    static void **buf;
    static size_t cap;
    rescan_round++;
    for (veles_arena *ar = __atomic_load_n(&arenas, __ATOMIC_ACQUIRE); ar; ar = ar->next) {
        size_t need = ar->bytes / 4096;
        if (cap < need) {
            free(buf);
            buf = malloc(need * sizeof *buf);
            if (!buf) oom();
            cap = need;
        }
        ULONG_PTR count = need;
        DWORD gran = 0;
        if (GetWriteWatch(0, ar->base, ar->bytes, buf, &count, &gran) != 0) continue; /* the last pause takes them all */
        if (gran) dirty_page = gran;
        /* the pages to take: in spans with something marked; reset in runs */
        size_t keep = 0;
        for (ULONG_PTR i = 0; i < count; i++) {
            veles_span *s = span_of(buf[i]);
            if (s && __atomic_load_n(&s->marked_any, __ATOMIC_RELAXED)) buf[keep++] = buf[i];
        }
        for (size_t i = 0; i < keep;) {
            size_t j = i + 1;
            while (j < keep && (char *)buf[j] == (char *)buf[j - 1] + dirty_page) j++;
            ResetWriteWatch(buf[i], (j - i) * dirty_page);
            i = j;
        }
        for (size_t i = 0; i < keep; i++) rescan_page(ms, buf[i]);
        pages += (int64_t)keep;
    }
#else
    (void)ms;
#endif
    return pages;
}

/* the cycle's coordinator, the program's threads waiting on it, and what
 * the cycle needs to know at its end */
static veles_lock *cycle_lock;
static veles_cond *cycle_cv;
static int cycle_pending;      /* cycle_lock: a cycle's marking is to run */
static size_t cycle_alloc_start;
static int coordinator_started;
static void coordinator_main(void *arg);

/* made with the collector, before any thread runs */
static void cycle_init(void) {
    cycle_lock = veles_lock_new();
    cycle_cv = veles_cond_new();
}

/* whether this collection marks concurrently */
static int concurrent_ok(void) {
    if (dirty_kind < 0) dirty_init();
    if (dirty_kind == DIRTY_NONE) return 0;
    if (!concurrent_forced && (live_bytes < CONCURRENT_FROM || veles_cpu_count() < 2)) return 0;
    if (!coordinator_started) {
        /* a stack of the usual size: the sanitizer's instrumentation needs
         * more than a small one gives */
        if (veles_thread_spawn(coordinator_main, NULL) != 0) return 0;
        coordinator_started = 1;
    }
    return 1;
}

/* the first pause of a concurrent collection (world stopped, heap_lock
 * held): what is left of the last sweep finished, and the roots taken (the
 * tracking was reset before it, cycle_prepare) */
static void cycle_begin(void) {
    gc_count++;
    if (gc_poison < 0) gc_poison = getenv("VELES_GC_POISON") != NULL;
    if (gc_trace < 0) gc_trace = getenv("VELES_GC_TRACE") != NULL;
    for (veles_thread *t = threads; t; t = t->next) {
        allocated_since_gc += t->tl_bytes;
        t->tl_bytes = 0;
    }
    int64_t ts = veles_time_monotonic_ns();
    sweep_kick();
    sweep_end();
    sweep_compact();
    mark_stack *ms = &collector_marks;
    ms->marked = 0;
    if (help_lock) {
        veles_lock_acquire(help_lock);
        helpers_marked = 0;
        veles_lock_release(help_lock);
    }
    int64_t t0 = veles_time_monotonic_ns();
    mark_roots(ms);
    /* the coordinator runs this: it holds no roots of its own, and every
     * thread of the program is stopped, its stack and registers recorded */
    scan_threads(ms);
    cycle_alloc_start = allocated_since_gc;
    __atomic_store_n(&gc_phase, GC_MARKING, __ATOMIC_SEQ_CST);
    sweep_ns = t0 - ts;
    mark_ns = veles_time_monotonic_ns() - t0;
}

/* Windows: the tracking reset only on spans that hold objects (heap_lock).
 * A page reset is write-protected until its next write, which takes a
 * fault — a few microseconds — and a program allocating while the helpers
 * mark fills free pages by the thousand: reset, each of them cost a fault
 * (+40 % on scratchpad gcpause, 2026-10-13). A span with nothing in it
 * keeps reading as written, so what is born there is rescanned by the
 * rounds while the program runs, whose reset of the pages written comes
 * after the allocator has moved on. (An unswept span's count is the one
 * before its sweep: high, so reset.) */
static void dirty_reset_used(void) {
#if defined(_WIN32)
    for (veles_arena *ar = arenas; ar; ar = ar->next) {
        size_t run = 0;
        for (size_t i = 0; i <= ar->nslots; i++) {
            veles_span *s = i < ar->nslots ? ar->slots[i] : NULL;
            int used = s && s->start && s->nused > 0 && !s->hot;
            if (used) {
                run++;
                continue;
            }
            if (run) ResetWriteWatch(ar->base + ((i - run) << SPAN_SHIFT), run << SPAN_SHIFT);
            run = 0;
        }
    }
#endif
}

/* before a concurrent cycle's first pause, by the coordinator while the
 * program runs: the last sweep finished (the helpers sweep too), then the
 * tracking reset — a write after it is seen, whenever it comes */
static void cycle_prepare(void) {
    sweep_kick();
    sweep_some(-1);
    veles_lock_acquire(heap_lock);
    if (dirty_kind == DIRTY_WATCH) dirty_reset_used();
    else dirty_reset();
    veles_lock_release(heap_lock);
}

/* the coordinator's turn to run the marking just begun */
static void cycle_signal(void) {
    veles_lock_acquire(cycle_lock);
    cycle_pending = 1;
    veles_cond_broadcast(cycle_cv);
    veles_lock_release(cycle_lock);
}

/* a thread that allocated far past the trigger while the cycle marks waits
 * for its end, in a safe region (the last pause needs it stopped) */
static int64_t waited_ns, waits; /* what wait_cycle cost the program (VELES_GC_TRACE) */

static void wait_cycle(void) {
    int64_t t = veles_time_monotonic_ns();
    veles_enter_safe();
    veles_lock_acquire(cycle_lock);
    while (__atomic_load_n(&gc_phase, __ATOMIC_ACQUIRE) != GC_IDLE) veles_cond_wait(cycle_cv, cycle_lock, -1);
    veles_lock_release(cycle_lock);
    veles_leave_safe();
    __atomic_add_fetch(&waited_ns, veles_time_monotonic_ns() - t, __ATOMIC_RELAXED);
    __atomic_add_fetch(&waits, 1, __ATOMIC_RELAXED);
}

/* stops every thread from the coordinator, which is not one of them: a
 * collection already stopping them is waited out */
static void stop_world_outside(void) {
    veles_lock_acquire(world_lock);
    while (veles_stop_requested) veles_cond_wait(world_cv, world_lock, 10);
    __atomic_store_n(&veles_stop_requested, 1, __ATOMIC_SEQ_CST);
    __atomic_add_fetch(&veles_attention, 1, __ATOMIC_SEQ_CST);
    while (!others_safe()) veles_cond_wait(world_cv, world_lock, 10);
    veles_lock_release(world_lock);
}

static void coordinator_main(void *arg) {
    (void)arg;
    mark_stack *ms = &collector_marks;
    for (;;) {
        veles_lock_acquire(cycle_lock);
        while (!cycle_pending) veles_cond_wait(cycle_cv, cycle_lock, -1);
        cycle_pending = 0;
        veles_lock_release(cycle_lock);

        /* the first pause: the last sweep finished and the tracking reset
         * before it, so it only takes the roots */
        cycle_prepare();
        int64_t b0 = veles_time_monotonic_ns();
        stop_world_outside();
        int64_t s0 = veles_time_monotonic_ns();
        veles_lock_acquire(heap_lock);
        cycle_begin();
        veles_lock_release(heap_lock);
        start_the_world();
        pause_done(b0, s0);

        /* marking while the program runs */
        int64_t c0 = veles_time_monotonic_ns();
        markers_now = markers_for();
        mark_all(ms);
        int rounds = 0;
        int64_t round_pages = 0;
        if (dirty_kind == DIRTY_WATCH) {
            while (rounds < PRECLEAN_ROUNDS) {
                rounds++;
                round_pages = preclean_round(ms);
                mark_all(ms);
                if (round_pages < PRECLEAN_ENOUGH) break;
            }
        }
        int64_t c1 = veles_time_monotonic_ns();

        /* the last pause */
        int64_t began = veles_time_monotonic_ns();
        stop_world_outside();
        int64_t stopped = veles_time_monotonic_ns();
        veles_lock_acquire(heap_lock);
        for (veles_thread *t = threads; t; t = t->next) {
            allocated_since_gc += t->tl_bytes;
            t->tl_bytes = 0;
        }
        int64_t t0 = veles_time_monotonic_ns();
        int64_t written = dirty_rescan(ms, 0); /* read, not reset: the next cycle resets what it needs */
        mark_roots(ms);
        scan_threads(ms);
        mark_all(ms);
        size_t live = ms->marked;
        if (help_lock) {
            veles_lock_acquire(help_lock);
            live += helpers_marked;
            veles_lock_release(help_lock);
        }
        /* live is what the markers traced */
        live_bytes = live;
        mark_alloc_last = allocated_since_gc - cycle_alloc_start;
        int64_t t1 = veles_time_monotonic_ns();
        sweep_begin();
        for (veles_thread *t = threads; t; t = t->next) memset(t->tl, 0, sizeof t->tl);
        for (size_t i = 0; i < nspans; i++) all_spans[i]->owner = NULL, all_spans[i]->hot = 0;
        allocated_since_gc = 0;
        set_goal();
        mark_ns = t1 - t0;
        sweep_ns = veles_time_monotonic_ns() - t1;
        veles_lock_acquire(cycle_lock);
        __atomic_store_n(&gc_phase, GC_IDLE, __ATOMIC_SEQ_CST);
        veles_cond_broadcast(cycle_cv);
        veles_lock_release(cycle_lock);
        veles_lock_release(heap_lock);
        start_the_world();
        if (gc_trace)
            fprintf(stderr, "[gc #%lld] marked while running for %.3f ms, %d rounds (the last took %lld pages); %lld pages read as written at the end; allocators waited %lld times, %.3f ms in all\n",
                    (long long)gc_count, (c1 - c0) / 1e6, rounds, (long long)round_pages, (long long)written, (long long)waits, waited_ns / 1e6);
        pause_done(began, stopped);
        sweep_kick();
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
