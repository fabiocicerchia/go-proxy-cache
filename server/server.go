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
	mux := http.NewServeMux()

	// handlers
	mux.HandleFunc("/", handler.HandleRequest)
	mux.HandleFunc("/healthcheck", handler.HandleHealthcheck)

	muxWithMiddlewares := http.TimeoutHandler(
		mux,
		time.Duration(timeout.Handler)*time.Second,
		"Timed Out\n",
	)

	server := &http.Server{
		Addr:              ":" + port,
		ReadTimeout:       time.Duration(timeout.Read) * time.Second,
		WriteTimeout:      time.Duration(timeout.Write) * time.Second,
		IdleTimeout:       time.Duration(timeout.Idle) * time.Second,
		ReadHeaderTimeout: time.Duration(timeout.ReadHeader) * time.Second,
		Handler:           muxWithMiddlewares,
	}

	if port == config.GetPortHTTPS() {
		srvtls.ServerOverrides(server, certManager, certFile, keyFile)
	}

	return server
}

// Start the GoProxyCache server.
func Start() {
	// Init configs
	config.InitConfigFromFileOrEnv("config.yml")

	serverConfig := config.Config.Server
	serverTLSConfig := serverConfig.TLS

	// Log setup values
	logger.LogSetup(serverConfig)

	// redis connect
	engine.Connect(config.Config.Cache)

	// ssl
	certManager := srvtls.InitCertManager(config.Config.Server.Forwarding.Host, serverTLSConfig.Email)

	// config server http & https
	serverHTTP := CreateServerConfig(
		serverConfig.Port.HTTP,
		serverConfig.Timeout,
		nil,
		nil,
		nil,
	)
	serverHTTPS := CreateServerConfig(
		serverConfig.Port.HTTPS,
		serverConfig.Timeout,
		certManager,
		&serverTLSConfig.CertFile,
		&serverTLSConfig.KeyFile,
	)

	// start server http & https
	go func() { log.Fatal(serverHTTPS.ListenAndServeTLS("", "")) }()
	log.Fatal(serverHTTP.ListenAndServe())
}
