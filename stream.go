/*

Stream UDP Protocal

Default R/W Buffer Size : 65536 bytes

Packet Format
[src index][remote index][data...]
src index : 4 bytes LittleEndian uint32
src index : 4 bytes LittleEndian uint32
data : byte slice

if len(packet) == 4
> ACK

buf[index][0]
0 : waiting for ACK only
1 : waiting for ACK then wait for Response
2 : waiting for Response wirte
3 : waiting for Response be read

*/

package stream

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"errors"
	"net"
	"runtime"
	"sync"
	"time"
)

// GetStream ...
func GetStream(listen string, list map[string]string) (*UDP, error) {
	gob.Register(map[string]interface{}{})
	gob.Register([]map[string]interface{}{})
	gob.Register([]interface{}{})

	laddr, err := net.ResolveUDPAddr("udp4", listen)
	listener, err := net.ListenUDP("udp4", laddr)
	if err != nil {
		return nil, err
	}

	udp := UDP{
		Listener:       listener,
		i:              1,
		mutex:          new(sync.Mutex),
		MaxSize:        65536,
		WriteTimeout:   2 * time.Second,
		WriteTimeLimit: 5,
		ReadTimeout:    10 * time.Second,
		servs:          make(map[string]*serv),
		close:          false,
	}

	if list == nil {
		list = make(map[string]string)
	}
	list["listen"] = listen
	for name, url := range list {
		err = udp.register(name, url)
		if err != nil {
			return nil, err
		}
	}

	return &udp, nil
}

// Close ...
func (udp *UDP) Close() {
	udp.close = true
	udp.Listener.Close()
}

// Listen ...
func (udp *UDP) Listen() (*Request, *Src, []byte, error) {
	for {
		data := make([]byte, udp.MaxSize)
		n, addr, err := udp.Listener.ReadFromUDP(data)
		if err != nil {
			if udp.close {
				return nil, nil, nil, nil
			}
			return nil, nil, nil, err
		}
		if n < 4 {
			continue
		}
		i := binary.LittleEndian.Uint32(data[0:4])
		if n == 4 {
			go udp.writers[i].write([]byte{6})
		}
		if n >= 8 {
			go udp.Listener.WriteToUDP(data[4:8], addr)
		}
		if n > 8 {
			if i == 0 {
				buf := bytes.NewBuffer(make([]byte, 0, udp.MaxSize))
				buf.Write(data[8:n])
				req := &Request{}
				err = gob.NewDecoder(buf).Decode(req)
				if err != nil {
					return nil, nil, nil, err
				}
				src := &Src{
					Addr: addr,
					I:    binary.LittleEndian.Uint32(data[4:8]),
				}
				return req, src, data[8:n], nil
			}
			go udp.writers[i].write(data[8:n])
		}
	}
}

func (udp *UDP) newWriter(src uint32) (*writer, []byte) {
	udp.mutex.Lock()
	i := udp.i
	udp.i++
	if udp.i == 100000 {
		udp.i = 1
	}
	if i%1000 == 0 {
		runtime.GC()
	}
	udp.mutex.Unlock()
	udp.writers[i] = &writer{
		i:   i,
		buf: make(chan []byte),
	}
	bi := make([]byte, 4)
	bsrc := []byte{0, 0, 0, 0}
	binary.LittleEndian.PutUint32(bi, i)
	if src != 0 {
		binary.LittleEndian.PutUint32(bsrc, src)
	}
	return udp.writers[i], append(bsrc, bi...)
}

func (w *writer) release() {
	w = nil
}

func (w *writer) write(data []byte) {
	w.buf <- data
}

func (w *writer) read(dur time.Duration) ([]byte, bool) {
	select {
	case <-time.After(dur):
		return nil, true
	case data := <-w.buf:
		return data, false
	}
}

func (udp *UDP) send(w *writer, data []byte, addr *net.UDPAddr) error {
	count := uint8(0)
	for count < udp.WriteTimeLimit {
		udp.Listener.WriteToUDP(data, addr)
		data, timeout := w.read(udp.WriteTimeout)
		if !timeout {
			if data[0] == 6 {
				return nil
			} else if data[0] == 5 {
				return errors.New("Target Serv Cant recognize you")
			}
		}
		count++
	}
	return errors.New("Stream send ACK timeout")
}

// Write error
func (udp *UDP) Write(data []byte, servName string) error {
	w, bi := udp.newWriter(0)
	defer w.release()
	s := udp.getServ(servName)
	err := udp.send(w, append(bi, data...), s.addr)
	return err
}

// Push error
func (udp *UDP) Push(push *Push, servName string) error {
	w, bi := udp.newWriter(0)
	defer w.release()
	s := udp.getServ(servName)

	obj, err := json.Marshal(push)
	if err != nil {
		return err
	}
	err = udp.send(w, append(bi, obj...), s.addr)
	return err
}

// SendReq error
func (udp *UDP) SendReq(req *Request, servName string) error {
	w, bi := udp.newWriter(0)
	defer w.release()
	s := udp.getServ(servName)

	data, err := udp.reqEncode(req)
	if err != nil {
		return err
	}
	err = udp.send(w, append(bi, data...), s.addr)
	return err
}

// Request ...
func (udp *UDP) Request(req *Request, servName string) (*Response, error) {
	w, bi := udp.newWriter(0)
	defer w.release()
	s := udp.getServ(servName)

	data, err := udp.reqEncode(req)
	if err != nil {
		return nil, err
	}
	err = udp.send(w, append(bi, data...), s.addr)
	if err != nil {
		return nil, err
	}
	var timeout bool
	data, timeout = w.read(udp.ReadTimeout)
	if timeout {
		return nil, errors.New("Stream Request Response timeout")
	}

	res, err := udp.resDecode(data)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Response error
func (udp *UDP) Response(res *Response, src *Src) error {
	w, bi := udp.newWriter(src.I)
	defer w.release()

	buf, err := udp.resEncode(res)
	if err != nil {
		return err
	}

	err = udp.send(w, append(bi, buf...), src.Addr)
	return err
}
