package api

import (
	"encoding/gob"
	"encoding/hex"
	"github.com/go-kit/log"
	"github.com/labstack/echo/v4"
	"goblock/core"
	"goblock/types"
	"net/http"
	"strconv"
)

type APIError struct {
	Error string
}

type TxResponse struct {
	TxCount uint
	Hashes  []string
}

type JSONBlock struct {
	Hash          string
	Version       uint32
	DataHash      string
	PrevBlockHash string
	Height        uint32
	Timestamp     int64
	Validator     string
	Signature     string

	TxResponse TxResponse
}

type ServerConfig struct {
	Logger     log.Logger
	ListenAddr string
}

type Server struct {
	txChan chan *core.Transaction
	ServerConfig
	bc *core.Blockchain
}

func NewServer(cfg ServerConfig, bc *core.Blockchain, txChan chan *core.Transaction) *Server {
	return &Server{
		txChan:       txChan,
		ServerConfig: cfg,
		bc:           bc,
	}
}

func (s *Server) Start() error {
	e := echo.New()

	e.GET("/block/:hashorid", s.handleGetBlock)
	e.GET("/tx/:hash", s.handleGetTx)
	e.POST("/tx", s.handlePostTx)

	return e.Start(s.ListenAddr)
}

func (s *Server) handlePostTx(c echo.Context) error {
	tx := &core.Transaction{}
	if err := gob.NewDecoder(c.Request().Body).Decode(tx); err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: err.Error()})
	}

	//fmt.Printf("%+v\n", tx)

	s.txChan <- tx

	return nil
}

func (s *Server) handleGetTx(c echo.Context) error {
	h := c.Param("hash")

	hBytes, err := hex.DecodeString(h)
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: err.Error()})
	}

	tx, err := s.bc.GetTxByHash(types.HashFromBytes(hBytes))
	if err != nil {
		return c.JSON(http.StatusBadRequest, APIError{Error: err.Error()})
	}

	return c.JSON(http.StatusOK, tx)
}

func (s *Server) handleGetBlock(c echo.Context) error {
	hashOrID := c.Param("hashorid")

	height, err := strconv.Atoi(hashOrID)

	if err == nil {
		block, err := s.bc.GetBlock(uint32(height))
		if err != nil {
			return c.JSON(http.StatusBadRequest, APIError{Error: err.Error()})
		}

		return c.JSON(http.StatusOK, intoJSONBlock(block))
	}

	// implement getBlockByHash
	b, err := hex.DecodeString(hashOrID)
	if err != nil {
		return err
	}

	block, err := s.bc.GetBlockByHash(types.HashFromBytes(b))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, intoJSONBlock(block))
}

func intoJSONBlock(block *core.Block) JSONBlock {
	txResponse := TxResponse{
		TxCount: uint(len(block.Transactions)),
		Hashes:  make([]string, len(block.Transactions)),
	}

	for i := 0; i < int(txResponse.TxCount); i++ {
		txResponse.Hashes[i] = block.Transactions[i].Hash(core.TxHasher{}).String()
	}

	return JSONBlock{
		Hash:          block.Hash(core.BlockHasher{}).String(),
		Version:       block.Version,
		Height:        block.Height,
		DataHash:      block.DataHash.String(),
		PrevBlockHash: block.PrevBlockHash.String(),
		Timestamp:     block.Timestamp,
		Validator:     block.Validator.Address().String(),
		Signature:     block.Signature.String(),

		TxResponse: txResponse,
	}
}
