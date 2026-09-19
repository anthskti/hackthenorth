/*
 * kvmd.c - tiny KVM daemon for QNX on the Raspberry Pi 5.
 *
 *   GET  /  or /stream  -> MJPEG video feed (multipart/x-mixed-replace), many viewers
 *                          open directly in a browser, VLC, ffplay, OpenCV...
 *   GET  /snapshot.jpg  -> one JPEG frame
 *   GET  /ui            -> small control page: video + keystroke text boxes
 *   POST /key           -> keyboard commands, one per line, forwarded to the
 *                          Leonardo on the serial port:
 *                            t<text>  p<hex>  r<hex>  k<hex>  a
 *                          (CORS enabled, so a page hosted anywhere can call it)
 *
 * Build (on the Pi):
 *   sudo apk add libjpeg-turbo-dev
 *   cc -O2 -o kvmd kvmd.c -lcamapi -lturbojpeg -lsocket
 *
 * Run (after kvm_setup.sh has set up /dev/ser1 and the sensor service):
 *   ./kvmd                       # defaults below
 *   ./kvmd -w 720 -h 480 -p 8080 -q 80 -s /dev/ser1
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <ctype.h>
#include <errno.h>
#include <fcntl.h>
#include <signal.h>
#include <time.h>
#include <unistd.h>
#include <pthread.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <netinet/tcp.h>
#include <arpa/inet.h>
#include <camera/camera_api.h>
#include <turbojpeg.h>

#ifndef MSG_NOSIGNAL
#define MSG_NOSIGNAL 0
#endif

/* ---------- settings ---------- */
static unsigned    g_w = 720, g_h = 480;
static int         g_port = 8080, g_quality = 80;
static const char *g_serial_path = "/dev/ser1";

static volatile sig_atomic_t g_running = 1;

/* ---------- latest JPEG frame, shared with viewers ---------- */
static pthread_mutex_t g_fmu = PTHREAD_MUTEX_INITIALIZER;
static pthread_cond_t  g_fcv = PTHREAD_COND_INITIALIZER;
static unsigned char  *g_jpeg = NULL;
static size_t          g_jpeg_len = 0;
static uint64_t        g_seq = 0;
static int             g_viewers = 0;      /* encode only when > 0 */

/* ---------- JPEG encoder state (used only by the camera thread) ---------- */
static tjhandle        g_tj;
static unsigned char  *g_planes[3];
static int             g_strides[3];

/* ---------- serial port to the Leonardo ---------- */
static int             g_ser = -1;
static pthread_mutex_t g_smu = PTHREAD_MUTEX_INITIALIZER;

