package cache

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2022 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/logger"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
)

var errMissingRedisConnection = errors.New("missing redis connection")
var errNotAllowed = errors.New("not allowed")
var errCannotFetchMetadata = errors.New("cannot fetch metadata")
var errCannotGetKey = errors.New("cannot get key")
var errCannotDecode = errors.New("cannot decode")
var errVaryWildcard = errors.New("vary: *")

// ErrEmptyValue - Error used when no data is available in Redis.
var ErrEmptyValue = errors.New("empty value")

// DefaultMinSoftExpirationTTL - Additional time to avoid cache stampede (min lower bound).
const DefaultMinSoftExpirationTTL time.Duration = 5 * time.Second // TODO: Make it customizable?

// DefaultMaxSoftExpirationTTL - Additional time to avoid cache stampede (max upper bound).
const DefaultMaxSoftExpirationTTL time.Duration = 10 * time.Second // TODO: Make it customizable?

// FreshSuffix - Used for saving a suffix for handling cache stampede.
const FreshSuffix = "/fresh"

// Object - Contains cache settings and current cached/cacheable object.
type Object struct {
	ReqID            string
	AllowedStatuses  []int
	AllowedMethods   []string
	CurrentURIObject URIObj
	DomainID         string
}

// URIObj - Holds details about the response.
type URIObj struct {
	URL             url.URL
	Method          string
	StatusCode      int
	RequestHeaders  http.Header
	ResponseHeaders http.Header
	Content         [][]byte
	Stale           bool
}

// IsStatusAllowed - Checks if a status code is allowed to be cached.
func (c Object) IsStatusAllowed() bool {
	return slice.ContainsInt(c.AllowedStatuses, c.CurrentURIObject.StatusCode)
}

// IsMethodAllowed - Checks if a HTTP method is allowed to be cached.
func (c Object) IsMethodAllowed() bool {
	return slice.ContainsString(c.AllowedMethods, c.CurrentURIObject.Method)
}

func getRandomSoftExpirationTTL() time.Duration {
	return time.Duration(rand.Intn(int(DefaultMaxSoftExpirationTTL)-int(DefaultMinSoftExpirationTTL)) + int(DefaultMinSoftExpirationTTL))
}

// GetHeadersChecksum - Returns a SHA256 based on the HTTP Request Headers.
func (u URIObj) GetHeadersChecksum(meta []string) string {
	var key []string

	if len(meta) == 0 {
		return ""
	}

	for _, k := range meta {
		if val, ok := u.RequestHeaders[k]; ok {
			key = append(key, strings.Join(val, utils.StringSeparatorTwo))
		}
	}

	data, err := json.Marshal(key)
	if err != nil {
		return ""
	}

	h := sha256.New()
	h.Write([]byte(data))

	return fmt.Sprintf("%x", h.Sum(nil))
}

// IsValid - Verifies the validity of a cacheable object.
func (c Object) IsValid() (bool, error) {
	if !c.IsStatusAllowed() || slice.LenSliceBytes(c.CurrentURIObject.Content) == 0 {
		return false, errors.Wrapf(errNotAllowed,
			"status %d - content length %d",
			c.CurrentURIObject.StatusCode,
			slice.LenSliceBytes(c.CurrentURIObject.Content))
	}

	return true, nil
}

func (c Object) handleMetadata(ctx context.Context, domainID string, targetURL url.URL, expiration time.Duration) ([]string, error) {
	meta, err := GetVary(c.CurrentURIObject.ResponseHeaders)
	if err != nil {
		return []string{}, err
	}

	_, err = StoreMetadata(ctx, domainID, c.CurrentURIObject.Method, targetURL, meta, expiration)
	if err != nil {
		return []string{}, err
	}

	return meta, nil
}

// StoreFullPage - Stores the whole page response in cache.
func (c Object) StoreFullPage(ctx context.Context, expiration time.Duration) (bool, error) {
	if !c.IsStatusAllowed() || !c.IsMethodAllowed() || expiration < 1 {
		logger.GetGlobal().WithFields(log.Fields{
			"ReqID": c.ReqID,
		}).Debugf(
			"Not allowed to be stored. Status: %v - Method: %v - Expiration: %v",
			c.IsStatusAllowed(),
			c.IsMethodAllowed(),
			expiration,
		)

		return false, nil
	}

	meta, err := c.handleMetadata(ctx, c.DomainID, c.CurrentURIObject.URL, expiration)
	if err != nil {
		return false, err
	}

	conn := engine.GetConn(c.DomainID)
	if conn == nil {
		return false, errors.Wrapf(errMissingRedisConnection, "Error for %s", c.DomainID)
	}

	encoded, err := conn.Encode(c.CurrentURIObject)
	if err != nil {
		return false, err
	}

	key := StorageKey(c.CurrentURIObject, meta)

	// HARD EVICTION
	expirationHard := expiration
	done, err := conn.Set(ctx, key+FreshSuffix, encoded, expirationHard)
	if err != nil {
		return done, err
	}

	// SOFT EVICTION
	expirationSoft := expiration + getRandomSoftExpirationTTL()
	if expiration == 0 {
		expirationSoft = 0
	}
	return conn.Set(ctx, key, encoded, expirationSoft)
}

