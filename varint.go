// Package monblob provides Monero varint encoding/decoding.
// Monero uses a 7-bit-per-byte varint with MSB as continuation flag.
package monblob

import (
	"errors"
	"io"
)

// ErrOverflow indicates that the varint value exceeds 64 bits.
var ErrOverflow = errors.New("varint overflow")

// EncodeVarintToWriter encodes a uint64 as a Monero-style varint and writes it to w.
// Each byte uses 7 bits for data, with the MSB set to 1 for continuation.
func EncodeVarintToWriter(w io.Writer, val uint64) error {
	for val >= 0x80 {
		b := byte(val) | 0x80
		if _, err := w.Write([]byte{b}); err != nil {
			return err
		}
		val >>= 7
	}
	_, err := w.Write([]byte{byte(val)})
	return err
}

// DecodeVarintFromReader reads a Monero-style varint from r and returns the decoded value.
func DecodeVarintFromReader(r io.Reader) (uint64, error) {
	var result uint64
	var shift int
	for {
		var b [1]byte
		if _, err := r.Read(b[:]); err != nil {
			return 0, err
		}
		if b[0]&0x80 != 0 {
			result |= uint64(b[0]&0x7F) << shift
			shift += 7
			if shift >= 64 {
				return 0, ErrOverflow
			}
		} else {
			result |= uint64(b[0]) << shift
			return result, nil
		}
	}
}

// EncodeVarint encodes a uint64 as a Monero-style varint and returns the byte slice.
func EncodeVarint(val uint64) []byte {
	var buf []byte
	for val >= 0x80 {
		buf = append(buf, byte(val)|0x80)
		val >>= 7
	}
	buf = append(buf, byte(val))
	return buf
}

// DecodeVarint decodes a Monero-style varint from the beginning of data.
// Returns the value, the number of bytes consumed, and any error.
func DecodeVarint(data []byte) (uint64, int, error) {
	var result uint64
	var shift int
	for i, b := range data {
		if b&0x80 != 0 {
			result |= uint64(b&0x7F) << shift
			shift += 7
			if shift >= 64 {
				return 0, 0, ErrOverflow
			}
		} else {
			result |= uint64(b) << shift
			return result, i + 1, nil
		}
	}
	return 0, 0, io.ErrUnexpectedEOF
}
