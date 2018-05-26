package network

import (
	"sync"

	kcp "github.com/somethinghero/kcp-go"
	"github.com/somethinghero/leaf/log"
)

//KCPServer kcp protocol server
type KCPServer struct {
	Addr            string
	CryptKey        string
	DataShard       int
	ParityShard     int
	MaxConnNum      int
	PendingWriteNum int
	NewAgent        func(*KCPConn) Agent
	ln              *kcp.Listener
	conns           KCPConnSet
	mutexConns      sync.Mutex
	wgLn            sync.WaitGroup
	wgConns         sync.WaitGroup

	// msg parser
	LenMsgLen    int
	MinMsgLen    uint32
	MaxMsgLen    uint32
	LittleEndian bool
	msgParser    *MsgParser
}

var (
	// SALT is use for pbkdf2 key expansion
	SALT = "kcp-server"
)

//Start start tcp server
func (server *KCPServer) Start() {
	server.init()
	go server.run()
}

func (server *KCPServer) init() {
	// pass := pbkdf2.Key([]byte(server.CryptKey), []byte(SALT), 4096, 32, sha1.New)
	// block, err := kcp.NewXTEABlockCrypt(pass[:16])
	// if err != nil {
	// 	log.Fatal("%v", err)
	// }
	//ln, err := kcp.ListenWithOptions(server.Addr, block, server.DataShard, server.ParityShard)
	ln, err := kcp.ListenWithOptions(server.Addr, nil, 0, 0)
	if err != nil {
		log.Fatal("%v", err)
	}

	if server.MaxConnNum <= 0 {
		server.MaxConnNum = 100
		log.Release("invalid MaxConnNum, reset to %v", server.MaxConnNum)
	}
	if server.PendingWriteNum <= 0 {
		server.PendingWriteNum = 100
		log.Release("invalid PendingWriteNum, reset to %v", server.PendingWriteNum)
	}
	if server.NewAgent == nil {
		log.Fatal("NewAgent must not be nil")
	}

	server.ln = ln
	server.conns = make(KCPConnSet)

	// msg parser
	msgParser := NewMsgParser()
	msgParser.SetMsgLen(server.LenMsgLen, server.MinMsgLen, server.MaxMsgLen)
	msgParser.SetByteOrder(server.LittleEndian)
	server.msgParser = msgParser
}

func (server *KCPServer) run() {
	server.wgLn.Add(1)
	defer server.wgLn.Done()

	for {
		conn, err := server.ln.AcceptKCP()
		if err == nil {
			conn.SetStreamMode(true)
			conn.SetWriteDelay(true)
			conn.SetNoDelay(1, 10, 2, 1)
			conn.SetMtu(1400)
			conn.SetWindowSize(4096, 4096)
			conn.SetACKNoDelay(true)

			server.mutexConns.Lock()
			if len(server.conns) >= server.MaxConnNum {
				server.mutexConns.Unlock()
				conn.Close()
				log.Error("too many connections")
				continue
			}
			server.conns[conn] = struct{}{}
			server.mutexConns.Unlock()

			server.wgConns.Add(1)

			kcpConn := newKCPConn(conn, server.PendingWriteNum, server.msgParser)
			agent := server.NewAgent(kcpConn)
			go func() {
				agent.Run()

				// cleanup
				kcpConn.Close()
				server.mutexConns.Lock()
				delete(server.conns, conn)
				server.mutexConns.Unlock()
				agent.OnClose()

				server.wgConns.Done()
			}()
		} else {
			log.Error("AcceptKCP error:%v", err.Error())
		}

	}
}

//Close close
func (server *KCPServer) Close() {
	server.ln.Close()
	server.wgLn.Wait()

	server.mutexConns.Lock()
	for conn := range server.conns {
		conn.Close()
	}
	server.conns = nil
	server.mutexConns.Unlock()
	server.wgConns.Wait()
}
