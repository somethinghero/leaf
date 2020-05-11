package network

import (
	"errors"
	"net"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/somethinghero/leaf/log"
)

//WebsocketConnSet manage all connections
type WebsocketConnSet map[*websocket.Conn]struct{}

//WSConn wb socket connection
type WSConn struct {
	sync.Mutex
	conn      *websocket.Conn
	writeChan chan []byte
	maxMsgLen uint32
	closeFlag bool
	msgParser *MsgParser
}

func newWSConn(conn *websocket.Conn, pendingWriteNum int, maxMsgLen uint32, msgParser *MsgParser) *WSConn {
	wsConn := new(WSConn)
	wsConn.conn = conn
	wsConn.writeChan = make(chan []byte, pendingWriteNum)
	wsConn.maxMsgLen = maxMsgLen
	wsConn.msgParser = msgParser

	go func() {
		for b := range wsConn.writeChan {
			if b == nil {
				break
			}
			// log.Debug("WS WriteMessage len:%v", len(b))
			err := conn.WriteMessage(websocket.BinaryMessage, b)
			if err != nil {
				break
			}
		}

		conn.Close()
		wsConn.Lock()
		wsConn.closeFlag = true
		wsConn.Unlock()
	}()

	return wsConn
}

func (wsConn *WSConn) doDestroy() {
	wsConn.conn.UnderlyingConn().(*net.TCPConn).SetLinger(0)
	wsConn.conn.Close()

	if !wsConn.closeFlag {
		close(wsConn.writeChan)
		wsConn.closeFlag = true
	}
}

//Destroy destroy connection
func (wsConn *WSConn) Destroy() {
	wsConn.Lock()
	defer wsConn.Unlock()

	wsConn.doDestroy()
}

//Close close connection
func (wsConn *WSConn) Close() {
	wsConn.Lock()
	defer wsConn.Unlock()
	if wsConn.closeFlag {
		return
	}

	wsConn.doWrite(nil)
	wsConn.closeFlag = true
}

func (wsConn *WSConn) doWrite(b []byte) {
	if len(wsConn.writeChan) == cap(wsConn.writeChan) {
		log.Debug("close conn: channel full")
		wsConn.doDestroy()
		return
	}

	wsConn.writeChan <- b
}

//Write b must not be modified by the others goroutines
func (wsConn *WSConn) Write(b []byte) (int, error) {
	wsConn.Lock()
	defer wsConn.Unlock()
	if wsConn.closeFlag || b == nil {
		return 0, errors.New("conn closed")
	}

	wsConn.doWrite(b)
	return len(b), nil
}

//LocalAddr get local addr
func (wsConn *WSConn) LocalAddr() net.Addr {
	return wsConn.conn.LocalAddr()
}

//RemoteAddr get remote addr
func (wsConn *WSConn) RemoteAddr() net.Addr {
	return wsConn.conn.RemoteAddr()
}

//ReadMsg goroutine not safe
func (wsConn *WSConn) ReadMsg() ([]byte, error) {
	_, b, err := wsConn.conn.ReadMessage()
	return b, err
}

//WriteMsg args must not be modified by the others goroutines
func (wsConn *WSConn) WriteMsg(args ...[]byte) error {
	return wsConn.msgParser.Write(wsConn, args...)
}
