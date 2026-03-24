package state

import (
	"testing"
	"time"
)

func TestEnqueueStateNotificationDropsWhenQueueIsFull(t *testing.T) {
	startStateNotificationDispatcher()
	startDropped := droppedNotifications.Load()

	originalQueue := stateNotificationQueue
	stateNotificationQueue = make(chan stateNotification)
	defer func() {
		stateNotificationQueue = originalQueue
	}()

	enqueueStateNotification(stateNotification{handler: func(string, any) {}, key: "k", value: 1})

	if got := droppedNotifications.Load(); got != startDropped+1 {
		t.Fatalf("expected dropped notification count to increase, got start=%d current=%d", startDropped, got)
	}
}

func TestSafelyRunStateNotificationRecoversPanics(t *testing.T) {
	done := make(chan struct{}, 1)
	safelyRunStateNotification(stateNotification{
		handler: func(string, any) {
			defer func() { done <- struct{}{} }()
			panic("boom")
		},
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected panicing handler to return control")
	}
}
