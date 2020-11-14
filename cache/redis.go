package cache_redis

import (
	"context"
	"encoding/gob"
	"strings"
	"time"

	"github.com/fabiocicerchia/go-proxy-cache/config"
	"github.com/fabiocicerchia/go-proxy-cache/utils"
	"github.com/go-redis/redis/v8"
)

type Response struct {
	StatusCode int
	Headers    map[string]string
	Content    string
}

var ctx = context.Background()

var rdb *redis.Client

func Connect(config config.Cache) bool {
	if rdb != nil {
		// test the connection
		_, err := rdb.Ping(ctx).Result()
		return err == nil
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     config.Host + ":" + config.Port,
		Password: config.Password,
		DB:       config.DB,
	})

	return true
}

// TODO: move out
func StoreFullPage(url string, status int, headers map[string]interface{}, reqHeaders map[string]string, content string, expiration time.Duration) (bool, error) {
	// TODO: move key out of redis module
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
	encodedBase64Value := utils.Base64Encode(valueToEncode)

	meta := GetVary(headersConverted)
	StoreMetadata(url, meta, expiration)
	key := CacheKey(url, meta, reqHeaders)

	// TODO: extract
	err := rdb.Set(ctx, key, encodedBase64Value, expiration).Err()
	if err != nil {
		return false, err
	}

	return true, nil
}

// TODO: move out
func RetrieveFullPage(url string, reqHeaders map[string]string) (statusCode int, headers map[string]string, content string, err error) {
	var response Response

	meta, err := FetchMetadata(url)
	if err != nil || len(meta) == 0 {
		return statusCode, headers, content, err
	}

	key := CacheKey(url, meta, reqHeaders)
	// TODO: extract
	encodedBase64Value, err := rdb.Get(ctx, key).Result()
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

func FetchMetadata(url string) (meta []string, err error) {
	key := "META@@" + url

	value, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return meta, err
	}

	// TODO: USE HGETALL? OR SET?
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

// TODO: move out
func CacheKey(url string, meta []string, reqHeaders map[string]string) string {
	key := []string{"GOB", url}

	vary := meta
	for _, k := range vary {
		key = append(key, reqHeaders[k])
	}

	cacheKey := strings.Join(key, "@@")

	return cacheKey
}

func GetVary(headers map[string]string) []string {
	vary := strings.Split(headers["Vary"], ",")
	for k, v := range vary {
		v = strings.Trim(v, " ")
		vary[k] = v
	}
	return vary
}
