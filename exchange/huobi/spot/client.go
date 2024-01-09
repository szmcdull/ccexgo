package spot

import (
	"github.com/szmcdull/ccexgo/exchange/huobi"
)

type (
	RestClient struct {
		*huobi.RestClient
		spotAccountID int
	}
)

const (
	SpotHost = "api.huobi.pro"
)

func NewRestClient(key, secret string) *RestClient {
	return &RestClient{
		RestClient: huobi.NewRestClient(key, secret, SpotHost),
	}
}
