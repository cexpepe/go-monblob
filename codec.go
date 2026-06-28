// Package monblob provides binary encoding and decoding for Monero transactions.
package monblob

import (
	"encoding/binary"
	"errors"
)

// Error definitions
var (
	ErrUnexpectedEOF    = errors.New("unexpected end of data")
	ErrUnknownTxInType  = errors.New("unknown transaction input type")
	ErrUnknownTxOutType = errors.New("unknown transaction output type")
	ErrVarintOverflow   = errors.New("varint overflow")
	ErrMaxSliceSize     = errors.New("slice size exceeds maximum allowed")
	ErrRecursionDepth   = errors.New("maximum recursion depth exceeded")
)

// Security limits
const (
	MaxInputs         = 4096
	MaxOutputs        = 4096
	MaxKeyOffsets     = 4096
	MaxSignatureSize  = 10 * 1024 * 1024
	MaxRecursionDepth = 32
	MaxVarBytesSize   = 10 * 1024 * 1024
)

// reader
type reader struct {
	data  []byte
	off   int
	depth int
}

func newReader(data []byte) *reader {
	return &reader{data: data, off: 0, depth: 0}
}

func (r *reader) readUint8(v *uint8) error {
	if r.off+1 > len(r.data) {
		return ErrUnexpectedEOF
	}
	*v = r.data[r.off]
	r.off++
	return nil
}

func (r *reader) readUint16(v *uint16) error {
	if r.off+2 > len(r.data) {
		return ErrUnexpectedEOF
	}
	*v = binary.LittleEndian.Uint16(r.data[r.off:r.off+2])
	r.off += 2
	return nil
}

func (r *reader) readUint32(v *uint32) error {
	if r.off+4 > len(r.data) {
		return ErrUnexpectedEOF
	}
	*v = binary.LittleEndian.Uint32(r.data[r.off:r.off+4])
	r.off += 4
	return nil
}

func (r *reader) readUint64(v *uint64) error {
	if r.off+8 > len(r.data) {
		return ErrUnexpectedEOF
	}
	*v = binary.LittleEndian.Uint64(r.data[r.off:r.off+8])
	r.off += 8
	return nil
}

func (r *reader) readFixedBytes(dst []byte) error {
	if r.off+len(dst) > len(r.data) {
		return ErrUnexpectedEOF
	}
	copy(dst, r.data[r.off:r.off+len(dst)])
	r.off += len(dst)
	return nil
}

func (r *reader) readVarint(v *uint64) error {
	val, n, err := DecodeVarint(r.data[r.off:])
	if err != nil {
		return err
	}
	*v = val
	r.off += n
	return nil
}

