package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestColorPrinter(t *testing.T) {
	p := &ColorPrinter{useColor: true}

	if p.Bold("text") != "\033[1mtext\033[0m" {
		t.Errorf("Bold parsing failed")
	}
	if p.Green("text") != "\033[32mtext\033[0m" {
		t.Errorf("Green parsing failed")
	}
	if p.Red("text") != "\033[31mtext\033[0m" {
		t.Errorf("Red parsing failed")
	}
	if p.Yellow("text") != "\033[33mtext\033[0m" {
		t.Errorf("Yellow parsing failed")
	}
	if p.Cyan("text") != "\033[36mtext\033[0m" {
		t.Errorf("Cyan parsing failed")
	}
	if p.Dim("text") != "\033[2mtext\033[0m" {
		t.Errorf("Dim parsing failed")
	}

	pNoColor := &ColorPrinter{useColor: false}
	if pNoColor.Bold("text") != "text" {
		t.Errorf("Without color failed")
	}
}

// captureStdout captures what's printed to os.Stdout
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestColorPrinter_OutputMethods(t *testing.T) {
	p := &ColorPrinter{useColor: false} // test without colors to easily match strings

	out := captureStdout(func() {
		p.Success("worked %s", "fine")
	})
	if !strings.Contains(out, "✓ worked fine") {
		t.Errorf("Success method failed, got %q", out)
	}

	out = captureStdout(func() {
		p.Warning("watch out %d", 1)
	})
	if !strings.Contains(out, "! watch out 1") {
		t.Errorf("Warning method failed, got %q", out)
	}

	out = captureStdout(func() {
		p.Info("note %s", "this")
	})
	if !strings.Contains(out, "→ note this") {
		t.Errorf("Info method failed, got %q", out)
	}

	out = captureStdout(func() {
		p.Step(1, 4, "Doing something")
	})
	if !strings.Contains(out, "[1/4] Doing something") {
		t.Errorf("Step method failed, got %q", out)
	}
	
	out = captureStdout(func() {
		p.Title("Hello")
	})
	if !strings.Contains(out, "Hello") {
		t.Errorf("Title failed")
	}

	out = captureStdout(func() {
		p.Subtitle("World")
	})
	if !strings.Contains(out, "World") {
		t.Errorf("Subtitle failed")
	}
}

func TestSpinner(t *testing.T) {
	p := &ColorPrinter{useColor: false}
	s := NewSpinner(p, "spinning")
	
	out := captureStdout(func() {
		s.Tick()
	})
	if !strings.Contains(out, "spinning") {
		t.Errorf("Spinner tick failed")
	}

	out = captureStdout(func() {
		s.Done()
	})
	if !strings.Contains(out, "✓ spinning") {
		t.Errorf("Spinner done failed")
	}

	out = captureStdout(func() {
		s.Fail()
	})
	if !strings.Contains(out, "✗ spinning") {
		t.Errorf("Spinner fail failed")
	}
}
