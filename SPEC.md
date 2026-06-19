# Delta-Pack UUID (DPUID) — Specification

Russian version: [SPEC.ru.md](SPEC.ru.md)

**Version:** 1.1.0  
**Status:** Ready  
**Output target:** 128-bit UUID-compatible value; or variable-length base64 string

---

## 1. Overview

Delta-Pack UUID is a self-describing binary encoding scheme that compresses a sequence
of integers with small absolute differences into a single 128-bit (UUID-compatible)
fixed-size value.

Key properties:

- **Order-agnostic encoding** — input order is not preserved; numbers are always
  re-sorted ascending by absolute value before encoding.
- **Sign-homogeneous** — all input numbers must share the same sign (all non-negative
  or all non-positive). Mixed-sign input is a precondition violation.
- **Self-describing** — field widths are derived from a single parameter `OUTPUT_POW`,
  which is computed from the input element type in every mode. This makes the format
  naturally efficient for smaller integer types (int8, int16, int32) in addition to int64.
- **Three encoding variants** — Variant 1 for sequences with arbitrary small deltas;
  Variant 2 (compact) for perfectly sequential unit-step series (all deltas == 1);
  Variant 3 (compact) for sets of identical values (all deltas == 0).
- **Two UUID modes** — Raw mode (128 usable bits) and UUIDv8 mode (122 usable bits,
  RFC 9562 compliant). UUIDv8 mode is the recommended default. Both modes derive
  `OUTPUT_POW` from the element type, capped so that `ELEMENT_BITS ≤ OUTPUT_BITS / 2`.
- **Base64 mode** — Variable-length output with no size constraint. `OUTPUT_POW`
  is derived from the element type with no upper cap on field widths.

The 128-bit output is structurally compatible with UUID storage fields. Raw mode makes
no claim of RFC compliance. UUIDv8 mode is fully RFC 9562 compliant. Base64 mode
produces a standard RFC 4648 base64 string of variable length.

---

## 2. Definitions

| Symbol | Definition |
|---|---|
| `N` | Count of input integers |
| `abs_sorted` | Input sorted ascending by `\|x\|` |
| `source_num` | `\|abs_sorted[0]\|` — absolute value of the element with the smallest absolute value |
| `is_negative` | `1` if `abs_sorted[0] < 0`, else `0` |
| `abs_values` | `[\|x\| for x in abs_sorted]` |
| `deltas` | `[abs_values[i+1] − abs_values[i] for i in 0..N−2]` — length N−1 |
| `S` | Stored value of `source_num_len_in_bits` field (see §2.1) |
| `D` | Stored value of `next_nums_len_in_bits` field (see §2.2) |
| `ELEMENT_BITS` | Bit width of one element in the input slice (8, 16, 32, or 64) |
| `OUTPUT_BITS` | Fixed output size in bits: `128` for UUID modes; not applicable for base64 |
| `OUTPUT_POW` | Derived from element type in all modes (see §3). Controls all field widths. |
| `DATA_BITS` | Usable data bits after reserved fields: `128` (raw), `122` (UUIDv8), or unbounded (base64) |

### 2.1 source_num_len_in_bits (S)

`S` is stored as **`actual_bit_width − 1`**, so the `source_num` field always
occupies at least 1 bit regardless of value.

```
actual_bit_width(x) = max(1, floor(log2(x)) + 1)
S = actual_bit_width(source_num) − 1     // stored value, range 0..63
source_num field width = S + 1 bits      // actual bits written/read
```

| source_num value | actual_bit_width | S (stored) |
|---|---|---|
| 0 | 1 | 0 |
| 1 | 1 | 0 |
| 2..3 | 2 | 1 |
| 2^63..2^64−1 | 64 | 63 |

> This encoding covers the full uint64 range (0 to 2^64 − 1).
> The field `S` occupies `OUTPUT_POW − 1 = 6` bits, storing values 0..63,
> representing actual widths 1..64.

### 2.2 next_nums_len_in_bits (D)

`D` is stored as the **exact effective bit width** of the maximum delta, with two
reserved sentinel values that act as variant discriminators.
`D` is **not** subject to the +1 rule used for S.

```
D = 0                           // Variant 3 sentinel: all deltas == 0
D = 1                           // Variant 2 sentinel: all deltas == 1
D = floor(log2(max_delta)) + 1  // Variant 1: max_delta >= 2, so D >= 2
```

| Condition | D (stored) | Variant |
|---|---|---|
| all deltas == 0 (or N == 1) | 0 | Variant 3 |
| all deltas == 1 | 1 | Variant 2 |
| max delta == 2..3 | 2 | Variant 1 |
| max delta == 4..7 | 3 | Variant 1 |
| max delta == 8..15 | 4 | Variant 1 |
| max delta == 2^30..2^31−1 | 31 | Variant 1 |

