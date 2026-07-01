package sync

import (
	"sync"

	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

type Notifier struct {
	mu          sync.RWMutex
	subscribers map[string]chan *syncpb.ChangeNotification
}

func NewNotifier() *Notifier {
	return &Notifier{
		subscribers: make(map[string]chan *syncpb.ChangeNotification),
	}
}

func (n *Notifier) Subscribe(deviceID string) <-chan *syncpb.ChangeNotification {
	n.mu.Lock()
	defer n.mu.Unlock()
	if ch, ok := n.subscribers[deviceID]; ok {
		return ch
	}
	ch := make(chan *syncpb.ChangeNotification, 16)
	n.subscribers[deviceID] = ch
	return ch
}

func (n *Notifier) Unsubscribe(deviceID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.subscribers, deviceID)
}

func (n *Notifier) Notify(notification *syncpb.ChangeNotification) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, ch := range n.subscribers {
		select {
		case ch <- notification:
		default:
		}
	}
}
