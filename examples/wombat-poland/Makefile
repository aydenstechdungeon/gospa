# ==============================================================================
# Makefile for Wombat-Poland Assembly Demonstration
# ==============================================================================

# Tools
AS = as
LD = ld

# Source and Binary
SRC = main.s
OBJ = main.o
EXE = wombat

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
