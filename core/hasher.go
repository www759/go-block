package core

import (
	"crypto/sha256"
	"encoding/binary"
	"goblock/types"
)

type Hasher[T any] interface {
	Hash(T) types.Hash
}

type BlockHasher struct{}

func (BlockHasher) Hash(h *Header) types.Hash {
	hash := sha256.Sum256(h.Bytes())
	return types.Hash(hash)
}

type TxHasher struct{}

func (TxHasher) Hash(tx *Transaction) types.Hash {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(tx.Nonce))

	data := append(tx.Data, buf...)
	return types.Hash(sha256.Sum256(data))
}
