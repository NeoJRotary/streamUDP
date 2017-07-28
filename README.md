# StreamUDP 0.1.0
> light and fast UDP communicate package with gob

Local Network is stable and fast, why not try UDP to make communication faster? It is recommend to change the Request and Response struct in `struct.go` for your needs.

## Default Settings
- MaxSize : 65535 ( buffer size of read )
- WriteTimeout : 2s ( Duration of write ACK )
- WriteTimeLimit : 5 ( Retry times of write )
- ReadTimeou : 10s ( Duration of read timeout )

## Methods
**func GetStream(listen string, list map[string]string) (\*UDP, error)**  
Return `*UDP` struct.  
\> listen : address string to listen on  
\> list : service list, see example usage for detail

**func (\*UDP) Listen() (\*Request, \*Src, []byte, error)**  
Return listen result.  
\> `*Request` : will be return if no error  
\> `*Src` : request from whom. Will need it to do `*UDP.Response()`  
\> []byte : original byte slice of Request. Can be used at `*UDP.Write()`  
\> error : error object  

**func (\*UDP) Close()**  
Close Listener.  

**func (\*UDP) Write(data []byte, servName string) error**  
Write byte slice to service. Useful when current service just do request routing, you can directly send []byte which get from `*UDP.Listen()` without use `*UDP.SendReq()` to skip gob encode.  
\> data : byte slice to send  
\> servName : service name  

**func (\*UDP) Push(push \*Push, servName string) error**  
Push JSON data to service, useful for communicating with services outside local GO enviroment. See example below for more info.  
\> push : `*Push` struct, this will convert to JSON format  
\> servName : service name  

**func (\*UDP) SendReq(req \*Request, servName string) error**  
Send `*Request` without waiting for `*Response` struct.  
\> req : `*Request` struct  
\> servName : service name  

**func (\*UDP) Request(req \*Request, servName string) (\*Response, error)**  
Send `*Request` then wait target service response.  
\> req : `*Request` struct  
\> servName : service name  

**func (\*UDP) Response(res \*Response, src \*Src)**  
Response other service's `*Request`.  
\> res : `*Response` struct  
\> src : `*Src` struct. Which get from `*UDP.Listen()`   


## Example Usage:
```
package main

import (
  "time"

  stream "./packages/streamUDP"
)

func main() {
  serv.udp = "localhost:9903"

  // list of other services in same enviroment
  udpServs := map[string]string{
    "rest":     "localhost:9900",
    "host":     "localhost:9901",
    "db":       "localhost:9902",
  }

  streamer, err = stream.GetStream(serv.udp, udpServs)
  if err != nil {
    fmt.Fatalln("stream init fail: " + err.Error())
  }
  defer streamer.Close()

  // change settings
  streamer.MaxSize = 4096
  streamer.WriteTimeout = 2 * time.Second
  streamer.WriteTimeLimit = 5
  streamer.ReadTimeout = 10 * time.Second

  // must run listen before make any request/write
  go func() {
    for {
      req, src, _, err := streamer.Listen()
      // streamer close
      if req == nil && err == nil {
        break
      }
      // check error
      if err != nil {
        fmt.Println("Listener Error :" + err.Error())
      } else {
        go streamAPI(req, src)
      }
    }
  }()

  // create a *Request
  req := &stream.Request{
    Type: "GetData",
    Data: map[string]string{"id": "12345"},
  }

  // send *Request without waiting for *Response
  err := streamer.SendReq(req, "host")
  if err != nil {
    fmt.Println(err)
  }

  // make *Request
  res, err := streamer.Request(req, "db")

  if err != nil {
    fmt.Println(err)
  } else {
    fmt.Println(res)
  }
}
```

## Service Not GO
If your target service is not GO process, you have to do ACK response at it's listen. Following is Node.JS example :
```
const dgram = require('dgram');

const udp = dgram.createSocket('udp4');

udp.on('message', (msg, rinfo) => {
  // here is ACK response
  udp.send(msg.slice(4, 8), rinfo.port, rinfo.address);

  // Get Push Data
  const push = JSON.parse(msg.slice(8).toString());
  console.log(push);
});

udp.bind(9900, 'localhost');
```

## Others
Below is some of my experiences on this project, if you have better solution or any advice please let me know : )  

**Why Array instead Channel**  
If you read my codes, you can find that I prepare an array and looping index to buffer the request. I built a pure `channel` version before but it can't pass stress test. When concurrent requests in high frequency keep coming, large amounts of goroutines will be created and GO have to keep searching who is waiting for particular `channel`. It is going to cost your memory and may over W/R timeout ( or you have to make it longer ). Tough the `Array` way need to do `Mutex.Lock()` to prevent concurrent read ( different requests may get same index ) which lock whole process. Instead, with manually GC it can control memory in a stable amount.

**Why not re-use Gob encoder/decoder**  
I built a "gob re-use" version before, but same issue; it can't pass stress test. Reusing encoder/decoder can reduce data size and memory usage in low stress situation. When concurrent requests come faster then encode/decode speed, process have to queue data and wait until encoder/decoder is available. Lots of goroutines is blocked and waiting but usually CPU have resource can do it concurrently.
