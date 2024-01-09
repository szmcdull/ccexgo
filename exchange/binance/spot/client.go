package spot

import "github.com/szmcdull/ccexgo/exchange/binance"

type (
	RestClient struct {
		*binance.RestClient
	}
)

func NewRestClient(key, secret string) *RestClient {
	return &RestClient{
		binance.NewRestClient(key, secret, "api.binance.com"),
	}
}
