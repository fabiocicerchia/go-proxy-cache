package main

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/server"
)

var configFile string
var logLevel log.Level
var logFile string
var logFileHandle *os.File
var verboseFlag bool
var testFlag bool

// AppVersion - The go-proxy-cache's version.
const AppVersion = "0.3.0"

func initFlags() {
	var debug, version bool

	flag.BoolVar(&debug, "debug", false, "enable debug")
	flag.BoolVar(&testFlag, "test", false, "test configuration")
	flag.BoolVar(&verboseFlag, "verbose", false, "enable verbose")
	flag.BoolVar(&version, "version", false, "display version")
	flag.StringVar(&configFile, "config", "config.yml", "config file")
	flag.StringVar(&logFile, "log", "", "log file (default stdout)")

	flag.Parse()

	if version {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	if testFlag {
		if _, err := config.Validate(configFile); err != nil {
			fmt.Println("Configuration file not valid.")
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Configuration file valid.")
		os.Exit(0)
	}

	logLevel = log.InfoLevel
	if debug {
		logLevel = log.DebugLevel
	}
}

func getLogFileWriter(logFile string) *os.File {
	f, err := os.OpenFile(filepath.Clean(logFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatal(err)
	}

	return f
}

func initLogs() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		DisableColors:   false,
		FullTimestamp:   true,
		TimestampFormat: "2006/01/02 15:04:05",
	})

	log.SetReportCaller(verboseFlag)
	log.SetLevel(logLevel)
	log.SetOutput(os.Stdout)

	if logFile != "" {
		logFileHandle = getLogFileWriter(logFile)
		log.SetOutput(io.MultiWriter(logFileHandle))
		log.RegisterExitHandler(closeLogFile)
	}
}

func closeLogFile() {
	if logFileHandle != nil {
		logFileHandle.Close()
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	initFlags()
	initLogs()

	log.Debugf("                                                                        __")
	log.Debugf(".-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.")
	log.Debugf("|  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|")
	log.Debugf("|___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|")
	log.Debugf("|_____|            |__|                   |_____|\n\n")
	log.Debugf("Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License")
	log.Debugf("Repo: https://github.com/fabiocicerchia/go-proxy-cache\n\n")

	log.Debugf("Version: %s\n", AppVersion)
	log.Debugf("Go: %s\n", runtime.Version())
	log.Debugf("Threads: %d\n", runtime.NumCPU())
	log.Debugf("OS: %s\n", runtime.GOOS)
	log.Debugf("Arch: %s\n\n", runtime.GOARCH)

	server.Run(configFile)
}
