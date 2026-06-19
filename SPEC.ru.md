# Delta-Pack UUID (DPUID) — Спецификация

English version: [SPEC.md](SPEC.md)

**Версия:** 1.2.0
**Статус:** Ready
**Целевой результат:** 128-битное UUID-совместимое значение или строка base64 переменной длины

---

## 1. Обзор

Delta-Pack UUID - это самодостаточная бинарная схема кодирования, которая
упаковывает последовательность целых чисел с небольшими абсолютными разницами в
одно 128-битное значение, совместимое с UUID-хранилищами, либо в строку base64
переменной длины.

Основные свойства:

- **Порядок входа не сохраняется.** Перед кодированием значения сортируются по
  возрастанию абсолютного значения.
- **Одинаковый знак.** Все входные значения должны быть sign-homogeneous: все
  неотрицательные или все неположительные. Смешанные положительные и
  отрицательные значения являются ошибкой валидации.
- **Самоописываемый payload.** Ширины полей выводятся из параметра `OUTPUT_POW`,
  который во всех режимах вычисляется из типа входных элементов. Благодаря этому
  для `int8`, `int16` и `int32` используются более компактные поля, чем для `int64`.
- **Три варианта кодирования.** Variant 1 хранит произвольные малые дельты,
  Variant 2 компактно кодирует последовательности с шагом 1, Variant 3 компактно
  кодирует одинаковые значения.
- **Два UUID mode.** Raw mode использует все 128 бит данных. UUIDv8 mode
  использует 122 бита данных и выставляет RFC 9562 UUIDv8 version/variant bits.
  UUIDv8 mode является рекомендуемым режимом по умолчанию. В обоих режимах
  `OUTPUT_POW` ограничивается так, чтобы `ELEMENT_BITS <= OUTPUT_BITS / 2`.
- **Base64 mode.** Результат имеет переменную длину и не ограничен фиксированным
  размером. `OUTPUT_POW` вычисляется из типа элемента без верхнего ограничения.

Raw mode структурно помещается в UUID-sized значение, но не является
RFC-compliant UUID. UUIDv8 mode соответствует RFC 9562. Base64 mode возвращает
стандартную строку base64 по RFC 4648.

---

## 2. Определения

| Символ | Значение |
|---|---|
| `N` | Количество входных чисел |
| `abs_sorted` | Входные значения, отсортированные по возрастанию `|x|` |
| `source_num` | `|abs_sorted[0]|` - абсолютное значение первого элемента после сортировки |
| `is_negative` | `1`, если кодируется неположительная группа, иначе `0` |
| `abs_values` | `[\|x\| for x in abs_sorted]` |
| `deltas` | `[abs_values[i+1] - abs_values[i] for i in 0..N-2]`, длина `N-1` |
| `S` | Хранимое значение поля `source_num_len_in_bits`, см. §2.1 |
| `D` | Хранимое значение поля `next_nums_len_in_bits`, см. §2.2 |
| `ELEMENT_BITS` | Разрядность одного элемента входного slice: 8, 16, 32 или 64 |
| `OUTPUT_BITS` | Фиксированный размер результата: 128 бит для UUID modes; неприменимо к base64 |
| `OUTPUT_POW` | Вычисляется из типа элемента во всех режимах, см. §3; определяет ширины полей |
| `DATA_BITS` | Доступные биты данных: 128 в raw, 122 в UUIDv8, без ограничения в base64 |

### 2.1 source_num_len_in_bits (`S`)

`S` хранится как `actual_bit_width - 1`. Поэтому поле `source_num` всегда занимает
минимум 1 бит, даже если значение равно нулю.

```text
actual_bit_width(x) = max(1, floor(log2(x)) + 1)
S = actual_bit_width(source_num) - 1
source_num field width = S + 1 bits
```

| `source_num` | `actual_bit_width` | `S` |
|---|---:|---:|
| `0` | 1 | 0 |
| `1` | 1 | 0 |
| `2..3` | 2 | 1 |
| `2^63..2^64-1` | 64 | 63 |

Поле `S` занимает `OUTPUT_POW - 1` бит. Для `int64` это 6 бит со значениями
`0..63`, которые представляют фактическую ширину `1..64` бит; для меньших типов
ширина поля уменьшается согласно §3.

### 2.2 next_nums_len_in_bits (`D`)

