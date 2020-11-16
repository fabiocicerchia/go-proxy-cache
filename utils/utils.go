package utils

import (
	"os"
	"strings"
)

// Get env var or default
func GetEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return fallback
}

// TODO: coverage
func GetHeaders(headers map[string][]string) map[string]interface{} {
	headersConverted := make(map[string]interface{})
	for k, v := range headers {
		str := []string{}
		for _, item := range v {
			str = append(str, item)
		}

		headersConverted[k] = strings.Join(str, " ") // TODO: is correct join " " ?
	}
	return headersConverted
}

func IfEmpty(val string, fallback string) string {
	if val == "" {
		return fallback
	}

	return val
}

func Contains(items []string, value string) bool {
	for _, v := range items {
		if v == value {
			return true
		}
	}
	return false
}

func GetByKeyCaseInsensitive(items map[string]interface{}, key string) interface{} {
	keyLower := strings.ToLower(key)
	for k, v := range items {
		if strings.ToLower(k) == keyLower {
			return v
		}
	}

	return nil
}

func CastToString(i interface{}) string {
	arr := i.([]string)
	if len(arr) > 0 {
		return arr[0]
	}

	return ""
}
