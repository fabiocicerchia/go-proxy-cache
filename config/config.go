package config

import (
	"strconv"
	"strings"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

var Config Configuration

type Configuration struct {
	Server Server
	Cache  Cache
}

type Server struct {
	Port       string
	TTL        int
	Forwarding Forward
}
type Forward struct {
	Host      string
	Scheme    string
	Endpoints []string
}
type Cache struct {
	Host            string
	Port            string
	Password        string
	DB              int
	AllowedStatuses []string
	AllowedMethods  []string
}

func InitConfig() {
	ttlSecs, _ := strconv.Atoi(utils.GetEnv("DEFAULT_TTL", "0"))

	lbEnpointList := utils.GetEnv("LB_ENDPOINT_LIST", "")
	endpoints := strings.Split(lbEnpointList, ",")

	cacheDb, err := strconv.Atoi(utils.GetEnv("REDIS_DB", "0"))
	if err != nil {
		cacheDb = 0
	}

	statuses := utils.GetEnv("CACHE_ALLOWED_STATUSES", "200,301,302")
	statusList := strings.Split(statuses, ",")

	methods := utils.GetEnv("CACHE_ALLOWED_METHODS", "HEAD,GET")
	methodList := strings.Split(methods, ",")

	Config = Configuration{
		Server: Server{
			Port: utils.GetEnv("SERVER_PORT", "8080"),
			TTL:  ttlSecs,
			Forwarding: Forward{
				Host:      utils.GetEnv("FORWARD_HOST", ""),
				Scheme:    utils.GetEnv("FORWARD_SCHEME", ""),
				Endpoints: endpoints,
			},
		},
		Cache: Cache{
			Host:            utils.GetEnv("REDIS_HOST", ""),
			Port:            utils.GetEnv("REDIS_PORT", "6379"),
			Password:        utils.GetEnv("REDIS_PASSWORD", ""),
			DB:              cacheDb,
			AllowedStatuses: statusList,
			AllowedMethods:  methodList,
		},
	}
}

func GetForwarding() Forward {
	return Config.Server.Forwarding
}
