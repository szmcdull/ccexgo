package future

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/szmcdull/ccexgo/exchange"
	"github.com/szmcdull/ccexgo/exchange/okex"
)

const (
	apiRawSymbolURI = "/api/futures/v3/instruments"
)

type (
	OkexSymbol struct {
		InstrumentID        string          `json:"instrument_id"`
		UnderlyingIndex     string          `json:"underlying_index"`
		QuoteCurrency       string          `json:"quote_currency"`
		TickSize            decimal.Decimal `json:"tick_size"`
		ContractVal         decimal.Decimal `json:"contract_val"`
		Listing             string          `json:"listing"`
		Delivery            string          `json:"delivery"`
		TradeIncrement      decimal.Decimal `json:"trade_increment"`
		Alias               string          `json:"alias"`
		Underlying          string          `json:"underlying"`
		BaseCurrency        string          `json:"base_currency"`
		SettlementCurrency  string          `json:"settlement_currency"`
		IsInverse           string          `json:"is_inverse"`
		ContractValCurrency string          `json:"contract_val_currency"`
		Category            string          `json:"category"`
	}

	Symbol struct {
		*exchange.BaseFutureSymbol
	}
)

var (
	alias2Type map[string]exchange.FutureType = map[string]exchange.FutureType{
		"this_week":  exchange.FutureTypeCW,
		"next_week":  exchange.FutureTypeNW,
		"quarter":    exchange.FutureTypeCQ,
		"bi_quarter": exchange.FutureTypeNQ,
	}

	symbolMapMu                    = sync.Mutex{}
	symbolMap   map[string]*Symbol = map[string]*Symbol{}
)

func (rc *RestClient) Symbols(ctx context.Context) ([]exchange.FuturesSymbol, error) {
	symbols, err := rc.RawSymbols(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]exchange.FuturesSymbol, len(symbols))
	for i, raw := range symbols {
		s, err := raw.Parse()
		if err != nil {
			return nil, err
		}
		ret[i] = s
	}
	return ret, nil
}

// RawSymbols return okex future symbols
func (rc *RestClient) RawSymbols(ctx context.Context) ([]OkexSymbol, error) {
	var symbols []OkexSymbol
	if err := rc.Request(ctx, http.MethodGet, apiRawSymbolURI, nil, nil, false, &symbols); err != nil {
		return nil, err
	}
	return symbols, nil
}

func (os *OkexSymbol) Parse() (*Symbol, error) {
	typ, ok := alias2Type[os.Alias]
	if !ok {
		return nil, errors.Errorf("unkown alias='%s'", os.Alias)
	}

	dt := fmt.Sprintf("%sT08:00:00.000Z", os.Delivery)
	st, err := okex.ParseTime(dt)
	if err != nil {
		return nil, err
	}

	return &Symbol{
		exchange.NewBaseFutureSymbol(os.Underlying, st, typ),
	}, nil
}

func (s *Symbol) String() string {
	st := s.SettleTime()
	return fmt.Sprintf("%s-%s", s.Index(), st.Format("060102"))
}

// Init start a background goroutine which used to update symbol map
func Init(ctx context.Context) error {
	if err := updateSymbolMap(ctx); err != nil {
		return err
	}

	timer := time.NewTicker(time.Minute)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return

			case <-timer.C:
				updateSymbolMap(ctx)
			}
		}
	}()
	return nil
}

func ParseSymbol(symbol string) (exchange.FuturesSymbol, error) {
	symbolMapMu.Lock()
	defer symbolMapMu.Unlock()
	sym, ok := symbolMap[symbol]
	if !ok {
		return nil, errors.Errorf("unkown symbol '%s'", symbol)
	}

	return sym, nil
}

func FetchSymbolByIndex(index string) []exchange.FuturesSymbol {
	var ret []exchange.FuturesSymbol
	symbolMapMu.Lock()
	defer symbolMapMu.Unlock()

	for _, s := range symbolMap {
		if s.Index() == index {
			ret = append(ret, s)
		}
	}
	return ret
}

func updateSymbolMap(ctx context.Context) error {
	client := NewRestClient("", "", "")
	symbols, err := client.Symbols(ctx)
	if err != nil {
		return err
	}

	sort.Slice(symbols, func(i, j int) bool {
		ti := symbols[i].SettleTime()
		tj := symbols[j].SettleTime()
		return ti.Before(tj)
	})

	m := map[string]*Symbol{}
	for _, s := range symbols {
		m[s.String()] = s.(*Symbol)
	}
	symbolMapMu.Lock()
	defer symbolMapMu.Unlock()
	symbolMap = m

	return nil
}