/* ---------- small control page served at /ui ---------- */
static const char UI_HTML[] =
"<!doctype html><html><head><meta charset='utf-8'>"
"<meta name='viewport' content='width=device-width,initial-scale=1'><title>KVM</title>"
"<style>"
"body{margin:0;background:#111;color:#eee;font:14px sans-serif;display:flex;"
"flex-direction:column;align-items:center;gap:8px;padding:8px}"
"img{max-width:100%;background:#000}"
".row{display:flex;gap:6px;width:100%;max-width:960px}"
"input{flex:1;padding:8px;background:#222;color:#eee;border:1px solid #444;border-radius:4px}"
"input:focus{border-color:#4c9aff;outline:none}"
"button{padding:8px 12px;background:#222;color:#eee;border:1px solid #444;border-radius:4px;cursor:pointer}"
"#s{color:#999;font-size:12px}"
"</style></head><body>"
"<img src='/stream' alt='screen'>"
"<div class='row'><input id='live' placeholder='Click here and type: every key is sent live' autocomplete='off'></div>"
"<div class='row'><input id='txt' placeholder='Or type text here, then Send (Enter = send + Enter)' autocomplete='off'>"
"<button id='send'>Send</button><button id='enter'>Enter key</button></div>"
"<div id='s'></div>"
"<script>"
"const NL=String.fromCharCode(10);"
"const s=document.getElementById('s');"
"let q=Promise.resolve();"
"function send(l){const b=[].concat(l).join(NL)+NL;"
"q=q.then(()=>fetch('/key',{method:'POST',body:b})"
".then(r=>{s.textContent=r.ok?'sent':'error '+r.status})"
".catch(e=>{s.textContent='error: '+e}));}"
"const hex=n=>n.toString(16).toUpperCase().padStart(2,'0');"
"const sp={Enter:0xB0,NumpadEnter:0xB0,Escape:0xB1,Backspace:0xB2,Tab:0xB3,Space:0x20,"
"CapsLock:0xC1,Insert:0xD1,Home:0xD2,PageUp:0xD3,Delete:0xD4,End:0xD5,PageDown:0xD6,"
"ArrowRight:0xD7,ArrowLeft:0xD8,ArrowDown:0xD9,ArrowUp:0xDA,"
"ControlLeft:0x80,ShiftLeft:0x81,AltLeft:0x82,MetaLeft:0x83,"
"ControlRight:0x84,ShiftRight:0x85,AltRight:0x86,MetaRight:0x87,"
"Minus:0x2D,Equal:0x3D,BracketLeft:0x5B,BracketRight:0x5D,Backslash:0x5C,"
"Semicolon:0x3B,Quote:0x27,Comma:0x2C,Period:0x2E,Slash:0x2F,Backquote:0x60};"
"function kc(c){if(c in sp)return sp[c];"
"if(c.length==4&&c.startsWith('Key'))return c.charCodeAt(3)+32;"
"if(c.length==6&&c.startsWith('Digit'))return c.charCodeAt(5);"
"if(c[0]=='F'){const n=parseInt(c.slice(1));if(n>=1&&n<=12)return 0xC1+n;}"
"return null;}"
"const live=document.getElementById('live');const held=new Set();"
"live.addEventListener('keydown',e=>{e.preventDefault();if(e.repeat)return;"
"const k=kc(e.code);if(k===null){s.textContent='unmapped key: '+e.code;return;}"
"held.add(k);send('p'+hex(k));});"
"live.addEventListener('keyup',e=>{e.preventDefault();const k=kc(e.code);"
"if(k===null||!held.has(k))return;held.delete(k);send('r'+hex(k));});"
"live.addEventListener('blur',()=>{held.clear();send('a');});"
"const txt=document.getElementById('txt');"
"function sendText(enter){"
"const t=[...txt.value].filter(ch=>ch>=' '&&ch<='~').join('');"
"const l=[];for(let i=0;i<t.length;i+=50)l.push('t'+t.slice(i,i+50));"
"if(enter)l.push('kB0');if(l.length)send(l);txt.value='';}"
"document.getElementById('send').onclick=()=>sendText(false);"
"document.getElementById('enter').onclick=()=>send('kB0');"
"txt.addEventListener('keydown',e=>{if(e.key=='Enter'){e.preventDefault();sendText(true);}});"
"</script></body></html>";

/* ===================================================================== */
/* Camera                                                                */
/* ===================================================================== */

static void vf_callback(camera_handle_t h, camera_buffer_t *buf, void *arg)
{
    (void)h; (void)arg;
    if (buf->frametype != CAMERA_FRAMETYPE_YCBYCR)
        return;

    pthread_mutex_lock(&g_fmu);
    int viewers = g_viewers;
    pthread_mutex_unlock(&g_fmu);
    if (viewers == 0)
        return;                                   /* nobody watching */

    const unsigned w = buf->framedesc.ycbycr.width;
    const unsigned hh = buf->framedesc.ycbycr.height;
    const unsigned stride = buf->framedesc.ycbycr.stride;
    if (w != g_w || hh != g_h)
        return;

    /* YUYV (packed 4:2:2) -> Y, U, V planes (planar 4:2:2) */
    const uint8_t *src = (const uint8_t *)buf->framebuf;
    for (unsigned y = 0; y < hh; y++) {
        const uint8_t *s = src + (size_t)y * stride;
        uint8_t *yd = g_planes[0] + (size_t)y * w;
        uint8_t *ud = g_planes[1] + (size_t)y * (w / 2);
        uint8_t *vd = g_planes[2] + (size_t)y * (w / 2);
        for (unsigned x = 0; x < w / 2; x++) {
            yd[2 * x]     = s[4 * x];
            ud[x]         = s[4 * x + 1];
            yd[2 * x + 1] = s[4 * x + 2];
            vd[x]         = s[4 * x + 3];
        }
    }

    unsigned char *jpeg = NULL;
    size_t len = 0;
    if (tj3CompressFromYUVPlanes8(g_tj, (const unsigned char * const *)g_planes,
                                  (int)w, g_strides, (int)hh, &jpeg, &len) != 0) {
        fprintf(stderr, "kvmd: jpeg: %s\n", tj3GetErrorStr(g_tj));
        tj3Free(jpeg);
        return;
    }

    pthread_mutex_lock(&g_fmu);
    unsigned char *old = g_jpeg;
    g_jpeg = jpeg;
    g_jpeg_len = len;
    g_seq++;
    pthread_cond_broadcast(&g_fcv);
    pthread_mutex_unlock(&g_fmu);
    tj3Free(old);
}

