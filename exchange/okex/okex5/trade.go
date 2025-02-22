package okex5

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/szmcdull/ccexgo/exchange"
)

type (
	CreateOrderReq struct {
		InstID     string    `json:"instId"`
		TDMode     TDMode    `json:"tdMode"`
		Ccy        string    `json:"ccy,omitempty"`
		ClOrderID  string    `json:"clOrderId,omitempty"`
		Tag        string    `json:"tag,omitempty"`
		Side       OrderSide `json:"side"`
		PosSide    PosSide   `json:"posSide,omitempty"`
		OrdType    OrdType   `json:"ordType"`
		Sz         string    `json:"sz"`
		Px         string    `json:"px,omitempty"`
		ReduecOnly bool      `json:"reduceOnly"`
	}

	CreateOrderResp struct {
		OrderID   string `json:"ordId"`
		ClOrderID string `json:"clOrdId"`
		Tag       string `json:"tag"`
		SCode     string `json:"sCode"`
		SMsg      string `json:"sMsg"`
	}

	CancelOrderReq struct {
		InstID  string `json:"instId"`
		OrdId   string `json:"ordId"`
		ClOrdID string `json:"clOrdId"`
	}

	CancelOrderResp struct {
		OrdID   string `json:"ordId"`
		ClOrdID string `json:"clOrdId"`
		SCode   string `json:"sCode"`
		SMsg    string `json:"sMsg"`
	}

	FetchOrderReq struct {
		InstID  string `json:"instId"`
		OrdID   string `json:"ordId"`
		ClOrdID string `json:"clOrdId"`
	}

	FillsReq struct {
		InstType InstType
		InstID   string
		OrdID    string
		Uly      string
		After    string
		Before   string
		Limit    string
	}

	Fill struct {
		InstType InstType  `json:"instType"`
		InstID   string    `json:"instId"`
		TradeID  string    `json:"tradeId"`
		OrdID    string    `json:"ordId"`
		ClOrdID  string    `json:"clOrdId"`
		BillID   string    `json:"billId"`
		Tag      string    `json:"tag"`
		FIllPx   string    `json:"fillPx"`
		FillSz   string    `json:"fillSz"`
		Side     OrderSide `json:"side"`
		PosSide  PosSide   `json:"posSide"`
		ExecType ExecType  `json:"execType"`
		FeeCcy   string    `json:"feeCcy"`
		Fee      string    `json:"fee"`
		Ts       string    `json:"ts"`
	}

	OrdersHistoryReq struct {
		InstType InstType      `json:"instType"`
		Uly      string        `json:"uly"`
		InstID   string        `json:"instId"`
		OrdType  OrdType       `json:"ordType"`
		State    string        `json:"state"`
		Category OrderCategory `json:"category"`
		After    string        `json:"after"`
		Before   string        `json:"before"`
		Limit    string        `json:"limit"`
	}

	Order struct {
		InstType    InstType      `json:"instType"`
		InstId      string        `json:"instId"`
		Ccy         string        `json:"ccy"`
		OrderID     string        `json:"ordId"`
		ClOrdID     string        `json:"clOrdId"`
		Tag         string        `json:"tag"`
		Px          string        `json:"px"`
		Sz          string        `json:"sz"`
		Pnl         string        `json:"pnl"`
		OrderType   OrdType       `json:"ordType"`
		Side        OrderSide     `json:"side"`
		PosSide     PosSide       `json:"posSide"`
		TDMode      TDMode        `json:"tdMode"`
		AccFillSZ   string        `json:"accFillSz"`
		FillPx      string        `json:"fillPx"`
		TradeID     string        `json:"tradeId"`
		FillSz      string        `json:"fillSz"`
		FillTime    string        `json:"fillTime"`
		AvgPx       string        `json:"avgPx"`
		State       OrderState    `json:"state"`
		Lever       string        `json:"lever"`
		TpTriggerPx string        `json:"tpTriggerPx"`
		TpOrdPx     string        `json:"tpOrdPx"`
		SlTriggerPx string        `json:"slTriggerPx"`
		SlOrdPx     string        `json:"slOrdPx"`
		FeeCcy      string        `json:"feeCcy"`
		Fee         string        `json:"fee"`
		RebateCcy   string        `json:"rebatCcy"`
		Rebat       string        `json:"rebat"`
		Category    OrderCategory `json:"category"`
		UTime       string        `json:"uTime"`
		CTime       string        `json:"cTime"`
	}
)

const (
	CreateOrderEndPoint = "/api/v5/trade/order"
	FetchOrderEndPoint  = CreateOrderEndPoint
	CancelOrderEndPoint = "/api/v5/trade/cancel-order"
	FillsEndPoint       = "/api/v5/trade/fills"
)

const ()

var ()

func (rc *RestClient) CreateOrder(ctx context.Context, req *CreateOrderReq) (*CreateOrderResp, error) {
	ret := []CreateOrderResp{}
	if err := rc.doPostJSON(ctx, CreateOrderEndPoint, req, &ret); err != nil {
		return nil, err
	}

	return &ret[0], nil
}

func (rc *RestClient) CancelOrder(ctx context.Context, req *CancelOrderReq) (*CancelOrderResp, error) {
	ret := []CancelOrderResp{}
	if err := rc.doPostJSON(ctx, CancelOrderEndPoint, req, &ret); err != nil {
		return nil, err
	}

	return &ret[0], nil
}

