package config

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/lestrrat-go/jwx/v2/jwk"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"github.com/fabiocicerchia/go-proxy-cache/utils/scheme"
)

// PasswordOmittedValue - Replacement value when showing passwords in configuration.
const PasswordOmittedValue = "*** OMITTED ***"

// SchemeWildcard - Label to be shown when no schema (http/https) is selected.
const SchemeWildcard = "*"

// jwksURLEnvPrefix - Prefix of the per-domain JWKS URL environment variable,
// completed with the domain name (JWT_JWKS_URL_example_com).
const jwksURLEnvPrefix = "JWT_JWKS_URL_"

func newFromEnv() Configuration {
	envConfig := Configuration{}

	err := envconfig.Process("", &envConfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	return envConfig
}

func getFromYaml(file string) (Configuration, error) {
	YamlConfig := Configuration{}

	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return YamlConfig, err
	}

	err = yaml.UnmarshalStrict(data, &YamlConfig)

	if err != nil {
		return YamlConfig, err
	}

	YamlConfig.Server.Upstream.Scheme = scheme.NormalizeScheme(YamlConfig.Server.Upstream.Scheme)

	return YamlConfig, err
}

// InitConfigFromFileOrEnv - Init the configuration in sequence: from a YAML file, from environment variables,
// then defaults.
func InitConfigFromFileOrEnv(file string) {
	Config.CopyOverWith(newFromEnv(), nil)

	YamlConfig := loadYAMLFile(file)

	// allow only the config file to specify overrides per domain
	Config.Domains = YamlConfig.Domains

	// JWT for the global configuration. Without this the global JwkCache was
	// never initialised (only per-domain configs got InitJWT), so a JWKS URL
	// configured globally could never be used for validation.
	if Config.Jwt.JwksUrl != "" {
		InitJWT(&Config.Jwt)
	}

	// DOMAINS
	copyGlobalOverDomainConfig(file)
}

func loadYAMLFile(file string) (YamlConfig Configuration) {
	_, err := os.Stat(file)
	if !os.IsNotExist(err) {
		YamlConfig, err = getFromYaml(file)
		if err != nil {
			log.Fatalf("Cannot unmarshal YAML: %s\n", err)
		}

		Config.CopyOverWith(YamlConfig, &file)
	}

	return YamlConfig
}

// InitJWT - Configure the jwk auto-refresh and save it into the JWT config
func InitJWT(jwtConfig *Jwt) {
	if jwtConfig.Context == nil {
		jwtConfig.Context = context.Background()
	}

	refreshIntervalDuration := time.Duration(jwtConfig.JwksRefreshInterval) * time.Minute
	jwtKeyFetcher := jwk.NewCache(jwtConfig.Context, jwk.WithRefreshWindow(refreshIntervalDuration))

	// Registering an empty URL is pointless and would only produce errors at
	// fetch time; validation is skipped anyway when no JWKS URL is configured.
	if jwtConfig.JwksUrl != "" {
		jwtKeyFetcher.Register(
			jwtConfig.JwksUrl,
			jwk.WithMinRefreshInterval(refreshIntervalDuration),
		)
	}

	jwtConfig.JwkCache = jwtKeyFetcher
}

func copyGlobalOverDomainConfig(file string) {
	if Config.Domains != nil {
		domains := Config.Domains
		for k, v := range domains {
			domain := Config
			domain.CopyOverWith(v, &file)
			domain.Domains = Domains{}
			domainName := k
			_, isJWKSUrl := os.LookupEnv(jwksURLEnvPrefix + domainName)
			if isJWKSUrl {
				domain.Jwt.JwksUrl = os.Getenv(jwksURLEnvPrefix + domainName)
			}
			if domain.Jwt.JwksUrl != "" {
				InitJWT(&domain.Jwt)
			}
			domains[k] = domain
		}

		Config.Domains = domains
	}
}

// Validate - Validate a YAML config file is syntactically valid.
func Validate(file string) (bool, error) {
	_, err := getFromYaml(file)
	return err != nil, err
}

func patchAbsFilePath(filePath string, relativeTo *string) string {
	abs, err := os.Getwd()

	if err == nil && relativeTo != nil && *relativeTo != "" {
		abs, err = filepath.Abs(*relativeTo)
		abs = filepath.Dir(abs)
	}

	if err == nil {
		if filePath != "" && !strings.HasPrefix(filePath, "/") {
			return filepath.Join(abs, filepath.Clean(filePath))
		}
	}

	return filePath
}
