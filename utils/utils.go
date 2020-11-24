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
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// StringSeparatorOne - Main text separator, used for joins.
const StringSeparatorOne = "@@"

// StringSeparatorTwo - Secondary text separator, used for joins.
const StringSeparatorTwo = "--"

// GetEnv - Gets environment variable or default.
func GetEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

// IfEmpty - Returns value if not empty, fallback otherwise.
func IfEmpty(val string, fallback string) string {
	if val == "" {
		return fallback
	}

	return val
}

// ContainsInt - Checks if an int value is contained in a slice.
func ContainsInt(items []int, value int) bool {
	for _, v := range items {
		if v == value {
			return true
		}
	}
	return false
}

// ContainsString - Checks if a string value is contained in a slice.
func ContainsString(items []string, value string) bool {
	for _, v := range items {
		if v == value {
			return true
		}
	}
	return false
}

// GetByKeyCaseInsensitive - Retrieves value by key matched case-insensitively.
func GetByKeyCaseInsensitive(items http.Header, key string) interface{} {
	keyLower := strings.ToLower(key)
	for k, v := range items {
		if strings.ToLower(k) == keyLower {
			return v
		}
	}

	return nil
}

// Unique - Returns a slice with unique values
func Unique(slice []string) []string {
	keys := make(map[string]bool)
	list := []string{}

	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}

	return list
}

// LenSliceBytes - Returns total length of a slice of bytes
func LenSliceBytes(data [][]byte) int {
	l := 0
	for _, v := range data {
		l += len(v)
	}

	return l
}

// Coalesce - Returns the original value if the conditions is not met, fallback value otherwise.
func Coalesce(value interface{}, fallback interface{}, condition bool) interface{} {
	// TODO: COVERAGE
	if condition {
		value = fallback
	}

	return value
}

// ConvertToDuration - Converts a string to time.Duration
func ConvertToDuration(value string) time.Duration {
	// TODO: COVERAGE
	duration, err := time.ParseDuration(value)
	if err != nil {
		return time.Duration(0)
	}
	return duration
}

// ConvertToInt - Converts a string to int
func ConvertToInt(value string) int {
	// TODO: COVERAGE
	val, _ := strconv.Atoi(value)
	return val
}

// ConvertToIntSlice - Converts a slice of strings to a slice of ints
func ConvertToIntSlice(value []string) []int {
	// TODO: COVERAGE
	var values []int
	for _, v := range value {
		values = append(values, ConvertToInt(v))
	}
	return values
}
