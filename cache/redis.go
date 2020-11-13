package cache_redis

import (
	"context"
	"encoding/gob"
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

func StoreFullPage(url string, status int, headers map[string]interface{}, content string, expiration time.Duration) (bool, error) {
	key := url

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

	err := rdb.Set(ctx, "GOB@@"+key, encodedBase64Value, expiration).Err()
	if err != nil {
		return false, err
	}

	return true, nil
}

func RetrieveFullPage(url string) (statusCode int, headers map[string]string, content string, err error) {
	key := url

	var response Response

	encodedBase64Value, err := rdb.Get(ctx, "GOB@@"+key).Result()
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
