package stub

import (
	"github.com/wwqdrh/natsrpc"
	"github.com/wwqdrh/natsrpc/engine"
)

type Server struct {
	worker   natsrpc.Worker
	rpc      *engine.RPC
	serverID int32
}

func NewServer(serverID int32, config natsrpc.Config) (*Server, error) {
	p := new(Server)
	p.worker = natsrpc.NewWorker()
	rpc, err := engine.NewRPC(serverID, p.worker, config.Nats)
	if err != nil {
		return nil, err
	}
	p.rpc = rpc
	p.serverID = serverID
	return p, nil
}

// TODO 添加关闭信号
func (p *Server) Run() {
	p.worker.Run()
	p.rpc.Run()
}

func (p *Server) Worker() natsrpc.Worker {
	return p.worker
}

func (p *Server) Post(f func()) {
	p.worker.Post(f)
}

func (p *Server) RegisterRequestMsgHandler(cb interface{}) {
	p.rpc.RegisterRequestMsgHandler(cb)
}

func (p *Server) GetServerById(serverID int32) engine.Server {
	return p.rpc.GetServerById(serverID)
}

//
//func (p *Server) RegisterServerMsg(msg proto.Message, f func(rpc.Server, proto.Message)) {
//	p.rpc.RegisterServerMsg(msg, f)
//}

func (p *Server) RegisterSessionMsgHandler(cb interface{}) {
	p.rpc.RegisterSessionMsgHandler(cb)
}

func (p *Server) RegisterServerHandler(cb interface{}) {
	p.rpc.RegisterServerMsgHandler(cb)
}

func (p *Server) ID() int32 {
	return p.serverID
}

func (p *Server) RPCSession(s engine.GateSessionID) engine.Session {
	return p.rpc.Session(s.GateID, s.SesID)
}
