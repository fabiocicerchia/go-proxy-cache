package jwt

import (
	"regexp"
)

func IsExcluded(excludedPaths []string, requestPath string) bool {
	for _, v := range excludedPaths {
		re := regexp.MustCompile(v)
		isExcluded := re.MatchString(requestPath)
		if isExcluded {
			return true
		}
	}

	return false
}