static void status_callback(camera_handle_t h, camera_devstatus_t st,
                            uint16_t extra, void *arg)
{
    (void)h; (void)arg;
    if (st == CAMERA_STATUS_VIEWFINDER_ERROR || st == CAMERA_STATUS_DISCONNECTED ||
        st == CAMERA_STATUS_CAMERA_UNPLUGGED)
        fprintf(stderr, "kvmd: camera status %d (extra %u)\n", (int)st, extra);
}

static int camera_start(camera_handle_t *out)
{
    camera_handle_t h = CAMERA_HANDLE_INVALID;
    int err = camera_open(CAMERA_UNIT_1, CAMERA_MODE_RW | CAMERA_MODE_ROLL, &h);
    if (err != CAMERA_EOK) { fprintf(stderr, "kvmd: camera_open: %d\n", err); return -1; }

    err = camera_set_vf_mode(h, CAMERA_VFMODE_VIDEO);
    if (err != CAMERA_EOK) fprintf(stderr, "kvmd: set_vf_mode: %d\n", err);

    err = camera_set_vf_property(h,
            CAMERA_IMGPROP_FORMAT, CAMERA_FRAMETYPE_YCBYCR,
            CAMERA_IMGPROP_WIDTH, g_w,
            CAMERA_IMGPROP_HEIGHT, g_h);
    if (err != CAMERA_EOK) {
        fprintf(stderr, "kvmd: set_vf_property %ux%u: %d\n", g_w, g_h, err);
        camera_close(h);
        return -1;
    }

    err = camera_start_viewfinder(h, vf_callback, status_callback, NULL);
    if (err != CAMERA_EOK) {
        fprintf(stderr, "kvmd: start_viewfinder: %d\n", err);
        camera_close(h);
        return -1;
    }
    *out = h;
    return 0;
}

/* ===================================================================== */
/* Serial / keyboard                                                     */
/* ===================================================================== */

/* Accept only the Leonardo's command format, so the browser can't send junk */
static int valid_cmd(const char *s)
{
    size_t n = strlen(s);
    switch (s[0]) {
    case 'a':
        return n == 1;
    case 'p': case 'r': case 'k':
        if (n < 2 || n > 3) return 0;
        for (const char *p = s + 1; *p; p++)
            if (!isxdigit((unsigned char)*p)) return 0;
        return 1;
    case 't':
        if (n < 2 || n > 61) return 0;          /* keep lines short for the Leonardo */
        for (const char *p = s + 1; *p; p++)
            if ((unsigned char)*p < 0x20 || (unsigned char)*p > 0x7e) return 0;
        return 1;
    default:
        return 0;
    }
}

/* Send one validated command line; returns 0 on success */
static int serial_send(const char *cmd)
{
    if (g_ser < 0) return -1;
    char line[80];
    int n = snprintf(line, sizeof line, "%s\n", cmd);

    pthread_mutex_lock(&g_smu);
    int ok = write(g_ser, line, (size_t)n) == n;
    /* Pace text so the Leonardo's 64-byte serial buffer never overflows
     * while it is busy typing (~3 ms per character). */
    if (ok && cmd[0] == 't')
        usleep((useconds_t)(strlen(cmd) * 4000));
    pthread_mutex_unlock(&g_smu);
    return ok ? 0 : -1;
}

