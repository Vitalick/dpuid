# Delta-Pack UUID (DPUID) — Спецификация

English version: [SPEC.md](SPEC.md)

**Версия:** 0.3.0  
**Статус:** Draft  
**Целевой размер результата:** 128-bit UUID-compatible value

---

## 1. Обзор

Delta-Pack UUID - это самодостаточная бинарная схема кодирования, которая
упаковывает последовательность целых чисел с небольшими абсолютными разницами в
одно фиксированное 128-битное значение, совместимое с UUID-хранилищами.

Основные свойства:

- **Порядок входа не сохраняется.** Перед кодированием значения сортируются по
  возрастанию абсолютного значения.
- **Одинаковый знак.** Все входные значения должны быть sign-homogeneous: все
  неотрицательные или все неположительные. Смешанные положительные и
  отрицательные значения являются ошибкой валидации.
- **Целые числа до 64 бит.** Формат хранит абсолютные значения в диапазоне
  `0..2^64-1`, поэтому подходит для `uint64`, `int64` и меньших signed/unsigned
  integer types.
- **Самоописываемый payload.** Ширины полей выводятся из параметра `OUTPUT_POW`,
  что оставляет возможность адаптировать схему к другим размерам результата.
- **Три варианта кодирования.** Variant 1 хранит произвольные малые дельты,
  Variant 2 компактно кодирует последовательности с шагом 1, Variant 3 компактно
  кодирует одинаковые значения.
- **Два UUID mode.** Raw mode использует все 128 бит данных. UUIDv8 mode
  использует 122 бита данных и выставляет RFC 9562 UUIDv8 version/variant bits.
  UUIDv8 mode является рекомендуемым режимом по умолчанию.

Raw mode структурно помещается в UUID-sized значение, но не является
RFC-compliant UUID. UUIDv8 mode является RFC 9562 compliant.

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
| `OUTPUT_POW` | `log2(OUTPUT_BITS)`, для 128 бит равно `7` |
| `OUTPUT_BITS` | `2^OUTPUT_POW`, для этой спецификации `128` |
| `DATA_BITS` | Количество бит данных: `128` в raw mode или `122` в UUIDv8 mode |

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

Поле `S` занимает `OUTPUT_POW - 1 = 6` бит и хранит значения `0..63`, которые
представляют фактическую ширину `1..64` бит.

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

## 3. Параметры для 128-bit output

| Поле | Формула | Значение |
|---|---|---:|
| `SOURCE_LEN_FIELD` | `OUTPUT_POW - 1` | 6 bits |
| `DELTA_LEN_FIELD` | `OUTPUT_POW - 2` | 5 bits |
| `COUNT_FIELD` для Variant 1 | `OUTPUT_POW - 1` | 6 bits |
| Максимальный `source_num` | `2^64 - 1` | полный `uint64` range |
| Максимальный `D` | `2^(OUTPUT_POW-2) - 1` | 31 |
| Максимальный `N` для Variant 1 | `2^(OUTPUT_POW-1)` | 64 |

Для generic `OUTPUT_POW = p`:

```text
SOURCE_LEN_FIELD = p - 1
DELTA_LEN_FIELD  = p - 2
COUNT_FIELD      = p - 1
```

---

## 4. UUID modes

### 4.1 Raw mode

Raw mode использует все 128 бит под DPUID data.

