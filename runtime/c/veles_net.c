/*
 * Veles bootstrap runtime — TCP sockets for std/net.
 *
 * Every socket is non-blocking. A call that cannot complete reports
 * "would block" (1) and the Veles side parks the task with the executor's
 * socket wait (veles_task_wait_io in veles_task.c), then retries; so one
 * thread serves many connections and nothing here ever blocks the
 * executor. Results: 0 on success, 1 for would-block, otherwise the
 * platform error number (errno, or the WSA code on Windows) which
 * os.ioError turns into an IoError. Strings follow veles_os.c: (data, len)
 * in, a veles_string out-pointer out.
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
#include <errno.h>

#if defined(_WIN32)
#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>
typedef SOCKET sock_t;
#define BAD_SOCK INVALID_SOCKET
#else
#include <unistd.h>
#include <fcntl.h>
#include <netdb.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <arpa/inet.h>
#include <signal.h>
typedef int sock_t;
#define BAD_SOCK (-1)
#endif

typedef struct {
    const char *data;
    int64_t len;
} veles_string;

/* A List<u8> from the Veles side: only data and len are read here. */
typedef struct {
    char *data;
    int64_t len;
} veles_bytes_view;

void *veles_alloc(int64_t size);

static void set_string(veles_string *out, const char *s, int64_t len) {
    char *buf = veles_alloc(len + 1);
    memcpy(buf, s, (size_t)len);
    buf[len] = 0;
    out->data = buf;
    out->len = len;
}

static int64_t last_error(void) {
#if defined(_WIN32)
    return WSAGetLastError();
#else
    return errno;
#endif
}

static bool would_block(int64_t code) {
#if defined(_WIN32)
    return code == WSAEWOULDBLOCK || code == WSAEINPROGRESS;
#else
    return code == EAGAIN || code == EWOULDBLOCK || code == EINPROGRESS;
#endif
}

static void net_init(void) {
#if defined(_WIN32)
    static bool started;
    if (!started) {
        WSADATA data;
        WSAStartup(MAKEWORD(2, 2), &data);
        started = true;
    }
#else
    static bool started;
    if (!started) {
        signal(SIGPIPE, SIG_IGN); /* a peer that went away is an error code, not a signal */
        started = true;
    }
#endif
}

static void set_nonblocking(sock_t s) {
#if defined(_WIN32)
    u_long on = 1;
    ioctlsocket(s, FIONBIO, &on);
#else
    int flags = fcntl(s, F_GETFL, 0);
    fcntl(s, F_SETFL, flags | O_NONBLOCK);
#endif
}

static void close_sock(sock_t s) {
#if defined(_WIN32)
    closesocket(s);
#else
    close(s);
#endif
}

/* resolve host:port; an empty host means every interface for a listener
 * and the loopback for a client */
static int64_t resolve(const char *host, int64_t hlen, int64_t port, bool passive, struct addrinfo **out) {
    char *h = NULL;
    if (hlen > 0) {
        h = veles_alloc(hlen + 1);
        memcpy(h, host, (size_t)hlen);
        h[hlen] = 0;
    }
    char p[16];
    snprintf(p, sizeof p, "%d", (int)port);
    struct addrinfo hints;
    memset(&hints, 0, sizeof hints);
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    hints.ai_protocol = IPPROTO_TCP;
    if (passive && !h) hints.ai_flags = AI_PASSIVE;
    int r = getaddrinfo(h, p, &hints, out);
    if (r != 0) {
#if defined(_WIN32)
        return WSAGetLastError();
#else
        return r == EAI_NONAME || r == EAI_SERVICE ? ENOENT : EIO;
#endif
    }
    return 0;
}

int64_t veles_net_listen(const char *host, int64_t hlen, int64_t port, int64_t *fd) {
    net_init();
    struct addrinfo *info = NULL;
    int64_t code = resolve(host, hlen, port, true, &info);
    if (code) return code;
    code = 0;
    for (struct addrinfo *ai = info; ai; ai = ai->ai_next) {
        sock_t s = socket(ai->ai_family, ai->ai_socktype, ai->ai_protocol);
        if (s == BAD_SOCK) {
            code = last_error();
            continue;
        }
#if !defined(_WIN32)
        int on = 1;
        setsockopt(s, SOL_SOCKET, SO_REUSEADDR, &on, sizeof on);
#endif
        if (bind(s, ai->ai_addr, (int)ai->ai_addrlen) != 0 || listen(s, 128) != 0) {
            code = last_error();
            close_sock(s);
            continue;
        }
        set_nonblocking(s);
        *fd = (int64_t)s;
        code = 0;
        break;
    }
    freeaddrinfo(info);
    return code;
}

