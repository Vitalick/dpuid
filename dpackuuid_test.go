package dpuid

import (
	"errors"
	"reflect"
	"testing"
)

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
