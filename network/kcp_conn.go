package network

import (
	"errors"
	"net"
	"sync"

	kcp "github.com/somethinghero/kcp-go"
	"github.com/somethinghero/leaf/log"
)

//KCPConnSet set off all connections
type KCPConnSet map[*kcp.UDPSession]struct{}

//KCPConn kcp connection
type KCPConn struct {
	sync.Mutex
	conn      *kcp.UDPSession
	writeChan chan []byte
	closeFlag bool
	msgParser *MsgParser
}

func newKCPConn(conn *kcp.UDPSession, pendingWriteNum int, msgParser *MsgParser) *KCPConn {
	kcpConn := new(KCPConn)
	kcpConn.conn = conn
	kcpConn.writeChan = make(chan []byte, pendingWriteNum)
	kcpConn.msgParser = msgParser

	go func() {
		for b := range kcpConn.writeChan {
			if b == nil {
				break
			}

			_, err := conn.Write(b)
			if err != nil {
				break
			}
		}

		conn.Close()
		kcpConn.Lock()
		kcpConn.closeFlag = true
		kcpConn.Unlock()
	}()

	return kcpConn
}

func (kcpConn *KCPConn) doDestroy() {
	kcpConn.conn.Close()

	if !kcpConn.closeFlag {
		close(kcpConn.writeChan)
		kcpConn.closeFlag = true
	}
}

//Destroy destroy
func (kcpConn *KCPConn) Destroy() {
	kcpConn.Lock()
	defer kcpConn.Unlock()

	kcpConn.doDestroy()
}

//Close close
func (kcpConn *KCPConn) Close() {
	kcpConn.Lock()
	defer kcpConn.Unlock()
	if kcpConn.closeFlag {
		return
	}

	kcpConn.doWrite(nil)
	kcpConn.closeFlag = true
}

func (kcpConn *KCPConn) doWrite(b []byte) {
	if len(kcpConn.writeChan) == cap(kcpConn.writeChan) {
		log.Debug("close conn: channel full")
		kcpConn.doDestroy()
		return
	}

	kcpConn.writeChan <- b
}

//Write b must not be modified by the others goroutines
func (kcpConn *KCPConn) Write(b []byte) (int, error) {
	kcpConn.Lock()
	defer kcpConn.Unlock()
	if kcpConn.closeFlag || b == nil {
		return 0, errors.New("conn closed")
	}

	kcpConn.doWrite(b)
	return len(b), nil
}

//Read read
func (kcpConn *KCPConn) Read(b []byte) (int, error) {
	return kcpConn.conn.Read(b)
}

//LocalAddr get local addr
func (kcpConn *KCPConn) LocalAddr() net.Addr {
	return kcpConn.conn.LocalAddr()
}

//RemoteAddr get remote addr
func (kcpConn *KCPConn) RemoteAddr() net.Addr {
	return kcpConn.conn.RemoteAddr()
}

//ReadMsg read msg
func (kcpConn *KCPConn) ReadMsg() ([]byte, error) {
	return kcpConn.msgParser.Read(kcpConn)
}

//WriteMsg write msg
func (kcpConn *KCPConn) WriteMsg(args ...[]byte) error {
	return kcpConn.msgParser.Write(kcpConn, args...)
}
