package margin

import "github.com/szmcdull/ccexgo/exchange/okex"

type (
	RestClient struct {
		*okex.RestClient
	}
)

func NewRestClient(key, secret, pass string) *RestClient {
	return &RestClient{
		okex.NewRestClient(key, secret, pass),
	}
}
