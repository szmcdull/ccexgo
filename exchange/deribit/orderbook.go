package deribit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/szmcdull/ccexgo/exchange"
	"github.com/szmcdull/ccexgo/internal/rpc"
	"github.com/szmcdull/ccexgo/misc/tconv"
)

type (
	BookStats struct {
		Volume      decimal.Decimal
		PriceChange decimal.Decimal
		Low         decimal.Decimal
		High        decimal.Decimal
	}

	BookGreeks struct {
		Delta decimal.Decimal
		Gamma decimal.Decimal
		Rho   decimal.Decimal
		Theta decimal.Decimal
		Vega  decimal.Decimal
	}
	RestBookData struct {
		Timestamp       int64            `json:"timestamp"`
		Stats           BookStats        `json:"stats"`
		State           string           `json:"state"`
		SettlementPrice decimal.Decimal  `json:"settlement_price"`
		OpenInterest    decimal.Decimal  `json:"open_interest"`
		MinPrice        decimal.Decimal  `json:"min_price"`
		MaxPrice        decimal.Decimal  `json:"max_price"`
		MarkPrice       decimal.Decimal  `json:"mark_price"`
		MarkIV          decimal.Decimal  `json:"mark_iv"`
		LastPrice       decimal.Decimal  `json:"last_price"`
		InstrumentName  string           `json:"instrument_name"`
		IndexPrice      decimal.Decimal  `json:"index_price"`
		ChangeID        int              `json:"change_id'`
		Bids            [][2]interface{} `json:"bids"`
		Asks            [][2]interface{} `json:"asks"`
		BestBidPrice    decimal.Decimal  `json:"best_bid_price"`
		BestBidAmount   decimal.Decimal  `json:"best_bid_amount"`
		BestAskPrice    decimal.Decimal  `json:"best_ask_price"`
		BestAskAmount   decimal.Decimal  `json:"best_ask_amount"`
		Funding8H       decimal.Decimal  `json:"funding_8h"`
		CurrentFunding  decimal.Decimal  `json:"current_funding"`
		Greeks          BookGreeks       `json:"greeks"`
	}

	RestBookReq struct {
		InstrumentName string `json:"instrument_name"`
		Depth          string `json:"depth,omitempty"`
	}
	BookData struct {
		Timestamp      int              `json:"timestamp"`
		InstrumentName string           `json:"instrument_name"`
		ChangeID       int              `json:"charge_id"`
		PrevChangeID   int              `json:"prev_change_id"`
		Bids           [][3]interface{} `json:"bids"`
		Asks           [][3]interface{} `json:"asks"`
	}

	BookSnapData struct {
		Timestamp      int64        `json:"timestamp"`
		InstrumentName string       `json:"instrument_name"`
		ChangeID       int          `json:"charge_id"`
		Bids           [][2]float64 `json:"bids"`
		Asks           [][2]float64 `json:"asks"`
	}

	ChOrderBook struct {
		sym exchange.Symbol
	}

	ChOrderBookSnap struct {
		sym   exchange.Symbol
		depth int
		group string
	}
)

const (
	RestOrderBookMethod = "public/get_order_book"
)

func init() {
	reigisterCB("book", parseNotifyBook)
}

// NewOrderBookChannel return channel for change of order book
func NewOrderBookChannel(sym exchange.Symbol) exchange.Channel {
	return &ChOrderBook{
		sym: sym,
	}
}

func (co *ChOrderBook) String() string {
	return fmt.Sprintf("book.%s.raw", co.sym.String())
}

// NewOrderBookSnap return channel for snapshot of orderbook
func NewOrderBookSnapChannel(sym exchange.Symbol, group string, depth int) exchange.Channel {
	return &ChOrderBookSnap{
		sym:   sym,
		depth: depth,
		group: group,
	}
}

func (cos *ChOrderBookSnap) String() string {
	return fmt.Sprintf("book.%s.%s.%d.100ms", cos.sym.String(), cos.group, cos.depth)
}

