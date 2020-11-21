package server

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/acme/autocert"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server/balancer"
	"github.com/fabiocicerchia/go-proxy-cache/server/handler"
	"github.com/fabiocicerchia/go-proxy-cache/server/logger"
	srvtls "github.com/fabiocicerchia/go-proxy-cache/server/tls"
)

// CreateServerConfig - Generates the http.Server configuration.
func CreateServerConfig(
	port string,
	timeout config.Timeout,
	certManager *autocert.Manager,
	certPair *srvtls.CertificatePair,
) *http.Server {
	mux := http.NewServeMux()

	// handlers
	mux.HandleFunc("/", handler.HandleRequest)
	mux.HandleFunc("/healthcheck", handler.HandleHealthcheck)

	muxWithMiddlewares := http.TimeoutHandler(
		mux,
		time.Duration(timeout.Handler)*time.Second,
		"Timed Out\n",
	)

	// TODO: TEST timeouts with custom handlers
	server := &http.Server{
		Addr:              ":" + port,
		ReadTimeout:       time.Duration(timeout.Read) * time.Second,
		WriteTimeout:      time.Duration(timeout.Write) * time.Second,
		IdleTimeout:       time.Duration(timeout.Idle) * time.Second,
		ReadHeaderTimeout: time.Duration(timeout.ReadHeader) * time.Second,
		Handler:           muxWithMiddlewares,
	}

	if port == config.GetPortHTTPS() {
		srvtls.ServerOverrides(server, certManager, certPair)
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
	)
	serverHTTPS := CreateServerConfig(
		serverConfig.Port.HTTPS,
		serverConfig.Timeout,
		certManager,
		&srvtls.CertificatePair{
			Cert: serverTLSConfig.CertFile,
			Key:  serverTLSConfig.KeyFile,
		},
	)

	// lb
	balancer.InitRoundRobin(config.Config.Server.Forwarding.Endpoints)

	// start server http & https
	go func() { log.Fatal(serverHTTPS.ListenAndServeTLS("", "")) }()
	log.Fatal(serverHTTP.ListenAndServe())
}
