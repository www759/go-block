package network

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/go-kit/log"
	"goblock/api"
	"goblock/core"
	"goblock/crypto"
	"goblock/types"
	"net"
	"os"
	"sync"
	"time"
)

var defaultBlockTime = 5 * time.Second

type ServerOpts struct {
	APIListenAddr string
	SeedNodes     []string
	ListenAddr    string
	ID            string
	Logger        log.Logger
	RPCDecodeFunc RPCDecodeFunc
	RPCProcessor  RPCProcessor
	BlockTime     time.Duration
	PrivateKey    *crypto.PrivateKey
}

type Server struct {
	TCPTransport *TCPTransport
	peerCh       chan *TCPPeer

	mutex   sync.RWMutex
	peerMap map[net.Addr]*TCPPeer

	ServerOpts
	memPool     *TxPool
	chain       *core.Blockchain
	isValidator bool
	rpcCh       chan RPC
	msgCh       chan []byte
	quitCh      chan struct{}
	txChan      chan *core.Transaction
}

func NewServer(opts ServerOpts) (*Server, error) {

	if opts.BlockTime == time.Duration(0) {
		opts.BlockTime = defaultBlockTime
	}

	if opts.RPCDecodeFunc == nil {
		opts.RPCDecodeFunc = DefaultRPCDecodeFunc
	}

	if opts.Logger == nil {
		opts.Logger = log.NewLogfmtLogger(os.Stderr)
		opts.Logger = log.With(opts.Logger, "addr", opts.ID)
	}

	chain, err := core.NewBlockchain(opts.Logger, genesisBlock())
	if err != nil {
		return nil, err
	}

	// api server
	txChan := make(chan *core.Transaction)
	if len(opts.APIListenAddr) > 0 {
		apiServerCfg := api.ServerConfig{
			Logger:     opts.Logger,
			ListenAddr: opts.APIListenAddr,
		}
		apiServer := api.NewServer(apiServerCfg, chain, txChan)

		go apiServer.Start()

		opts.Logger.Log("msg", "JSON API server running", "port", opts.APIListenAddr)
	}

	peerCh := make(chan *TCPPeer)
	tr := NewTCPTransport(opts.ListenAddr, peerCh)
	s := &Server{
		TCPTransport: tr,
		peerCh:       peerCh,
		peerMap:      make(map[net.Addr]*TCPPeer),
		ServerOpts:   opts,
		chain:        chain,
		memPool:      NewTxPool(),
		isValidator:  opts.PrivateKey != nil,
		rpcCh:        make(chan RPC),
		msgCh:        make(chan []byte),
		quitCh:       make(chan struct{}, 1),
		txChan:       txChan,
	}
	s.TCPTransport.peerCh = peerCh

	if s.RPCProcessor == nil {
		s.RPCProcessor = s
	}

	if s.isValidator {
		go s.validateLoop()
	}

	return s, nil
}

func (s *Server) bootStrapNetwork() {
	for _, addr := range s.SeedNodes {
		fmt.Println("trying to connect to: ", addr)

		go func(addr string) {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				fmt.Printf("could not connect to %+v\n", conn)
				return
			}

			s.peerCh <- &TCPPeer{
				conn: conn,
			}
		}(addr)
	}
}

// Start 打开监听端口, 对到达通道的数据进行处理: 新节点到来（发送GetStatus），rpc消息
func (s *Server) Start() {
	s.TCPTransport.Start()
	time.Sleep(1 * time.Second)

	s.bootStrapNetwork()

	s.Logger.Log("msg", "accepting TCP connection on", "addr", s.ListenAddr, "id", s.ID)

free:
	for {
		select {
		case peer := <-s.peerCh:

			s.peerMap[peer.conn.RemoteAddr()] = peer

			go peer.readLoop(s.rpcCh)

			if err := s.sendGetStatusMessage(peer); err != nil {
				s.Logger.Log("sever.Start.sendGetStatusMessage err: ", err)
				continue
			}
			fmt.Printf("peerCh get New Peer: %+v\n", peer)

		case tx := <-s.txChan:
			if err := s.ProcessTransaction(tx); err != nil {
				s.Logger.Log("process Tx error", err)
			}

		case rpc := <-s.rpcCh:
			msg, err := s.RPCDecodeFunc(rpc)
			if err != nil {
				s.Logger.Log("error", err)
			}
			if err := s.RPCProcessor.ProcessMessage(msg); err != nil {
				if err != core.ErrBlockKnown {
					s.Logger.Log("error", err)
				}
			}

		case <-s.quitCh:
			break free
		}
	}
	s.Logger.Log("msg", "Server is shutting down")
}

