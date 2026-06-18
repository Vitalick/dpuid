// Package dpackuuid packs same-sign int64 sequences into UUID-compatible
// Delta-Pack UUID values and unpacks them back to absolute-value sorted slices.
package dpackuuid

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
	"sort"

	"github.com/google/uuid"
)

const (
	outputPow  = 7
	rawBits    = 128
	uuidv8Bits = 122

	sourceLenBits = outputPow - 1
	deltaLenBits  = outputPow - 2
	countBits     = outputPow - 1
	maxDeltaBits  = (1 << deltaLenBits) - 1
	maxV1Deltas   = (1 << countBits) - 1
)

// Mode selects how DPUID data is embedded into the returned UUID value.
type Mode int

const (
	// ModeUUIDv8 stores 122 DPUID bits around RFC 9562 UUID version and variant
	// marker bits. This is the recommended default.
	ModeUUIDv8 Mode = iota

	// ModeRaw uses all 128 UUID bits for DPUID data and does not produce an
	// RFC-compliant UUID.
	ModeRaw
)

var (
	ErrEmptyInput      = errors.New("dpackuuid: empty input")
	ErrMixedSigns      = errors.New("dpackuuid: mixed signs")
	ErrDeltaOverflow   = errors.New("dpackuuid: max delta too large")
	ErrCountOverflow   = errors.New("dpackuuid: too many numbers")
	ErrTotalOverflow   = errors.New("dpackuuid: encoded payload exceeds output size")
	ErrInvalidMode     = errors.New("dpackuuid: invalid mode")
	ErrInvalidUUIDv8   = errors.New("dpackuuid: invalid UUIDv8 markers")
	ErrPayloadOverflow = errors.New("dpackuuid: payload exceeds input size")
	ErrValueOverflow   = errors.New("dpackuuid: decoded value overflows int64")
)

// Pack encodes values in UUIDv8 mode.
func Pack(values []int64) (uuid.UUID, error) {
	return PackMode(values, ModeUUIDv8)
}

// PackMode encodes values according to SPEC.md.
//
// The input order is not preserved. Decoding returns values sorted ascending by
// absolute value. Values must be sign-homogeneous: all non-negative or all
// non-positive.
func PackMode(values []int64, mode Mode) (uuid.UUID, error) {
	dataBits, err := modeDataBits(mode)
	if err != nil {
		return uuid.Nil, err
	}
	if len(values) == 0 {
		return uuid.Nil, ErrEmptyInput
	}

	hasPositive, hasNegative := false, false
	items := make([]packedValue, len(values))
	for i, v := range values {
		if v > 0 {
			hasPositive = true
		}
		if v < 0 {
			hasNegative = true
		}
		items[i] = packedValue{value: v, abs: absInt64(v)}
	}
	if hasPositive && hasNegative {
		return uuid.Nil, ErrMixedSigns
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].abs == items[j].abs {
			return items[i].value < items[j].value
		}
		return items[i].abs < items[j].abs
	})

	source := items[0].abs
	sourceWidth := bitWidth(source)
	s := sourceWidth - 1
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
		return uuid.Nil, fmt.Errorf("%w: need %d bits, max %d", ErrDeltaOverflow, d, maxDeltaBits)
	}
	if variant == 1 {
		if len(deltas) > maxV1Deltas {
			return uuid.Nil, fmt.Errorf("%w: have %d deltas, max %d", ErrCountOverflow, len(deltas), maxV1Deltas)
		}
		used := 19 + s + d*len(deltas)
		if used > dataBits {
			return uuid.Nil, fmt.Errorf("%w: need %d bits, have %d", ErrTotalOverflow, used, dataBits)
		}
	}

	w := newBitWriter(dataBits)
	w.writeBool(isNegative)
	w.write(uint64(s), sourceLenBits)
	w.write(source, sourceWidth)
	w.write(uint64(d), deltaLenBits)
	if variant == 1 {
		w.write(uint64(len(deltas)), countBits)
		for _, delta := range deltas {
			w.write(delta, d)
		}
	} else {
		w.write(uint64(len(deltas)), dataBits-w.pos)
	}

	if mode == ModeUUIDv8 {
		return insertUUIDv8(w.bytes), nil
	}
	return uuid.UUID(w.bytes), nil
}

// Unpack decodes a UUIDv8 DPUID value.
func Unpack(id uuid.UUID) ([]int64, error) {
	return UnpackMode(id, ModeUUIDv8)
}

