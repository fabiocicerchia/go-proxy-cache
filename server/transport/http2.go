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
	"net/http"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/cache"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

// ItemsKeyValueCount - Amount of items allowed in a pair key-value.
const ItemsKeyValueCount int = 2

// LinkItem - A URL item contained in a Link HTTP Header.
type LinkItem struct {
	URL    string
	Rel    string
	NoPush bool
	Params map[string]string
}

// PushProxiedResources - Start HTTP/2 Push of the resources needed. @deprecated
func PushProxiedResources(lwr *response.LoggedResponseWriter, uriobj *cache.URIObj) {
	pusher, ok := lwr.ResponseWriter.(http.Pusher)
	if !ok {
		return
	}

	for _, link := range ParseMultiple(uriobj.ResponseHeaders.Values("Link")) {
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
		link := parseItem(item)

		if link.URL != "" {
			links = append(links, link)
		}
	}

	return links
}

func parseItem(item string) LinkItem {
	link := LinkItem{Params: make(map[string]string)}
	for _, subpart := range strings.Split(item, ";") {
		fillLinkItem(&link, subpart)
	}

	return link
}

func fillLinkItem(link *LinkItem, subpart string) {
	subpart = strings.Trim(subpart, " ")
	if subpart == "" {
		return
	}

	if strings.HasPrefix(subpart, "<") && strings.HasSuffix(subpart, ">") {
		link.URL = strings.Trim(subpart, "<>")
		return
	}

	key, val := extractParam(subpart)
	if key == "" {
		return
	}

	// RFC5988 Standard params: rel, anchor, rev, hreflang, media, title, title*, type.
	switch strings.ToLower(key) {
	case "rel":
		link.Rel = val
	case "nopush":
		link.NoPush = true
	default:
		link.Params[key] = strings.Trim(val, `"`)
	}
}

func extractParam(param string) (key, val string) {
	parts := strings.SplitN(param, "=", ItemsKeyValueCount)
	if len(parts) == 1 {
		return parts[0], ""
	}

	if len(parts) != ItemsKeyValueCount {
		return "", ""
	}

	key = parts[0]
	val = strings.Trim(parts[1], `"`)

	return key, val
}
