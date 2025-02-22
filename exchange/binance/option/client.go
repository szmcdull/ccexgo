package option

import (
	"context"

	"github.com/pkg/errors"
	"github.com/szmcdull/ccexgo/exchange/binance"
)

type (
	RestClient struct {
		*binance.RestClient
		wsAddr string
	}

	RestResp struct {
		Code int         `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}
)

func NewRestClient(key, secret string) *RestClient {
	return &RestClient{
		wsAddr:     "vstream.binance.com",
		RestClient: binance.NewRestClient(key, secret, "vapi.binance.com"),
	}
}

func NewTestRestClient(key, secret string) *RestClient {
	return &RestClient{
		wsAddr:     "testnetws.binanceops.com",
		RestClient: binance.NewRestClient(key, secret, "testnet.binanceops.com"),
	}
}

// GetRequest do get request. the dst field will be wrapped in restResp data field
func (rc *RestClient) GetRequest(ctx context.Context, endPoint string, req binance.GetRestReq, sign bool, dst interface{}) error {
	resp := RestResp{
		Data: dst,
	}
	if err := rc.RestClient.GetRequest(ctx, endPoint, req, sign, &resp); err != nil {
		return errors.WithMessagef(err, "query endPoint='%s' fail", endPoint)
	}

	if resp.Code != 0 {
		return errors.Errorf("invalid resp code=%d msg=%s", resp.Code, resp.Msg)
	}
	return nil
}
