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
	"github.com/fabiocicerchia/go-proxy-cache/cache/engine/client"
	"github.com/fabiocicerchia/go-proxy-cache/config"
)

var rdb map[string]*client.RedisClient

func GetConn(connName string) *client.RedisClient {
	if conn, ok := rdb[connName]; ok {
		return conn
	}

	return rdb[connName]
}

func InitConn(connName string, config config.Cache) {
	if rdb == nil {
		rdb = make(map[string]*client.RedisClient)
	}

	rdb[connName] = client.Connect(config)
}