> **Why D = 0 and D = 1 are unambiguous discriminators:**  
> In Variant 1, at least one delta is ≥ 2, so `floor(log2(max_delta)) + 1 ≥ 2`,
> meaning D ≥ 2 always. The values D = 0 and D = 1 are therefore never produced
> by Variant 1 encoding and serve as safe, unambiguous variant markers.

---

## 3. Parameters

### 3.1 OUTPUT_POW derivation (all modes)

`OUTPUT_POW` is derived from `ELEMENT_BITS` in every mode using the same formula,
with an upper cap in UUID modes:

```
OUTPUT_POW = min(log₂(ELEMENT_BITS × 2), log₂(OUTPUT_BITS))
```

- **Base64 mode:** `OUTPUT_BITS = ∞`, so the cap never applies:
  `OUTPUT_POW = log₂(ELEMENT_BITS × 2)`
- **UUID modes (raw / UUIDv8):** `OUTPUT_BITS = 128`, cap = `log₂(128) = 7`:
  `OUTPUT_POW = min(log₂(ELEMENT_BITS × 2), 7)`

The cap ensures `ELEMENT_BITS ≤ OUTPUT_BITS / 2`. For all standard types (int8..int64)
with 128-bit output, the cap is 64 bits, so int64 sits exactly at the boundary and
smaller types naturally use a reduced OUTPUT_POW with more compact fields.

#### OUTPUT_POW table

| Element type | ELEMENT_BITS | Base64 OUTPUT_POW | UUID OUTPUT_POW (128-bit, cap=7) |
|---|---|---|---|
| int8 / uint8   | 8  | 4 | **4** |
| int16 / uint16 | 16 | 5 | **5** |
| int32 / uint32 | 32 | 6 | **6** |
| int64 / uint64 | 64 | 7 | **7** (at cap) |

### 3.2 Field widths (function of OUTPUT_POW = p)

| Field | Formula | p=4 (int8) | p=5 (int16) | p=6 (int32) | p=7 (int64) |
|---|---|---|---|---|---|
| `SOURCE_LEN_FIELD` | `p − 1` | 3 bits | 4 bits | 5 bits | 6 bits |
| `DELTA_LEN_FIELD` | `p − 2` | 2 bits | 3 bits | 4 bits | 5 bits |
| `COUNT_FIELD` | `p − 1` | 3 bits | 4 bits | 5 bits | 6 bits |
| Max S (stored) | `2^(p−1) − 1` | 7 | 15 | 31 | 63 |
| Max source_num bits | `2^(p−1)` | 8 | 16 | 32 | 64 |
| Max D | `2^(p−2) − 1` | 3 | 7 | 15 | 31 |
| Max Δ | `2^(2^(p−2)) − 1` | 7 | 127 | 32767 | 2³¹−1 |
| Max N (V1 + base64 V2/V3) | `2^(p−1)` | 8 | 16 | 32 | 64 |
| `HEADER_BITS` | `3p − 2 + S` | `10+S` | `13+S` | `16+S` | `19+S` |

---

## 4. UUID Modes

### 4.1 Raw mode

The full 128 bits are used for DPUID data. No UUID version or variant bits are set.
Use only in closed systems where both encoder and decoder are controlled.

```
OUTPUT_BITS = 128
DATA_BITS   = 128
OUTPUT_POW  = min(log₂(ELEMENT_BITS × 2), 7)   // capped at 7
```

### 4.2 UUIDv8 mode (recommended)

RFC 9562 defines **version 8** for custom UUID formats. Two fields are fixed:

| UUID bits | Field | Value |
|---|---|---|
| 48–51 | version | `0x8` (binary `1000`) |
| 64–65 | variant | `0b10` |

These 6 bits are **not part of DPUID data**. The remaining 122 bits carry all encoded
information.

```
OUTPUT_BITS = 128
DATA_BITS   = 122
OUTPUT_POW  = min(log₂(ELEMENT_BITS × 2), 7)   // capped at 7
```

DPUID data bits are mapped around the reserved positions:

```
UUID bit range   DPUID data bit range
─────────────────────────────────────
0  .. 47    →   0  .. 47     (48 bits)
48 .. 51    →   version = 0x8  (not data)
52 .. 63    →   48 .. 59     (12 bits)
64 .. 65    →   variant = 10   (not data)
66 .. 127   →   60 .. 121    (62 bits)
```

#### Encode (insert UUID markers)

```
// dpuid_bits: 122-bit value, packed MSB-first
top48 = dpuid_bits >> 74                       // DPUID bits 121..74
mid12 = (dpuid_bits >> 62) & 0xFFF             // DPUID bits 73..62
bot62 = dpuid_bits & 0x3FFFFFFFFFFFFFFFFFFFFFFF // DPUID bits 61..0

uuid  = (top48 << 80)
      | (uint128(0x8) << 76)   // version 8
      | (mid12 << 64)
      | (uint128(0x2) << 62)   // variant 10
      | (bot62 << 0)
```

#### Decode (extract DPUID bits)

