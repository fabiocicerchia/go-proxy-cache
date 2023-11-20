package config

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jinzhu/copier"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/fabiocicerchia/go-proxy-cache/utils/scheme"
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
)

// PasswordOmittedValue - Replacement value when showing passwords in configuration.
const PasswordOmittedValue = "*** OMITTED ***"

// SchemeWildcard - Label to be shown when no schema (http/https) is selected.
const SchemeWildcard = "*"

func newFromEnv() Configuration {
	envConfig := Configuration{}

	err := envconfig.Process("", &envConfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	return envConfig
}

func getFromYaml(file string) (Configuration, error) {
	YamlConfig := Configuration{}

	data, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return YamlConfig, err
	}

	err = yaml.UnmarshalStrict(data, &YamlConfig)

	if err != nil {
		return YamlConfig, err
	}

	YamlConfig.Server.Upstream.Scheme = scheme.NormalizeScheme(YamlConfig.Server.Upstream.Scheme)

	return YamlConfig, err
}

// InitConfigFromFileOrEnv - Init the configuration in sequence: from a YAML file, from environment variables,
// then defaults.
func InitConfigFromFileOrEnv(file string) {
	Config.CopyOverWith(newFromEnv(), nil)

	YamlConfig := loadYAMLFilefile(file)

	// allow only the config file to specify overrides per domain
	Config.Domains = YamlConfig.Domains

	// DOMAINS
	copyGlobalOverDomainConfig(file)
}

func loadYAMLFilefile(file string) (YamlConfig Configuration) {
	_, err := os.Stat(file)
	if !os.IsNotExist(err) {
		YamlConfig, err = getFromYaml(file)
		if err != nil {
			log.Fatalf("Cannot unmarshal YAML: %s\n", err)
		}

		Config.CopyOverWith(YamlConfig, &file)
	}

	return YamlConfig
}

func copyGlobalOverDomainConfig(file string) {
	if Config.Domains != nil {
		domains := Config.Domains
		for k, v := range domains {
			domain := Config
			domain.CopyOverWith(v, &file)
			domain.Domains = Domains{}
			domains[k] = domain
		}

		Config.Domains = domains
	}
}

// Validate - Validate a YAML config file is syntactically valid.
func Validate(file string) (bool, error) {
	_, err := getFromYaml(file)
	return err != nil, err
}

func patchAbsFilePath(filePath string, relativeTo *string) string {
	abs, err := os.Getwd()

	if err == nil && relativeTo != nil && *relativeTo != "" {
		abs, err = filepath.Abs(*relativeTo)
		abs = filepath.Dir(abs)
	}

	if err == nil {
		if filePath != "" && !strings.HasPrefix(filePath, "/") {
			return filepath.Join(abs, filepath.Clean(filePath))
		}
	}

	return filePath
}

// CopyOverWith - Copies the Configuration over another (preserving not defined settings).
func (c *Configuration) CopyOverWith(overrides Configuration, file *string) {
	c.copyOverWithServer(overrides.Server)
	c.copyOverWithTLS(overrides.Server, file)
	c.copyOverWithTimeout(overrides.Server)
	c.copyOverWithUpstream(overrides.Server)
	c.copyOverWithCache(overrides.Cache)
	c.copyOverWithTracing(overrides.Tracing)
	c.copyOverWithLog(overrides.Log)
}

// --- SERVER.
func (c *Configuration) copyOverWithServer(overrides Server) {
	c.Server.Port.HTTP = utils.Coalesce(overrides.Port.HTTP, c.Server.Port.HTTP).(string)
	c.Server.Port.HTTPS = utils.Coalesce(overrides.Port.HTTPS, c.Server.Port.HTTPS).(string)
	c.Server.GZip = utils.Coalesce(overrides.GZip, c.Server.GZip).(bool)
	c.Server.Internals.ListeningAddress = utils.Coalesce(overrides.Internals.ListeningAddress, c.Server.Internals.ListeningAddress).(string)
	c.Server.Internals.ListeningPort = utils.Coalesce(overrides.Internals.ListeningPort, c.Server.Internals.ListeningPort).(string)
}

