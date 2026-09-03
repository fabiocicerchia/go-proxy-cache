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
	"crypto/tls"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
)

// CopyOverWith - Copies the Configuration over another (preserving not defined settings).
func (c *Configuration) CopyOverWith(overrides Configuration, file *string) {
	c.copyOverWithServer(overrides.Server)
	c.copyOverWithTLS(overrides.Server, file)
	c.copyOverWithTimeout(overrides.Server)
	c.copyOverWithUpstream(overrides.Server)
	c.copyOverWithCache(overrides.Cache)
	c.copyOverWithTracing(overrides.Tracing)
	c.copyOverWithLog(overrides.Log)
	c.copyOverWithJwt(overrides.Jwt)
}

// --- SERVER.
func (c *Configuration) copyOverWithServer(overrides Server) {
	c.Server.Port.HTTP = utils.Coalesce(overrides.Port.HTTP, c.Server.Port.HTTP).(string)
	c.Server.Port.HTTPS = utils.Coalesce(overrides.Port.HTTPS, c.Server.Port.HTTPS).(string)
	c.Server.GZip = utils.Coalesce(overrides.GZip, c.Server.GZip).(bool)
	c.Server.Internals.ListeningAddress = utils.Coalesce(overrides.Internals.ListeningAddress, c.Server.Internals.ListeningAddress).(string)
	c.Server.Internals.ListeningPort = utils.Coalesce(overrides.Internals.ListeningPort, c.Server.Internals.ListeningPort).(string)
	c.Server.Purge.AllowedIPs = utils.Coalesce(overrides.Purge.AllowedIPs, c.Server.Purge.AllowedIPs).([]string)
	c.Server.Purge.Secret = utils.Coalesce(overrides.Purge.Secret, c.Server.Purge.Secret).(string)
	c.Server.Purge.SecretHeader = utils.Coalesce(overrides.Purge.SecretHeader, c.Server.Purge.SecretHeader).(string)
}

// --- TLS.
func (c *Configuration) copyOverWithTLS(overrides Server, file *string) {
	c.Server.TLS.Auto = utils.Coalesce(overrides.TLS.Auto, c.Server.TLS.Auto).(bool)
	c.Server.TLS.Email = utils.Coalesce(overrides.TLS.Email, c.Server.TLS.Email).(string)
	c.Server.TLS.CertFile = utils.Coalesce(overrides.TLS.CertFile, c.Server.TLS.CertFile).(string)
	c.Server.TLS.KeyFile = utils.Coalesce(overrides.TLS.KeyFile, c.Server.TLS.KeyFile).(string)
	c.Server.TLS.Override = utils.Coalesce(overrides.TLS.Override, c.Server.TLS.Override).(*tls.Config)
	c.Server.TLS.CertCacheDir = utils.Coalesce(overrides.TLS.CertCacheDir, c.Server.TLS.CertCacheDir).(string)

	c.Server.TLS.CertFile = patchAbsFilePath(c.Server.TLS.CertFile, file)
	c.Server.TLS.KeyFile = patchAbsFilePath(c.Server.TLS.KeyFile, file)
}

// --- TIMEOUT.
func (c *Configuration) copyOverWithTimeout(overrides Server) {
	c.Server.Timeout.Read = utils.Coalesce(overrides.Timeout.Read, c.Server.Timeout.Read).(time.Duration)
	c.Server.Timeout.ReadHeader = utils.Coalesce(overrides.Timeout.ReadHeader, c.Server.Timeout.ReadHeader).(time.Duration)
	c.Server.Timeout.Write = utils.Coalesce(overrides.Timeout.Write, c.Server.Timeout.Write).(time.Duration)
	c.Server.Timeout.Idle = utils.Coalesce(overrides.Timeout.Idle, c.Server.Timeout.Idle).(time.Duration)
	c.Server.Timeout.Handler = utils.Coalesce(overrides.Timeout.Handler, c.Server.Timeout.Handler).(time.Duration)
}