```
top48 = uuid >> 80
mid12 = (uuid >> 64) & 0xFFF
bot62 = uuid & 0x3FFFFFFFFFFFFFFFFFFFFFFF

dpuid_bits = (top48 << 74) | (mid12 << 62) | bot62
```

---

### 4.3 Base64 mode

Base64 mode produces a **variable-length byte array** encoded as a standard RFC 4648
base64 string. There is no fixed output size constraint.

#### OUTPUT_POW derivation

`OUTPUT_POW` is derived from `ELEMENT_BITS` with no upper cap (see §3.1):

```
OUTPUT_POW = log₂(ELEMENT_BITS × 2)
```

| Element type | ELEMENT_BITS | OUTPUT_POW | S field | D field | Count field | Max N | Max delta bits |
|---|---|---|---|---|---|---|---|
| int8 / uint8   | 8  | 4 | 3 bits | 2 bits | 3 bits | 8  | 3 (max Δ = 7)    |
| int16 / uint16 | 16 | 5 | 4 bits | 3 bits | 4 bits | 16 | 7 (max Δ = 127)  |
| int32 / uint32 | 32 | 6 | 5 bits | 4 bits | 5 bits | 32 | 15 (max Δ = 32767) |
| int64 / uint64 | 64 | 7 | 6 bits | 5 bits | 6 bits | 64 | 31 (max Δ = 2^31−1) |

#### Differences from UUID modes

1. **No DATA_BITS cap.** Total bit count is not constrained; precondition P6 does not apply.
2. **Explicit COUNT_FIELD for all variants.** Variants 2 and 3 use the same `OUTPUT_POW − 1`
   bit COUNT_FIELD as Variant 1 (cannot use "all remaining bits" without a fixed end).
3. **Byte padding.** After all data bits, zero-fill to the next byte boundary before base64 encoding.
4. **No UUID markers.** No version/variant bits are injected.

#### Output size

```
total_bits  = HEADER_BITS + D*(N−1)          // V1: HEADER = 3p−2+S; V2/V3: D*(N−1) = 0
            = (3*OUTPUT_POW − 2 + S) + D*(N−1)
total_bytes = ceil(total_bits / 8)
base64_len  = ceil(total_bytes / 3) * 4      // with standard padding
```

#### Typical sizes (int64, OUTPUT_POW = 7)

| Scenario | S | D | N | total bits | bytes | base64 chars |
|---|---|---|---|---|---|---|
| 20 nums, Δ≤15 | 40 | 4 | 20 | 19+40+4×19 = 135 | 17 | 24 |
| 20 nums, Δ=1 (V2) | 40 | 1 | 20 | 19+40 = 59 | 8 | 12 |
| 64 nums, Δ≤7 | 32 | 3 | 64 | 19+32+3×63 = 240 | 30 | 40 |
| 1 num (V3) | 20 | 0 | 1 | 19+20 = 39 | 5 | 8 |

---

### 4.4 Byte payload API and fixed-size base64

The implementation exposes the DPUID bit stream as a reusable byte payload:

```go
PackBytes[T](values, outputBits) -> []byte
UnpackBytes[T](data, inputBits) -> []T
```

A zero bit size selects the variable-length layout from §4.3. A positive bit
size selects a fixed payload: Variant 1 uses an explicit `OUTPUT_POW - 1` count,
while Variants 2 and 3 use all bits remaining after `D` for count, as in UUID
modes. Unwritten bits are zero and the returned slice has `ceil(bits / 8)` bytes.

UUIDv8 and raw UUID reuse this codec with 122 and 128 data bits respectively.
Base64 wrappers encode the resulting bytes using RFC 4648. The unsized base64
API passes zero; the sized base64 API passes its requested bit limit.

For a positive arbitrary bit limit, the field-width cap is based on the smallest
power-of-two container that covers the limit. Thus both 122 and 128 data bits use
cap 7.

---

## 5. Preconditions

The encoder MUST reject input that violates any of the following.
`OUTPUT_POW` and its derived limits depend on the element type (see §3).

| # | Condition | Applies to | Error |
|---|---|---|---|
| P1 | `N ≥ 1` | all modes | empty input |
| P2 | All values share the same sign: all `≥ 0` or all `≤ 0` | all modes | mixed signs |
| P3 | `S ≤ 2^(OUTPUT_POW−1) − 1` — satisfied automatically when the source_num fits in `ELEMENT_BITS` | all modes | source_num too wide |
| P4 | `D ≤ 2^(OUTPUT_POW−2) − 1` | all modes | max delta too large for the element type |
| P5 | `N − 1 ≤ 2^(OUTPUT_POW−1) − 1` | V1 (UUID modes); **all variants** (base64) | too many numbers |
| P6 | `HEADER_BITS + D*(N−1) ≤ DATA_BITS`, i.e. `(3p−2+S) + D*(N−1) ≤ DATA_BITS` | UUID modes only | overflow |

---

## 6. Variant Selection

