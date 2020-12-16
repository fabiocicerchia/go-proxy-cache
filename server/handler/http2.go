package handler

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
	"regexp"

	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

// PushProxiedResources - Start HTTP/2 Push of the resources needed.
func PushProxiedResources(lwr *response.LoggedResponseWriter) {
	pusher, ok := lwr.ResponseWriter.(http.Pusher)
	if !ok {
		return
	}

	links := lwr.Header().Values("Link")

	for _, url := range processLinks(links) {
		err := pusher.Push(url, nil)
		if err != nil {
			panic(err)
		}
	}

	lwr.Header().Del("Links")
}

func processLinks(values []string) []string {
	var links []string

	re := regexp.MustCompile(`<([^>]+)>`)

	for _, val := range values {
		matches := re.FindAllSubmatch([]byte(val), -1)
		for _, processedLink := range matches {
			links = append(links, string(processedLink[1]))
		}
	}

	return links
}