`D` хранит эффективную ширину максимальной дельты, но значения `0` и `1`
зарезервированы как discriminators для compact variants.

```text
D = 0                          // Variant 3: все дельты равны 0
D = 1                          // Variant 2: все дельты равны 1
D = floor(log2(max_delta)) + 1 // Variant 1: max_delta >= 2, D >= 2
```

| Условие | `D` | Variant |
|---|---:|---|
| Все дельты равны `0` или `N == 1` | 0 | Variant 3 |
| Все дельты равны `1` | 1 | Variant 2 |
| Максимальная дельта `2..3` | 2 | Variant 1 |
| Максимальная дельта `4..7` | 3 | Variant 1 |
| Максимальная дельта `2^30..2^31-1` | 31 | Variant 1 |

Если последовательность содержит смесь дельт `0` и `1`, это Variant 1. В таком
случае encoder должен использовать `D >= 2`, потому что `D = 0` и `D = 1`
зарезервированы.

---

## 3. Параметры

### 3.1 Вычисление OUTPUT_POW во всех режимах

Во всех режимах `OUTPUT_POW` вычисляется из `ELEMENT_BITS` по одной формуле.
В UUID modes дополнительно действует верхнее ограничение:

```text
OUTPUT_POW = min(log2(ELEMENT_BITS * 2), log2(OUTPUT_BITS))
```

- Base64 mode: `OUTPUT_BITS = infinity`, поэтому `OUTPUT_POW = log2(ELEMENT_BITS * 2)`.
- UUID modes: `OUTPUT_BITS = 128`, поэтому `OUTPUT_POW = min(log2(ELEMENT_BITS * 2), 7)`.

Ограничение гарантирует `ELEMENT_BITS <= OUTPUT_BITS / 2`. Для стандартных типов
`int8..int64` и 128-битного UUID тип `int64` находится точно на границе, а меньшие
типы получают уменьшенный `OUTPUT_POW`.

| Тип элемента | `ELEMENT_BITS` | Base64 `OUTPUT_POW` | UUID `OUTPUT_POW` |
|---|---:|---:|---:|
| `int8` / `uint8` | 8 | 4 | **4** |
| `int16` / `uint16` | 16 | 5 | **5** |
| `int32` / `uint32` | 32 | 6 | **6** |
| `int64` / `uint64` | 64 | 7 | **7** (верхняя граница) |

### 3.2 Ширины полей при OUTPUT_POW = p

| Поле | Формула | p=4 | p=5 | p=6 | p=7 |
|---|---|---:|---:|---:|---:|
| `SOURCE_LEN_FIELD` | `p - 1` | 3 бита | 4 бита | 5 бит | 6 бит |
| `DELTA_LEN_FIELD` | `p - 2` | 2 бита | 3 бита | 4 бита | 5 бит |
| `COUNT_FIELD` | `p - 1` | 3 бита | 4 бита | 5 бит | 6 бит |
| Максимальный `S` | `2^(p-1) - 1` | 7 | 15 | 31 | 63 |
| Максимальная ширина `source_num` | `2^(p-1)` | 8 | 16 | 32 | 64 |
| Максимальный `D` | `2^(p-2) - 1` | 3 | 7 | 15 | 31 |
| Максимальная delta | `2^(2^(p-2)) - 1` | 7 | 127 | 32767 | `2^31-1` |
| Максимальный `N` (V1 и base64 V2/V3) | `2^(p-1)` | 8 | 16 | 32 | 64 |
| `HEADER_BITS` | `3p - 2 + S` | `10+S` | `13+S` | `16+S` | `19+S` |

---

## 4. UUID modes

### 4.1 Raw mode

Raw mode использует все 128 бит под DPUID data.

```text
OUTPUT_BITS = 128
DATA_BITS   = 128
OUTPUT_POW  = min(log2(ELEMENT_BITS * 2), 7)
```

Raw mode не выставляет UUID version/variant bits и не должен использоваться в
системах, которые требуют RFC-compliant UUID.

### 4.2 UUIDv8 mode

UUIDv8 mode выставляет два служебных поля по RFC 9562:

| UUID bits | Поле | Значение |
|---|---|---|
| `48..51` | version | `0x8` |
| `64..65` | variant | `0b10` |

Эти 6 бит не являются DPUID data. Остальные 122 бита используются для payload.

