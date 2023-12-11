package natsrpc

import (
	"errors"
	"net"
	"reflect"
	"sync"

	"encoding/binary"
	"fmt"

	"hash/crc32"

	"github.com/wwqdrh/gokit/logger"
	"google.golang.org/protobuf/proto"
)

// 字符串转为16位整形哈希
func StringHash(s string) (hash uint16) {
	for _, c := range s {

		ch := uint16(c)

		hash = hash + ((hash) << 5) + ch + (ch << 7)
	}

	return
}

func CRC32Hash(s string) uint32 {
	return crc32.ChecksumIEEE([]byte(s))
}

type Mgr struct {
	wsAddr         string
	sesID2Client   map[int32]*Client
	sesMutex       sync.Mutex
	onNew, onClose func(conn Session)
	processor      *Processor
	worker         Worker
	wss            *WSServer

	close func()
}

func NewMgr(wsAddr string, worker Worker) *Mgr {
	p := &Mgr{
		wsAddr:       wsAddr,
		sesID2Client: make(map[int32]*Client),
		worker:       worker,
		processor:    NewProcessor(),
	}
	return p
}

func CheckArgs1MsgFun(cb interface{}) (err error, funValue reflect.Value, msgType reflect.Type) {
	cbType := reflect.TypeOf(cb)
	if cbType.Kind() != reflect.Func {
		err = errors.New("cb not a func")
		return
	}

	numArgs := cbType.NumIn()
	if numArgs != 2 {
		err = errors.New("cb param num args !=2")
		return
	}

	msgType = cbType.In(1)
	if msgType.Kind() != reflect.Ptr {
		err = errors.New("cb param args1 not ptr")
		return
	}

	funValue = reflect.ValueOf(cb)
	return
}

//func (p *Mgr) Init(wsAddr string, worker service.Worker) {
//	p.wsAddr = wsAddr
//	p.sesID2Client = make(map[int32]*Client)
//	p.worker = worker
//}

func (p *Mgr) Run() {
	//创建websocket server
	wss := NewWSServer(p.wsAddr, func(conn Conn) *Client {
		c := NewClient(conn, p)
		return c
	})
	p.wss = wss
	wss.Start()
	p.close = wss.Close
}

func (p *Mgr) ListenAddr() *net.TCPAddr {
	return p.wss.ListenAddr()
}

func (p *Mgr) Close() {
	p.close()
}

func (p *Mgr) Post(f func()) {
	p.worker.Post(f)
}

func (p *Mgr) RegisterEvent(onNew, onClose func(conn Session)) {
	p.onNew = onNew
	p.onClose = onClose
}

func (p *Mgr) GetSession(sesID int32) (Session, bool) {
	p.sesMutex.Lock()
	defer p.sesMutex.Unlock()

	v, ok := p.sesID2Client[sesID]
	return v, ok
}

//func (p *Mgr) RegisterSessionMsgHandler(msg proto.Message, f func(Session, proto.Message)) {
//	p.processor.Register(msg)
//	p.processor.SetHandler(msg, f)
//}

func (p *Mgr) RegisterSessionMsgHandler(cb interface{}) {
	err, funValue, msgType := CheckArgs1MsgFun(cb)
	if err != nil {
		logger.DefaultLogger.Errorx("RegisterServerMsgHandler: %s", nil, err.Error())
		return
	}
	msg := reflect.New(msgType).Elem().Interface().(proto.Message)
	p.processor.RegisterSessionMsgHandler(msg, func(s Session, message proto.Message) {
		funValue.Call([]reflect.Value{reflect.ValueOf(s), reflect.ValueOf(message)})
	})
}

func (p *Mgr) RegisterRawSessionMsgHandler(msg proto.Message, handler MsgHandler) {
	p.processor.RegisterSessionMsgHandler(msg, handler)
}

type Processor struct {
	littleEndian bool
	msgID2Info   map[uint32]*MsgInfo
}

