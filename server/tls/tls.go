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

	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"golang.org/x/crypto/acme/autocert"
)

var httpsDomains []string
var tlsConfig *crypto_tls.Config

// G402 (CWE-295): TLS MinVersion too low. (Confidence: HIGH, Severity: HIGH)
// It can be ignored as it is customisable, but the default is TLSv1.2.
var defaultTlsConfig = &crypto_tls.Config{
	// Causes servers to use Go's default ciphersuite preferences,
	// which are tuned to avoid attacks. Does nothing on clients.
	PreferServerCipherSuites: true,
	CurvePreferences:         config.Config.Server.TLS.Override.CurvePreferences,
	MinVersion:               config.Config.Server.TLS.Override.MinVersion,
	MaxVersion:               config.Config.Server.TLS.Override.MaxVersion,
	CipherSuites:             config.Config.Server.TLS.Override.CipherSuites,
} // #nosec

var errMissingCertificate = errors.New("missing certificate")
var errMissingCertificateOrKey = errors.New("missing certificate file and/or key file")

// ServerOverrides - Overrides the http.Server configuration for TLS.
func ServerOverrides(domain string, server *http.Server, domainConfig config.Server) (err error) {
	if domainConfig.TLS.Auto {
		certManager := InitCertManager(domainConfig.Upstream.Host, domainConfig.TLS.Email)
		server.TLSConfig = certManager.TLSConfig()

		return nil
	}

	tlsConfig, err = Config(domain, domainConfig.TLS)
	if err != nil {
		return err
	}

	server.TLSConfig = tlsConfig
	// TODO: check this: server.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),

	return nil
}

// Config - Returns a TLS configuration.
func Config(domain string, domainConfigTLS config.TLS) (*crypto_tls.Config, error) {
	if domainConfigTLS.CertFile == "" || domainConfigTLS.KeyFile == "" {
		return nil, errMissingCertificateOrKey
	}

	cert, err := crypto_tls.LoadX509KeyPair(domainConfigTLS.CertFile, domainConfigTLS.KeyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := defaultTlsConfig

	// If GetCertificate is nil or returns nil, then the certificate is
	// retrieved from NameToCertificate. If NameToCertificate is nil, the
	// best element of Certificates will be used.
	// Ref: https://golang.org/pkg/crypto/tls/#Config.GetCertificate
	tlsConfig.Certificates = append(tlsConfig.Certificates, cert)

	// TODO: THIS COULD LEAD TO CONFLICTS WHEN SHARING THE SAME PORT.
	if domainConfigTLS.Override != nil {
		tlsConfig.CurvePreferences = domainConfigTLS.Override.CurvePreferences
		tlsConfig.MinVersion = domainConfigTLS.Override.MinVersion
		tlsConfig.MaxVersion = domainConfigTLS.Override.MaxVersion
		tlsConfig.CipherSuites = domainConfigTLS.Override.CipherSuites
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

	httpsDomains = append(httpsDomains, host)

	certManager := &autocert.Manager{
		Cache:      autocert.DirCache(cacheDir),
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(httpsDomains...),
		Email:      email,
	}

	return certManager
}