```text
OUTPUT_BITS = 128
DATA_BITS   = 122
OUTPUT_POW  = min(log2(ELEMENT_BITS * 2), 7)
```

Mapping DPUID data в UUID:

```text
UUID bits 0..47    <- DPUID bits 0..47
UUID bits 48..51   <- version = 0x8
UUID bits 52..63   <- DPUID bits 48..59
UUID bits 64..65   <- variant = 0b10
UUID bits 66..127  <- DPUID bits 60..121
```

Decoder в UUIDv8 mode должен проверить version `8` и variant `10`. Если markers
невалидны, значение должно быть отклонено.

### 4.3 Base64 mode

Base64 mode формирует byte array переменной длины и кодирует его стандартным
base64 по RFC 4648. Ограничения на фиксированный размер результата нет.

```text
OUTPUT_POW = log2(ELEMENT_BITS * 2)
```

| Тип элемента | `ELEMENT_BITS` | `OUTPUT_POW` | Поле S | Поле D | Count | Max N | Max D |
|---|---:|---:|---:|---:|---:|---:|---:|
| `int8` / `uint8` | 8 | 4 | 3 | 2 | 3 | 8 | 3 |
| `int16` / `uint16` | 16 | 5 | 4 | 3 | 4 | 16 | 7 |
| `int32` / `uint32` | 32 | 6 | 5 | 4 | 5 | 32 | 15 |
| `int64` / `uint64` | 64 | 7 | 6 | 5 | 6 | 64 | 31 |

Отличия от UUID modes:

1. Ограничения `DATA_BITS` нет, поэтому precondition P6 не применяется.
2. Во всех variants используется явное поле count шириной `OUTPUT_POW - 1`.
3. После данных добавляются нули до границы байта.
4. UUID version/variant markers не добавляются.

```text
total_bits  = (3 * OUTPUT_POW - 2 + S) + D * (N - 1) // V1; для V2/V3 последнее слагаемое равно 0
total_bytes = ceil(total_bits / 8)
base64_len  = ceil(total_bytes / 3) * 4
```

| Сценарий для int64 | S | D | N | Биты | Байты | Символы base64 |
|---|---:|---:|---:|---:|---:|---:|
| 20 чисел, delta <= 15 | 40 | 4 | 20 | 135 | 17 | 24 |
| 20 чисел, delta = 1 | 40 | 1 | 20 | 59 | 8 | 12 |
| 64 числа, delta <= 7 | 32 | 3 | 64 | 240 | 30 | 40 |
| 1 число | 20 | 0 | 1 | 39 | 5 | 8 |

---

## 5. Preconditions encoder

Encoder должен отклонить вход, если нарушено любое условие:

`OUTPUT_POW` и зависящие от него пределы определяются типом элемента, см. §3.

| Код | Условие | Где применяется | Ошибка |
|---|---|---|---|
| P1 | `N >= 1` | Все режимы | empty input |
| P2 | Все значения одного знака: все `>= 0` или все `<= 0` | Все режимы | mixed signs |
| P3 | `S <= 2^(OUTPUT_POW-1) - 1` | Все режимы | source too wide |
| P4 | `D <= 2^(OUTPUT_POW-2) - 1` | Все режимы | delta слишком велика для типа элемента |
| P5 | `N - 1 <= 2^(OUTPUT_POW-1) - 1` | V1 в UUID; все variants в base64 | too many numbers |
| P6 | `(3p-2+S) + D*(N-1) <= DATA_BITS` | Только UUID modes | overflow |

Unsigned values всегда считаются неотрицательными.

---

## 6. Выбор variant

```text
if all deltas == 0 or N == 1:
    variant = 3
    D = 0
else if all deltas == 1:
    variant = 2
    D = 1
else:
    variant = 1
    D = max(2, floor(log2(max_delta)) + 1)
```

Decoder выбирает variant по `D`:

| `D` | Variant | Смысл |
|---:|---|---|
| 0 | Variant 3 | Все дельты неявно равны 0 |
| 1 | Variant 2 | Все дельты неявно равны 1 |
| `>= 2` | Variant 1 | Дельты явно записаны в payload |

---

## 7. Bit layout

Все поля записываются MSB first.

### 7.1 Variant 1 - general deltas

Layout одинаков во всех режимах. В UUID modes хвост заполняется нулями до
`DATA_BITS`, в base64 mode — до границы байта.

