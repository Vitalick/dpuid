# dpackuuid

[![Go Reference](https://pkg.go.dev/badge/github.com/Vitalick/dpackuuid.svg)](https://pkg.go.dev/github.com/Vitalick/dpackuuid)
[![Go Report Card](https://goreportcard.com/badge/github.com/Vitalick/dpackuuid)](https://goreportcard.com/report/github.com/Vitalick/dpackuuid)
[![Tests](https://github.com/Vitalick/dpackuuid/actions/workflows/test.yml/badge.svg)](https://github.com/Vitalick/dpackuuid/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Vitalick/dpackuuid)](go.mod)
[![gofmt](https://img.shields.io/badge/gofmt-yes-00ADD8)](https://pkg.go.dev/cmd/gofmt)
[![License](https://img.shields.io/github/license/Vitalick/dpackuuid)](LICENSE)

Delta-Pack UUID packs a same-sign integer sequence into one UUID-sized
value using the format described in [SPEC.md](SPEC.md). It is optimized for
sequences where neighboring absolute values have small deltas.

Russian README: [README.ru.md](README.ru.md).

## Features

- RFC 9562 UUIDv8 mode by default, using `github.com/google/uuid.UUID`.
- Raw 128-bit mode for closed systems that do not need RFC UUID markers.
- Generic input and output for `int`, `int8`..`int64`, `uint`, and `uint8`..`uint64`.
- Three SPEC variants: general deltas, sequential unit-step values, and identical values.
- No runtime dependencies except `github.com/google/uuid`.
- Go 1.20 module target.

## Install

```sh
go get github.com/Vitalick/dpackuuid
```

## Usage

```go
package main

import (
	"fmt"
	"log"

	"github.com/Vitalick/dpackuuid"
)

func main() {
	id, err := dpackuuid.Pack([]int64{1_000_040, 1_000_010, 1_000_030, 1_000_000})
	if err != nil {
		log.Fatal(err)
	}

	values, err := dpackuuid.Unpack(id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(id.String())
	fmt.Println(values) // [1000000 1000010 1000030 1000040]
}
```

`Pack` and `Unpack` use UUIDv8 mode. Use `PackMode` and `UnpackMode` for raw mode:

```go
id, err := dpackuuid.PackMode(values, dpackuuid.ModeRaw)
values, err = dpackuuid.UnpackMode(id, dpackuuid.ModeRaw)
```

Use the generic API for other integer types, including unsigned values:

```go
id, err := dpackuuid.PackValues([]uint64{1<<63 + 6, 1<<63 + 2, 1<<63 + 4})
if err != nil {
	log.Fatal(err)
}

values, err := dpackuuid.UnpackValues[uint64](id)
if err != nil {
	log.Fatal(err)
}

fmt.Println(values) // [9223372036854775810 9223372036854775812 9223372036854775814]
```

## Important Behavior

Input values must be sign-homogeneous: all values are non-negative or all values
are non-positive. Zero can be encoded with either group. Mixed positive and
negative values are rejected.

Unsigned values are always treated as non-negative. When unpacking into a typed
slice, every decoded value must fit into the requested type or `ErrValueOverflow`
is returned.

The input order is not preserved. Decoded values are sorted by absolute value,
as required by the SPEC.

Raw mode uses all 128 bits for DPUID data and does not produce an RFC-compliant
UUID. UUIDv8 mode is the recommended default for UUID-aware systems.

## Validation

The package exposes sentinel errors such as `ErrEmptyInput`, `ErrMixedSigns`,
`ErrDeltaOverflow`, `ErrCountOverflow`, `ErrTotalOverflow`, `ErrInvalidMode`,
`ErrInvalidUUIDv8`, and `ErrPayloadOverflow`. Returned errors can be checked with
`errors.Is`.

## Benchmarks

Snapshot from `go test -bench=. ./...` on `linux/amd64`, AMD Ryzen 9 3900X:

```text
BenchmarkPackVariant1-24                   1703496     709.6 ns/op
BenchmarkUnpackVariant1-24                 1517302     807.5 ns/op
BenchmarkPackSequentialVariant2-24         1759216     730.2 ns/op
BenchmarkUnpackSequentialVariant2-24        917816      1140 ns/op
BenchmarkPackIdenticalVariant3-24          1751436     673.4 ns/op
BenchmarkUnpackIdenticalVariant3-24         963639      1144 ns/op
```

## Development

```sh
go test ./...
go test -bench=. ./...
gofmt -w .
```