/* ===================================================================== */
/* Input logging                                                         */
/* ===================================================================== */

static const char *key_name(unsigned k, char *buf, size_t n)
{
    switch (k) {
    case 0x80: return "L-Ctrl";   case 0x81: return "L-Shift";
    case 0x82: return "L-Alt";    case 0x83: return "L-Super";
    case 0x84: return "R-Ctrl";   case 0x85: return "R-Shift";
    case 0x86: return "R-Alt";    case 0x87: return "R-Super";
    case 0xB0: return "Enter";    case 0xB1: return "Esc";
    case 0xB2: return "Backspace";case 0xB3: return "Tab";
    case 0xC1: return "CapsLock"; case 0xCE: return "PrintScreen";
    case 0xCF: return "ScrollLock"; case 0xD0: return "Pause";
    case 0xD1: return "Insert";   case 0xD2: return "Home";
    case 0xD3: return "PageUp";   case 0xD4: return "Delete";
    case 0xD5: return "End";      case 0xD6: return "PageDown";
    case 0xD7: return "Right";    case 0xD8: return "Left";
    case 0xD9: return "Down";     case 0xDA: return "Up";
    case 0xED: return "Menu";     case 0x20: return "Space";
    }
    if (k >= 0xC2 && k <= 0xCD) { snprintf(buf, n, "F%u", k - 0xC1); return buf; }
    if (k > 0x20 && k < 0x7F)   { snprintf(buf, n, "'%c'", (char)k); return buf; }
    snprintf(buf, n, "0x%02X", k);
    return buf;
}

static void log_cmd(const char *ip, const char *cmd, const char *result)
{
    char ts[16], kb[16], what[96];
    time_t now = time(NULL);
    struct tm tm;
    localtime_r(&now, &tm);
    strftime(ts, sizeof ts, "%H:%M:%S", &tm);

    unsigned k = (cmd[0] && cmd[1]) ? (unsigned)strtoul(cmd + 1, NULL, 16) : 0;
    switch (cmd[0]) {
    case 'p': snprintf(what, sizeof what, "press    %s", key_name(k, kb, sizeof kb)); break;
    case 'r': snprintf(what, sizeof what, "release  %s", key_name(k, kb, sizeof kb)); break;
    case 'k': snprintf(what, sizeof what, "tap      %s", key_name(k, kb, sizeof kb)); break;
    case 'a': snprintf(what, sizeof what, "release  all keys"); break;
    case 't': snprintf(what, sizeof what, "type     \"%.60s\"", cmd + 1); break;
    default:  snprintf(what, sizeof what, "?        %.40s", cmd); break;
    }
    fprintf(stderr, "[%s] %-15s %s%s%s\n", ts, ip, what,
            result ? "  -> " : "", result ? result : "");
}

/* ===================================================================== */
/* HTTP                                                                  */
/* ===================================================================== */

static int send_all(int fd, const void *data, size_t len)
{
    const char *p = data;
    while (len > 0) {
        ssize_t n = send(fd, p, len, MSG_NOSIGNAL);
        if (n < 0) { if (errno == EINTR) continue; return -1; }
        p += n;
        len -= (size_t)n;
    }
    return 0;
}

static void send_simple(int fd, const char *status, const char *type, const char *body)
{
    char hdr[256];
    size_t blen = body ? strlen(body) : 0;
    int n = snprintf(hdr, sizeof hdr,
        "HTTP/1.1 %s\r\nContent-Type: %s\r\nContent-Length: %zu\r\n"
        "Access-Control-Allow-Origin: *\r\n"
        "Cache-Control: no-store\r\nConnection: close\r\n\r\n",
        status, type, blen);
    send_all(fd, hdr, (size_t)n);
    if (blen) send_all(fd, body, blen);
}