// --- UPSTREAM.
func (c *Configuration) copyOverWithUpstream(overrides Server) {
	c.Server.Upstream.Host = utils.Coalesce(overrides.Upstream.Host, c.Server.Upstream.Host).(string)
	c.Server.Upstream.Port = utils.Coalesce(overrides.Upstream.Port, c.Server.Upstream.Port).(string)
	c.Server.Upstream.Scheme = utils.Coalesce(overrides.Upstream.Scheme, c.Server.Upstream.Scheme).(string)
	c.Server.Upstream.BalancingAlgorithm = utils.Coalesce(overrides.Upstream.BalancingAlgorithm, c.Server.Upstream.BalancingAlgorithm).(string)
	c.Server.Upstream.Endpoints = utils.Coalesce(overrides.Upstream.Endpoints, c.Server.Upstream.Endpoints).([]string)
	c.Server.Upstream.HTTP2HTTPS = utils.Coalesce(overrides.Upstream.HTTP2HTTPS, c.Server.Upstream.HTTP2HTTPS).(bool)
	c.Server.Upstream.InsecureBridge = utils.Coalesce(overrides.Upstream.InsecureBridge, c.Server.Upstream.InsecureBridge).(bool)
	c.Server.Upstream.RedirectStatusCode = utils.Coalesce(overrides.Upstream.RedirectStatusCode, c.Server.Upstream.RedirectStatusCode).(int)
	c.Server.Upstream.HealthCheck.StatusCodes = utils.Coalesce(overrides.Upstream.HealthCheck.StatusCodes, c.Server.Upstream.HealthCheck.StatusCodes).([]string)
	c.Server.Upstream.HealthCheck.Timeout = utils.Coalesce(overrides.Upstream.HealthCheck.Timeout, c.Server.Upstream.HealthCheck.Timeout).(time.Duration)
	c.Server.Upstream.HealthCheck.Interval = utils.Coalesce(overrides.Upstream.HealthCheck.Interval, c.Server.Upstream.HealthCheck.Interval).(time.Duration)
	c.Server.Upstream.HealthCheck.Port = utils.Coalesce(overrides.Upstream.HealthCheck.Port, c.Server.Upstream.HealthCheck.Port).(string)
	c.Server.Upstream.HealthCheck.Scheme = utils.Coalesce(overrides.Upstream.HealthCheck.Scheme, c.Server.Upstream.HealthCheck.Scheme).(string)
	c.Server.Upstream.HealthCheck.AllowInsecure = utils.Coalesce(overrides.Upstream.HealthCheck.AllowInsecure, c.Server.Upstream.HealthCheck.AllowInsecure).(bool)
	c.Server.Upstream.CollapsedForwarding = utils.Coalesce(overrides.Upstream.CollapsedForwarding, c.Server.Upstream.CollapsedForwarding).(bool)

	c.Server.Upstream.Scheme = utils.IfEmpty(c.Server.Upstream.Scheme, SchemeWildcard)
}

// --- CACHE.
func (c *Configuration) copyOverWithCache(overrides Cache) {
	c.Cache.Hosts = utils.Coalesce(overrides.Hosts, c.Cache.Hosts).([]string)
	c.Cache.Password = utils.Coalesce(overrides.Password, c.Cache.Password).(string)
	c.Cache.DB = utils.Coalesce(overrides.DB, c.Cache.DB).(int)
	c.Cache.TTL = utils.Coalesce(overrides.TTL, c.Cache.TTL).(int)
	c.Cache.AllowedStatuses = utils.Coalesce(overrides.AllowedStatuses, c.Cache.AllowedStatuses).([]int)
	c.Cache.AllowedMethods = utils.Coalesce(overrides.AllowedMethods, c.Cache.AllowedMethods).([]string)

	c.Cache.AllowedMethods = append(c.Cache.AllowedMethods, "HEAD", "GET")
	c.Cache.AllowedMethods = slice.Unique(c.Cache.AllowedMethods)
}

// --- TRACING.
func (c *Configuration) copyOverWithTracing(overrides Tracing) {
	c.Tracing.JaegerEndpoint = utils.Coalesce(overrides.JaegerEndpoint, c.Tracing.JaegerEndpoint).(string)
	c.Tracing.Enabled = utils.Coalesce(overrides.Enabled, c.Tracing.Enabled).(bool)
	// TODO: when starting is not using the default value set in the tag. it might happen to other properties as well.
	c.Tracing.SamplingRatio = utils.Coalesce(overrides.SamplingRatio, c.Tracing.SamplingRatio).(float64)
}

// --- LOG.
func (c *Configuration) copyOverWithLog(overrides Log) {
	c.Log.SentryDsn = utils.Coalesce(overrides.SentryDsn, c.Log.SentryDsn).(string)
	c.Log.SyslogProtocol = utils.Coalesce(overrides.SyslogProtocol, c.Log.SyslogProtocol).(string)
	c.Log.SyslogEndpoint = utils.Coalesce(overrides.SyslogEndpoint, c.Log.SyslogEndpoint).(string)
}

// --- JWT.
func (c *Configuration) copyOverWithJwt(overrides Jwt) {
	c.Jwt.ExcludedPaths = utils.Coalesce(overrides.ExcludedPaths, c.Jwt.ExcludedPaths).([]string)
	c.Jwt.AllowedScopes = utils.Coalesce(overrides.AllowedScopes, c.Jwt.AllowedScopes).([]string)
	c.Jwt.JwksUrl = utils.Coalesce(overrides.JwksUrl, c.Jwt.JwksUrl).(string)
	c.Jwt.JwksRefreshInterval = utils.Coalesce(overrides.JwksRefreshInterval, c.Jwt.JwksRefreshInterval).(int)
	c.Jwt.Context = context.Background()
	c.Jwt.Logger = log.New()
}
