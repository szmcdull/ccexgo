package exchange

import (
	"context"
	"sync"

	"github.com/NadiaSama/ccexgo/internal/rpc"
)

type (
	ConnCB func(addr string) (rpc.Conn, error)

	Client struct {
		NewConn ConnCB
		Conn    rpc.Conn
		Addr    string
		Key     string
		Secret  string
		SubMu   sync.Mutex
		Sub     map[string]interface{}
	}
)

//NewClient got a new client instance
func NewClient(cb ConnCB, addr, key, secret string) *Client {
	return &Client{
		NewConn: cb,
		Addr:    addr,
		Key:     key,
		Secret:  secret,
		Sub:     make(map[string]interface{}),
	}
}

//Run create wsconn and start conn running loop
func (c *Client) Run(ctx context.Context) error {
	conn, err := c.NewConn(c.Addr)
	if err != nil {
		return err
	}
	c.Conn = conn
	go c.Conn.Run(ctx, c)
	return nil
}
