package utils

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

// ConvertToDuration - Converts a string to time.Duration
func ConvertToDuration(value string) time.Duration {
	duration, err := time.ParseDuration(value)
	if err != nil {
		return time.Duration(0)
	}
	return duration
}

// ConvertToInt - Converts a string to int
func ConvertToInt(value string) int {
	val, _ := strconv.Atoi(value)
	return val
}

// ConvertToIntSlice - Converts a slice of strings to a slice of ints
func ConvertToIntSlice(value []string) []int {
	values := []int{}
	for _, v := range value {
		values = append(values, ConvertToInt(v))
	}
	return values
}
