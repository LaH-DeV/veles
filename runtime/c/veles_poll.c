/*
 * Veles bootstrap runtime — the reactor: when sockets are ready, for the
 * executor's socket waits (veles_task.c; plan E3).
 *
 * A task whose socket call would block parks, and the executor arms the
 * descriptor here for reading, writing or both — the union of what the
 * tasks parked on it want. Arming is one-shot: a descriptor reported ready
 * is disarmed until the executor arms it again for the tasks still parked
 * on it. One thread at a time waits for readiness, for as long as it has
 * nothing else to do, and any thread can interrupt that wait.
 *
 * Backends:
 *   Linux    epoll with EPOLLONESHOT; an eventfd interrupts a wait.
 *   Windows  poll requests to the AFD driver (\Device\Afd), the readiness
 *            interface WSAPoll and select are built on, completing on an
 *            I/O completion port — as libuv, mio and wepoll use it. A
 *            posted packet interrupts a wait.
 *   others   poll() over the armed descriptors with a self-pipe (macOS
 *            until plan A7 brings kqueue). Linux builds it too when
 *            VELES_POLL_FALLBACK is defined, which is how it is tested.
 *
 * Before this, every wait rebuilt a pollfd array of every parked socket
 * under the runtime lock: O(parked) per wait, and with several threads the
 * lock was what they did (bench/httphello: 20× Go, more threads slower).
 *
 * Thread safety: every function may be called from any thread; each
 * backend guards its own state. The executor waits from one thread at a
 * time (poller_busy), and arms and forgets while that wait is in progress.
 */
/* glibc declares its extensions (pthread_getattr_np, ...) only when asked. */
#if defined(__linux__) && !defined(_GNU_SOURCE)
#define _GNU_SOURCE
#endif
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <errno.h>

#if defined(_WIN32)
#include <winsock2.h>
#include <windows.h>
#include <winternl.h>
#elif defined(__linux__) && !defined(VELES_POLL_FALLBACK)
#define VELES_POLL_EPOLL 1
#include <unistd.h>
#include <sys/epoll.h>
#include <sys/eventfd.h>
#include <sys/syscall.h>
#include <time.h>
#else
#include <unistd.h>
#include <fcntl.h>
#include <poll.h>
#include <pthread.h>
#endif

/* what veles_poll_wait reports: a descriptor and which ways it is ready */
typedef struct veles_poll_event {
    int64_t fd;
    int32_t read;
    int32_t write;
} veles_poll_event;

static void poll_fatal(const char *what) {
    fprintf(stderr, "veles: the socket reactor failed: %s\n", what);
    exit(101);
}

/* a wake is pending: set by veles_poll_wake, cleared by the wait that
 * consumes it, so many wakes cost one system call */
static int64_t wake_pending;

#if defined(_WIN32)
/* ---- Windows: AFD poll requests on an I/O completion port ------------------ */

#define AFD_POLL_RECEIVE      0x0001
#define AFD_POLL_SEND         0x0004
#define AFD_POLL_DISCONNECT   0x0008
#define AFD_POLL_ABORT        0x0010
#define AFD_POLL_LOCAL_CLOSE  0x0020
#define AFD_POLL_ACCEPT       0x0080
#define AFD_POLL_CONNECT_FAIL 0x0100
#define IOCTL_AFD_POLL        0x00012024

#define AFD_READ  (AFD_POLL_RECEIVE | AFD_POLL_ACCEPT | AFD_POLL_DISCONNECT | AFD_POLL_ABORT | AFD_POLL_CONNECT_FAIL)
#define AFD_WRITE (AFD_POLL_SEND | AFD_POLL_ABORT | AFD_POLL_CONNECT_FAIL)

#ifndef SIO_BASE_HANDLE
#define SIO_BASE_HANDLE 0x48000022
#endif
#ifndef SIO_BSP_HANDLE_SELECT
#define SIO_BSP_HANDLE_SELECT 0x4800001C
#endif
#ifndef SIO_BSP_HANDLE_POLL
#define SIO_BSP_HANDLE_POLL 0x4800001D
#endif
#ifndef STATUS_CANCELLED
#define STATUS_CANCELLED ((NTSTATUS)0xC0000120)
#endif
#ifndef STATUS_SUCCESS
#define STATUS_SUCCESS ((NTSTATUS)0x00000000)
#endif
#ifndef STATUS_PENDING
#define STATUS_PENDING ((NTSTATUS)0x00000103)
#endif

