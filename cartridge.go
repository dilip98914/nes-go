package main

import "fmt"

type Cartridge struct {
	PRGROM   []byte
	CHRROM   []byte
	Mapper   uint16
	PRGBanks uint8
	CHRBanks uint8
}

func NewCartridge(rom []byte) (*Cartridge, error) {
	if len(rom) < 16 {
		return nil, fmt.Errorf("invalid ROM: header too small")
	}

	header := rom[:16]
	if string(header[:4]) != "NES\x1A" {
		return nil, fmt.Errorf("invalid NES signature")
	}
	mapper := uint16(header[6]>>4) |
		uint16(header[7]&0xF0)

	if header[7]&0x0C == 0x08 {
		return nil, fmt.Errorf("NES 2.0 not supported yet")
	}

	prgSize := int(header[4]) * 16 * 1024
	chrSize := int(header[5]) * 8 * 1024

	offset := 16

	if header[6]&0x04 != 0 {
		offset += 512
	}
	required := offset + prgSize + chrSize
	if prgSize == 0 || len(rom) < required {
		return nil, fmt.Errorf("invalid or incomplete cartridge data")
	}
	prgROM := rom[offset : offset+prgSize]
	offset += prgSize

	chrROM := rom[offset : offset+chrSize]

	return &Cartridge{
		PRGROM:   prgROM,
		CHRROM:   chrROM,
		Mapper:   mapper,
		PRGBanks: header[4],
		CHRBanks: header[5],
	}, nil
}
