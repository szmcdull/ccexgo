package spot

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/szmcdull/ccexgo/exchange"
	"github.com/szmcdull/ccexgo/exchange/okex"
	"github.com/szmcdull/ccexgo/internal/rpc"
)

type (
	depth5Raw struct {
		Asks         [][3]decimal.Decimal `json:"asks"`
		Bids         [][3]decimal.Decimal `json:"bids"`
		InstrumentID string               `json:"instrument_id"`
		Timestamp    string               `json:"timestamp"`
	}

	DepthElem struct {
		Price  decimal.Decimal
		Amount decimal.Decimal
		Orders decimal.Decimal
	}

	//Depth5
	Depth5 struct {
		Asks   []DepthElem
		Bids   []DepthElem
		Symbol exchange.SpotSymbol
		Time   time.Time
	}

	Depth5Channel struct {
		sym exchange.SpotSymbol
	}
)

const (
	depth5Table = "spot/depth5"
)

func init() {
	okex.SubscribeCB(depth5Table, parseDepth5)
}

func NewDepth5Channel(sym exchange.SpotSymbol) exchange.Channel {
	return &Depth5Channel{
		sym: sym,
	}
}

func (dc *Depth5Channel) String() string {
	return fmt.Sprintf("%s:%s", depth5Table, dc.sym.String())
}

func parseDepth5(table string, action string, raw json.RawMessage) (*rpc.Notify, error) {
	var ds []depth5Raw
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, err
	}
	if len(ds) != 1 {
		return nil, errors.Errorf("invalid dpth5 len")
	}
	d := ds[0]

	ts, err := okex.ParseTime(d.Timestamp)
	if err != nil {
		return nil, errors.WithMessagef(err, "bad okex timestamp '%s'", d.Timestamp)
	}

	sym, err := ParseSymbol(d.InstrumentID)
	if err != nil {
		return nil, err
	}

	processArr := func(src [][3]decimal.Decimal, dst []DepthElem) error {
		for i, v := range src {

			dst[i] = DepthElem{
				Price:  v[0],
				Amount: v[1],
				Orders: v[2],
			}
		}
		return nil
	}
	asks := make([]DepthElem, len(d.Asks))
	if err := processArr(d.Asks, asks); err != nil {
		return nil, err
	}
	bids := make([]DepthElem, len(d.Asks))
	if err := processArr(d.Bids, bids); err != nil {
		return nil, err
	}

	return &rpc.Notify{
		Method: table,
		Params: &Depth5{
			Bids:   bids,
			Asks:   asks,
			Time:   ts,
			Symbol: sym,
		},
	}, nil
}
