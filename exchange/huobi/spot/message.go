package spot

import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/szmcdull/ccexgo/exchange/huobi"
	"github.com/szmcdull/ccexgo/internal/rpc"
)

type (
	CodeC struct {
		*huobi.CodeC
	}
)

func NewCodeC() *CodeC {
	return &CodeC{
		huobi.NewCodeC(),
	}
}
func (cc *CodeC) Decode(raw []byte) (rpc.Response, error) {
	msg, err := cc.Decompress(raw)
	if err != nil {
		return nil, err
	}

	var resp huobi.Response
	if err := json.Unmarshal(msg, &resp); err != nil {
		return nil, errors.WithMessagef(err, "bad response '%s'", string(msg))
	}

	ret, err := resp.Parse(msg)

	if ret != nil {
		return ret, nil
	}

	if err != nil && err != huobi.SkipError {
		return nil, err
	}

	var (
		r interface{}
	)

	if IsTradeDetailChanel(resp.Ch) {
		r, err = ParseTradeTick(resp.Ch, resp.TS, resp.Tick)
	} else {
		r, err = ParseDepth(resp.Ch, resp.TS, resp.Tick)
	}

	if err != nil {
		return nil, err
	}

	return &rpc.Notify{
		Method: resp.Ch,
		Params: r,
	}, nil
}
