package utils

import (
	"os"
)

// Get env var or default
func GetEnv(key, fallback *string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return *fallback
}
