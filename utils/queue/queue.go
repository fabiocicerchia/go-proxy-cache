package queue

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"time"

	"github.com/sdeoras/dispatcher"
)

// MaxConcurrency - How many concurrent functions can be executed.
// TODO: Make it customizable?
const MaxConcurrency = 10

// Dispatcher - Global queue dispatcher.
var Dispatcher dispatcher.Dispatcher

// Init - Init a new dispatcher.
func Init() {
	Dispatcher = dispatcher.New(MaxConcurrency)
}

// WaitForCompletion - Waits all functions are completed.
func WaitForCompletion() {
	for Dispatcher.IsRunning() {
		time.Sleep(time.Second)
	}
}
