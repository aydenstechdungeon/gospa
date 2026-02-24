package tailwind

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/aydenstechdungeon/gospa/plugin"
)

type TailwindPlugin struct {
	mu      sync.Mutex
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	stopped bool
}

func New() *TailwindPlugin {
	return &TailwindPlugin{}
}

func (p *TailwindPlugin) Name() string {
	return "tailwind"
}

func (p *TailwindPlugin) Init() error {
	return nil
}

func (p *TailwindPlugin) Dependencies() []plugin.Dependency {
	return []plugin.Dependency{
		{Type: plugin.DepBun, Name: "tailwindcss", Version: "latest"},
		{Type: plugin.DepBun, Name: "@tailwindcss/postcss", Version: "latest"},
		{Type: plugin.DepBun, Name: "postcss", Version: "latest"},
	}
}

func (p *TailwindPlugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
	switch hook {
	case plugin.BeforeDev:
		if p.isInstalled() {
			go p.watchWithContext()
		}
	case plugin.AfterDev:
		// Graceful shutdown when dev server stops
		p.Stop()
	case plugin.BeforeBuild:
		if p.isInstalled() {
			return p.compile(true)
		}
	}
	return nil
}

// Stop gracefully stops the Tailwind watcher.
func (p *TailwindPlugin) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return
	}
	p.stopped = true

	if p.cancel != nil {
		p.cancel()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
	}
	fmt.Println("Tailwind: watcher stopped")
}

func (p *TailwindPlugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Name:        "add:tailwind",
			Description: "Install and configure Tailwind CSS v4",
			Action:      p.install,
		},
	}
}

func (p *TailwindPlugin) isInstalled() bool {
	_, err := os.Stat("static/css/app.css")
	return err == nil
}

func (p *TailwindPlugin) install(args []string) error {
	fmt.Println("Installing Tailwind CSS v4...")

	// 1. Install dependencies with bun
	fmt.Println("Running: bun add -d tailwindcss @tailwindcss/postcss postcss")
	cmd := exec.Command("bun", "add", "-d", "tailwindcss", "@tailwindcss/postcss", "postcss")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install tailwind: %w", err)
	}

	// 2. Create static/css directory
	if err := os.MkdirAll("static/css", 0755); err != nil {
		return err
	}

	// 3. Create app.css with v4 syntax
	appCSS := "@import \"tailwindcss\";\n"
	if err := os.WriteFile("static/css/app.css", []byte(appCSS), 0644); err != nil {
		return err
	}

	// 4. Create postcss.config.js (optional but good for v4)
	postcssConfig := `export default {
  plugins: {
    "@tailwindcss/postcss": {},
  }
}
`
	if err := os.WriteFile("postcss.config.js", []byte(postcssConfig), 0644); err != nil {
		return err
	}

	fmt.Println("✓ Tailwind CSS v4 installed!")
	fmt.Println("Added static/css/app.css and configured PostCSS.")
	return nil
}

func (p *TailwindPlugin) watchWithContext() {
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	p.mu.Unlock()
	
	fmt.Println("Tailwind: starting watcher...")
	
	cmd := exec.CommandContext(ctx, "bunx", "@tailwindcss/cli", "-i", "./static/css/app.css", "-o", "./static/dist/app.css", "--watch")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	p.mu.Lock()
	p.cmd = cmd
	p.mu.Unlock()
	
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Println("Tailwind: watcher stopped gracefully")
		} else {
			fmt.Fprintf(os.Stderr, "Tailwind watcher failed: %v\n", err)
		}
	}
}

func (p *TailwindPlugin) compile(minify bool) error {
	fmt.Println("Tailwind: compiling for production...")
	args := []string{"@tailwindcss/cli", "-i", "./static/css/app.css", "-o", "./static/dist/app.css"}
	if minify {
		args = append(args, "--minify")
	}
	cmd := exec.Command("bunx", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