```text
Offset               Width                    Field
0                    1                        is_negative
1                    OUTPUT_POW - 1           S
OUTPUT_POW           S + 1                    source_num
OUTPUT_POW + S + 1   OUTPUT_POW - 2           D, D >= 2
2*OUTPUT_POW + S - 1 OUTPUT_POW - 1           count of deltas, N - 1
3*OUTPUT_POW + S - 2 D * (N - 1)              delta values, fixed-width
...                  padding                  UUID: до DATA_BITS; base64: до границы байта
```

Total data bits:

```text
(3 * OUTPUT_POW - 2 + S) + D * (N - 1) <= DATA_BITS // UUID modes
```

### 7.2 Variant 2 - все дельты равны 1

#### UUID modes

```text
Offset                      Width                              Field
0                           1                                  is_negative
1                           OUTPUT_POW - 1                     S
OUTPUT_POW                  S + 1                              source_num
OUTPUT_POW + S + 1          OUTPUT_POW - 2                     D = 1
2*OUTPUT_POW + S - 1        DATA_BITS - (3*OUTPUT_POW-2+S)    count, N - 1
```

Count занимает все оставшиеся биты.

#### Base64 mode

```text
Offset                      Width                 Field
0                           1                     is_negative
1                           OUTPUT_POW - 1        S
OUTPUT_POW                  S + 1                 source_num
OUTPUT_POW + S + 1          OUTPUT_POW - 2        D = 1
2*OUTPUT_POW + S - 1        OUTPUT_POW - 1        count, N - 1
3*OUTPUT_POW + S - 2        padding               до границы байта
```

Дельты не записываются: каждая следующая absolute value равна предыдущей плюс 1.

### 7.3 Variant 3 - все дельты равны 0

Структура совпадает с Variant 2; отличается только значение `D`.

#### UUID modes

```text
Offset                      Width                              Field
0                           1                                  is_negative
1                           OUTPUT_POW - 1                     S
OUTPUT_POW                  S + 1                              source_num
OUTPUT_POW + S + 1          OUTPUT_POW - 2                     D = 0
2*OUTPUT_POW + S - 1        DATA_BITS - (3*OUTPUT_POW-2+S)    count, N - 1
```

#### Base64 mode

```text
Offset                      Width                 Field
0                           1                     is_negative
1                           OUTPUT_POW - 1        S
OUTPUT_POW                  S + 1                 source_num
OUTPUT_POW + S + 1          OUTPUT_POW - 2        D = 0
2*OUTPUT_POW + S - 1        OUTPUT_POW - 1        count, N - 1
3*OUTPUT_POW + S - 2        padding               до границы байта
```

Дельты не записываются: все reconstructed absolute values равны `source_num`.

---

## 8. Capacity reference

### 8.1 Общая формула

```text
HEADER_BITS = 3 * OUTPUT_POW - 2 + S

N_max (V1, UUID)   = floor((DATA_BITS - HEADER_BITS) / D) + 1
N_max (V1, base64) = 2^(OUTPUT_POW - 1)

count_bits = DATA_BITS - HEADER_BITS
N_max (V2/V3, UUID)   = 2^count_bits
N_max (V2/V3, base64) = 2^(OUTPUT_POW - 1)
```

### 8.2 UUID modes: максимальный N для Variant 1

Raw mode (`DATA_BITS = 128`):

| Тип | p | S | D=2 | D=3 | D=4 |
|---|---:|---:|---:|---:|---:|
| int8, S=7 | 4 | 7 | 56 | 38 | 28 |
| int16, S=15 | 5 | 15 | 51 | 34 | 26 |
| int32, S=31 | 6 | 31 | 41 | 28 | 21 |
| int64, S=31 | 7 | 31 | 40 | 27 | 20 |
| int64, S=39 | 7 | 39 | 36 | 24 | 18 |
| int64, S=63 | 7 | 63 | 24 | 16 | 12 |

UUIDv8 mode (`DATA_BITS = 122`):

| Тип | p | S | D=2 | D=3 | D=4 |
|---|---:|---:|---:|---:|---:|
| int8, S=7 | 4 | 7 | 50 | 34 | 25 |
| int16, S=15 | 5 | 15 | 45 | 30 | 23 |
| int32, S=31 | 6 | 31 | 35 | 23 | 18 |
| int64, S=31 | 7 | 31 | 34 | 23 | 17 |
| int64, S=39 | 7 | 39 | 30 | 20 | 15 |
| int64, S=63 | 7 | 63 | 18 | 12 | 9 |