func (client *Client) FetchOrderBook(ctx context.Context, symbol exchange.Symbol, maxDepth int) (*exchange.OrderBook, error) {
	var ob RestBookData
	req := RestBookReq{
		InstrumentName: symbol.String(),
		Depth:          fmt.Sprintf("%d", maxDepth),
	}
	if err := client.call(ctx, RestOrderBookMethod, &req, &ob, false); err != nil {
		return nil, err
	}

	return ob.Transform(symbol)
}

func (ob *RestBookData) Transform(sym exchange.Symbol) (*exchange.OrderBook, error) {
	bids := make([]exchange.OrderElem, len(ob.Bids))
	for i, bid := range ob.Bids {
		bids[i].Price = bid[0].(float64)
		bids[i].Amount = bid[1].(float64)
	}

	asks := make([]exchange.OrderElem, len(ob.Asks))
	for i, ask := range ob.Asks {
		asks[i].Price = ask[0].(float64)
		asks[i].Amount = ask[1].(float64)
	}

	bk := *ob
	return &exchange.OrderBook{
		Symbol: sym,
		Bids:   bids,
		Asks:   asks,
		Raw:    &bk,
	}, nil
}

func parseNotifyBook(resp *Notify) (*rpc.Notify, error) {
	fields := strings.Split(resp.Channel, ".")

	if len(fields) == 3 {
		var bn BookData
		if err := json.Unmarshal(resp.Data, &bn); err != nil {
			return nil, errors.WithMessage(err, "unarshal orderbookNotify")
		}
		sym, err := ParseOptionSymbol(fields[1])
		if err != nil {
			return nil, errors.WithMessagef(err, "parse orderbookNotify symbol %s", fields[1])
		}
		notify := &rpc.Notify{
			Method: subscriptionMethod,
		}
		on := &exchange.OrderBookNotify{
			Symbol: sym,
			Asks:   make([]exchange.OrderElem, len(bn.Asks)),
			Bids:   make([]exchange.OrderElem, len(bn.Bids)),
			Raw:    &bn,
		}

		if err := processArr(on.Asks, bn.Asks); err != nil {
			return nil, err
		}
		if err := processArr(on.Bids, bn.Bids); err != nil {
			return nil, err
		}
		notify.Params = on
		return notify, nil
	} else if len(fields) == 5 {
		var bn BookSnapData
		if err := json.Unmarshal(resp.Data, &bn); err != nil {
			return nil, errors.WithMessage(err, "unmarshal orderbook fail")
		}

		sym, err := ParseOptionSymbol(fields[1])
		if err != nil {
			return nil, errors.WithMessage(err, "parseSymbol fail")
		}

		notify := &rpc.Notify{
			Method: subscriptionMethod,
		}

		on := &exchange.OrderBook{
			Symbol:  sym,
			Bids:    make([]exchange.OrderElem, len(bn.Bids)),
			Asks:    make([]exchange.OrderElem, len(bn.Asks)),
			Created: tconv.Milli2Time(int64(bn.Timestamp)),
			Raw:     &bn,
		}
		processSnapArr(bn.Bids, on.Bids)
		processSnapArr(bn.Asks, on.Asks)
		notify.Params = on
		return notify, nil
	} else {
		return nil, errors.Errorf("unkown channel='%s'", resp.Channel)
	}
}

func processArr(d []exchange.OrderElem, s [][3]interface{}) (ret error) {
	defer func() {
		if err := recover(); err != nil {
			ret = err.(error)
		}
	}()

	for i, v := range s {
		op := v[0].(string)
		price := v[1].(float64)
		amount := v[2].(float64)

		if op == "new" || op == "change" {
			d[i].Amount = amount
			d[i].Price = price
		} else if op == "delete" {
			d[i].Amount = 0
			d[i].Price = price
		} else {
			ret = errors.Errorf("unkown op %s", op)
			return
		}
	}
	return
}

func processSnapArr(src [][2]float64, dst []exchange.OrderElem) {
	for i, v := range src {
		elem := exchange.OrderElem{
			Price:  v[0],
			Amount: v[1],
		}
		dst[i] = elem
	}
}
