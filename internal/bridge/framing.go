package bridge

import (
	"encoding/binary"
	"errors"
	"io"
)

const MaxMessage = 65536

func ReadFrame(r io.Reader) ([]byte, error) {
	var size uint32
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return nil, err
	}
	if size == 0 || size > MaxMessage {
		return nil, errors.New("invalid native message size")
	}
	b := make([]byte, size)
	_, err := io.ReadFull(r, b)
	return b, err
}
func WriteFrame(w io.Writer, b []byte) error {
	if len(b) == 0 || len(b) > MaxMessage {
		return errors.New("invalid native message size")
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(len(b))); err != nil {
		return err
	}
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}