/* the port a listener is bound to (the system's choice when asked for 0) */
int64_t veles_net_port(int64_t fd) {
    struct sockaddr_storage addr;
    socklen_t len = sizeof addr;
    if (getsockname((sock_t)fd, (struct sockaddr *)&addr, &len) != 0) return 0;
    if (addr.ss_family == AF_INET) return ntohs(((struct sockaddr_in *)&addr)->sin_port);
    if (addr.ss_family == AF_INET6) return ntohs(((struct sockaddr_in6 *)&addr)->sin6_port);
    return 0;
}

int64_t veles_net_accept(int64_t fd, int64_t *conn) {
    sock_t s = accept((sock_t)fd, NULL, NULL);
    if (s == BAD_SOCK) {
        int64_t code = last_error();
        return would_block(code) ? 1 : code;
    }
    set_nonblocking(s);
    int on = 1;
    setsockopt(s, IPPROTO_TCP, TCP_NODELAY, (const char *)&on, sizeof on);
    *conn = (int64_t)s;
    return 0;
}

/* 0: connected; 1: in progress (wait for writable, then connect_result) */
int64_t veles_net_connect(const char *host, int64_t hlen, int64_t port, int64_t *fd) {
    net_init();
    struct addrinfo *info = NULL;
    int64_t code = resolve(hlen > 0 ? host : "127.0.0.1", hlen > 0 ? hlen : 9, port, false, &info);
    if (code) return code;
    code = 0;
    for (struct addrinfo *ai = info; ai; ai = ai->ai_next) {
        sock_t s = socket(ai->ai_family, ai->ai_socktype, ai->ai_protocol);
        if (s == BAD_SOCK) {
            code = last_error();
            continue;
        }
        set_nonblocking(s);
        int on = 1;
        setsockopt(s, IPPROTO_TCP, TCP_NODELAY, (const char *)&on, sizeof on);
        if (connect(s, ai->ai_addr, (int)ai->ai_addrlen) == 0) {
            *fd = (int64_t)s;
            code = 0;
            break;
        }
        code = last_error();
        if (would_block(code)) {
            *fd = (int64_t)s;
            code = 1;
            break;
        }
        close_sock(s);
    }
    freeaddrinfo(info);
    return code;
}

/* after a pending connect became writable: 0 or the connection's error */
int64_t veles_net_connect_result(int64_t fd) {
    int err = 0;
    socklen_t len = sizeof err;
    if (getsockopt((sock_t)fd, SOL_SOCKET, SO_ERROR, (char *)&err, &len) != 0) return last_error();
    return err;
}

/* 0 with the bytes in out (none = the peer closed), 1 would block */
int64_t veles_net_recv(int64_t fd, int64_t max, veles_string *out) {
    if (max < 1) max = 1;
    if (max > 1 << 20) max = 1 << 20;
    char *buf = veles_alloc(max + 1);
#if defined(_WIN32)
    int n = recv((sock_t)fd, buf, (int)max, 0);
#else
    ssize_t n = recv((sock_t)fd, buf, (size_t)max, 0);
#endif
    if (n < 0) {
        int64_t code = last_error();
        return would_block(code) ? 1 : code;
    }
    buf[n] = 0;
    out->data = buf;
    out->len = n;
    return 0;
}

/* sends from offset; *sent receives how many bytes went out this call */
int64_t veles_net_send(int64_t fd, veles_bytes_view *bytes, int64_t offset, int64_t *sent) {
    int64_t left = bytes->len - offset;
    if (left <= 0) {
        *sent = 0;
        return 0;
    }
#if defined(_WIN32)
    if (left > 0x7fffffff) left = 0x7fffffff;
    int n = send((sock_t)fd, bytes->data + offset, (int)left, 0);
#else
    ssize_t n = send((sock_t)fd, bytes->data + offset, (size_t)left, MSG_NOSIGNAL);
#endif
    if (n < 0) {
        int64_t code = last_error();
        *sent = 0;
        return would_block(code) ? 1 : code;
    }
    *sent = n;
    return 0;
}

void veles_net_close(int64_t fd) {
    close_sock((sock_t)fd);
}

/* half-close: no more sends; the peer reads end-of-stream */
int64_t veles_net_shutdown_write(int64_t fd) {
#if defined(_WIN32)
    if (shutdown((sock_t)fd, SD_SEND) != 0) return last_error();
#else
    if (shutdown((sock_t)fd, SHUT_WR) != 0) return last_error();
#endif
    return 0;
}

