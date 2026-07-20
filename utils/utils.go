package utils

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net"
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
	if val == "" {
		return ""
	}

	// net.SplitHostPort correctly handles IPv6 literals wrapped in brackets,
	// e.g. "[::1]:8080". The naive strings.Split(val, ":") approach mangled
	// those addresses (and bare IPv6 literals), producing invalid hosts.
	if host, _, err := net.SplitHostPort(val); err == nil {
		return host
	}

	// No port present. Strip surrounding brackets from a bracketed IPv6 literal.
	if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
		return val[1 : len(val)-1]
	}

	return val
}
