package utils

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"os"
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

// Coalesce - Returns the original value if the conditions is not met, fallback value otherwise.
func Coalesce(value interface{}, fallback interface{}) interface{} {
	if IsEmpty(value) {
		value = fallback
	}

	return value
}

// IsEmpty - Checks whether a value is empty.
func IsEmpty(value interface{}) bool {
	switch t := value.(type) {
	case int:
		return t == 0
	case string:
		return t == ""
	case bool:
		return !t
	case []int:
		return len(t) == 0 || IsEmpty(t[0])
	case []string:
		return len(t) == 0 || IsEmpty(t[0])
	case time.Duration:
		return t == 0
	default:
		return value == nil
	}
}

// StripPort - Removes the port from a string like hostname:port.
func StripPort(val string) string {
	valParts := strings.Split(val, ":")

	max := len(valParts) - 1
	if max <= 0 {
		max = 1
	}

	return strings.Join(valParts[:max], ":")
}