typedef struct {
    HANDLE handle;
    ULONG events;
    NTSTATUS status;
} afd_poll_handle_info;

typedef struct {
    LARGE_INTEGER timeout;
    ULONG count;
    ULONG exclusive;
    afd_poll_handle_info handles[1];
} afd_poll_info;

typedef NTSTATUS(NTAPI *nt_create_file_fn)(PHANDLE, ACCESS_MASK, POBJECT_ATTRIBUTES, PIO_STATUS_BLOCK, PLARGE_INTEGER,
                                           ULONG, ULONG, ULONG, ULONG, PVOID, ULONG);
typedef NTSTATUS(NTAPI *nt_ioctl_fn)(HANDLE, HANDLE, PIO_APC_ROUTINE, PVOID, PIO_STATUS_BLOCK, ULONG, PVOID, ULONG, PVOID,
                                     ULONG);
typedef NTSTATUS(NTAPI *nt_cancel_fn)(HANDLE, PIO_STATUS_BLOCK, PIO_STATUS_BLOCK);

static nt_ioctl_fn nt_ioctl;
static nt_cancel_fn nt_cancel;
static HANDLE iocp, afd;

/* One socket's poll request. The kernel writes into iosb and info until
 * the request completes, so the record lives (in malloc memory) until its
 * completion is read, even after the socket is forgotten. */
typedef struct sock_state {
    IO_STATUS_BLOCK iosb; /* first: a completion's OVERLAPPED pointer is this record */
    afd_poll_info info;
    SOCKET sock;
    HANDLE base;          /* the socket under any layered provider */
    ULONG pending;        /* events of the request in flight; 0 = none */
    ULONG wanted;         /* events the executor armed */
    int cancelling;
    int forgotten;
} sock_state;

/* sockets → records: open addressing, linear probing, no deletions in
 * place (a removal re-inserts the rest of its run). The table is split in
 * shards by the socket's hash, each under a critical section of its own:
 * every socket wait arms its socket here, and one lock for all of them was
 * found taken by every other arm at 32 threads (2026-10-10). A record's
 * fields are its shard's, the completion included. */
typedef struct poll_shard {
    CRITICAL_SECTION lock;
    sock_state **table;
    size_t cap, len;
} poll_shard;

#define POLL_SHARDS 32
static poll_shard shards[POLL_SHARDS];

static uint64_t sock_hash(SOCKET s) {
    return (uint64_t)s * 0x9E3779B97F4A7C15ull;
}

static poll_shard *shard_of(SOCKET s) {
    return &shards[(sock_hash(s) >> 59) & (POLL_SHARDS - 1)];
}

static size_t slot_of(poll_shard *sh, SOCKET s) {
    return (size_t)(sock_hash(s) >> 32) & (sh->cap - 1);
}

static sock_state *table_find(poll_shard *sh, SOCKET s) {
    if (!sh->cap) return NULL;
    for (size_t i = slot_of(sh, s);; i = (i + 1) & (sh->cap - 1)) {
        if (!sh->table[i]) return NULL;
        if (sh->table[i]->sock == s) return sh->table[i];
    }
}

static void table_put(poll_shard *sh, sock_state *st);

static void table_grow(poll_shard *sh) {
    sock_state **old = sh->table;
    size_t old_cap = sh->cap;
    sh->cap = sh->cap ? sh->cap * 2 : 16;
    sh->table = calloc(sh->cap, sizeof *sh->table);
    if (!sh->table) poll_fatal("out of memory");
    sh->len = 0;
    for (size_t i = 0; i < old_cap; i++) {
        if (old[i]) table_put(sh, old[i]);
    }
    free(old);
}

static void table_put(poll_shard *sh, sock_state *st) {
    if ((sh->len + 1) * 2 > sh->cap) table_grow(sh);
    size_t i = slot_of(sh, st->sock);
    while (sh->table[i]) i = (i + 1) & (sh->cap - 1);
    sh->table[i] = st;
    sh->len++;
}

static void table_remove(poll_shard *sh, SOCKET s) {
    if (!sh->cap) return;
    size_t i = slot_of(sh, s);
    for (;; i = (i + 1) & (sh->cap - 1)) {
        if (!sh->table[i]) return;
        if (sh->table[i]->sock == s) break;
    }
    sh->table[i] = NULL;
    sh->len--;
    for (size_t j = (i + 1) & (sh->cap - 1); sh->table[j]; j = (j + 1) & (sh->cap - 1)) {
        sock_state *st = sh->table[j];
        sh->table[j] = NULL;
        sh->len--;
        table_put(sh, st);
    }
}

