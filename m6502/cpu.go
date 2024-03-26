package m6502

import (
	"fmt"
	"strings"
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
	opcode      OpCode
	args        string
	mnemonic    string
	decodedArgs string
	registers   string
	ticks       uint64
}

func (d *decode) String() string {
	return fmt.Sprintf("%04X  %02X %-5s %4s %-26s  %25s",
		d.pc, d.opcode, d.args, d.mnemonic, d.decodedArgs, d.registers)
}

//go:generate stringer -type=Index
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
		cpu.PerformRst()
		cpu.Rst = false
	default:
		cycles = 0
	}
	return
}

func (cpu *M6502) push(value uint8) {
	cpu.Memory.Store(0x0100|uint16(cpu.Registers.StackPtr), value)
	cpu.Registers.StackPtr--
}

func (cpu *M6502) push16(value uint16) {
	cpu.push(uint8(value >> 8))
	cpu.push(uint8(value))
}

func (cpu *M6502) pull() (value uint8) {
	cpu.Registers.StackPtr++
	value = cpu.Memory.Fetch(0x0100 | uint16(cpu.Registers.StackPtr))
	return
}

func (cpu *M6502) pull16() (value uint16) {
	low := cpu.pull()
	high := cpu.pull()

	value = (uint16(high) << 8) | uint16(low)
	return
}

func (cpu *M6502) PerformIrq() {
	cpu.push16(cpu.Registers.ProgramCounter)
	cpu.push(uint8((cpu.Registers.ProcStatus | Unused) & ^BreakCmd))

	cpu.Registers.ProcStatus |= InterruptDisable

	low := cpu.Memory.Fetch(0xfffe)
	high := cpu.Memory.Fetch(0xffff)

	cpu.Registers.ProgramCounter = (uint16(high) << 8) | uint16(low)
}

func (cpu *M6502) PerformNmi() {
	cpu.push16(cpu.Registers.ProgramCounter)
	cpu.push(uint8((cpu.Registers.ProcStatus | Unused) & ^BreakCmd))

	cpu.Registers.ProcStatus |= InterruptDisable

	low := cpu.Memory.Fetch(0xfffa)
	high := cpu.Memory.Fetch(0xfffb)

	cpu.Registers.ProgramCounter = (uint16(high) << 8) | uint16(low)
}

func (cpu *M6502) PerformRst() {
	low := cpu.Memory.Fetch(0xfffc)
	high := cpu.Memory.Fetch(0xfffd)

	cpu.Registers.ProgramCounter = (uint16(high) << 8) | uint16(low)
}

func (cpu *M6502) DisableDecimalMode() {
	cpu.decimalMode = false
}

func (cpu *M6502) EnableDecode() {
	cpu.decode.enable = true
}

func (cpu *M6502) ToggleDecode() bool {
	cpu.decode.enable = !cpu.decode.enable
	return cpu.decode.enable
}

type BadOpCodeError OpCode

func (b BadOpCodeError) Error() string {
	return fmt.Sprintf("No such opcode %#02x", OpCode(b))
}

type BrkOpCodeError OpCode

func (b BrkOpCodeError) Error() string {
	return fmt.Sprintf("Executed BRK opcode")
}

func (cpu *M6502) Execute() (cycles uint16, error error) {
	cycles += cpu.PerformInterrupts()

	opcode := OpCode(cpu.Memory.Fetch(cpu.Registers.ProgramCounter))
	inst := cpu.Instructions.opcodes[opcode]

	if inst == nil {
		return 0, BadOpCodeError(opcode)
	}

	if cpu.decode.enable {
		cpu.decode.pc = cpu.Registers.ProgramCounter
		cpu.decode.opcode = opcode
		cpu.decode.args = ""
		cpu.decode.mnemonic = inst.Mnemonic
		cpu.decode.decodedArgs = ""
		cpu.decode.registers = cpu.Registers.String()
	}

	cpu.Registers.ProgramCounter++
	cycles += cpu.Instructions.Excute(cpu, opcode)

	if cpu.decode.enable {
		fmt.Println(cpu.decode.String())
	}

	if cpu.breakError && opcode == 0x00 {
		return cycles, BrkOpCodeError(opcode)
	}

	return cycles, nil
}

func (cpu *M6502) Run() (err error) {
	for {
		if _, err = cpu.Execute(); err != nil {
			return
		}
	}
}

func (cpu *M6502) setZFlag(value uint8) uint8 {
	if value == 0 {
		cpu.Registers.ProcStatus |= ZeroFlag
	} else {
		cpu.Registers.ProcStatus &= ^ZeroFlag
	}

	return value
}

