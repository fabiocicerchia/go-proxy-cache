package transport

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
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
// See RFC 7230, section 6.1.
func removeConnectionHeaders(h http.Header) {
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = strings.TrimSpace(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
}

func copyResponse(dst io.Writer, chunks [][]byte) {
	for _, chunk := range chunks {
		_, _ = dst.Write(chunk)

		if fl, ok := dst.(http.Flusher); ok {
			fl.Flush()
		}
	}
}

// ServeCachedResponse - Serve a cached response.
func ServeCachedResponse(ctx context.Context, lwr *response.LoggedResponseWriter, uriobj cache.URIObj) {
	ctxWC, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		select {
		case <-ctx.Done():
			cancel()
		case <-ctxWC.Done():
		}
	}()

	res := http.Response{
		StatusCode: uriobj.StatusCode,
		Header:     uriobj.ResponseHeaders,
	}

	announcedTrailers := handleHeaders(lwr, res)

	PushProxiedResources(lwr, &uriobj)

	handleBody(lwr.ResponseWriter, uriobj.Content)
	handleTrailer(announcedTrailers, lwr, res)
}

func handleHeaders(lwr *response.LoggedResponseWriter, res http.Response) int {
	removeConnectionHeaders(res.Header)

	for _, h := range HopHeaders {
		res.Header.Del(h)
	}

	res.Header.Del(response.CacheStatusHeader)
	lwr.CopyHeaders(res.Header)

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

	return announcedTrailers
}

func handleBody(res http.ResponseWriter, content [][]byte) {
	copyResponse(res, content)
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
		lwr.CopyHeaders(res.Trailer)
		return
	}

	for k, vv := range res.Trailer {
		for _, v := range vv {
			lwr.Header().Add(http.TrailerPrefix+k, v)
		}
	}
}
