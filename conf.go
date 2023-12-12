package natsrpc

import (
	"encoding/json"
	"os"

	"net"

	"github.com/nats-io/nats.go"
	"github.com/wwqdrh/gokit/logger"
	"go.uber.org/zap"
)

type Conn interface {
	ReadMsg() ([]byte, error)
	WriteMsg(args []byte) error
	LocalAddr() net.Addr
	ID() int32
	Close()
}

type Config struct {
	Nats string `json:"nats"`
}

func ReadConfig(filename string) (*Config, error) {
	natsUrl := os.Getenv("NATS_RPC_URL")
	if natsUrl != "" {
		logger.DefaultLogger.Info("natsrpc", zap.String("natsUrl", natsUrl))
		return &Config{Nats: natsUrl}, nil
	}
	if filename == "" {
		return &Config{Nats: nats.DefaultURL}, nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	p := &Config{}
	err = json.NewDecoder(file).Decode(p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func IsExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
