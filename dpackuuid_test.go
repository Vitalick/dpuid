package dpuid

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

func TestPackUnpackUUIDv8Variant1(t *testing.T) {
	input := []int64{1_000_040, 1_000_010, 1_000_030, 1_000_000, 1_000_020}
	want := []int64{1_000_000, 1_000_010, 1_000_020, 1_000_030, 1_000_040}

	id, err := Pack(input)
	if err != nil {
		t.Fatalf("Pack() error = %v", err)
	}
	if id.Version() != 8 {
		t.Fatalf("Version() = %d, want 8", id.Version())
	}
	if id.Variant() != uuid.RFC4122 {
		t.Fatalf("Variant() = %v, want RFC4122", id.Variant())
	}

	got, err := Unpack(id)
	if err != nil {
		t.Fatalf("Unpack() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unpack() = %#v, want %#v", got, want)
	}
}

func TestPackUnpackModes(t *testing.T) {
	tests := []struct {
		name   string
		mode   Mode
		input  []int64
		output []int64
	}{
		{
			name:   "raw variant 1",
			mode:   ModeRaw,
			input:  []int64{10, 2, 6, 14},
			output: []int64{2, 6, 10, 14},
		},
		{
			name:   "uuidv8 sequential variant 2",
			mode:   ModeUUIDv8,
			input:  []int64{7, 5, 6, 8},
			output: []int64{5, 6, 7, 8},
		},
		{
			name:   "uuidv8 identical variant 3",
			mode:   ModeUUIDv8,
			input:  []int64{42, 42, 42},
			output: []int64{42, 42, 42},
		},
		{
			name:   "mixed zero and one deltas use general variant",
			mode:   ModeUUIDv8,
			input:  []int64{2, 1, 1},
			output: []int64{1, 1, 2},
		},
		{
			name:   "zeros with negative values stay non-positive",
			mode:   ModeUUIDv8,
			input:  []int64{-2, 0, -1},
			output: []int64{0, -1, -2},
		},
		{
			name:   "min int64",
			mode:   ModeUUIDv8,
			input:  []int64{math.MinInt64},
			output: []int64{math.MinInt64},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := PackMode(tt.input, tt.mode)
			if err != nil {
				t.Fatalf("PackMode() error = %v", err)
			}

			got, err := UnpackMode(id, tt.mode)
			if err != nil {
				t.Fatalf("UnpackMode() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.output) {
				t.Fatalf("UnpackMode() = %#v, want %#v", got, tt.output)
			}
		})
	}
}

func TestPackUnpackVariant1WithZeroOne(t *testing.T) {
	const start = int64(123_456_789_012)
	input := make([]int64, 15)
	want := make([]int64, 15)
	previousNum := start
	for i := range input {
		input[i] = previousNum + int64(i%2)
		want[i] = input[i]
		previousNum = input[i]
	}

	id, err := Pack(input)
	if err != nil {
		t.Fatalf("Pack() error = %v", err)
	}
	got, err := Unpack(id)
	if err != nil {
		t.Fatalf("Unpack() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unpack() = %#v, want %#v", got, want)
	}
}

func TestPackUnpackVariant1WithBigValues(t *testing.T) {
	const start = int64(123_456_789_012)
	input := make([]int64, 15)
	want := make([]int64, 15)
	previousNum := start
	for i := range input {
		input[i] = previousNum + rand.Int63n(10)
		want[i] = input[i]
		previousNum = input[i]
	}

	id, err := Pack(input)
	if err != nil {
		t.Fatalf("Pack() error = %v", err)
	}
	got, err := Unpack(id)
	if err != nil {
		t.Fatalf("Unpack() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unpack() = %#v, want %#v", got, want)
	}
}

func TestPackUnpackVariant1WithBigValuesHighDelta(t *testing.T) {
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

	id, err := Pack(input)
	if err != nil {
		t.Fatalf("Pack() error = %v", err)
	}
	got, err := Unpack(id)
	if err != nil {
		t.Fatalf("Unpack() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unpack() = %#v, want %#v", got, want)
	}
}

func TestPackUnpackVariant2WithHundredValues(t *testing.T) {
	const start = int64(12_345_678_901)
	input := make([]int64, 100)
	want := make([]int64, 100)
	for i := range input {
		input[i] = start + int64(len(input)-1-i)
		want[i] = start + int64(i)
	}

	id, err := Pack(input)
	if err != nil {
		t.Fatalf("Pack() error = %v", err)
	}
	got, err := Unpack(id)
	if err != nil {
		t.Fatalf("Unpack() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unpack() = %#v, want %#v", got, want)
	}
}

func TestPackUnpackVariant3WithHundredValues(t *testing.T) {
	const value = int64(12_345_678_901)
	input := make([]int64, 100)
	want := make([]int64, 100)
	for i := range input {
		input[i] = value
		want[i] = value
	}

	id, err := Pack(input)
	if err != nil {
		t.Fatalf("Pack() error = %v", err)
	}
	got, err := Unpack(id)
	if err != nil {
		t.Fatalf("Unpack() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Unpack() = %#v, want %#v", got, want)
	}
}

