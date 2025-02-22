package websocket

import (
	"context"

	"github.com/pkg/errors"
	"github.com/szmcdull/ccexgo/exchange"
)

type (
	//Gen is a interface for unify rpc connection and update subscription channel
	Gen interface {
		NewConn(ctx context.Context) (Conn, error)
		//Channel return channels which need to be subscribe.
		Channels(ctx context.Context, oldChannel []exchange.Channel) (newChannels []exchange.Channel, notify chan struct{}, err error)
	}

	//Keeper is a struct which used to make websocket connection auto reconnect and auto update subscribe channels
	Keeper struct {
		channels []exchange.Channel
		conn     Conn
		gen      Gen
		done     chan struct{}
		close    chan struct{}
		closed   bool
		ech      chan error
	}
)

func NewKeeper(gen Gen) *Keeper {
	return &Keeper{
		gen:   gen,
		done:  make(chan struct{}, 0),
		ech:   make(chan error, 1),
		close: make(chan struct{}),
	}
}

func (k *Keeper) Loop(ctx context.Context) {
	defer close(k.done)
	for {
		conn, err := k.gen.NewConn(ctx)
		if err == context.Canceled || err == context.DeadlineExceeded || k.closed {
			return
		}
		if err != nil {
			k.pushErrorClose(err)
			continue
		}
		k.conn = conn
		k.channels = nil
		k.connLoop(ctx)
	}
}

// Close loop manualy
func (k *Keeper) Close() {
	k.closed = true
	close(k.close)
}

// ECh push error when error happen
func (k *Keeper) ECh() chan error {
	return k.ech
}

func (k *Keeper) Done() chan struct{} {
	return k.done
}

func (k *Keeper) connLoop(ctx context.Context) {
	notify, err := k.updateSubscribe(ctx)
	if err != nil {
		k.pushErrorClose(err)
		return
	}
	for {
		select {
		case <-k.conn.Done():
			if err := k.conn.Error(); err != nil {
				k.pushErrorClose(err)
			}
			return

		case <-k.close:
			if err := k.conn.Close(); err != nil {
				k.pushErrorClose(err)
				return
			}

		case <-notify:
			notify, err = k.updateSubscribe(ctx)
			if err != nil {
				k.pushErrorClose(err)
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

func (k *Keeper) updateSubscribe(ctx context.Context) (chan struct{}, error) {
	channels, notify, err := k.gen.Channels(ctx, k.channels)
	if err != nil {
		return nil, err
	}

	if k.channels != nil {
		if err := k.conn.UnSubscribe(ctx, k.channels...); err != nil {
			return nil, errors.WithMessage(err, "unsubscribe channel fail")
		}
	}
	for _, c := range channels {
		//okex subscirbe multi channel not work
		if err := k.conn.Subscribe(ctx, c); err != nil {
			return nil, errors.WithMessage(err, "subscribe channel fail")
		}
	}
	k.channels = channels
	return notify, nil
}

func (k *Keeper) pushErrorClose(err error) {
	if k.conn != nil {
		k.conn.Close()
	}

	select {
	case k.ech <- err:
	default:
	}
}