func (rc *RestClient) FetchOrder(ctx context.Context, req *FetchOrderReq) (*Order, error) {
	ret := []Order{}
	values := url.Values{}
	values.Add("instId", req.InstID)
	if len(req.OrdID) != 0 {
		values.Add("ordId", req.OrdID)
	} else if len(req.ClOrdID) != 0 {
		values.Add("clOrdId", req.ClOrdID)
	} else {
		return nil, errors.Errorf("ordID or clOrdId is required")
	}
	if err := rc.Request(ctx, http.MethodGet, FetchOrderEndPoint, values, nil, true, &ret); err != nil {
		return nil, err
	}

	return &ret[0], nil
}

func (rc *RestClient) OrdersHistory(ctx context.Context, param *OrdersHistoryReq) ([]Order, error) {
	values := url.Values{}
	values.Add("instType", string(param.InstType))

	if param.InstID != "" {
		values.Add("instId", param.InstID)
	}

	if param.Uly != "" {
		values.Add("uly", param.Uly)
	}

	if param.OrdType != "" {
		values.Add("ordType", string(param.OrdType))
	}

	if param.State != "" {
		values.Add("state", param.State)
	}

	if param.After != "" {
		values.Add("after", param.After)
	}

	if param.Before != "" {
		values.Add("before", param.Before)
	}

	if param.Category != "" {
		values.Add("category", string(param.Category))
	}

	if param.Limit != "" {
		values.Add("limit", param.Limit)
	}

	var ret []Order
	if err := rc.Request(ctx, http.MethodGet, "/api/v5/trade/orders-history", values, nil, true, &ret); err != nil {
		return nil, err
	}

	return ret, nil
}

func (rc *RestClient) Fills(ctx context.Context, param *FillsReq) ([]Fill, error) {
	values := url.Values{}
	if param.InstType != "" && param.InstType != InstTypeAny {
		values.Add("instType", string(param.InstType))
	}

	if param.InstID != "" {
		values.Add("instId", param.InstID)
	}

	if param.Uly != "" {
		values.Add("uly", param.Uly)
	}

	if param.OrdID != "" {
		values.Add("ordId", param.OrdID)
	}

	if param.After != "" {
		values.Add("after", param.After)
	}

	if param.Before != "" {
		values.Add("before", param.Before)
	}

	if param.Limit != "" {
		values.Add("limit", param.Limit)
	}

	var ret []Fill
	if err := rc.Request(ctx, http.MethodGet, FillsEndPoint, values, nil, true, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (rc *RestClient) Trades(ctx context.Context, req *exchange.TradeReqParam) ([]exchange.Trade, error) {
	var it InstType
	switch req.Symbol.(type) {
	case exchange.MarginSymbol:
		it = InstTypeMargin

	case exchange.OptionSymbol:
		it = InstTypeOption

	case exchange.SwapSymbol:
		it = InstTypeSwap

	case exchange.SpotSymbol:
		it = InstTypeSpot

	}

	var limit string
	if req.Limit != 0 {
		limit = strconv.Itoa(req.Limit)
	}

	fills, err := rc.Fills(ctx, &FillsReq{
		InstType: it,
		InstID:   req.Symbol.String(),
		After:    req.EndID,
		Before:   req.StartID,
		Limit:    limit,
	})
	if err != nil {
		return nil, err
	}

	ret := []exchange.Trade{}
	for _, f := range fills {
		t, err := f.Parse()
		if err != nil {
			return nil, err
		}

		ret = append(ret, *t)
	}
	return ret, nil
}

func (f *Fill) Parse() (*exchange.Trade, error) {
	var (
		symbol exchange.Symbol
		err    error
	)

	switch f.InstType {
	case InstTypeSwap:
		symbol, err = ParseSwapSymbol(f.InstID)
		if err != nil {
			return nil, err
		}

	case InstTypeMargin:
		symbol, err = ParseMarginSymbol(f.InstID)
		if err != nil {
			return nil, err
		}

	case InstTypeSpot:
		symbol, err = ParseSpotSymbol(f.InstID)
		if err != nil {
			return nil, err
		}

	}

	var (
		fee    decimal.Decimal
		price  decimal.Decimal
		amount decimal.Decimal
		ts     time.Time
	)

	fee, err = decimal.NewFromString(f.Fee)
	if err != nil {
		return nil, errors.WithMessage(err, "invalid fee")
	}
	price, err = decimal.NewFromString(f.FIllPx)
	if err != nil {
		return nil, errors.WithMessage(err, "invalid px")
	}
	amount, err = decimal.NewFromString(f.FillSz)
	if err != nil {
		return nil, errors.WithMessage(err, "invalid sz")
	}
	ts, err = ParseTimestamp(f.Ts)
	if err != nil {
		return nil, errors.WithMessage(err, "invalid ts")
	}

	var side exchange.OrderSide
	if f.Side == OrderSideBuy {
		side = exchange.OrderSideBuy
	} else {
		side = exchange.OrderSideSell
	}

	return &exchange.Trade{
		ID:          f.BillID,
		OrderID:     f.OrdID,
		Symbol:      symbol,
		Amount:      amount,
		Price:       price,
		Fee:         fee,
		FeeCurrency: f.FeeCcy,
		IsMaker:     f.ExecType == ExecTypeMaker,
		Raw:         *f,
		Side:        side,
		Time:        ts,
	}, nil
}

func (rc *RestClient) doPostJSON(ctx context.Context, endPoint string, obj interface{}, dst interface{}) error {
	raw, err := json.Marshal(obj)
	if err != nil {
		return errors.WithMessage(err, "marshal json error")
	}

	body := bytes.NewBuffer(raw)
	if err := rc.Request(ctx, http.MethodPost, endPoint, nil, body, true, dst); err != nil {
		return err
	}
	return nil
}
