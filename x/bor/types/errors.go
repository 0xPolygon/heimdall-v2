package types

import "errors"

var (
	ErrInvalidChainID    = errors.New("invalid bor chain id")
	ErrInvalidSpan       = errors.New("invalid span")
	ErrInvalidSeedLength = errors.New("invalid seed length")
	ErrFailedToQueryBor  = errors.New("failed to query bor")
)
