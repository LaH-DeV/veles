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
    size_t free_hint;
    struct veles_span *next; /* in size class list */
    int cls;
} veles_span;

static const size_t class_sizes[] = {16, 32, 48, 64, 96, 128, 192, 256, 384, 512, 768, 1024, 1536, 2048, 4096, 8192, 16384, 32768};
#define NCLASSES (sizeof class_sizes / sizeof class_sizes[0])

static veles_span *classes[NCLASSES];
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
    for (size_t n = 0; n < s->nobjs; n++) {
        size_t i = (s->free_hint + n) % s->nobjs;
        if (!s->used[i]) {
            s->used[i] = 1;
            s->free_hint = i + 1;
            char *obj = s->start + i * s->objsize;
            memset(obj, 0, s->objsize);
            *(veles_desc **)obj = desc;
            return obj + HEADER;
        }
    }
    return NULL;
}

void *veles_gc_alloc(veles_desc *desc, int64_t size) {
    if (size < 0) size = 0;
    size_t need = (size_t)size + HEADER;
    if (gc_enabled && allocated_since_gc > gc_threshold) veles_gc_collect();
    allocated_since_gc += need;
    int cls = class_for(need);
    if (cls < 0) {
        size_t bytes = (need + SPAN_SIZE - 1) & ~(SPAN_SIZE - 1);
        veles_span *s = new_span(-1, bytes, bytes);
        s->nobjs = 1;
        return alloc_in_span(s, desc);
    }
    for (veles_span *s = classes[cls]; s; s = s->next) {
        void *p = alloc_in_span(s, desc);
        if (p) return p;
    }
    veles_span *s = new_span(cls, class_sizes[cls], SPAN_SIZE);
    s->next = classes[cls];
    classes[cls] = s;
    void *p = alloc_in_span(s, desc);
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
    if (nroots == cap_roots) {
        cap_roots = cap_roots ? cap_roots * 2 : 32;
        roots = realloc(roots, cap_roots * sizeof *roots);
        if (!roots) oom();
    }
    roots[nroots].addr = addr;
    roots[nroots].desc = desc;
    nroots++;
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

static void scan_range(const char *lo, const char *hi) {
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

static void scan_stack(void) {
    jmp_buf regs;
    setjmp(regs); /* spill callee-saved registers onto the stack */
    char *sp = (char *)&regs;
    scan_range((const char *)regs, (const char *)regs + sizeof regs);
    if (stack_base && sp < stack_base) scan_range(sp, stack_base);
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

void veles_gc_init(void) {
    stack_base = find_stack_base();
    const char *env = getenv("VELES_GC_THRESHOLD");
    if (env) gc_min_threshold = gc_threshold = (size_t)strtoull(env, NULL, 10);
    if (getenv("VELES_GC_OFF")) gc_enabled = false;
}

/* ---- sweeping ------------------------------------------------------------- */

static void sweep(void) {
    live_bytes = 0;
    size_t w = 0;
    for (size_t i = 0; i < nspans; i++) {
        veles_span *s = all_spans[i];
        size_t live = 0;
        for (size_t k = 0; k < s->nobjs; k++) {
            if (s->used[k] && !s->marks[k]) {
                s->used[k] = 0;
            } else if (s->used[k]) {
                live++;
            }
            s->marks[k] = 0;
        }
        s->free_hint = 0;
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

void veles_gc_collect(void) {
    gc_count++;
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
    drain();
    sweep();
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
