package cache

type ByteView struct {
	b []byte
}

func (bv ByteView) Len() int64 {
	return int64(len(bv.b))
}

func (bv ByteView) ByteSlice() []byte {
	return cloneBytes(bv.b)
}

func (bv ByteView) String() string {
	//return *(*string)(unsafe.Pointer(&bv.b))
	return string(bv.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
