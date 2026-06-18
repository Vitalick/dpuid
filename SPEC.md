# Delta-Pack UUID (DPUID) — Specification

Russian version: [SPEC.ru.md](SPEC.ru.md)

**Version:** 0.3.0  
**Status:** Draft  
**Output target:** 128-bit UUID-compatible value

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
  making the format extensible to other output sizes (256-bit, 64-bit, etc.).
- **Three encoding variants** — Variant 1 for sequences with arbitrary small deltas;
  Variant 2 (compact) for perfectly sequential unit-step series (all deltas == 1);
  Variant 3 (compact) for sets of identical values (all deltas == 0).
- **Two UUID modes** — Raw mode (128 usable bits) and UUIDv8 mode (122 usable bits,
  RFC 9562 compliant). UUIDv8 mode is the recommended default.

The 128-bit output is structurally compatible with UUID storage fields. Raw mode makes
no claim of RFC compliance. UUIDv8 mode is fully RFC 9562 compliant.

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
| `OUTPUT_POW` | `log₂(OUTPUT_BITS)` = `7` for 128-bit output |
| `OUTPUT_BITS` | `2^OUTPUT_POW` = `128` |
| `DATA_BITS` | Usable data bits: `128` (raw) or `122` (UUIDv8) |

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

## 3. Parameters (for OUTPUT_BITS = 128, OUTPUT_POW = 7)

| Field | Formula | Concrete value |
|---|---|---|
| `SOURCE_LEN_FIELD` width | `OUTPUT_POW − 1` | **6 bits** (stores S, range 0..63) |
| `DELTA_LEN_FIELD` width | `OUTPUT_POW − 2` | **5 bits** (stores D, range 0..31) |
| `COUNT_FIELD` width (Variant 1) | `OUTPUT_POW − 1` | **6 bits** |
| Max `source_num` | `2^64 − 1` | full uint64 range |
| Max `D` (`next_nums_len_in_bits`) | `2^(OUTPUT_POW−2) − 1` | **31** |
| Max `N` in Variant 1 | `2^(OUTPUT_POW−1)` | **64** |
| Max `N` in Variant 2 or 3 | `2^(DATA_BITS − 13 − S)` | see §8 |

For a generic `OUTPUT_POW = p`:

```
SOURCE_LEN_FIELD = p − 1  bits  →  S range 0..(2^(p−1) − 1), source_num up to 2^(2^(p−1)) − 1
DELTA_LEN_FIELD  = p − 2  bits  →  D range 0..(2^(p−2) − 1)
COUNT_FIELD      = p − 1  bits  →  up to 2^(p−1) − 1 deltas (Variant 1)
```

---

## 4. UUID Modes

### 4.1 Raw mode

The full 128 bits are used for DPUID data. No UUID version or variant bits are set.
Use only in closed systems where both encoder and decoder are controlled.