type MsgInfo struct {
	msgType    reflect.Type
	msgHandler MsgHandler
}

type MsgHandler func(client Session, msg proto.Message)

func NewProcessor() *Processor {
	p := new(Processor)
	p.littleEndian = false
	p.msgID2Info = make(map[uint32]*MsgInfo)
	return p
}

func (p *Processor) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

//func (p *Processor) Register(msg proto.Message) {
//	msgID, msgType := util.ProtoHash(msg)
//	if _, ok := p.msgID2Info[msgID]; ok {
//		logrus.Errorf("message %s is already registered", msgType)
//		return
//	}
//
//	msgInfo := new(MsgInfo)
//	msgInfo.msgType = msgType
//	p.msgID2Info[msgID] = msgInfo
//	return
//}
//
//func (p *Processor) SetHandler(msg proto.Message, msgHandler MsgHandler) {
//	msgID, msgType := util.ProtoHash(msg)
//	msgInfo, ok := p.msgID2Info[msgID]
//	if !ok {
//		logrus.Errorf("message %s not registered", msgType)
//		return
//	}
//
//	msgInfo.msgHandler = msgHandler
//}

func (p *Processor) RegisterSessionMsgHandler(msg proto.Message, handler MsgHandler) {
	msgID, msgType := ProtoHash(msg)
	if _, ok := p.msgID2Info[msgID]; ok {
		logger.DefaultLogger.Errorx("message %s is already registered", nil, msgType)
		return
	}

	msgInfo := new(MsgInfo)
	msgInfo.msgType = msgType
	msgInfo.msgHandler = handler
	p.msgID2Info[msgID] = msgInfo
}

func ProtoHash(msg proto.Message) (uint32, reflect.Type) {
	msgName := proto.MessageName(msg)
	return CRC32Hash(string(msgName)), reflect.TypeOf(msg)
}

func (p *Processor) Handle(msg proto.Message, client Session) error {
	msgID, msgType := ProtoHash(msg)
	msgInfo, ok := p.msgID2Info[msgID]
	if !ok {
		logger.DefaultLogger.Errorx("message %s not registered", nil, msgType)
		return nil
	}

	if msgInfo.msgHandler != nil {
		msgInfo.msgHandler(client, msg)
	}

	return nil
}

func (p *Processor) Unmarshal(data []byte) (proto.Message, error) {
	if len(data) < 4 {
		return nil, errors.New("protobuf data too short")
	}

	var msgID uint32
	if p.littleEndian {
		msgID = binary.LittleEndian.Uint32(data)
	} else {
		msgID = binary.BigEndian.Uint32(data)
	}

	msgInfo, exist := p.msgID2Info[msgID]
	if !exist {
		return nil, errors.New(fmt.Sprintf("msgID:%d not registered", msgID))
	}

	msg := reflect.New(msgInfo.msgType.Elem()).Interface().(proto.Message)
	return msg, proto.Unmarshal(data[4:], msg.(proto.Message))
}

func (p *Processor) Marshal(msg proto.Message) ([]byte, error) {
	msgID, _ := ProtoHash(msg)

	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return p.Encode(msgID, data), nil
	//msgIDData := make([]byte, 2)
	//if p.littleEndian {
	//	binary.LittleEndian.PutUint16(msgIDData, msgID)
	//} else {
	//	binary.BigEndian.PutUint16(msgIDData, msgID)
	//}
	//
	////TODO 性能优化
	//ret := append(msgIDData, data...)
	//return ret, nil
}

func (p *Processor) Encode(msgID uint32, data []byte) []byte {
	msgIDData := make([]byte, 4)
	if p.littleEndian {
		binary.LittleEndian.PutUint32(msgIDData, msgID)
	} else {
		binary.BigEndian.PutUint32(msgIDData, msgID)
	}

	//TODO 性能优化
	ret := append(msgIDData, data...)
	return ret
}
