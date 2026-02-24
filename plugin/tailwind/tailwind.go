package tailwind

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/aydenstechdungeon/gospa/plugin"
)

type TailwindPlugin struct{}

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
			go p.watch()
		}
	case plugin.BeforeBuild:
		if p.isInstalled() {
			return p.compile(true)
		}
	}
	return nil
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

func (p *TailwindPlugin) watch() {
	fmt.Println("Tailwind: starting watcher...")
	// Use the tailwind CLI to watch
	// We assume its outputting to static/dist/app.css or similar
	cmd := exec.Command("bunx", "@tailwindcss/cli", "-i", "./static/css/app.css", "-o", "./static/dist/app.css", "--watch")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Tailwind watcher failed: %v\n", err)
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
