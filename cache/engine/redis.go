package engine

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine/client"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
)

var rdb map[string]*client.RedisClient

// rdbMu - Guards the connection map.
//
// Connections used to be created once at boot and read lock-free thereafter.
// The Kubernetes ingress controller can create one while requests are already
// being served, so both sides now synchronise.
var rdbMu sync.RWMutex

// GetConn - Retrieves the Redis connection.
func GetConn(connName string) *client.RedisClient {
	rdbMu.RLock()
	conn, ok := rdb[connName]
	rdbMu.RUnlock()

	if ok {
		return conn
	}

	logger.GetGlobal().Errorf("Missing redis connection for %s", connName)

	return nil
}

// InitConn - Initialises the Redis connection.
func InitConn(connName string, config config.Cache, logger *log.Logger) {
	logger.Debugf("New redis connection for %s", connName)
	conn := client.Connect(connName, config, logger)

	rdbMu.Lock()
	defer rdbMu.Unlock()

	if rdb == nil {
		rdb = make(map[string]*client.RedisClient)
	}

	rdb[connName] = conn
}

// HasConn - Whether a connection is already registered under a name.
func HasConn(connName string) bool {
	rdbMu.RLock()
	defer rdbMu.RUnlock()

	_, ok := rdb[connName]

	return ok
}
