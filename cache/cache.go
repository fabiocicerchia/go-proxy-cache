package cache

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

// Response - Holds details about the response
type Response struct {
	Method     string
	StatusCode int
	Headers    http.Header
	Content    string
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
func StoreFullPage(url string, method string, status int, headers http.Header, reqHeaders http.Header, content string, expiration time.Duration) (bool, error) {
	if !IsStatusAllowed(status) || !IsMethodAllowed(method) {
		return false, nil
	}

	if expiration < 1 {
		return false, nil
	}

	response := &Response{
		Method:     method,
		StatusCode: status,
		Headers:    headers,
		Content:    content,
	}

	encoded, err := engine.Encode(response)
	if err != nil {
		return false, err
	}

	meta, err := GetVary(headers)
	if err != nil {
		return false, err
	}

	_, err = StoreMetadata(method, url, meta, expiration)
	if err != nil {
		return false, err
	}

	key := StorageKey(method, url, meta, reqHeaders)

	return engine.Set(key, encoded, expiration)
}

// RetrieveFullPage - Retrieves the whole page response from cache.
func RetrieveFullPage(method string, url string, reqHeaders http.Header) (int, http.Header, string, error) {
	response := &Response{}

	meta, err := FetchMetadata(method, url)
	if err != nil {
		return 0, http.Header{}, "", err
	}

	key := StorageKey(method, url, meta, reqHeaders)

	encoded, err := engine.Get(key)
	if err != nil {
		return 0, http.Header{}, "", err
	}

	err = engine.Decode(encoded, response)
	if err != nil {
		return 0, http.Header{}, "", err
	}

	return response.StatusCode, response.Headers, response.Content, nil
}

// PurgeFullPage - Deletes the whole page response from cache.
func PurgeFullPage(method string, url string) (bool, error) {
	err := DeleteMetadata(method, url)
	if err != nil {
		return false, err
	}

	var meta []string
	key := StorageKey(method, url, meta, http.Header{})

	keyPattern := strings.Replace(key, "@@PURGE@@", "@@*@@", 1) + "*"
	affected, err := engine.DelWildcard(keyPattern)
	if err != nil {
		return false, err
	}

	done := affected > 0

	return done, nil
}

// StorageKey - Returns the cache key for the requested URL.
func StorageKey(method string, url string, meta []string, reqHeaders http.Header) string {
	key := []string{"DATA", method, url}

	vary := meta
	for _, k := range vary {
		// TODO: USE get/values?
		if val, ok := reqHeaders[k]; ok {
			key = append(key, val[0])
		}
	}

	storageKey := strings.Join(key, "@@")

	return storageKey
}

// FetchMetadata - Returns the cache metadata for the requested URL.
func FetchMetadata(method string, url string) ([]string, error) {
	key := "META@@" + method + "@@" + url

	return engine.List(key)
}

// DeleteMetadata - Removes the cache metadata for the requested URL.
func DeleteMetadata(method string, url string) error {
	key := "META@@" + method + "@@" + url

	return engine.Del(key)
}

// StoreMetadata - Saves the cache metadata for the requested URL.
func StoreMetadata(method string, url string, meta []string, expiration time.Duration) (bool, error) {
	key := "META@@" + method + "@@" + url

	err := engine.Del(key) // ignore error
	err = engine.Push(key, meta)
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
