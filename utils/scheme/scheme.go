package scheme

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"strings"
)

// SchemeHTTPS - HTTPS scheme.
const SchemeHTTPS = "https"

// SchemeHTTP - HTTP scheme.
const SchemeHTTP = "http"

var allowedSchemes = map[string]string{"HTTP": SchemeHTTP, "HTTPS": SchemeHTTPS}

// NormalizeScheme - Normalize the URL scheme (http or https).
func NormalizeScheme(scheme string) string {
	if val, ok := allowedSchemes[strings.ToUpper(scheme)]; ok {
		return val
	}

	return ""
}
