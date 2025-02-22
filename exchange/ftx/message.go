package ftx

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/szmcdull/ccexgo/exchange"
	"github.com/szmcdull/ccexgo/internal/rpc"
)

type (
	CodeC struct {
		*exchange.CodeC
		orderBook map[string]*OrderBook
		trade     map[string]*Trade
	}

	callParam struct {
		Channel string `json:"channel,omitempty"`
		Market  string `json:"market,omitempty"`
		OP      string `json:"op,omitempty"`
	}

	callResponse struct {
		Channel string          `json:"channel"`
		Market  string          `json:"market"`
		Type    string          `json:"type"`
		Code    int             `json:"code"`
		Msg     string          `json:"msg"`
		Data    json.RawMessage `json:"data"`
	}

	authArgs struct {
		Key  string `json:"key"`
		Sign string `json:"sign"`
		Time int64  `json:"time"`
	}

	authParam struct {
		Args authArgs `json:"args"`
		OP   string   `json:"op"`
	}
)

const (
	typeError        = "error"
	typeSubscribed   = "subscribed"
	typeUnSubscribed = "unsubscribed"
	typePong         = "pong"
	typeInfo         = "info"
	typePartial      = "partial"
	typeUpdate       = "update"

	codeReconnet = 20001

	channelOrderBook        = "orderbook"
	channelOrders           = "orders"
	channelFills            = "fills"
	channelTrades           = "trades"
	channelTicker           = "ticker"
	channelMarkets          = "markets"
	channelOrderbookGrouped = "orderbookGrouped"
	channelFtxPay           = "ftxPay" // todo
)

func NewCodeC() *CodeC {
	return &CodeC{
		exchange.NewCodeC(),
		make(map[string]*OrderBook),
		make(map[string]*Trade),
	}
}

func (cc *CodeC) Decode(raw []byte) (rpc.Response, error) {
	var cr callResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, err
	}

	id := subID(cr.Channel, cr.Market)

	if cr.Type == typeError {
		ret := &rpc.Result{
			ID:     id,
			Error:  errors.Errorf("error msg: %s code: %d", cr.Msg, cr.Code),
			Result: raw,
		}
		return ret, nil
	}

	switch cr.Type {
	case typeSubscribed:
		fallthrough
	case typeUnSubscribed:
		ret := &rpc.Result{
			ID:     id,
			Result: raw,
		}
		return ret, nil

	case typePong:
		ret := &rpc.Notify{
			Method: typePong,
		}
		return ret, nil

	case typeInfo:
		if cr.Code == codeReconnet {
			return nil, rpc.NewStreamError(errors.Errorf("ftx ws reset info %s", string(raw)))
		}
		ret := &rpc.Notify{
			Method: id,
			Params: cr.Data,
		}
		return ret, nil

	case typePartial:
		switch cr.Channel {
		case channelOrderBook:
			sym, err := ParseSymbol(cr.Market)
			if err != nil {
				return nil, errors.Errorf("unknow market '%s'", cr.Market)
			}
			ob := NewOrderBook(sym)
			notify, err := ob.Init(&cr)
			if err != nil {
				return nil, err
			}
			cc.orderBook[cr.Market] = ob

			return &rpc.Notify{
				Method: id,
				Params: notify,
			}, nil

		default:
			return nil, errors.Errorf("unsupport partial data %s %s", cr.Channel, cr.Market)
		}

	case typeUpdate:
		var param interface{}
		switch cr.Channel {
		case channelOrders:
			o, err := cc.parseOrder(cr.Data)
			if err != nil {
				return nil, err
			}
			param = o

		case channelFills:
			f, err := cc.parseFills(cr.Data)
			if err != nil {
				return nil, err
			}
			param = f

		case channelOrderBook:
			fmt.Println("orderbook Update")
			ob, ok := cc.orderBook[cr.Market]
			if !ok {
				return nil, errors.Errorf("unkown market '%s'", cr.Market)
			}
			f, err := ob.Update(&cr)
			if err != nil {
				return nil, err
			}
			param = f

		case channelTrades:
			fmt.Println("tradesUpdate")
			f, err := cc.parseTrades(cr.Data)
			if err != nil {
				return nil, err
			}
			param = f

		default:
			return nil, errors.Errorf("unsupport channel '%s'", cr.Channel)
		}
		ret := &rpc.Notify{
			Method: id,
			Params: param,
		}
		return ret, nil

	default:
		return nil, errors.Errorf("unsupport type '%s'", cr.Type)
	}
}

func (cc *CodeC) parseOrder(raw []byte) (*exchange.Order, error) {
	var order Order
	if err := json.Unmarshal(raw, &order); err != nil {
		return nil, err
	}
	return parseOrderInternal(&order)
}

func (cc *CodeC) parseFills(raw []byte) (*Fill, error) {
	var fill FillNotify
	if err := json.Unmarshal(raw, &fill); err != nil {
		return nil, err
	}

	return parseFillInternal(&fill)
}

func (cc *CodeC) parseTrades(raw []byte) ([]*Trade, error) {
	var trade []*TradeNotify
	if err := json.Unmarshal(raw, &trade); err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	return parseTradesInternal(trade)
}

func subID(channel string, market string) string {
	if len(market) == 0 {
		return channel
	}

	return fmt.Sprintf("%s.%s", channel, market)
}
