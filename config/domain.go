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

// DomainConf - Returns the configuration for the requested domain (Global Access).
func DomainConf(domain string, scheme string) (Configuration, bool) {
	return Current().DomainConf(domain, scheme)
}

// DomainConf - Returns the configuration for the requested domain.
//
// Kept as a method for backward compatibility: it builds a throwaway snapshot
// so callers holding a Configuration value (rather than the published one)
// still resolve domains with the same precedence rules.
func (c *Configuration) DomainConf(domain string, scheme string) (Configuration, bool) {
	return NewSnapshot(*c, c.Domains).DomainConf(domain, scheme)
}
