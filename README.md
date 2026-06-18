# dpackuuid

[![Go Reference](https://pkg.go.dev/badge/github.com/Vitalick/dpackuuid.svg)](https://pkg.go.dev/github.com/Vitalick/dpackuuid)
[![Go Report Card](https://goreportcard.com/badge/github.com/Vitalick/dpackuuid)](https://goreportcard.com/report/github.com/Vitalick/dpackuuid)
[![Tests](https://github.com/Vitalick/dpackuuid/actions/workflows/test.yml/badge.svg)](https://github.com/Vitalick/dpackuuid/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Vitalick/dpackuuid)](go.mod)
[![gofmt](https://img.shields.io/badge/gofmt-yes-00ADD8)](https://pkg.go.dev/cmd/gofmt)
[![License](https://img.shields.io/github/license/Vitalick/dpackuuid)](LICENSE)

Delta-Pack UUID packs a same-sign sequence of `int64` values into one UUID-sized
value using the format described in [SPEC.md](SPEC.md). It is optimized for
sequences where neighboring absolute values have small deltas.

Russian README: [README.ru.md](README.ru.md).

## Features

- RFC 9562 UUIDv8 mode by default, using `github.com/google/uuid.UUID`.
- Raw 128-bit mode for closed systems that do not need RFC UUID markers.
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

## Important Behavior

Input values must be sign-homogeneous: all values are non-negative or all values
are non-positive. Zero can be encoded with either group. Mixed positive and
negative values are rejected.

The input order is not preserved. Decoded values are sorted by absolute value,
as required by the SPEC.

Raw mode uses all 128 bits for DPUID data and does not produce an RFC-compliant
UUID. UUIDv8 mode is the recommended default for UUID-aware systems.

## Validation

The package exposes sentinel errors such as `ErrEmptyInput`, `ErrMixedSigns`,
`ErrDeltaOverflow`, `ErrCountOverflow`, `ErrTotalOverflow`, `ErrInvalidMode`,
`ErrInvalidUUIDv8`, and `ErrPayloadOverflow`. Returned errors can be checked with
`errors.Is`.

## Development

```sh
go test ./...
go test -bench=. ./...
gofmt -w .
```