void veles_poll_init(void) {
    if (iocp) return;
    HMODULE ntdll = GetModuleHandleW(L"ntdll.dll");
    nt_create_file_fn nt_create = ntdll ? (nt_create_file_fn)(void *)GetProcAddress(ntdll, "NtCreateFile") : NULL;
    nt_ioctl = ntdll ? (nt_ioctl_fn)(void *)GetProcAddress(ntdll, "NtDeviceIoControlFile") : NULL;
    nt_cancel = ntdll ? (nt_cancel_fn)(void *)GetProcAddress(ntdll, "NtCancelIoFileEx") : NULL;
    if (!nt_create || !nt_ioctl || !nt_cancel) poll_fatal("ntdll has no NtCreateFile/NtDeviceIoControlFile/NtCancelIoFileEx");
    for (int i = 0; i < POLL_SHARDS; i++) InitializeCriticalSectionAndSpinCount(&shards[i].lock, 1000);
    HANDLE port = CreateIoCompletionPort(INVALID_HANDLE_VALUE, NULL, 0, 0);
    if (!port) poll_fatal("CreateIoCompletionPort");
    static WCHAR name[] = L"\\Device\\Afd\\Veles";
    UNICODE_STRING uname = {(USHORT)(sizeof name - sizeof(WCHAR)), (USHORT)sizeof name, name};
    OBJECT_ATTRIBUTES attr = {.Length = sizeof attr, .ObjectName = &uname};
    IO_STATUS_BLOCK iosb;
    HANDLE h;
    NTSTATUS st = nt_create(&h, SYNCHRONIZE, &attr, &iosb, NULL, 0, FILE_SHARE_READ | FILE_SHARE_WRITE, FILE_OPEN, 0, NULL, 0);
    if (st != STATUS_SUCCESS) poll_fatal("cannot open \\Device\\Afd");
    if (!CreateIoCompletionPort(h, port, 0, 0)) poll_fatal("cannot attach \\Device\\Afd to the completion port");
    SetFileCompletionNotificationModes(h, FILE_SKIP_SET_EVENT_ON_HANDLE);
    afd = h;
    iocp = port;
}

/* sends st's poll request for st->wanted; 0, or nonzero if AFD refused it */
static int submit(sock_state *st) {
    ULONG events = AFD_POLL_LOCAL_CLOSE;
    if (st->wanted & 1) events |= AFD_READ;
    if (st->wanted & 2) events |= AFD_WRITE;
    st->info.timeout.QuadPart = INT64_MAX;
    st->info.count = 1;
    st->info.exclusive = FALSE;
    st->info.handles[0].handle = st->base;
    st->info.handles[0].status = 0;
    st->info.handles[0].events = events;
    st->iosb.Status = STATUS_PENDING;
    NTSTATUS s = nt_ioctl(afd, NULL, NULL, &st->iosb, &st->iosb, IOCTL_AFD_POLL, &st->info, sizeof st->info, &st->info,
                          sizeof st->info);
    /* completed at once or not, the completion is queued to the port */
    if (s != STATUS_SUCCESS && s != STATUS_PENDING) return 1;
    st->pending = st->wanted;
    return 0;
}

int64_t veles_poll_arm(int64_t fd, int64_t read, int64_t write) {
    ULONG wanted = (read ? 1u : 0u) | (write ? 2u : 0u);
    poll_shard *sh = shard_of((SOCKET)fd);
    EnterCriticalSection(&sh->lock);
    sock_state *st = table_find(sh, (SOCKET)fd);
    if (!st) {
        /* the base socket under any layered provider; the two others are
         * what such a provider may answer instead (wepoll's order) */
        static const DWORD ioctls[] = {SIO_BASE_HANDLE, SIO_BSP_HANDLE_POLL, SIO_BSP_HANDLE_SELECT};
        HANDLE base = NULL;
        DWORD bytes;
        for (int i = 0; i < 3 && !base; i++) {
            HANDLE h;
            if (WSAIoctl((SOCKET)fd, ioctls[i], NULL, 0, &h, sizeof h, &bytes, NULL, NULL) == 0 && h != (HANDLE)INVALID_SOCKET) base = h;
        }
        if (!base) {
            LeaveCriticalSection(&sh->lock);
            return 1; /* not a socket: the caller treats it as ready */
        }
        st = calloc(1, sizeof *st);
        if (!st) poll_fatal("out of memory");
        st->sock = (SOCKET)fd;
        st->base = base;
        table_put(sh, st);
    }
    st->wanted = wanted;
    int failed = 0;
    if (st->pending) {
        /* a request in flight that covers what is wanted reports it; one
         * that does not is cancelled, and its completion sends the new one */
        if ((st->pending & wanted) != wanted && !st->cancelling) {
            IO_STATUS_BLOCK cancel;
            st->cancelling = 1;
            nt_cancel(afd, &st->iosb, &cancel);
        }
    } else if (wanted) {
        failed = submit(st);
    }
    LeaveCriticalSection(&sh->lock);
    return failed;
}

