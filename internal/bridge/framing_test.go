package bridge

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNativeFramingRoundTrip(t *testing.T) {
	var b bytes.Buffer
	msg := []byte(`{"ok":true}`)
	if err := WriteFrame(&b, msg); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFrame(&b)
	if err != nil || !bytes.Equal(got, msg) {
		t.Fatal(string(got), err)
	}
}
func TestFramingRejectsOversizeAndTruncation(t *testing.T) {
	for _, size := range []uint32{0, MaxMessage + 1, 10} {
		var b bytes.Buffer
		_ = binary.Write(&b, binary.LittleEndian, size)
		if _, err := ReadFrame(&b); err == nil {
			t.Fatal(size)
		}
	}
}
