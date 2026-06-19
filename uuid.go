package dpuid

import "github.com/google/uuid"

// Pack encodes values in UUIDv8 mode.
func Pack(values []int64) (uuid.UUID, error) {
	return PackValues(values)
}

// PackMode encodes values according to SPEC.md.
//
// The input order is not preserved. Decoding returns values sorted ascending by
// absolute value. Values must be sign-homogeneous: all non-negative or all
// non-positive.
func PackMode(values []int64, mode Mode) (uuid.UUID, error) {
	return PackValuesMode(values, mode)
}

// PackValues encodes signed or unsigned integer values in UUIDv8 mode.
func PackValues[T Integer](values []T) (uuid.UUID, error) {
	return PackValuesMode(values, ModeUUIDv8)
}

// PackValuesMode encodes signed or unsigned integer values according to SPEC.md.
func PackValuesMode[T Integer](values []T, mode Mode) (uuid.UUID, error) {
	dataBits, err := modeDataBits(mode)
	if err != nil {
		return uuid.Nil, err
	}
	data, err := PackBytes(values, dataBits)
	if err != nil {
		return uuid.Nil, err
	}
	var packed [16]byte
	copy(packed[:], data)
	if mode == ModeUUIDv8 {
		return insertUUIDv8(packed), nil
	}
	return uuid.UUID(packed), nil
}

// Unpack decodes a UUIDv8 DPUID value.
func Unpack(id uuid.UUID) ([]int64, error) {
	return UnpackMode(id, ModeUUIDv8)
}

// UnpackMode decodes a DPUID value encoded in the selected mode.
func UnpackMode(id uuid.UUID, mode Mode) ([]int64, error) {
	return UnpackValuesMode[int64](id, mode)
}

// UnpackValues decodes a UUIDv8 DPUID value into the requested integer type.
func UnpackValues[T Integer](id uuid.UUID) ([]T, error) {
	return UnpackValuesMode[T](id, ModeUUIDv8)
}

// UnpackValuesMode decodes a DPUID value encoded in the selected mode into the
// requested integer type.
func UnpackValuesMode[T Integer](id uuid.UUID, mode Mode) ([]T, error) {
	dataBits, err := modeDataBits(mode)
	if err != nil {
		return nil, err
	}
	data := [16]byte(id)
	if mode == ModeUUIDv8 {
		data, err = extractUUIDv8(id)
		if err != nil {
			return nil, err
		}
	}
	return UnpackBytes[T](data[:], dataBits)
}

func insertUUIDv8(data [16]byte) uuid.UUID {
	var out [16]byte
	dataBytes := data[:]
	outBytes := out[:]
	for i := 0; i < 48; i++ {
		setBit(&outBytes, i, getBit(dataBytes, i))
	}
	for i := 0; i < 12; i++ {
		setBit(&outBytes, 52+i, getBit(dataBytes, 48+i))
	}
	for i := 0; i < 62; i++ {
		setBit(&outBytes, 66+i, getBit(dataBytes, 60+i))
	}

	out[6] = (out[6] & 0x0f) | 0x80
	out[8] = (out[8] & 0x3f) | 0x80
	return uuid.UUID(out)
}

func extractUUIDv8(id uuid.UUID) ([16]byte, error) {
	if id.Version() != 8 || id.Variant() != uuid.RFC4122 {
		return [16]byte{}, ErrInvalidUUIDv8
	}

	src := [16]byte(id)
	var data [16]byte
	srcBytes := src[:]
	dataBytes := data[:]
	for i := 0; i < 48; i++ {
		setBit(&dataBytes, i, getBit(srcBytes, i))
	}
	for i := 0; i < 12; i++ {
		setBit(&dataBytes, 48+i, getBit(srcBytes, 52+i))
	}
	for i := 0; i < 62; i++ {
		setBit(&dataBytes, 60+i, getBit(srcBytes, 66+i))
	}
	return data, nil
}
