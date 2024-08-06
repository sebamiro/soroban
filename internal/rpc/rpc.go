package rpc

import (
	"encoding/json"
	"net/http"
)

type HTTP interface {
	Do(req *http.Request) (*http.Response, error)
}

type Request struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      uint64      `json:"id"`
}

type Response struct {
	Version string           `json:"jsonrpc"`
	ID      uint64           `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   any           `json:"error,omitempty"`
}

