package engine

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

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine/client"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
)

var rdb map[string]*client.RedisClient

// GetConn - Retrieves the Redis connection.
func GetConn(connName string) *client.RedisClient {
	if conn, ok := rdb[connName]; ok {
		return conn
	}

	logger.GetGlobal().Errorf("Missing redis connection for %s", connName)

	return nil
}

// InitConn - Initialises the Redis connection.
func InitConn(connName string, config config.Cache, logger *log.Logger) {
	if rdb == nil {
		rdb = make(map[string]*client.RedisClient)
	}

	logger.Debugf("New redis connection for %s", connName)
	rdb[connName] = client.Connect(connName, config, logger)
}
