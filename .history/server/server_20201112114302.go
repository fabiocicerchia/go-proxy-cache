package server

import (
	"bufio"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	cache_redis "github.com/fabiocicerchia/go-proxy-cache/cache/redis"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// --- LOG

// Log the redirect url
func logRequest(proxyUrl string) {
	log.Printf("proxy_url: %s\n", proxyUrl)
}

// Log the env variables required for a reverse proxy
func logSetup() {
	forward_to := utils.GetEnv("FORWARD_TO", "")

	log.Printf("Server will run on: %s\n", GetListenAddress())
	log.Printf("Redirecting to url: %s\n", forward_to)
}

// --- LOGIC

// Get the port to listen on
func GetListenAddress() string {
	port := utils.GetEnv("PORT", "8080")
	return ":" + port
}

func Start() {
	// Log setup values
	logSetup()

	// redis connect
	cache_redis.Connect()

	// start server
	http.HandleFunc("/", handleRequestAndRedirect)
	if err := http.ListenAndServe(GetListenAddress(), nil); err != nil {
		panic(err)
	}
}

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}

func serveCachedContent(res *http.ResponseWriter, fullUrl string) bool {
	code, headers, page, _ := cache_redis.RetrieveFullPage(fullUrl)

	if code == 200 && page != "" {
		res.WriteHeader(code)
		for k, v := range headers {
			res.Header().Add(k, v)
		}
		res.Write(page)

		return true
	}

	return false
}

type ResponseWriterDecorator interface {
	Header() http.Header
	Write([]byte) (int, error)
	WriteHeader(statusCode int)
}

// A response represents the server side of an HTTP response.
type atomicBool int32
type chunkWriter struct {
	res         *response
	header      http.Header
	wroteHeader bool
	chunking    bool
}
type response struct {
	conn                *conn
	req                 *http.Request
	reqBody             io.ReadCloser
	cancelCtx           context.CancelFunc
	wroteHeader         bool
	wroteContinue       bool
	wants10KeepAlive    bool
	wantsClose          bool
	w                   *bufio.Writer
	cw                  chunkWriter
	handlerHeader       http.Header
	calledHeader        bool
	written             int64
	contentLength       int64
	status              int
	closeAfterReply     bool
	requestBodyLimitHit bool
	trailers            []string
	handlerDone         atomicBool
	dateBuf             [len(http.TimeFormat)]byte
	clenBuf             [10]byte
	statusBuf           [3]byte
	closeNotifyCh       chan bool
	didCloseNotify      int32
}

func (w *response) Header() http.Header {
	if w.cw.header == nil && w.wroteHeader && !w.cw.wroteHeader {
		// Accessing the header between logically writing it
		// and physically writing it means we need to allocate
		// a clone to snapshot the logically written state.
		w.cw.header = w.handlerHeader.Clone()
	}
	w.calledHeader = true
	return w.handlerHeader
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res ResponseWriterDecorator, req *http.Request) {
	url := utils.GetProxyUrl()

	logRequest(url)

	fullUrl := req.URL.String()
	if !serveCachedContent(&res, fullUrl) {
		serveReverseProxy(url, res, req)

		status = res.Header()
		headers = res.Header()
		content = res.Header()
		cache_redis.StoreFullPage(url, status, headers, content, 0)
	}
}
