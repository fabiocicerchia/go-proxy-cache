package tls

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	crypto_tls "crypto/tls"
	"io/ioutil"
	"net/http"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"golang.org/x/crypto/acme/autocert"
)

// CertificatePair - Pair of certificate and key.
type CertificatePair struct {
	Cert string
	Key  string
}

// ServerOverrides - Overrides the http.Server configuration for TLS.
func ServerOverrides(
	domain string,
	server *http.Server,
	certPair *CertificatePair,
) {
	domainConfig := config.DomainConf(domain)

	tlsConfig, err := Config(*&certPair.Cert, *&certPair.Key)
	if err != nil {
		log.Fatal(err)
		return
	}
	server.TLSConfig = tlsConfig

	if domainConfig.Server.TLS.Auto {
		certManager := InitCertManager(domainConfig.Server.Forwarding.Host, domainConfig.Server.TLS.Email)

		server.TLSConfig = certManager.TLSConfig()
	}
}

// Config - Returns a TLS configuration.
func Config(certFile string, keyFile string) (*crypto_tls.Config, error) {
	cert, err := crypto_tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &crypto_tls.Config{
		// TODO: SINCE IT IS NOT OVERRIDABLE IT IS FINE...
		PreferServerCipherSuites: config.Config.Server.TLS.Override.PreferServerCipherSuites,
		CurvePreferences:         config.Config.Server.TLS.Override.CurvePreferences,
		MinVersion:               config.Config.Server.TLS.Override.MinVersion,
		MaxVersion:               config.Config.Server.TLS.Override.MaxVersion,
		CipherSuites:             config.Config.Server.TLS.Override.CipherSuites,
		Certificates:             []crypto_tls.Certificate{cert},
	}

	return tlsConfig, nil
}

// InitCertManager - Initialise the Certification Manager for auto generation.
func InitCertManager(host string, email string) *autocert.Manager {
	cacheDir, err := ioutil.TempDir("", "cache_dir")
	if err != nil {
		log.Fatal(err)
		return nil
	}

	certManager := &autocert.Manager{
		Cache:      autocert.DirCache(cacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(host),
		Email:      email,
	}

	return certManager
}
