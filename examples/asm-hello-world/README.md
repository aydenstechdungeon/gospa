# Assembly Architectural Touchstones

This directory contains standalone, bare-metal assembly demonstrations for the x86-64 Linux architecture.

## Why Assembly in GoSPA?

While GoSPA is primarily a high-level framework for Single Page Applications, we believe that understanding the underlying machine execution is crucial for:

1.  **Compiler Optimization Performance**: Go's own compiler uses Plan 9 assembly conventions. These examples serve as a bridge to understanding how instructions map to the CPU.
2.  **Zero-Dependency System Programming**: These scripts use direct Linux `syscalls`, bypassing `libc` and the Go runtime entirely.
3.  **Low-Level Insight**: Demonstrating the "true" costs of I/O and process exit.

## Included Examples

-   `hello_linux_x86_64.s`: A pure assembly "Hello World" using `sys_write` and `sys_exit`.

## Usage

A `Makefile` is provided for convenience.

```bash
# Build all examples
make

# Run the hello world example
make run

# Clean up binaries
make clean
```

> [!NOTE]
> These examples are currently targeted at `x86_64` Linux environments. Execution on other architectures (ARM/M1) or OSes (macOS/Windows) is not supported natively.
