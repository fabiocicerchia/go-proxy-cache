package jwt

import "strings"

func IsIncluded(paths []string, searchterm string) bool {
	for _, v := range paths {
		hasprefix := strings.HasPrefix(searchterm, v)
		if hasprefix {
			return true
		}
	}
	return false
}
