# State Management Patterns

Best practices for building large-scale reactive applications with GoSPA.

## 1. Minimal Reactive Surface

Only mark variables as `$state()` if they actually change and need to trigger UI updates. Over-reactivity can lead to unnecessary processing.

## 2. Server-side Validation

Always validate state updates on the server. The `StateMap` provides an `OnChange` hook that is perfect for this.

```go
stateMap.OnChange = func(key string, value any) {
    if key == "email" {
        validateEmail(value.(string))
    }
}
```

## 3. Use Derived for Logic

Instead of manually updating multiple states, use `$derived()` to compute values from source state. This ensures your UI is always consistent with the underlying data.

## 4. Batch Complex Updates

When performing multiple related state changes, wrap them in a `Batch()` call to prevent intermediate, inconsistent states from being synchronized to the client.

## 5. Pruning and Cleanup

Dispose of `Derived` and `Effect` primitives when they are no longer needed to prevent memory leaks, especially in long-running server processes or complex client-side interactions.
