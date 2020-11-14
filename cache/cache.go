package redis

import (
	"errors"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

func StoreFullPage(url string, status int, headers map[string]interface{}, reqHeaders map[string]interface{}, content string, expiration time.Duration) (bool, error) {
	response := &Response{
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
	StoreMetadata(url, meta, expiration)
	key := CacheKey(url, meta, reqHeaders)

	return Set(key, encodedBase64Value, expiration)
}

func RetrieveFullPage(url string, reqHeaders map[string]interface{}) (statusCode int, headers map[string]interface{}, content string, err error) {
	response := &Response{}

	meta, err := FetchMetadata(url)
	if err != nil || len(meta) == 0 {
		return statusCode, headers, content, err
	}

	key := CacheKey(url, meta, reqHeaders)

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

func CacheKey(url string, meta []string, reqHeaders map[string]interface{}) string {
	key := []string{"DATA", url}

	vary := meta
	for _, k := range vary {
		key = append(key, reqHeaders[k].(string))
	}

	cacheKey := strings.Join(key, "@@")

	return cacheKey
}

func FetchMetadata(url string) (meta []string, err error) {
	key := "META@@" + url

	meta, err = rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return meta, err
	}

	return meta, nil
}

func StoreMetadata(url string, meta []string, expiration time.Duration) (bool, error) {
	key := "META@@" + url

	err := rdb.LPush(ctx, key, meta).Err()
	if err != nil {
		return false, err
	}

	err = rdb.Expire(ctx, key, expiration).Err()
	if err != nil {
		// TODO: use transaction
		_ = rdb.Del(ctx, key).Err()

		return false, err
	}

	return true, nil
}

func GetVary(headers map[string]interface{}) (varyList []string, err error) {
	vary := headers["Vary"].(string)

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
