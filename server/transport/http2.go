package transport

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net/http"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

// LinkItem - A URL item contained in a Link HTTP Header.
type LinkItem struct {
	URL    string
	Rel    string
	NoPush bool
	Params map[string]string
}

// PushProxiedResources - Start HTTP/2 Push of the resources needed.
func PushProxiedResources(lwr *response.LoggedResponseWriter, uriobj *cache.URIObj) {
	pusher, ok := lwr.ResponseWriter.(http.Pusher)
	if !ok {
		return
	}

	links := uriobj.ResponseHeaders.Values("Link")

	for _, link := range ParseMultiple(links) {
		if link.Rel != "preload" || link.NoPush {
			continue
		}

		if err := pusher.Push(link.URL, nil); err != nil {
			panic(err)
		}
	}

	uriobj.ResponseHeaders.Del("Link")
}

// ParseMultiple - Processes and extracts the URLs contained in multiple Link HTTP Headers.
func ParseMultiple(headers []string) []LinkItem {
	links := make([]LinkItem, 0)

	for _, header := range headers {
		links = append(links, Parse(header)...)
	}

	return links
}

// Parse - Processes and extracts the URLs contained in a Link HTTP Header.
func Parse(value string) (links []LinkItem) {
	for _, item := range strings.Split(value, ",") {
		link := LinkItem{Params: make(map[string]string)}

		for _, subpart := range strings.Split(item, ";") {
			subpart = strings.Trim(subpart, " ")
			if subpart == "" {
				continue
			}

			if strings.HasPrefix(subpart, "<") && strings.HasSuffix(subpart, ">") {
				link.URL = strings.Trim(subpart, "<>")
				continue
			}

			key, val := extractParam(subpart)
			if key == "" {
				continue
			}

			// RFC5988 Standard params: rel, anchor, rev, hreflang, media, title, title*, type.
			if strings.ToLower(key) == "rel" {
				link.Rel = val
			} else if strings.ToLower(key) == "nopush" {
				link.NoPush = true
			} else {
				link.Params[key] = strings.Trim(val, `"`)
			}
		}

		if link.URL != "" {
			links = append(links, link)
		}
	}

	return links
}

func extractParam(param string) (key, val string) {
	parts := strings.SplitN(param, "=", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}

	if len(parts) != 2 {
		return "", ""
	}

	key = parts[0]
	val = strings.Trim(parts[1], `"`)

	return key, val
}