```
DATA_BITS = 128
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
DATA_BITS = 122
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

## 5. Preconditions

The encoder MUST reject input that violates any of the following:

| # | Condition | Error |
|---|---|---|
| P1 | `N ≥ 1` | empty input |
| P2 | All values share the same sign: all `≥ 0` or all `≤ 0` | mixed signs |
| P3 | `S ≤ 63` (always satisfied for uint64 inputs) | source_num too wide |
| P4 | `D ≤ 2^(OUTPUT_POW−2) − 1` (= 31 for 128-bit) | max delta too large |
| P5 | `N − 1 ≤ 2^(OUTPUT_POW−1) − 1` (= 63) for Variant 1 | too many numbers |
| P6 | Total encoded bits ≤ `DATA_BITS` | overflow (see §7) |

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
Unused trailing bits are set to `0`.

### 7.1 Variant 1 — general deltas

```
Offset        Width                    Field
─────────────────────────────────────────────────────────────────────
0             1                        is_negative
1             OUTPUT_POW − 1 = 6       S = source_num_len_in_bits (stored as actual_width − 1)
7             S + 1                    source_num value  (S+1 bits)
8 + S         OUTPUT_POW − 2 = 5       D = next_nums_len_in_bits  (≥ 2 in Variant 1)
13 + S        OUTPUT_POW − 1 = 6       count of deltas (N − 1)
19 + S        D × (N − 1)              delta values (each D bits, fixed-width)
19+S+D*(N-1)  padding                  zero-fill to DATA_BITS
```

**Total data bits used:**

```
1 + 6 + (S+1) + 5 + 6 + D*(N−1)
= 19 + S + D*(N−1)   ≤ DATA_BITS
```

### 7.2 Variant 2 — all deltas == 1

```
Offset      Width                     Field
─────────────────────────────────────────────────────────────────────
0           1                         is_negative
1           OUTPUT_POW − 1 = 6        S = source_num_len_in_bits (stored as actual_width − 1)
7           S + 1                     source_num value  (S+1 bits)
8 + S       OUTPUT_POW − 2 = 5        D = 1  (Variant 2 marker)
13 + S      DATA_BITS − 13 − S        count of deltas (N − 1), all remaining bits
```

**Total data bits used:** always `DATA_BITS`.  
Count field width = `DATA_BITS − 13 − S` bits.  
All deltas are implicitly 1; no delta values are stored.

### 7.3 Variant 3 — all deltas == 0 (identical values, or N == 1)

```
Offset      Width                     Field
─────────────────────────────────────────────────────────────────────
0           1                         is_negative
1           OUTPUT_POW − 1 = 6        S = source_num_len_in_bits (stored as actual_width − 1)
7           S + 1                     source_num value  (S+1 bits)
8 + S       OUTPUT_POW − 2 = 5        D = 0  (Variant 3 marker)
13 + S      DATA_BITS − 13 − S        count of deltas (N − 1), all remaining bits
```

**Total data bits used:** always `DATA_BITS`.  
Count field width = `DATA_BITS − 13 − S` bits.  
All deltas are implicitly 0; no delta values are stored.

> Variants 2 and 3 share identical layout. They differ only in D value (1 vs 0)
> and the implicit delta applied during reconstruction.

---

## 8. Capacity Reference

### 8.1 Variant 1: maximum N

```
N_max = floor((DATA_BITS − 19 − S) / D) + 1
```

**Raw mode (DATA_BITS = 128):**

| S (stored) | actual source bits | D = 3 (max Δ=7) | D = 4 (max Δ=15) | D = 5 (max Δ=31) |
|---|---|---|---|---|
| 0  | 1  | 37 | 28 | 22 |
| 7  | 8  | 35 | 26 | 21 |
| 15 | 16 | 32 | 24 | 19 |
| 23 | 24 | 29 | 22 | 17 |
| 31 | 32 | 27 | 20 | 16 |
| 39 | 40 | 24 | 18 | 14 |
| 47 | 48 | 22 | 16 | 13 |
| 63 | 64 | 17 | 12 | 10 |

**UUIDv8 mode (DATA_BITS = 122):**

| S (stored) | actual source bits | D = 3 (max Δ=7) | D = 4 (max Δ=15) | D = 5 (max Δ=31) |
|---|---|---|---|---|
| 0  | 1  | 35 | 26 | 21 |
| 7  | 8  | 32 | 24 | 19 |
| 15 | 16 | 29 | 22 | 17 |
| 23 | 24 | 27 | 20 | 16 |
| 31 | 32 | 24 | 18 | 14 |
| 39 | 40 | 22 | 16 | 13 |
| 47 | 48 | 19 | 14 | 11 |
| 63 | 64 | 14 | 11 | 8  |

### 8.2 Variant 2: count field width and max N

```
count_field_bits = DATA_BITS − 13 − S
N_max            = 2^count_field_bits
```

| S (stored) | actual source bits | Raw count bits | Raw N_max | UUIDv8 count bits | UUIDv8 N_max |
|---|---|---|---|---|---|
| 0  | 1  | 115 | 2^115 | 109 | 2^109 |
| 31 | 32 | 84  | 2^84  | 78  | 2^78  |
| 39 | 40 | 76  | 2^76  | 70  | 2^70  |
| 47 | 48 | 68  | 2^68  | 62  | 2^62  |
| 63 | 64 | 52  | 2^52  | 46  | 2^46  |

For Variant 2, `N` is effectively unlimited for any practical application.

---

## 9. Encoding Algorithm

```
function encode(input []int64, mode: raw|uuidv8) → uuid

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

  // Step 6: determine DATA_BITS
  DATA_BITS = (mode == uuidv8) ? 122 : 128

  // Step 7: validate P3–P6
  assert S <= 63                              // always true for uint64
  assert D <= 2^(OUTPUT_POW-2) - 1           // = 31
  if variant == 1:
    assert N - 1 <= 2^(OUTPUT_POW-1) - 1     // = 63
    assert 19 + S + D*(N-1) <= DATA_BITS
  // Variants 2 and 3 always fit (count fills all remaining bits)

  // Step 8: pack bits into DATA_BITS-wide value (MSB first)
  out = BitWriter(DATA_BITS)
  out.write(is_negative,  1)
  out.write(S,            OUTPUT_POW - 1)     // 6 bits
  out.write(source_num,   S + 1)              // S+1 bits
  out.write(D,            OUTPUT_POW - 2)     // 5 bits

  if variant == 1:
    out.write(N - 1,      OUTPUT_POW - 1)     // 6 bits
    for each d in deltas:
      out.write(d,        D)
  else:
    // Variants 2 and 3: remaining bits hold N-1 (delta values implicit from D)
    out.write(N - 1,      DATA_BITS - out.position())

  // Step 9: apply UUID mode
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
function decode(u uuid, mode: raw|uuidv8) → []int64

  // Step 1: extract data bits
  if mode == uuidv8:
    data = extract_uuidv8_data(u)    // 122-bit value
    DATA_BITS = 122
  else:
    data = uint128(u)
    DATA_BITS = 128

  // Step 2: read fixed-width header fields (MSB first)
  r = BitReader(data, DATA_BITS)

  is_negative = r.read(1)
  S           = r.read(OUTPUT_POW - 1)       // stored actual_width − 1
  source_num  = r.read(S + 1)                // read S+1 bits
  D           = r.read(OUTPUT_POW - 2)

  // Step 3: read count and deltas
  if D == 0:
    // Variant 3: all deltas are 0; count fills remaining bits
    count  = r.read(DATA_BITS - r.position())
    deltas = repeat(0, count)
  else if D == 1:
    // Variant 2: all deltas are 1; count fills remaining bits
    count  = r.read(DATA_BITS - r.position())
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