func TestPackValidation(t *testing.T) {
	tests := []struct {
		name  string
		input []int64
		want  error
	}{
		{
			name:  "empty",
			input: nil,
			want:  ErrEmptyInput,
		},
		{
			name:  "mixed signs",
			input: []int64{-1, 0, 1},
			want:  ErrMixedSigns,
		},
		{
			name:  "delta overflow",
			input: []int64{0, 1 << 32},
			want:  ErrDeltaOverflow,
		},
		{
			name:  "variant 1 count overflow",
			input: nonSequential(65),
			want:  ErrCountOverflow,
		},
		{
			name:  "variant 1 total overflow",
			input: []int64{1<<62 + 0, 1<<62 + 4, 1<<62 + 8, 1<<62 + 12, 1<<62 + 16, 1<<62 + 20, 1<<62 + 24, 1<<62 + 28, 1<<62 + 32, 1<<62 + 36, 1<<62 + 40, 1<<62 + 44, 1<<62 + 48, 1<<62 + 52, 1<<62 + 56},
			want:  ErrTotalOverflow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Pack(tt.input)
			if !errors.Is(err, tt.want) {
				t.Fatalf("Pack() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestModeValidation(t *testing.T) {
	_, err := PackMode([]int64{1}, Mode(99))
	if !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("PackMode() error = %v, want %v", err, ErrInvalidMode)
	}

	_, err = UnpackMode(uuid.Nil, Mode(99))
	if !errors.Is(err, ErrInvalidMode) {
		t.Fatalf("UnpackMode() error = %v, want %v", err, ErrInvalidMode)
	}
}

func TestUnpackRejectsInvalidUUIDv8Markers(t *testing.T) {
	id, err := Pack([]int64{1, 2, 3})
	if err != nil {
		t.Fatalf("Pack() error = %v", err)
	}
	id[6] &^= 0xf0

	_, err = Unpack(id)
	if !errors.Is(err, ErrInvalidUUIDv8) {
		t.Fatalf("Unpack() error = %v, want %v", err, ErrInvalidUUIDv8)
	}
}

func TestGoogleUUIDStringRoundTrip(t *testing.T) {
	id, err := Pack([]int64{100, 104, 108})
	if err != nil {
		t.Fatalf("Pack() error = %v", err)
	}

	parsed, err := uuid.Parse(id.String())
	if err != nil {
		t.Fatalf("uuid.Parse() error = %v", err)
	}
	got, err := Unpack(parsed)
	if err != nil {
		t.Fatalf("Unpack() error = %v", err)
	}
	if want := []int64{100, 104, 108}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Unpack() = %#v, want %#v", got, want)
	}
}

func TestRawValueIsNotRequiredToBeUUIDv8(t *testing.T) {
	id, err := PackMode([]int64{1, 5, 9}, ModeRaw)
	if err != nil {
		t.Fatalf("PackMode() error = %v", err)
	}

	_, err = Unpack(id)
	if !errors.Is(err, ErrInvalidUUIDv8) {
		t.Fatalf("Unpack() error = %v, want %v", err, ErrInvalidUUIDv8)
	}

	got, err := UnpackMode(id, ModeRaw)
	if err != nil {
		t.Fatalf("UnpackMode() error = %v", err)
	}
	if want := []int64{1, 5, 9}; !reflect.DeepEqual(got, want) {
		t.Fatalf("UnpackMode() = %#v, want %#v", got, want)
	}
}

func TestPackUnpackGenericSignedIntegers(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		assertGenericRoundTrip(t, []int{-4, -2, -3}, []int{-2, -3, -4})
	})
	t.Run("int8", func(t *testing.T) {
		assertGenericRoundTrip(t, []int8{-4, -2, -3}, []int8{-2, -3, -4})
	})
	t.Run("int16", func(t *testing.T) {
		assertGenericRoundTrip(t, []int16{-4, -2, -3}, []int16{-2, -3, -4})
	})
	t.Run("int32", func(t *testing.T) {
		assertGenericRoundTrip(t, []int32{-4, -2, -3}, []int32{-2, -3, -4})
	})
	t.Run("int64", func(t *testing.T) {
		assertGenericRoundTrip(t, []int64{-4, -2, -3}, []int64{-2, -3, -4})
	})
}

