package m6502

const (
	DefaultMemorySize uint32 = 65536
)

type Memory interface {
	Reset()
	Fetch(address uint16) uint8
	Store(address uint16, value uint8) (oldValue uint8)
}

type BasicMemory struct {
	M             []uint8
	DisableReads  bool
	DisableWrites bool
}
