package main

import "fmt"

type CPU struct {
	A      byte
	X      byte
	Y      byte
	PC     uint16
	SP     byte
	Status byte

	Bus    *Bus
	Cycles uint64

	// Additional cycles for branches and indexed page crossings.
	extraCycles uint64
}

func (cpu *CPU) Reset() error {
	cpu.A = 0
	cpu.X = 0
	cpu.Y = 0
	cpu.SP = 0xFD
	cpu.Status = 0x24

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
	cpu.extraCycles = 0

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

	cpu.extraCycles = 0

	var cycles uint64

	switch opcode {

	// 0x00 - 0x1F

	case 0x00: // BRK
		cycles = 7
		err = cpu.ExecuteBRKImplied()

	case 0x01: // ORA (Indirect,X)
		cycles = 6
		err = cpu.ExecuteORAIndexedIndirect()

	case 0x05: // ORA Zero Page
		cycles = 3
		err = cpu.ExecuteORAZeroPage()

	case 0x06: // ASL Zero Page
		cycles = 5
		err = cpu.ExecuteASLZeroPage()

	case 0x08: // PHP
		cycles = 3
		err = cpu.ExecutePHPImplied()

	case 0x09: // ORA Immediate
		cycles = 2
		err = cpu.ExecuteORAImmediate()

	case 0x0A: // ASL Accumulator
		cycles = 2
		err = cpu.ExecuteASLAccumulator()

	case 0x0D: // ORA Absolute
		cycles = 4
		err = cpu.ExecuteORAAbsolute()

	case 0x0E: // ASL Absolute
		cycles = 6
		err = cpu.ExecuteASLAbsolute()

	case 0x10: // BPL
		cycles = 2
		err = cpu.ExecuteBPLRelative()

	case 0x11: // ORA (Indirect),Y
		cycles = 5
		err = cpu.ExecuteORAIndirectIndexed()

	case 0x15: // ORA Zero Page,X
		cycles = 4
		err = cpu.ExecuteORAZeroPageX()

	case 0x16: // ASL Zero Page,X
		cycles = 6
		err = cpu.ExecuteASLZeroPageX()

	case 0x18: // CLC
		cycles = 2
		err = cpu.ExecuteCLCImplied()

	case 0x19: // ORA Absolute,Y
		cycles = 4
		err = cpu.ExecuteORAAbsoluteY()

	case 0x1D: // ORA Absolute,X
		cycles = 4
		err = cpu.ExecuteORAAbsoluteX()

	case 0x1E: // ASL Absolute,X
		cycles = 7
		err = cpu.ExecuteASLAbsoluteX()

	// 0x20 - 0x3F

	case 0x20: // JSR
		cycles = 6
		err = cpu.ExecuteJSRAbsolute()

	case 0x21: // AND (Indirect,X)
		cycles = 6
		err = cpu.ExecuteANDIndexedIndirect()

	case 0x24: // BIT Zero Page
		cycles = 3
		err = cpu.ExecuteBITZeroPage()

	case 0x25: // AND Zero Page
		cycles = 3
		err = cpu.ExecuteANDZeroPage()

	case 0x26: // ROL Zero Page
		cycles = 5
		err = cpu.ExecuteROLZeroPage()

	case 0x28: // PLP
		cycles = 4
		err = cpu.ExecutePLPImplied()

	case 0x29: // AND Immediate
		cycles = 2
		err = cpu.ExecuteANDImmediate()

	case 0x2A: // ROL Accumulator
		cycles = 2
		err = cpu.ExecuteROLAccumulator()

	case 0x2C: // BIT Absolute
		cycles = 4
		err = cpu.ExecuteBITAbsolute()

	case 0x2D: // AND Absolute
		cycles = 4
		err = cpu.ExecuteANDAbsolute()

	case 0x2E: // ROL Absolute
		cycles = 6
		err = cpu.ExecuteROLAbsolute()

	case 0x30: // BMI
		cycles = 2
		err = cpu.ExecuteBMIRelative()

	case 0x31: // AND (Indirect),Y
		cycles = 5
		err = cpu.ExecuteANDIndirectIndexed()

	case 0x35: // AND Zero Page,X
		cycles = 4
		err = cpu.ExecuteANDZeroPageX()

	case 0x36: // ROL Zero Page,X
		cycles = 6
		err = cpu.ExecuteROLZeroPageX()

	case 0x38: // SEC
		cycles = 2
		err = cpu.ExecuteSECImplied()

	case 0x39: // AND Absolute,Y
		cycles = 4
		err = cpu.ExecuteANDAbsoluteY()

	case 0x3D: // AND Absolute,X
		cycles = 4
		err = cpu.ExecuteANDAbsoluteX()

	case 0x3E: // ROL Absolute,X
		cycles = 7
		err = cpu.ExecuteROLAbsoluteX()

	// 0x40 - 0x5F

	case 0x40: // RTI
		cycles = 6
		err = cpu.ExecuteRTIImplied()

	case 0x41: // EOR (Indirect,X)
		cycles = 6
		err = cpu.ExecuteEORIndexedIndirect()

	case 0x45: // EOR Zero Page
		cycles = 3
		err = cpu.ExecuteEORZeroPage()

	case 0x46: // LSR Zero Page
		cycles = 5
		err = cpu.ExecuteLSRZeroPage()

	case 0x48: // PHA
		cycles = 3
		err = cpu.ExecutePHAImplied()

	case 0x49: // EOR Immediate
		cycles = 2
		err = cpu.ExecuteEORImmediate()

	case 0x4A: // LSR Accumulator
		cycles = 2
		err = cpu.ExecuteLSRAccumulator()

	case 0x4C: // JMP Absolute
		cycles = 3
		err = cpu.ExecuteJMPAbsolute()

	case 0x4D: // EOR Absolute
		cycles = 4
		err = cpu.ExecuteEORAbsolute()

	case 0x4E: // LSR Absolute
		cycles = 6
		err = cpu.ExecuteLSRAbsolute()

	case 0x50: // BVC
		cycles = 2
		err = cpu.ExecuteBVCRelative()

	case 0x51: // EOR (Indirect),Y
		cycles = 5
		err = cpu.ExecuteEORIndirectIndexed()

	case 0x55: // EOR Zero Page,X
		cycles = 4
		err = cpu.ExecuteEORZeroPageX()

	case 0x56: // LSR Zero Page,X
		cycles = 6
		err = cpu.ExecuteLSRZeroPageX()

	case 0x58: // CLI
		cycles = 2
		err = cpu.ExecuteCLIImplied()

	case 0x59: // EOR Absolute,Y
		cycles = 4
		err = cpu.ExecuteEORAbsoluteY()

	case 0x5D: // EOR Absolute,X
		cycles = 4
		err = cpu.ExecuteEORAbsoluteX()

	case 0x5E: // LSR Absolute,X
		cycles = 7
		err = cpu.ExecuteLSRAbsoluteX()

	// 0x60 - 0x7F

	case 0x60: // RTS
		cycles = 6
		err = cpu.ExecuteRTSImplied()

	case 0x61: // ADC (Indirect,X)
		cycles = 6
		err = cpu.ExecuteADCIndexedIndirect()

	case 0x65: // ADC Zero Page
		cycles = 3
		err = cpu.ExecuteADCZeroPage()

	case 0x66: // ROR Zero Page
		cycles = 5
		err = cpu.ExecuteRORZeroPage()

	case 0x68: // PLA
		cycles = 4
		err = cpu.ExecutePLAImplied()

	case 0x69: // ADC Immediate
		cycles = 2
		err = cpu.ExecuteADCImmediate()

	case 0x6A: // ROR Accumulator
		cycles = 2
		err = cpu.ExecuteRORAccumulator()

	case 0x6C: // JMP Indirect
		cycles = 5
		err = cpu.ExecuteJMPIndirect()

	case 0x6D: // ADC Absolute
		cycles = 4
		err = cpu.ExecuteADCAbsolute()

	case 0x6E: // ROR Absolute
		cycles = 6
		err = cpu.ExecuteRORAbsolute()

	case 0x70: // BVS
		cycles = 2
		err = cpu.ExecuteBVSRelative()

	case 0x71: // ADC (Indirect),Y
		cycles = 5
		err = cpu.ExecuteADCIndirectIndexed()

	case 0x75: // ADC Zero Page,X
		cycles = 4
		err = cpu.ExecuteADCZeroPageX()

	case 0x76: // ROR Zero Page,X
		cycles = 6
		err = cpu.ExecuteRORZeroPageX()

	case 0x78: // SEI
		cycles = 2
		err = cpu.ExecuteSEIImplied()

	case 0x79: // ADC Absolute,Y
		cycles = 4
		err = cpu.ExecuteADCAbsoluteY()

	case 0x7D: // ADC Absolute,X
		cycles = 4
		err = cpu.ExecuteADCAbsoluteX()

	case 0x7E: // ROR Absolute,X
		cycles = 7
		err = cpu.ExecuteRORAbsoluteX()

	// 0x80 - 0x9F

	case 0x81: // STA (Indirect,X)
		cycles = 6
		err = cpu.ExecuteSTAIndexedIndirect()

	case 0x84: // STY Zero Page
		cycles = 3
		err = cpu.ExecuteSTYZeroPage()

	case 0x85: // STA Zero Page
		cycles = 3
		err = cpu.ExecuteSTAZeroPage()

	case 0x86: // STX Zero Page
		cycles = 3
		err = cpu.ExecuteSTXZeroPage()

	case 0x88: // DEY
		cycles = 2
		err = cpu.ExecuteDEYImplied()

	case 0x8A: // TXA
		cycles = 2
		err = cpu.ExecuteTXAImplied()

	case 0x8C: // STY Absolute
		cycles = 4
		err = cpu.ExecuteSTYAbsolute()

	case 0x8D: // STA Absolute
		cycles = 4
		err = cpu.ExecuteSTAAbsolute()

	case 0x8E: // STX Absolute
		cycles = 4
		err = cpu.ExecuteSTXAbsolute()

	case 0x90: // BCC
		cycles = 2
		err = cpu.ExecuteBCCRelative()

	case 0x91: // STA (Indirect),Y
		cycles = 6
		err = cpu.ExecuteSTAIndirectIndexed()

	case 0x94: // STY Zero Page,X
		cycles = 4
		err = cpu.ExecuteSTYZeroPageX()

	case 0x95: // STA Zero Page,X
		cycles = 4
		err = cpu.ExecuteSTAZeroPageX()

	case 0x96: // STX Zero Page,Y
		cycles = 4
		err = cpu.ExecuteSTXZeroPageY()

	case 0x98: // TYA
		cycles = 2
		err = cpu.ExecuteTYAImplied()

	case 0x99: // STA Absolute,Y
		cycles = 5
		err = cpu.ExecuteSTAAbsoluteY()

	case 0x9A: // TXS
		cycles = 2
		err = cpu.ExecuteTXSImplied()

	case 0x9D: // STA Absolute,X
		cycles = 5
		err = cpu.ExecuteSTAAbsoluteX()

	// 0xA0 - 0xBF

	case 0xA0: // LDY Immediate
		cycles = 2
		err = cpu.ExecuteLDYImmediate()

	case 0xA1: // LDA (Indirect,X)
		cycles = 6
		err = cpu.ExecuteLDAIndexedIndirect()

	case 0xA2: // LDX Immediate
		cycles = 2
		err = cpu.ExecuteLDXImmediate()

	case 0xA4: // LDY Zero Page
		cycles = 3
		err = cpu.ExecuteLDYZeroPage()

	case 0xA5: // LDA Zero Page
		cycles = 3
		err = cpu.ExecuteLDAZeroPage()

	case 0xA6: // LDX Zero Page
		cycles = 3
		err = cpu.ExecuteLDXZeroPage()

	case 0xA8: // TAY
		cycles = 2
		err = cpu.ExecuteTAYImplied()

	case 0xA9: // LDA Immediate
		cycles = 2
		err = cpu.ExecuteLDAImmediate()

	case 0xAA: // TAX
		cycles = 2
		err = cpu.ExecuteTAXImplied()

	case 0xAC: // LDY Absolute
		cycles = 4
		err = cpu.ExecuteLDYAbsolute()

	case 0xAD: // LDA Absolute
		cycles = 4
		err = cpu.ExecuteLDAAbsolute()

	case 0xAE: // LDX Absolute
		cycles = 4
		err = cpu.ExecuteLDXAbsolute()

	case 0xB0: // BCS
		cycles = 2
		err = cpu.ExecuteBCSRelative()

	case 0xB1: // LDA (Indirect),Y
		cycles = 5
		err = cpu.ExecuteLDAIndirectIndexed()

	case 0xB4: // LDY Zero Page,X
		cycles = 4
		err = cpu.ExecuteLDYZeroPageX()

	case 0xB5: // LDA Zero Page,X
		cycles = 4
		err = cpu.ExecuteLDAZeroPageX()

	case 0xB6: // LDX Zero Page,Y
		cycles = 4
		err = cpu.ExecuteLDXZeroPageY()

	case 0xB8: // CLV
		cycles = 2
		err = cpu.ExecuteCLVImplied()

	case 0xB9: // LDA Absolute,Y
		cycles = 4
		err = cpu.ExecuteLDAAbsoluteY()

	case 0xBA: // TSX
		cycles = 2
		err = cpu.ExecuteTSXImplied()

	case 0xBC: // LDY Absolute,X
		cycles = 4
		err = cpu.ExecuteLDYAbsoluteX()

	case 0xBD: // LDA Absolute,X
		cycles = 4
		err = cpu.ExecuteLDAAbsoluteX()

	case 0xBE: // LDX Absolute,Y
		cycles = 4
		err = cpu.ExecuteLDXAbsoluteY()

	// 0xC0 - 0xDF

	case 0xC0: // CPY Immediate
		cycles = 2
		err = cpu.ExecuteCPYImmediate()

	case 0xC1: // CMP (Indirect,X)
		cycles = 6
		err = cpu.ExecuteCMPIndexedIndirect()

	case 0xC4: // CPY Zero Page
		cycles = 3
		err = cpu.ExecuteCPYZeroPage()

	case 0xC5: // CMP Zero Page
		cycles = 3
		err = cpu.ExecuteCMPZeroPage()

	case 0xC6: // DEC Zero Page
		cycles = 5
		err = cpu.ExecuteDECZeroPage()

	case 0xC8: // INY
		cycles = 2
		err = cpu.ExecuteINYImplied()

	case 0xC9: // CMP Immediate
		cycles = 2
		err = cpu.ExecuteCMPImmediate()

	case 0xCA: // DEX
		cycles = 2
		err = cpu.ExecuteDEXImplied()

	case 0xCC: // CPY Absolute
		cycles = 4
		err = cpu.ExecuteCPYAbsolute()

	case 0xCD: // CMP Absolute
		cycles = 4
		err = cpu.ExecuteCMPAbsolute()

	case 0xCE: // DEC Absolute
		cycles = 6
		err = cpu.ExecuteDECAbsolute()

	case 0xD0: // BNE
		cycles = 2
		err = cpu.ExecuteBNERelative()

	case 0xD1: // CMP (Indirect),Y
		cycles = 5
		err = cpu.ExecuteCMPIndirectIndexed()

	case 0xD5: // CMP Zero Page,X
		cycles = 4
		err = cpu.ExecuteCMPZeroPageX()

	case 0xD6: // DEC Zero Page,X
		cycles = 6
		err = cpu.ExecuteDECZeroPageX()

	case 0xD8: // CLD
		cycles = 2
		err = cpu.ExecuteCLDImplied()

	case 0xD9: // CMP Absolute,Y
		cycles = 4
		err = cpu.ExecuteCMPAbsoluteY()

	case 0xDD: // CMP Absolute,X
		cycles = 4
		err = cpu.ExecuteCMPAbsoluteX()

	case 0xDE: // DEC Absolute,X
		cycles = 7
		err = cpu.ExecuteDECAbsoluteX()

	// 0xE0 - 0xFF

	case 0xE0: // CPX Immediate
		cycles = 2
		err = cpu.ExecuteCPXImmediate()

	case 0xE1: // SBC (Indirect,X)
		cycles = 6
		err = cpu.ExecuteSBCIndexedIndirect()

	case 0xE4: // CPX Zero Page
		cycles = 3
		err = cpu.ExecuteCPXZeroPage()

	case 0xE5: // SBC Zero Page
		cycles = 3
		err = cpu.ExecuteSBCZeroPage()

	case 0xE6: // INC Zero Page
		cycles = 5
		err = cpu.ExecuteINCZeroPage()

	case 0xE8: // INX
		cycles = 2
		err = cpu.ExecuteINXImplied()

	case 0xE9: // SBC Immediate
		cycles = 2
		err = cpu.ExecuteSBCImmediate()

	case 0xEA: // NOP
		cycles = 2
		err = cpu.ExecuteNOPImplied()

	case 0xEC: // CPX Absolute
		cycles = 4
		err = cpu.ExecuteCPXAbsolute()

	case 0xED: // SBC Absolute
		cycles = 4
		err = cpu.ExecuteSBCAbsolute()

	case 0xEE: // INC Absolute
		cycles = 6
		err = cpu.ExecuteINCAbsolute()

	case 0xF0: // BEQ
		cycles = 2
		err = cpu.ExecuteBEQRelative()

	case 0xF1: // SBC (Indirect),Y
		cycles = 5
		err = cpu.ExecuteSBCIndirectIndexed()

	case 0xF5: // SBC Zero Page,X
		cycles = 4
		err = cpu.ExecuteSBCZeroPageX()

	case 0xF6: // INC Zero Page,X
		cycles = 6
		err = cpu.ExecuteINCZeroPageX()

	case 0xF8: // SED
		cycles = 2
		err = cpu.ExecuteSEDImplied()

	case 0xF9: // SBC Absolute,Y
		cycles = 4
		err = cpu.ExecuteSBCAbsoluteY()

	case 0xFD: // SBC Absolute,X
		cycles = 4
		err = cpu.ExecuteSBCAbsoluteX()

	case 0xFE: // INC Absolute,X
		cycles = 7
		err = cpu.ExecuteINCAbsoluteX()

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

	cycles += cpu.extraCycles

	cpu.Cycles += cycles
	cpu.Bus.Clock(cycles)

	return nil
}
