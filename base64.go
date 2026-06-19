package dpuid

import (
	"encoding/base64"
	"fmt"
)

// PackBase64 encodes values into a variable-length RFC 4648 base64 string.
func PackBase64[T Integer](values []T) (string, error) {
	return PackBase64Bits(values, 0)
}

// PackBase64Bits encodes values into base64 using the requested payload size.
// A zero outputBits value selects the minimum byte-aligned payload size.
func PackBase64Bits[T Integer](values []T, outputBits int) (string, error) {
	data, err := PackBytes(values, outputBits)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// UnpackBase64 decodes a variable-length RFC 4648 base64 DPUID payload.
func UnpackBase64[T Integer](value string) ([]T, error) {
	return UnpackBase64Bits[T](value, 0)
}

// UnpackBase64Bits decodes a base64 DPUID payload with the requested bit size.
// A zero inputBits value selects the variable-length byte-aligned layout.
func UnpackBase64Bits[T Integer](value string, inputBits int) ([]T, error) {
	data, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBase64, err)
	}
	return UnpackBytes[T](data, inputBits)
}
