package natsrpc

import (
	"errors"
	"net"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/wwqdrh/gokit/logger"
	"go.uber.org/zap"
)

type WSConn struct {
	sync.Mutex
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	closeFlag bool

	connid int32
}

func NewWSConn(conn *websocket.Conn, connid int32) *WSConn {
	wsc := new(WSConn)
	wsc.send = make(chan []byte, 256)
	wsc.conn = conn
	wsc.connid = connid
	return wsc
}

func (p *WSConn) writePump() {
	for data := range p.send {
		if data == nil {
			break
		}
		err := p.conn.WriteMessage(websocket.BinaryMessage, data)
		if err != nil {
			break
		}
	}
	p.conn.Close()
	p.Lock()
	p.closeFlag = true
	p.Unlock()
}

func (p *WSConn) WriteMsg(args []byte) error {
	p.Lock()
	defer p.Unlock()

	if p.closeFlag {
		return errors.New("socket closeFlag is true")
	}

	if len(p.send) == cap(p.send) {
		close(p.send)
		p.closeFlag = true
		return errors.New("send buffer full")
	}

	p.send <- args
	return nil
}

func (p *WSConn) doWrite(data []byte) {
	p.send <- data
}

func (p *WSConn) ReadMsg() ([]byte, error) {
	_, data, err := p.conn.ReadMessage()
	return data, err
}

func (p *WSConn) ID() int32 {
	return p.connid
}

func (p *WSConn) Close() {
	p.Lock()
	defer p.Unlock()
	if p.closeFlag {
		return
	}
	p.doWrite(nil)
	p.closeFlag = true
}

func (p *WSConn) LocalAddr() net.Addr {
	return p.conn.LocalAddr()
}

func (p *WSConn) RemoteAddr() net.Addr {
	return p.conn.RemoteAddr()
}

// 这里可以定义handshake规则：超时时间等
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	Subprotocols: []string{"avatar-fight"},
} // use default options

type WSServer struct {
	sync.Mutex
	addr      string
	wg        sync.WaitGroup
	conns     map[*websocket.Conn]struct{}
	connid    int32
	newClient func(conn Conn) *Client
	ln        net.Listener

	close func()
}

func NewWSServer(addr string, newClient func(conn Conn) *Client) *WSServer {
	return &WSServer{
		addr:      addr,
		newClient: newClient,
		conns:     make(map[*websocket.Conn]struct{}),
	}
}

func (p *WSServer) Start() error {
	ln, err := net.Listen("tcp", p.addr)
	if err != nil {
		logger.DefaultLogger.Fatal("启动失败，端口被占用", zap.String("addr", p.addr))
		return err
	}

	p.ln = ln
	httpSvr := &http.Server{
		Addr:    p.addr,
		Handler: p,
	}

	p.close = func() {
		httpSvr.Close()
	}
	go func() {
		err := httpSvr.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			logger.DefaultLogger.Fatalx("WSServer Serve: %s", nil, err.Error())
			return
		}
	}()
	return nil
}

func (p *WSServer) Close() {
	p.close()
}

func (p *WSServer) ListenAddr() *net.TCPAddr {
	return p.ln.Addr().(*net.TCPAddr)
}

func (p *WSServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.DefaultLogger.Errorx("ServerHttp upgrader.Upgrade", nil, err.Error())
		return
	}

	p.wg.Add(1)
	defer p.wg.Done()

	p.Lock()
	p.conns[conn] = struct{}{}
	p.Unlock()

	connid := p.NewConnID()
	wsc := NewWSConn(conn, connid)
	go wsc.writePump()

	c := p.newClient(wsc)
	c.OnNew()
	c.ReadLoop()

	wsc.Close()
	p.Lock()
	delete(p.conns, conn)
	p.Unlock()
	c.OnClose()
}

func (p *WSServer) NewConnID() int32 {
	return atomic.AddInt32(&p.connid, 1)
}
