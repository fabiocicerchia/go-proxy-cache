package utils

import (
	"os"
)

// Get env var or default
func GetEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

// Get the url for a given proxy condition
func GetProxyUrl() string {
	return GetEnv("FORWARD_TO", "")
}
