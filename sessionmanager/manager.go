package sessionmanager

import (
	"sync"

	"gopkg.in/olahol/melody.v1"
)

// SessionManager manage sessions
type SessionManager struct {
	sessionGroup map[string]map[*melody.Session]bool
	rwLock       *sync.RWMutex
}

// Join join a session into a group
func (sm *SessionManager) Join(groupID string, session *melody.Session) {
	sm.rwLock.Lock()
	defer sm.rwLock.Unlock()
	if _, ok := sm.sessionGroup[groupID]; !ok {
		sm.sessionGroup[groupID] = make(map[*melody.Session]bool)
	}
	sm.sessionGroup[groupID][session] = true
	return
}

// Release release a session from a group
func (sm *SessionManager) Release(groupID string, session *melody.Session) {
	sm.rwLock.Lock()
	defer sm.rwLock.Unlock()
	if !session.IsClosed() {
		session.Close()
	}
	delete(sm.sessionGroup[groupID], session)
	if len(sm.sessionGroup[groupID]) == 0 {
		delete(sm.sessionGroup, groupID)
	}
	return
}

// GetSessions get sessions of a group
func (sm *SessionManager) GetSessions(groupID string) []*melody.Session {
	sessions := make([]*melody.Session, 0)
	sm.rwLock.RLock()
	defer sm.rwLock.RUnlock()
	for s := range sm.sessionGroup[groupID] {
		sessions = append(sessions, s)
	}
	return sessions
}

// New create a new SessionManager
func New() *SessionManager {
	return &SessionManager{
		sessionGroup: make(map[string]map[*melody.Session]bool),
		rwLock:       new(sync.RWMutex),
	}
}

// ShowSessions shows all sessions
func (sm *SessionManager) ShowSessions() map[string][]string {
	res := make(map[string][]string)
	for group, sessions := range sm.sessionGroup {
		res[group] = make([]string, 0)
		for s := range sessions {
			id, exists := s.Get("id")
			if !exists {
				id = "unknown"
			}
			res[group] = append(res[group], id.(string))
		}
	}
	return res
}
