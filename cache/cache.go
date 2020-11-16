package cache

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/cache/engine"
	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

type Response struct {
	Method     string
	StatusCode int
	Headers    map[string]interface{}
	Content    string
}

func IsStatusAllowed(statusCode int) bool {
	return utils.Contains(config.Config.Cache.AllowedStatuses, strconv.Itoa(statusCode))
}

func IsMethodAllowed(method string) bool {
	return utils.Contains(config.Config.Cache.AllowedMethods, method)
}

func StoreFullPage(url string, method string, status int, headers map[string]interface{}, reqHeaders map[string]interface{}, content string, expiration time.Duration) (bool, error) {
	if !IsStatusAllowed(status) || !IsMethodAllowed(method) {
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
		// TODO: log
		return false, err
	}

	meta, err := GetVary(headers)
	if err != nil {
		return false, err
	}
	StoreMetadata(method, url, meta, expiration)

	key := CacheKey(method, url, meta, reqHeaders)

	return engine.Set(key, encoded, expiration)
}

func RetrieveFullPage(method, url string, reqHeaders map[string]interface{}) (statusCode int, headers map[string]interface{}, content string, err error) {
	response := &Response{}

	meta, err := FetchMetadata(method, url)
	if err != nil || len(meta) == 0 {
		return statusCode, headers, content, err
	}

	key := CacheKey(method, url, meta, reqHeaders)

	encoded, err := engine.Get(key)
	if err != nil {
		return statusCode, headers, content, err
	}

	err = engine.Decode(encoded, response)
	if err != nil {
		return statusCode, headers, content, err
	}

	return response.StatusCode, response.Headers, response.Content, nil
}

func PurgeFullPage(method, url string) (bool, error) {
	err := DeleteMetadata(method, url)
	if err != nil {
		return false, err
	}

	var meta []string
	reqHeaders := make(map[string]interface{})
	key := CacheKey(method, url, meta, reqHeaders)

	err = engine.DelWildcard(key)
	if err != nil {
		return false, err
	}

	return true, nil
}

func CacheKey(method, url string, meta []string, reqHeaders map[string]interface{}) string {
	key := []string{"DATA", method, url}

	vary := meta
	for _, k := range vary {
		if val, ok := reqHeaders[k]; ok {
			key = append(key, val.(string))
		} else {
			key = append(key, "")
		}
	}

	cacheKey := strings.Join(key, "@@")

	return cacheKey
}

func FetchMetadata(method, url string) (meta []string, err error) {
	key := "META@@" + method + "@@" + url

	return engine.List(key)
}

func DeleteMetadata(method, url string) error {
	key := "META@@" + method + "@@" + url

	return engine.Del(key)
}

func StoreMetadata(method, url string, meta []string, expiration time.Duration) (bool, error) {
	key := "META@@" + method + "@@" + url

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

func GetVary(headers map[string]interface{}) (varyList []string, err error) {
	var vary string
	if value, ok := headers["Vary"]; ok {
		vary = value.(string)
	}

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
