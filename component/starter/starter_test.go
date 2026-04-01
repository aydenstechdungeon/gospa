package starter

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAlert(t *testing.T) {
	// Testing Alert rendering
	// Assuming Alert function exists in alert.go
	component := Alert(AlertProps{
		Title:   "Test Alert",
		Message: "This is a message",
		Variant: AlertVariantInfo,
	})

	w := httptest.NewRecorder()
	err := component.Render(context.Background(), w)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	out := w.Body.String()
	if !strings.Contains(out, "Test Alert") {
		t.Errorf("missing title")
	}
	if !strings.Contains(out, "This is a message") {
		t.Errorf("missing message")
	}
}

func TestButton(t *testing.T) {
	// Button component takes children, but the props don't have Text field
	component := Button(ButtonProps{
		Variant: ButtonPrimary,
	})

	w := httptest.NewRecorder()
	err := component.Render(context.Background(), w)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	out := w.Body.String()
	if !strings.Contains(out, "button") {
		t.Errorf("missing button tag")
	}
}

func TestCard(t *testing.T) {
	component := Card(CardProps{
		Title: "Card Title",
	})

	w := httptest.NewRecorder()
	err := component.Render(context.Background(), w)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	out := w.Body.String()
	if !strings.Contains(out, "Card Title") {
		t.Errorf("missing card title")
	}
}

func TestInput(t *testing.T) {
	component := Input(InputProps{
		Label:       "Username",
		Placeholder: "Enter username",
		Value:       "testuser",
	})

	w := httptest.NewRecorder()
	err := component.Render(context.Background(), w)
	if err != nil {
		t.Fatalf("failed to render: %v", err)
	}

	out := w.Body.String()
	if !strings.Contains(out, "Username") {
		t.Errorf("missing label")
	}
	if !strings.Contains(out, "Enter username") {
		t.Errorf("missing placeholder")
	}
	if !strings.Contains(out, "testuser") {
		t.Errorf("missing value")
	}
}