/* Wait for a frame newer than *seen and copy it; returns length or 0 on timeout */
static size_t wait_frame(uint64_t *seen, unsigned char **copy, size_t *cap, int timeout_s)
{
    size_t len = 0;
    pthread_mutex_lock(&g_fmu);
    while (g_seq == *seen && g_running) {
        struct timespec ts;
        clock_gettime(CLOCK_REALTIME, &ts);
        ts.tv_sec += timeout_s;
        if (pthread_cond_timedwait(&g_fcv, &g_fmu, &ts) == ETIMEDOUT)
            break;
    }
    if (g_seq != *seen && g_jpeg) {
        *seen = g_seq;
        len = g_jpeg_len;
        if (len > *cap) {
            unsigned char *nb = realloc(*copy, len);
            if (!nb) { pthread_mutex_unlock(&g_fmu); return 0; }
            *copy = nb;
            *cap = len;
        }
        memcpy(*copy, g_jpeg, len);
    }
    pthread_mutex_unlock(&g_fmu);
    return len;
}

static void viewers_add(int d)
{
    pthread_mutex_lock(&g_fmu);
    g_viewers += d;
    pthread_mutex_unlock(&g_fmu);
}

static void serve_stream(int fd)
{
    static const char hdr[] =
        "HTTP/1.1 200 OK\r\n"
        "Content-Type: multipart/x-mixed-replace; boundary=frame\r\n"
        "Access-Control-Allow-Origin: *\r\n"
        "Cache-Control: no-cache, no-store, must-revalidate\r\n"
        "Pragma: no-cache\r\nConnection: close\r\n\r\n";
    if (send_all(fd, hdr, sizeof hdr - 1) < 0) return;

    int one = 1;
    setsockopt(fd, IPPROTO_TCP, TCP_NODELAY, &one, sizeof one);

    viewers_add(1);
    uint64_t seen = 0;
    unsigned char *copy = NULL;
    size_t cap = 0;

    while (g_running) {
        size_t len = wait_frame(&seen, &copy, &cap, 2);
        if (len == 0) continue;                 /* no frame yet: keep waiting */
        char ph[128];
        int n = snprintf(ph, sizeof ph,
            "--frame\r\nContent-Type: image/jpeg\r\nContent-Length: %zu\r\n\r\n", len);
        if (send_all(fd, ph, (size_t)n) < 0 ||
            send_all(fd, copy, len) < 0 ||
            send_all(fd, "\r\n", 2) < 0)
            break;                              /* viewer went away */
    }
    free(copy);
    viewers_add(-1);
}

static void serve_snapshot(int fd)
{
    viewers_add(1);
    uint64_t seen;
    pthread_mutex_lock(&g_fmu);
    seen = g_seq;                               /* want a fresh frame */
    pthread_mutex_unlock(&g_fmu);
    unsigned char *copy = NULL;
    size_t cap = 0;
    size_t len = wait_frame(&seen, &copy, &cap, 3);
    viewers_add(-1);

    if (len == 0) {
        send_simple(fd, "503 Service Unavailable", "text/plain", "no frame from camera\n");
    } else {
        char hdr[160];
        int n = snprintf(hdr, sizeof hdr,
            "HTTP/1.1 200 OK\r\nContent-Type: image/jpeg\r\nContent-Length: %zu\r\n"
            "Access-Control-Allow-Origin: *\r\n"
            "Cache-Control: no-store\r\nConnection: close\r\n\r\n", len);
        send_all(fd, hdr, (size_t)n);
        send_all(fd, copy, len);
    }
    free(copy);
}

/* body: one command per line */
static void serve_key(int fd, char *body, const char *ip)
{
    int sent = 0, bad = 0;
    char *save = NULL;
    for (char *line = strtok_r(body, "\n", &save); line; line = strtok_r(NULL, "\n", &save)) {
        size_t n = strlen(line);
        if (n && line[n - 1] == '\r') line[n - 1] = '\0';
        if (!line[0]) continue;
        if (!valid_cmd(line)) { log_cmd(ip, line, "REJECTED (invalid)"); bad++; continue; }
        if (serial_send(line) == 0) { log_cmd(ip, line, NULL); sent++; }
        else {
            log_cmd(ip, line, "FAILED (serial port unavailable)");
            send_simple(fd, "503 Service Unavailable", "text/plain", "serial port unavailable\n");
            return;
        }
    }
    if (bad && !sent)
        send_simple(fd, "400 Bad Request", "text/plain", "invalid command\n");
    else
        send_simple(fd, "204 No Content", "text/plain", NULL);
}

