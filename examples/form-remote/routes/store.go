package routes

import (
	"sync"
	"time"
)

// Message represents a submitted message in the guestbook
type Message struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// PaginatedMessages holds messages with pagination info
type PaginatedMessages struct {
	Messages   []Message `json:"messages"`
	Total      int       `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"pageSize"`
	TotalPages int       `json:"totalPages"`
}

// MessageStore is a thread-safe in-memory store for messages
type MessageStore struct {
	mu       sync.RWMutex
	messages []Message
	nextID   int
}

var store = &MessageStore{
	messages: []Message{},
	nextID:   1,
}

// AddMessage adds a new message to the store
func (s *MessageStore) AddMessage(name, content string) Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := Message{
		ID:        s.nextID,
		Name:      name,
		Content:   content,
		Timestamp: time.Now(),
	}
	s.nextID++
	s.messages = append([]Message{msg}, s.messages...)
	return msg
}

// GetMessages returns messages with pagination
func (s *MessageStore) GetMessages(page, pageSize int) PaginatedMessages {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.messages)
	if total == 0 {
		return PaginatedMessages{
			Messages:   []Message{},
			Total:      0,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: 0,
		}
	}

	totalPages := (total + pageSize - 1) / pageSize
	if page < 1 {
		page = 1
	}
	if page > totalPages && totalPages > 0 {
		page = totalPages
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= total {
		start = 0
		end = 0
	}
	if end > total {
		end = total
	}

	return PaginatedMessages{
		Messages:   s.messages[start:end],
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// GetAllMessages returns all messages (for initial state)
func (s *MessageStore) GetAllMessages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// GetStore returns the global message store instance
func GetStore() *MessageStore {
	return store
}
