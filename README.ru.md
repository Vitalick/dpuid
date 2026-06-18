# dpackuuid

[![Go Reference](https://pkg.go.dev/badge/github.com/Vitalick/dpackuuid.svg)](https://pkg.go.dev/github.com/Vitalick/dpackuuid)
[![Go Report Card](https://goreportcard.com/badge/github.com/Vitalick/dpackuuid)](https://goreportcard.com/report/github.com/Vitalick/dpackuuid)
[![Tests](https://github.com/Vitalick/dpackuuid/actions/workflows/test.yml/badge.svg)](https://github.com/Vitalick/dpackuuid/actions/workflows/test.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Vitalick/dpackuuid)](go.mod)
[![gofmt](https://img.shields.io/badge/gofmt-yes-00ADD8)](https://pkg.go.dev/cmd/gofmt)
[![License](https://img.shields.io/github/license/Vitalick/dpackuuid)](LICENSE)

Delta-Pack UUID упаковывает sign-homogeneous последовательность `int64` в одно
UUID-sized значение по формату из [SPEC.md](SPEC.md). Формат лучше всего подходит
для последовательностей, где соседние абсолютные значения отличаются небольшими
дельтами.

English README: [README.md](README.md).

## Возможности

- RFC 9562 UUIDv8 mode по умолчанию, публичный тип `github.com/google/uuid.UUID`.
- Raw 128-bit mode для закрытых систем, где не нужны RFC UUID markers.
- Все три варианта из SPEC: общие дельты, последовательные значения с шагом 1 и одинаковые значения.
- Без runtime-зависимостей, кроме `github.com/google/uuid`.
- Целевая версия модуля Go 1.20.

## Установка

```sh
go get github.com/Vitalick/dpackuuid
```

## Использование

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

`Pack` и `Unpack` используют UUIDv8 mode. Для raw mode используйте `PackMode` и
`UnpackMode`:

```go
id, err := dpackuuid.PackMode(values, dpackuuid.ModeRaw)
values, err = dpackuuid.UnpackMode(id, dpackuuid.ModeRaw)
```

## Важное Поведение

Входные значения должны быть sign-homogeneous: все значения неотрицательные или
все значения неположительные. Ноль можно кодировать с любой группой. Смешанные
положительные и отрицательные значения отклоняются.

Порядок входных значений не сохраняется. При распаковке значения возвращаются
отсортированными по абсолютному значению, как требует SPEC.

Raw mode использует все 128 бит под DPUID data и не создает RFC-compliant UUID.
UUIDv8 mode является рекомендуемым режимом по умолчанию для систем, которые
проверяют UUID-структуру.

## Валидация

Пакет экспортирует sentinel errors: `ErrEmptyInput`, `ErrMixedSigns`,
`ErrDeltaOverflow`, `ErrCountOverflow`, `ErrTotalOverflow`, `ErrInvalidMode`,
`ErrInvalidUUIDv8` и `ErrPayloadOverflow`. Возвращаемые ошибки можно проверять
через `errors.Is`.

## Разработка

```sh
go test ./...
go test -bench=. ./...
gofmt -w .
```
