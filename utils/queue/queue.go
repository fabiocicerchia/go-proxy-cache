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

// TODO: Make it customizable?
const MaxConcurrency = 10

var Dispatcher dispatcher.Dispatcher

func Init() {
	Dispatcher = dispatcher.New(MaxConcurrency)
}

func WaitForCompletion() {
	for Dispatcher.IsRunning() {
		time.Sleep(time.Second)
	}
}
