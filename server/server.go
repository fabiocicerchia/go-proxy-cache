package server

import (
	"log"
	"net/http"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
)

func CreateServerConfig(port string, timeout config.Timeout, certManager *autocert.Manager, certFile *string, keyFile *string) *http.Server {
	// TODO: COVERAGE
	mux := http.NewServeMux()

	// handlers
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodConnect {
			handler.HandleTunneling(w, r)
		} else {
			handler.HandleRequestAndProxy(w, r)
		}
	})
	mux.HandleFunc("/healthcheck", handler.HandleHealthcheck)

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  time.Duration(timeout.Read) * time.Second,
		WriteTimeout: time.Duration(timeout.Write) * time.Second,
		IdleTimeout:  time.Duration(timeout.Idle) * time.Second,
		Handler:      mux,
	}

	if port == "443" {
		tlsConfig, err := srvtls.TLSConfig(*certFile, *keyFile)
		if err != nil {
			log.Fatal(err)
			return nil
		}
		server.TLSConfig = tlsConfig

		if config.Config.Server.TLS.Auto {
			server.TLSConfig = certManager.TLSConfig()
		}
	}

	return server
}

// Start the GoProxyCache server.
func Start() {
	// Init configs
	config.InitConfigFromFileOrEnv("config.yml")

	serverConfig := config.Config.Server
	serverTlsConfig := serverConfig.TLS

	// Log setup values
	logger.LogSetup(serverConfig)

	// redis connect
	engine.Connect(config.Config.Cache)

	// ssl
	certManager := srvtls.InitCertManager(config.Config.Server.Forwarding.Host, serverTlsConfig.Email)

	// config server
	serverHTTP := CreateServerConfig(serverConfig.Port.HTTP, serverConfig.Timeout, nil, nil, nil)
	serverHTTPS := CreateServerConfig(serverConfig.Port.HTTPS, serverConfig.Timeout, certManager, &serverTlsConfig.CertFile, &serverTlsConfig.KeyFile)

	// start server
	go func() { log.Fatal(serverHTTP.ListenAndServe()) }()
	go func() { log.Fatal(serverHTTPS.ListenAndServeTLS("", "")) }()
}
