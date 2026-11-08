package main

import "fmt"

const RAMSIZE = 2048

type Bus struct {
	RAM    [RAMSIZE]byte
	PRGRAM [8192]byte

	PPU    PPU
	Mapper *Mapper4
	APU    APU
}

func (b *Bus) CpuRead(addr uint16) (byte, error) {
	// fmt.Printf("BUS READ: 0x%04X\n", addr)

	// Internal ram & its mirrors
	if addr <= 0x1FFF {
		ramAddr := addr % RAMSIZE
		return b.RAM[ramAddr], nil
	}

	// ppu registers and mirrors
	if addr >= 0x2000 && addr <= 0x3FFF {
		return b.PPU.CPURead(addr), nil
	}
	// fmt.Printf("CHECK APU: addr=0x%04X, equal=%v\n", addr, addr == 0x4015)

	if addr == 0x4015 {
		return b.APU.CPURead(addr), nil
	}

	// cartirage PRG RAM
	if addr >= 0x6000 && addr <= 0x7FFF {
		return b.PRGRAM[addr-0x6000], nil
	}

	// cartridge PRG ROM
	if addr >= 0x8000 && b.Mapper != nil {
		offset, ok := b.Mapper.CPURead(addr)
		if !ok {
			return 0, fmt.Errorf("mapper cannot read the addr: 0x%04X", addr)
		}
		if int(offset) >= len(b.Mapper.PRGROM) {
			return 0, fmt.Errorf("PRG ROM offset out of bounds: %d", offset)
		}
		return b.Mapper.PRGROM[offset], nil
	}

	return 0, fmt.Errorf(
		"DEBUG: unsupported address 0x%04X",
		addr,
	)
}

func (b *Bus) CpuWrite(addr uint16, value byte) error {
	// Internal RAM and its mirrors.
	if addr <= 0x1FFF {
		ramAddr := addr % RAMSIZE
		b.RAM[ramAddr] = value
		return nil
	}

	// ppu
	if addr >= 0x2000 && addr <= 0x3FFF {
		b.PPU.CPUWrite(addr, value)
		return nil
	}
	// apu
	if addr >= 0x4000 && addr <= 0x4017 {
		b.APU.CPUWrite(addr, value)
		return nil
	}

	if addr >= 0x6000 && addr <= 0x7FFF {
		b.PRGRAM[addr-0x6000] = value
		return nil
	}

	if addr >= 0x8000 && b.Mapper != nil {
		b.Mapper.CPUWrite(addr, value)
		return nil
	}

	return fmt.Errorf("unsupported CPU write address: 0x%04X", addr)
}

func (b *Bus) Clock(cpuCycles uint64) {
	for i := uint64(0); i < cpuCycles*3; i++ {
		b.PPU.Clock()
	}
}
