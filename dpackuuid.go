// Package dpuid packs same-sign integer sequences into Delta-Pack UUID byte,
// base64, or UUID values and unpacks them into absolute-value sorted slices.
package dpuid

import (
	"fmt"
	"math"
	"math/bits"
	"reflect"
	"sort"
)

// PackBytes encodes values into an MSB-first DPUID payload. A zero outputBits
// value selects a variable-length byte-aligned payload; a positive value sets
// the exact usable bit limit.
func PackBytes[T Integer](values []T, outputBits int) ([]byte, error) {
	if outputBits < 0 {
		return nil, ErrInvalidBitLimit
	}
	if len(values) == 0 {
		return nil, ErrEmptyInput
	}

	signed, elementBits, err := integerInfo[T]()
	if err != nil {
		return nil, err
	}
	outputPow, err := codecOutputPow(elementBits, outputBits)
	if err != nil {
		return nil, err
	}
	sourceLenBits := outputPow - 1
	deltaLenBits := outputPow - 2
	countBits := outputPow - 1
	maxDeltaBits := (1 << deltaLenBits) - 1
	maxDeltas := (1 << countBits) - 1

	hasPositive, hasNegative := false, false
	items := make([]packedValue, len(values))
	for i, v := range values {
		item, err := newPackedValue(v, signed)
		if err != nil {
			return nil, err
		}
		hasPositive = hasPositive || item.positive
		hasNegative = hasNegative || item.negative
		items[i] = item
	}
	if hasPositive && hasNegative {
		return nil, ErrMixedSigns
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].abs == items[j].abs {
			return items[i].negative && !items[j].negative
		}
		return items[i].abs < items[j].abs
	})

	source := items[0].abs
	sourceWidth := bitWidth(source)
	s := sourceWidth - 1
	if sourceWidth > 1<<sourceLenBits {
		return nil, fmt.Errorf("%w: source needs %d bits, max %d", ErrValueOverflow, sourceWidth, 1<<sourceLenBits)
	}
	isNegative := hasNegative

	deltas := make([]uint64, 0, len(items)-1)
	maxDelta := uint64(0)
	allZero, allOne := true, true
	prev := source
	for _, item := range items[1:] {
		delta := item.abs - prev
		deltas = append(deltas, delta)
		if delta != 0 {
			allZero = false
		}
		if delta != 1 {
			allOne = false
		}
		if delta > maxDelta {
			maxDelta = delta
		}
		prev = item.abs
	}

	d := 0
	variant := 3
	if len(deltas) > 0 && !allZero {
		if allOne {
			d = 1
			variant = 2
		} else {
			d = bitWidth(maxDelta)
			if d < 2 {
				d = 2
			}
			variant = 1
		}
	}

	if d > maxDeltaBits {
		return nil, fmt.Errorf("%w: need %d bits, max %d", ErrDeltaOverflow, d, maxDeltaBits)
	}
	if (outputBits == 0 || variant == 1) && len(deltas) > maxDeltas {
		return nil, fmt.Errorf("%w: have %d deltas, max %d", ErrCountOverflow, len(deltas), maxDeltas)
	}

	headerBits := 3*outputPow - 2 + s
	usedBits := headerBits
	if variant == 1 {
		usedBits += d * len(deltas)
	} else if outputBits > 0 {
		usedBits = 2*outputPow + s - 1
	}
	limit := outputBits
	if limit == 0 {
		limit = bytesForBits(usedBits) * 8
	} else if usedBits > limit {
		return nil, fmt.Errorf("%w: need %d bits, have %d", ErrTotalOverflow, usedBits, limit)
	}

	w := newBitWriter(limit)
	w.writeBool(isNegative)
	w.write(uint64(s), sourceLenBits)
	w.write(source, sourceWidth)
	w.write(uint64(d), deltaLenBits)
	if outputBits == 0 || variant == 1 {
		w.write(uint64(len(deltas)), countBits)
	} else {
		remaining := limit - w.pos
		if !fitsUnsigned(uint64(len(deltas)), remaining) {
			return nil, fmt.Errorf("%w: have %d deltas, field width %d", ErrCountOverflow, len(deltas), remaining)
		}
		w.write(uint64(len(deltas)), remaining)
	}
	if variant == 1 {
		for _, delta := range deltas {
			w.write(delta, d)
		}
	}
	return w.bytes, nil
}

