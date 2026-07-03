package sync_test

import (
	"testing"
	"time"

	syncsvc "github.com/inkOrCloud/EchoVault/echovault-server/internal/service/sync"
	syncpb "github.com/inkOrCloud/EchoVault/echovault-server/api/grpc/generated/echo_vault/sync/v1"
)

func TestSubscribeAndNotify(t *testing.T) {
	t.Parallel()
	n := syncsvc.NewNotifier()
	ch := n.Subscribe("dev-001")
	defer n.Unsubscribe("dev-001")

	go func() {
		n.Notify(&syncpb.ChangeNotification{
			EntityType: "song",
			Action:     "created",
			NewVersion: 1,
		})
	}()

	select {
	case notif := <-ch:
		if notif.GetEntityType() != "song" {
			t.Errorf("EntityType = %q, want %q", notif.GetEntityType(), "song")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for notification")
	}
}

func TestSubscribe_MultipleDevices(t *testing.T) {
	t.Parallel()
	n := syncsvc.NewNotifier()
	ch1 := n.Subscribe("dev-001")
	ch2 := n.Subscribe("dev-002")
	defer n.Unsubscribe("dev-001")
	defer n.Unsubscribe("dev-002")

	n.Notify(&syncpb.ChangeNotification{NewVersion: 1})

	<-ch1
	<-ch2
}

func TestUnsubscribe_StopsReceiving(t *testing.T) {
	t.Parallel()
	n := syncsvc.NewNotifier()
	ch := n.Subscribe("dev-001")
	n.Unsubscribe("dev-001")

	n.Notify(&syncpb.ChangeNotification{NewVersion: 1})

	select {
	case <-ch:
		t.Fatal("received notification after unsubscribe")
	case <-time.After(100 * time.Millisecond):
	}
}