### 8.3 UUID modes: максимальный N для Variants 2/3

```text
count_bits = DATA_BITS - (3p - 2 + S)
N_max = 2^count_bits
```

| Тип | p | S | Raw bits | Raw N_max | UUIDv8 bits | UUIDv8 N_max |
|---|---:|---:|---:|---:|---:|---:|
| int8, S=7 | 4 | 7 | 109 | `2^109` | 103 | `2^103` |
| int16, S=15 | 5 | 15 | 100 | `2^100` | 94 | `2^94` |
| int32, S=31 | 6 | 31 | 81 | `2^81` | 75 | `2^75` |
| int64, S=31 | 7 | 31 | 78 | `2^78` | 72 | `2^72` |
| int64, S=39 | 7 | 39 | 70 | `2^70` | 64 | `2^64` |
| int64, S=63 | 7 | 63 | 46 | `2^46` | 40 | `2^40` |

### 8.4 Размер результата base64

```text
total_bits  = (3p - 2 + S) + D * (N - 1) // V1; для V2/V3 последнее слагаемое равно 0
total_bytes = ceil(total_bits / 8)
base64_len  = ceil(total_bytes / 3) * 4
```

| Тип | p | S | D | N | Биты | Байты | Символы base64 |
|---|---:|---:|---:|---:|---:|---:|---:|
| int8 | 4 | 7 | 3 | 8 | 38 | 5 | 8 |
| int8, V2 | 4 | 7 | 1 | 8 | 17 | 3 | 4 |
| int32 | 6 | 15 | 4 | 32 | 155 | 20 | 28 |
| int64 | 7 | 39 | 4 | 20 | 134 | 17 | 24 |
| int64, V2 | 7 | 39 | 1 | 64 | 58 | 8 | 12 |
| int64 | 7 | 63 | 3 | 64 | 271 | 34 | 48 |

---

## 9. Encoding algorithm

```text
function encode(input, mode: raw|uuidv8|base64) -> uuid|string
  assert len(input) >= 1
  assert all values are all >= 0 or all <= 0

  abs_sorted = sort input by absolute value ascending
  source_num = abs(abs_sorted[0])
  is_negative = input group is non-positive and contains a negative value
  abs_values = map(abs, abs_sorted)
  deltas = differences between adjacent abs_values

  S = actual_bit_width(source_num) - 1

  if all deltas == 0 or N == 1:
      variant = 3
      D = 0
  else if all deltas == 1:
      variant = 2
      D = 1
  else:
      variant = 1
      D = max(2, actual_bit_width(max_delta))

  ELEMENT_BITS = bit width of input element type
  if mode == base64:
      OUTPUT_POW = log2(ELEMENT_BITS * 2)
      DATA_BITS = infinity
  else:
      OUTPUT_POW = min(log2(ELEMENT_BITS * 2), 7)
      DATA_BITS = 122 if mode == uuidv8 else 128

  assert S <= 2^(OUTPUT_POW - 1) - 1
  assert D <= 2^(OUTPUT_POW - 2) - 1
  if variant == 1 or mode == base64:
      assert N - 1 <= 2^(OUTPUT_POW - 1) - 1
  if mode != base64:
      assert (3*OUTPUT_POW - 2 + S) + D*(N - 1) <= DATA_BITS

  write is_negative, S, source_num, D

  if mode == base64:
      write count using OUTPUT_POW - 1 bits
      if variant == 1:
          write explicit deltas
      pad with zeros to byte boundary
      return base64_encode(bytes)
  else:
      if variant == 1:
          write count using OUTPUT_POW - 1 bits
          write explicit deltas
      else:
          write count into all remaining DATA_BITS
      return UUIDv8 with markers if mode == uuidv8 else raw 128-bit value
```

`actual_bit_width(0)` равно `1`.

---

## 10. Decoding algorithm

