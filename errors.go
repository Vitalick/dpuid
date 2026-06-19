package dpuid

import "errors"

var (
	ErrEmptyInput      = errors.New("dpuid: empty input")
	ErrMixedSigns      = errors.New("dpuid: mixed signs")
	ErrDeltaOverflow   = errors.New("dpuid: max delta too large")
	ErrCountOverflow   = errors.New("dpuid: too many numbers")
	ErrTotalOverflow   = errors.New("dpuid: encoded payload exceeds output size")
	ErrInvalidMode     = errors.New("dpuid: invalid mode")
	ErrInvalidUUIDv8   = errors.New("dpuid: invalid UUIDv8 markers")
	ErrInvalidBase64   = errors.New("dpuid: invalid base64")
	ErrInvalidBitLimit = errors.New("dpuid: invalid bit limit")
	ErrPayloadOverflow = errors.New("dpuid: payload exceeds input size")
	ErrValueOverflow   = errors.New("dpuid: value overflows target integer type")
)