// UnpackMode decodes a DPUID value encoded in the selected mode.
func UnpackMode(id uuid.UUID, mode Mode) ([]int64, error) {
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

	r := newBitReader(data, dataBits)
	isNegative, err := r.readBool()
	if err != nil {
		return nil, err
	}
	s, err := r.read(sourceLenBits)
	if err != nil {
		return nil, err
	}
	sourceWidth := int(s) + 1
	if sourceWidth > dataBits-13 {
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
	if d > maxDeltaBits {
		return nil, fmt.Errorf("%w: D=%d", ErrPayloadOverflow, d)
	}

	var deltas []uint64
	switch d {
	case 0, 1:
		count, err := r.readCount(dataBits - r.pos)
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
		used := 19 + int(s) + int(d)*int(count)
		if used > dataBits {
			return nil, fmt.Errorf("%w: need %d bits, have %d", ErrPayloadOverflow, used, dataBits)
		}
		deltas = make([]uint64, int(count))
		for i := range deltas {
			deltas[i], err = r.read(int(d))
			if err != nil {
				return nil, err
			}
		}
	}

	out := make([]int64, 0, len(deltas)+1)
	cur := source
	first, err := applySign(cur, isNegative)
	if err != nil {
		return nil, err
	}
	out = append(out, first)
	for _, delta := range deltas {
		if cur > math.MaxUint64-delta {
			return nil, ErrValueOverflow
		}
		cur += delta
		next, err := applySign(cur, isNegative)
		if err != nil {
			return nil, err
		}
		out = append(out, next)
	}

	return out, nil
}

type packedValue struct {
	value int64
	abs   uint64
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

func absInt64(v int64) uint64 {
	if v >= 0 {
		return uint64(v)
	}
	return uint64(-(v + 1)) + 1
}

func bitWidth(v uint64) int {
	if v == 0 {
		return 1
	}
	return bits.Len64(v)
}

func applySign(v uint64, negative bool) (int64, error) {
	if negative {
		if v == uint64(1)<<63 {
			return math.MinInt64, nil
		}
		if v > uint64(1)<<63 {
			return 0, ErrValueOverflow
		}
		return -int64(v), nil
	}
	if v > math.MaxInt64 {
		return 0, ErrValueOverflow
	}
	return int64(v), nil
}

type bitWriter struct {
	bytes [16]byte
	pos   int
	limit int
}

func newBitWriter(limit int) *bitWriter {
	return &bitWriter{limit: limit}
}

func (w *bitWriter) writeBool(v bool) {
	if v {
		w.write(1, 1)
		return
	}
	w.write(0, 1)
}

func (w *bitWriter) write(v uint64, width int) {
	for i := width - 1; i >= 0; i-- {
		if ((v >> i) & 1) == 1 {
			setBit(&w.bytes, w.pos, true)
		}
		w.pos++
	}
}

type bitReader struct {
	bytes [16]byte
	pos   int
	limit int
}

func newBitReader(bytes [16]byte, limit int) *bitReader {
	return &bitReader{bytes: bytes, limit: limit}
}

func (r *bitReader) readBool() (bool, error) {
	v, err := r.read(1)
	return v == 1, err
}

func (r *bitReader) read(width int) (uint64, error) {
	if width < 0 || width > 64 || r.pos+width > r.limit {
		return 0, ErrPayloadOverflow
	}
	var out uint64
	for i := 0; i < width; i++ {
		out <<= 1
		if getBit(r.bytes, r.pos) {
			out |= 1
		}
		r.pos++
	}
	return out, nil
}

func (r *bitReader) readCount(width int) (uint64, error) {
	if width < 0 || r.pos+width > r.limit {
		return 0, ErrPayloadOverflow
	}
	var out uint64
	for i := 0; i < width; i++ {
		if out > uint64(math.MaxInt)/2 {
			return 0, fmt.Errorf("%w: count exceeds max int", ErrCountOverflow)
		}
		out <<= 1
		if getBit(r.bytes, r.pos) {
			out |= 1
		}
		r.pos++
	}
	return out, nil
}

func insertUUIDv8(data [16]byte) uuid.UUID {
	var out [16]byte
	for i := 0; i < 48; i++ {
		setBit(&out, i, getBit(data, i))
	}
	for i := 0; i < 12; i++ {
		setBit(&out, 52+i, getBit(data, 48+i))
	}
	for i := 0; i < 62; i++ {
		setBit(&out, 66+i, getBit(data, 60+i))
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
	for i := 0; i < 48; i++ {
		setBit(&data, i, getBit(src, i))
	}
	for i := 0; i < 12; i++ {
		setBit(&data, 48+i, getBit(src, 52+i))
	}
	for i := 0; i < 62; i++ {
		setBit(&data, 60+i, getBit(src, 66+i))
	}
	return data, nil
}

func getBit(bytes [16]byte, pos int) bool {
	return (bytes[pos/8] & (1 << (7 - pos%8))) != 0
}

func setBit(bytes *[16]byte, pos int, value bool) {
	mask := byte(1 << (7 - pos%8))
	if value {
		bytes[pos/8] |= mask
		return
	}
	bytes[pos/8] &^= mask
}
