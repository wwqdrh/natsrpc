package natsrpc

import (
	"github.com/wwqdrh/gokit/logger"
	"google.golang.org/protobuf/proto"

	"reflect"
)

type Session interface {
	SendMsg(msg proto.Message)
	SendRawMsg(msgID uint32, data []byte)
	ID() int32
	Close()
}

type Client struct {
	conn Conn
	mgr  *Mgr
}

func NewClient(conn Conn, mgr *Mgr) *Client {
	p := &Client{
		conn: conn,
		mgr:  mgr,
	}
	return p
}

func (p *Client) ReadLoop() {
	for {
		data, err := p.conn.ReadMsg()
		if err != nil {
			logger.DefaultLogger.Errorx("read message: %s", nil, err.Error())
			break
		}

		msg, err := p.mgr.processor.Unmarshal(data)
		if err != nil {
			logger.DefaultLogger.Errorx("unmarshal message error: %v", nil, err)
			break
		}
		p.mgr.Post(func() {
			err = p.mgr.processor.Handle(msg, p)
		})
	}
}

func (p *Client) OnNew() {
	p.mgr.Post(func() {
		p.mgr.sesID2Client[p.conn.ID()] = p
		p.mgr.onNew(p)
	})
}

func (p *Client) OnClose() {
	p.mgr.Post(func() {
		delete(p.mgr.sesID2Client, p.conn.ID())
		p.mgr.onClose(p)
	})
}

func (p *Client) SendMsg(msg proto.Message) {
	data, err := p.mgr.processor.Marshal(msg)
	if err != nil {
		logger.DefaultLogger.Errorx("marshal message %v error: %v", nil, reflect.TypeOf(msg), err)
		return
	}
	err = p.conn.WriteMsg(data)
	if err != nil {
		logger.DefaultLogger.Errorx("write message %v error: %v", nil, reflect.TypeOf(msg), err)
	}
}

func (p *Client) SendRawMsg(msgID uint32, data []byte) {
	newData := p.mgr.processor.Encode(msgID, data)
	err := p.conn.WriteMsg(newData)
	if err != nil {
		logger.DefaultLogger.Errorx("write message error: %v", nil, err)
	}
}

func (p *Client) ID() int32 {
	return p.conn.ID()
}

func (p *Client) Close() {
	p.conn.Close()
}
