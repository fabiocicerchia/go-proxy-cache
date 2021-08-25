package convert

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"strconv"
	"time"
)

// ToDuration - Converts a string to time.Duration.
func ToDuration(value string) time.Duration {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return time.Duration(0)
	}

	return duration
}

// ToInt - Converts a string to int.
func ToInt(value string) int {
	val, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}

	return val
}

// ToIntSlice - Converts a slice of strings to a slice of ints.
func ToIntSlice(value []string) []int {
	values := []int{}
	for _, v := range value {
		values = append(values, ToInt(v))
	}

	return values
}
