package state

import (
	"testing"
	"time"
)

func TestEnqueueStateNotificationFallsBackToSyncWhenQueueIsFull(t *testing.T) {
	startStateNotificationDispatcher()
	startDropped := droppedNotifications.Load()
	done := make(chan struct{}, 1)

	originalQueue := stateNotificationQueue
	stateNotificationQueue = make(chan stateNotification)
	defer func() {
		stateNotificationQueue = originalQueue
	}()

	enqueueStateNotification(stateNotification{
		handler: func(string, any) { done <- struct{}{} },
		key:     "k",
		value:   1,
	})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected synchronous fallback to invoke handler")
	}

	if got := droppedNotifications.Load(); got != startDropped {
		t.Fatalf("expected no dropped notification increment, got start=%d current=%d", startDropped, got)
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
