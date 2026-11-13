package server

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import "context"

// controller - A background process that supplies the servers with routes.
//
// Declared here rather than imported so this file, and Run's signature with
// it, stays identical whichever build it is compiled into.
type controller interface {
	Run(ctx context.Context) error
}

// options - What Run was asked to do beyond reading the configuration file.
type options struct {
	// start - Builds the listeners and returns whatever drives them. Nil for
	// the ordinary case, where the configuration file describes the domains.
	start func(*Servers) (controller, error)
}

// Option - Changes how Run sets the servers up.
type Option func(*options)

func newOptions(opts []Option) options {
	var o options

	for _, apply := range opts {
		apply(&o)
	}

	return o
}