void veles_poll_forget(int64_t fd) {
    if (!iocp) return;
    poll_shard *sh = shard_of((SOCKET)fd);
    EnterCriticalSection(&sh->lock);
    sock_state *st = table_find(sh, (SOCKET)fd);
    if (st) {
        table_remove(sh, (SOCKET)fd);
        if (st->pending) {
            /* its completion frees it; the close completes it anyway */
            IO_STATUS_BLOCK cancel;
            st->forgotten = 1;
            nt_cancel(afd, &st->iosb, &cancel);
        } else {
            free(st);
        }
    }
    LeaveCriticalSection(&sh->lock);
}

void veles_poll_wake(void) {
    if (__atomic_exchange_n(&wake_pending, 1, __ATOMIC_SEQ_CST)) return;
    PostQueuedCompletionStatus(iocp, 0, 0, NULL);
}

int64_t veles_poll_wait(int64_t timeout_ns, veles_poll_event *out, int64_t max) {
    OVERLAPPED_ENTRY entries[128];
    ULONG got = 0;
    if (max > 128) max = 128;
    int64_t timeout_ms = timeout_ns < 0 ? -1 : (timeout_ns + 999999) / 1000000; /* whole milliseconds, rounded up */
    DWORD ms = timeout_ms < 0 ? INFINITE : (timeout_ms > 0x7ffffffe ? 0x7ffffffe : (DWORD)timeout_ms);
    if (!GetQueuedCompletionStatusEx(iocp, entries, (ULONG)max, &got, ms, FALSE)) return 0;
    int64_t n = 0;
    for (ULONG i = 0; i < got; i++) {
        if (!entries[i].lpOverlapped) {
            __atomic_store_n(&wake_pending, 0, __ATOMIC_SEQ_CST);
            continue;
        }
        sock_state *st = (sock_state *)entries[i].lpOverlapped;
        poll_shard *sh = shard_of(st->sock);
        EnterCriticalSection(&sh->lock);
        st->pending = 0;
        st->cancelling = 0;
        if (st->forgotten) {
            LeaveCriticalSection(&sh->lock);
            free(st);
            continue;
        }
        NTSTATUS s = st->iosb.Status;
        if (s == STATUS_CANCELLED) {
            /* cancelled to change what it waits for */
            if (st->wanted && submit(st)) {
                out[n++] = (veles_poll_event){(int64_t)st->sock, 1, 1};
                st->wanted = 0;
            }
            LeaveCriticalSection(&sh->lock);
            continue;
        }
        ULONG ev = 0;
        if (s < 0) {
            ev = AFD_READ | AFD_WRITE; /* the request failed: let the retried call say why */
        } else if (st->info.count >= 1) {
            ev = st->info.handles[0].events;
        }
        if (ev & AFD_POLL_LOCAL_CLOSE) {
            /* closed without being forgotten: the record goes */
            table_remove(sh, st->sock);
            out[n++] = (veles_poll_event){(int64_t)st->sock, 1, 1};
            LeaveCriticalSection(&sh->lock);
            free(st);
            continue;
        }
        int r = (ev & AFD_READ) != 0, w = (ev & AFD_WRITE) != 0;
        if (!r && !w) {
            if (st->wanted && submit(st)) {
                out[n++] = (veles_poll_event){(int64_t)st->sock, 1, 1};
                st->wanted = 0;
            }
            LeaveCriticalSection(&sh->lock);
            continue;
        }
        st->wanted = 0; /* one-shot: the executor arms it again */
        out[n++] = (veles_poll_event){(int64_t)st->sock, r, w};
        LeaveCriticalSection(&sh->lock);
    }
    return n;
}

#elif defined(VELES_POLL_EPOLL)
/* ---- Linux: epoll ----------------------------------------------------------- */

static int epfd = -1, evfd = -1;
#define WAKE_TAG UINT64_MAX

