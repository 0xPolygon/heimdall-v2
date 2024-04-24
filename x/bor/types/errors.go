package types

import "errors"

var (
	ErrInvalidChainID = errors.New("invalid bor chain id")
	ErrInvalidSpan    = errors.New("invalid span")
)
