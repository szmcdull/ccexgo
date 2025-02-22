package ftx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/szmcdull/ccexgo/exchange"
)

type (
	Order struct {
		CreatedAt     string          `json:"createdAt"`
		FilledSize    decimal.Decimal `json:"filledSize"`
		Future        string          `json:"future"`
		ID            int64           `json:"id"`
		Market        string          `json:"market"`
		Price         decimal.Decimal `json:"price"`
		AvgFillPrice  decimal.Decimal `json:"avgFillPrice"`
		RemainingSize decimal.Decimal `json:"remainingSize"`
		Side          string          `json:"side"`
		Size          decimal.Decimal `json:"size"`
		Status        string          `json:"status"`
		Type          string          `json:"type"`
		ReduceOnly    bool            `json:"reduceOnly"`
		IOC           bool            `json:"ioc"`
		PostOnly      bool            `json:"postOnly"`
		ClientID      string          `json:"clientId"`
	}

	OrderReq struct {
		Market   string  `json:"market"`
		Side     string  `json:"side"`
		Price    float64 `json:"price"`
		Type     string  `json:"type"`
		Size     float64 `json:"size"`
		ClientID string  `json:"clientId,omitempty"`
	}

	OrdersHistoryReq struct {
		Market    string `json:"market"`
		Side      string `json:"side"`
		OrderType string `json:"orderType"`
		StartTime int    `json:"startTime"`
		EndTime   int    `json:"endTime"`
	}

	OrderChannel struct {
		symbol exchange.Symbol
	}
)

const (
	ftxOrderNew   = "new"
	ftxOrderOpen  = "open"
	ftxOrderClose = "closed"

	orderEndPoint        = "/orders"
	orderHistoryEndPoint = "/orders/history"
)

var (
	typeMap map[string]exchange.OrderType = map[string]exchange.OrderType{
		"limit":  exchange.OrderTypeLimit,
		"market": exchange.OrderTypeMarket,
	}

	typeRMap map[exchange.OrderType]string = map[exchange.OrderType]string{
		exchange.OrderTypeLimit:  "limit",
		exchange.OrderTypeMarket: "market",
	}

	sideMap map[string]exchange.OrderSide = map[string]exchange.OrderSide{
		"buy":  exchange.OrderSideBuy,
		"sell": exchange.OrderSideSell,
	}

	sideRMap map[exchange.OrderSide]string = map[exchange.OrderSide]string{
		exchange.OrderSideBuy:  "buy",
		exchange.OrderSideSell: "sell",
	}
)

func NewOrderChannel(sym exchange.Symbol) exchange.Channel {
	return &OrderChannel{
		symbol: sym,
	}
}

func (oc *OrderChannel) String() string {
	return oc.symbol.String()
}

func (rc *RestClient) OrderNew(ctx context.Context, req *exchange.OrderRequest, options ...exchange.OrderReqOption) (*exchange.Order, error) {
	side, ok := sideRMap[req.Side]
	if !ok {
		return nil, errors.Errorf("unkown orderside '%d'", req.Side)
	}
	typ, ok := typeRMap[req.Type]
	if !ok {
		return nil, errors.Errorf("unkown order type '%d'", req.Type)
	}

	p, _ := req.Price.Float64()
	a, _ := req.Amount.Float64()
	cid := ""
	if req.ClientID != nil {
		cid = req.ClientID.String()
	}
	or := OrderReq{
		Market:   req.Symbol.String(),
		Price:    p,
		Size:     a,
		Side:     side,
		Type:     typ,
		ClientID: cid,
	}
	b, _ := json.Marshal(or)
	buf := bytes.NewBuffer(b)

	var o Order
	if err := rc.request(ctx, http.MethodPost, orderEndPoint, nil, buf, true, &o); err != nil {
		return nil, err
	}
	return rc.parseOrder(&o)
}

