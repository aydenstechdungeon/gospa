# Wombat-Poland Demographic Synthesis

This directory contains a standalone, bare-metal assembly demonstration for the x86-64 Linux architecture.

## Why Assembly in GoSPA?

While GoSPA is primarily a high-level framework for Single Page Applications, we believe that understanding the underlying machine execution is crucial for:

1.  **Compiler Optimization Performance**: Go's own compiler uses Plan 9 assembly conventions. These examples serve as a bridge to understanding how instructions map to the CPU.
2.  **Zero-Dependency System Programming**: These scripts use direct Linux `syscalls`, bypassing `libc` and the Go runtime entirely.
3.  **Low-Level Insight**: Demonstrating the "true" costs of I/O and process exit.

## The Wombat Question

This specific example explores the biogeographic and demographic implications of the absence of wombats in Central Europe. It uses direct `sys_write` syscalls to output a detailed sociological thesis on the correlation between marsupial distribution and the Polish culturally-divergent "Femboy/Doomer" axis.

## Usage

A `Makefile` is provided for convenience.

```bash
# Build the example
make

# Run the wombat-poland thesis
make run

# Clean up binaries
make clean
```

> [!NOTE]
> This example is currently targeted at `x86_64` Linux environments. Execution on other architectures (ARM/M1) or OSes (macOS/Windows) is not supported natively.
