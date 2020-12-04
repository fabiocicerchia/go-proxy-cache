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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/fabiocicerchia/go-proxy-cache/utils/slice"
	log "github.com/sirupsen/logrus"
)

// CacheObj - Contains cache settings and current cached/cacheable object.
type CacheObj struct {
	AllowedStatuses []int
	AllowedMethods  []string
	CurrentObj      URIObj
}

// URIObj - Holds details about the response
type URIObj struct {
	URL             url.URL
	Host            string
	Method          string
	StatusCode      int
	RequestHeaders  http.Header
	ResponseHeaders http.Header
	Content         [][]byte
}

// IsStatusAllowed - Checks if a status code is allowed to be cached.
func (c CacheObj) IsStatusAllowed() bool {
	return slice.ContainsInt(c.AllowedStatuses, c.CurrentObj.StatusCode)
}

// IsMethodAllowed - Checks if a HTTP method is allowed to be cached.
func (c CacheObj) IsMethodAllowed() bool {
	return slice.ContainsString(c.AllowedMethods, c.CurrentObj.Method)
}

// IsValid - Verifies the validity of a cacheable object.
func (c CacheObj) IsValid() (bool, error) {
	if !c.IsStatusAllowed() || slice.LenSliceBytes(c.CurrentObj.Content) == 0 {
		return false, fmt.Errorf(
			"not allowed. status %d - content length %d",
			c.CurrentObj.StatusCode,
			slice.LenSliceBytes(c.CurrentObj.Content),
		)
	}

	return true, nil
}

func (c CacheObj) handleMetadata(targetURL url.URL, expiration time.Duration) ([]string, error) {
	meta, err := GetVary(c.CurrentObj.ResponseHeaders)
	if err != nil {
		return []string{}, err
	}

	_, err = StoreMetadata(c.CurrentObj.Method, targetURL, meta, expiration)
	if err != nil {
		return []string{}, err
	}

	return meta, nil
}

// StoreFullPage - Stores the whole page response in cache.
func (c CacheObj) StoreFullPage(expiration time.Duration) (bool, error) {
	if !c.IsStatusAllowed() || !c.IsMethodAllowed() || expiration < 1 {
		return false, nil
	}

	targetURL := c.CurrentObj.URL
	targetURL.Host = c.CurrentObj.Host

	meta, err := c.handleMetadata(targetURL, expiration)
	if err != nil {
		return false, err
	}

	encoded, err := engine.GetConn(targetURL.Host).Encode(c.CurrentObj)
	if err != nil {
		return false, err
	}

	key := StorageKey(c.CurrentObj.Method, targetURL, meta, c.CurrentObj.RequestHeaders)

	return engine.GetConn(targetURL.Host).Set(key, encoded, expiration)
}

// RetrieveFullPage - Retrieves the whole page response from cache.
func (c *CacheObj) RetrieveFullPage(method string, url url.URL, reqHeaders http.Header) error {
	obj := &URIObj{}

	meta, err := FetchMetadata(method, url)
	if err != nil {
		return fmt.Errorf("cannot fetch metadata: %s", err)
	}

	key := StorageKey(method, url, meta, reqHeaders)
	log.Debugf("StorageKey: %s", key)

	encoded, err := engine.GetConn(url.Host).Get(key)
	if err != nil {
		return fmt.Errorf("cannot get key: %s", err)
	}

	err = engine.GetConn(url.Host).Decode(encoded, obj)
	if err != nil {
		return fmt.Errorf("cannot decode: %s", err)
	}

	c.CurrentObj = *obj

	return nil
}

// PurgeFullPage - Deletes the whole page response from cache.
func (c CacheObj) PurgeFullPage(method string, url url.URL) (bool, error) {
	err := PurgeMetadata(url)
	if err != nil {
		return false, err
	}

	var meta []string
	key := StorageKey(method, url, meta, http.Header{})

	match := utils.StringSeparatorOne + "PURGE" + utils.StringSeparatorOne
	replace := utils.StringSeparatorOne + "*" + utils.StringSeparatorOne
	keyPattern := strings.Replace(key, match, replace, 1) + "*"
	affected, err := engine.GetConn(url.Host).DelWildcard(keyPattern)
	if err != nil {
		return false, err
	}

	done := affected > 0

	return done, nil
}

// StorageKey - Returns the cache key for the requested URL.
func StorageKey(method string, url url.URL, meta []string, reqHeaders http.Header) string {
	key := []string{"DATA", method, url.String()}

	vary := meta
	for _, k := range vary {
		if val, ok := reqHeaders[k]; ok {
			key = append(key, strings.Join(val, utils.StringSeparatorTwo))
		}
	}

	storageKey := strings.Join(key, utils.StringSeparatorOne)

	return storageKey
}

// FetchMetadata - Returns the cache metadata for the requested URL.
func FetchMetadata(method string, url url.URL) ([]string, error) {
	key := "META" + utils.StringSeparatorOne + method + utils.StringSeparatorOne + url.String()

	return engine.GetConn(url.Host).List(key)
}

// PurgeMetadata - Purges the cache metadata for the requested URL.
func PurgeMetadata(url url.URL) error {
	keyPattern := "META" + utils.StringSeparatorOne + "*" + utils.StringSeparatorOne + url.String()

	_, err := engine.GetConn(url.Host).DelWildcard(keyPattern)
	return err
}

// DeleteMetadata - Removes the cache metadata for the requested URL.
func DeleteMetadata(method string, url url.URL) error {
	key := "META" + utils.StringSeparatorOne + method + utils.StringSeparatorOne + url.String()

	return engine.GetConn(url.Host).Del(key)
}

// StoreMetadata - Saves the cache metadata for the requested URL.
func StoreMetadata(method string, url url.URL, meta []string, expiration time.Duration) (bool, error) {
	key := "META" + utils.StringSeparatorOne + method + utils.StringSeparatorOne + url.String()

	_ = engine.GetConn(url.Host).Del(key) //nolint:golint,errcheck
	err := engine.GetConn(url.Host).Push(key, meta)
	if err != nil {
		return false, err
	}

	err = engine.GetConn(url.Host).Expire(key, expiration)
	if err != nil {
		// TODO: use transaction
		_ = engine.GetConn(url.Host).Del(key)

		return false, err
	}

	return true, nil
}

// GetVary - Returns the content from the Vary HTTP header.
func GetVary(headers http.Header) ([]string, error) {
	var varyList []string
	vary := headers.Get("Vary")

	if vary == "*" {
		return varyList, errors.New("vary: *")
	}

	varyList = strings.Split(vary, ",")

	for k, v := range varyList {
		v = strings.Trim(v, " ")
		varyList[k] = v
	}

	return varyList, nil
}
