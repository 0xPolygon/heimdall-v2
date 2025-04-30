package types

import "errors"

var (
	ErrInvalidChainID            = errors.New("invalid bor chain id")
	ErrInvalidSpan               = errors.New("invalid span")
	ErrInvalidLastHeimdallSpanID = errors.New("invalid last heimdall span id")
	ErrInvalidLastBorSpanID      = errors.New("invalid last bor span id")
	ErrInvalidLastBorBlock       = errors.New("invalid last bor block")
	ErrInvalidSeedLength         = errors.New("invalid seed length")
	ErrFailedToQueryBor          = errors.New("failed to query bor")
)
