package core

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"goblock/crypto"
	"goblock/types"
	"testing"
	"time"
)

func TestHashBlock(t *testing.T) {
	b := randomBlock(t, 0, types.Hash{})
	fmt.Println(b.Hash(BlockHasher{}))
}

func TestBlock_Sign(t *testing.T) {
	privKey := crypto.GeneratePrivateKey()
	b := randomBlock(t, 0, types.Hash{})

	assert.Nil(t, b.Sign(privKey))
	assert.NotNil(t, b.Signature)
}

func TestBlock_Verify(t *testing.T) {
	privKey := crypto.GeneratePrivateKey()
	b := randomBlock(t, 0, types.Hash{})

	assert.Nil(t, b.Sign(privKey))
	assert.Nil(t, b.Verify())

	// 签名者改变
	otherPrivKey := crypto.GeneratePrivateKey()
	b.Validator = otherPrivKey.PublicKey()
	assert.NotNil(t, b.Verify())

	// 数据改变
	b.Height = 10
	assert.NotNil(t, b.Verify())
}

func TestDecodeEncodeBlock(t *testing.T) {
	b := randomBlock(t, 1, types.Hash{})
	buf := &bytes.Buffer{}
	assert.Nil(t, b.Encode(NewGobBlockEncoder(buf)))

	bb := new(Block)
	assert.Nil(t, bb.Decode(NewGobBlockDecoder(buf)))
	assert.Equal(t, b, bb)
}

func randomBlock(t *testing.T, height uint32, prevBlockHash types.Hash) *Block {
	privKey := crypto.GeneratePrivateKey()
	tx := RandomTxWithSignature(t)
	header := &Header{
		Version:       1,
		PrevBlockHash: prevBlockHash,
		Height:        height,
		Timestamp:     time.Now().UnixNano(),
	}

	b, err := NewBlock(header, []*Transaction{tx})
	assert.Nil(t, err)
	dataHash, err := CalculateDataHash(b.Transactions)
	assert.Nil(t, err)
	b.DataHash = dataHash
	assert.Nil(t, b.Sign(privKey))

	return b
}
