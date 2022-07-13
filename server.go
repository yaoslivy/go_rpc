package go_rpc

import (
	"encoding/json"
	"fmt"
	"go_rpc/codec"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
)

// 通信过程中需要协商消息的编解码方式

const MagicNumber = 0x3bef5c

//消息的编解码方式
type Option struct {
	MagicNumber int        // 标志着这一次rpc请求
	CodecType   codec.Type // 客户端可以选择的不同编码类型
}

var DefaultOption = &Option{
	MagicNumber: MagicNumber,
	CodecType:   codec.GobType,
}

// 定义传输的编码方式
// Option{MagicNumber: xxx, CodecType: xxx} | Header{ServiceMethod ...} | Body interface{}|
// 客户端固定采用JSON编码Option，之后的Header和Body的编码方式由Option的CodecType指定
// 服务端先使用JSON编码Option，然后通过Option的CodecType解码剩余的内容
// |Option | Header1 | Body1 | Header2 | Body2 |...

type Server struct{}

func NewServer() *Server {
	return &Server{}
}

var DefaultServer = NewServer() //设置一个默认的Server实例

//在server上接收连接
func (server *Server) Accept(lis net.Listener) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Println("rpc server: accept error:", err)
			return
		}
		go server.ServeConn(conn) //开启子协程处理
	}
}

//先使用json.NewDecoder 反序列化得到的Option实例，检查MagicNumber和CodecType的值是否正确
//然后根据CodecType得到对应的消息编解码器
func (server *Server) ServeConn(conn io.ReadWriteCloser) {
	defer func() {
		_ = conn.Close()
	}()

	var opt Option
	if err := json.NewDecoder(conn).Decode(&opt); err != nil {
		log.Println("rpc server: options error: ", err)
		return
	}
	if opt.MagicNumber != MagicNumber {
		log.Printf("rpc server: invalid magin number %x", opt.MagicNumber)
		return
	}
	f := codec.NewCodecFuncMap[opt.CodecType]

	if f == nil {
		log.Printf("rpc server: invalid codec type %s", opt.CodecType)
		return
	}
	//得到对应的消息编解码器，接下来的处理交给serverCodec
	server.serverCodec(f(conn))

}

var invalidRequest = struct{}{}

func (server *Server) serverCodec(cc codec.Codec) {
	sending := new(sync.Mutex) //确保发送完整的响应
	wg := new(sync.WaitGroup)  //等待，直到所有的请求都被处理
	for {
		// 读取请求 打印出header信息
		req, err := server.readRequest(cc)
		if err != nil {
			if req == nil {
				break
			}
			req.h.Error = err.Error()
			// 回复请求， 暂时回复geerpc resp ${req.h.Seq}
			server.sendResponse(cc, req.h, invalidRequest, sending)
			continue
		}
		wg.Add(1)
		//处理请求
		go server.handleRequest(cc, req, sending, wg)
	}
	wg.Wait()
	_ = cc.Close
}

type request struct {
	h            *codec.Header //请求头
	argv, replyv reflect.Value // 请求argv和replyv
}

//读取请求
func (server *Server) readRequestHeader(cc codec.Codec) (*codec.Header, error) {
	var h codec.Header
	if err := cc.ReadHeader(&h); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			log.Println("rpc server: read head error:", err)
		}
		return nil, err
	}
	return &h, nil
}

func (server *Server) readRequest(cc codec.Codec) (*request, error) {
	h, err := server.readRequestHeader(cc)
	if err != nil {
		return nil, err
	}
	req := &request{h: h}
	// TODO: 现在还不知道request argv的类型
	// 只返回string，先将body作为字符串处理
	req.argv = reflect.New(reflect.TypeOf(""))
	//
	if err = cc.ReadBody(req.argv.Interface()); err != nil {
		log.Println("rpc server: read argv err:", err)
	}
	return req, nil
}

func (server *Server) sendResponse(cc codec.Codec, h *codec.Header,
	body interface{}, sending *sync.Mutex) {
	sending.Lock()
	defer sending.Unlock()
	if err := cc.Write(h, body); err != nil {
		log.Println("rpc server: write response error:", err)
	}
}

func (server *Server) handleRequest(cc codec.Codec, req *request,
	sending *sync.Mutex, wg *sync.WaitGroup) {
	//TODO 应该调用注册rpc方法去得到正确的replyv
	// 先只是打印出argv和发送hello信息
	defer wg.Done()
	log.Println(req.h, req.argv.Elem())
	req.replyv = reflect.ValueOf(fmt.Sprintf("geerpc resp %d", req.h.Seq))
	server.sendResponse(cc, req.h, req.replyv.Interface(), sending)

}

func Accept(lis net.Listener) {
	DefaultServer.Accept(lis)
}
