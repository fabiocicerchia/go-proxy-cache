package server

import (
	"github.com/fabiocicerchia/go-proxy-cache/config"
)

func tearDownServer() {
	config.Config = config.Configuration{}
}
