package utils

import (
	"os"
)

// Get env var or default
func GetEnv(key string, fallback *string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	if fallback != nil {
		return fallback
	}

	return ""
}

// Get the url for a given proxy condition
func GetProxyUrl() string {
	forward_to := GetEnv("FORWARD_TO")

	return forward_to
}
