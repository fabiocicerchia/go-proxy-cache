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
	"github.com/jinzhu/copier"
	log "github.com/sirupsen/logrus"
)

// Print - Shows the current configuration.
func Print() {
	obfuscatedConfig := Configuration{}

	err := copier.CopyWithOption(&obfuscatedConfig, &Config, copier.Option{DeepCopy: true})
	if err != nil {
		log.Errorf("Couldn't print configuration: %s", err)
		return
	}

	obfuscatedConfig.Cache.Password = PasswordOmittedValue
	if obfuscatedConfig.Server.Purge.Secret != "" {
		obfuscatedConfig.Server.Purge.Secret = PasswordOmittedValue
	}

	for k, v := range obfuscatedConfig.Domains {
		v.Cache.Password = PasswordOmittedValue
		if v.Server.Purge.Secret != "" {
			v.Server.Purge.Secret = PasswordOmittedValue
		}
		obfuscatedConfig.Domains[k] = v
	}

	log.Debug("Config Settings:\n")
	log.Debugf("%+v\n", obfuscatedConfig)
}
