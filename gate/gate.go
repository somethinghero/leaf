package gate

import (
	"net"
	"reflect"
	"time"

	"github.com/somethinghero/leaf/chanrpc"
	"github.com/somethinghero/leaf/log"
	"github.com/somethinghero/leaf/network"
)

//Gate Gate
type Gate struct {
	MaxConnNum      int
	PendingWriteNum int
	MaxMsgLen       uint32
	Processor       network.Processor
	AgentChanRPC    *chanrpc.Server
	KCPProcessor    network.Processor
	KCPAgentChanRPC *chanrpc.Server

	// websocket
	WSAddr      string
	HTTPTimeout time.Duration
	CertFile    string
	KeyFile     string

	// tcp
	TCPAddr      string
	LenMsgLen    int
	LittleEndian bool

	//kcp
	KCPAddr         string
	KCPLenMsgLen    int
	KCPLittleEndian bool
}

//Run Run
func (gate *Gate) Run(closeSig chan bool) {
	var wsServer *network.WSServer
	if gate.WSAddr != "" {
		wsServer = new(network.WSServer)
		wsServer.Addr = gate.WSAddr
		wsServer.MaxConnNum = gate.MaxConnNum
		wsServer.PendingWriteNum = gate.PendingWriteNum
		wsServer.MaxMsgLen = gate.MaxMsgLen
		wsServer.HTTPTimeout = gate.HTTPTimeout
		wsServer.CertFile = gate.CertFile
		wsServer.KeyFile = gate.KeyFile
		wsServer.NewAgent = func(conn *network.WSConn) network.Agent {
			a := &agent{conn: conn, processor: gate.Processor, rpc: gate.AgentChanRPC}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go("NewAgent", a)
			}
			return a
		}
	}

	var tcpServer *network.TCPServer
	if gate.TCPAddr != "" {
		tcpServer = new(network.TCPServer)
		tcpServer.Addr = gate.TCPAddr
		tcpServer.MaxConnNum = gate.MaxConnNum
		tcpServer.PendingWriteNum = gate.PendingWriteNum
		tcpServer.LenMsgLen = gate.LenMsgLen
		tcpServer.MaxMsgLen = gate.MaxMsgLen
		tcpServer.LittleEndian = gate.LittleEndian
		tcpServer.NewAgent = func(conn *network.TCPConn) network.Agent {
			a := &agent{conn: conn, processor: gate.Processor, rpc: gate.AgentChanRPC}
			if gate.AgentChanRPC != nil {
				gate.AgentChanRPC.Go("NewAgent", a)
			}
			return a
		}
	}

	var kcpServer *network.KCPServer
	if gate.KCPAddr != "" {
		kcpServer = new(network.KCPServer)
		kcpServer.Addr = gate.KCPAddr
		kcpServer.MaxConnNum = gate.MaxConnNum
		kcpServer.PendingWriteNum = gate.PendingWriteNum
		kcpServer.LenMsgLen = gate.KCPLenMsgLen
		kcpServer.MaxMsgLen = gate.MaxMsgLen
		kcpServer.LittleEndian = gate.KCPLittleEndian
		kcpServer.NewAgent = func(conn *network.KCPConn) network.Agent {
			a := &agent{conn: conn, processor: gate.KCPProcessor, rpc: gate.KCPAgentChanRPC}
			if gate.KCPAgentChanRPC != nil {
				gate.KCPAgentChanRPC.Go("NewKCPAgent", a)
			}
			return a
		}
	}

	if wsServer != nil {
		wsServer.Start()
	}
	if tcpServer != nil {
		tcpServer.Start()
	}
	if kcpServer != nil {
		kcpServer.Start()
	}
	<-closeSig
	if wsServer != nil {
		wsServer.Close()
	}
	if tcpServer != nil {
		tcpServer.Close()
	}
	if kcpServer != nil {
		kcpServer.Close()
	}
}

//OnDestroy OnDestroy
func (gate *Gate) OnDestroy() {}

type agent struct {
	conn      network.Conn
	processor network.Processor
	rpc       *chanrpc.Server
	userData  interface{}
}

func (a *agent) Run() {
	for {
		data, err := a.conn.ReadMsg()
		if err != nil {
			log.Debug("read message: %v", err)
			break
		}
		if a.processor != nil {
			msg, err := a.processor.Unmarshal(data)
			if err != nil {
				log.Debug("unmarshal message error: %v", err)
				break
			}
			err = a.processor.Route(msg, a)
			if err != nil {
				log.Debug("route message error: %v", err)
				break
			}
		}
	}
}

func (a *agent) OnClose() {
	if a.rpc != nil {
		err := a.rpc.Call0("CloseAgent", a)
		if err != nil {
			log.Error("chanrpc error: %v", err)
		}
	}
}

func (a *agent) WriteMsg(msg interface{}) {
	if a.processor != nil {
		data, err := a.processor.Marshal(msg)
		if err != nil {
			log.Error("marshal message %v error: %v", reflect.TypeOf(msg), err)
			return
		}
		err = a.conn.WriteMsg(data...)
		if err != nil {
			log.Error("write message %v error: %v", reflect.TypeOf(msg), err)
		}
	}
}

func (a *agent) LocalAddr() net.Addr {
	return a.conn.LocalAddr()
}

func (a *agent) RemoteAddr() net.Addr {
	return a.conn.RemoteAddr()
}

func (a *agent) Close() {
	a.conn.Close()
}

func (a *agent) Destroy() {
	a.conn.Destroy()
}

func (a *agent) UserData() interface{} {
	return a.userData
}

func (a *agent) SetUserData(data interface{}) {
	a.userData = data
}
