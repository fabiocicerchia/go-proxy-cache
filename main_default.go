//go:build !k8s
// +build !k8s

package main

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import "github.com/fabiocicerchia/go-proxy-cache/server"

// registerExtraFlags - No extra flags in the standalone build.
func registerExtraFlags() {}

// applyExtraFlags - Nothing to apply.
func applyExtraFlags() {}

// runOptions - The standalone proxy reads its domains from the config file.
func runOptions() []server.Option { return nil }
