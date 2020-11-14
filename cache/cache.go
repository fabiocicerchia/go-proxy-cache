package cache_redis

import (
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/utils"
)

func StoreFullPage(url string, status int, headers map[string]interface{}, reqHeaders map[string]string, content string, expiration time.Duration) (bool, error) {
	headersConverted := make(map[string]string)
	for k, v := range headers {
		headersConverted[k] = v.(string)
	}

	response := &Response{
		StatusCode: status,
		Headers:    headersConverted,
		Content:    content,
	}

	valueToEncode, err := utils.MsgpackEncode(response)
	if err != nil {
		// TODO: LOG
		return false, err
	}
	encodedBase64Value := string(utils.Base64Encode(valueToEncode))

	meta := GetVary(headersConverted)
	StoreMetadata(url, meta, expiration)
	key := CacheKey(url, meta, reqHeaders)

	return Set(key, encodedBase64Value, expiration)
}

func RetrieveFullPage(url string, reqHeaders map[string]string) (statusCode int, headers map[string]string, content string, err error) {
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

	decodedValue := utils.Base64Decode([]byte(encodedBase64Value))
	err = utils.MsgpackDecode(decodedValue, response)
	if err != nil {
		return statusCode, headers, content, err
	}

	statusCode = response.StatusCode
	headers = response.Headers
	content = response.Content

	return statusCode, headers, content, nil
}

func CacheKey(url string, meta []string, reqHeaders map[string]string) string {
	key := []string{"DATA", url}

	vary := meta
	for _, k := range vary {
		key = append(key, reqHeaders[k])
	}

	cacheKey := strings.Join(key, "@@")

	return cacheKey
}

func FetchMetadata(url string) (meta []string, err error) {
	key := "META@@" + url

	value, err := Get(key)
	if err != nil {
		return meta, err
	}

	// TODO: USE different redis structure
	meta = strings.Split(value, ",")

	return meta, nil
}

func StoreMetadata(url string, meta []string, expiration time.Duration) (bool, error) {
	key := "META@@" + url

	serialized := strings.Join(meta, ",")
	err := rdb.Set(ctx, key, serialized, expiration).Err()
	if err != nil {
		return false, err
	}

	return true, nil
}

func GetVary(headers map[string]string) []string {
	vary := strings.Split(headers["Vary"], ",")
	for k, v := range vary {
		v = strings.Trim(v, " ")
		vary[k] = v
	}
	return vary
}
