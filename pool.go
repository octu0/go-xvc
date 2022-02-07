package xvc

import (
	"bytes"
	"sync"
)

type BufferPool interface {
	Get() *bytes.Buffer
	Put(*bytes.Buffer)
}

type simpleBufferPool struct {
	pool *sync.Pool
}

func (p *simpleBufferPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

func (p *simpleBufferPool) Put(b *bytes.Buffer) {
	p.pool.Put(b)
}

func newSimpleBufferPool(size int) *simpleBufferPool {
	return &simpleBufferPool{
		pool: &sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, size))
			},
		},
	}
}
