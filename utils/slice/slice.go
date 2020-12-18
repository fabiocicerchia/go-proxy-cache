package slice

import (
	"net/http"
	"strings"
)

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

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
	// TODO: COVERAGE

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
