package state

import "sync"

type State int

const (
	Idle State = iota
	Creating
	Editing
	Searching
	ChangingTime
	CreatingPluginEntry
	CatalogSearching
)

type UserState struct {
	State       State
	EditEntryID int // used when Editing

	// used when CreatingPluginEntry
	PluginSlug string
	FieldIndex int
	Values     map[string]string
	Selected   []string // in-progress multi-choice selection

	// used by the diary catalogue
	CatalogQuery string
	CatalogPage  int
}

type Manager struct {
	mu     sync.RWMutex
	states map[int64]*UserState
}

func NewManager() *Manager {
	return &Manager{states: make(map[int64]*UserState)}
}

func (m *Manager) Get(uid int64) *UserState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.states[uid]; ok {
		return s
	}
	return &UserState{State: Idle}
}

func (m *Manager) Set(uid int64, s *UserState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[uid] = s
}

func (m *Manager) Reset(uid int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.states, uid)
}
