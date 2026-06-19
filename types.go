package dpuid

// Mode selects how DPUID data is embedded into the returned UUID value.
type Mode int

const (
	// ModeUUIDv8 stores 122 DPUID bits around RFC 9562 UUID version and variant
	// marker bits. This is the recommended default.
	ModeUUIDv8 Mode = iota

	// ModeRaw uses all 128 UUID bits for DPUID data and does not produce an
	// RFC-compliant UUID.
	ModeRaw
)

// Integer is any built-in signed or unsigned integer type up to 64 bits.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}