```text
function decode(input, mode: raw|uuidv8|base64, ELEMENT_BITS) -> numbers
  if mode == base64:
      bytes = base64_decode(input)
      reader = BitReader(bytes)
      OUTPUT_POW = log2(ELEMENT_BITS * 2)
      DATA_BITS = len(bytes) * 8
  else if mode == uuidv8:
      validate and extract 122 UUIDv8 data bits
      OUTPUT_POW = min(log2(ELEMENT_BITS * 2), 7)
      DATA_BITS = 122
  else:
      read raw 128 data bits
      OUTPUT_POW = min(log2(ELEMENT_BITS * 2), 7)
      DATA_BITS = 128

  read is_negative
  read S using OUTPUT_POW - 1 bits
  read source_num using S + 1 bits
  read D using OUTPUT_POW - 2 bits

  if D == 0:
      count = OUTPUT_POW - 1 bits in base64, otherwise all remaining bits
      deltas = repeat(0, count)
  else if D == 1:
      count = OUTPUT_POW - 1 bits in base64, otherwise all remaining bits
      deltas = repeat(1, count)
  else:
      count = read OUTPUT_POW - 1 bits
      deltas = read count values, D bits each

  reconstruct absolute values by cumulative sum
  apply sign
  validate each output value fits requested integer type
```

Decoded slice всегда отсортирован по возрастанию абсолютного значения. Исходный
порядок входа восстановить невозможно.

---

## 11. Validation errors

Encoder:

| Код | Проверка | Где применяется |
|---|---|---|
| `E_EMPTY` | Входной slice пуст | Все режимы |
| `E_MIXED_SIGN` | Есть и положительные, и отрицательные значения | Все режимы |
| `E_DELTA_OVERFLOW` | `D > 2^(OUTPUT_POW-2) - 1` | Все режимы |
| `E_COUNT_OVERFLOW` | `N-1 > 2^(OUTPUT_POW-1) - 1` | V1 в UUID; все variants в base64 |
| `E_TOTAL_OVERFLOW` | Payload не помещается в `DATA_BITS` | Только UUID modes |

Decoder:

| Код | Проверка | Где применяется |
|---|---|---|
| `D_UUID_MARKERS` | Невалидные UUID version/variant bits | UUIDv8 |
| `D_SOURCE_LEN` | `S + 1 > DATA_BITS - (3p-2)` | UUID modes |
| `D_DELTA_LEN` | `D > 2^(OUTPUT_POW-2) - 1` | Все режимы |
| `D_COUNT` | Count больше `2^(OUTPUT_POW-1) - 1` | Все режимы |
| `D_PAYLOAD_OVERFLOW` | `19 + S + D*count > DATA_BITS` для V1 | UUID modes |
| `D_INVALID_BASE64` | Строку невозможно декодировать как base64 | Base64 mode |
| `D_VALUE_OVERFLOW` | Значение не помещается в запрошенный integer type | Все режимы |

---

## 12. Edge cases

| Случай | Поведение |
|---|---|
| `N = 1` | Variant 3, `D = 0`, count = 0 |
| `source_num = 0` | `S = 0`, поле `source_num` занимает 1 бит |
| Все значения одинаковые | Variant 3 |
| Все значения `0` | `is_negative = 0`, Variant 3 |
| Есть `0` и отрицательные значения | Кодируется как неположительная группа, decoded slice может начинаться с `0` |
| `math.MinInt64` | Абсолютное значение `2^63`, валидно |
| `uint64 > math.MaxInt64` | Валидно для unsigned output; при распаковке в signed type будет overflow |
| Смесь дельт `0` и `1` | Variant 1 с `D = 2`, потому что `D = 0/1` зарезервированы |

---

## 13. Worked example

### 13.1 UUIDv8 mode — int64 slice

Input:

```text
[1_000_040, 1_000_010, 1_000_030, 1_000_000, 1_000_020]
```

После сортировки:

```text
[1_000_000, 1_000_010, 1_000_020, 1_000_030, 1_000_040]
```

Поля:

```text
source_num  = 1_000_000
is_negative = 0
deltas      = [10, 10, 10, 10]
S           = 19
D           = 4
variant     = 1
```

Bit budget:

```text
19 + S + D*(N-1) = 19 + 19 + 4*4 = 54 bits <= 122
```

Payload:

```text
is_negative          0             1 bit
S                    010011        6 bits
source_num           0xF4240       20 bits
D                    00100         5 bits
count                000100        6 bits
delta[0..3]          1010 each     16 bits total
padding              zeros
```

В UUIDv8 mode после packing в DPUID stream выставляются UUID version `8` и
variant `10`.

### 13.2 Base64 mode — int8 slice

