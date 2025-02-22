package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"github.com/szmcdull/ccexgo/exchange"
	"github.com/szmcdull/ccexgo/misc/request"
)

type (
	//Binance Rest client instance
	RestClient struct {
		key     string
		secret  string
		apiHost string
	}

	//RestReq basic binance rest request instance add recvWindow param support
	RestReq struct {
		*exchange.RestReq
	}

	GetRestReq interface {
		Values() (url.Values, error)
	}
)

func NewRestClient(key, secret, host string) *RestClient {
	ret := &RestClient{
		key:     key,
		secret:  secret,
		apiHost: host,
	}
	return ret
}

func NewRestReq() *RestReq {
	return &RestReq{
		exchange.NewRestReq(),
	}
}

func (rr *RestReq) RecvWindow(window int) *RestReq {
	rr.AddFields("recvWindow", window)
	return rr
}

func (rc *RestClient) signature(param string) string {
	h := hmac.New(sha256.New, []byte(rc.secret))
	h.Write([]byte(param))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GetRequest helper method to send http GET request
func (rc *RestClient) GetRequest(ctx context.Context, endPoint string, req GetRestReq, sign bool, dst interface{}) error {
	values, err := req.Values()
	if err != nil {
		return errors.WithMessage(err, "build param fail")
	}

	if err := rc.Request(ctx, http.MethodGet, endPoint, values, nil, sign, dst); err != nil {
		return errors.WithMessagef(err, "request %s fail", endPoint)
	}
	return nil
}

func (rc *RestClient) Request(ctx context.Context, method, endPoint string, param url.Values, data io.Reader, signed bool, dst interface{}) error {
	return rc.request(ctx, method, endPoint, param, data, signed, dst)
}

func (rc *RestClient) request(ctx context.Context, method, endPoint string, param url.Values, data io.Reader, signed bool, dst interface{}) error {
	req, err := rc.buildRequest(ctx, method, endPoint, param, data, signed)
	if err != nil {
		return err
	}
	rerr := request.DoReqWithCtx(req, func(resp *http.Response, ierr error) error {
		if ierr != nil {
			return ierr
		}
		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if err := json.Unmarshal(content, dst); err != nil {
			return err
		}

		if api, ok := dst.(APIIF); ok {
			if code := api.ECode(); code != 0 {
				return errors.Errorf("api error code=%d message=%s", api.ECode(), api.EMessage())
			}
		}

		return nil
	})
	return rerr
}

func (rc *RestClient) buildRequest(ctx context.Context, method, endPoint string, values url.Values, data io.Reader, signed bool) (*http.Request, error) {
	if signed {
		if values == nil {
			values = url.Values{}
		}
		values.Add("timestamp", fmt.Sprintf("%d", timeStamp()))
	}
	query := values.Encode()
	if data != nil {
		body, err := ioutil.ReadAll(data)
		if err != nil {
			return nil, errors.WithMessage(err, "read data fail")
		}
		if len(body) != 0 {
			query = fmt.Sprintf("%s&%s", query, string(body))
		}
	}
	if signed {
		sig := rc.signature(query)
		query = fmt.Sprintf("%s&signature=%s", query, sig)
	}

	u := url.URL{Scheme: "https", Path: endPoint, RawQuery: query, Host: rc.apiHost}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, errors.WithMessage(err, "get request fail")
	}
	if len(rc.key) != 0 {
		req.Header.Add("X-MBX-APIKEY", rc.key)
	}
	return req, nil
}

func timeStamp() int64 {
	now := time.Now()
	return now.UnixNano() / 1e6
}
