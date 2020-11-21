package cache

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	return utils.Contains(config.Config.Cache.AllowedStatuses, strconv.Itoa(statusCode))
}

// IsMethodAllowed - Checks if a HTTP method is allowed to be cached.
func IsMethodAllowed(method string) bool {
	return utils.Contains(config.Config.Cache.AllowedMethods, method)
}

// StoreFullPage - Stores the whole page response in cache.
func StoreFullPage(
	obj URIObj,
	expiration time.Duration,
) (bool, error) {
	if !IsStatusAllowed(obj.StatusCode) || !IsMethodAllowed(obj.Method) || expiration < 1 {
		return false, nil
	}

	targetUrl := obj.URL
	targetUrl.Host = obj.Host

	meta, err := GetVary(obj.ResponseHeaders)
	if err != nil {
		return false, err
	}

	_, err = StoreMetadata(obj.Method, targetUrl, meta, expiration)
	if err != nil {
		return false, err
	}

	encoded, err := engine.Encode(obj)
	if err != nil {
		return false, err
	}

	key := StorageKey(obj.Method, targetUrl, meta, obj.RequestHeaders)

	return engine.Set(key, encoded, expiration)
}

// RetrieveFullPage - Retrieves the whole page response from cache.
func RetrieveFullPage(method string, url url.URL, reqHeaders http.Header) (int, http.Header, [][]byte, error) {
	obj := &URIObj{}

	meta, err := FetchMetadata(method, url)
	if err != nil {
		return 0, http.Header{}, [][]byte{}, fmt.Errorf("Cannot fetch metadata: %s", err)
	}

	key := StorageKey(method, url, meta, reqHeaders)
	log.Infof("StorageKey: %s", key)

	encoded, err := engine.Get(key)
	if err != nil {
		return 0, http.Header{}, [][]byte{}, fmt.Errorf("Cannot get key: %s", err)
	}

	err = engine.Decode(encoded, obj)
	if err != nil {
		return 0, http.Header{}, [][]byte{}, fmt.Errorf("Cannot decode: %s", err)
	}

	return obj.StatusCode, obj.ResponseHeaders, obj.Content, nil
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
	affected, err := engine.DelWildcard(keyPattern)
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

	return engine.List(key)
}

// DeleteMetadata - Removes the cache metadata for the requested URL.
func DeleteMetadata(method string, url url.URL) error {
	key := "META" + utils.StringSeparatorOne + method + utils.StringSeparatorOne + url.String()

	return engine.Del(key)
}

// StoreMetadata - Saves the cache metadata for the requested URL.
func StoreMetadata(method string, url url.URL, meta []string, expiration time.Duration) (bool, error) {
	key := "META" + utils.StringSeparatorOne + method + utils.StringSeparatorOne + url.String()

	_ = engine.Del(key) //nolint:golint,errcheck
	err := engine.Push(key, meta)
	if err != nil {
		return false, err
	}

	err = engine.Expire(key, expiration)
	if err != nil {
		// TODO: use transaction
		_ = engine.Del(key)

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
