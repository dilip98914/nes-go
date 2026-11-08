package main

type PPU struct {
	// ppu registers
	Control byte
	Mask    byte
	Status  byte
	OAMAddr byte
	OAMData byte
	Scroll  byte
	Address byte
	Data    byte
	// Last value driven onto the PPU data bus.
	Openbus byte

	Scanline int
	Dot      int
	Cycles   uint64
}

func (p *PPU) CPURead(addr uint16) byte {
	register := addr & 0x0007
	switch register {
	case 2:
		// PPUSTATUS is readable.
		value := p.Status
		// Reading PPUSTATUS clears the VBlank flag.

		p.Status &^= 0x80
		return value
	default:
		// Other registers are write-only.
		return p.Openbus
	}
}

func (p *PPU) CPUWrite(addr uint16, value byte) {
	register := addr & 0x0007
	p.Openbus = value
	switch register {
	case 0:
		p.Control = value
	case 1:
		p.Mask = value
	// case 2:
	case 3:
		p.OAMAddr = value
	case 4:
		p.OAMData = value
	case 5:
		p.Scroll = value
	case 6:
		p.Address = value
	case 7:
		p.Data = value

	}
}

func (p *PPU) Clock() {
	p.Cycles++
	p.Dot++
	if p.Dot >= 341 {
		p.Dot = 0
		p.Scanline++

		if p.Scanline >= 262 {
			p.Scanline = 0
		}
	}

	// vblank begins at scaline 241,dot1
	if p.Scanline == 241 && p.Dot == 1 {
		p.Status |= 0x80
	}

	// clear vblank & sprite status flags at pre render scanline
	if p.Scanline == 261 && p.Dot == 1 {
		p.Status &^= 0xE0
	}

}
