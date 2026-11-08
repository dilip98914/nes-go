package main

type APU struct {
	Status byte

	Registers [0x18]byte
}

func (a *APU) CPURead(addr uint16) byte {
	if addr == 0x4015 {
		value := a.Status
		a.Status &^= 0x40
		return value
	}

	return 0
}

func (a *APU) CPUWrite(addr uint16, value byte) {
	if addr >= 0x4000 && addr <= 0x4013 {
		a.Registers[addr-0x4000] = value
		return
	}

	switch addr {
	case 0x4015:
		a.Status = value & 0x1F

	case 0x4017:
		a.Registers[0x17] = value
	}
}
