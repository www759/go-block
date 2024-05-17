package network

import (
	"bytes"
	"fmt"
	"net"
)

type TCPPeer struct {
	conn     net.Conn
	OutGoing bool
}

func (p *TCPPeer) Send(b []byte) error {
	_, err := p.conn.Write(b)
	return err
}

func (p *TCPPeer) readLoop(rpcCh chan RPC) {
	buf := make([]byte, 2048)
	for {
		n, err := p.conn.Read(buf)
		if err != nil {
			//fmt.Println("read error: %s", err)
			continue
		}

		msg := buf[:n]
		rpcCh <- RPC{
			From:    p.conn.RemoteAddr(),
			Payload: bytes.NewReader(msg),
		}

		//fmt.Println(string(msg))
	}
}

type TCPTransport struct {
	peerCh     chan *TCPPeer
	ListenAddr string
	listener   net.Listener
}

func NewTCPTransport(addr string, peerCh chan *TCPPeer) *TCPTransport {
	return &TCPTransport{
		peerCh:     peerCh,
		ListenAddr: addr,
	}
}

func (t *TCPTransport) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("accept error from %+v", conn)
			continue
		}

		peer := &TCPPeer{conn: conn}

		t.peerCh <- peer

	}
}

func (t *TCPTransport) Start() error {
	ln, err := net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}
	t.listener = ln

	go t.acceptLoop()

	fmt.Printf("TCP transport listening to port: %s\n", t.ListenAddr)

	return nil
}
