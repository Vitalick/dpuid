package dpuid

type bitWriter struct {
	bytes []byte
	pos   int
	limit int
}

func newBitWriter(limit int) *bitWriter {
	return &bitWriter{bytes: make([]byte, bytesForBits(limit)), limit: limit}
}

func (w *bitWriter) writeBool(v bool) {
	if v {
		w.write(1, 1)
		return
	}
	w.write(0, 1)
}

func (w *bitWriter) write(v uint64, width int) {
	for i := width - 1; i >= 0; i-- {
		if i < 64 && ((v>>i)&1) == 1 {
			setBit(&w.bytes, w.pos, true)
		}
		w.pos++
	}
}
