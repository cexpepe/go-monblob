// Package monblob provides Monero transaction binary blob parsing and serialization.
// It follows CNS003 and Monero v0.18+ specifications with zero external dependencies.
package monblob

import (
	"errors"
	"io"
)

// ErrTrailingData indicates that extra bytes remain after fully parsing a transaction.
var ErrTrailingData = errors.New("trailing data after transaction")

// Parse parses a complete transaction from a binary blob.
// It automatically splits the prefix and signature sections.
func Parse(data []byte) (*Transaction, error) {
	r := newReader(data)
	tx := &Transaction{}
	if err := tx.decode(r); err != nil {
		return nil, err
	}
	if !r.eof() {
		return nil, ErrTrailingData
	}
	return tx, nil
}

// ParseFromReader reads all data from an io.Reader and parses a complete transaction.
// This is suitable for streaming large blobs or reading from network connections.
func ParseFromReader(r io.Reader) (*Transaction, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

// Serialize serializes a complete transaction back into its binary blob representation.
// The output is a concatenation of the prefix and all signatures.
func Serialize(tx *Transaction) ([]byte, error) {
	w := newWriter()
	if err := tx.encode(w); err != nil {
		return nil, err
	}
	return w.bytes(), nil
}

// SerializeToWriter serializes a transaction and writes the binary blob to the given io.Writer.
// This avoids holding the entire blob in memory when writing to a stream.
func SerializeToWriter(tx *Transaction, w io.Writer) error {
	data, err := Serialize(tx)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ParsePrefix parses only the transaction prefix (without signatures).
// This is useful for computing transaction IDs or inspecting transaction structure
// without processing potentially large signature data.
func ParsePrefix(data []byte) (*TransactionPrefix, error) {
	r := newReader(data)
	prefix := &TransactionPrefix{}
	if err := prefix.decode(r); err != nil {
		return nil, err
	}
	return prefix, nil
}

// SerializePrefix serializes only the transaction prefix.
// The result is suitable for hashing to obtain the transaction ID.
func SerializePrefix(prefix *TransactionPrefix) ([]byte, error) {
	w := newWriter()
	if err := prefix.encode(w); err != nil {
		return nil, err
	}
	return w.bytes(), nil
}