```
IF all values in deltas == 0  OR  N == 1:
    variant = 3          // identical values or single element; D stored as 0
ELSE IF all values in deltas == 1:
    variant = 2          // unit-step series; D stored as 1
ELSE:
    variant = 1          // general case; D = floor(log2(max_delta)) + 1  (≥ 2)
```

The decoder reads the `DELTA_LEN_FIELD` (D) and branches:

| D value | Variant | Meaning |
|---|---|---|
| `0` | Variant 3 | all deltas == 0; count in remaining bits |
| `1` | Variant 2 | all deltas == 1; count in remaining bits |
| `≥ 2` | Variant 1 | general deltas; explicit count + delta fields follow |

---

## 7. Bit Layout

All fields are written **MSB first** (big-endian bit order) into the data bit stream.

### 7.1 Variant 1 — general deltas

Layout is identical across all modes. UUID modes zero-fill the tail to `DATA_BITS`;
base64 mode zero-fills to the next byte boundary.

```
Offset        Width                    Field
─────────────────────────────────────────────────────────────────────
0             1                        is_negative
1             OUTPUT_POW − 1 = 6       S = source_num_len_in_bits (stored as actual_width − 1)
7             S + 1                    source_num value  (S+1 bits)
8 + S         OUTPUT_POW − 2 = 5       D = next_nums_len_in_bits  (≥ 2 in Variant 1)
13 + S        OUTPUT_POW − 1 = 6       count of deltas (N − 1)
19 + S        D × (N − 1)              delta values (each D bits, fixed-width)
19+S+D*(N-1)  padding                  UUID: zero-fill to DATA_BITS / Base64: zero-fill to byte
```

**Total data bits used:**

```
1 + 6 + (S+1) + 5 + 6 + D*(N−1)
= 19 + S + D*(N−1)   ≤ DATA_BITS  (UUID modes)
```

### 7.2 Variant 2 — all deltas == 1

**UUID modes** and **base64 mode** differ in how the count is stored.

#### UUID modes (raw / UUIDv8)

```
Offset      Width                     Field
─────────────────────────────────────────────────────────────────────
0           1                         is_negative
1           OUTPUT_POW − 1 = 6        S = source_num_len_in_bits (stored as actual_width − 1)
7           S + 1                     source_num value  (S+1 bits)
8 + S       OUTPUT_POW − 2 = 5        D = 1  (Variant 2 marker)
13 + S      DATA_BITS − 13 − S        count of deltas (N − 1), all remaining bits
```

Total bits = `DATA_BITS`. Count field = `DATA_BITS − 13 − S` bits.

#### Base64 mode

```
Offset      Width                     Field
─────────────────────────────────────────────────────────────────────
0           1                         is_negative
1           OUTPUT_POW − 1 = 6        S = source_num_len_in_bits (stored as actual_width − 1)
7           S + 1                     source_num value  (S+1 bits)
8 + S       OUTPUT_POW − 2 = 5        D = 1  (Variant 2 marker)
13 + S      OUTPUT_POW − 1 = 6        count of deltas (N − 1)  [explicit, same width as V1]
19 + S      padding                   zero-fill to next byte boundary
```

Total bits = `19 + S`, padded to byte. Count field = `OUTPUT_POW − 1` bits.

All deltas are implicitly 1; no delta values are stored in either sub-mode.

### 7.3 Variant 3 — all deltas == 0 (identical values, or N == 1)

Identical structure to Variant 2 in both sub-modes; only `D` differs (0 vs 1).

#### UUID modes

```
Offset      Width                     Field
─────────────────────────────────────────────────────────────────────
0           1                         is_negative
1           OUTPUT_POW − 1 = 6        S (stored as actual_width − 1)
7           S + 1                     source_num value
8 + S       OUTPUT_POW − 2 = 5        D = 0  (Variant 3 marker)
13 + S      DATA_BITS − 13 − S        count of deltas (N − 1), all remaining bits
```

#### Base64 mode

```
Offset      Width                     Field
─────────────────────────────────────────────────────────────────────
0           1                         is_negative
1           OUTPUT_POW − 1 = 6        S (stored as actual_width − 1)
7           S + 1                     source_num value
8 + S       OUTPUT_POW − 2 = 5        D = 0  (Variant 3 marker)
13 + S      OUTPUT_POW − 1 = 6        count of deltas (N − 1)  [explicit, same width as V1]
19 + S      padding                   zero-fill to next byte boundary
```

> Variants 2 and 3 share identical layout within each sub-mode. They differ only in
> D value (1 vs 0) and the implicit delta applied during reconstruction.

---

## 8. Capacity Reference

### 8.1 General formula

```
HEADER_BITS = 3 × OUTPUT_POW − 2 + S       // same for all modes and variants

// Variant 1:
N_max (V1)  = floor((DATA_BITS − HEADER_BITS) / D) + 1   // UUID modes
N_max (V1)  = 2^(OUTPUT_POW − 1)                          // base64 (COUNT_FIELD limit)

// Variants 2 and 3 — UUID modes:
count_bits  = DATA_BITS − HEADER_BITS
N_max (V2/V3, UUID) = 2^count_bits                        // fills all remaining bits

// Variants 2 and 3 — base64 mode:
N_max (V2/V3, base64) = 2^(OUTPUT_POW − 1)               // same as V1
```