Input: `[]int8{10, 13, 11, 12}`. Здесь `ELEMENT_BITS = 8`, поэтому
`OUTPUT_POW = 4`, ширины полей S/D/count равны 3/2/3 бита.

После сортировки: `[10, 11, 12, 13]`.

```text
source_num   = 10
is_negative  = 0
deltas       = [1, 1, 1]
actual_width = 4
S            = 3
D            = 1 // Variant 2
count        = 3
```

Проверка P5: `N - 1 = 3 <= 2^(4-1) - 1 = 7`.

```text
[0]       is_negative = 0       1 bit
[1..3]    S = 3 = 011           3 bits
[4..7]    source_num = 1010      4 bits
[8..9]    D = 1 = 01             2 bits
[10..12]  count = 3 = 011        3 bits
[13..15]  padding = 000          3 bits

bit stream: 0 011 1010 01 011 000
bytes:      00111010 01011000 = 0x3A 0x58
base64:     "Olg="
```

При декодировании `"Olg="` получаются `D = 1`, `count = 3` и результат
`[10, 11, 12, 13]`.

---

## 14. Ограничения

1. **Mixed signs не поддерживаются.** Такие наборы нужно разделять на несколько
   DPUID значений.
2. **Порядок входа не сохраняется.** Decode возвращает sorted-by-absolute-value
   sequence.
3. **Raw mode не является RFC UUID.** Используйте UUIDv8 mode для систем, которые
   валидируют UUID-структуру.
4. **Ограничение N в base64 mode.** Явное поле count ограничивает `N` значением
   `2^(OUTPUT_POW-1)`, например 64 для `int64`. Более длинный input нужно делить
   на несколько вызовов encode.
5. **Тип элемента должен быть известен decoder.** `ELEMENT_BITS` нельзя вывести
   из bit stream. Caller должен передать тип через generic parameter или отдельно.

---

## 15. Расширение OUTPUT_POW

`OUTPUT_POW = p` определяет ширины полей и capacity. Общая формула:

```text
OUTPUT_POW = min(log2(ELEMENT_BITS * 2), log2(OUTPUT_BITS))
```

Для UUID modes `OUTPUT_BITS = 128`, для base64 mode верхнего ограничения нет.

### UUID modes

| Тип | `ELEMENT_BITS` | p | S field | D field | Count | Max N (V1) | Max N (V2/V3) |
|---|---:|---:|---:|---:|---:|---:|---:|
| int8 / uint8 | 8 | 4 | 3 | 2 | 3 | 8 | `2^(DATA_BITS-10-S)` |
| int16 / uint16 | 16 | 5 | 4 | 3 | 4 | 16 | `2^(DATA_BITS-13-S)` |
| int32 / uint32 | 32 | 6 | 5 | 4 | 5 | 32 | `2^(DATA_BITS-16-S)` |
| int64 / uint64 | 64 | 7 | 6 | 5 | 6 | 64 | `2^(DATA_BITS-19-S)` |

### Base64 mode

| Тип | `ELEMENT_BITS` | p | S field | D field | Count | Max N |
|---|---:|---:|---:|---:|---:|---:|
| int8 / uint8 | 8 | 4 | 3 | 2 | 3 | 8 |
| int16 / uint16 | 16 | 5 | 4 | 3 | 4 | 16 |
| int32 / uint32 | 32 | 6 | 5 | 4 | 5 | 32 |
| int64 / uint64 | 64 | 7 | 6 | 5 | 6 | 64 |

---

## 16. Changelog

| Версия | Дата | Изменения |
|---|---|---|
| 1.2.0 | 2026-06-19 | UUID modes вычисляют `OUTPUT_POW` из типа элемента с ограничением `log2(OUTPUT_BITS)`; обновлены §§3–5, 8–10, 14–15 |
| 1.1.0 | 2026-06-19 | Добавлен base64 mode; для V2/V3 в base64 используется явное поле count |
| 1.0.0 | 2026-06-18 | Публичный релиз |
| 0.3.0 | 2026-06-18 | Добавлен Variant 3 (`D=0`), Variant 2 переведен на discriminator `D=1`, обновлены layout и algorithms |
| 0.2.0 | 2026-06-18 | Добавлен UUIDv8 mode, `S` теперь хранится как `actual_width-1` |
| 0.1.0 | 2026-06-18 | Initial draft |
