package okex5

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/szmcdull/ccexgo/exchange"
	"github.com/szmcdull/ccexgo/internal/rpc"
)

type (
	WSClient struct {
		*exchange.WSClient
		data   chan interface{}
		key    string
		secret string
		passwd string
	}

	Okex5Channel struct {
		Channel  string   `json:"channel"`
		InstType InstType `json:"instType,omitempty"`
		Uly      string   `json:"uly,omitempty"`
		InstID   string   `json:"instId,omitempty"`
	}
)

const (
	WebSocketPublicAddr     = "wss://ws.okx.com:8443/ws/v5/public"
	WebSocketPrivateAddr    = "wss://ws.okx.com:8443/ws/v5/private"
	WebSocketSimPublicAddr  = "wss://wspap.okx.com:8443/ws/v5/public?brokerId=9999"
	WebSocketSimPrivateAdrr = "wss://wspap.okx.com:8443/ws/v5/private?brokerId=9999"

	MethodSubscribe   = "subscribe"
	MethodUnSubscribe = "unsubscribe"
)

func NewWSPublicClient(data chan interface{}) *WSClient {
	return newWSClient(WebSocketPublicAddr, data)
}

func NewTestWSPublicClient(data chan interface{}) *WSClient {
	return newWSClient(WebSocketSimPublicAddr, data)
}

func newWSClient(addr string, data chan interface{}) *WSClient {
	ret := &WSClient{
		data: data,
	}
	ret.WSClient = exchange.NewWSClient(addr, NewCodec(), ret)
	return ret
}

func (ws *WSClient) Run(ctx context.Context) error {
	if err := ws.WSClient.Run(ctx); err != nil {
		return err
	}

	go func() {
		ticker := time.NewTicker(time.Second * 25)
		for {
			select {
			case <-ticker.C:
				ws.WSClient.Call(ctx, "1", pingMethod, "", nil)
			case <-ws.WSClient.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (ws *WSClient) Handle(ctx context.Context, notify *rpc.Notify) {
	data := &exchange.WSNotify{
		Exchange: "okex",
		Chan:     notify.Method,
		Data:     notify.Params,
	}

	select {
	case ws.data <- data:
	default:
	}
}

func (ws *WSClient) Subscribe(ctx context.Context, channels ...exchange.Channel) error {
	if len(channels) != 1 {
		return errors.Errorf("only 1 channel is support")
	}

	c := channels[0].(*Okex5Channel)

	var resp wsResp
	if err := ws.Call(ctx, MethodSubscribe, MethodSubscribe, []Okex5Channel{*c}, &resp); err != nil {
		return errors.WithMessage(err, "subscribe error")
	}
	return nil
}

func (ws *WSClient) UnSubscribe(ctx context.Context, channels ...exchange.Channel) error {
	if len(channels) != 1 {
		return errors.Errorf("only 1 channel is support")
	}

	c := channels[0].(*Okex5Channel)

	var resp wsResp
	if err := ws.Call(ctx, MethodUnSubscribe, MethodUnSubscribe, []Okex5Channel{*c}, &resp); err != nil {
		return errors.WithMessage(err, "subscribe error")
	}
	return nil

}

func (oc *Okex5Channel) String() string {
	return ""
}
