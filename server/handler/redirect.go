package handler

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func RedirectToHTTPS(w http.ResponseWriter, req *http.Request, redirectStatusCode int) {
	targetUrl := req.URL
	targetUrl.Scheme = "https"

	target := targetUrl.String()

	log.Infof("Redirect to: %s", target)

	http.Redirect(w, req, target, redirectStatusCode)
}