/* the peer's address as "host:port" */
void veles_net_peer(int64_t fd, veles_string *out) {
    struct sockaddr_storage addr;
    socklen_t len = sizeof addr;
    char host[NI_MAXHOST], port[NI_MAXSERV];
    if (getpeername((sock_t)fd, (struct sockaddr *)&addr, &len) != 0 ||
        getnameinfo((struct sockaddr *)&addr, len, host, sizeof host, port, sizeof port,
                    NI_NUMERICHOST | NI_NUMERICSERV) != 0) {
        set_string(out, "", 0);
        return;
    }
    char text[NI_MAXHOST + NI_MAXSERV + 4];
    if (addr.ss_family == AF_INET6) {
        snprintf(text, sizeof text, "[%s]:%s", host, port);
    } else {
        snprintf(text, sizeof text, "%s:%s", host, port);
    }
    set_string(out, text, (int64_t)strlen(text));
}

/* ---- IoError.kind (D76) ----------------------------------------------------
 * The platform's error number, as the portable kind std/prelude's `IoKind`
 * names. It lives here because only this file sees both errno and the
 * Winsock codes. The numbers are the enum's values; 0 is `Other`. Each
 * errno case is guarded: a C library may leave a POSIX name undefined. */
int64_t veles_io_kind(int64_t code) {
    enum {
        OTHER, NOT_FOUND, PERMISSION_DENIED, ALREADY_EXISTS, NOT_A_DIRECTORY,
        IS_A_DIRECTORY, DIRECTORY_NOT_EMPTY, CONNECTION_REFUSED,
        CONNECTION_RESET, CONNECTION_ABORTED, TIMED_OUT, ADDRESS_IN_USE,
        ADDRESS_NOT_AVAILABLE, BROKEN_PIPE, INTERRUPTED, INVALID_INPUT,
        INVALID_DATA
    };
#if defined(_WIN32)
    switch (code) {
    case WSAECONNREFUSED: return CONNECTION_REFUSED;
    case WSAECONNRESET: return CONNECTION_RESET;
    case WSAECONNABORTED: return CONNECTION_ABORTED;
    case WSAETIMEDOUT: return TIMED_OUT;
    case WSAEADDRINUSE: return ADDRESS_IN_USE;
    case WSAEADDRNOTAVAIL: return ADDRESS_NOT_AVAILABLE;
    case WSAEINTR: return INTERRUPTED;
    case WSAEINVAL: return INVALID_INPUT;
    case WSAEACCES: return PERMISSION_DENIED;
    case WSAHOST_NOT_FOUND: return NOT_FOUND;
    }
#endif
    switch (code) {
#ifdef ENOENT
    case ENOENT: return NOT_FOUND;
#endif
#ifdef EACCES
    case EACCES: return PERMISSION_DENIED;
#endif
#ifdef EPERM
    case EPERM: return PERMISSION_DENIED;
#endif
#ifdef EEXIST
    case EEXIST: return ALREADY_EXISTS;
#endif
#ifdef ENOTDIR
    case ENOTDIR: return NOT_A_DIRECTORY;
#endif
#ifdef EISDIR
    case EISDIR: return IS_A_DIRECTORY;
#endif
#ifdef ENOTEMPTY
    case ENOTEMPTY: return DIRECTORY_NOT_EMPTY;
#endif
#ifdef ECONNREFUSED
    case ECONNREFUSED: return CONNECTION_REFUSED;
#endif
#ifdef ECONNRESET
    case ECONNRESET: return CONNECTION_RESET;
#endif
#ifdef ECONNABORTED
    case ECONNABORTED: return CONNECTION_ABORTED;
#endif
#ifdef ETIMEDOUT
    case ETIMEDOUT: return TIMED_OUT;
#endif
#ifdef EADDRINUSE
    case EADDRINUSE: return ADDRESS_IN_USE;
#endif
#ifdef EADDRNOTAVAIL
    case EADDRNOTAVAIL: return ADDRESS_NOT_AVAILABLE;
#endif
#ifdef EPIPE
    case EPIPE: return BROKEN_PIPE;
#endif
#ifdef EINTR
    case EINTR: return INTERRUPTED;
#endif
#ifdef EINVAL
    case EINVAL: return INVALID_INPUT;
#endif
#ifdef EILSEQ
    case EILSEQ: return INVALID_DATA;
#endif
    }
    return OTHER;
}
