package handler

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func RedirectToHTTPS(w http.ResponseWriter, req *http.Request, redirectStatusCode int) {
	targetURL := req.URL
	targetURL.Scheme = "https"

	target := targetURL.String()

	log.Infof("Redirect to: %s", target)

	http.Redirect(w, req, target, redirectStatusCode)
}