func (cpu *M6502) setNFlag(value uint8) uint8 {
	cpu.Registers.ProcStatus = (cpu.Registers.ProcStatus & ^NegativeFlag) | Status(value&uint8(NegativeFlag))
	return value
}

func (cpu *M6502) setZNFlags(value uint8) uint8 {
	cpu.setZFlag(value)
	cpu.setNFlag(value)
	return value
}

func (cpu *M6502) setCFlagAddition(value uint16) uint16 {
	cpu.Registers.ProcStatus = (cpu.Registers.ProcStatus & ^CarryFlag) | Status(value>>8&uint16(CarryFlag))
	return value
}

func (cpu *M6502) setVFlagAddition(term1 uint16, term2 uint16, result uint16) uint16 {
	cpu.Registers.ProcStatus = (cpu.Registers.ProcStatus & ^OverflowFlag) | Status((^(term1^term2)&(term1^result)&uint16(NegativeFlag))>>1)
	return result
}

func (cpu *M6502) immidiateAddress() (result uint16) {
	result = cpu.Registers.ProgramCounter
	cpu.Registers.ProgramCounter++

	if cpu.decode.enable {
		value := cpu.Memory.Fetch(result)
		cpu.decode.args = fmt.Sprintf("%02X", value)
		cpu.decode.decodedArgs = fmt.Sprintf("#$")
	}
	return
}

func (cpu *M6502) IndexToRegister(which Index) uint8 {
	var index uint8

	switch which {
	case X:
		index = cpu.Registers.IndexX

	case Y:
		index = cpu.Registers.IndexY
	}
	return index
}

func (cpu *M6502) zeroPageAddress() (result uint16) {
	result = uint16(cpu.Memory.Fetch(cpu.Registers.ProgramCounter))
	cpu.Registers.ProgramCounter++

	if cpu.decode.enable {
		cpu.decode.args = fmt.Sprintf("%02X", result)
		cpu.decode.decodedArgs = fmt.Sprintf("$%02X", result)
	}
	return
}

func (cpu *M6502) zeroPageIndexedAddress(index Index) (result uint16) {
	value := cpu.Memory.Fetch(cpu.Registers.ProgramCounter)
	result = uint16(value + cpu.IndexToRegister(index))
	cpu.Registers.ProgramCounter++

	if cpu.decode.enable {
		cpu.decode.args = fmt.Sprintf("$%02X", value)
		cpu.decode.decodedArgs = fmt.Sprintf("$%02X,%s @ %02X", value, index.String(), result)
	}
	return
}

func (cpu *M6502) load(address uint16, register *uint8) {
	value := cpu.setZNFlags(cpu.Memory.Fetch(address))
	*register = value

	if cpu.decode.enable {
		if !strings.HasPrefix(cpu.decode.decodedArgs, "#") && !strings.HasSuffix(cpu.decode.decodedArgs, " = ") {
			cpu.decode.decodedArgs += fmt.Sprintf(" = ")
		}
		cpu.decode.decodedArgs += fmt.Sprintf("%02X", value)
	}
}

func (cpu *M6502) absoluteAddress() (result uint16) {
	low := cpu.Memory.Fetch(cpu.Registers.ProgramCounter)
	high := cpu.Memory.Fetch(cpu.Registers.ProgramCounter + 1)
	cpu.Registers.ProgramCounter += 2

	result = uint16(high)<<8 | uint16(low)

	if cpu.decode.enable {
		cpu.decode.args = fmt.Sprintf("%02X %02X", low, high)
		cpu.decode.decodedArgs = fmt.Sprintf("$%04X", result)
	}
	return
}

func (cpu *M6502) Lda(address uint16) {
	cpu.load(address, &cpu.Registers.Accumulator)
}

func (cpu *M6502) controlAddress(opcode OpCode, status *InstructionStatus) (address uint16) {
	if opcode&0x10 == 0 {
		switch (opcode >> 2) & 0x03 {
		case 0x00:
			address = cpu.immidiateAddress()
		case 0x01:
			address = cpu.zeroPageAddress()
		case 0x02:
			address = 0 // Unused
		case 0x03:
			address = cpu.absoluteAddress()
		}
	} else {
		switch (opcode >> 2) & 0x03 {
		case 0x00:
			address = cpu.immidiateAddress()
		case 0x01:
			address = cpu.zeroPageAddress()
		case 0x02:
			address = 0
		case 0x03:
			address = cpu.absoluteAddress()
		}
	}
}