// --- TLS.
func (c *Configuration) copyOverWithTLS(overrides Server, file *string) {
	c.Server.TLS.Auto = utils.Coalesce(overrides.TLS.Auto, c.Server.TLS.Auto).(bool)
	c.Server.TLS.Email = utils.Coalesce(overrides.TLS.Email, c.Server.TLS.Email).(string)
	c.Server.TLS.CertFile = utils.Coalesce(overrides.TLS.CertFile, c.Server.TLS.CertFile).(string)
	c.Server.TLS.KeyFile = utils.Coalesce(overrides.TLS.KeyFile, c.Server.TLS.KeyFile).(string)
	c.Server.TLS.Override = utils.Coalesce(overrides.TLS.Override, c.Server.TLS.Override).(*tls.Config)

	c.Server.TLS.CertFile = patchAbsFilePath(c.Server.TLS.CertFile, file)
	c.Server.TLS.KeyFile = patchAbsFilePath(c.Server.TLS.KeyFile, file)
}

// --- TIMEOUT.
func (c *Configuration) copyOverWithTimeout(overrides Server) {
	c.Server.Timeout.Read = utils.Coalesce(overrides.Timeout.Read, c.Server.Timeout.Read).(time.Duration)
	c.Server.Timeout.ReadHeader = utils.Coalesce(overrides.Timeout.ReadHeader, c.Server.Timeout.ReadHeader).(time.Duration)
	c.Server.Timeout.Write = utils.Coalesce(overrides.Timeout.Write, c.Server.Timeout.Write).(time.Duration)
	c.Server.Timeout.Idle = utils.Coalesce(overrides.Timeout.Idle, c.Server.Timeout.Idle).(time.Duration)
	c.Server.Timeout.Handler = utils.Coalesce(overrides.Timeout.Handler, c.Server.Timeout.Handler).(time.Duration)
}

// --- UPSTREAM.
func (c *Configuration) copyOverWithUpstream(overrides Server) {
	c.Server.Upstream.Host = utils.Coalesce(overrides.Upstream.Host, c.Server.Upstream.Host).(string)
	c.Server.Upstream.Port = utils.Coalesce(overrides.Upstream.Port, c.Server.Upstream.Port).(string)
	c.Server.Upstream.Scheme = utils.Coalesce(overrides.Upstream.Scheme, c.Server.Upstream.Scheme).(string)
	c.Server.Upstream.BalancingAlgorithm = utils.Coalesce(overrides.Upstream.BalancingAlgorithm, c.Server.Upstream.BalancingAlgorithm).(string)
	c.Server.Upstream.Endpoints = utils.Coalesce(overrides.Upstream.Endpoints, c.Server.Upstream.Endpoints).([]string)
	c.Server.Upstream.HTTP2HTTPS = utils.Coalesce(overrides.Upstream.HTTP2HTTPS, c.Server.Upstream.HTTP2HTTPS).(bool)
	c.Server.Upstream.InsecureBridge = utils.Coalesce(overrides.Upstream.InsecureBridge, c.Server.Upstream.InsecureBridge).(bool)
	c.Server.Upstream.RedirectStatusCode = utils.Coalesce(overrides.Upstream.RedirectStatusCode, c.Server.Upstream.RedirectStatusCode).(int)
	c.Server.Upstream.HealthCheck.StatusCodes = utils.Coalesce(overrides.Upstream.HealthCheck.StatusCodes, c.Server.Upstream.HealthCheck.StatusCodes).([]string)
	c.Server.Upstream.HealthCheck.Timeout = utils.Coalesce(overrides.Upstream.HealthCheck.Timeout, c.Server.Upstream.HealthCheck.Timeout).(time.Duration)
	c.Server.Upstream.HealthCheck.Interval = utils.Coalesce(overrides.Upstream.HealthCheck.Interval, c.Server.Upstream.HealthCheck.Interval).(time.Duration)
	c.Server.Upstream.HealthCheck.Port = utils.Coalesce(overrides.Upstream.HealthCheck.Port, c.Server.Upstream.HealthCheck.Port).(string)
	c.Server.Upstream.HealthCheck.Scheme = utils.Coalesce(overrides.Upstream.HealthCheck.Scheme, c.Server.Upstream.HealthCheck.Scheme).(string)
	c.Server.Upstream.HealthCheck.AllowInsecure = utils.Coalesce(overrides.Upstream.HealthCheck.AllowInsecure, c.Server.Upstream.HealthCheck.AllowInsecure).(bool)

	c.Server.Upstream.Scheme = utils.IfEmpty(c.Server.Upstream.Scheme, SchemeWildcard)
}