func (r *reader) readVarintSlice(v *[]uint64) error {
	var count uint64
	if err := r.readVarint(&count); err != nil {
		return err
	}
	if count > MaxKeyOffsets {
		return ErrMaxSliceSize
	}
	*v = make([]uint64, count)
	for i := uint64(0); i < count; i++ {
		if err := r.readVarint(&(*v)[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *reader) readVarBytes(v *[]byte) error {
	var length uint64
	if err := r.readVarint(&length); err != nil {
		return err
	}
	if length > MaxVarBytesSize {
		return ErrMaxSliceSize
	}
	if r.off+int(length) > len(r.data) {
		return ErrUnexpectedEOF
	}
	*v = r.data[r.off : r.off+int(length)]
	r.off += int(length)
	return nil
}

func (r *reader) checkDepth() error {
	r.depth++
	if r.depth > MaxRecursionDepth {
		return ErrRecursionDepth
	}
	return nil
}

func (r *reader) eof() bool {
	return r.off >= len(r.data)
}

// writer
type writer struct {
	buf []byte
}

func newWriter() *writer {
	return &writer{buf: make([]byte, 0, 2048)}
}

func (w *writer) writeUint8(v uint8) error {
	w.buf = append(w.buf, v)
	return nil
}

func (w *writer) writeUint16(v uint16) error {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	w.buf = append(w.buf, b[:]...)
	return nil
}

func (w *writer) writeUint32(v uint32) error {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	w.buf = append(w.buf, b[:]...)
	return nil
}

func (w *writer) writeUint64(v uint64) error {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	w.buf = append(w.buf, b[:]...)
	return nil
}

func (w *writer) writeFixedBytes(src []byte) error {
	w.buf = append(w.buf, src...)
	return nil
}

func (w *writer) writeVarint(v uint64) error {
	w.buf = append(w.buf, EncodeVarint(v)...)
	return nil
}

func (w *writer) writeVarintSlice(v []uint64) error {
	if err := w.writeVarint(uint64(len(v))); err != nil {
		return err
	}
	for _, val := range v {
		if err := w.writeVarint(val); err != nil {
			return err
		}
	}
	return nil
}

func (w *writer) writeVarBytes(v []byte) error {
	if err := w.writeVarint(uint64(len(v))); err != nil {
		return err
	}
	w.buf = append(w.buf, v...)
	return nil
}

func (w *writer) bytes() []byte {
	return w.buf
}

// ==================== Decoding ====================

func (tx *Transaction) decode(r *reader) error {
	if err := r.checkDepth(); err != nil {
		return err
	}
	defer func() { r.depth-- }()

	if err := tx.Prefix.decode(r); err != nil {
		return err
	}

	remaining := r.data[r.off:]
	if len(remaining) > MaxSignatureSize {
		return ErrMaxSliceSize
	}
	tx.Signatures = [][]byte{remaining}
	r.off = len(r.data)
	return nil
}

func (p *TransactionPrefix) decode(r *reader) error {
	if err := r.checkDepth(); err != nil {
		return err
	}
	defer func() { r.depth-- }()

	if err := r.readVarint(&p.Version); err != nil {
		return err
	}
	if err := r.readVarint(&p.UnlockTime); err != nil {
		return err
	}

	var numInputs uint64
	if err := r.readVarint(&numInputs); err != nil {
		return err
	}
	if numInputs > MaxInputs {
		return ErrMaxSliceSize
	}
	p.Inputs = make([]TxIn, numInputs)
	for i := range p.Inputs {
		if err := p.Inputs[i].decode(r); err != nil {
			return err
		}
	}

	var numOutputs uint64
	if err := r.readVarint(&numOutputs); err != nil {
		return err
	}
	if numOutputs > MaxOutputs {
		return ErrMaxSliceSize
	}

	p.Outputs = make([]TxOut, 0, numOutputs)
	for i := uint64(0); i < numOutputs; i++ {
		outStart := r.off
		var out TxOut
		if err := out.decode(r); err != nil {
			// Rollback to the start of this output and treat everything from there as Extra.
			r.off = outStart
			p.Extra = append([]byte(nil), r.data[r.off:]...)
			r.off = len(r.data)
			return nil
		}
		p.Outputs = append(p.Outputs, out)
	}

	// All outputs parsed successfully: everything after is Extra.
	p.Extra = append([]byte(nil), r.data[r.off:]...)
	r.off = len(r.data)

	return nil
}

func (in *TxIn) decode(r *reader) error {
	if err := r.readUint8(&in.Type); err != nil {
		return err
	}
	switch in.Type {
	case TxInTypeGen:
		in.Gen = &TxInGen{}
		return in.Gen.decode(r)
	case TxInTypeToKey, TxInTypeToKeyTagged:
		in.ToKey = &TxInToKey{}
		return in.ToKey.decode(r)
	default:
		return ErrUnknownTxInType
	}
}

func (g *TxInGen) decode(r *reader) error {
	return r.readVarint(&g.Height)
}

func (t *TxInToKey) decode(r *reader) error {
	if err := r.readVarint(&t.Amount); err != nil {
		return err
	}
	if err := r.readVarintSlice(&t.KeyOffsets); err != nil {
		return err
	}
	return r.readFixedBytes(t.KeyImage[:])
}

func (out *TxOut) decode(r *reader) error {
	if err := r.readVarint(&out.Amount); err != nil {
		return err
	}
	return out.Target.decode(r)
}

func (t *TxOutTarget) decode(r *reader) error {
	if err := r.readUint8(&t.Type); err != nil {
		return err
	}
	switch t.Type {
	case TxOutTypeToKey:
		t.ToKey = &[32]byte{}
		return r.readFixedBytes(t.ToKey[:])
	case TxOutTypeToKeyTagged, 0x03:
		t.Tagged = &TaggedKey{}
		if err := r.readUint32(&t.Tagged.ViewTag); err != nil {
			return err
		}
		return r.readFixedBytes(t.Tagged.PublicKey[:])
	default:
		return ErrUnknownTxOutType
	}
}

// ==================== Encoding ====================

func (tx *Transaction) encode(w *writer) error {
	if err := tx.Prefix.encode(w); err != nil {
		return err
	}
	if len(tx.Signatures) > 0 {
		if err := w.writeFixedBytes(tx.Signatures[0]); err != nil {
			return err
		}
	}
	return nil
}

func (p *TransactionPrefix) encode(w *writer) error {
	if err := w.writeVarint(p.Version); err != nil {
		return err
	}
	if err := w.writeVarint(p.UnlockTime); err != nil {
		return err
	}

	if err := w.writeVarint(uint64(len(p.Inputs))); err != nil {
		return err
	}
	for _, in := range p.Inputs {
		if err := in.encode(w); err != nil {
			return err
		}
	}

	if err := w.writeVarint(uint64(len(p.Outputs))); err != nil {
		return err
	}
	for _, out := range p.Outputs {
		if err := out.encode(w); err != nil {
			return err
		}
	}

	if err := w.writeFixedBytes(p.Extra); err != nil {
		return err
	}
	return nil
}

func (in *TxIn) encode(w *writer) error {
	if err := w.writeUint8(in.Type); err != nil {
		return err
	}
	switch in.Type {
	case TxInTypeGen:
		return in.Gen.encode(w)
	case TxInTypeToKey, TxInTypeToKeyTagged:
		return in.ToKey.encode(w)
	default:
		return ErrUnknownTxInType
	}
}

func (g *TxInGen) encode(w *writer) error {
	return w.writeVarint(g.Height)
}

func (t *TxInToKey) encode(w *writer) error {
	if err := w.writeVarint(t.Amount); err != nil {
		return err
	}
	if err := w.writeVarintSlice(t.KeyOffsets); err != nil {
		return err
	}
	return w.writeFixedBytes(t.KeyImage[:])
}

func (out *TxOut) encode(w *writer) error {
	if err := w.writeVarint(out.Amount); err != nil {
		return err
	}
	return out.Target.encode(w)
}

func (t *TxOutTarget) encode(w *writer) error {
	if err := w.writeUint8(t.Type); err != nil {
		return err
	}
	switch t.Type {
	case TxOutTypeToKey:
		return w.writeFixedBytes(t.ToKey[:])
	case TxOutTypeToKeyTagged, 0x03:
		if err := w.writeUint32(t.Tagged.ViewTag); err != nil {
			return err
		}
		return w.writeFixedBytes(t.Tagged.PublicKey[:])
	default:
		return ErrUnknownTxOutType
	}
}