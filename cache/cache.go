package cache

//                                                                         __
// .-----.-----.______.-----.----.-----.--.--.--.--.______.----.---.-.----|  |--.-----.
// |  _  |  _  |______|  _  |   _|  _  |_   _|  |  |______|  __|  _  |  __|     |  -__|
// |___  |_____|      |   __|__| |_____|__.__|___  |      |____|___._|____|__|__|_____|
// |_____|            |__|                   |_____|
//
// Copyright (c) 2020 Fabio Cicerchia. https://fabiocicerchia.it. MIT License
// Repo: https://github.com/fabiocicerchia/go-proxy-cache

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
	log "github.com/sirupsen/logrus"
)

var errMissingRedisConnection = errors.New("missing redis connection")
var errNotAllowed = errors.New("not allowed")
var errCannotFetchMetadata = errors.New("cannot fetch metadata")
var errCannotGetKey = errors.New("cannot get key")
var errCannotDecode = errors.New("cannot decode")
var errVaryWildcard = errors.New("vary: *")

// Object - Contains cache settings and current cached/cacheable object.
type Object struct {
	AllowedStatuses  []int
	AllowedMethods   []string
	CurrentURIObject URIObj
	DomainID         string
}

// URIObj - Holds details about the response.
type URIObj struct {
	URL             url.URL
	Host            string
	Scheme          string
	Method          string
	StatusCode      int
	RequestHeaders  http.Header
	ResponseHeaders http.Header
	Content         [][]byte
}

// IsStatusAllowed - Checks if a status code is allowed to be cached.
func (c Object) IsStatusAllowed() bool {
	return slice.ContainsInt(c.AllowedStatuses, c.CurrentURIObject.StatusCode)
}

// IsMethodAllowed - Checks if a HTTP method is allowed to be cached.
func (c Object) IsMethodAllowed() bool {
	return slice.ContainsString(c.AllowedMethods, c.CurrentURIObject.Method)
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

func (c Object) handleMetadata(domainID string, targetURL url.URL, expiration time.Duration) ([]string, error) {
	meta, err := GetVary(c.CurrentURIObject.ResponseHeaders)
	if err != nil {
		return []string{}, err
	}

	_, err = StoreMetadata(domainID, c.CurrentURIObject.Method, targetURL, meta, expiration)
	if err != nil {
		return []string{}, err
	}

	return meta, nil
}

// StoreFullPage - Stores the whole page response in cache.
func (c Object) StoreFullPage(expiration time.Duration) (bool, error) {
	if !c.IsStatusAllowed() || !c.IsMethodAllowed() || expiration < 1 {
		log.Debugf(
			"Not allowed to be stored. Status: %v - Method: %v - Expiration: %v",
			c.IsStatusAllowed(),
			c.IsMethodAllowed(),
			expiration,
		)

		return false, nil
	}

	targetURL := c.CurrentURIObject.URL
	targetURL.Scheme = c.CurrentURIObject.Scheme
	targetURL.Host = c.CurrentURIObject.Host

	meta, err := c.handleMetadata(c.DomainID, targetURL, expiration)
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

	key := StorageKey(c.CurrentURIObject.Method, targetURL, meta, c.CurrentURIObject.RequestHeaders)

	return conn.Set(key, encoded, expiration)
}

// RetrieveFullPage - Retrieves the whole page response from cache.
func (c *Object) RetrieveFullPage(method string, url url.URL, reqHeaders http.Header) error {
	obj := &URIObj{}

	meta, err := FetchMetadata(c.DomainID, method, url)
	if err != nil {
		return errors.Wrap(errCannotFetchMetadata, err.Error())
	}

	conn := engine.GetConn(c.DomainID)
	if conn == nil {
		return errors.Wrapf(errMissingRedisConnection, "Error for %s", c.DomainID)
	}

	key := StorageKey(method, url, meta, reqHeaders)
	log.Debugf("StorageKey: %s", key)

	encoded, err := conn.Get(key)
	if err != nil {
		return errors.Wrap(errCannotGetKey, err.Error())
	}

	err = conn.Decode(encoded, obj)
	if err != nil {
		return errors.Wrap(errCannotDecode, err.Error())
	}

	c.CurrentURIObject = *obj

	return nil
}

// PurgeFullPage - Deletes the whole page response from cache.
func (c Object) PurgeFullPage(method string, url url.URL) (bool, error) {
	err := PurgeMetadata(c.DomainID, url)
	if err != nil {
		return false, err
	}

	conn := engine.GetConn(c.DomainID)
	if conn == nil {
		return false, errors.Wrapf(errMissingRedisConnection, "Error for %s", c.DomainID)
	}

	var meta []string
	key := StorageKey(method, url, meta, http.Header{})

	match := utils.StringSeparatorOne + "PURGE" + utils.StringSeparatorOne
	replace := utils.StringSeparatorOne + "*" + utils.StringSeparatorOne
	keyPattern := strings.Replace(key, match, replace, 1) + "*"

	affected, err := conn.DelWildcard(keyPattern)
	if err != nil {
		return false, err
	}

	done := affected > 0

	return done, nil
}

// StorageKey - Returns the cache key for the requested URL.
func StorageKey(method string, url url.URL, meta []string, reqHeaders http.Header) string {
	key := []string{"DATA", method, url.String()}

	for _, k := range meta {
		if val, ok := reqHeaders[k]; ok {
			key = append(key, strings.Join(val, utils.StringSeparatorTwo))
		}
	}

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
func PurgeMetadata(domainID string, url url.URL) error {
	keyPattern := "META" + utils.StringSeparatorOne + "*" + utils.StringSeparatorOne + url.String()

	conn := engine.GetConn(domainID)
	if conn == nil {
		return errors.Wrapf(errMissingRedisConnection, "Error for %s", domainID)
	}

	_, err := conn.DelWildcard(keyPattern)

	return err
}

// StoreMetadata - Saves the cache metadata for the requested URL.
func StoreMetadata(domainID string, method string, url url.URL, meta []string, expiration time.Duration) (bool, error) {
	key := "META" + utils.StringSeparatorOne + method + utils.StringSeparatorOne + url.String()

	conn := engine.GetConn(domainID)
	if conn == nil {
		return false, errors.Wrapf(errMissingRedisConnection, "Error for %s", domainID)
	}

	_ = conn.Del(key)

	err := conn.Push(key, meta)
	if err != nil {
		return false, err
	}

	err = conn.Expire(key, expiration)
	if err != nil {
		// TODO: use transaction
		_ = conn.Del(key)

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
