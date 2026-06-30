// Package monblob provides Keccak-256 hashing for transaction IDs.
package monblob

import "golang.org/x/crypto/sha3"

// Keccak256 computes the Keccak-256 hash of the input data.
func Keccak256(data []byte) [32]byte {
	h := sha3.NewLegacyKeccak256()
	h.Write(data)
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// Hash computes the transaction ID for a complete Transaction.
func (tx *Transaction) Hash() [32]byte {
	prefixBytes, _ := SerializePrefix(&tx.Prefix)
	return Keccak256(prefixBytes)
}

// HashPrefix computes the transaction ID directly from a TransactionPrefix.
func HashPrefix(prefix *TransactionPrefix) [32]byte {
	prefixBytes, _ := SerializePrefix(prefix)
	return Keccak256(prefixBytes)
}
