package auth

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

type limitEntry struct {
	count  int
	resets time.Time
}

type limiter struct {
	mu      sync.Mutex
	entries map[string]limitEntry
}

func newLimiter() *limiter { return &limiter{entries: make(map[string]limitEntry)} }

func (l *limiter) allow(key string, maximum int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	entry, ok := l.entries[key]
	//record request if the reset time is AFTER now
	if !ok || now.After(entry.resets) {
		l.entries[key] = limitEntry{count: 1, resets: now.Add(window)}
		return true
	}

	//if attempts exhausted, block request
	if entry.count >= maximum {
		return false
	}

	//update attempt count
	entry.count++
	l.entries[key] = entry

	return true
}

func requestIP(r *http.Request) string {
	value := r.RemoteAddr
	if index := strings.LastIndex(value, ":"); index > 0 {
		return value[:index]
	}
	return value
}
