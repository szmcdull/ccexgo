package swap

import (
	"time"

	"github.com/szmcdull/ccexgo/exchange"
)

func (rc *RestClient) Property() exchange.Property {
	return exchange.Property{
		Trades: &exchange.TradesProp{
			MaxDuration: time.Hour * 168,
			SuportID:    false,
			SupportTime: true,
		},
		Finance: &exchange.FinanceProp{
			MaxDuration: time.Hour * 168,
			SuportID:    false,
			SupportTime: true,
		},
	}
}
