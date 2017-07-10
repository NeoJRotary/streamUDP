package stream

import (
	"net"
	"sync"
	"time"
)

// UDP ...
type UDP struct {
	Listener       *net.UDPConn
	i              uint32
	mutex          *sync.Mutex
	writers        [100000]*writer
	MaxSize        int
	WriteTimeout   time.Duration
	ReadTimeout    time.Duration
	WriteTimeLimit uint8
	servs          map[string]*serv
	close          bool
}

// Src ...
type Src struct {
	I    uint32
	Addr *net.UDPAddr
}

// Serv ...
type serv struct {
	url  string
	addr *net.UDPAddr
}

type writer struct {
	i   uint32
	buf (chan []byte)
}

// Request ...
type Request struct {
	Serv  string                 `json:"serv"`
	Type  string                 `json:"type"`
	Group string                 `json:"group"`
	User  string                 `json:"user"`
	Data  map[string]interface{} `json:"data"`
}

// Response ...
type Response struct {
	Result int
	Data   map[string]interface{}
}

// Push ...
type Push struct {
	Serv string                 `json:"serv"`
	User []interface{}          `json:"user"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}
