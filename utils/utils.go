package utils

import (
	"net/http"
	"os"
	"strings"
)

const StringSeparatorOne = "@@"
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

// Contains - Checks if a value is contained in a slice.
func Contains(items []string, value string) bool {
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
