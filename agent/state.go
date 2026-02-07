package agent

import (
	"sync"
	"time"
)

// AgentState represents the current state of the agent
type AgentState int

const (
	StateIdle     AgentState = iota
	StateRunning
	StateError
	StateCompleted
)

// String returns the string representation of AgentState
func (s AgentState) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateRunning:
		return "running"
	case StateError:
		return "error"
	case StateCompleted:
		return "completed"
	default:
		return "unknown"
	}
}

// StateManager manages agent state in a thread-safe manner
type StateManager struct {
	state  AgentState
	mu     sync.RWMutex
	metrics *Metrics
}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
	return &StateManager{
		state:  StateIdle,
		metrics: &Metrics{},
	}
}

// GetState returns the current agent state (thread-safe)
func (sm *StateManager) GetState() AgentState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state
}

// setState updates the agent state (thread-safe)
func (sm *StateManager) setState(state AgentState) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state = state
}

// GetMetrics returns a copy of the current metrics (thread-safe)
func (sm *StateManager) GetMetrics() Metrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return *sm.metrics
}

// incrementToolCalls increments the tool call counters
func (sm *StateManager) incrementToolCalls(success bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.metrics.TotalToolCalls++
	if success {
		sm.metrics.SuccessfulToolCalls++
	} else {
		sm.metrics.FailedToolCalls++
	}
}

// incrementIterations increments the iteration counter
func (sm *StateManager) incrementIterations() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.metrics.TotalIterations++
}

// setStartTime sets the start time for metrics
func (sm *StateManager) setStartTime(t time.Time) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	// Metrics doesn't have start time field, could be added if needed
}

// reset resets the state and metrics for a new run
func (sm *StateManager) reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state = StateIdle
	sm.metrics = &Metrics{}
}
