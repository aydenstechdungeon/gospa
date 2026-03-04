# Troubleshooting HMR & Development Server

## "[HMR] Disconnected" or Connection Issues

### Problem
Browser console shows:

```
[HMR] Disconnected
[HMR] Reconnecting... (1/10)
Max reconnect attempts reached
```

### Causes & Solutions

#### 1. Dev Server Not Running

Ensure you're running in development mode:

```bash
go run main.go  # Dev mode is default
gospa dev       # Using CLI
```

#### 2. Wrong HMR Port/Path

Check your HMR configuration:

```go
app := gospa.New(gospa.Config{
    DevMode: true,
    HMR: gospa.HMRConfig{
        Enabled: true,
        WSPath:  "/_gospa/hmr/ws",
    },
})
```

Client automatically detects the path, but verify in DevTools Network tab.

#### 3. HTTPS/WSS Mismatch

For HTTPS development sites, ensure WSS is used:

```javascript
// Automatically handled by GoSPA, but verify:
const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
const hmrUrl = `${protocol}//${window.location.host}/_gospa/hmr/ws`;
```

---

## Full Page Reload Instead of Hot Update

### Problem
Changes trigger a full browser reload instead of seamless hot update.

### Causes & Solutions

| Symptom | Cause | Fix |
|---------|-------|-----|
| `.go` or `.templ` file changed | Requires server rebuild | Expected - Go templates need re-render |
| State lost on reload | No state preservation hook | Implement `window.__gospaPreserveState` |
| CSS changes reload page | File in `IgnorePaths` | Check HMR config |

#### Implementing State Preservation

```javascript
// In your app initialization
window.__gospaPreserveState = () => {
    // Return serializable state
    return {
        formData: document.getElementById('myForm').value,
        scrollPosition: window.scrollY,
    };
};

window.__gospaRestoreState = (state) => {
    // Restore after reload
    if (state.formData) {
        document.getElementById('myForm').value = state.formData;
    }
    if (state.scrollPosition) {
        window.scrollTo(0, state.scrollPosition);
    }
};
```

---

## CSS/Style Changes Not Applying

### Problem
Modifying CSS files doesn't trigger updates, or styles don't refresh.

### Solutions

#### 1. Check CSS is Being Watched

```go
app := gospa.New(gospa.Config{
    DevMode: true,
    HMR: gospa.HMRConfig{
        Enabled:     true,
        WatchPaths:  []string{"./static/css", "./styles"},
        IgnorePaths: []string{"./node_modules"},
    },
})
```

#### 2. Verify CSS Link Tag

Ensure CSS is linked directly (not via JS injection):

```html
<!-- Good - HMR can detect this -->
<link rel="stylesheet" href="/static/css/app.css">

<!-- Bad - HMR may miss this -->
<script>
    document.head.appendChild(Object.assign(
        document.createElement('link'),
        { rel: 'stylesheet', href: '/static/css/app.css' }
    ));
</script>
```

#### 3. Hard Reload if Needed

Sometimes cached styles interfere. Force clear:

```bash
# Clear browser cache
Ctrl+Shift+R  # Chrome/Windows
Cmd+Shift+R   # Chrome/Mac
```

---

## HMR Works for Some Files but Not Others

### Problem
Some file types update, others don't.

### Solutions

#### 1. Check File Extensions

GoSPA HMR watches these by default:
- `.go` - Server files (triggers rebuild)
- `.templ` - Template files (triggers rebuild)
- `.css` - Styles (hot update)
- `.js`, `.ts` - JavaScript (hot update)

Add custom extensions:

```go
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        Extensions: []string{".go", ".templ", ".css", ".scss", ".js", ".ts"},
    },
})
```

#### 2. Verify File is in Watched Directory

```go
// Add additional watch paths
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        WatchPaths: []string{
            "./routes",
            "./components",
            "./static",
            "./assets",  // Add custom paths
        },
    },
})
```

---

## Dev Server Port Already in Use

### Problem

```
listen tcp :3000: bind: address already in use
```

### Solutions

#### 1. Change Port in Config

```go
app := gospa.New(gospa.Config{
    Port: 3001,  // Use different port
})
```

#### 2. Kill Existing Process

```bash
# Find and kill process on port 3000
lsof -ti:3000 | xargs kill -9  # macOS/Linux
netstat -ano | findstr :3000   # Windows (then taskkill /PID <id>)
```

---

## Slow Dev Server Startup

### Problem
Development server takes too long to start.

### Solutions

#### 1. Disable Unused Features

```go
app := gospa.New(gospa.Config{
    DevMode:     true,
    WebSocket:   false,  // Disable if not needed
    EnableSSE:   false,  // Disable if not needed
    EnableCSRF:  false,  // Disable in dev
    CompressState: false, // Disable compression
})
```

#### 2. Reduce Watch Scope

```go
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        WatchPaths: []string{"./routes"},  // Narrow scope
        IgnorePaths: []string{
            "./node_modules",
            "./.git",
            "./tmp",
            "./vendor",
        },
    },
})
```

---

## "Too Many Open Files" Error

### Problem

```
fatal error: pipe failed: too many open files
```

### Solution
Increase file descriptor limit:

```bash
# macOS
ulimit -n 10240

