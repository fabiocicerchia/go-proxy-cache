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
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	log "github.com/sirupsen/logrus"
)

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
func IsStatusAllowed(statusCode int) bool {
	return utils.ContainsInt(config.Config.Cache.AllowedStatuses, statusCode)
}

// IsMethodAllowed - Checks if a HTTP method is allowed to be cached.
func IsMethodAllowed(method string) bool {
	return utils.ContainsString(config.Config.Cache.AllowedMethods, method)
}

// StoreFullPage - Stores the whole page response in cache.
func StoreFullPage(
	obj URIObj,
	expiration time.Duration,
) (bool, error) {
	if !IsStatusAllowed(obj.StatusCode) || !IsMethodAllowed(obj.Method) || expiration < 1 {
		return false, nil
	}

	targetURL := obj.URL
	targetURL.Host = obj.Host

	meta, err := GetVary(obj.ResponseHeaders)
	if err != nil {
		return false, err
	}

	_, err = StoreMetadata(obj.Method, targetURL, meta, expiration)
	if err != nil {
		return false, err
	}

	encoded, err := engine.GetConn("global").Encode(obj)
	if err != nil {
		return false, err
	}

	key := StorageKey(obj.Method, targetURL, meta, obj.RequestHeaders)

	return engine.GetConn("global").Set(key, encoded, expiration)
}

// RetrieveFullPage - Retrieves the whole page response from cache.
func RetrieveFullPage(method string, url url.URL, reqHeaders http.Header) (URIObj, error) {
	obj := &URIObj{}

	meta, err := FetchMetadata(method, url)
	if err != nil {
		return *obj, fmt.Errorf("Cannot fetch metadata: %s", err)
	}

	key := StorageKey(method, url, meta, reqHeaders)
	log.Debugf("StorageKey: %s", key)

	encoded, err := engine.GetConn("global").Get(key)
	if err != nil {
		return *obj, fmt.Errorf("Cannot get key: %s", err)
	}

	err = engine.GetConn("global").Decode(encoded, obj)
	if err != nil {
		return *obj, fmt.Errorf("Cannot decode: %s", err)
	}

	return *obj, nil
}

// PurgeFullPage - Deletes the whole page response from cache.
func PurgeFullPage(method string, url url.URL) (bool, error) {
	err := DeleteMetadata(method, url)
	if err != nil {
		return false, err
	}

	var meta []string
	key := StorageKey(method, url, meta, http.Header{})

	match := utils.StringSeparatorOne + "PURGE" + utils.StringSeparatorOne
	replace := utils.StringSeparatorOne + "*" + utils.StringSeparatorOne
	keyPattern := strings.Replace(key, match, replace, 1) + "*"
	affected, err := engine.GetConn("global").DelWildcard(keyPattern)
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

	return engine.GetConn("global").List(key)
}

// DeleteMetadata - Removes the cache metadata for the requested URL.
func DeleteMetadata(method string, url url.URL) error {
	key := "META" + utils.StringSeparatorOne + method + utils.StringSeparatorOne + url.String()

	return engine.GetConn("global").Del(key)
}

// StoreMetadata - Saves the cache metadata for the requested URL.
func StoreMetadata(method string, url url.URL, meta []string, expiration time.Duration) (bool, error) {
	key := "META" + utils.StringSeparatorOne + method + utils.StringSeparatorOne + url.String()

	_ = engine.GetConn("global").Del(key) //nolint:golint,errcheck
	err := engine.GetConn("global").Push(key, meta)
	if err != nil {
		return false, err
	}

	err = engine.GetConn("global").Expire(key, expiration)
	if err != nil {
		// TODO: use transaction
		_ = engine.GetConn("global").Del(key)

		return false, err
	}

	return true, nil
}

// GetVary - Returns the content from the Vary HTTP header.
func GetVary(headers http.Header) ([]string, error) {
	var varyList []string
	vary := headers.Get("Vary")

	if vary == "*" {
		return varyList, errors.New("Vary: *")
	}

	varyList = strings.Split(vary, ",")

	for k, v := range varyList {
		v = strings.Trim(v, " ")
		varyList[k] = v
	}

	return varyList, nil
}