```text
DATA_BITS = 128
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
DATA_BITS = 122
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

---

## 5. Preconditions encoder

Encoder должен отклонить вход, если нарушено любое условие:

| Код | Условие | Ошибка |
|---|---|---|
| P1 | `N >= 1` | empty input |
| P2 | Все значения одного знака: все `>= 0` или все `<= 0` | mixed signs |
| P3 | `S <= 63` | source too wide |
| P4 | `D <= 31` | max delta too large |
| P5 | Для Variant 1: `N - 1 <= 63` | too many numbers |
| P6 | Total encoded bits `<= DATA_BITS` | overflow |

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

Все поля записываются MSB first. Неиспользованные хвостовые биты заполняются
нулями.

### 7.1 Variant 1 - general deltas

```text
Offset        Width              Field
0             1                  is_negative
1             6                  S
7             S + 1              source_num
8 + S         5                  D, D >= 2
13 + S        6                  count of deltas, N - 1
19 + S        D * (N - 1)        delta values, fixed-width
```

Total data bits:

```text
19 + S + D * (N - 1) <= DATA_BITS
```

### 7.2 Variant 2 - все дельты равны 1

```text
Offset        Width              Field
0             1                  is_negative
1             6                  S
7             S + 1              source_num
8 + S         5                  D = 1
13 + S        DATA_BITS-13-S     count of deltas, N - 1
```

Дельты не записываются: каждая следующая absolute value равна предыдущей плюс 1.

### 7.3 Variant 3 - все дельты равны 0

```text
Offset        Width              Field
0             1                  is_negative
1             6                  S
7             S + 1              source_num
8 + S         5                  D = 0
13 + S        DATA_BITS-13-S     count of deltas, N - 1
```

Дельты не записываются: все reconstructed absolute values равны `source_num`.

---

## 8. Capacity reference

Для Variant 1:

```text
N_max = floor((DATA_BITS - 19 - S) / D) + 1
```

Для Variant 2 и Variant 3:

```text
count_field_bits = DATA_BITS - 13 - S
N_max = 2^count_field_bits
```

На практике compact variants позволяют хранить намного больше значений, чем
Variant 1, потому что сами дельты не занимают место.

---

## 9. Encoding algorithm

```text
function encode(input, mode) -> uuid
  assert len(input) >= 1
  assert all values are all >= 0 or all <= 0

  abs_sorted = sort input by absolute value ascending
  source_num = abs(abs_sorted[0])
  is_negative = input group is non-positive and contains a negative value
  abs_values = map(abs, abs_sorted)
  deltas = differences between adjacent abs_values

  S = actual_bit_width(source_num) - 1

  if all deltas == 0 or N == 1:
      D = 0
  else if all deltas == 1:
      D = 1
  else:
      D = max(2, actual_bit_width(max_delta))

  validate D, count, total bit budget

  write is_negative, S, source_num, D
  if D >= 2:
      write count and explicit deltas
  else:
      write count into all remaining bits

  if mode == UUIDv8:
      insert UUIDv8 markers
  else:
      return raw 128-bit value
```

`actual_bit_width(0)` равно `1`.

---

## 10. Decoding algorithm

```text
function decode(uuid, mode) -> numbers
  if mode == UUIDv8:
      validate and extract UUIDv8 data bits
  else:
      read raw 128 bits

  read is_negative
  read S
  read source_num using S + 1 bits
  read D

  if D == 0:
      count = remaining bits
      deltas = repeat(0, count)
  else if D == 1:
      count = remaining bits
      deltas = repeat(1, count)
  else:
      count = 6-bit field
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

| Код | Проверка |
|---|---|
| `E_EMPTY` | Входной slice пуст |
| `E_MIXED_SIGN` | Есть и положительные, и отрицательные значения |
| `E_DELTA_OVERFLOW` | `D > 31` |
| `E_COUNT_OVERFLOW` | Для Variant 1 слишком много дельт |
| `E_TOTAL_OVERFLOW` | Payload не помещается в `DATA_BITS` |

Decoder:

| Код | Проверка |
|---|---|
| `D_UUID_MARKERS` | В UUIDv8 mode невалидные version/variant bits |
| `D_SOURCE_LEN` | `source_num` не помещается в payload |
| `D_PAYLOAD_OVERFLOW` | Заявленный payload выходит за пределы data bits |
| `D_COUNT` | Count слишком большой для практического выделения slice |
| `D_VALUE_OVERFLOW` | Значение не помещается в requested output integer type |

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

---

## 14. Ограничения

1. **Mixed signs не поддерживаются.** Такие наборы нужно разделять на несколько
   DPUID значений.
2. **Порядок входа не сохраняется.** Decode возвращает sorted-by-absolute-value
   sequence.
3. **Raw mode не является RFC UUID.** Используйте UUIDv8 mode для систем, которые
   валидируют UUID-структуру.
4. **Фиксированный output size.** Эта спецификация описывает 128-bit target.
   Межразмерная совместимость не определена.

---

## 15. Расширение OUTPUT_POW

Чтобы адаптировать формат к другому output size, нужно заменить `OUTPUT_POW = 7`
на `p = log2(OUTPUT_BITS)`.

| Output | `p` | Source-len field | Delta-len field | Count field |
|---|---:|---:|---:|---:|
| 64-bit | 6 | 5 bits | 4 bits | 5 bits |
| 128-bit | 7 | 6 bits | 5 bits | 6 bits |
| 256-bit | 8 | 7 bits | 6 bits | 7 bits |

---

## 16. Changelog

| Версия | Дата | Изменения |
|---|---|---|
| 0.3.0 | 2026-06-18 | Добавлен Variant 3 (`D=0`), Variant 2 переведен на discriminator `D=1`, обновлены layout и algorithms |
| 0.2.0 | 2026-06-18 | Добавлен UUIDv8 mode, `S` теперь хранится как `actual_width-1` |
| 0.1.0 | 2026-06-18 | Initial draft |
