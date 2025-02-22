package exchange

import (
	"fmt"
	"reflect"
	"time"

	"github.com/emirpasic/gods/trees/btree"
	"github.com/emirpasic/gods/utils"
	"github.com/pkg/errors"
	"github.com/szmcdull/ccexgo/misc/float"
)

type (
	OrderElem struct {
		Price  float64
		Amount float64
	}

	//OrderBookNotify change of current orderbook
	//OrderElem.Amount == 0 means delete
	OrderBookNotify struct {
		Symbol Symbol
		Bids   []OrderElem
		Asks   []OrderElem
		Raw    interface{}
	}

	//OrderBookDS is the ds which hold orderbook info
	OrderBookDS struct {
		symbol  Symbol
		bids    *btree.Tree
		asks    *btree.Tree
		updated time.Time
	}

	//OrderBook get via OrderBookDS.Snapshot
	OrderBook struct {
		Symbol  Symbol
		Bids    []OrderElem
		Asks    []OrderElem
		Created time.Time
		Raw     interface{}
	}
)

func init() {
	typ := reflect.TypeOf(&OrderBookNotify{})
	subRegister(typ, orderbookHandler)
}

func (c *Client) OrderBook(symbol Symbol) (*OrderBook, error) {
	c.SubMu.Lock()
	defer c.SubMu.Unlock()
	key := orderBookKey(symbol)
	ins, ok := c.Sub[key]
	if !ok {
		return nil, errors.Errorf("unkown symbol %s", symbol.String())
	}
	ds := ins.(*OrderBookDS)
	return ds.Snapshot(), nil
}

func (notify *OrderBookNotify) Key() string {
	return orderBookKey(notify.Symbol)
}

func NewOrderBookDS(notify *OrderBookNotify) *OrderBookDS {
	return &OrderBookDS{
		symbol:  notify.Symbol,
		bids:    newBook(notify.Bids),
		asks:    newBook(notify.Asks),
		updated: time.Now(),
	}
}

func (ds *OrderBookDS) Update(notify *OrderBookNotify) {
	updateTree := func(dest *btree.Tree, src []OrderElem) {
		for _, elem := range src {
			if float.Equal(elem.Price, 0.0) {
				continue
			}

			if float.Equal(elem.Amount, 0.0) {
				if _, ok := dest.Get(elem.Price); !ok {
					continue
				}
				dest.Remove(elem.Price)
			} else {
				dest.Put(elem.Price, elem.Amount)
			}
		}
	}

	updateTree(ds.bids, notify.Bids)
	updateTree(ds.asks, notify.Asks)
	ds.updated = time.Now()
}

func (ds *OrderBookDS) Snapshot() *OrderBook {
	ret := &OrderBook{
		Symbol:  ds.symbol,
		Bids:    make([]OrderElem, ds.bids.Size()),
		Asks:    make([]OrderElem, ds.asks.Size()),
		Created: ds.updated,
	}

	biter := ds.bids.Iterator()
	biter.End()
	i := 0
	for biter.Prev() {
		ret.Bids[i].Price = biter.Key().(float64)
		ret.Bids[i].Amount = biter.Value().(float64)
		i++
	}

	aiter := ds.asks.Iterator()
	aiter.Begin()
	i = 0
	for aiter.Next() {
		ret.Asks[i].Price = aiter.Key().(float64)
		ret.Asks[i].Amount = aiter.Value().(float64)
		i++
	}

	return ret
}

func orderbookHandler(ds interface{}, msg handlerMsg) interface{} {
	notify := msg.(*OrderBookNotify)
	if ds == nil {
		return NewOrderBookDS(notify)
	}

	ob := ds.(*OrderBookDS)
	ob.Update(notify)
	return ob
}

func newBook(data []OrderElem) *btree.Tree {
	l := len(data)
	if l < 3 {
		l = 3
	}
	tree := btree.NewWith(l, utils.Float64Comparator)
	for _, depth := range data {
		if float.Equal(depth.Price, 0.0) {
			continue
		}
		tree.Put(depth.Price, depth.Amount)
	}
	return tree
}

func orderBookKey(symbol Symbol) string {
	return fmt.Sprintf("book.%s", symbol.String())
}
