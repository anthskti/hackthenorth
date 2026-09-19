package main

import "sync"

type FrameBuffer struct {
	mu   sync.RWMutex
	jpeg []byte
}

func NewFrameBuffer() *FrameBuffer {
	return &FrameBuffer{}
}

func (b *FrameBuffer) Put(jpeg []byte) {
	if len(jpeg) == 0 {
		return
	}
	cp := make([]byte, len(jpeg))
	copy(cp, jpeg)
	b.mu.Lock()
	b.jpeg = cp
	b.mu.Unlock()
}

func (b *FrameBuffer) Latest() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.jpeg) == 0 {
		return nil
	}
	cp := make([]byte, len(b.jpeg))
	copy(cp, b.jpeg)
	return cp
}

func (b *FrameBuffer) LatestOrMock() []byte {
	if got := b.Latest(); len(got) > 0 {
		return got
	}
	frame, err := mockJPEGFrame()
	if err != nil {
		return nil
	}
	b.Put(frame)
	return frame
}
