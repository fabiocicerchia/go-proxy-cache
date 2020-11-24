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

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/server/response"
)

// HandleHealthcheck - Returns healthcheck status.
func HandleHealthcheck(res http.ResponseWriter, req *http.Request) {
	lwr := response.NewLoggedResponseWriter(res)

	lwr.WriteHeader(http.StatusOK)
	_ = response.WriteBody(lwr, "HTTP OK\n")

	if engine.GetConn("global").Ping() {
		_ = response.WriteBody(lwr, "REDIS OK\n")
	} else {
		_ = response.WriteBody(lwr, "REDIS KO\n")
	}
}