### 8.2 UUID modes — Variant 1 maximum N by element type

**Raw mode (DATA_BITS = 128):**

| Element type | p | S | D = 2 (max Δ=3) | D = 3 (max Δ=7) | D = 4 (max Δ=15) |
|---|---|---|---|---|---|
| int8  | 4 | 0..7  | floor((118−S)/2)+1 | floor((118−S)/3)+1 | floor((118−S)/4)+1 |
| int8, S=7  | 4 | 7 | 56 | 38 | 28 |
| int16 | 5 | 0..15 | floor((115−S)/2)+1 | floor((115−S)/3)+1 | floor((115−S)/4)+1 |
| int16, S=15 | 5 | 15 | 51 | 34 | 26 |
| int32 | 6 | 0..31 | floor((112−S)/2)+1 | floor((112−S)/3)+1 | floor((112−S)/4)+1 |
| int32, S=31 | 6 | 31 | 41 | 28 | 21 |
| int64 | 7 | 0..63 | floor((109−S)/2)+1 | floor((109−S)/3)+1 | floor((109−S)/4)+1 |
| int64, S=31 | 7 | 31 | 40 | 27 | 20 |
| int64, S=39 | 7 | 39 | 36 | 24 | 18 |
| int64, S=63 | 7 | 63 | 24 | 16 | 12 |

**UUIDv8 mode (DATA_BITS = 122):**

| Element type | p | S | D = 2 | D = 3 | D = 4 |
|---|---|---|---|---|---|
| int8, S=7  | 4 | 7  | 50 | 34 | 25 |
| int16, S=15 | 5 | 15 | 45 | 30 | 23 |
| int32, S=31 | 6 | 31 | 35 | 23 | 18 |
| int64, S=31 | 7 | 31 | 34 | 23 | 17 |
| int64, S=39 | 7 | 39 | 30 | 20 | 15 |
| int64, S=63 | 7 | 63 | 18 | 12 | 9  |

### 8.3 UUID modes — Variants 2 / 3 maximum N

```
count_bits = DATA_BITS − (3p − 2 + S)
N_max      = 2^count_bits
```

| Element type | p | S | Raw count bits | Raw N_max | UUIDv8 count bits | UUIDv8 N_max |
|---|---|---|---|---|---|---|
| int8,  S=7  | 4 | 7  | 109 | 2^109 | 103 | 2^103 |
| int16, S=15 | 5 | 15 | 100 | 2^100 | 94  | 2^94  |
| int32, S=31 | 6 | 31 | 81  | 2^81  | 75  | 2^75  |
| int64, S=31 | 7 | 31 | 78  | 2^78  | 72  | 2^72  |
| int64, S=39 | 7 | 39 | 70  | 2^70  | 64  | 2^64  |
| int64, S=63 | 7 | 63 | 46  | 2^46  | 40  | 2^40  |

For UUID Variants 2 and 3, N is effectively unlimited for any practical application.

### 8.4 Base64 mode output sizes

```
total_bits  = (3p − 2 + S) + D*(N−1)    // V1; V2/V3: D*(N−1) = 0
total_bytes = ceil(total_bits / 8)
base64_len  = ceil(total_bytes / 3) × 4
```

| Element type | p | S | D | N | total bits | bytes | base64 chars |
|---|---|---|---|---|---|---|---|
| int8,  S=7  | 4 | 7  | 3 | 8 | 17+21=38 | 5 | 8 |
| int8,  S=7  | 4 | 7  | 1 (V2) | 8 | 17 | 3 | 4 |
| int32, S=15 | 6 | 15 | 4 | 32 | 16+15+4×31=155 | 20 | 28 |
| int64, S=39 | 7 | 39 | 4 | 20 | 19+39+4×19=134 | 17 | 24 |
| int64, S=39 | 7 | 39 | 1 (V2) | 64 | 19+39=58 | 8 | 12 |
| int64, S=63 | 7 | 63 | 3 | 64 | 19+63+3×63=271 | 34 | 48 |

---

## 9. Encoding Algorithm