void veles_poll_init(void) {
    if (epfd >= 0) return;
    int e = epoll_create1(EPOLL_CLOEXEC);
    if (e < 0) poll_fatal("epoll_create1");
    int w = eventfd(0, EFD_NONBLOCK | EFD_CLOEXEC);
    if (w < 0) poll_fatal("eventfd");
    struct epoll_event ev = {.events = EPOLLIN, .data.u64 = WAKE_TAG};
    if (epoll_ctl(e, EPOLL_CTL_ADD, w, &ev) != 0) poll_fatal("epoll_ctl on the eventfd");
    evfd = w;
    epfd = e;
}

int64_t veles_poll_arm(int64_t fd, int64_t read, int64_t write) {
    struct epoll_event ev = {.events = EPOLLONESHOT, .data.u64 = (uint64_t)fd};
    if (read) ev.events |= EPOLLIN | EPOLLRDHUP;
    if (write) ev.events |= EPOLLOUT;
    for (int tries = 0; tries < 2; tries++) {
        if (epoll_ctl(epfd, EPOLL_CTL_MOD, (int)fd, &ev) == 0) return 0;
        if (errno != ENOENT) break;
        if (epoll_ctl(epfd, EPOLL_CTL_ADD, (int)fd, &ev) == 0) return 0;
        if (errno != EEXIST) break;
    }
    return errno ? errno : 1; /* EPERM for a file epoll cannot watch: the caller treats it as ready */
}

void veles_poll_forget(int64_t fd) {
    if (epfd < 0) return;
    struct epoll_event ev = {0};
    epoll_ctl(epfd, EPOLL_CTL_DEL, (int)fd, &ev);
}

void veles_poll_wake(void) {
    if (__atomic_exchange_n(&wake_pending, 1, __ATOMIC_SEQ_CST)) return;
    uint64_t one = 1;
    ssize_t r = write(evfd, &one, sizeof one);
    (void)r;
}

/* epoll_pwait2 (Linux 5.11) takes the timeout in nanoseconds; an older
 * kernel answers ENOSYS once, and the wait falls back to epoll_wait's
 * milliseconds, rounded up (F9) */
static int no_pwait2;

static int epoll_wait_ns(struct epoll_event *evs, int max, int64_t timeout_ns) {
#if defined(SYS_epoll_pwait2)
    if (!__atomic_load_n(&no_pwait2, __ATOMIC_RELAXED)) {
        struct timespec ts, *tp = NULL;
        if (timeout_ns >= 0) {
            ts.tv_sec = timeout_ns / 1000000000;
            ts.tv_nsec = timeout_ns % 1000000000;
            tp = &ts;
        }
        long got = syscall(SYS_epoll_pwait2, epfd, evs, max, tp, NULL, 0);
        if (got >= 0 || errno != ENOSYS) return (int)got;
        __atomic_store_n(&no_pwait2, 1, __ATOMIC_RELAXED);
    }
#endif
    int64_t ms = timeout_ns < 0 ? -1 : (timeout_ns + 999999) / 1000000;
    if (ms > 0x7fffffff) ms = 0x7fffffff;
    return epoll_wait(epfd, evs, max, (int)ms);
}

int64_t veles_poll_wait(int64_t timeout_ns, veles_poll_event *out, int64_t max) {
    struct epoll_event evs[128];
    if (max > 128) max = 128;
    int got = epoll_wait_ns(evs, (int)max, timeout_ns);
    int64_t n = 0;
    for (int i = 0; i < got; i++) {
        if (evs[i].data.u64 == WAKE_TAG) {
            uint64_t v;
            ssize_t r = read(evfd, &v, sizeof v);
            (void)r;
            __atomic_store_n(&wake_pending, 0, __ATOMIC_SEQ_CST);
            continue;
        }
        uint32_t e = evs[i].events;
        out[n].fd = (int64_t)evs[i].data.u64;
        out[n].read = (e & (EPOLLIN | EPOLLRDHUP | EPOLLHUP | EPOLLERR)) != 0;
        out[n].write = (e & (EPOLLOUT | EPOLLHUP | EPOLLERR)) != 0;
        n++;
    }
    return n;
}

#else
/* ---- elsewhere: poll() over the armed set ------------------------------------ */

typedef struct {
    int fd;
    short events;
} armed;

static pthread_mutex_t poll_mutex = PTHREAD_MUTEX_INITIALIZER;
static armed *items;
static size_t nitems, cap_items;
static int wake_pipe[2] = {-1, -1};
static int64_t waiting; /* threads inside poll(): an arm must interrupt them */

