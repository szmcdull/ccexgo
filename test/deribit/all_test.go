package deribit

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/szmcdull/ccexgo/exchange"
	"github.com/szmcdull/ccexgo/exchange/deribit"
)

func TestAll(t *testing.T) {
	baseCtx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	ch := make(chan interface{}, 4)
	defer cancel()

	if err := deribit.Init(baseCtx, true); err != nil {
		t.Fatalf("init fail error=%s", err.Error())
	}

	var (
		mu    sync.Mutex
		index exchange.IndexNotify
		book  exchange.OrderBookNotify
	)
	go func() {
		for {
			select {
			case <-baseCtx.Done():
				return

			case n := <-ch:
				notify := n.(*exchange.WSNotify)
				switch t := notify.Data.(type) {
				case *exchange.IndexNotify:
					mu.Lock()
					index = *t
					mu.Unlock()

				case *exchange.OrderBookNotify:
					mu.Lock()
					book = *t
					mu.Unlock()
				}
			}
		}
	}()

	key := os.Getenv("D_KEY")
	secret := os.Getenv("D_SECRET")
	if key == "" || secret == "" {
		t.Fatalf("missing env D_KEY D_SECRET")
	}

	client := deribit.NewTestWSClient(key, secret, ch)
	if err := client.Run(baseCtx); err != nil {
		t.Fatalf("running the loop fail %s", err.Error())
	}

	tc := deribit.NewIndexChannel("btc_usd")
	if err := client.Subscribe(baseCtx, tc); err != nil {
		t.Fatalf("subscribe index fail %s", err.Error())
	}
	instruments, err := client.OptionFetchInstruments(baseCtx, "BTC", false)
	if err != nil {
		t.Fatalf("load instrument error %s", err.Error())
	}

	var sym exchange.OptionSymbol
	mu.Lock()
	indexPrice := index.Price
	mu.Unlock()
	fmt.Printf("got index price %s\n", indexPrice)
	for _, i := range instruments {
		if i.SettlementPeriod != "day" {
			continue
		}

		if i.Strike.GreaterThan(indexPrice) {
			sym, _ = deribit.ParseOptionSymbol(i.InstrumentName)
			fmt.Printf("GOT SYMBOL %v %v\n", sym, i)
			break
		}
	}

	oc := deribit.NewOrderBookChannel(sym)
	if err := client.Subscribe(baseCtx, oc); err != nil {
		t.Fatalf("subscribe orderbook fail %s", err.Error())
	}

	//wait goroutine handle orderbook update
	time.Sleep(2 * time.Second)

	mu.Lock()
	fmt.Printf("got orderbook %+v\n", book)
	var price float64
	if len(book.Asks) != 0 {
		price = book.Asks[0].Price - 0.0005
		if price < 0.0 {
			price = -1.0 * price
		}
	} else {
		price = 0.0005
	}
	mu.Unlock()

	fmt.Printf("ORDER %s %f\n", sym.String(), price)
	req := exchange.OrderRequest{
		Symbol: sym,
		Price:  decimal.NewFromFloat(price),
		Amount: decimal.NewFromFloat(0.1),
		Type:   exchange.OrderTypeLimit,
		Side:   exchange.OrderSideBuy,
	}
	//create a order with price will not being executed
	order, err := client.CreateOrder(baseCtx, &req)
	if err != nil {
		t.Fatalf("create order fail %v", err.Error())
	}
	if order.Status != exchange.OrderStatusOpen || !order.Created.Equal(order.Updated) || order.Symbol.String() != sym.String() {
		t.Errorf("bad order status %v", *order)
	}

	if _, err := client.CancelOrder(baseCtx, order); err != nil {
		t.Errorf("cancel order fail %s", err.Error())
	}

	if order, err := client.FetchOrder(baseCtx, order); err != nil {
		t.Errorf("fetch order fail %s", err.Error())
	} else {
		if order.Status != exchange.OrderStatusCancel {
			t.Errorf("test cancel fail %v", *order)
		}
	}

	//test creat a fok order
	if order, err = client.CreateOrder(baseCtx, &req,
		exchange.NewTimeInForceOption(exchange.TimeInForceFOK),
		exchange.NewPostOnlyOption(false),
	); err != nil {
		t.Errorf("test create fok order fail %s", err.Error())
	} else if order.Status != exchange.OrderStatusCancel {
		t.Errorf("fok order executed %v", *order)
	}

	if err := client.UnSubscribe(baseCtx, tc); err != nil {
		t.Errorf("unsubscribe fail %s", err.Error())
	}

	if err := client.UnSubscribe(baseCtx, oc); err != nil {
		t.Errorf("unsubscribe fail %s", err.Error())
	}

	var balance deribit.AccountSummaryResp
	if err := client.Call(baseCtx, deribit.AccountSummaryMethod, &deribit.AccountSummaryReq{
		Currency: "BTC",
	}, &balance, true); err != nil {
		t.Fatalf("get account summary fail error=%s", err.Error())
	}

	if !balance.MarginBalance.Sub(balance.InitialMargin).Equal(balance.AvailableFunds) {
		t.Errorf("margin not equal margin=%s inital=%s balance=%s", balance.MarginBalance, balance.InitialMargin, balance.AvailableFunds)
	}

	var settlement deribit.SettlementResp
	if err := client.Call(baseCtx, deribit.SettlementMethodByInstrument, &deribit.SettlementReq{
		InstrumentName: "BTC-5JAN21-32750-P",
		Type:           deribit.SettlementTypeDelivery,
	}, &settlement, true); err != nil {
		t.Fatalf("get settlement fail error=%s", err.Error())
	}

	fmt.Printf("%+v\n", settlement)

	if err := client.Call(baseCtx, deribit.PublicSettlementMethodByInstrument, &deribit.PublicSettlementByInstrumentReq{
		InstrumentName: "BTC-20JAN21-36000-P",
		Type:           deribit.SettlementTypeDelivery,
	}, &settlement, false); err != nil {
		t.Fatalf("get public settlement fail error=%s", err.Error())
	}
	fmt.Printf("%+v\n", settlement)
}
