# natsrpc

a rpc framework base on nats

# usage

## gate

```go
mgr := &clientMgr{
    clients: make(map[natsrpc.Session]struct{}),
}

g, err := stub.NewGate(100, *addr, natsrpc.Config{Nats: "127.0.0.1:4222"})
if err != nil {
    fmt.Println(err)
    return
}

g.RegisterNetWorkEvent(func(conn natsrpc.Session) {
    mgr.clients[conn] = struct{}{}
}, func(conn natsrpc.Session) {
    delete(mgr.clients, conn)
})
g.RouteSessionMsg((*pb.ReqHello)(nil), BServerID)

g.Run()
```

## server

```go
s, err := stub.NewServer(101, natsrpc.Config{Nats: "127.0.0.1:4222"})
if err != nil {
    fmt.Println(err)
    return
}

//注册客户端消息事件handler
s.RegisterSessionMsgHandler(func(client engine.Session, req *pb.ReqHello) {
    resp := &pb.RespHello{Name: "回复您的请求"}
    fmt.Println("收到客户端来的消息:", req.Name)
    client.SendMsg(resp)
})

//注册send事件handler
s.RegisterServerHandler(func(server engine.Server, req *pb.ReqSend) {
    fmt.Println("收到gate来的消息:", req.Name)
})

//注册request事件handler
s.RegisterRequestMsgHandler(func(server engine.RequestServer, req *pb.ReqRequest) {
    resp := &pb.RespRequest{Name: "我是request返回消息"}
    fmt.Println("收到gate来的request消息:", req.Name)
    server.Answer(resp)
})

//注册call事件handler
s.RegisterRequestMsgHandler(func(server engine.RequestServer, req *pb.ReqCall) {
    resp := &pb.RespCall{Name: "我是call返回消息"}
    fmt.Println("收到gate来的call消息:", req.Name)
    server.Answer(resp)
})

s.Run()
```