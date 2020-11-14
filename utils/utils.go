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

// func GetHeadersFromInterface(headers map[string]interface{}) map[string]string {
// 	headersConverted := make(map[string]string)
// 	for k, v := range headers {
// 		str := []string{v.(string)}

// 		headersConverted[k] = strings.Join(str, " ") // TODO: is correct join " " ?
// 	}
// 	return headersConverted
// }

func GetHeaders(headers map[string][]string) map[string]string {
	headersConverted := make(map[string]string)
	for k, v := range headers {
		str := []string{}
		for _, item := range v {
			str = append(str, item)
		}

		headersConverted[k] = strings.Join(str, " ") // TODO: is correct join " " ?
	}
	return headersConverted
}
