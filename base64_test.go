package dpuid

import (
	"errors"
	"reflect"
	"testing"
)

func TestBase64Int8Example(t *testing.T) {
	encoded, err := PackBase64([]int8{10, 13, 11, 12})
	if err != nil {
		t.Fatalf("PackBase64() error = %v", err)
	}
	if encoded != "Olg=" {
		t.Fatalf("PackBase64() = %q, want %q", encoded, "Olg=")
	}

	got, err := UnpackBase64[int8](encoded)
	if err != nil {
		t.Fatalf("UnpackBase64() error = %v", err)
	}
	if want := []int8{10, 11, 12, 13}; !reflect.DeepEqual(got, want) {
		t.Fatalf("UnpackBase64() = %#v, want %#v", got, want)
	}
}

func TestBase64BitsRoundTrip(t *testing.T) {
	encoded, err := PackBase64Bits([]uint32{100, 104, 108}, 96)
	if err != nil {
		t.Fatalf("PackBase64Bits() error = %v", err)
	}
	got, err := UnpackBase64Bits[uint32](encoded, 96)
	if err != nil {
		t.Fatalf("UnpackBase64Bits() error = %v", err)
	}
	if want := []uint32{100, 104, 108}; !reflect.DeepEqual(got, want) {
		t.Fatalf("UnpackBase64Bits() = %#v, want %#v", got, want)
	}
}

func TestByteAndBase64Validation(t *testing.T) {
	if _, err := PackBytes([]int64{1}, -1); !errors.Is(err, ErrInvalidBitLimit) {
		t.Fatalf("PackBytes() error = %v, want %v", err, ErrInvalidBitLimit)
	}
	if _, err := UnpackBase64[int64]("not base64!"); !errors.Is(err, ErrInvalidBase64) {
		t.Fatalf("UnpackBase64() error = %v, want %v", err, ErrInvalidBase64)
	}

	data, err := PackBytes([]int8{10, 13, 11, 12}, 0)
	if err != nil {
		t.Fatalf("PackBytes() error = %v", err)
	}
	data[len(data)-1] |= 1
	if _, err = UnpackBytes[int8](data, 0); !errors.Is(err, ErrPayloadOverflow) {
		t.Fatalf("UnpackBytes() padding error = %v, want %v", err, ErrPayloadOverflow)
	}

	data, err = PackBytes([]int16{10, 13, 20}, 64)
	if err != nil {
		t.Fatalf("PackBytes() error = %v", err)
	}
	if _, err = UnpackBytes[int16](data[:len(data)-1], 64); !errors.Is(err, ErrInvalidBitLimit) {
		t.Fatalf("UnpackBytes() length error = %v, want %v", err, ErrInvalidBitLimit)
	}
}

func TestPackUnpackBase64Variant1WithBigValuesHighDelta(t *testing.T) {
	input := []int64{
		43270164902,
		43270164917,
		43270164924,
		43270164937,
		43270164950,
		43270164964,
		43270164979,
		43270164988,
		43270164999,
	}
	want := []int64{
		43270164902,
		43270164917,
		43270164924,
		43270164937,
		43270164950,
		43270164964,
		43270164979,
		43270164988,
		43270164999,
	}

	id, err := PackBase64(input)
	if err != nil {
		t.Fatalf("Pack() error = %v", err)
	}
	got, err := UnpackBase64[int64](id)
	if err != nil {
		t.Fatalf("Unpack() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unpack() = %#v, want %#v", got, want)
	}
}
