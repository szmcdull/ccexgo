package deribit

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/szmcdull/ccexgo/internal/rpc"
)

//deribit message json serizlize

const (
	JsonRPCVersion     = "2.0"
	subscriptionMethod = "subscription"
)

type (
	Notify struct {
		Data    json.RawMessage `json:"data"`
		Channel string          `json:"channel"`
	}

	Error struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}
	Response struct {
		JsonRPC string          `json:"jsonrpc"`
		ID      int64           `json:"id"`
		Error   Error           `json:"error"`
		Result  json.RawMessage `json:"result"`
		Method  string          `json:"method"`
		Params  Notify          `json:"params"`
	}

	Request struct {
		ID      int64       `json:"id"`
		Method  string      `json:"method"`
		JsonRPC string      `json:"jsonrpc"`
		Params  interface{} `json:"params"`
	}

	Codec struct {
	}

	notifyParseCB func(*Notify) (*rpc.Notify, error)
)

var (
	notifyParseMap map[string]notifyParseCB = make(map[string]notifyParseCB)
)

func reigisterCB(key string, cb notifyParseCB) {
	_, ok := notifyParseMap[key]
	if ok {
		panic(fmt.Sprintf("duplicate cb %s register", key))
	}
	notifyParseMap[key] = cb
}

func (cc *Codec) Decode(raw []byte) (rpc.Response, error) {
	var resp Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}

	if resp.Method == subscriptionMethod {
		resp, err := parseNotify(&resp)
		if err != nil {
			return nil, errors.WithMessage(err, "parse response error")
		}
		return resp, nil
	}

	var err error
	if resp.Error.Code != 0 {
		err = NewError(resp.Error.Code, resp.Error.Message)
	} else {
		err = nil
	}

	return &rpc.Result{
		ID:     strconv.FormatInt(resp.ID, 10),
		Error:  err,
		Result: resp.Result,
	}, nil
}

func (cc *Codec) Encode(req rpc.Request) ([]byte, error) {
	id, err := strconv.ParseInt(req.ID(), 10, 64)
	if err != nil {
		return nil, errors.WithMessagef(err, "bad req id '%s'", req.ID())
	}
	r := Request{
		ID:      id,
		Method:  req.Method(),
		Params:  req.Params(),
		JsonRPC: JsonRPCVersion,
	}

	return json.Marshal(&r)
}

func parseNotify(resp *Response) (rpc.Response, error) {
	fields := strings.Split(resp.Params.Channel, ".")
	if len(fields) == 0 {
		return nil, errors.Errorf("bad resp channel %v", resp.Params)
	}

	cb, ok := notifyParseMap[fields[0]]
	if !ok {
		return nil, errors.Errorf("unsupport channel %s", resp.Params.Channel)
	}
	r, err := cb(&resp.Params)
	if err != nil {
		return nil, errors.WithMessagef(err, "parse notify channel=%s, data=%s",
			resp.Params.Channel, string(resp.Params.Data))
	}
	return r, nil
}
