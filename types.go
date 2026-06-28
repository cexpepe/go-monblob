// Package monblob defines all data structures for Monero transaction parsing.
// Types follow CNS003 and Monero v0.18+ specifications.
package monblob

// Transaction represents a complete Monero transaction with its prefix and signatures.
type Transaction struct {
	Prefix     TransactionPrefix
	Signatures [][]byte // Each input contributes one signature group (ring signature or generation signature)
}

// TransactionPrefix holds the non-signature part of a transaction.
// This is the portion hashed to compute the transaction ID.
type TransactionPrefix struct {
	Version    uint64      // Current version is 2
	UnlockTime uint64      // 0 means immediate unlock (block height or timestamp otherwise)
	Inputs     []TxIn
	Outputs    []TxOut
	Extra      []byte
}

// TxIn represents a transaction input with a type tag.
// The actual input data is stored in either Gen or ToKey based on Type.
type TxIn struct {
	Type  uint8      // 0xFF=Gen, 0x01=ToKey, 0x02=ToKeyTagged
	Gen   *TxInGen   // Used when Type == TxInTypeGen
	ToKey *TxInToKey // Used when Type == TxInTypeToKey or TxInTypeToKeyTagged
}

// TxIn type constants.
const (
	TxInTypeGen         = 0xFF
	TxInTypeToKey       = 0x01
	TxInTypeToKeyTagged = 0x02
)

// TxInGen represents a coinbase (miner) transaction input.
type TxInGen struct {
	Height uint64 // Block height at which this coinbase was generated
}

// TxInToKey represents a standard transaction input referencing outputs in the ring.
type TxInToKey struct {
	Amount      uint64   // Always 0 in Monero (amounts are hidden)
	KeyOffsets  []uint64 // Ring member indices as offsets from the global output index
	KeyImage    [32]byte // Key image to prevent double-spending
}

// TxOut represents a transaction output.
type TxOut struct {
	Amount uint64      // Always 0 in Monero
	Target TxOutTarget
}

// TxOutTarget is a union of output target types.
type TxOutTarget struct {
	Type   uint8      // 0x01=ToKey, 0x02=ToKeyTagged
	ToKey  *[32]byte  // Standard output public key
	Tagged *TaggedKey // Tagged output (v0.18+)
}

// TaggedKey represents a view-tagged output public key.
type TaggedKey struct {
	ViewTag   uint32
	PublicKey [32]byte
}

// TxOut type constants.
const (
	TxOutTypeToKey       = 0x01
	TxOutTypeToKeyTagged = 0x02
)

// ExtraField represents a single field in the transaction extra section.
// The extra section is a TLV (Tag-Length-Value) structure, but the length
// and format depend on the tag. This struct stores the raw data for flexibility.
type ExtraField struct {
	Tag  uint8 // 0x00=padding, 0x01=pubkey, 0x02=nonce, 0x04=additional_pubkeys
	Data []byte // Raw data excluding the tag byte
}

// Extra tag constants as defined in Monero specifications.
const (
	ExtraTagPadding           = 0x00 // Padding bytes, skipped entirely
	ExtraTagPublicKey         = 0x01 // 32-byte public key (tx_public_key)
	ExtraTagExtraNonce        = 0x02 // Variable length extra nonce: [length][data]
	ExtraTagMergeMining       = 0x03 // Deprecated merge mining info
	ExtraTagAdditionalPubKeys = 0x04 // [count][pubkey1][pubkey2]... (additional public keys)
	ExtraTagMinerGate         = 0xDE // Miner gate tag (reserved)
)

// ExtraNonce sub-tags used inside ExtraTagExtraNonce.
const (
	ExtraNonceTagPaymentID          = 0x00 // 32-byte plain payment ID (deprecated since v0.15)
	ExtraNonceTagEncryptedPaymentID = 0x01 // 8-byte encrypted payment ID
)

// ExtraPublicKey is a convenience type for a 32-byte public key.
type ExtraPublicKey [32]byte

// ExtraAdditionalPubKeys holds the parsed additional public keys from ExtraTagAdditionalPubKeys.
type ExtraAdditionalPubKeys struct {
	Count uint8
	Keys  [][32]byte
}

// ExtraNonce holds the parsed extra nonce data.
type ExtraNonce struct {
	Tag  uint8 // Sub-tag (e.g., ExtraNonceTagPaymentID)
	Data []byte // Raw data corresponding to the sub-tag
}