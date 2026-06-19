# Delta-Pack UUID (DPUID)

[![Go Reference](https://pkg.go.dev/badge/github.com/Vitalick/dpuid.svg)](https://pkg.go.dev/github.com/Vitalick/dpuid)
[![Go Report Card](https://goreportcard.com/badge/github.com/Vitalick/dpuid)](https://goreportcard.com/report/github.com/Vitalick/dpuid)
[![Tests](https://github.com/Vitalick/dpuid/actions/workflows/test.yml/badge.svg)](https://github.com/Vitalick/dpuid/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Vitalick/dpuid)](go.mod)
[![gofmt](https://img.shields.io/badge/gofmt-yes-00ADD8)](https://pkg.go.dev/cmd/gofmt)
[![License](https://img.shields.io/github/license/Vitalick/dpuid)](LICENSE)

Delta-Pack UUID упаковывает sign-homogeneous целочисленную последовательность в
DPUID byte payload, UUID или строку base64 по формату из [SPEC.ru.md](SPEC.ru.md)
([English SPEC](SPEC.md)). Формат лучше всего подходит для последовательностей,
где соседние абсолютные значения отличаются небольшими дельтами.

English README: [README.md](README.md).

## Возможности

- RFC 9562 UUIDv8 mode по умолчанию, публичный тип `github.com/google/uuid.UUID`.
- Raw 128-bit mode для закрытых систем, где не нужны RFC UUID markers.
- Byte payload переменной или фиксированной длины и base64 adapters.
- Generic input/output для `int`, `int8`..`int64`, `uint` и `uint8`..`uint64`.
- Все три варианта из SPEC: общие дельты, последовательные значения с шагом 1 и одинаковые значения.
- Без runtime-зависимостей, кроме `github.com/google/uuid`.
- Целевая версия модуля Go 1.20.

## Установка

```sh
go get github.com/Vitalick/dpuid
```

## Использование

```go
package main

import (
	"fmt"
	"log"

	"github.com/Vitalick/dpuid"
)

func main() {
	id, err := dpuid.Pack([]int64{1_000_040, 1_000_010, 1_000_030, 1_000_000})
	if err != nil {
		log.Fatal(err)
	}

	values, err := dpuid.Unpack(id)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(id.String())
	fmt.Println(values) // [1000000 1000010 1000030 1000040]
}
```

`Pack` и `Unpack` используют UUIDv8 mode. Для raw mode используйте `PackMode` и
`UnpackMode`:

```go
id, err := dpuid.PackMode(values, dpuid.ModeRaw)
values, err = dpuid.UnpackMode(id, dpuid.ModeRaw)
```

Для других целочисленных типов, включая unsigned, используйте generic API:

```go
id, err := dpuid.PackValues([]uint64{1<<63 + 6, 1<<63 + 2, 1<<63 + 4})
if err != nil {
	log.Fatal(err)
}

values, err := dpuid.UnpackValues[uint64](id)
if err != nil {
	log.Fatal(err)
}

fmt.Println(values) // [9223372036854775810 9223372036854775812 9223372036854775814]
```

Если UUID или текстовый transport не нужен, используйте byte codec напрямую.
Нулевой размер выбирает минимальный byte-aligned payload, положительный задаёт
точный bit limit:

```go
data, err := dpuid.PackBytes([]int16{20, 10, 13}, 0)
values, err := dpuid.UnpackBytes[int16](data, 0)

fixed, err := dpuid.PackBytes([]int16{20, 10, 13}, 64)
values, err = dpuid.UnpackBytes[int16](fixed, 64)
```

Base64 использует тот же byte codec:

```go
encoded, err := dpuid.PackBase64([]int8{10, 13, 11, 12}) // "Olg="
values8, err := dpuid.UnpackBase64[int8](encoded)

fixedEncoded, err := dpuid.PackBase64Bits([]uint32{100, 104, 108}, 96)
values32, err := dpuid.UnpackBase64Bits[uint32](fixedEncoded, 96)
```

## Важное Поведение

Входные значения должны быть sign-homogeneous: все значения неотрицательные или
все значения неположительные. Ноль можно кодировать с любой группой. Смешанные
положительные и отрицательные значения отклоняются.

Unsigned значения всегда считаются неотрицательными. Generic type функции
распаковки должен совпадать с типом, использованным при упаковке: разрядность
элемента определяет ширины полей encoded payload.

Порядок входных значений не сохраняется. При распаковке значения возвращаются
отсортированными по абсолютному значению, как требует SPEC.

Raw mode использует все 128 бит под DPUID data и не создает RFC-compliant UUID.
UUIDv8 mode является рекомендуемым режимом по умолчанию для систем, которые
проверяют UUID-структуру.

## Валидация

Пакет экспортирует sentinel errors: `ErrEmptyInput`, `ErrMixedSigns`,
`ErrDeltaOverflow`, `ErrCountOverflow`, `ErrTotalOverflow`, `ErrInvalidMode`,
`ErrInvalidUUIDv8`, `ErrInvalidBase64`, `ErrInvalidBitLimit` и
`ErrPayloadOverflow`. Возвращаемые ошибки можно проверять через `errors.Is`.

## Бенчмарки

Snapshot с `go test -bench=. ./...` на `linux/amd64`, AMD Ryzen 9 3900X:

```text
BenchmarkPackVariant1-24                   1703496     709.6 ns/op
BenchmarkUnpackVariant1-24                 1517302     807.5 ns/op
BenchmarkPackSequentialVariant2-24         1759216     730.2 ns/op
BenchmarkUnpackSequentialVariant2-24        917816      1140 ns/op
BenchmarkPackIdenticalVariant3-24          1751436     673.4 ns/op
BenchmarkUnpackIdenticalVariant3-24         963639      1144 ns/op
```

## Разработка

```sh
go test ./...
go test -bench=. ./...
gofmt -w .
```
