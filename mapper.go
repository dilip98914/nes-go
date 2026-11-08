package main

const PRGBankSize = 1024 * 8

type Mapper interface {
	CPURead(addr uint16) (uint32, bool)
	CPUWrite(addr uint16, value byte) bool
	PPURead(addr uint16) (uint32, bool)
	PPUWrite(addr uint16) (uint32, bool)
}

type Mapper4 struct {
	PRGROM        []byte
	PRGBankCount  uint8
	BankSelect    byte
	BankRegisters [8]byte
}

func NewMapper4(cart *Cartridge) *Mapper4 {
	return &Mapper4{
		PRGROM:       cart.PRGROM,
		PRGBankCount: uint8(len(cart.PRGROM) / PRGBankSize),
	}
}

func (m *Mapper4) CPURead(addr uint16) (uint32, bool) {
	if addr < 0x8000 || m.PRGBankCount == 0 {
		return 0, false
	}
	var bank uint8
	lastBank := m.PRGBankCount - 1
	secondLastBank := m.PRGBankCount - 2
	prgMode := m.BankSelect & 0x40
	switch {
	case addr <= 0x9FFF:
		if prgMode == 0 {
			bank = m.BankRegisters[6] & 0x3F
		} else {
			bank = secondLastBank
		}
	case addr <= 0xBFFF:
		bank = m.BankRegisters[7] & 0x3F

	case addr <= 0xDFFF:
		if prgMode == 0 {
			bank = secondLastBank
		} else {
			bank = m.BankRegisters[6] & 0x3F
		}

	default:
		bank = lastBank
	}

	bank %= m.PRGBankCount
	offset := uint32(bank)*PRGBankSize + uint32(addr&0x1FFF)
	return offset, true
}

func (m *Mapper4) CPUWrite(addr uint16, value byte) bool {
	switch addr & 0xE001 {
	case 0x8000:
		// Select which bank register to update.
		m.BankSelect = value

	case 0x8001:
		register := m.BankSelect & 0x07
		m.BankRegisters[register] = value

	default:
		return false
	}
	return true
}
