package server

import (
	"stockmind/internal/agent"
	"sync"

	"github.com/google/uuid"
)

type SessionStream struct {
	events      []agent.ChatEvent
	subscribers []chan agent.ChatEvent
	mu          sync.Mutex
	isComplete  bool
}

type StreamManager struct {
	streams map[uuid.UUID]*SessionStream
	mu      sync.RWMutex
}

func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams: make(map[uuid.UUID]*SessionStream),
	}
}

func (sm *StreamManager) CreateStream(sessionID uuid.UUID) *SessionStream {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	stream := &SessionStream{
		events:      make([]agent.ChatEvent, 0),
		subscribers: make([]chan agent.ChatEvent, 0),
		isComplete:  false,
	}
	sm.streams[sessionID] = stream
	return stream
}

func (sm *StreamManager) GetStream(sessionID uuid.UUID) (*SessionStream, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	stream, exists := sm.streams[sessionID]
	return stream, exists
}

func (sm *StreamManager) RemoveStream(sessionID uuid.UUID) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.streams, sessionID)
}

func (ss *SessionStream) AddEvent(event agent.ChatEvent) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	ss.events = append(ss.events, event)
	
	for _, sub := range ss.subscribers {
		select {
		case sub <- event:
		default:
		}
	}
}

func (ss *SessionStream) Close() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	ss.isComplete = true
	// Send a final isEnd event if not already sent
	finalEvent := agent.ChatEvent{IsEnd: true}
	ss.events = append(ss.events, finalEvent)
	
	for _, sub := range ss.subscribers {
		select {
		case sub <- finalEvent:
		default:
		}
		close(sub)
	}
	ss.subscribers = nil
}

func (ss *SessionStream) Subscribe() (chan agent.ChatEvent, []agent.ChatEvent, bool) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	ch := make(chan agent.ChatEvent, 100)
	
	if !ss.isComplete {
		ss.subscribers = append(ss.subscribers, ch)
	}
	
	return ch, ss.events, ss.isComplete
}

func (ss *SessionStream) Unsubscribe(ch chan agent.ChatEvent) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	
	for i, sub := range ss.subscribers {
		if sub == ch {
			ss.subscribers = append(ss.subscribers[:i], ss.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}