```
function encode(input []int64, mode: raw|uuidv8|base64) → uuid|string

  // Step 1: validate P1–P2
  assert len(input) >= 1
  assert all_same_sign(input)

  // Step 2: sort by absolute value ascending
  abs_sorted = sort_by_abs(input)

  // Step 3: derive base fields
  source_num  = abs(abs_sorted[0])
  is_negative = (abs_sorted[0] < 0) ? 1 : 0
  abs_values  = map(abs, abs_sorted)
  deltas      = [abs_values[i+1] - abs_values[i] for i in 0..N-2]

  // Step 4: compute field values
  actual_width = max(1, floor(log2(source_num)) + 1)
  S = actual_width - 1                        // stored as actual_width − 1

  max_delta = max(deltas, default=0)

  // Step 5: select variant
  if all_equal(deltas, 0) or N == 1:
    variant = 3
    D = 0
  else if all_equal(deltas, 1):
    variant = 2
    D = 1
  else:
    variant = 1
    D = floor(log2(max_delta)) + 1            // max_delta >= 2 here, so D >= 2

  // Step 6: determine OUTPUT_POW and DATA_BITS
  if mode == base64:
    OUTPUT_POW = log2(ELEMENT_BITS * 2)          // no cap
    DATA_BITS  = ∞
  else:
    // UUID modes: derive OUTPUT_POW from element type, capped at 7
    OUTPUT_POW = min(log2(ELEMENT_BITS * 2), 7)
    DATA_BITS  = (mode == uuidv8) ? 122 : 128

  // Step 7: validate P3–P6
  assert S <= 2^(OUTPUT_POW-1) - 1           // P3: always true for the element type
  assert D <= 2^(OUTPUT_POW-2) - 1           // P4
  if mode == base64:
    assert N - 1 <= 2^(OUTPUT_POW-1) - 1     // P5: applies to ALL variants in base64
  else:
    if variant == 1:
      assert N - 1 <= 2^(OUTPUT_POW-1) - 1   // P5: V1 only in UUID modes
    assert 19 + S + D*(N-1) <= DATA_BITS      // P6: UUID modes only

  // Step 8: pack bits (MSB first)
  out = BitWriter()
  out.write(is_negative,  1)
  out.write(S,            OUTPUT_POW - 1)
  out.write(source_num,   S + 1)
  out.write(D,            OUTPUT_POW - 2)

  if mode == base64:
    // All variants use explicit COUNT_FIELD
    out.write(N - 1,      OUTPUT_POW - 1)
    if variant == 1:
      for each d in deltas:
        out.write(d,      D)
    // V2/V3: no delta values written; delta is implicit from D
    out.pad_to_byte()                         // zero-fill to next byte boundary
    return base64_encode(out.to_bytes())

  else:
    // UUID modes
    if variant == 1:
      out.write(N - 1,    OUTPUT_POW - 1)
      for each d in deltas:
        out.write(d,      D)
    else:
      // V2/V3: count fills all remaining bits
      out.write(N - 1,    DATA_BITS - out.position())

    if mode == uuidv8:
      return insert_uuidv8_markers(out.to_uint122())
    else:
      return out.to_uint128()
```

`actual_bit_width(x)`:

```
if x == 0: return 1       // special case: store as a single 0 bit
return floor(log2(x)) + 1
```

---

## 10. Decoding Algorithm

```
function decode(input: uuid|string, mode: raw|uuidv8|base64) → []int64

  // Step 1: extract data bits and determine OUTPUT_POW
  if mode == base64:
    raw_bytes   = base64_decode(input)
    r           = BitReader(raw_bytes)
    OUTPUT_POW  = log2(ELEMENT_BITS * 2)     // must be known by caller (element type)
    DATA_BITS   = len(raw_bytes) * 8
  else if mode == uuidv8:
    r           = BitReader(extract_uuidv8_data(input), 122)
    OUTPUT_POW  = min(log2(ELEMENT_BITS * 2), 7)   // capped at 7
    DATA_BITS   = 122
  else:
    r           = BitReader(uint128(input), 128)
    OUTPUT_POW  = min(log2(ELEMENT_BITS * 2), 7)   // capped at 7
    DATA_BITS   = 128

  // Step 2: read fixed-width header fields (MSB first)
  is_negative = r.read(1)
  S           = r.read(OUTPUT_POW - 1)       // stored actual_width − 1
  source_num  = r.read(S + 1)                // read S+1 bits
  D           = r.read(OUTPUT_POW - 2)

  // Step 3: read count and deltas
  if D == 0:
    // Variant 3: all deltas are 0
    if mode == base64:
      count = r.read(OUTPUT_POW - 1)         // explicit COUNT_FIELD
    else:
      count = r.read(DATA_BITS - r.position()) // all remaining bits
    deltas = repeat(0, count)

  else if D == 1:
    // Variant 2: all deltas are 1
    if mode == base64:
      count = r.read(OUTPUT_POW - 1)         // explicit COUNT_FIELD
    else:
      count = r.read(DATA_BITS - r.position()) // all remaining bits
    deltas = repeat(1, count)

  else:
    // Variant 1: D >= 2
    count  = r.read(OUTPUT_POW - 1)
    deltas = [r.read(D) for _ in 0..count-1]

  // Step 4: reconstruct absolute values
  abs_values = [source_num]
  cur = source_num
  for d in deltas:
    cur += d
    abs_values.append(cur)

  // Step 5: apply sign
  if is_negative:
    return [-x for x in abs_values]
  else:
    return abs_values
```

> The decoded slice is always sorted ascending by absolute value.
> The original insertion order is **not** recoverable.

