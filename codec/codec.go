package codec

import "io"

type Header struct {
	ServiceMethod string // Service.Method 服务名和方法名，通常与Go中的结构体和方法映射
	Seq           uint64 //请求的序号，可以认为是某个请求的ID，用来区分不同的请求
	Error         string //错误信息，客户端置为空，服务端如果发生错误，将错误信息置于Error中
}

type Type string

//实现不同的codec实例
type Codec interface {
	io.Closer
	ReadHeader(*Header) error
	ReadBody(interface{}) error
	Write(*Header, interface{}) error
}

//返回构造函数
type NewCodecFunc func(io.ReadWriteCloser) Codec

const (
	GobType  Type = "application/gob"
	JsonType Type = "application/json"
)

var NewCodecFuncMap map[Type]NewCodecFunc

func init() {
	NewCodecFuncMap = make(map[Type]NewCodecFunc)
	NewCodecFuncMap[GobType] = NewGobCodec
}
