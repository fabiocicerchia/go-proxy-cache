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
func (rc RequestCall) RedirectToHTTPS() {
	targetURL := rc.GetRequestURL()
	targetURL.Scheme = SchemeHTTPS

	log.Infof("Redirect to: %s", targetURL.String())

	http.Redirect(rc.Response, &rc.Request, targetURL.String(), rc.DomainConfig.Server.Upstream.RedirectStatusCode)
}
