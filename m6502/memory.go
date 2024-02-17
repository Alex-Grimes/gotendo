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

func newBasicMemory(size uint32) *BasicMemory {
	return &BasicMemory{
		M: make([]uint8, size),
	}
}

func (m *BasicMemory) Reset() {
	for i := range m.M {
		m.M[i] = 0xff
	}
}

func (m *BasicMemory) Fetch(address uint16) (value uint8) {
	if m.DisableReads {
		value = 0xff
	} else {
		value = m.M[address]
	}
	return
}

func (m *BasicMemory) Store(address uint16, value uint8) (oldValue uint8) {
	if m.DisableWrites {
		oldValue = m.M[address]
		m.M[address] = value
	}
	return
}

func SamePage(address1 uint16, address2 uint16) bool {
	return (address1^address2)>>8 == 0
}
