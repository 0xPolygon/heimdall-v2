// Package rest provides HTTP types and primitives for REST
// requests validation and responses handling.
package rest

import (
	"encoding/json"
)

const (
	DefaultPage  = 1
	DefaultLimit = 30 // should be consistent with tendermint/tendermint/rpc/core/pipe.go:19
)

// ResponseWithHeight defines a response object type that wraps an original
// response with a height.
type ResponseWithHeight struct {
	Height int64           `json:"height"`
	Result json.RawMessage `json:"result"`
}

// NewResponseWithHeight creates a new ResponseWithHeight instance
func NewResponseWithHeight(height int64, result json.RawMessage) ResponseWithHeight {
	return ResponseWithHeight{
		Height: height,
		Result: result,
	}
}