# Linux (temporary)
ulimit -n 65536

# Linux (permanent) - add to /etc/security/limits.conf
* soft nofile 65536
* hard nofile 65536
```

---

## Template Changes Not Reflecting

### Problem
Modifying `.templ` files doesn't show changes.

### Solutions

#### 1. Ensure Templ is Installed

```bash
go install github.com/a-h/templ/cmd/templ@latest
```

#### 2. Regenerate Templates

```bash
templ generate  # Or use gospa CLI which handles this
gospa generate
```

#### 3. Check Generated Files Are Being Watched

```go
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        WatchPaths: []string{
            "./routes",
            "./_templ",  // Generated templates
        },
    },
})
```

---

## DevTools Shows No HMR WebSocket

### Problem
Can't find HMR WebSocket connection in DevTools.

### Debugging Steps

1. **Check Console for `[HMR]` logs**
   - Should see `[HMR] Connected` on successful connection

2. **Network Tab > WS Filter**
   - Look for WebSocket to `/_gospa/hmr/ws`
   - Check messages being sent/received

3. **Verify DevMode is Enabled**

```javascript
// In browser console
console.log(window.__GOSPA_HMR__);  // Should show HMR state
```

4. **Check for Errors**

```javascript
// Enable debug logging
window.__GOSPA_DEBUG__ = true;
```

---

## Environment Variables Not Reloading

### Problem
Changes to `.env` files require full server restart.

### Solution

This is expected behavior. GoSPA doesn't hot-reload environment variables. Use a restart:

```bash
# Use air for auto-restart on .env changes
air --build.cmd "go run main.go" --build.include_ext "go,templ,env"
```

Or implement config hot-reload:

```go
// Custom config watcher
func watchConfig(app *gospa.Application) {
    watcher, _ := fsnotify.NewWatcher()
    watcher.Add(".env")
    
    go func() {
        for event := range watcher.Events {
            if event.Op&fsnotify.Write == fsnotify.Write {
                // Reload config
                loadConfig()
            }
        }
    }()
}
```

---

## Common Mistakes

### Mistake 1: Production Build in Dev Mode

```go
// BAD - Forces dev mode even in production
app := gospa.New(gospa.Config{
    DevMode: true,  // Never hardcode in committed code
})

// GOOD - Use environment variable
app := gospa.New(gospa.Config{
    DevMode: os.Getenv("ENV") == "development",
})
```

### Mistake 2: Watching Node Modules

```go
// BAD - Watches too much
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        WatchPaths: []string{"."},  // Watches everything!
    },
})

// GOOD - Be specific
app := gospa.New(gospa.Config{
    HMR: gospa.HMRConfig{
        WatchPaths:  []string{"./routes", "./static"},
        IgnorePaths: []string{"./node_modules", "./.git"},
    },
})
```

### Mistake 3: Not Handling HMR Errors

```javascript
// BAD - Silent failures
import { initHMR } from '@gospa/runtime';
initHMR();

// GOOD - Handle errors
import { initHMR } from '@gospa/runtime';
initHMR({
    onError: (error) => {
        console.error('[HMR] Error:', error);
        // Optionally show user notification
    },
    onConnect: () => console.log('[HMR] Ready'),
    onDisconnect: () => console.warn('[HMR] Lost connection'),
});
```
