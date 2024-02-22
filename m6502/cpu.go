package m6502

import (
	"fmt"
)

type Status uint8

const (
	CarryFlag        Status = 1 << iota // Carry
	ZeroFlag                            // Zero
	InterruptDisable                    // Interrupt Disable
	DecimalMode                         // Decimal Mode
	BreakCmd                            // Break
	Unused                              // Unused
	OverflowFlag                        // Overflow
	NegativeFlag                        // Negative
)

type Registers struct {
	Accumulator    uint8
	IndexX         uint8
	IndexY         uint8
	ProcStatus     Status
	StackPtr       uint8
	ProgramCounter uint16
}

func NewRegisters() (reg Registers) {
	reg = Registers{}
	reg.Reset()
	return
}

func (reg *Registers) Reset() {
	reg.Accumulator = 0
	reg.IndexX = 0
	reg.IndexY = 0
	reg.ProcStatus = InterruptDisable | Unused
	reg.StackPtr = 0xfd
	reg.ProgramCounter = 0xfffc
}

type decode struct {
	enable      bool
	pc          uint16
	opCode      OpCode
	args        string
	mnemonic    string
	decodedArgs string
	registers   string
	ticks       uint64
}

type Interrupt uint8

const (
	IRQ Interrupt = iota
	Nmi
	Rst
)

type Index uint8

const (
	X Index = iota
	Y
)

func (reg *Registers) String() string {
	return fmt.Sprintf("A:%02x X:%02x Y:%02x P:%02x SP:%02x PC:%04x",
		reg.Accumulator, reg.IndexX, reg.IndexY, reg.ProcStatus, reg.StackPtr, reg.ProgramCounter)
}

type M6502 struct {
	decode       decode
	Nmi          bool
	IRQ          bool
	Rst          bool
	Registers    Registers
	Memory       Memory
	Instructions InstructionTable
	decimalMode  bool
	breakError   bool
}

func newM6502(mem Memory) *M6502 {
	instructions := NewInstructionTable()
	instructions.InitInstructions()
	return &M6502{
		decode:       decode{},
		Registers:    NewRegisters(),
		Memory:       mem,
		Instructions: instructions,
		decimalMode:  true,
		breakError:   false,
		Nmi:          false,
		IRQ:          false,
		Rst:          false,
	}
}

func (cpu *M6502) Reset() {
	cpu.Registers.Reset()
	cpu.Memory.Reset()
	cpu.PerformRst()
}

func (cpu *M6502) Interrupt(which Interrupt, state bool) {
	switch which {
	case IRQ:
		cpu.IRQ = state
	case Nmi:
		cpu.Nmi = state
	case Rst:
		cpu.Rst = state
	}
}

func (cpu *M6502) InterruptLine(which Interrupt) func(state bool) {
	return func(state bool) {
		if cpu != nil {
			cpu.Interrupt(which, state)
		}
	}
}

func (cpu *M6502) GetInterrupt(which Interrupt) (state bool) {
	switch which {
	case IRQ:
		state = cpu.IRQ
	case Nmi:
		state = cpu.Nmi
	case Rst:
		state = cpu.Rst
	}
	return
}

func (cpu *M6502) PerformInterrupts() (cycles uint16) {
	cycles = 7 // default cycles for an interrupt

	switch {
	case cpu.IRQ && cpu.Registers.ProcStatus&InterruptDisable == 0:
		cpu.PerformIrq()
		cpu.IRQ = false
	case cpu.Nmi:
		cpu.PerformNmi()
		cpu.Nmi = false
	case cpu.Rst:
		cpu.performRst()
		cpu.Rst = false
	default:
		cycles = 0
	}
	return
}

// TODO Build Loader
func (cpu *M6502) load() {
	return
}

func (cpu *M6502) Lda(address uint16) {
	cpu.load(address, &cpu.Registers.Accumulator)
}