// UnpackBytes decodes an MSB-first DPUID payload. A zero inputBits value selects
// the variable-length byte-aligned layout; a positive value is the exact usable
// bit limit and must match the supplied buffer size.
func UnpackBytes[T Integer](data []byte, inputBits int) ([]T, error) {
	if inputBits < 0 {
		return nil, ErrInvalidBitLimit
	}
	_, elementBits, err := integerInfo[T]()
	if err != nil {
		return nil, err
	}
	decoded, err := unpackNumbers(data, inputBits, elementBits)
	if err != nil {
		return nil, err
	}

	out := make([]T, len(decoded))
	for i, value := range decoded {
		out[i], err = castDecodedValue[T](value)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func unpackNumbers(data []byte, inputBits, elementBits int) ([]decodedValue, error) {
	if inputBits > 0 && len(data) != bytesForBits(inputBits) {
		return nil, fmt.Errorf("%w: got %d bytes for %d bits", ErrInvalidBitLimit, len(data), inputBits)
	}
	if inputBits == 0 && len(data) == 0 {
		return nil, ErrPayloadOverflow
	}
	limit := inputBits
	if limit == 0 {
		limit = len(data) * 8
	}
	if !zeroBits(data, limit, len(data)*8) {
		return nil, fmt.Errorf("%w: non-zero bits outside bit limit", ErrPayloadOverflow)
	}

	outputPow, err := codecOutputPow(elementBits, inputBits)
	if err != nil {
		return nil, err
	}
	sourceLenBits := outputPow - 1
	deltaLenBits := outputPow - 2
	countBits := outputPow - 1
	maxDeltaBits := (1 << deltaLenBits) - 1

	r := newBitReader(data, limit)
	isNegative, err := r.readBool()
	if err != nil {
		return nil, err
	}
	s, err := r.read(sourceLenBits)
	if err != nil {
		return nil, err
	}
	sourceWidth := int(s) + 1
	minimumTail := deltaLenBits
	if inputBits == 0 {
		minimumTail += countBits
	}
	if sourceWidth > limit-r.pos-minimumTail {
		return nil, fmt.Errorf("%w: source width %d", ErrPayloadOverflow, sourceWidth)
	}
	source, err := r.read(sourceWidth)
	if err != nil {
		return nil, err
	}
	d, err := r.read(deltaLenBits)
	if err != nil {
		return nil, err
	}
	if d > uint64(maxDeltaBits) {
		return nil, fmt.Errorf("%w: D=%d", ErrPayloadOverflow, d)
	}

	var deltas []uint64
	switch d {
	case 0, 1:
		width := limit - r.pos
		if inputBits == 0 {
			width = countBits
		}
		count, err := r.readCount(width)
		if err != nil {
			return nil, err
		}
		deltas = make([]uint64, int(count))
		for i := range deltas {
			deltas[i] = d
		}
	default:
		count, err := r.read(countBits)
		if err != nil {
			return nil, err
		}
		used := 3*outputPow - 2 + int(s) + int(d)*int(count)
		if used > limit {
			return nil, fmt.Errorf("%w: need %d bits, have %d", ErrPayloadOverflow, used, limit)
		}
		deltas = make([]uint64, int(count))
		for i := range deltas {
			deltas[i], err = r.read(int(d))
			if err != nil {
				return nil, err
			}
		}
	}
	if inputBits == 0 {
		padding := limit - r.pos
		if padding >= 8 || !zeroBits(data, r.pos, limit) {
			return nil, fmt.Errorf("%w: invalid byte padding", ErrPayloadOverflow)
		}
	} else if d >= 2 && !zeroBits(data, r.pos, limit) {
		return nil, fmt.Errorf("%w: non-zero payload padding", ErrPayloadOverflow)
	}

	out := make([]decodedValue, 0, len(deltas)+1)
	cur := source
	out = append(out, decodedValue{abs: cur, negative: isNegative})
	for _, delta := range deltas {
		if cur > math.MaxUint64-delta {
			return nil, ErrValueOverflow
		}
		cur += delta
		out = append(out, decodedValue{abs: cur, negative: isNegative})
	}

	return out, nil
}

type packedValue struct {
	abs      uint64
	negative bool
	positive bool
}

type decodedValue struct {
	abs      uint64
	negative bool
}

func modeDataBits(mode Mode) (int, error) {
	switch mode {
	case ModeUUIDv8:
		return uuidv8Bits, nil
	case ModeRaw:
		return rawBits, nil
	default:
		return 0, ErrInvalidMode
	}
}

func codecOutputPow(elementBits, bitLimit int) (int, error) {
	outputPow := bits.Len(uint(elementBits*2)) - 1
	if bitLimit > 0 {
		capPow := bits.Len(uint(bitLimit - 1))
		if capPow < outputPow {
			outputPow = capPow
		}
	}
	if outputPow < 3 {
		return 0, fmt.Errorf("%w: %d bits are too small", ErrInvalidBitLimit, bitLimit)
	}
	return outputPow, nil
}

func bytesForBits(bitCount int) int {
	return (bitCount + 7) / 8
}

func fitsUnsigned(value uint64, width int) bool {
	return width >= 64 || width >= 0 && value < uint64(1)<<width
}

func zeroBits(data []byte, from, to int) bool {
	for pos := from; pos < to; pos++ {
		if getBit(data, pos) {
			return false
		}
	}
	return true
}

func newPackedValue[T Integer](v T, signed bool) (packedValue, error) {
	rv := reflect.ValueOf(v)
	if signed {
		n := rv.Int()
		if n < 0 {
			return packedValue{abs: absInt64(n), negative: true}, nil
		}
		return packedValue{abs: uint64(n), positive: n > 0}, nil
	}

	n := rv.Uint()
	return packedValue{abs: n, positive: n > 0}, nil
}

func absInt64(v int64) uint64 {
	return uint64(-(v + 1)) + 1
}

func bitWidth(v uint64) int {
	if v == 0 {
		return 1
	}
	return bits.Len64(v)
}

func integerInfo[T Integer]() (signed bool, width int, err error) {
	var zero T
	t := reflect.TypeOf(zero)
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return true, t.Bits(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return false, t.Bits(), nil
	default:
		return false, 0, fmt.Errorf("%w: %s", ErrValueOverflow, t)
	}
}

func castDecodedValue[T Integer](value decodedValue) (T, error) {
	signed, width, err := integerInfo[T]()
	if err != nil {
		var zero T
		return zero, err
	}

	var zero T
	rv := reflect.New(reflect.TypeOf(zero)).Elem()
	if signed {
		maxPositive := uint64(1)<<(width-1) - 1
		maxNegativeAbs := uint64(1) << (width - 1)
		if value.negative {
			if value.abs > maxNegativeAbs {
				return zero, fmt.Errorf("%w: -%d does not fit", ErrValueOverflow, value.abs)
			}
			if value.abs == maxNegativeAbs {
				rv.SetInt(-1 << (width - 1))
				return rv.Interface().(T), nil
			}
			rv.SetInt(-int64(value.abs))
			return rv.Interface().(T), nil
		}
		if value.abs > maxPositive {
			return zero, fmt.Errorf("%w: %d does not fit", ErrValueOverflow, value.abs)
		}
		rv.SetInt(int64(value.abs))
		return rv.Interface().(T), nil
	}

	if value.negative && value.abs != 0 {
		return zero, fmt.Errorf("%w: negative value does not fit unsigned type", ErrValueOverflow)
	}
	if width < 64 && value.abs > uint64(1)<<width-1 {
		return zero, fmt.Errorf("%w: %d does not fit", ErrValueOverflow, value.abs)
	}
	rv.SetUint(value.abs)
	return rv.Interface().(T), nil
}

func getBit(bytes []byte, pos int) bool {
	return (bytes[pos/8] & (1 << (7 - pos%8))) != 0
}

func setBit(bytes *[]byte, pos int, value bool) {
	mask := byte(1 << (7 - pos%8))
	if value {
		(*bytes)[pos/8] |= mask
		return
	}
	(*bytes)[pos/8] &^= mask
}
