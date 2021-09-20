package response

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/logger"
)

// LoggedResponseWriter - Decorator for http.ResponseWriter.
func (lwr LoggedResponseWriter) GetLogger() *log.Entry {
	return logger.GetGlobal().WithFields(log.Fields{
		"ReqID": lwr.ReqID,
	})
}
