package main

import (
	"bytes"
	"goblock/core"
	"goblock/crypto"
	"goblock/network"
	"log"
	"net/http"
	"time"
)

func main() {
	privKey := crypto.GeneratePrivateKey()
	localNode := makeServer("LOCAL_NODE", &privKey, ":3000", []string{":4000"}, ":9000")
	go localNode.Start()

	remoteNode := makeServer("REMOTE_NODE", nil, ":4000", []string{":5051"}, "")
	go remoteNode.Start()

	remoteNodeB := makeServer("REMOTE_NODE_B", nil, ":5051", nil, "")
	go remoteNodeB.Start()

	go func() {
		time.Sleep(6 * time.Second)
		lateNode := makeServer("LATE_NODE", nil, ":6051", []string{":4000"}, "")
		go lateNode.Start()
	}()

	time.Sleep(1 * time.Second)

	txSenderTicker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			txSender()
			<-txSenderTicker.C
		}
	}()

	select {}

}

func makeServer(id string, pk *crypto.PrivateKey, addr string, seedNodes []string, apiListenAddr string) *network.Server {
	opts := network.ServerOpts{
		APIListenAddr: apiListenAddr,
		SeedNodes:     seedNodes,
		ListenAddr:    addr,
		PrivateKey:    pk,
		ID:            id,
	}

	s, err := network.NewServer(opts)
	if err != nil {
		log.Fatal(err)
	}

	return s
}

func txSender() {
	//conn, err := net.Dial("tcp", ":3000")
	//if err != nil {
	//	panic(err)
	//}

	privKey := crypto.GeneratePrivateKey()
	tx := core.NewTransaction(Contract())
	tx.Sign(privKey)

	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", "http://localhost:9000/tx", buf)
	if err != nil {
		panic(err)
	}

	client := http.Client{}
	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}

	//fmt.Printf("%+v", resp)
	//msg := network.NewMessage(network.MessageTypeTx, buf.Bytes())
	//
	//_, err = conn.Write(msg.Bytes())
	//if err != nil {
	//	panic(err)
	//}
}

func Contract() []byte {
	data := []byte{0x02, 0x0a, 0x03, 0x0a, 0x0b, 0x4f, 0x0c, 0x4f, 0x0c, 0x46, 0x0c, 0x03, 0x0a, 0x0d, 0x0f}
	pushFoo := []byte{0x4f, 0x0c, 0x4f, 0x0c, 0x46, 0x0c, 0x03, 0x0a, 0x0d, 0xae}
	data = append(data, pushFoo...)

	return data
}

//var transports = []network.Transport{
//	network.NewLocalTransport("LOCAL"),
//	//network.NewLocalTransport("REMOTE_A"),
//	//network.NewLocalTransport("REMOTE_B"),
//	//network.NewLocalTransport("REMOTE_C"),
//
//}
//
//func main() {
//	fmt.Println("-------hello block--------")
//
//	//initRemoteServers(transports)
//
//	//localNode := transports[0]
//	//remoteNodeA := transports[1]
//	//remoteNodeC := transports[3]
//
//	//go func() {
//	//	for {
//	//		if err := sendTransaction(remoteNodeA, localNode.Addr()); err != nil {
//	//			logrus.Error(err)
//	//		}
//	//		time.Sleep(2 * time.Second)
//	//	}
//	//}()
//
//	go func() {
//		time.Sleep(11 * time.Second)
//
//		trLate := network.NewLocalTransport("LATE_REMOTE")
//		transports[0].Connect(trLate)
//		lateServer := makeServer(string(trLate.Addr()), trLate, nil)
//
//		go lateServer.Start()
//	}()
//
//	privKey := crypto.GeneratePrivateKey()
//	localServer := makeServer(string(transports[0].Addr()), transports[0], &privKey)
//	localServer.Start()
//}

//func initRemoteServers(trs []network.Transport) {
//	for i := 1; i < len(trs)-1; i++ {
//		//id := fmt.Sprintf("REMOTE_%d", i)
//		s := makeServer(string(trs[i].Addr()), trs[i], nil)
//		go s.Start()
//	}
//}
//

//
//func sendGetStatusMessage(tr network.Transport, to network.NetAddr) error {
//	var (
//		getStatusMsg = new(network.GetStatusMessage)
//		buf          = new(bytes.Buffer)
//	)
//	if err := gob.NewEncoder(buf).Encode(getStatusMsg); err != nil {
//		return err
//	}
//
//	msg := network.NewMessage(network.MessageTypeGetStatus, buf.Bytes())
//
//	return tr.SendMessage(to, msg.Bytes())
//}
//
//func sendTransaction(tr network.Transport, to network.NetAddr) error {
//	privKey := crypto.GeneratePrivateKey()
//	tx := core.NewTransaction(Contract())
//	tx.Sign(privKey)
//
//	buf := &bytes.Buffer{}
//	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
//		return err
//	}
//
//	msg := network.NewMessage(network.MessageTypeTx, buf.Bytes())
//
//	return tr.SendMessage(to, msg.Bytes())
//}
