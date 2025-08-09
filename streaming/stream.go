package streaming

import (
	"io"
	"sync"

	"github.com/tributary-ai/llm-invocation/types"
)

type responseStream struct {
	ch     <-chan *types.StreamChunk
	closed bool
	mu     sync.Mutex
}

func NewResponseStream(ch <-chan *types.StreamChunk) *responseStream {
	return &responseStream{
		ch: ch,
	}
}

func (s *responseStream) Next() (*types.StreamChunk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.closed {
		return nil, io.EOF
	}
	
	chunk, ok := <-s.ch
	if !ok {
		s.closed = true
		return nil, io.EOF
	}
	
	if chunk.Error != nil {
		return chunk, chunk.Error
	}
	
	if chunk.Done {
		s.closed = true
	}
	
	return chunk, nil
}

func (s *responseStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.closed {
		s.closed = true
		// Drain the channel to prevent goroutine leaks
		go func() {
			for range s.ch {
				// Drain remaining chunks
			}
		}()
	}
	
	return nil
}

func (s *responseStream) Aggregate() (*types.InvocationResponse, error) {
	aggregator := NewStreamAggregator()
	
	for {
		chunk, err := s.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		
		aggregator.Add(*chunk)
	}
	
	return aggregator.Aggregate()
}