func TestPackUnpackGenericUnsignedIntegers(t *testing.T) {
	t.Run("uint", func(t *testing.T) {
		assertGenericRoundTrip(t, []uint{4, 2, 3}, []uint{2, 3, 4})
	})
	t.Run("uint8", func(t *testing.T) {
		assertGenericRoundTrip(t, []uint8{4, 2, 3}, []uint8{2, 3, 4})
	})
	t.Run("uint16", func(t *testing.T) {
		assertGenericRoundTrip(t, []uint16{4, 2, 3}, []uint16{2, 3, 4})
	})
	t.Run("uint32", func(t *testing.T) {
		assertGenericRoundTrip(t, []uint32{4, 2, 3}, []uint32{2, 3, 4})
	})
	t.Run("uint64", func(t *testing.T) {
		assertGenericRoundTrip(t, []uint64{1<<63 + 6, 1<<63 + 2, 1<<63 + 4}, []uint64{1<<63 + 2, 1<<63 + 4, 1<<63 + 6})
	})
}

func TestGenericUnpackRequiresEncodingType(t *testing.T) {
	id, err := PackValues([]uint16{255, 256})
	if err != nil {
		t.Fatalf("PackValues() error = %v", err)
	}

	_, err = UnpackValues[uint8](id)
	if err == nil {
		t.Fatal("UnpackValues[uint8]() error = nil, want type mismatch rejection")
	}
}

func TestPackUnpackBytes(t *testing.T) {
	tests := []struct {
		name string
		bits int
		in   []int16
		want []int16
	}{
		{name: "dynamic variant 1", in: []int16{20, 10, 13}, want: []int16{10, 13, 20}},
		{name: "dynamic variant 2", in: []int16{13, 10, 12, 11}, want: []int16{10, 11, 12, 13}},
		{name: "dynamic variant 3", in: []int16{42, 42, 42}, want: []int16{42, 42, 42}},
		{name: "fixed variant 1", bits: 64, in: []int16{20, 10, 13}, want: []int16{10, 13, 20}},
		{name: "fixed variant 2", bits: 64, in: []int16{13, 10, 12, 11}, want: []int16{10, 11, 12, 13}},
		{name: "fixed variant 3", bits: 64, in: []int16{42, 42, 42}, want: []int16{42, 42, 42}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := PackBytes(tt.in, tt.bits)
			if err != nil {
				t.Fatalf("PackBytes() error = %v", err)
			}
			got, err := UnpackBytes[int16](data, tt.bits)
			if err != nil {
				t.Fatalf("UnpackBytes() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("UnpackBytes() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

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

func TestElementTypeControlsFieldLimits(t *testing.T) {
	_, err := PackBytes([]int8{0, 8}, 0)
	if !errors.Is(err, ErrDeltaOverflow) {
		t.Fatalf("PackBytes[int8]() error = %v, want %v", err, ErrDeltaOverflow)
	}

	values := []int8{0, 2, 4, 6, 8, 10, 12, 14, 16}
	_, err = PackBytes(values, 0)
	if !errors.Is(err, ErrCountOverflow) {
		t.Fatalf("PackBytes[int8]() error = %v, want %v", err, ErrCountOverflow)
	}

	if _, err = PackBytes([]int16{0, 8}, 0); err != nil {
		t.Fatalf("PackBytes[int16]() error = %v", err)
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

func BenchmarkPackVariant1(b *testing.B) {
	values := []int64{1_000_040, 1_000_010, 1_000_030, 1_000_000, 1_000_020}
	for i := 0; i < b.N; i++ {
		_, _ = Pack(values)
	}
}

func BenchmarkUnpackVariant1(b *testing.B) {
	id, err := Pack([]int64{1_000_040, 1_000_010, 1_000_030, 1_000_000, 1_000_020})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Unpack(id)
	}
}

func BenchmarkPackSequentialVariant2(b *testing.B) {
	values := []int64{10, 11, 12, 13, 14, 15, 16, 17}
	for i := 0; i < b.N; i++ {
		_, _ = Pack(values)
	}
}

func BenchmarkUnpackSequentialVariant2(b *testing.B) {
	id, err := Pack([]int64{10, 11, 12, 13, 14, 15, 16, 17})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Unpack(id)
	}
}

func BenchmarkPackIdenticalVariant3(b *testing.B) {
	values := []int64{42, 42, 42, 42, 42, 42, 42, 42}
	for i := 0; i < b.N; i++ {
		_, _ = Pack(values)
	}
}

func BenchmarkUnpackIdenticalVariant3(b *testing.B) {
	id, err := Pack([]int64{42, 42, 42, 42, 42, 42, 42, 42})
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Unpack(id)
	}
}

func nonSequential(n int) []int64 {
	out := make([]int64, n)
	for i := range out {
		out[i] = int64(i * 2)
	}
	return out
}

func assertGenericRoundTrip[T Integer](t *testing.T, input, want []T) {
	t.Helper()

	id, err := PackValues(input)
	if err != nil {
		t.Fatalf("PackValues() error = %v", err)
	}
	got, err := UnpackValues[T](id)
	if err != nil {
		t.Fatalf("UnpackValues() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("UnpackValues() = %#v, want %#v", got, want)
	}
}
