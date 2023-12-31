package stub

import (
	"time"

	"github.com/wwqdrh/natsrpc"
	"github.com/wwqdrh/natsrpc/engine"
	"google.golang.org/protobuf/proto"
)

type Gate struct {
	worker      natsrpc.Worker
	rpc         *engine.RPC
	networkMgr  *natsrpc.Mgr
	onCloseFuns []func()
}

func NewGate(serverID int32, addr string, config natsrpc.Config) (*Gate, error) {
	p := new(Gate)
	p.worker = natsrpc.NewWorker()
	p.networkMgr = natsrpc.NewMgr(addr, p.worker)
	rpc, err := engine.NewRPC(serverID, p.worker, config.Nats)
	if err != nil {
		return nil, err
	}
	p.rpc = rpc
	p.rpc.RegisterSend2Session(func(sesID int32, msgID uint32, data []byte) {
		if ses, ok := p.GetSession(sesID); ok {
			ses.SendRawMsg(msgID, data)
		}
	})
	return p, nil
}

// TODO 添加关闭信号
func (p *Gate) Run() {
	p.worker.Run()
	p.rpc.Run()
	p.networkMgr.Run()
}

func (p *Gate) Close() {
	for _, v := range p.onCloseFuns {
		v()
	}
	p.rpc.Close()
	p.worker.Close()
}

// 注册关闭事件 worker线程外
func (p *Gate) RegisterCloseFunc(f func()) {
	p.onCloseFuns = append(p.onCloseFuns, f)
}

func (p *Gate) Post(f func()) {
	p.worker.Post(f)
}

func (p *Gate) AfterPost(duration time.Duration, f func()) {
	p.worker.AfterPost(duration, f)
}

func (p *Gate) RegisterNetWorkEvent(onNew, onClose func(conn natsrpc.Session)) {
	p.networkMgr.RegisterEvent(onNew, onClose)
}

func (p *Gate) RegisterSessionMsgHandler(cb interface{}) {
	p.networkMgr.RegisterSessionMsgHandler(cb)
}

func (p *Gate) RegisterRequestMsgHandler(cb interface{}) {
	p.rpc.RegisterRequestMsgHandler(cb)
}

func (p *Gate) RegisterServerHandler(cb interface{}) {
	p.rpc.RegisterServerMsgHandler(cb)
}

func (p *Gate) GetServerById(serverID int32) engine.Server {
	return p.rpc.GetServerById(serverID)
}

func (p *Gate) GetSession(sesID int32) (natsrpc.Session, bool) {
	return p.networkMgr.GetSession(sesID)
}

func (p *Gate) RouteSessionMsg(msg proto.Message, serverID int32) {
	p.networkMgr.RegisterRawSessionMsgHandler(msg, func(s natsrpc.Session, msg proto.Message) {
		p.GetServerById(serverID).RouteSession2Server(s.ID(), msg)
	})
}

func (p *Gate) RegisterRawSessionMsgHandler(msg proto.Message, f func(s natsrpc.Session, message proto.Message)) {
	p.networkMgr.RegisterRawSessionMsgHandler(msg, f)
}