func (s *Server) validateLoop() {
	ticker := time.NewTicker(s.BlockTime)

	s.Logger.Log("msg", "Starting validator loop", "blockTime", s.BlockTime)

	for {
		<-ticker.C
		s.createNewBlock()
	}
}

func (s *Server) ProcessMessage(msg *DecodeMessage) error {
	switch t := msg.Data.(type) {
	case *core.Transaction:
		return s.ProcessTransaction(t)
	case *core.Block:
		return s.processBlock(t)
	case *GetStatusMessage:
		return s.processGetStatusMessage(msg.From, t)
	case *StatusMessage:
		return s.processStatusMessage(msg.From, t)
	case *GetBlocksMessage:
		return s.processGetBlocksMessage(msg.From, t)
	case *BlocksMessage:
		return s.processBlocksMessage(t)
	default:
		return nil
	}
	return nil
}

func (s *Server) sendGetStatusMessage(peer *TCPPeer) error {
	getStatusMsg := new(GetStatusMessage)
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(getStatusMsg); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeGetStatus, buf.Bytes())

	if err := peer.Send(msg.Bytes()); err != nil {
		return err
	}
	return nil
}

// 广播一堆字节
func (s *Server) broadcast(payload []byte) error {
	s.mutex.RLock()
	defer s.mutex.RLock()

	for netAddr, peer := range s.peerMap {
		if err := peer.Send(payload); err != nil {
			fmt.Printf("peer send error => addr %s [err: %s]", netAddr, err)
		}
	}
	return nil
}

func (s *Server) processGetStatusMessage(from net.Addr, data *GetStatusMessage) error {
	s.Logger.Log("msg", "received GET_STATUS message", "from", from)
	statusMessage := &StatusMessage{
		ID:            s.ID,
		CurrentHeight: s.chain.Height(),
	}

	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(statusMessage); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeStatus, buf.Bytes())

	s.mutex.RLock()
	defer s.mutex.RUnlock()
	peer, ok := s.peerMap[from]
	if !ok {
		return fmt.Errorf("peer %s not known", peer.conn.RemoteAddr())
	}
	return peer.Send(msg.Bytes())
}

// TODO: 此处不同---------
func (s *Server) processStatusMessage(from net.Addr, data *StatusMessage) error {
	s.Logger.Log("msg", "received STATUS message", "from", from)

	if data.CurrentHeight <= s.chain.Height() {
		s.Logger.Log("msg", "cannot sync blockHeight to low", "ourHeight", s.chain.Height(), "theirHeight", data.CurrentHeight, "addr", from)
		return nil
	}

	getBlocksMessage := &GetBlocksMessage{
		From: s.chain.Height() + 1,
		To:   0,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(getBlocksMessage); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeGetBlocks, buf.Bytes())

	s.mutex.RLock()
	defer s.mutex.RUnlock()
	peer, ok := s.peerMap[from]
	if !ok {
		return fmt.Errorf("peer %s not known", peer.conn.RemoteAddr())
	}
	return peer.Send(msg.Bytes())
}

func (s *Server) processGetBlocksMessage(from net.Addr, data *GetBlocksMessage) error {
	s.Logger.Log("msg", "received GET_BLOCKS message", "from", from)

	blocks := []*core.Block{}

	if data.To == 0 {
		for i := int(data.From); i <= int(s.chain.Height()); i++ {
			block, err := s.chain.GetBlock(uint32(i))
			if err != nil {
				return err
			}

			blocks = append(blocks, block)
		}
	}

	//fmt.Println("-----------", blocks[0])

	blocksMessage := &BlocksMessage{
		Blocks: blocks,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(blocksMessage); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeBlocks, buf.Bytes())

	s.mutex.RLock()
	defer s.mutex.RUnlock()
	peer, ok := s.peerMap[from]
	if !ok {
		return fmt.Errorf("peer %s not known", peer.conn.RemoteAddr())
	}
	return peer.Send(msg.Bytes())
}