void veles_poll_init(void) {
    pthread_mutex_lock(&poll_mutex);
    if (wake_pipe[0] < 0) {
        if (pipe(wake_pipe) != 0) poll_fatal("pipe");
        for (int i = 0; i < 2; i++) {
            fcntl(wake_pipe[i], F_SETFL, fcntl(wake_pipe[i], F_GETFL) | O_NONBLOCK);
            fcntl(wake_pipe[i], F_SETFD, FD_CLOEXEC);
        }
    }
    pthread_mutex_unlock(&poll_mutex);
}

void veles_poll_wake(void) {
    if (__atomic_exchange_n(&wake_pending, 1, __ATOMIC_SEQ_CST)) return;
    char c = 1;
    ssize_t r = write(wake_pipe[1], &c, 1);
    (void)r;
}

static size_t find_item(int fd) {
    for (size_t i = 0; i < nitems; i++) {
        if (items[i].fd == fd) return i;
    }
    return (size_t)-1;
}

int64_t veles_poll_arm(int64_t fd, int64_t read, int64_t write) {
    short events = (short)((read ? POLLIN : 0) | (write ? POLLOUT : 0));
    pthread_mutex_lock(&poll_mutex);
    size_t i = find_item((int)fd);
    if (i == (size_t)-1) {
        if (nitems == cap_items) {
            cap_items = cap_items ? cap_items * 2 : 64;
            armed *grown = realloc(items, cap_items * sizeof *items);
            if (!grown) poll_fatal("out of memory");
            items = grown;
        }
        i = nitems++;
        items[i].fd = (int)fd;
    }
    items[i].events = events;
    int interrupt = __atomic_load_n(&waiting, __ATOMIC_SEQ_CST) > 0;
    pthread_mutex_unlock(&poll_mutex);
    if (interrupt) veles_poll_wake(); /* a poll in progress does not watch it yet */
    return 0;
}

void veles_poll_forget(int64_t fd) {
    pthread_mutex_lock(&poll_mutex);
    size_t i = find_item((int)fd);
    if (i != (size_t)-1) items[i] = items[--nitems];
    pthread_mutex_unlock(&poll_mutex);
}

int64_t veles_poll_wait(int64_t timeout_ns, veles_poll_event *out, int64_t max) {
    int64_t timeout_ms = timeout_ns < 0 ? -1 : (timeout_ns + 999999) / 1000000; /* whole milliseconds, rounded up */
    pthread_mutex_lock(&poll_mutex);
    size_t n = nitems;
    struct pollfd *fds = malloc((n + 1) * sizeof *fds);
    if (!fds) poll_fatal("out of memory");
    fds[0].fd = wake_pipe[0];
    fds[0].events = POLLIN;
    fds[0].revents = 0;
    for (size_t i = 0; i < n; i++) {
        fds[i + 1].fd = items[i].fd;
        fds[i + 1].events = items[i].events;
        fds[i + 1].revents = 0;
    }
    __atomic_add_fetch(&waiting, 1, __ATOMIC_SEQ_CST);
    pthread_mutex_unlock(&poll_mutex);
    if (timeout_ms > 0x7fffffff) timeout_ms = 0x7fffffff;
    int r = poll(fds, (nfds_t)(n + 1), timeout_ms < 0 ? -1 : (int)timeout_ms);
    pthread_mutex_lock(&poll_mutex);
    __atomic_sub_fetch(&waiting, 1, __ATOMIC_SEQ_CST);
    int64_t got = 0;
    if (r > 0) {
        if (fds[0].revents) {
            char buf[64];
            while (read(wake_pipe[0], buf, sizeof buf) > 0) {}
            __atomic_store_n(&wake_pending, 0, __ATOMIC_SEQ_CST);
        }
        for (size_t i = 1; i <= n && got < max; i++) {
            short re = fds[i].revents;
            if (!re) continue;
            /* one-shot: reported once, then disarmed, unless it was armed
             * again (or forgotten) while the poll ran */
            size_t k = find_item(fds[i].fd);
            if (k == (size_t)-1 || items[k].events != fds[i].events) continue;
            items[k] = items[--nitems];
            out[got].fd = fds[i].fd;
            out[got].read = (re & (POLLIN | POLLHUP | POLLERR | POLLNVAL)) != 0;
            out[got].write = (re & (POLLOUT | POLLHUP | POLLERR | POLLNVAL)) != 0;
            got++;
        }
    }
    pthread_mutex_unlock(&poll_mutex);
    free(fds);
    return got;
}
#endif
