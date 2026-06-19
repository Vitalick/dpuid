package dpuid

type bitReader struct {
	bytes []byte
	pos   int
	limit int
}

func newBitReader(bytes []byte, limit int) *bitReader {
	return &bitReader{bytes: bytes, limit: limit}
}

func (r *bitReader) readBool() (bool, error) {
	v, err := r.read(1)
	return v == 1, err
}

func (r *bitReader) read(width int) (uint64, error) {
	if width < 0 || width > 64 || r.pos+width > r.limit {
		return 0, ErrPayloadOverflow
	}
	var out uint64
	for i := 0; i < width; i++ {
		out <<= 1
		if getBit(r.bytes, r.pos) {
			out |= 1
		}
		r.pos++
	}
	return out, nil
}

func (r *bitReader) readCount(width int) (uint64, error) {
	if width < 0 || r.pos+width > r.limit {
		return 0, ErrPayloadOverflow
	}
	var out uint64
	for i := 0; i < width; i++ {
		if out > uint64(math.MaxInt)/2 {
			return 0, fmt.Errorf("%w: count exceeds max int", ErrCountOverflow)
		}
		out <<= 1
		if getBit(r.bytes, r.pos) {
			out |= 1
		}
		r.pos++
	}
	return out, nil
}
