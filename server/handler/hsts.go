package handler

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import "github.com/go-http-utils/headers"

// SetHSTSHeader - Adds Strict-Transport-Security to HTTPS responses when
// enabled, mitigating MITM/SSL-stripping downgrade attacks (#11).
func (rc RequestCall) SetHSTSHeader() {
	hsts := rc.DomainConfig.Server.TLS.HSTS
	if !hsts.Enabled || rc.GetScheme() != SchemeHTTPS {
		return
	}

	rc.Response.Header().Set(headers.StrictTransportSecurity, hsts.Header())
}
