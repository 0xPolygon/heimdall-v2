package types

import "errors"

var (
	ErrInvalidChainID          = errors.New("invalid bor chain id")
	ErrInvalidSpan             = errors.New("invalid span")
	ErrInvalidLastBorSpanID    = errors.New("invalid last bor span id")
	ErrInvalidSeedLength       = errors.New("invalid seed length")
	ErrFailedToQueryBor        = errors.New("failed to query bor")
	ErrLatestMilestoneNotFound = errors.New("latest milestone not found")
)
