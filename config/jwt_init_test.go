//go:build all || unit
// +build all unit

package config_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"testing"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

func newJwtConfigWithTestLogger(jwksURL string, issuer string, audience string) (*config.Jwt, *logrustest.Hook) {
	logger, hook := logrustest.NewNullLogger()

	return &config.Jwt{
		JwksUrl:  jwksURL,
		Issuer:   issuer,
		Audience: audience,
		Logger:   logger,
	}, hook
}

func TestInitJWTWarnsWhenIssuerAndAudienceAreMissing(t *testing.T) {
	jwtConfig, hook := newJwtConfigWithTestLogger("http://jwks.example.com/jwks.json", "", "")

	config.InitJWT(jwtConfig)

	entry := hook.LastEntry()
	assert.NotNil(t, entry, "a warning is expected when JWT is enabled without issuer/audience")
	assert.Equal(t, logrus.WarnLevel, entry.Level)
	assert.Contains(t, entry.Message, "issuer")
	assert.Contains(t, entry.Message, "audience")
}

func TestInitJWTWarnsWhenOnlyIssuerIsConfigured(t *testing.T) {
	jwtConfig, hook := newJwtConfigWithTestLogger("http://jwks.example.com/jwks.json", "issuer", "")

	config.InitJWT(jwtConfig)

	entry := hook.LastEntry()
	assert.NotNil(t, entry, "a warning is expected when the audience is missing")
	assert.Equal(t, logrus.WarnLevel, entry.Level)
}

func TestInitJWTWarnsWhenOnlyAudienceIsConfigured(t *testing.T) {
	jwtConfig, hook := newJwtConfigWithTestLogger("http://jwks.example.com/jwks.json", "", "audience")

	config.InitJWT(jwtConfig)

	entry := hook.LastEntry()
	assert.NotNil(t, entry, "a warning is expected when the issuer is missing")
	assert.Equal(t, logrus.WarnLevel, entry.Level)
}

func TestInitJWTDoesNotWarnWhenIssuerAndAudienceAreConfigured(t *testing.T) {
	jwtConfig, hook := newJwtConfigWithTestLogger("http://jwks.example.com/jwks.json", "issuer", "audience")

	config.InitJWT(jwtConfig)

	assert.Nil(t, hook.LastEntry(), "no warning is expected when both claims are configured")
}

func TestInitJWTDoesNotWarnWhenJWTIsDisabled(t *testing.T) {
	jwtConfig, hook := newJwtConfigWithTestLogger("", "", "")

	config.InitJWT(jwtConfig)

	assert.Nil(t, hook.LastEntry(), "no warning is expected when no JWKS URL is configured")
}

func TestCopyOverWithMergesJwtIssuerAndAudience(t *testing.T) {
	global := config.Configuration{}
	global.Jwt.Issuer = "global-issuer"
	global.Jwt.Audience = "global-audience"

	overridden := global
	overridden.CopyOverWith(config.Configuration{Jwt: config.Jwt{Issuer: "domain-issuer"}}, nil)

	assert.Equal(t, "domain-issuer", overridden.Jwt.Issuer, "the domain override wins")
	assert.Equal(t, "global-audience", overridden.Jwt.Audience, "an unset override keeps the global value")
}
