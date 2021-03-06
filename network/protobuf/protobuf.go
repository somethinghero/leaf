package protobuf

import (
	//"bytes"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/somethinghero/leaf/chanrpc"
	"github.com/somethinghero/leaf/log"
	"github.com/somethinghero/xxtea-go/xxtea"
	//"math"
	"reflect"
)

var (
	cryptKey = "skyyyloveyyforeverforever"
	keybuf   = []byte(cryptKey)
)

//Processor formate:
// -------------------------
// | id | protobuf message |
// -------------------------
type Processor struct {
	littleEndian bool
	msgInfo      map[string]*MsgInfo
}

//MsgInfo msg info
type MsgInfo struct {
	msgType       reflect.Type
	msgRouter     *chanrpc.Server
	msgHandler    MsgHandler
	msgRawHandler MsgHandler
}

//MsgHandler msg handler
type MsgHandler func([]interface{})

// type MsgRaw struct {
// 	msgID      uint16
// 	msgRawData []byte
// }

//NewProcessor new processor
func NewProcessor() *Processor {
	p := new(Processor)
	p.littleEndian = false
	p.msgInfo = make(map[string]*MsgInfo)
	return p
}

//SetByteOrder It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

//Register It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) Register(msg proto.Message) string {
	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("protobuf message pointer required")
	}
	msgName := proto.MessageName(msg)
	if _, ok := p.msgInfo[msgName]; ok {
		log.Fatal("message %s is already registered", msgType)
	}
	i := new(MsgInfo)
	i.msgType = msgType
	p.msgInfo[msgName] = i
	return msgName
}

//SetRouter It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetRouter(msg proto.Message, msgRouter *chanrpc.Server) {
	msgName := proto.MessageName(msg)
	_, ok := p.msgInfo[msgName]
	if !ok {
		log.Fatal("message %v not registered", msgName)
	}

	p.msgInfo[msgName].msgRouter = msgRouter
}

//SetHandler It's dangerous to call the method on routing or marshaling (unmarshaling)
func (p *Processor) SetHandler(msg proto.Message, msgHandler MsgHandler) {
	msgName := proto.MessageName(msg)
	_, ok := p.msgInfo[msgName]
	if !ok {
		log.Fatal("message %v not registered", msgName)
	}

	p.msgInfo[msgName].msgHandler = msgHandler
}

// It's dangerous to call the method on routing or marshaling (unmarshaling)
// func (p *Processor) SetRawHandler(id uint16, msgRawHandler MsgHandler) {
// 	// if id >= uint16(len(p.msgInfo)) {
// 	// 	log.Fatal("message id %v not registered", id)
// 	// }
// 	if _, ok := p.msgInfo[id]; !ok {
// 		log.Fatal("message id %v not registered", id)
// 	}
// 	p.msgInfo[id].msgRawHandler = msgRawHandler
// }

//Route goroutine safe
func (p *Processor) Route(msg interface{}, userData interface{}) error {
	// raw
	// if msgRaw, ok := msg.(MsgRaw); ok {
	// 	// if msgRaw.msgID >= uint16(len(p.msgInfo)) {
	// 	// 	return fmt.Errorf("message id %v not registered", msgRaw.msgID)
	// 	// }
	// 	if _, ok := p.msgInfo[msgRaw.msgID]; !ok {
	// 		return fmt.Errorf("message id %v not registered", msgRaw.msgID)
	// 	}
	// 	i := p.msgInfo[msgRaw.msgID]
	// 	if i.msgRawHandler != nil {
	// 		i.msgRawHandler([]interface{}{msgRaw.msgID, msgRaw.msgRawData, userData})
	// 	}
	// 	return nil
	// }

	// protobuf
	//msgType := reflect.TypeOf(msg)
	protoMsg, ok := msg.(proto.Message)
	if !ok {
		return fmt.Errorf("only surport proto msg")
	}
	msgName := proto.MessageName(protoMsg)
	i, ok := p.msgInfo[msgName]
	if !ok {
		return fmt.Errorf("message %v not registered", msgName)
	}
	if i.msgHandler != nil {
		i.msgHandler([]interface{}{msg, userData})
	}
	if i.msgRouter != nil {
		i.msgRouter.Go(msgName, msg, userData)
	}
	return nil
}

//Unmarshal goroutine safe
func (p *Processor) Unmarshal(data []byte) (interface{}, error) {
	if len(data) < 2 {
		return nil, errors.New("protobuf data too short 1")
	}

	// namelen
	var namelen uint16
	if p.littleEndian {
		namelen = binary.LittleEndian.Uint16(data)
	} else {
		namelen = binary.BigEndian.Uint16(data)
	}
	if namelen <= 0 {
		return nil, errors.New("protobuf namelen too short")
	}
	if len(data) < (2 + int(namelen)) {
		return nil, errors.New("protobuf data too short 2")
	}
	//name
	name := string(data[2 : 2+namelen])
	// msg
	i, ok := p.msgInfo[name]
	if !ok {
		return nil, fmt.Errorf("message name %v not registered", name)
	}
	if i.msgRawHandler != nil {
		return nil, nil
	}
	msg := reflect.New(i.msgType.Elem()).Interface()
	//decrypt
	decryptdata, _ := xxtea.DecryptExt(data[2+namelen:], keybuf)
	return msg, proto.UnmarshalMerge(decryptdata, msg.(proto.Message))
}

//Marshal goroutine safe
func (p *Processor) Marshal(msg interface{}) ([][]byte, error) {
	//msgType := reflect.TypeOf(msg)

	protoMsg, ok := msg.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("only surport proto msg")
	}
	msgName := proto.MessageName(protoMsg)

	bufNamelen := make([]byte, 2)
	bufName := []byte(msgName)
	namelen := len(bufName)

	if p.littleEndian {
		binary.LittleEndian.PutUint16(bufNamelen, uint16(namelen))
	} else {
		binary.BigEndian.PutUint16(bufNamelen, uint16(namelen))
	}
	// data
	data, err := proto.Marshal(msg.(proto.Message))
	//encrypt
	endata := xxtea.EncryptExt(data, keybuf)
	// fmt.Printf("Marshal endata len:%v\n", len(endata))
	// fmt.Printf("Marshal all len:%v\n", len(bufNamelen)+len(bufName)+len(endata))
	return [][]byte{bufNamelen, bufName, endata}, err
}

// goroutine safe
// func (p *Processor) Range(f func(id uint16, t reflect.Type)) {
// 	for id, i := range p.msgInfo {
// 		f(uint16(id), i.msgType)
// 	}
// }
