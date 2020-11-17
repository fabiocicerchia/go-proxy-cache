package config

import (
	"io/ioutil"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"

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

// InitConfigFromFileOrEnv - Init the configuration in sequence: from a YAML file, from environment variables, then defaults.
func InitConfigFromFileOrEnv(file string) {
	Config = Configuration{}

	data, _ := ioutil.ReadFile(file)
	_ = yaml.Unmarshal([]byte(data), &Config)

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

	if Config.Server.Port == "" {
		Config.Server.Port = utils.GetEnv("SERVER_PORT", "8080")
	}
	if Config.Server.TTL == 0 {
		Config.Server.TTL = ttlSecs
	}

	if Config.Server.Forwarding.Host == "" {
		Config.Server.Forwarding.Host = utils.GetEnv("FORWARD_HOST", "")
	}
	if Config.Server.Forwarding.Scheme == "" {
		Config.Server.Forwarding.Scheme = utils.GetEnv("FORWARD_SCHEME", "")
	}
	if len(Config.Server.Forwarding.Endpoints) == 0 {
		Config.Server.Forwarding.Endpoints = endpoints
	}

	if Config.Cache.Host == "" {
		Config.Cache.Host = utils.GetEnv("REDIS_HOST", "")
	}
	if Config.Cache.Port == "" {
		Config.Cache.Port = utils.GetEnv("REDIS_PORT", "6379")
	}
	if Config.Cache.Password == "" {
		Config.Cache.Password = utils.GetEnv("REDIS_PASSWORD", "")
	}
	if Config.Cache.DB == 0 {
		Config.Cache.DB = cacheDb
	}
	if len(Config.Cache.AllowedStatuses) == 0 {
		Config.Cache.AllowedStatuses = statusList
	}
	if len(Config.Cache.AllowedMethods) == 0 {
		Config.Cache.AllowedMethods = methodList
	}
}

// GetForwarding - Returns the forwarding configs.
func GetForwarding() Forward {
	return Config.Server.Forwarding
}
