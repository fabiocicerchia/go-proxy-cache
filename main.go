package main

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/server"
	"github.com/fabiocicerchia/go-proxy-cache/telemetry/metrics"
)

var configFile string
var logFile string
var verboseFlag bool
var testFlag bool

// AppVersion - The go-proxy-cache's version.
const AppVersion = "1.3.0"

// GitCommit - The go-proxy-cache's git commit reference.
const GitCommit = "NA"

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
		printVersion()
	}

	if testFlag {
		testConfiguration(configFile)
	}

	if debug {
		logger.SetDebugLevel()
	}
}

func printVersion() {
	fmt.Println(AppVersion)
	os.Exit(0)
}

func testConfiguration(configFile string) {
	if _, err := config.Validate(configFile); err != nil {
		fmt.Println("Configuration file not valid.")
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Configuration file valid.")
	os.Exit(0)
}

func printBanner() {
	log.Debugf("                                                                        __")
	log.Debugf(".-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.")
	log.Debugf("|  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|")
	log.Debugf("|___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|")
	log.Debugf("|_____|            |__|                   |_____|\n\n")
	log.Debugf("Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License")
	log.Debugf("Repo: https://github.com/fabiocicerchia/go-proxy-cache\n\n")
}

func printSupport() {
	fmt.Println("💗 Support the Project 💗")
	fmt.Println("")
	fmt.Println("This project is only maintained by one person, [Fabio Cicerchia](https://github.com/fabiocicerchia).")
	fmt.Println("It started as a simple caching service, now it has a lot of pro functionalities just for FREE 😎")
	fmt.Println("Maintaining a project is a very time consuming activity, especially when done alone 💪")
	fmt.Println("I really want to make this project better and become super cool 🚀")
	fmt.Println("")
	fmt.Println("Two commercial versions have been planned: [PRO and PREMIUM](https://kodebeat.com/goproxycache.html).")
	fmt.Println("")
	fmt.Println("The development of the COMMUNITY version will continue, but priority will be given to the [COMMERCIAL versions](https://kodebeat.com/goproxycache.html).")
	fmt.Println("  - If you'd like to support this open-source project I'll appreciate any kind of [contribution](https://github.com/sponsors/fabiocicerchia).")
	fmt.Println("  - If you'd like to sponsor the commercial version, please [get in touch with me](mail:info@fabiocicerchia.it).")
	fmt.Println("")
	fmt.Println("---")
	fmt.Println("")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	initFlags()
	logger.InitLogs(verboseFlag, logFile)

	printBanner()
	printSupport()

	log.Debugf("Version: %s\n", AppVersion)
	log.Debugf("Go: %s\n", runtime.Version())
	log.Debugf("Threads: %d\n", runtime.NumCPU())
	log.Debugf("OS: %s\n", runtime.GOOS)
	log.Debugf("Arch: %s\n\n", runtime.GOARCH)

	metrics.SetBuildInfo(GitCommit, AppVersion)

	server.Run(AppVersion, configFile)
}
