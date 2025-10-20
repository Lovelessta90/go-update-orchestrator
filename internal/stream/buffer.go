package stream

import (
	"sync"
)

// BufferPool manages a pool of reusable byte buffers to reduce allocations.
type BufferPool struct {
	pool       sync.Pool
	bufferSize int
}

// NewBufferPool creates a new buffer pool with the specified buffer size.
func NewBufferPool(bufferSize int) *BufferPool {
	return &BufferPool{
		bufferSize: bufferSize,
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, bufferSize)
				return &buf
			},
		},
	}
}

// Get retrieves a buffer from the pool.
func (p *BufferPool) Get() *[]byte {
	return p.pool.Get().(*[]byte)
}

// Put returns a buffer to the pool for reuse.
func (p *BufferPool) Put(buf *[]byte) {
	// Reset buffer length but keep capacity
	*buf = (*buf)[:p.bufferSize]
	p.pool.Put(buf)
}