// RetrieveFullPage - Retrieves the whole page response from cache.
func (c *Object) RetrieveFullPage() error {
	obj := &URIObj{}

	meta, err := FetchMetadata(c.DomainID, c.CurrentURIObject.Method, c.CurrentURIObject.URL)
	if err != nil {
		return errors.Wrap(errCannotFetchMetadata, err.Error())
	}

	conn := engine.GetConn(c.DomainID)
	if conn == nil {
		return errors.Wrapf(errMissingRedisConnection, "Error for %s", c.DomainID)
	}

	key := StorageKey(c.CurrentURIObject, meta)
	logger.GetGlobal().WithFields(log.Fields{
		"ReqID": c.ReqID,
	}).Debugf("StorageKey: %s", key)

	var stale bool = false
	encoded, err := conn.Get(key + FreshSuffix)
	if err != nil || encoded == "" {
		stale = true
		encoded, err = conn.Get(key)
		if err != nil {
			return errors.Wrap(errCannotGetKey, err.Error())
		}
	}

	if encoded == "" {
		return ErrEmptyValue
	}

	err = conn.Decode(encoded, obj)
	if err != nil {
		return errors.Wrap(errCannotDecode, err.Error())
	}

	c.CurrentURIObject = *obj
	c.CurrentURIObject.Stale = stale

	return nil
}

// PurgeFullPage - Deletes the whole page response from cache.
func (c Object) PurgeFullPage(ctx context.Context) (bool, error) {
	err := PurgeMetadata(ctx, c.DomainID, c.CurrentURIObject.URL)
	if err != nil {
		return false, err
	}

	conn := engine.GetConn(c.DomainID)
	if conn == nil {
		return false, errors.Wrapf(errMissingRedisConnection, "Error for %s", c.DomainID)
	}

	key := StorageKey(c.CurrentURIObject, []string{})

	match := utils.StringSeparatorOne + "PURGE" + utils.StringSeparatorOne
	replace := utils.StringSeparatorOne + "*" + utils.StringSeparatorOne
	keyPattern := strings.Replace(key, match, replace, 1) + "*"

	affected, err := conn.DelWildcard(ctx, keyPattern)
	if err != nil {
		return false, err
	}

	done := affected > 0

	return done, nil
}

// StorageKey - Returns the cache key for the requested URL.
func StorageKey(currentURIObject URIObj, meta []string) string {
	key := []string{"DATA", currentURIObject.Method, currentURIObject.URL.String(), currentURIObject.GetHeadersChecksum(meta)}
	storageKey := strings.Join(key, utils.StringSeparatorOne)

	return storageKey
}

// FetchMetadata - Returns the cache metadata for the requested URL.
func FetchMetadata(domainID string, method string, url url.URL) ([]string, error) {
	key := "META" + utils.StringSeparatorOne + method + utils.StringSeparatorOne + url.String()

	conn := engine.GetConn(domainID)
	if conn == nil {
		return []string{}, errors.Wrapf(errMissingRedisConnection, "Error for %s", domainID)
	}

	return conn.List(key)
}

// PurgeMetadata - Purges the cache metadata for the requested URL.
func PurgeMetadata(ctx context.Context, domainID string, url url.URL) error {
	keyPattern := "META" + utils.StringSeparatorOne + "*" + utils.StringSeparatorOne + url.String()

	conn := engine.GetConn(domainID)
	if conn == nil {
		return errors.Wrapf(errMissingRedisConnection, "Error for %s", domainID)
	}

	_, err := conn.DelWildcard(ctx, keyPattern)

	return err
}

// StoreMetadata - Saves the cache metadata for the requested URL.
func StoreMetadata(ctx context.Context, domainID string, method string, url url.URL, meta []string, expiration time.Duration) (bool, error) {
	key := "META" + utils.StringSeparatorOne + method + utils.StringSeparatorOne + url.String()

	conn := engine.GetConn(domainID)
	if conn == nil {
		return false, errors.Wrapf(errMissingRedisConnection, "Error for %s", domainID)
	}

	_ = conn.Del(ctx, key)

	err := conn.Push(ctx, key, meta)
	if err != nil {
		return false, err
	}

	err = conn.Expire(key, expiration+getRandomSoftExpirationTTL())
	if err != nil {
		_ = conn.Del(ctx, key)
		return false, err
	}

	return true, nil
}

// GetVary - Returns the content from the Vary HTTP header.
func GetVary(headers http.Header) ([]string, error) {
	vary := headers.Get("Vary")

	if vary == "*" {
		return []string{}, errVaryWildcard
	}

	varyList := strings.Split(vary, ",")
	for k, v := range varyList {
		varyList[k] = strings.Trim(v, " ")
	}

	return varyList, nil
}
