package delivery

import (
	"github.com/pkg/errors"
	"github.com/szmcdull/ccexgo/exchange/binance"
	"github.com/szmcdull/ccexgo/internal/rpc"
	"github.com/tidwall/gjson"
)

type (
	CodeC struct {
		*binance.CodeC
	}
)

func NewCodeC() *CodeC {
	return &CodeC{
		CodeC: binance.NewCodeC(),
	}
}

func (cc *CodeC) Decode(raw []byte) (rpc.Response, error) {
	return cc.DecodeByCB(raw, func(g *gjson.Result) (rpc.Response, error) {

		if g.Get("e").String() == "bookTicker" {
			notify := ParseBookTickerNotify(g)
			return &rpc.Notify{Params: notify, Method: "bookTicker"}, nil
		}
		return nil, errors.Errorf("bad notify msg=%s", g.Raw)
	})
}