func (s *Server) processBlocksMessage(data *BlocksMessage) error {
	s.Logger.Log("msg", "received BLOCKS message")

	for _, block := range data.Blocks {
		//fmt.Println("++++++++++++", block)
		if err := s.chain.AddBlock(block); err != nil {
			return err
		}
	}

	return nil
}

// ProcessTransaction 将Tx添加到mempool中并广播给其他peers
func (s *Server) ProcessTransaction(tx *core.Transaction) error {
	hash := tx.Hash(core.TxHasher{})

	if s.memPool.Has(hash) {
		return nil
	}

	if err := tx.Verify(); err != nil {
		return err
	}

	tx.SetFirstSeen(time.Now().UnixNano())

	//s.Logger.Log(
	//	"msg", "adding new tx to the mempool",
	//	"hash", hash,
	//	"mempool length", s.memPool.Len(),
	//)

	go s.broadcastTx(tx)

	return s.memPool.Add(tx)
}

// TODO: find a way to make sure we dont keep syncing when we are at the highest
func (s *Server) requestBlocksLoop(peer net.Addr) error {
	ticker := time.NewTicker(3 * time.Second)
	for {
		ourHeight := s.chain.Height()
		s.Logger.Log("msg", "requesting new blocks", "currentHeight", ourHeight)

		getBlocksMessage := &GetBlocksMessage{
			From: ourHeight + 1,
			To:   0,
		}
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(getBlocksMessage); err != nil {
			return err
		}

		msg := NewMessage(MessageTypeGetBlocks, buf.Bytes())

		s.mutex.RLock()
		defer s.mutex.RUnlock()

		peer, ok := s.peerMap[peer]
		if !ok {
			return fmt.Errorf("peer %s not known", peer.conn.RemoteAddr())
		}
		if err := peer.Send(msg.Bytes()); err != nil {
			s.Logger.Log("error", "failed to send to peer", err, "peer", peer)
		}

		<-ticker.C
	}
	return nil
}

func (s *Server) processBlock(b *core.Block) error {
	if err := s.chain.AddBlock(b); err != nil {
		return err
	}

	go s.broadcastBlock(b)

	return nil
}

func (s *Server) broadcastTx(tx *core.Transaction) error {
	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeTx, buf.Bytes())

	return s.broadcast(msg.Bytes())
}

func (s *Server) broadcastBlock(b *core.Block) error {
	buf := &bytes.Buffer{}
	if err := b.Encode(core.NewGobBlockEncoder(buf)); err != nil {
		return err
	}

	msg := NewMessage(MessageTypeBlock, buf.Bytes())

	return s.broadcast(msg.Bytes())
}

// 创建一个区块并添加到链上
func (s *Server) createNewBlock() error {
	currentHeader, err := s.chain.GetHeader(s.chain.Height())
	if err != nil {
		return err
	}

	// TODO：
	txx := s.memPool.Transactions()

	block, err := core.NewBlockFromPrevHeader(currentHeader, txx)
	if err != nil {
		return err
	}

	if err := block.Sign(*s.PrivateKey); err != nil {
		return err
	}

	if err := s.chain.AddBlock(block); err != nil {
		return err
	}

	s.memPool.Flush()

	go s.broadcastBlock(block)

	return nil
}

// 创建创世区块，没有交易被添加
func genesisBlock() *core.Block {
	header := &core.Header{
		Version:   1,
		DataHash:  types.Hash{},
		Timestamp: 000000,
		Height:    0,
	}

	b, _ := core.NewBlock(header, nil)

	privKey := crypto.GeneratePrivateKey()
	if err := b.Sign(privKey); err != nil {
		panic(err)
	}
	return b
}
