# ==============================================================================
# Makefile for x86_64 Assembly Demonstration
# ==============================================================================
# This build script facilitates the assembly and linking of the Linux-native
# "Hello World" example. It's intended for academic exploration within the
# GoSPA ecosystem.
# ==============================================================================

# Tools
AS = as
LD = ld

# Source and Binary
SRC = hello_linux_x86_64.s
OBJ = hello.o
EXE = hello

.PHONY: all clean run

all: $(EXE)

$(EXE): $(OBJ)
	$(LD) -o $@ $^

$(OBJ): $(SRC)
	$(AS) -o $@ $<

run: $(EXE)
	./$(EXE)

clean:
	rm -f $(OBJ) $(EXE)

# ==============================================================================
