package core

import (
	"fmt"
	"github.com/go-kit/log"
	"goblock/types"
	"sync"
)

type Blockchain struct {
	logger      log.Logger
	store       Storage
	lock        sync.RWMutex
	headers     []*Header
	blocks      []*Block
	txStore     map[types.Hash]*Transaction
	blocksStore map[types.Hash]*Block
	validator   Validator
	// TODO: make this an interface
	contractState *State
}

func NewBlockchain(l log.Logger, genesis *Block) (*Blockchain, error) {
	bc := &Blockchain{
		contractState: NewState(),
		logger:        l,
		headers:       []*Header{},
		store:         NewMemorystore(),
		txStore:       make(map[types.Hash]*Transaction),
		blocksStore:   make(map[types.Hash]*Block),
	}
	bc.validator = NewBlockValidator(bc)
	err := bc.addBlockWithoutValidation(genesis)
	return bc, err
}

func (bc *Blockchain) SetValidator(v Validator) {
	bc.validator = v
}

func (bc *Blockchain) AddBlock(b *Block) error {
	// validate
	if err := bc.validator.ValidateBlock(b); err != nil {
		return err
	}

	// run vm code
	for _, tx := range b.Transactions {
		bc.logger.Log("msg", "executing code", "len", len(tx.Data), "hash", tx.Hash(&TxHasher{}))

		vm := NewVM(tx.Data, bc.contractState)
		if err := vm.Run(); err != nil {
			return err
		}

		//fmt.Printf("State: %+v\n", vm.contractState)

		//result := vm.stack.Pop()
		//fmt.Printf("VM RESULT: %+v\n", result)
	}

	// add to blockchain
	return bc.addBlockWithoutValidation(b)
}

func (bc *Blockchain) HasBlock(height uint32) bool {
	return bc.Height() >= height
}

func (bc *Blockchain) Height() uint32 {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	// [0, 1, 2, 3] => height 3
	return uint32(len(bc.headers) - 1)
}

func (bc *Blockchain) GetHeader(height uint32) (*Header, error) {
	if height > bc.Height() {
		return nil, fmt.Errorf("given height (%d) too high", height)
	}

	bc.lock.Lock()
	defer bc.lock.Unlock()

	return bc.headers[height], nil
}

func (bc *Blockchain) GetTxByHash(hash types.Hash) (*Transaction, error) {
	bc.lock.Lock()
	bc.lock.Unlock()

	tx, ok := bc.txStore[hash]
	if !ok {
		return nil, fmt.Errorf("could not find tx with hash (%s)", hash)
	}

	return tx, nil
}

func (bc *Blockchain) GetBlock(height uint32) (*Block, error) {
	if height > bc.Height() {
		return nil, fmt.Errorf("given height (%d) too high", height)
	}

	bc.lock.Lock()
	defer bc.lock.Unlock()

	return bc.blocks[height], nil
}

func (bc *Blockchain) GetBlockByHash(hash types.Hash) (*Block, error) {
	bc.lock.Lock()
	bc.lock.Unlock()

	block, ok := bc.blocksStore[hash]
	if !ok {
		return nil, fmt.Errorf("block with hash (%s) not found", hash)
	}

	return block, nil
}

func (bc *Blockchain) addBlockWithoutValidation(b *Block) error {
	bc.lock.Lock()
	bc.headers = append(bc.headers, b.Header)
	bc.blocks = append(bc.blocks, b)
	bc.blocksStore[b.Hash(BlockHasher{})] = b
	defer bc.lock.Unlock()

	for _, tx := range b.Transactions {
		bc.txStore[tx.Hash(TxHasher{})] = tx
	}

	bc.logger.Log(
		"msg", "adding new block",
		"hash", b.Hash(BlockHasher{}),
		"height", b.Height,
		"transactions", len(b.Transactions),
	)

	return bc.store.Put(b)
}