---

## 11. Validation Rules

### Encoder

| Code | Check | Applies to | Description |
|---|---|---|---|
| `E_EMPTY` | `N < 1` | all | Input is empty |
| `E_MIXED_SIGN` | signs not homogeneous | all | Mix of positive and negative values |
| `E_DELTA_OVERFLOW` | `D > 2^(OUTPUT_POW−2) − 1` | all | Max delta too wide for the element type |
| `E_COUNT_OVERFLOW` | `N−1 > 2^(OUTPUT_POW−1) − 1` | V1 (UUID); all variants (base64) | Too many elements |
| `E_TOTAL_OVERFLOW` | total bits > DATA_BITS | UUID modes only | Encoded payload exceeds output size |

### Decoder

| Code | Check | Applies to | Description |
|---|---|---|---|
| `D_SOURCE_LEN` | `S + 1 > DATA_BITS − (3p−2)` | UUID modes | source_num would overrun buffer |
| `D_DELTA_LEN` | `D > 2^(OUTPUT_POW−2) − 1` | all | D field value out of valid range |
| `D_COUNT` | count > `2^(OUTPUT_POW−1) − 1` | all | count field out of range |
| `D_PAYLOAD_OVERFLOW` | `19 + S + D*count > DATA_BITS` (V1) | UUID modes | Payload exceeds buffer |
| `D_INVALID_BASE64` | base64 decode fails | base64 | Malformed base64 string |

---

## 12. Edge Cases

| Case | Behaviour |
|---|---|
| `N = 1` | Variant 3 (D = 0); count = 0; decodes to single-element slice |
| `source_num = 0` | S = 0; source_num field occupies 1 bit (writes/reads `0`) |
| All values equal | All deltas = 0 → Variant 3 (D = 0); count = N−1; reconstructed as N copies of source_num |
| All values = 0, N > 1 | is_negative = 0; S = 0; all deltas = 0 → Variant 3 |
| `INT64_MIN` as source_num | `\|INT64_MIN\|` = 2^63; actual_width = 64; S = 63 → **valid**, within uint64 range |

---

## 13. Worked Example

**Input:** `[1_000_040, 1_000_010, 1_000_030, 1_000_000, 1_000_020]`  
All positive. Mode: UUIDv8. DATA_BITS = 122.

**Step 1 — Sort by absolute value:**
```
[1_000_000, 1_000_010, 1_000_020, 1_000_030, 1_000_040]
```

**Step 2 — Fields:**
```
source_num      = 1_000_000  (= 0xF4240)
is_negative     = 0
deltas          = [10, 10, 10, 10]
actual_width    = 20   (2^19 < 1_000_000 < 2^20)
S               = 19   (stored as actual_width − 1)
max_delta       = 10
D               = 4    (2^3 < 10 < 2^4)
variant         = 1
```

**Step 3 — Bit budget:**
```
19 + S + D*(N-1) = 19 + 19 + 4*4 = 54 bits  ≤ 122  ✓
```

**Step 4 — Packed layout (into 122-bit DPUID stream):**
```
[0]         is_negative          = 0              →  1 bit
[1..6]      S = 19               = 010011         →  6 bits
[7..26]     source_num (20 bits) = 0xF4240        → 20 bits
[27..31]    D = 4                = 00100          →  5 bits
[32..37]    count = 4            = 000100         →  6 bits
[38..41]    delta[0] = 10        = 1010           →  4 bits
[42..45]    delta[1] = 10        = 1010           →  4 bits
[46..49]    delta[2] = 10        = 1010           →  4 bits
[50..53]    delta[3] = 10        = 1010           →  4 bits
[54..121]   padding              = 0...0          → 68 bits
```

**Step 5 — Insert UUIDv8 markers (122 → 128 bits):**
```
UUID bits 0-47   ← DPUID bits 0-47
UUID bits 48-51  ← 0x8  (version 8)
UUID bits 52-63  ← DPUID bits 48-59
UUID bits 64-65  ← 0b10 (variant)
UUID bits 66-127 ← DPUID bits 60-121
```

---

### 13.2 Base64 mode — int8 slice

**Input:** `[]int8{10, 13, 11, 12}` (4 elements, ELEMENT_BITS = 8)

**Step 1 — OUTPUT_POW:**
```
OUTPUT_BITS = 8 × 2 = 16
OUTPUT_POW  = 4
Field widths: S=3 bits, D=2 bits, count=3 bits
```

**Step 2 — Sort by absolute value:**
```
[10, 11, 12, 13]
```

**Step 3 — Fields:**
```
source_num   = 10
is_negative  = 0
deltas       = [1, 1, 1]
actual_width = 4   (2^3 < 10 < 2^4)
S            = 3   (actual_width − 1)
all deltas == 1 → variant = 2, D = 1
```

**Step 4 — Validate P5 (base64 applies to all variants):**
```
N − 1 = 3 ≤ 2^(4−1) − 1 = 7  ✓
```

