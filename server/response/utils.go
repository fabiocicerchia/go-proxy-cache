package response

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/logger"
)

// GetLogger - Get logger instance with RequestID.
func (lwr LoggedResponseWriter) GetLogger() *log.Entry {
	return logger.GetGlobal().WithFields(log.Fields{
		"ReqID": lwr.ReqID,
	})
}
