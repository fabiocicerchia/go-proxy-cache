//go:build all || unit
// +build all unit

package config_test

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2023 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fabiocicerchia/go-proxy-cache/config"
)

// TestTLSOverrideSurvivesMerge - server/tls.newDefaultTLSConfig dereferences
// config.Server.TLS.Override unconditionally, so a nil there is a startup
// SIGSEGV on every HTTPS listener.
func TestTLSOverrideSurvivesMerge(t *testing.T) {
	config.InitConfigFromFileOrEnv("../test/full-setup/config.yml")

	assert.NotNil(t, config.Config.Server.TLS.Override)

	for name, domain := range config.Config.Domains {
		assert.NotNil(t, domain.Server.TLS.Override, "domain %s", name)
	}
}

const mergeYAML = `
server:
  upstream:
    host: global.example
cache:
  eviction_policy: allkeys-lru
  negative_ttl:
    404: 30
    502: 10
circuit_breaker:
  threshold: 9
log:
  format: "$host custom"
domains:
  inherits:
    server:
      upstream:
        host: inherits.example
  overrides:
    server:
      upstream:
        host: overrides.example
      tls:
        hsts:
          enabled: true
    cache:
      eviction_policy: volatile-ttl
      negative_ttl:
        404: 5
  clears:
    server:
      upstream:
        host: clears.example
    cache:
      negative_ttl: {}
`

// loadYAML - Loads yaml as the config file, restoring the global Config afterwards.
func loadYAML(t *testing.T, yaml string) {
	t.Helper()

	orig := config.Config
	t.Cleanup(func() { config.Config = orig })

	file := filepath.Join(t.TempDir(), "config.yml")
	assert.NoError(t, os.WriteFile(file, []byte(yaml), 0o600))

	config.InitConfigFromFileOrEnv(file)
}

// TestYAMLMergeKeepsCacheFields - negative_ttl and eviction_policy must survive
// the YAML load, globally and per domain. A domain setting its own
// negative_ttl replaces the global map (no key-by-key merge); `{}` clears it.
func TestYAMLMergeKeepsCacheFields(t *testing.T) {
	loadYAML(t, mergeYAML)

	assert.Equal(t, map[int]int{404: 30, 502: 10}, config.Config.Cache.NegativeTTL)
	assert.Equal(t, "allkeys-lru", config.Config.Cache.EvictionPolicy)
	assert.Equal(t, uint32(9), config.Config.CircuitBreaker.Threshold)
	assert.Equal(t, "$host custom", config.Config.Log.Format)

	inherits := config.Config.Domains["inherits"]
	assert.Equal(t, map[int]int{404: 30, 502: 10}, inherits.Cache.NegativeTTL)
	assert.Equal(t, "allkeys-lru", inherits.Cache.EvictionPolicy)
	assert.Equal(t, uint32(9), inherits.CircuitBreaker.Threshold)
	assert.False(t, inherits.Server.TLS.HSTS.Enabled)

	overrides := config.Config.Domains["overrides"]
	assert.Equal(t, map[int]int{404: 5}, overrides.Cache.NegativeTTL)
	assert.Equal(t, "volatile-ttl", overrides.Cache.EvictionPolicy)
	assert.True(t, overrides.Server.TLS.HSTS.Enabled)

	assert.Empty(t, config.Config.Domains["clears"].Cache.NegativeTTL)
}

// notMerged - Fields CopyOverWith intentionally does not take from overrides.
var notMerged = map[string]string{
	"Domains":      "only the config file defines domains, set directly by InitConfigFromFileOrEnv",
	"Jwt.JwkCache": "runtime state, built by InitJWT after merging",
	"Jwt.Context":  "runtime state, always reset to context.Background()",
	"Jwt.Logger":   "runtime state, always reset to a new logger",
}

// TestCopyOverWithIsTotal - Every exported config leaf must be carried by
// CopyOverWith: set in overrides it wins, left unset it keeps the base value.
// Adding a field to a config struct without merging it fails here; if the
// field is deliberately not overridable, list it in notMerged with a reason.
func TestCopyOverWithIsTotal(t *testing.T) {
	var paths []string
	collectLeaves(reflect.TypeOf(config.Configuration{}), "", &paths)
	assert.Greater(t, len(paths), 50)

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			// override wins
			base, overrides := config.Configuration{}, config.Configuration{}
			want := nonZero(field(&overrides, path))
			base.CopyOverWith(overrides, nil)
			assertCarried(t, path, want, field(&base, path))

			// unset override keeps base
			base, overrides = config.Configuration{}, config.Configuration{}
			want = nonZero(field(&base, path))
			base.CopyOverWith(overrides, nil)
			assertCarried(t, path, want, field(&base, path))
		})
	}
}

func collectLeaves(typ reflect.Type, prefix string, paths *[]string) {
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		path := prefix + f.Name
		if !f.IsExported() || notMerged[path] != "" {
			continue
		}
		if f.Type.Kind() == reflect.Struct {
			collectLeaves(f.Type, path+".", paths)
			continue
		}
		*paths = append(*paths, path)
	}
}

func field(c *config.Configuration, path string) reflect.Value {
	v := reflect.ValueOf(c).Elem()
	for _, name := range strings.Split(path, ".") {
		v = v.FieldByName(name)
	}

	return v
}

// nonZero - Sets v to a non-zero value of its type and returns it.
func nonZero(v reflect.Value) interface{} {
	switch v.Kind() {
	case reflect.String:
		v.SetString("/nonzero") // absolute, so TLS file paths are not rewritten
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int64:
		v.SetInt(7)
	case reflect.Uint32:
		v.SetUint(7)
	case reflect.Float64:
		v.SetFloat(0.7)
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 1, 1)
		nonZero(s.Index(0))
		v.Set(s)
	case reflect.Map:
		m := reflect.MakeMap(v.Type())
		k, e := reflect.New(v.Type().Key()).Elem(), reflect.New(v.Type().Elem()).Elem()
		nonZero(k)
		nonZero(e)
		m.SetMapIndex(k, e)
		v.Set(m)
	case reflect.Ptr:
		v.Set(reflect.New(v.Type().Elem()))
	default:
		panic("nonZero: unsupported kind " + v.Kind().String() + ", extend it")
	}

	return v.Interface()
}

func assertCarried(t *testing.T, path string, want interface{}, got reflect.Value) {
	t.Helper()

	if path == "Cache.AllowedMethods" { // HEAD and GET are always appended
		assert.Subset(t, got.Interface(), want)
		return
	}

	assert.Equal(t, want, got.Interface())
}
