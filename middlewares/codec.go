package moddlewares

type Codec interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) ([]byte, error)
}

var dc = &defaultCodec{}

type defaultCodec struct {
}

func (c *defaultCodec) Encode(msg []byte) (data []byte, err error) {
	data = msg
	return
}

func (c *defaultCodec) Decode(msg []byte) (data []byte, err error) {
	data = msg
	return
}
