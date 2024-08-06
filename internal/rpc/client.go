package rpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
)

// Client implements remote calls to http server
type Client struct {
	HTTP HTTP
	URL  string

	id uint64
}

func (c Client) http() HTTP {
	if c.HTTP == nil {
		return http.DefaultClient
	}
	return c.HTTP
}

// Call remote server with given method and arguments
func (c Client) Call(method string, args ...interface{}) (*Response, error) {
	var b []byte
	var err error

	switch {
	case len(args) == 0:
		b, err = json.Marshal(Request{Version: "2.0", Method: method, ID: atomic.AddUint64(&c.id, 1)})
	case len(args) == 1:
		b, err = json.Marshal(Request{Version: "2.0", Method: method, Params: args[0], ID: atomic.AddUint64(&c.id, 1)})
	default:
		b, err = json.Marshal(Request{Version: "2.0", Method: method, Params: args, ID: atomic.AddUint64(&c.id, 1)})
	}
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.URL, bytes.NewBuffer(b))
	if err != nil {
		return nil, errors.Join(errors.New("rpc, request creation:"), err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.http().Do(req)
	if err != nil {
		return nil, errors.Join(errors.New("rpc, request execution:"), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status %s for %s", resp.Status, method)
	}

	r := Response{}
	if err = json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, errors.Join(errors.New("rpc, response json unmarshaling:"), err)
	}
	if r.Error != nil {
		return nil, fmt.Errorf("%s", r.Error)
	}
	return &r, nil
}
