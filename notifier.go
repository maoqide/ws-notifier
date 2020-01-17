package notifier

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/maoqide/melody"

	"github.com/maoqide/ws-notifier/sessionmanager"
)

var notifier = New()

// NotifyMessage is common struct for notifier
type NotifyMessage struct {
	Type    string      `json:"type"`
	Code    int32       `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type workerFunc func(string, chan int8, *Notifier) error

// Notifier wrapped websocket operation for notifier
type Notifier struct {
	SessionManager *sessionmanager.SessionManager
	Melody         *melody.Melody
	workers        map[string]chan int8
	lock           *sync.RWMutex
}

// Default return default initialized notifier, recommended.
func Default() *Notifier {
	return notifier
}

// New creates a Notifier
func New() *Notifier {
	m := melody.New()
	upgrader := websocket.Upgrader{}
	upgrader.HandshakeTimeout = time.Second * 2
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}
	m.Upgrader = &upgrader

	sm := sessionmanager.New()
	m.HandleConnect(func(s *melody.Session) {
		g, exists := s.Get("group")
		if !exists {
			return
		}
		sm.Join(g.(string), s)
		if v, ok := s.Get("message"); ok {
			s.Write([]byte(v.(string)))
			s.Del("message")
		}
	})

	m.HandleMessage(func(s *melody.Session, msg []byte) {
	})
	m.HandlePong(func(s *melody.Session) {
	})
	m.HandleDisconnect(func(s *melody.Session) {
		g, exists := s.Get("group")
		if !exists {
			s.Close()
		}
		sm.Release(g.(string), s)
	})
	// m.Config.PongWait = 600 * time.Second
	// m.Config.PingPeriod = 601 * time.Second
	return &Notifier{
		Melody:         m,
		SessionManager: sm,
		workers:        make(map[string]chan int8),
		lock:           new(sync.RWMutex),
	}
}

// Notify start notify worker process
func (n *Notifier) Notify(groupID string, f workerFunc, timeout time.Duration) error {
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.workers[groupID]; ok {
		return nil
	}
	n.workers[groupID] = make(chan int8)
	go f(groupID, n.workers[groupID], n)
	go func() {
		timer := time.NewTimer(timeout)
		for {
			select {
			case <-n.workers[groupID]:
				n.ReleaseWorker(groupID)
				// close all sessions of the group if worker exited, reconnection is needed from frontend.
				n.CloseGroupWithMsg(groupID, []byte{})
				return
			case <-timer.C:
				n.workers[groupID] <- 1
			// kill worker goroutine when all session closed.
			case <-time.Tick(time.Second):
				if n.GroupLen(groupID) == 0 {
					n.workers[groupID] <- 2
				}
			}
		}
	}()
	return nil
}

// ReleaseWorker release worker for a group, usually called from a workerFunc when goroutine exited
func (n *Notifier) ReleaseWorker(groupID string) {
	n.lock.Lock()
	defer n.lock.Unlock()
	if _, ok := n.workers[groupID]; !ok {
		return
	}
	close(n.workers[groupID])
	delete(n.workers, groupID)
	return
}

// GroupBroadcast broadcast message to a group
func (n *Notifier) GroupBroadcast(msg []byte, groupID string) error {
	if n.GroupLen(groupID) == 0 {
		return errors.New("no active session")
	}

	return n.Melody.BroadcastFilter(msg, func(s *melody.Session) bool {
		group, exists := s.Get("group")
		return exists && (group.(string) == groupID)
	})
}

// Broadcast broadcast message to all
func (n *Notifier) Broadcast(msg []byte) error {
	return n.Melody.Broadcast(msg)
}

// Close close all websocket connections
func (n *Notifier) Close() error {
	n.SessionManager = nil
	for _, c := range n.workers {
		close(c)
	}
	return n.Melody.Close()
}

// CloseWithMsg close all websocket connections with messages.
// Use the FormatCloseMessage function to format a proper close message payload.
func (n *Notifier) CloseWithMsg(msg []byte) error {
	n.SessionManager = nil
	for _, c := range n.workers {
		close(c)
	}
	return n.Melody.CloseWithMsg(msg)
}

// CloseGroupWithMsg close all websocket connections of a group with messages.
// Use the FormatCloseMessage function to format a proper close message payload.
func (n *Notifier) CloseGroupWithMsg(groupID string, msg []byte) error {
	sessions := n.SessionManager.GetSessions(groupID)
	for _, s := range sessions {
		s.CloseWithMsg(msg)
	}
	return nil
}

// IsClosed return status of websocket
func (n *Notifier) IsClosed() bool {
	return n.Melody.IsClosed()
}

// Len return the number of connected sessions.
func (n *Notifier) Len() int {
	return n.Melody.Len()
}

// GroupLen return the number of connected sessions of a group.
func (n *Notifier) GroupLen(groupID string) int {
	return len(n.SessionManager.GetSessions(groupID))
}

// HandleRequest upgrades http requests to websocket connections and dispatches them to be handled by the melody instance.
func (n *Notifier) HandleRequest(w http.ResponseWriter, r *http.Request) error {
	return n.Melody.HandleRequest(w, r)
}

// HandleRequestWithKeys does the same as HandleRequest but populates session.Keys with keys.
func (n *Notifier) HandleRequestWithKeys(w http.ResponseWriter, r *http.Request, keys map[string]interface{}) error {
	return n.Melody.HandleRequestWithKeys(w, r, keys)
}

// FormatCloseMessage formats closeCode and text as a WebSocket close message.
func FormatCloseMessage(closeCode int, text string) []byte {
	return websocket.FormatCloseMessage(closeCode, text)
}

// ShowWorkers shows all workers
func (n *Notifier) ShowWorkers() []string {
	n.lock.RLock()
	defer n.lock.RUnlock()
	res := make([]string, 0)
	for w := range n.workers {
		res = append(res, w)
	}
	return res
}
