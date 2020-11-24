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

	log "github.com/sirupsen/logrus"
)

// RedirectToHTTPS - Redirects from HTTP to HTTPS.
func RedirectToHTTPS(w http.ResponseWriter, req *http.Request, redirectStatusCode int) {
	targetURL := req.URL
	targetURL.Scheme = "https"

	target := targetURL.String()

	log.Infof("Redirect to: %s", target)

	http.Redirect(w, req, target, redirectStatusCode)
}
