package queue

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
