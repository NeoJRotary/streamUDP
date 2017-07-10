package stream

import (
	"bytes"
	"encoding/gob"
	"errors"
	"net"
)

func (udp *UDP) register(name string, url string) error {
	addr, err := net.ResolveUDPAddr("udp4", url)
	if err != nil {
		return err
	}
	udp.servs[name] = &serv{
		url:  url,
		addr: addr,
	}
	return nil
}

func (udp *UDP) getServ(servName string) *serv {
	return udp.servs[servName]
}

func (udp *UDP) resolveServ(addr *net.UDPAddr) (string, *serv, error) {
	for k, v := range udp.servs {
		if v.addr.String() == addr.String() {
			return k, v, nil
		}
	}
	return "", nil, errors.New("cant resolve target : " + addr.String())
}

func (udp *UDP) reqEncode(req *Request) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, udp.MaxSize))
	err := gob.NewEncoder(buf).Encode(req)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (udp *UDP) reqDecode(data []byte) (*Request, error) {
	buf := bytes.NewBuffer(data)
	req := &Request{}
	err := gob.NewDecoder(buf).Decode(req)
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (udp *UDP) resEncode(res *Response) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, udp.MaxSize))
	err := gob.NewEncoder(buf).Encode(res)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (udp *UDP) resDecode(data []byte) (*Response, error) {
	buf := bytes.NewBuffer(data)
	res := &Response{}
	err := gob.NewDecoder(buf).Decode(res)
	if err != nil {
		return nil, err
	}
	return res, nil
}
