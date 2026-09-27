/*
 * Veles runtime — the per-thread block (D66).
 *
 * Everything the runtime keeps per OS thread lives in one struct, reached
 * through veles_tls_get(). On Windows the MinGW toolchain supports only
 * emulated thread-locals, where every access is a function call; the
 * executor and the allocator touch theirs on every operation, so the
 * block is found through a TLS slot of the thread environment block
 * instead: one load from %gs. Elsewhere it is an ordinary thread-local.
 */
#ifndef VELES_TLS_H
#define VELES_TLS_H

#include <stdint.h>
#include <setjmp.h>
#if defined(_WIN32)
#include <intrin.h>
#endif

struct veles_task;
struct veles_thread;

/* the registers VELES_CAPTURE_ASM stores: rbx rbp rdi rsi r12-r15, the
 * caller's stack pointer, xmm6-15 (callee-saved on Windows) — and one
 * word of padding, so the jmp_buf after it stays 16-byte aligned */
#define VELES_CAPTURE_WORDS 30
#define VELES_CAPTURE_SP 8

typedef struct veles_tls {
    /* first, at offset 0, where the assembly below writes it */
    uint64_t capture[VELES_CAPTURE_WORDS];
    jmp_buf panic_return;         /* veles_task.c: where a panicking task unwinds to */
    struct veles_task *task;      /* veles_task.c: the task this thread is running */
    int64_t in_resume;            /* veles_task.c: inside a task's frame */
    int64_t rt_depth;             /* veles_task.c: how often it holds the runtime lock */
    struct veles_worker *worker;  /* veles_task.c: this thread's run queue, if it runs tasks */
    struct veles_thread *thread;  /* veles_gc.c: the collector's record of this thread */
    int64_t lock_tag;             /* veles_sync.c: the tag a Mutex holder leaves in the lock word */
    int64_t callback_depth;       /* veles_ffi.c: C frames below, from calls into Veles */
} veles_tls;

/* allocates the block of a thread on its first use (veles_sync.c) */
veles_tls *veles_tls_first(void);

#if defined(_WIN32) && defined(__x86_64__)
/* the TLS slot holding the block, below 64 so it sits in the TEB itself */
extern uint32_t veles_tls_slot;

static inline veles_tls *veles_tls_get(void) {
    veles_tls *t = (veles_tls *)__readgsqword(0x1480 + veles_tls_slot * 8);
    return t ? t : veles_tls_first();
}
#else
extern __thread veles_tls *veles_tls_block;

static inline veles_tls *veles_tls_get(void) {
    veles_tls *t = veles_tls_block;
    return t ? t : veles_tls_first();
}
#endif

/* VELES_CAPTURE_ASM stores the calling function's callee-saved registers
 * and stack pointer in its thread's block.capture, exactly as they are at
 * the call: it runs at the entry of a naked function, before any code
 * that could save one of them in a frame of its own and reuse it. A
 * thread becoming safe is scanned from this record (veles_gc.c), and the
 * frame that called stays live for as long as the thread is safe. A thread
 * with no block yet has nothing to record. Clobbers rax and r10 only. */
#if defined(__x86_64__) && defined(_WIN32)
#define VELES_CAPTURE_ASM                        \
    "movl veles_tls_slot(%rip), %eax\n\t"        \
    "movq %gs:0x1480(,%rax,8), %rax\n\t"         \
    "testq %rax, %rax\n\t"                       \
    "jz 1f\n\t"                                  \
    "movq %rbx, 0(%rax)\n\t"                     \
    "movq %rbp, 8(%rax)\n\t"                     \
    "movq %rdi, 16(%rax)\n\t"                    \
    "movq %rsi, 24(%rax)\n\t"                    \
    "movq %r12, 32(%rax)\n\t"                    \
    "movq %r13, 40(%rax)\n\t"                    \
    "movq %r14, 48(%rax)\n\t"                    \
    "movq %r15, 56(%rax)\n\t"                    \
    "leaq 8(%rsp), %r10\n\t"                     \
    "movq %r10, 64(%rax)\n\t"                    \
    "movdqu %xmm6, 72(%rax)\n\t"                 \
    "movdqu %xmm7, 88(%rax)\n\t"                 \
    "movdqu %xmm8, 104(%rax)\n\t"                \
    "movdqu %xmm9, 120(%rax)\n\t"                \
    "movdqu %xmm10, 136(%rax)\n\t"               \
    "movdqu %xmm11, 152(%rax)\n\t"               \
    "movdqu %xmm12, 168(%rax)\n\t"               \
    "movdqu %xmm13, 184(%rax)\n\t"               \
    "movdqu %xmm14, 200(%rax)\n\t"               \
    "movdqu %xmm15, 216(%rax)\n\t"               \
    "1:\n\t"
#elif defined(__x86_64__)
#define VELES_CAPTURE_ASM                        \
    "movq veles_tls_block@gottpoff(%rip), %rax\n\t" \
    "movq %fs:(%rax), %rax\n\t"                  \
    "testq %rax, %rax\n\t"                       \
    "jz 1f\n\t"                                  \
    "movq %rbx, 0(%rax)\n\t"                     \
    "movq %rbp, 8(%rax)\n\t"                     \
    "movq %r12, 32(%rax)\n\t"                    \
    "movq %r13, 40(%rax)\n\t"                    \
    "movq %r14, 48(%rax)\n\t"                    \
    "movq %r15, 56(%rax)\n\t"                    \
    "leaq 8(%rsp), %r10\n\t"                     \
    "movq %r10, 64(%rax)\n\t"                    \
    "1:\n\t"
#endif

#endif
