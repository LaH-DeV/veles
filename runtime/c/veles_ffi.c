/* Veles runtime — memory that crosses into C (D69).
 *
 * The collector's memory never becomes a raw pointer C may keep (D50's
 * provenance rule): what C keeps is copied into memory the collector does
 * not manage, and freed explicitly. These are the copies and the
 * allocator behind std/ffi; a C callback's panic guard is here too. */

#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

typedef struct {
    const char *data;
    int64_t len;
} veles_string;

void *veles_alloc(int64_t size);
void veles_panic(const char *msg, int64_t len);

static void *checked(void *p, int64_t n) {
    if (!p && n > 0) {
        static const char msg[] = "out of memory: the C allocator refused a request";
        veles_panic(msg, (int64_t)sizeof msg - 1);
    }
    return p;
}

/* a malloc'd, NUL-terminated copy; the caller has refused interior NULs */
char *veles_ffi_cstring(const char *s, int64_t len) {
    char *buf = checked(malloc((size_t)len + 1), len + 1);
    memcpy(buf, s, (size_t)len);
    buf[len] = 0;
    return buf;
}

/* the offset of the first NUL byte in s, or -1 */
int64_t veles_ffi_nul_at(const char *s, int64_t len) {
    const char *p = memchr(s, 0, (size_t)len);
    return p ? (int64_t)(p - s) : -1;
}

static void set_string(veles_string *out, const char *s, int64_t len) {
    char *buf = veles_alloc(len + 1);
    memcpy(buf, s, (size_t)len);
    buf[len] = 0;
    out->data = buf;
    out->len = len;
}

/* a Veles string from C text up to its NUL */
void veles_ffi_from_cstring(const char *p, veles_string *out) {
    set_string(out, p, (int64_t)strlen(p));
}

/* n bytes from C, as a string buffer (std turns it into a List<u8>) */
void veles_ffi_bytes(const char *p, int64_t n, veles_string *out) {
    set_string(out, p, n);
}

/* zeroed, unmanaged memory */
void *veles_ffi_alloc(int64_t n) {
    return checked(calloc(1, (size_t)(n > 0 ? n : 1)), n);
}

void veles_ffi_free(void *p) {
    free(p);
}

/* ---- callbacks: a panic must not unwind through C frames ----------------
 * An `extern "C" fun` runs between veles_ffi_enter and veles_ffi_leave;
 * veles_panic asks veles_ffi_in_callback before it unwinds, and ends the
 * process instead when C frames are below. The depth is per thread: C
 * frames are below one worker's stack, not another's. */

#include "veles_tls.h"
#define callback_depth (veles_tls_get()->callback_depth)

int64_t veles_callback_enter(void);
void veles_callback_leave(int64_t saved);

/* returns what veles_ffi_leave takes back: the thread's safe-region state
 * while it was in C (veles_gc.c) */
int64_t veles_ffi_enter(void) {
    callback_depth++;
    return veles_callback_enter();
}

void veles_ffi_leave(int64_t saved) {
    callback_depth--;
    veles_callback_leave(saved);
}

int64_t veles_ffi_in_callback(void) {
    return callback_depth;
}

/* ---- handles: a Veles value C holds as an opaque `void *` ----------------
 * A handle is an index into a table the collector scans, so the value stays
 * alive while C holds it and the pointer C sees is not a GC address. Slot 0
 * is never used, so no handle is NULL. */

void veles_gc_root(void *slot, void *desc);
void *veles_alloc_words(int64_t size);
typedef struct veles_lock veles_lock;
veles_lock *veles_lock_new(void);
void veles_lock_enter(veles_lock *l);
void veles_lock_release(veles_lock *l);

static void **handles;      /* the boxed values; a free slot is NULL */
static int64_t handle_cap;
static int64_t handle_free; /* a hint: the lowest slot that may be free */
static veles_lock *handle_lock; /* workers share the table (D66) */

/* at startup, before any thread but the first exists */
void veles_ffi_init(void) {
    handle_lock = veles_lock_new();
    veles_gc_root(&handles, NULL);
}

void *veles_ffi_handle_new(void *box) {
    veles_lock_enter(handle_lock);
    if (!handles) {
        handle_cap = 64;
        handles = veles_alloc_words(handle_cap * (int64_t)sizeof(void *));
        handle_free = 1;
    }
    int64_t i = handle_free;
    while (i < handle_cap && handles[i]) i++;
    if (i == handle_cap) {
        void **bigger = veles_alloc_words(2 * handle_cap * (int64_t)sizeof(void *));
        memcpy(bigger, handles, (size_t)handle_cap * sizeof(void *));
        handles = bigger;
        handle_cap *= 2;
    }
    handles[i] = box;
    handle_free = i + 1;
    veles_lock_release(handle_lock);
    return (void *)(uintptr_t)i;
}

void *veles_ffi_handle_get(void *h) {
    int64_t i = (int64_t)(uintptr_t)h;
    veles_lock_enter(handle_lock);
    void *box = (i > 0 && i < handle_cap) ? handles[i] : NULL;
    veles_lock_release(handle_lock);
    if (!box) {
        static const char msg[] = "an ffi handle that was released, or never made";
        veles_panic(msg, (int64_t)sizeof msg - 1);
    }
    return box;
}

void veles_ffi_handle_release(void *h) {
    int64_t i = (int64_t)(uintptr_t)h;
    veles_lock_enter(handle_lock);
    if (i > 0 && i < handle_cap) {
        handles[i] = NULL;
        if (i < handle_free) handle_free = i;
    }
    veles_lock_release(handle_lock);
}
