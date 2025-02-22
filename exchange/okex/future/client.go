package future

import "github.com/szmcdull/ccexgo/exchange/okex"

type (
	RestClient struct {
		*okex.RestClient
	}
)

func NewRestClient(key, secret, password string) *RestClient {
	return &RestClient{
		okex.NewRestClient(key, secret, password),
	}
}
