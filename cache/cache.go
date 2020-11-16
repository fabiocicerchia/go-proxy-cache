package redis

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

func StoreFullPage(url string, method string, status int, headers map[string]interface{}, reqHeaders map[string]interface{}, content string, expiration time.Duration) (bool, error) {
	if utils.Contains(config.Config.Cache.AllowedStatuses, strconv.Itoa(status)) {
		return false, nil
	}

	if utils.Contains(config.Config.Cache.AllowedMethods, method) {
		return false, nil
	}

	response := &Response{
		Method:     method,
		StatusCode: status,
		Headers:    headers,
		Content:    content,
	}

	valueToEncode, err := utils.MsgpackEncode(response)
	if err != nil {
		// TODO: log
		return false, err
	}
	encodedBase64Value := string(utils.Base64Encode(valueToEncode))

	meta, err := GetVary(headers)
	if err != nil {
		return false, err
	}
	StoreMetadata(method, url, meta, expiration)
	key := CacheKey(method, url, meta, reqHeaders)

	return Set(key, encodedBase64Value, expiration)
}

func RetrieveFullPage(method, url string, reqHeaders map[string]interface{}) (statusCode int, headers map[string]interface{}, content string, err error) {
	response := &Response{}

	meta, err := FetchMetadata(method, url)
	if err != nil || len(meta) == 0 {
		return statusCode, headers, content, err
	}

	key := CacheKey(method, url, meta, reqHeaders)

	encodedBase64Value, err := Get(key)
	if err != nil {
		return statusCode, headers, content, err
	}

	decodedValue, err := utils.Base64Decode([]byte(encodedBase64Value))
	if err != nil {
		return statusCode, headers, content, err
	}

	err = utils.MsgpackDecode(decodedValue, response)
	if err != nil {
		return statusCode, headers, content, err
	}

	statusCode = response.StatusCode
	headers = response.Headers
	content = response.Content

	return statusCode, headers, content, nil
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

	return LRange(key)
}

func StoreMetadata(method, url string, meta []string, expiration time.Duration) (bool, error) {
	key := "META@@" + method + "@@" + url

	err := LPush(key, meta)
	if err != nil {
		return false, err
	}

	err = Expire(key, expiration)
	if err != nil {
		// TODO: use transaction
		_ = rdb.Del(ctx, key).Err()

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