func (rc *RestClient) parseOrder(o *Order) (*exchange.Order, error) {
	return parseOrderInternal(o)
}
func parseOrderInternal(o *Order) (*exchange.Order, error) {
	ct, err := time.Parse("2006-01-02T15:04:05.000000Z07:00", o.CreatedAt)
	if err != nil {
		return nil, errors.WithMessagef(err, "bad create time '%s'", o.CreatedAt)
	}
	var os exchange.OrderStatus
	if o.Status == ftxOrderNew || o.Status == ftxOrderOpen {
		os = exchange.OrderStatusOpen
	} else {
		if o.FilledSize == o.Size {
			os = exchange.OrderStatusDone
		} else {
			os = exchange.OrderStatusCancel
		}
	}

	symbol, err := ParseSymbol(o.Market)
	if err != nil {
		return nil, errors.WithMessagef(err, "parse symbol fail")
	}

	typ, ok := typeMap[o.Type]
	if !ok {
		return nil, errors.Errorf("unkown order type '%s'", o.Type)
	}

	side, ok := sideMap[o.Side]
	if !ok {
		return nil, errors.Errorf("unkown order side '%s'", o.Side)
	}

	order := &exchange.Order{
		ID:       exchange.NewIntID(o.ID),
		Symbol:   symbol,
		Amount:   o.Size,
		Filled:   o.FilledSize,
		Price:    o.Price,
		AvgPrice: o.AvgFillPrice,
		Created:  ct,
		Updated:  ct,
		Status:   os,
		Side:     side,
		Type:     typ,
		Raw:      o,
	}
	return order, nil
}

// OrderCancel only ID field is required
func (rc *RestClient) OrderCancel(ctx context.Context, order *exchange.Order) error {
	endPoint := fmt.Sprintf("%s/%s", orderEndPoint, order.ID.String())

	if err := rc.request(ctx, http.MethodDelete, endPoint, nil, nil, true, nil); err != nil {
		return err
	}
	return nil
}

// OrderFetch only ID field is required
func (rc *RestClient) OrderFetch(ctx context.Context, order *exchange.Order) (*exchange.Order, error) {
	endPoint := fmt.Sprintf("%s/%s", orderEndPoint, order.ID.String())

	var resp Order
	if err := rc.request(ctx, http.MethodGet, endPoint, nil, nil, true, &resp); err != nil {
		return nil, err
	}
	return rc.parseOrder(&resp)
}

// Orders return open orders
func (rc *RestClient) Orders(ctx context.Context, symbol exchange.Symbol) ([]*exchange.Order, error) {
	var orders []Order
	var param url.Values
	if symbol != nil {
		param = url.Values{}
		param.Add("markets", symbol.String())
	}
	if err := rc.request(ctx, http.MethodGet, "/orders", param, nil, true, &orders); err != nil {
		return nil, err
	}

	ret := make([]*exchange.Order, len(orders))
	for i, o := range orders {
		to, e := rc.parseOrder(&o)
		if e != nil {
			return nil, e
		}
		ret[i] = to
	}

	return ret, nil
}

func parseTime(ts string) (time.Time, error) {
	ct, err := time.Parse("2006-01-02T15:04:05.000000Z07:00", ts)
	if err != nil {
		return time.Time{}, errors.WithMessagef(err, "bad create time '%s'", ts)
	}
	return ct, nil
}

func NewOrderHistoryReq(market, side, orderType string, startTime, endTime int) *OrdersHistoryReq {
	return &OrdersHistoryReq{
		Market:    market,
		Side:      side,
		OrderType: orderType,
		StartTime: startTime,
		EndTime:   endTime,
	}
}

func (rc *RestClient) OrdersHistory(ctx context.Context, param *OrdersHistoryReq) ([]*Order, error) {
	values := url.Values{}

	if param.Market != "" {
		values.Add("market", param.Market)
	}

	if param.Side != "" {
		values.Add("side", param.Side)
	}

	if param.OrderType != "" {
		values.Add("orderType", param.OrderType)
	}

	if param.StartTime != 0 {
		values.Add("startTime", strconv.Itoa(param.StartTime))
	}

	if param.EndTime != 0 {
		values.Add("endTime", strconv.Itoa(param.EndTime))
	}

	var ret []*Order
	if err := rc.request(ctx, http.MethodGet, orderHistoryEndPoint, values, nil, true, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}
