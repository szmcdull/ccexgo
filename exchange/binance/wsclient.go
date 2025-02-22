package binance

import (
	"context"
	"sync"
	"time"

	"github.com/go-kit/log/level"
	"github.com/pkg/errors"
	"github.com/szmcdull/ccexgo/exchange"
	"github.com/szmcdull/ccexgo/internal/rpc"
	"github.com/szmcdull/ccexgo/misc/ctxlog"
)

type (
	ListenKeyClient interface {
		GetListenKeyAddr(ctx context.Context) (string, error)
		PersistListenKey(ctx context.Context) error
		DeleteListenKey(ctx context.Context) error
	}

	// WSClient common private wsclient for binance
	WSClient struct {
		rpc.Conn
		handler rpc.Handler
		codec   rpc.Codec
		client  ListenKeyClient
	}

	NotifyClient struct {
		*exchange.WSClient
		data chan interface{}
		mu   sync.Mutex
	}
)

func NewWSClient(codec rpc.Codec, handler rpc.Handler, client ListenKeyClient) *WSClient {
	return &WSClient{
		handler: handler,
		codec:   codec,
		client:  client,
	}
}

func (ws *WSClient) Run(ctx context.Context) error {
	addr, err := ws.client.GetListenKeyAddr(ctx)
	if err != nil {
		return errors.WithMessage(err, "get listenKey addr fail")
	}

	logger := ctxlog.GetSafeLog(ctx)
	level.Debug(logger).Log("message", "get listenKeyAddr", "addr", addr)

	go func() {
		ticker := time.NewTicker(time.Minute * 59)

		defer func() {
			ws.client.DeleteListenKey(context.Background())
		}()
		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if err := ws.client.PersistListenKey(ctx); err != nil {
					level.Warn(logger).Log("message", "persist listenKey fail", "error", err.Error())
					return
				}
			}
		}
	}()

	stream, err := rpc.NewWebsocketStream(addr, ws.codec)
	if err != nil {
		return errors.WithMessage(err, "create websocket stream fail")
	}

	conn := rpc.NewConn(stream)
	ws.Conn = conn

	go ws.Conn.Run(ctx, ws.handler)
	return nil
}

func NewNotifyClient(addr string, codec rpc.Codec, data chan interface{}, handler rpc.Handler) *NotifyClient {
	ret := &NotifyClient{
		data: data,
	}

	if handler == nil {
		handler = ret
	}

	ret.WSClient = exchange.NewWSClient(addr, codec, handler)
	return ret
}

func (nc *NotifyClient) Handle(ctx context.Context, notify *rpc.Notify) {
	nc.Push(notify.Method, notify.Params)
}

func (nc *NotifyClient) Push(ch string, data interface{}) {
	notify := &exchange.WSNotify{Exchange: Exchange, Chan: ch, Data: data}
	select {
	case nc.data <- notify:
	default:
	}
}

func (wcl *NotifyClient) Subscribe(ctx context.Context, channels ...exchange.Channel) error {
	wcl.mu.Lock()
	defer wcl.mu.Unlock()

	param := make([]string, 0, len(channels))
	for _, c := range channels {
		param = append(param, c.String())
	}

	if err := wcl.Call(ctx, "1", MethodSubscibe, param, nil); err != nil {
		return errors.WithMessage(err, "subscribe error")
	}
	return nil
}

func (wcl *NotifyClient) UnSubscribe(ctx context.Context, channels ...exchange.Channel) error {
	wcl.mu.Lock()
	defer wcl.mu.Unlock()

	param := make([]string, 0, len(channels))
	for _, c := range channels {
		param = append(param, c.String())
	}

	if err := wcl.Call(ctx, "1", MethodUnSubscribe, param, nil); err != nil {
		return errors.WithMessage(err, "subscribe error")
	}
	return nil
}
