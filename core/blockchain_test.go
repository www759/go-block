package core

//
//import (
//	"github.com/stretchr/testify/assert"
//	"goblock/types"
//	"testing"
//)
//
//func TestNewBlockchain(t *testing.T) {
//	bc, err := NewBlockchain(randomBlock(t, 0, types.Hash{}))
//	assert.Nil(t, err)
//	assert.NotNil(t, bc.validator)
//	assert.Equal(t, bc.Height(), uint32(0))
//}
//
//func TestHasBlock(t *testing.T) {
//	bc := newBlockchainWithGenesis(t)
//	assert.True(t, bc.HasBlock(0))
//	assert.False(t, bc.HasBlock(1))
//	assert.False(t, bc.HasBlock(100))
//}
//
//func TestAddBlock(t *testing.T) {
//	bc := newBlockchainWithGenesis(t)
//	lenBlocks := 1000
//	for i := 1; i <= lenBlocks; i++ {
//		prevBlockHash := getPrevBlockHash(t, bc, uint32(i))
//		block := randomBlock(t, uint32(i), prevBlockHash)
//
//		assert.Nil(t, bc.AddBlock(block))
//	}
//
//	assert.Equal(t, bc.Height(), uint32(lenBlocks))
//	assert.Equal(t, len(bc.headers), lenBlocks+1)
//
//	// too low
//	assert.NotNil(t, bc.AddBlock(randomBlock(t, 123, types.Hash{})))
//	// too high
//	assert.NotNil(t, bc.AddBlock(randomBlock(t, 1234, types.Hash{})))
//}
//
//func TestGetHeader(t *testing.T) {
//	bc := newBlockchainWithGenesis(t)
//	lenBlocks := 1000
//	for i := 1; i <= lenBlocks; i++ {
//		prevBlockHash := getPrevBlockHash(t, bc, uint32(i))
//		block := randomBlock(t, uint32(i), prevBlockHash)
//		assert.Nil(t, bc.AddBlock(block))
//		header, err := bc.GetHeader(uint32(i))
//		assert.Nil(t, err)
//		assert.Equal(t, header, block.Header)
//	}
//}
//
//func newBlockchainWithGenesis(t *testing.T) *Blockchain {
//	bc, err := NewBlockchain(randomBlock(t, 0, types.Hash{}))
//	assert.Nil(t, err)
//
//	return bc
//}
//
//func getPrevBlockHash(t *testing.T, bc *Blockchain, height uint32) types.Hash {
//	prevHeader, err := bc.GetHeader(height - 1)
//	assert.Nil(t, err)
//
//	return BlockHasher{}.Hash(prevHeader)
//}
