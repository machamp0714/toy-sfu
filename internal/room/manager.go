package room

import "sync"

// Manager tracks all active rooms, creating them lazily and removing them
// once the last participant leaves.
type Manager struct {
	mu    sync.Mutex
	rooms map[string]*Room
}

// NewManager returns an empty Manager.
func NewManager() *Manager {
	return &Manager{rooms: make(map[string]*Room)}
}

// GetOrCreate returns the Room for name, creating it if it doesn't exist yet.
func (m *Manager) GetOrCreate(name string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	r, ok := m.rooms[name]
	if !ok {
		r = newRoom()
		m.rooms[name] = r
	}
	return r
}

// Leave removes participantID from the named room. If the room becomes
// empty as a result, the room itself is removed from the Manager.
func (m *Manager) Leave(name, participantID string) {
	m.mu.Lock()
	r, ok := m.rooms[name]
	m.mu.Unlock()
	if !ok {
		return
	}

	if r.Leave(participantID) {
		m.mu.Lock()
		if cur, ok := m.rooms[name]; ok && cur == r && r.Count() == 0 {
			delete(m.rooms, name)
		}
		m.mu.Unlock()
	}
}