struct client {
    int  fd;
    char ip[INET_ADDRSTRLEN];
};

static void *client_thread(void *arg)
{
    struct client *c = arg;
    int fd = c->fd;
    char ip[INET_ADDRSTRLEN];
    memcpy(ip, c->ip, sizeof ip);
    free(c);
    char req[8192];
    size_t used = 0;
    char *hdr_end = NULL;

    /* read request headers */
    while (used < sizeof req - 1) {
        ssize_t n = recv(fd, req + used, sizeof req - 1 - used, 0);
        if (n <= 0) { close(fd); return NULL; }
        used += (size_t)n;
        req[used] = '\0';
        if ((hdr_end = strstr(req, "\r\n\r\n")) != NULL) break;
    }
    if (!hdr_end) { close(fd); return NULL; }

    char method[8] = {0}, path[256] = {0};
    sscanf(req, "%7s %255s", method, path);
    char *q = strchr(path, '?');
    if (q) *q = '\0';                           /* ignore query strings */

    if (strcmp(method, "GET") == 0) {
        if (strcmp(path, "/") == 0 || strcmp(path, "/stream") == 0) serve_stream(fd);
        else if (strcmp(path, "/snapshot.jpg") == 0)                serve_snapshot(fd);
        else if (strcmp(path, "/ui") == 0)
            send_simple(fd, "200 OK", "text/html; charset=utf-8", UI_HTML);
        else send_simple(fd, "404 Not Found", "text/plain", "not found\n");
    } else if (strcmp(method, "OPTIONS") == 0) {
        /* CORS preflight, so web pages hosted elsewhere can POST /key */
        static const char pre[] =
            "HTTP/1.1 204 No Content\r\n"
            "Access-Control-Allow-Origin: *\r\n"
            "Access-Control-Allow-Methods: GET, POST, OPTIONS\r\n"
            "Access-Control-Allow-Headers: Content-Type\r\n"
            "Access-Control-Max-Age: 86400\r\n"
            "Connection: close\r\n\r\n";
        send_all(fd, pre, sizeof pre - 1);
    } else if (strcmp(method, "POST") == 0 && strcmp(path, "/key") == 0) {
        size_t clen = 0;
        const char *cl = strcasestr(req, "\r\nContent-Length:");
        if (cl) clen = strtoul(cl + 17, NULL, 10);
        if (clen > 4096) {
            send_simple(fd, "413 Payload Too Large", "text/plain", "too large\n");
        } else {
            char body[4097];
            size_t have = used - (size_t)(hdr_end + 4 - req);
            if (have > clen) have = clen;
            memcpy(body, hdr_end + 4, have);
            while (have < clen) {
                ssize_t n = recv(fd, body + have, clen - have, 0);
                if (n <= 0) break;
                have += (size_t)n;
            }
            body[have] = '\0';
            serve_key(fd, body, ip);
        }
    } else {
        send_simple(fd, "405 Method Not Allowed", "text/plain", "method not allowed\n");
    }

    close(fd);
    return NULL;
}

/* ===================================================================== */
/* main                                                                  */
/* ===================================================================== */

static void on_signal(int sig) { (void)sig; g_running = 0; }

static void usage(const char *prog)
{
    fprintf(stderr,
        "usage: %s [-w width] [-h height] [-p port] [-q quality] [-s serial]\n",
        prog);
}