**Step 5 — Packed layout:**
```
[0]      is_negative = 0     →  1 bit
[1..3]   S = 3        = 011  →  3 bits
[4..7]   source_num (4 bits) = 1010  →  4 bits
[8..9]   D = 1        = 01   →  2 bits
[10..12] count = 3    = 011  →  3 bits    ← explicit, not "all remaining"
[13..15] padding = 000        →  3 bits   ← zero-fill to byte boundary
```

**Total: 16 bits = 2 bytes**

```
bit stream: 0 011 1010 01 011 000
bytes:      0011 1010  0101 1000  →  0x3A 0x58
base64:     base64("0x3A58")      →  "Olg="
```

**Step 6 — Decode "Olg=":**
```
bytes      = 0x3A 0x58
bit stream = 0 011 1010 01 011 000
is_negative = 0
S           = 3  → source_num width = 4
source_num  = 1010₂ = 10
D           = 01₂ = 1  → Variant 2
count       = 011₂ = 3
deltas      = [1, 1, 1]
result      = [10, 11, 12, 13]  ✓
```

---

## 14. Limitations

1. **Same-sign constraint.** Numbers with mixed signs cannot be encoded. Callers must
   partition mixed sets by sign and encode each part separately.

2. **Order not preserved.** The decoded slice is always sorted ascending by absolute
   value. The original insertion order is not recoverable.

3. **Non-RFC UUID (raw mode only).** Raw mode output does not set UUID version/variant
   bits and must not be passed to systems that validate UUID structure.

4. **N cap in base64 mode.** Variants 2/3 in base64 mode use a fixed COUNT_FIELD,
   capping N at `2^(OUTPUT_POW−1)` (e.g. 64 for int64). For larger sequences,
   callers must split input into multiple encode calls.

5. **Element type must be known at decode time.** The decoder cannot infer `ELEMENT_BITS`
   from the bit stream alone (in any mode); the caller must supply it (via the generic
   type parameter, or out-of-band). UUID modes use `OUTPUT_POW = min(log₂(ELEMENT_BITS×2), 7)`;
   base64 mode uses `OUTPUT_POW = log₂(ELEMENT_BITS×2)`.

---

## 15. Extension: Generic OUTPUT_POW

`OUTPUT_POW = p` controls all field widths and capacity. The unified derivation rule is:

```
OUTPUT_POW = min(log₂(ELEMENT_BITS × 2), log₂(OUTPUT_BITS))
```

where `OUTPUT_BITS = 128` for UUID modes and `OUTPUT_BITS = ∞` (no cap) for base64.

### UUID modes — field widths by element type

| Element type | ELEMENT_BITS | `p` | S field | D field | Count | Max N (V1) | Max N (V2/3) |
|---|---|---|---|---|---|---|---|
| int8 / uint8   | 8  | 4 | 3 bits | 2 bits | 3 bits | 8 | 2^(DATA_BITS−10−S) |
| int16 / uint16 | 16 | 5 | 4 bits | 3 bits | 4 bits | 16 | 2^(DATA_BITS−13−S) |
| int32 / uint32 | 32 | 6 | 5 bits | 4 bits | 5 bits | 32 | 2^(DATA_BITS−16−S) |
| int64 / uint64 | 64 | 7 | 6 bits | 5 bits | 6 bits | 64 | 2^(DATA_BITS−19−S) |

### Base64 mode — field widths by element type (no cap)

| Element type | ELEMENT_BITS | `p` | S field | D field | Count | Max N (all variants) |
|---|---|---|---|---|---|---|
| int8 / uint8   | 8  | 4 | 3 bits | 2 bits | 3 bits | 8  |
| int16 / uint16 | 16 | 5 | 4 bits | 3 bits | 4 bits | 16 |
| int32 / uint32 | 32 | 6 | 5 bits | 4 bits | 5 bits | 32 |
| int64 / uint64 | 64 | 7 | 6 bits | 5 bits | 6 bits | 64 |

---

## 16. Changelog

| Version | Date | Notes |
|---|---|---|
| 1.1.0 | 2026-06-19 | UUID modes now derive OUTPUT_POW from element type, capped at log₂(OUTPUT_BITS); unified derivation formula `min(log₂(ELEMENT_BITS×2), log₂(OUTPUT_BITS))` in §3.1; updated §4.1–4.2, §5, §8, §9, §10, §14, §15 |
| 1.1.0 | 2026-06-19 | Added base64 mode (§4.3, §8.4, §13.2); OUTPUT_POW derived from element type in base64; V2/V3 explicit COUNT_FIELD in base64 |
| 1.0.0 | 2026-06-18 | Public release |
| 0.3.0 | 2026-06-18 | Added Variant 3 (D=0); Variant 2 discriminator changed to D=1 |
| 0.2.0 | 2026-06-18 | Added UUIDv8 mode; S encoding changed to actual_width−1 |
| 0.1.0 | 2026-06-18 | Initial draft |
