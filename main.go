//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache
package main

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/server"
)

func main() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors:   false,
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})
	log.SetReportCaller(false)
	// TODO: Configurable
	log.SetOutput(os.Stdout)
	// TODO: Configurable
	log.SetLevel(log.InfoLevel)

	// start server
	server.Start()
}