int main(int argc, char **argv)
{
    int opt;
    while ((opt = getopt(argc, argv, "w:h:p:q:s:")) != -1) {
        switch (opt) {
        case 'w': g_w = (unsigned)atoi(optarg); break;
        case 'h': g_h = (unsigned)atoi(optarg); break;
        case 'p': g_port = atoi(optarg); break;
        case 'q': g_quality = atoi(optarg); break;
        case 's': g_serial_path = optarg; break;
        default: usage(argv[0]); return 1;
        }
    }
    if (g_w == 0 || g_h == 0 || (g_w & 1)) {
        fprintf(stderr, "kvmd: width must be even and non-zero\n");
        return 1;
    }

    struct sigaction sa;
    memset(&sa, 0, sizeof sa);
    sa.sa_handler = on_signal;                  /* no SA_RESTART: accept() returns */
    sigaction(SIGINT, &sa, NULL);
    sigaction(SIGTERM, &sa, NULL);
    signal(SIGPIPE, SIG_IGN);

    /* JPEG encoder: planar 4:2:2 input */
    g_tj = tj3Init(TJINIT_COMPRESS);
    if (!g_tj) { fprintf(stderr, "kvmd: tj3Init failed\n"); return 1; }
    tj3Set(g_tj, TJPARAM_SUBSAMP, TJSAMP_422);
    tj3Set(g_tj, TJPARAM_QUALITY, g_quality);
    g_strides[0] = (int)g_w;
    g_strides[1] = g_strides[2] = (int)(g_w / 2);
    g_planes[0] = malloc((size_t)g_w * g_h);
    g_planes[1] = malloc((size_t)(g_w / 2) * g_h);
    g_planes[2] = malloc((size_t)(g_w / 2) * g_h);
    if (!g_planes[0] || !g_planes[1] || !g_planes[2]) { fprintf(stderr, "kvmd: out of memory\n"); return 1; }

    /* serial (keyboard); video still works if this fails */
    g_ser = open(g_serial_path, O_WRONLY | O_NOCTTY);
    if (g_ser < 0)
        fprintf(stderr, "kvmd: warning: cannot open %s (%s); keyboard disabled\n",
                g_serial_path, strerror(errno));

    camera_handle_t cam;
    if (camera_start(&cam) < 0) return 1;

    int srv = socket(AF_INET, SOCK_STREAM, 0);
    if (srv < 0) { perror("socket"); return 1; }
    int one = 1;
    setsockopt(srv, SOL_SOCKET, SO_REUSEADDR, &one, sizeof one);
    struct sockaddr_in addr;
    memset(&addr, 0, sizeof addr);
    addr.sin_family = AF_INET;
    addr.sin_addr.s_addr = htonl(INADDR_ANY);
    addr.sin_port = htons((uint16_t)g_port);
    if (bind(srv, (struct sockaddr *)&addr, sizeof addr) < 0) { perror("bind"); return 1; }
    if (listen(srv, 16) < 0) { perror("listen"); return 1; }

    fprintf(stderr, "kvmd: %ux%u, quality %d, http://<pi-ip>:%d/  (Ctrl+C to stop)\n",
            g_w, g_h, g_quality, g_port);

    while (g_running) {
        struct sockaddr_in peer;
        socklen_t plen = sizeof peer;
        int fd = accept(srv, (struct sockaddr *)&peer, &plen);
        if (fd < 0) { if (errno == EINTR) continue; perror("accept"); break; }
        struct client *c = malloc(sizeof *c);
        if (!c) { close(fd); continue; }
        c->fd = fd;
        if (!inet_ntop(AF_INET, &peer.sin_addr, c->ip, sizeof c->ip))
            strcpy(c->ip, "?");
        pthread_t t;
        pthread_attr_t at;
        pthread_attr_init(&at);
        pthread_attr_setdetachstate(&at, PTHREAD_CREATE_DETACHED);
        if (pthread_create(&t, &at, client_thread, c) != 0) {
            close(fd);
            free(c);
        }
        pthread_attr_destroy(&at);
    }

    fprintf(stderr, "kvmd: shutting down\n");
    close(srv);
    pthread_mutex_lock(&g_fmu);
    pthread_cond_broadcast(&g_fcv);             /* wake viewers so they exit */
    pthread_mutex_unlock(&g_fmu);
    if (g_ser >= 0) { serial_send("a"); close(g_ser); }   /* release all keys */
    camera_stop_viewfinder(cam);
    camera_close(cam);
    tj3Destroy(g_tj);
    return 0;
}
