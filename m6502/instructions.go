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
