package transport

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/server/response"
	log "github.com/sirupsen/logrus"
)

// HopHeaders - List of Ho-by-hop headers.
// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var HopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

// removeConnectionHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
func removeConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = strings.TrimSpace(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

func copyResponse(dst io.Writer, src io.Reader, chunks [][]byte) error {
	// bodyBytes, err := ioutil.ReadAll(src)
	// if err != nil {
	// 	log.Warnf("ERROR: %s", err)
	// }

	// _, err = dst.Write(bodyBytes)
	// return err

	for _, chunk := range chunks {
		_, _ = dst.Write(chunk)
		if fl, ok := dst.(http.Flusher); ok {
			fl.Flush()
		}
	}
	return nil
}

// shouldPanicOnCopyError reports whether the reverse proxy should
// panic with http.ErrAbortHandler. This is the right thing to do by
// default, but Go 1.10 and earlier did not, so existing unit tests
// weren't expecting panics. Only panic in our own tests, or when
// running under the HTTP server.
func shouldPanicOnCopyError(ctx context.Context) bool {
	if ctx.Value(http.ServerContextKey) != nil {
		// We seem to be running under an HTTP server, so
		// it'll recover the panic.
		return true
	}
	// Otherwise act like Go 1.10 and earlier to not break
	// existing tests.
	return false
}

// ServeResponse - Serve a cached response.
func ServeResponse(
	ctx context.Context,
	lwr *response.LoggedResponseWriter,
	res http.Response,
	url url.URL,
	chunks [][]byte,
) {
	if cn, ok := lwr.ResponseWriter.(http.CloseNotifier); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		notifyChan := cn.CloseNotify()
		go func() {
			select {
			case <-notifyChan:
				cancel()
			case <-ctx.Done():
			}
		}()
	}

	// HTTP Headers
	removeConnectionHeaders(res.Header)
	for _, h := range HopHeaders {
		res.Header.Del(h)
	}
	response.CopyHeaders(lwr.Header(), res.Header)

	// The "Trailer" header isn't included in the Transport's response,
	// at least for *http.Transport. Build it up from Trailer.
	announcedTrailers := len(res.Trailer)
	if announcedTrailers > 0 {
		trailerKeys := make([]string, 0, len(res.Trailer))
		for k := range res.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		lwr.Header().Add("Trailer", strings.Join(trailerKeys, ", "))
	}

	lwr.WriteHeader(res.StatusCode)

	err := copyResponse(lwr, res.Body, chunks)
	if err != nil {
		defer res.Body.Close()
		// Since we're streaming the response, if we run into an error all we can do
		// is abort the request. Issue 23643: ReverseProxy should use ErrAbortHandler
		// on read error while copying body.
		if !shouldPanicOnCopyError(ctx) {
			log.Errorf("suppressing panic for copyResponse error in test; copy error: %v", err)
			return
		}
		panic(http.ErrAbortHandler)
	}
	res.Body.Close() // close now, instead of defer, to populate res.Trailer

	handleTrailer(announcedTrailers, lwr, res)
}

func handleTrailer(announcedTrailers int, lwr *response.LoggedResponseWriter, res http.Response) {
	if len(res.Trailer) > 0 {
		// Force chunking if we saw a response trailer.
		// This prevents net/http from calculating the length for short
		// bodies and adding a Content-Length.
		if fl, ok := lwr.ResponseWriter.(http.Flusher); ok {
			fl.Flush()
		}
	}

	if len(res.Trailer) == announcedTrailers {
		response.CopyHeaders(lwr.Header(), res.Trailer)
		return
	}

	for k, vv := range res.Trailer {
		k = http.TrailerPrefix + k
		for _, v := range vv {
			lwr.Header().Add(k, v)
		}
	}
}
