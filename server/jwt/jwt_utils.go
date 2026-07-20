package jwt

import (
	"regexp"
	"sync"
)

// compiledPatterns caches compiled regexes so that excluded paths are not
// recompiled on every request (JWTHandler runs on the hot path).
var (
	compiledPatterns   = make(map[string]*regexp.Regexp)
	compiledPatternsMu sync.RWMutex
)

func getCompiledPattern(pattern string) *regexp.Regexp {
	compiledPatternsMu.RLock()
	re, ok := compiledPatterns[pattern]
	compiledPatternsMu.RUnlock()
	if ok {
		return re
	}

	re = regexp.MustCompile(pattern)

	compiledPatternsMu.Lock()
	compiledPatterns[pattern] = re
	compiledPatternsMu.Unlock()

	return re
}

func IsExcluded(excludedPaths []string, requestPath string) bool {
	for _, v := range excludedPaths {
		re := getCompiledPattern(v)
		isExcluded := re.MatchString(requestPath)
		if isExcluded {
			return true
		}
	}

	return false
}
