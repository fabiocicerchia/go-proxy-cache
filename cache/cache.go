package cache_redis

import (
	"encoding/gob"
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
		Content:    string(utils.Base64Encode([]byte(content))),
	}

	valueToEncode := utils.EncodeGob(response)
	encodedBase64Value := string(utils.Base64Encode(valueToEncode))

	meta := GetVary(headersConverted)
	StoreMetadata(url, meta, expiration)
	key := CacheKey(url, meta, reqHeaders)

	return Set(key, encodedBase64Value, expiration)
}

func RetrieveFullPage(url string, reqHeaders map[string]string) (statusCode int, headers map[string]string, content string, err error) {
	var response Response

	meta, err := FetchMetadata(url)
	if err != nil || len(meta) == 0 {
		return statusCode, headers, content, err
	}

	key := CacheKey(url, meta, reqHeaders)

	encodedBase64Value, err := Get(key)
	if err != nil {
		return statusCode, headers, content, err
	}

	gob.Register(Response{})
	decodedValue := utils.Base64Decode([]byte(encodedBase64Value))
	utils.DecodeGob(decodedValue, &response)

	statusCode = response.StatusCode
	headers = response.Headers
	content = string(utils.Base64Decode([]byte(response.Content)))

	return statusCode, headers, content, nil
}

func CacheKey(url string, meta []string, reqHeaders map[string]string) string {
	key := []string{"GOB", url}

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

// func GetCacheKey(req http.Request, resp server.LoggedResponseWriter) string {
// 	url := req.URL.String()

// 	key := []string{"GOB", url}

// 	vary := strings.Split(resp.Header().Get("Vary"), ",")
// 	for _, k := range vary {
// 		key = append(key, resp.Header().Get(k))
// 	}

// 	cacheKey := strings.Join(key, "@@")

// 	return cacheKey
// }
