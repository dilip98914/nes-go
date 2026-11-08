package main

import "fmt"

type CPU struct {
	// cpu registers
	A  byte
	X  byte
	Y  byte
	PC uint16
	SP byte

	// status flag
	Status byte

	//connection to system memory
	Bus    *Bus
	Cycles uint64
}

func (cpu *CPU) Reset() error {
	cpu.A = 0
	cpu.X = 0
	cpu.Y = 0
	//stack 0x0100–0x01FF
	cpu.SP = 0xFD
	cpu.Status = 0x24

	// reset vector read
	low, err := cpu.Bus.CpuRead(0xFFFC)
	if err != nil {
		return err
	}

	high, err := cpu.Bus.CpuRead(0xFFFD)
	if err != nil {
		return err
	}

	cpu.PC = uint16(low) | uint16(high)<<8

	cpu.Cycles = 0
	return nil
}

func (cpu *CPU) FetchByte() (byte, error) {
	value, err := cpu.Bus.CpuRead(cpu.PC)
	if err != nil {
		return 0, err
	}
	cpu.PC++
	return value, nil
}

func (cpu *CPU) Step() error {
	opcodeAddress := cpu.PC
	opcode, err := cpu.FetchByte()
	if err != nil {
		return err
	}
	var cycles uint64

	switch opcode {
	case 0xA9: //LDA immediate
		cycles = 2
		return cpu.ExecuteLDA()
	case 0x78:
		cycles = 2
		cpu.ExecuteSEI()
		return nil
	case 0xD8:
		cycles = 2
		cpu.ExecuteCLD()
		return nil
	case 0x8D:
		cycles = 4
		return cpu.ExecuteSTAAbsolute()
	case 0xA2:
		cycles = 2
		return cpu.ExecuteLDX()
	case 0xAD: // LDA absolute
		cycles = 4
		return cpu.ExecuteLDAAbsolute()
	case 0xCA: // DEX
		cycles = 2
		cpu.ExecuteDEX()
	case 0x9A: // TXS
		cycles = 2
		cpu.ExecuteTXS()
	case 0xD0: // BNE
		branchTaken := cpu.Status&0x02 == 0
		nextPC := cpu.PC + 1

		cycles = 2
		err = cpu.ExecuteBNE()

		if err == nil && branchTaken {
			cycles++

			if nextPC&0xFF00 != cpu.PC&0xFF00 {
				cycles++
			}
		}
	case 0x10: // BPL - Branch if Plus
		branchTaken := cpu.Status&0x80 == 0
		nextPC := cpu.PC + 1

		cycles = 2
		err = cpu.ExecuteBPL()
		if err == nil && branchTaken {
			cycles++
			//additional cycle if branch crosses a page
			if nextPC&0xFF00 != cpu.PC&0xFF00 {
				cycles++
			}
		}

	default:
		return fmt.Errorf(
			"unsupported opcode: 0x%02X at address 0x%04X",
			opcode,
			opcodeAddress,
		)
	}

	if err != nil {
		return err
	}
	cpu.Cycles += cycles
	cpu.Bus.Clock(cycles)
	return nil
}
