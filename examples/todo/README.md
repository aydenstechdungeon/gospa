# GoSPA Todo App Example

A feature-rich todo application demonstrating GoSPA's reactive state management, real-time synchronization, and modern UI patterns.

## Features

- ✅ **Add tasks** - Create new todos with a sleek input interface
- ✅ **Toggle completion** - Mark tasks as complete/incomplete with animated checkboxes
- ✅ **Delete tasks** - Remove individual todos with hover-reveal delete buttons
- ✅ **Filter views** - Switch between All, Active, and Completed views
- ✅ **Mark all complete** - Bulk toggle all tasks at once
- ✅ **Clear completed** - Remove all completed tasks in one click
- ✅ **localStorage persistence** - Tasks persist across page refreshes
- ✅ **Responsive design** - Works on desktop and mobile
- ✅ **Smooth animations** - Strikethrough, slide-in, and checkbox animations

## Tech Stack

- **Backend**: Go + GoSPA framework
- **Templating**: [templ](https://templ.guide/)
- **Styling**: Tailwind CSS v4 (via CDN)
- **State Management**: GoSPA client-side reactive primitives (`Rune`, `Effect`, `Derived`)
- **Fonts**: Space Grotesk (headers), IBM Plex Sans (body)

## Project Structure

```
examples/todo/
├── main.go              # Application entry point
├── go.mod              # Go module dependencies
├── go.sum              # Dependency checksums
├── routes/
│   ├── layout.templ    # Base HTML layout with styles
│   ├── page.templ      # Main todo app page with reactive JavaScript
│   └── page_templ.go   # Compiled templ Go code (generated)
└── README.md           # This file
```

## Running Locally

### Prerequisites

- Go 1.24 or later
- Templ CLI (for regenerating `.templ` files)

### Setup

1. Navigate to the todo directory:
   ```bash
   cd examples/todo
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Generate templ files (if needed):
   ```bash
   templ generate
   ```

4. Run the application:
   ```bash
   go run main.go
   ```

5. Open your browser to `http://localhost:3000`

## How It Works

### Reactive State

The app uses GoSPA's client-side reactive state management:

```javascript
// Initial state declaration in the template
data-gospa-state='{"todos":[],"filter":"all","inputValue":""}'

// Access state via __GOSPA__ global
const state = __GOSPA__.getState('todo-app');

// Subscribe to changes
state.subscribe('todos', updateUI);

// Update state
state.set('todos', newTodos);
```

### State Persistence

An `Effect`-like pattern subscribes to todo changes and persists to `localStorage`:

```javascript
state.subscribe('todos', function(todos) {
    localStorage.setItem('gospa-todos', JSON.stringify(todos));
});
```

On page load, saved todos are restored:

```javascript
const saved = localStorage.getItem('gospa-todos');
if (saved) {
    state.set('todos', JSON.parse(saved));
}
```

### Derived Values

The UI calculates derived values from state without additional storage:

- **filteredTodos** - Filtered based on current filter selection
- **activeCount** - Count of incomplete todos
- **completedCount** - Count of completed todos

### Animations

Custom CSS animations provide visual feedback:

- `slideIn` - New todo items animate in
- `strike` - Strikethrough animation on completion
- `checkMorph` - Checkbox morphs when checked

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Enter` | Add new todo |
| `Escape` | Clear input |

## Design System

### Colors

- **Background**: `#0a0a0f` (deep void)
- **Card**: `rgba(15, 23, 42, 0.6)` with backdrop blur
- **Accent Active**: `#22d3ee` (cyan)
- **Accent Complete**: `#a78bfa` (violet)
- **Text Primary**: `#f8fafc`
- **Text Secondary**: `#94a3b8`

### Typography

- **Display Font**: Space Grotesk (700 weight)
- **Body Font**: IBM Plex Sans (400/500/600 weight)

## Customization

### Adding More Filters

Edit the filter buttons in `page.templ`:

```html
<button class="filter-tab" data-filter="your-filter" onclick="setFilter('your-filter')">
    Your Label
</button>
```

Update the filter logic in the `updateUI()` function:

```javascript
const filteredTodos = todos.filter(t => {
    if (filter === 'your-filter') return /* your condition */;
    // ...
});
```

### Changing Animations

Animations are defined in `layout.templ` within the `<style>` block. Key animations:

- `.todo-item` - Entry animation
- `.todo-text.completed::after` - Strikethrough animation
- `.checkbox-visual` - Checkbox morph animation

## API Reference

### Todo Object

```typescript
interface Todo {
    id: string;           // Unique identifier
    text: string;         // Todo content
    completed: boolean;   // Completion status
    createdAt: number;      // Creation timestamp
}
```

### State Shape

```typescript
interface TodoState {
    todos: Todo[];        // Array of todos
    filter: 'all' | 'active' | 'completed';
    inputValue: string;   // Current input value
}
```

### JavaScript Functions

- `addTodo()` - Adds a new todo from input
- `toggleTodo(id)` - Toggles completion status
- `deleteTodo(id)` - Removes a todo
- `setFilter(filter)` - Changes the current filter
- `toggleAll()` - Toggles all todos
- `clearCompleted()` - Removes all completed todos

## Learn More

- [GoSPA Documentation](../../docs/)
- [Reactive Primitives](../../docs/STATE_PRIMITIVES.md)
- [Islands Architecture](../../docs/ISLANDS.md)

## License

Same as GoSPA - see [LICENSE](../../LICENSE)
