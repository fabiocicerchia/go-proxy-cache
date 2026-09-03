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
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

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

// domainsCacheMu - Guards access to Configuration.domainsCache.
var domainsCacheMu sync.RWMutex

// DomainConf - Returns the configuration for the requested domain (Global Access).
func DomainConf(domain string, scheme string) (Configuration, bool) {
	return Config.DomainConf(domain, scheme)
}

// DomainConf - Returns the configuration for the requested domain.
//
// The result is memoized on the receiver. A pointer receiver is required so the
// cache persists across calls (a value receiver mutated a throwaway copy, so the
// memoization never actually took effect and every request re-scanned all
// domains). Access is guarded by domainsCacheMu since DomainConf runs on the
// request path and is invoked concurrently.
func (c *Configuration) DomainConf(domain string, scheme string) (Configuration, bool) {
	keyCache := fmt.Sprintf("%s%s%s", domain, utils.StringSeparatorOne, scheme)

	domainsCacheMu.RLock()
	if c.domainsCache != nil {
		if val, ok := c.domainsCache[keyCache]; ok {
			domainsCacheMu.RUnlock()
			log.Debugf("Cached configuration for %s", keyCache)
			return val, true
		}
	}
	domainsCacheMu.RUnlock()

	conf, found := c.domainConfLookup(utils.StripPort(domain), scheme)

	// Only cache positive lookups. Caching a miss would store a zero-value
	// Configuration that a later call would return as a hit (found == true).
	if found {
		domainsCacheMu.Lock()
		if c.domainsCache == nil {
			c.domainsCache = make(map[string]Configuration)
		}
		c.domainsCache[keyCache] = conf
		domainsCacheMu.Unlock()
	}

	return conf, found
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