| Code | Check | Description |
|---|---|---|
| `E_EMPTY` | `N < 1` | Input is empty |
| `E_MIXED_SIGN` | signs not homogeneous | Mix of positive and negative values |
| `E_DELTA_OVERFLOW` | `D > 31` | Max delta requires more than 31 bits |
| `E_COUNT_OVERFLOW` | `N−1 > 63` (V1 only) | Too many elements for Variant 1 |
| `E_TOTAL_OVERFLOW` | total bits > DATA_BITS | Encoded payload exceeds output size |

### Decoder

| Code | Check | Description |
|---|---|---|
| `D_SOURCE_LEN` | `S + 1 > DATA_BITS − 13` | source_num would consume more bits than available |
| `D_DELTA_LEN` | `D > 31` | D field value out of valid range |
| `D_COUNT` | (V1) count > 63 | count field out of range |
| `D_PAYLOAD_OVERFLOW` | `19 + S + D*count > DATA_BITS` (V1) | Payload exceeds buffer |

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

## 14. Limitations

1. **Same-sign constraint.** Numbers with mixed signs cannot be encoded. Callers must
   partition mixed sets by sign and encode each part separately.

2. **Order not preserved.** The decoded slice is always sorted ascending by absolute
   value. The original insertion order is not recoverable.

3. **Non-RFC UUID (raw mode only).** Raw mode output does not set UUID version/variant
   bits and must not be passed to systems that validate UUID structure.

4. **Fixed output size.** The format is tied to `OUTPUT_BITS`. Cross-size interop
   (e.g., reading a 256-bit DPUID with a 128-bit decoder) is undefined behaviour.

---

## 15. Extension: Generic OUTPUT_POW

To adapt the format to a different output size, replace `OUTPUT_POW = 7` with
`p = log₂(OUTPUT_BITS)`. All field widths scale accordingly:

| Output | `p` | Source-len field | Delta-len field | Count field (V1) |
|---|---|---|---|---|
| 64-bit  | 6 | 5 bits | 4 bits | 5 bits |
| 128-bit | 7 | 6 bits | 5 bits | 6 bits |
| 256-bit | 8 | 7 bits | 6 bits | 7 bits |

---

## 16. Changelog

| Version | Date | Notes |
|---|---|---|
| 0.3.0 | 2026-06-18 | Added Variant 3 (D=0, all deltas == 0); repurposed Variant 2 discriminator from D=0 to D=1; updated §2.2, §6, §7, §9, §10, §12, §14 |
| 0.2.0 | 2026-06-18 | Added UUIDv8 mode (§4); changed S encoding to actual_width−1 (§2.1) covering full uint64 range |
| 0.1.0 | 2026-06-18 | Initial draft |