// --- CACHE.
func (c *Configuration) copyOverWithCache(overrides Cache) {
	c.Cache.Hosts = utils.Coalesce(overrides.Hosts, c.Cache.Hosts).([]string)
	c.Cache.Password = utils.Coalesce(overrides.Password, c.Cache.Password).(string)
	c.Cache.DB = utils.Coalesce(overrides.DB, c.Cache.DB).(int)
	c.Cache.TTL = utils.Coalesce(overrides.TTL, c.Cache.TTL).(int)
	c.Cache.AllowedStatuses = utils.Coalesce(overrides.AllowedStatuses, c.Cache.AllowedStatuses).([]int)
	c.Cache.AllowedMethods = utils.Coalesce(overrides.AllowedMethods, c.Cache.AllowedMethods).([]string)

	c.Cache.AllowedMethods = append(c.Cache.AllowedMethods, "HEAD", "GET")
	c.Cache.AllowedMethods = slice.Unique(c.Cache.AllowedMethods)
}

// --- TRACING.
func (c *Configuration) copyOverWithTracing(overrides Tracing) {
	c.Tracing.JaegerEndpoint = utils.Coalesce(overrides.JaegerEndpoint, c.Tracing.JaegerEndpoint).(string)
	c.Tracing.Enabled = utils.Coalesce(overrides.Enabled, c.Tracing.Enabled).(bool)
	c.Tracing.SamplingRatio = utils.Coalesce(overrides.SamplingRatio, c.Tracing.SamplingRatio).(float64)
}

// --- LOG.
func (c *Configuration) copyOverWithLog(overrides Log) {
	c.Log.SentryDsn = utils.Coalesce(overrides.SentryDsn, c.Log.SentryDsn).(string)
	c.Log.SyslogProtocol = utils.Coalesce(overrides.SyslogProtocol, c.Log.SyslogProtocol).(string)
	c.Log.SyslogEndpoint = utils.Coalesce(overrides.SyslogEndpoint, c.Log.SyslogEndpoint).(string)
}

// Print - Shows the current configuration.
func Print() {
	obfuscatedConfig := Configuration{}

	err := copier.CopyWithOption(&obfuscatedConfig, &Config, copier.Option{DeepCopy: true})
	if err != nil {
		log.Errorf("Couldn't print configuration: %s", err)
		return
	}

	obfuscatedConfig.Cache.Password = PasswordOmittedValue

	for k, v := range obfuscatedConfig.Domains {
		v.Cache.Password = PasswordOmittedValue
		obfuscatedConfig.Domains[k] = v
	}

	log.Debug("Config Settings:\n")
	log.Debugf("%+v\n", obfuscatedConfig)
}

// GetDomains - Returns a list of domains.
func GetDomains() []DomainSet {
	domains := make(map[string]DomainSet)

	// add global upstream server...
	domains[Config.Server.Upstream.Host+utils.StringSeparatorOne+Config.Server.Upstream.Scheme] = DomainSet{
		Host:   Config.Server.Upstream.Host,
		Scheme: Config.Server.Upstream.Scheme,
	}

	for _, v := range Config.Domains {
		domains[v.Server.Upstream.Host+utils.StringSeparatorOne+v.Server.Upstream.Scheme] = DomainSet{
			Host:   v.Server.Upstream.Host,
			Scheme: v.Server.Upstream.Scheme,
		}
	}

	return getSliceFromMap(domains)
}

func getSliceFromMap(domains map[string]DomainSet) []DomainSet {
	domainsUnique := make([]DomainSet, 0, len(domains))
	for _, d := range domains {
		domainsUnique = append(domainsUnique, d)
	}

	return domainsUnique
}

// DomainConf - Returns the configuration for the requested domain (Global Access).
func DomainConf(domain string, scheme string) (Configuration, bool) {
	return Config.DomainConf(domain, scheme)
}

// DomainConf - Returns the configuration for the requested domain.
func (c Configuration) DomainConf(domain string, scheme string) (Configuration, bool) {
	var found bool

	// Memoization
	if c.domainsCache == nil {
		c.domainsCache = make(map[string]Configuration)
	}

	keyCache := fmt.Sprintf("%s%s%s", domain, utils.StringSeparatorOne, scheme)
	if val, ok := c.domainsCache[keyCache]; ok {
		log.Debugf("Cached configuration for %s", keyCache)
		return val, true
	}

	c.domainsCache[keyCache], found = c.domainConfLookup(utils.StripPort(domain), scheme)

	return c.domainsCache[keyCache], found
}

func (c Configuration) domainConfLookup(domain string, scheme string) (Configuration, bool) {
	// First round: host & scheme
	for _, v := range c.Domains {
		if v.Server.Upstream.Host == domain && v.Server.Upstream.Scheme == scheme {
			return v, true
		}
	}

	// Second round: host
	for _, v := range c.Domains {
		if v.Server.Upstream.Host == domain {
			return v, true
		}
	}

	// Third round: global
	if c.Server.Upstream.Host == domain {
		return c, true
	}

	return Configuration{}, false
}
