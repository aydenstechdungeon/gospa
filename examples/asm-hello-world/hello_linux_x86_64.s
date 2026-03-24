# ==============================================================================
# GoSPA: Low-Level Assembly Demonstration
# ==============================================================================
# Architecture: x86_64
# OS: Linux (System V AMD64 ABI)
# Toolchain: GNU Assembler (as) & Linker (ld)
#
# This file serves as a architectural touchstone within the GoSPA project.
# While Go is the primary language, understanding the machine-level
# execution is vital for high-performance SPA rendering and compiler
# optimization strategies.
# ==============================================================================

.section .data
    # The message string followed by a newline (0x0a)
    hello_msg:
        .ascii "Hello, World from the depths of GoSPA Assembly!\n"
    
    # Calculate the message length at assembly time
    hello_len = . - hello_msg

.section .text
    # Export the entry point for the linker (ld)
    .global _start

_start:
    # --------------------------------------------------------------------------
    # SYSCALL: sys_write (rax = 1)
    # --------------------------------------------------------------------------
    # rdi: uint fd       = 1 (STDOUT_FILENO)
    # rsi: const char *  = hello_msg (pointer to string)
    # rdx: size_t count  = hello_len (number of bytes)
    # --------------------------------------------------------------------------
    movq $1, %rax           # syscall number for sys_write
    movq $1, %rdi           # file descriptor 1 (stdout)
    leaq hello_msg(%rip), %rsi # load address of message (RIP-relative)
    movq $hello_len, %rdx    # message length
    syscall                 # invoke the kernel

    # --------------------------------------------------------------------------
    # SYSCALL: sys_exit (rax = 60)
    # --------------------------------------------------------------------------
    # rdi: int error_code = 0 (SUCCESS)
    # --------------------------------------------------------------------------
    movq $60, %rax          # syscall number for sys_exit
    xorq %rdi, %rdi         # exit status 0 (xor-clearing is idiomatic)
    syscall                 # invoke the kernel

# ==============================================================================
# Compilation Instructions:
# ==============================================================================
# as -o hello.o hello_linux_x86_64.s
# ld -o hello hello.o
# ./hello
# ==============================================================================
