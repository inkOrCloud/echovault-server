package sync

import (
	"sync"

	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

const notifierBufferSize = 16

// Notifier sends change notifications to subscribed devices.
type Notifier struct {
	mu          sync.RWMutex
	subscribers map[string]chan *syncpb.ChangeNotification
}

// NewNotifier creates a new Notifier.
func NewNotifier() *Notifier {
	return &Notifier{
		subscribers: make(map[string]chan *syncpb.ChangeNotification),
	}
}

// Subscribe returns a notification channel for a device.
func (n *Notifier) Subscribe(deviceID string) <-chan *syncpb.ChangeNotification {
	n.mu.Lock()
	defer n.mu.Unlock()
	if ch, ok := n.subscribers[deviceID]; ok {
		return ch
	}
	ch := make(chan *syncpb.ChangeNotification, notifierBufferSize)
	n.subscribers[deviceID] = ch
	return ch
}

// Unsubscribe removes a device from notifications.
func (n *Notifier) Unsubscribe(deviceID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	delete(n.subscribers, deviceID)
}

// Notify sends a notification to all subscribers.
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
