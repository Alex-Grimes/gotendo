package m6502

type OpCode uint8

type Instruction struct {
	Mnemonic string
	OpCode   OpCode
	Exec     func(*M6502) (status InstructionStatus) // todo: build out M6502 struct
}

type InstructionTable struct {
	opcodes         []*Instruction
	cycles          []uint16
	cyclesPageCross []uint16
}

type InstructionStatus uint16

const (
	PageCross InstructionStatus = 1 << iota
	Branched
)

func NewInstructionTable() InstructionTable {
	instructions := InstructionTable{
		opcodes: make([]*Instruction, 0x100),
		cycles: []uint16{
			7, 6, 0, 8, 3, 3, 5, 5, 3, 2, 2, 2, 4, 4, 6, 6,
			2, 5, 0, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7,
			6, 6, 0, 8, 3, 3, 5, 5, 4, 2, 2, 2, 4, 4, 6, 6,
			2, 5, 0, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7,
			6, 6, 0, 8, 3, 3, 5, 5, 3, 2, 2, 2, 3, 4, 6, 6,
			2, 5, 0, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7,
			6, 6, 0, 8, 3, 3, 5, 5, 4, 2, 2, 2, 5, 4, 6, 6,
			2, 5, 0, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7,
			2, 6, 2, 6, 3, 3, 3, 3, 2, 2, 2, 2, 4, 4, 4, 4,
			2, 6, 0, 6, 4, 4, 4, 4, 2, 5, 2, 5, 5, 5, 5, 5,
			2, 6, 2, 6, 3, 3, 3, 3, 2, 2, 2, 2, 4, 4, 4, 4,
			2, 5, 0, 5, 4, 4, 4, 4, 2, 4, 2, 4, 4, 4, 4, 4,
			2, 6, 2, 8, 3, 3, 5, 5, 2, 2, 2, 2, 4, 4, 6, 6,
			2, 5, 0, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7,
			2, 6, 2, 8, 3, 3, 5, 5, 2, 2, 2, 2, 4, 4, 6, 6,
			2, 5, 0, 8, 4, 4, 6, 6, 2, 4, 2, 7, 4, 4, 7, 7,
		},
		cyclesPageCross: []uint16{
			7, 6, 0, 8, 3, 3, 5, 5, 3, 2, 2, 2, 4, 4, 6, 6,
			3, 6, 0, 8, 4, 4, 6, 6, 2, 5, 2, 7, 5, 5, 7, 7,
			6, 6, 0, 8, 3, 3, 5, 5, 4, 2, 2, 2, 4, 4, 6, 6,
			3, 6, 0, 8, 4, 4, 6, 6, 2, 5, 2, 7, 5, 5, 7, 7,
			6, 6, 0, 8, 3, 3, 5, 5, 3, 2, 2, 2, 3, 4, 6, 6,
			3, 6, 0, 8, 4, 4, 6, 6, 2, 5, 2, 7, 5, 5, 7, 7,
			6, 6, 0, 8, 3, 3, 5, 5, 4, 2, 2, 2, 5, 4, 6, 6,
			3, 6, 0, 8, 4, 4, 6, 6, 2, 5, 2, 7, 5, 5, 7, 7,
			2, 6, 2, 6, 3, 3, 3, 3, 2, 2, 2, 2, 4, 4, 4, 4,
			3, 6, 0, 6, 4, 4, 4, 4, 2, 5, 2, 5, 5, 5, 5, 5,
			2, 6, 2, 6, 3, 3, 3, 3, 2, 2, 2, 2, 4, 4, 4, 4,
			3, 6, 0, 6, 4, 4, 4, 4, 2, 5, 2, 5, 5, 5, 5, 5,
			2, 6, 2, 8, 3, 3, 5, 5, 2, 2, 2, 2, 4, 4, 6, 6,
			3, 6, 0, 8, 4, 4, 6, 6, 2, 5, 2, 7, 5, 5, 7, 7,
			2, 6, 2, 8, 3, 3, 5, 5, 2, 2, 2, 2, 4, 4, 6, 6,
			3, 6, 0, 8, 4, 4, 6, 6, 2, 5, 2, 7, 5, 5, 7, 7,
		},
	}
	return instructions
}

func (instructions InstructionTable) Excute(cpu *M6502, opcode OpCode) (cycles uint16) {
	inst := instructions.opcodes[opcode]

	if inst == nil {
		return
	}

	status := inst.Exec(cpu)

	if status&PageCross == 0 {
		cycles = instructions.cycles[opcode]
	} else {
		cycles = instructions.cyclesPageCross[opcode]
	}

	if status&Branched != 0 {
		cycles++
	}

	return
}

func (instructions InstructionTable) AddInstruction(inst *Instruction) {
	instructions.opcodes[inst.OpCode] = inst
}

func (instructions InstructionTable) RemoveInstruction(opcode OpCode) {
	instructions.opcodes[opcode] = nil
}

func (instructions InstructionTable) InitInstructions() {
	// LDA

	for _, o := range []OpCode{0xa1, 0xa5, 0xa9, 0xad, 0xb1, 0xb5, 0xb9, 0xbd} {
		opcode := o

		instructions.AddInstruction(&Instruction{
			Mnemonic: "LDA",
			OpCode:   opcode,
			Exec: func(cpu *M6502) (status InstructionStatus) {
				cpu.Lda(cpu.aluAddress(opcode, &status))
				return
			},
		})
	}

	// LDX

	// LDY

	// STA

	// STX

	// STY

	// TAX

	// TAY

	// TXA

	// TYA

	// TSX

	// TXS

	// PHA

	// PHP

	// PLA

	// PLP

	// AND

	// EOR

	// ORA

	// BIT

	// ADC

	// SBC

	// DCP

	// ISB

	// SLO

	// RLA

	// SRE

	// RRA

	// CMP

	// CPX

	// CPY

	// INC

	// INX

	// INY

	// DEC

	// DEX

	// DEY

	// ASL

	// LSR

	// ROL

	// ROR

	// JMP

	// JSR

	// RTS

	// BCC

	// BCS

	// BEQ

	// BMI

	// BNE

	// BPL

	// BVC

	// BVS

	// CLC

	// CLD

	// CLI

	// CLV

	// SEC

	// SED

	// SEI

	// BRK

	// NOP

	// LAX

	// SAX

	// ANC

	// ALR

	// ARR

	// AXS

	// SHY

	// SHX

	// RTI
}
