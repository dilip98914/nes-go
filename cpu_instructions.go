package main

const (
	flagCarry     byte = 1 << 0
	flagZero      byte = 1 << 1
	flagInterrupt byte = 1 << 2
	flagDecimal   byte = 1 << 3
	flagBreak     byte = 1 << 4
	flagUnused    byte = 1 << 5
	flagOverflow  byte = 1 << 6
	flagNegative  byte = 1 << 7
)

// -------------------------
// Shared CPU helpers
// -------------------------

func (cpu *CPU) setZN(value byte) {
	if value == 0 {
		cpu.Status |= flagZero
	} else {
		cpu.Status &^= flagZero
	}

	if value&0x80 != 0 {
		cpu.Status |= flagNegative
	} else {
		cpu.Status &^= flagNegative
	}

	cpu.Status |= flagUnused
}

func (cpu *CPU) fetchWord() (uint16, error) {
	low, err := cpu.FetchByte()
	if err != nil {
		return 0, err
	}

	high, err := cpu.FetchByte()
	if err != nil {
		return 0, err
	}

	return uint16(low) | uint16(high)<<8, nil
}

// Address modes:
// zp, zpx, zpy, abs, absx, absy, indx, indy
func (cpu *CPU) address(mode string) (uint16, bool, error) {
	switch mode {
	case "zp":
		v, err := cpu.FetchByte()
		return uint16(v), false, err

	case "zpx":
		v, err := cpu.FetchByte()
		return uint16(byte(v + cpu.X)), false, err

	case "zpy":
		v, err := cpu.FetchByte()
		return uint16(byte(v + cpu.Y)), false, err

	case "abs":
		v, err := cpu.fetchWord()
		return v, false, err

	case "absx":
		base, err := cpu.fetchWord()
		if err != nil {
			return 0, false, err
		}
		addr := base + uint16(cpu.X)
		return addr, base&0xFF00 != addr&0xFF00, nil

	case "absy":
		base, err := cpu.fetchWord()
		if err != nil {
			return 0, false, err
		}
		addr := base + uint16(cpu.Y)
		return addr, base&0xFF00 != addr&0xFF00, nil

	case "indx":
		zp, err := cpu.FetchByte()
		if err != nil {
			return 0, false, err
		}

		ptr := byte(zp + cpu.X)
		low, err := cpu.Bus.CpuRead(uint16(ptr))
		if err != nil {
			return 0, false, err
		}

		high, err := cpu.Bus.CpuRead(uint16(byte(ptr + 1)))
		if err != nil {
			return 0, false, err
		}

		return uint16(low) | uint16(high)<<8, false, nil

	case "indy":
		zp, err := cpu.FetchByte()
		if err != nil {
			return 0, false, err
		}

		low, err := cpu.Bus.CpuRead(uint16(zp))
		if err != nil {
			return 0, false, err
		}

		high, err := cpu.Bus.CpuRead(uint16(byte(zp + 1)))
		if err != nil {
			return 0, false, err
		}

		base := uint16(low) | uint16(high)<<8
		addr := base + uint16(cpu.Y)

		return addr, base&0xFF00 != addr&0xFF00, nil
	}

	return 0, false, nil
}

func (cpu *CPU) readOperand(mode string) (byte, bool, error) {
	if mode == "imm" {
		v, err := cpu.FetchByte()
		return v, false, err
	}

	addr, crossed, err := cpu.address(mode)
	if err != nil {
		return 0, false, err
	}

	v, err := cpu.Bus.CpuRead(addr)
	return v, crossed, err
}

func (cpu *CPU) writeOperand(mode string, value byte) error {
	addr, _, err := cpu.address(mode)
	if err != nil {
		return err
	}
	return cpu.Bus.CpuWrite(addr, value)
}

func (cpu *CPU) push(value byte) error {
	err := cpu.Bus.CpuWrite(0x0100|uint16(cpu.SP), value)
	if err != nil {
		return err
	}
	cpu.SP--
	return nil
}

func (cpu *CPU) pop() (byte, error) {
	cpu.SP++
	return cpu.Bus.CpuRead(0x0100 | uint16(cpu.SP))
}

func (cpu *CPU) compare(register, value byte) {
	result := uint16(register) - uint16(value)

	if register >= value {
		cpu.Status |= flagCarry
	} else {
		cpu.Status &^= flagCarry
	}

	cpu.setZN(byte(result))
}

func (cpu *CPU) adc(value byte) {
	// Decimal arithmetic is disabled on the NES 2A03.
	carry := uint16(0)
	if cpu.Status&flagCarry != 0 {
		carry = 1
	}

	sum := uint16(cpu.A) + uint16(value) + carry
	result := byte(sum)

	if sum > 0xFF {
		cpu.Status |= flagCarry
	} else {
		cpu.Status &^= flagCarry
	}

	if (^(cpu.A ^ value) & (cpu.A ^ result) & 0x80) != 0 {
		cpu.Status |= flagOverflow
	} else {
		cpu.Status &^= flagOverflow
	}

	cpu.A = result
	cpu.setZN(cpu.A)
}

func (cpu *CPU) branch(condition bool) error {
	offset, err := cpu.FetchByte()
	if err != nil {
		return err
	}

	if condition {
		oldPC := cpu.PC
		cpu.PC = uint16(int32(cpu.PC) + int32(int8(offset)))

		cpu.extraCycles++

		if oldPC&0xFF00 != cpu.PC&0xFF00 {
			cpu.extraCycles++
		}
	}

	return nil
}

func (cpu *CPU) modifyMemory(mode, operation string) error {
	addr, _, err := cpu.address(mode)
	if err != nil {
		return err
	}

	value, err := cpu.Bus.CpuRead(addr)
	if err != nil {
		return err
	}

	var result byte

	switch operation {
	case "ASL":
		if value&0x80 != 0 {
			cpu.Status |= flagCarry
		} else {
			cpu.Status &^= flagCarry
		}
		result = value << 1

	case "LSR":
		if value&1 != 0 {
			cpu.Status |= flagCarry
		} else {
			cpu.Status &^= flagCarry
		}
		result = value >> 1

	case "ROL":
		carry := byte(0)
		if cpu.Status&flagCarry != 0 {
			carry = 1
		}
		if value&0x80 != 0 {
			cpu.Status |= flagCarry
		} else {
			cpu.Status &^= flagCarry
		}
		result = (value << 1) | carry

	case "ROR":
		carry := byte(0)
		if cpu.Status&flagCarry != 0 {
			carry = 0x80
		}
		if value&1 != 0 {
			cpu.Status |= flagCarry
		} else {
			cpu.Status &^= flagCarry
		}
		result = (value >> 1) | carry

	case "INC":
		result = value + 1

	case "DEC":
		result = value - 1
	}

	cpu.setZN(result)
	return cpu.Bus.CpuWrite(addr, result)
}

func (cpu *CPU) modifyAccumulator(operation string) {
	value := cpu.A
	var result byte

	switch operation {
	case "ASL":
		if value&0x80 != 0 {
			cpu.Status |= flagCarry
		} else {
			cpu.Status &^= flagCarry
		}
		result = value << 1

	case "LSR":
		if value&1 != 0 {
			cpu.Status |= flagCarry
		} else {
			cpu.Status &^= flagCarry
		}
		result = value >> 1

	case "ROL":
		carry := byte(0)
		if cpu.Status&flagCarry != 0 {
			carry = 1
		}
		if value&0x80 != 0 {
			cpu.Status |= flagCarry
		} else {
			cpu.Status &^= flagCarry
		}
		result = (value << 1) | carry

	case "ROR":
		carry := byte(0)
		if cpu.Status&flagCarry != 0 {
			carry = 0x80
		}
		if value&1 != 0 {
			cpu.Status |= flagCarry
		} else {
			cpu.Status &^= flagCarry
		}
		result = (value >> 1) | carry
	}

	cpu.A = result
	cpu.setZN(result)
}

// -------------------------
// LDA
// -------------------------

func (cpu *CPU) ExecuteLDAImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.A = v
	cpu.setZN(cpu.A)
	return nil
}

func (cpu *CPU) ExecuteLDAZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.A = v
	cpu.setZN(cpu.A)
	return nil
}

func (cpu *CPU) ExecuteLDAZeroPageX() error {
	v, _, err := cpu.readOperand("zpx")
	if err != nil {
		return err
	}
	cpu.A = v
	cpu.setZN(cpu.A)
	return nil
}

func (cpu *CPU) ExecuteLDAAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.A = v
	cpu.setZN(cpu.A)
	return nil
}

func (cpu *CPU) ExecuteLDAAbsoluteX() error {
	v, crossed, err := cpu.readOperand("absx")
	if err != nil {
		return err
	}
	cpu.A = v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

func (cpu *CPU) ExecuteLDAAbsoluteY() error {
	v, crossed, err := cpu.readOperand("absy")
	if err != nil {
		return err
	}
	cpu.A = v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

func (cpu *CPU) ExecuteLDAIndexedIndirect() error {
	v, _, err := cpu.readOperand("indx")
	if err != nil {
		return err
	}
	cpu.A = v
	cpu.setZN(cpu.A)
	return nil
}

func (cpu *CPU) ExecuteLDAIndirectIndexed() error {
	v, crossed, err := cpu.readOperand("indy")
	if err != nil {
		return err
	}
	cpu.A = v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

// -------------------------
// LDX
// -------------------------

func (cpu *CPU) ExecuteLDXImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.X = v
	cpu.setZN(cpu.X)
	return nil
}

func (cpu *CPU) ExecuteLDXZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.X = v
	cpu.setZN(cpu.X)
	return nil
}

func (cpu *CPU) ExecuteLDXZeroPageY() error {
	v, _, err := cpu.readOperand("zpy")
	if err != nil {
		return err
	}
	cpu.X = v
	cpu.setZN(cpu.X)
	return nil
}

func (cpu *CPU) ExecuteLDXAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.X = v
	cpu.setZN(cpu.X)
	return nil
}

func (cpu *CPU) ExecuteLDXAbsoluteY() error {
	v, crossed, err := cpu.readOperand("absy")
	if err != nil {
		return err
	}
	cpu.X = v
	cpu.setZN(cpu.X)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

// -------------------------
// LDY
// -------------------------

func (cpu *CPU) ExecuteLDYImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.Y = v
	cpu.setZN(cpu.Y)
	return nil
}

func (cpu *CPU) ExecuteLDYZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.Y = v
	cpu.setZN(cpu.Y)
	return nil
}

func (cpu *CPU) ExecuteLDYZeroPageX() error {
	v, _, err := cpu.readOperand("zpx")
	if err != nil {
		return err
	}
	cpu.Y = v
	cpu.setZN(cpu.Y)
	return nil
}

func (cpu *CPU) ExecuteLDYAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.Y = v
	cpu.setZN(cpu.Y)
	return nil
}

func (cpu *CPU) ExecuteLDYAbsoluteX() error {
	v, crossed, err := cpu.readOperand("absx")
	if err != nil {
		return err
	}
	cpu.Y = v
	cpu.setZN(cpu.Y)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

// -------------------------
// STA / STX / STY
// -------------------------

func (cpu *CPU) ExecuteSTAIndexedIndirect() error {
	return cpu.writeOperand("indx", cpu.A)
}
func (cpu *CPU) ExecuteSTAZeroPage() error {
	return cpu.writeOperand("zp", cpu.A)
}
func (cpu *CPU) ExecuteSTAZeroPageX() error {
	return cpu.writeOperand("zpx", cpu.A)
}
func (cpu *CPU) ExecuteSTAAbsolute() error {
	return cpu.writeOperand("abs", cpu.A)
}
func (cpu *CPU) ExecuteSTAAbsoluteX() error {
	return cpu.writeOperand("absx", cpu.A)
}
func (cpu *CPU) ExecuteSTAAbsoluteY() error {
	return cpu.writeOperand("absy", cpu.A)
}
func (cpu *CPU) ExecuteSTAIndirectIndexed() error {
	return cpu.writeOperand("indy", cpu.A)
}

func (cpu *CPU) ExecuteSTXZeroPage() error {
	return cpu.writeOperand("zp", cpu.X)
}
func (cpu *CPU) ExecuteSTXZeroPageY() error {
	return cpu.writeOperand("zpy", cpu.X)
}
func (cpu *CPU) ExecuteSTXAbsolute() error {
	return cpu.writeOperand("abs", cpu.X)
}

func (cpu *CPU) ExecuteSTYZeroPage() error {
	return cpu.writeOperand("zp", cpu.Y)
}
func (cpu *CPU) ExecuteSTYZeroPageX() error {
	return cpu.writeOperand("zpx", cpu.Y)
}
func (cpu *CPU) ExecuteSTYAbsolute() error {
	return cpu.writeOperand("abs", cpu.Y)
}

// -------------------------
// ORA / AND / EOR
// -------------------------

func (cpu *CPU) ExecuteORAIndexedIndirect() error {
	v, _, err := cpu.readOperand("indx")
	if err != nil {
		return err
	}
	cpu.A |= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteORAZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.A |= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteORAImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.A |= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteORAAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.A |= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteORAIndirectIndexed() error {
	v, crossed, err := cpu.readOperand("indy")
	if err != nil {
		return err
	}
	cpu.A |= v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteORAZeroPageX() error {
	v, _, err := cpu.readOperand("zpx")
	if err != nil {
		return err
	}
	cpu.A |= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteORAAbsoluteY() error {
	v, crossed, err := cpu.readOperand("absy")
	if err != nil {
		return err
	}
	cpu.A |= v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteORAAbsoluteX() error {
	v, crossed, err := cpu.readOperand("absx")
	if err != nil {
		return err
	}
	cpu.A |= v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

func (cpu *CPU) ExecuteANDIndexedIndirect() error {
	v, _, err := cpu.readOperand("indx")
	if err != nil {
		return err
	}
	cpu.A &= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteANDZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.A &= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteANDImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.A &= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteANDAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.A &= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteANDIndirectIndexed() error {
	v, crossed, err := cpu.readOperand("indy")
	if err != nil {
		return err
	}
	cpu.A &= v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteANDZeroPageX() error {
	v, _, err := cpu.readOperand("zpx")
	if err != nil {
		return err
	}
	cpu.A &= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteANDAbsoluteY() error {
	v, crossed, err := cpu.readOperand("absy")
	if err != nil {
		return err
	}
	cpu.A &= v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteANDAbsoluteX() error {
	v, crossed, err := cpu.readOperand("absx")
	if err != nil {
		return err
	}
	cpu.A &= v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

func (cpu *CPU) ExecuteEORIndexedIndirect() error {
	v, _, err := cpu.readOperand("indx")
	if err != nil {
		return err
	}
	cpu.A ^= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteEORZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.A ^= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteEORImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.A ^= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteEORAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.A ^= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteEORIndirectIndexed() error {
	v, crossed, err := cpu.readOperand("indy")
	if err != nil {
		return err
	}
	cpu.A ^= v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteEORZeroPageX() error {
	v, _, err := cpu.readOperand("zpx")
	if err != nil {
		return err
	}
	cpu.A ^= v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteEORAbsoluteY() error {
	v, crossed, err := cpu.readOperand("absy")
	if err != nil {
		return err
	}
	cpu.A ^= v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteEORAbsoluteX() error {
	v, crossed, err := cpu.readOperand("absx")
	if err != nil {
		return err
	}
	cpu.A ^= v
	cpu.setZN(cpu.A)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

// -------------------------
// ADC / SBC
// -------------------------

func (cpu *CPU) ExecuteADCIndexedIndirect() error {
	v, _, err := cpu.readOperand("indx")
	if err != nil {
		return err
	}
	cpu.adc(v)
	return nil
}
func (cpu *CPU) ExecuteADCZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.adc(v)
	return nil
}
func (cpu *CPU) ExecuteADCImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.adc(v)
	return nil
}
func (cpu *CPU) ExecuteADCAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.adc(v)
	return nil
}
func (cpu *CPU) ExecuteADCIndirectIndexed() error {
	v, crossed, err := cpu.readOperand("indy")
	if err != nil {
		return err
	}
	cpu.adc(v)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteADCZeroPageX() error {
	v, _, err := cpu.readOperand("zpx")
	if err != nil {
		return err
	}
	cpu.adc(v)
	return nil
}
func (cpu *CPU) ExecuteADCAbsoluteY() error {
	v, crossed, err := cpu.readOperand("absy")
	if err != nil {
		return err
	}
	cpu.adc(v)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteADCAbsoluteX() error {
	v, crossed, err := cpu.readOperand("absx")
	if err != nil {
		return err
	}
	cpu.adc(v)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

func (cpu *CPU) ExecuteSBCIndexedIndirect() error {
	v, _, err := cpu.readOperand("indx")
	if err != nil {
		return err
	}
	cpu.adc(^v)
	return nil
}
func (cpu *CPU) ExecuteSBCZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.adc(^v)
	return nil
}
func (cpu *CPU) ExecuteSBCImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.adc(^v)
	return nil
}
func (cpu *CPU) ExecuteSBCAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.adc(^v)
	return nil
}
func (cpu *CPU) ExecuteSBCIndirectIndexed() error {
	v, crossed, err := cpu.readOperand("indy")
	if err != nil {
		return err
	}
	cpu.adc(^v)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteSBCZeroPageX() error {
	v, _, err := cpu.readOperand("zpx")
	if err != nil {
		return err
	}
	cpu.adc(^v)
	return nil
}
func (cpu *CPU) ExecuteSBCAbsoluteY() error {
	v, crossed, err := cpu.readOperand("absy")
	if err != nil {
		return err
	}
	cpu.adc(^v)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteSBCAbsoluteX() error {
	v, crossed, err := cpu.readOperand("absx")
	if err != nil {
		return err
	}
	cpu.adc(^v)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

// -------------------------
// CMP / CPX / CPY
// -------------------------

func (cpu *CPU) ExecuteCMPIndexedIndirect() error {
	v, _, err := cpu.readOperand("indx")
	if err != nil {
		return err
	}
	cpu.compare(cpu.A, v)
	return nil
}
func (cpu *CPU) ExecuteCMPZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.compare(cpu.A, v)
	return nil
}
func (cpu *CPU) ExecuteCMPImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.compare(cpu.A, v)
	return nil
}
func (cpu *CPU) ExecuteCMPAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.compare(cpu.A, v)
	return nil
}
func (cpu *CPU) ExecuteCMPIndirectIndexed() error {
	v, crossed, err := cpu.readOperand("indy")
	if err != nil {
		return err
	}
	cpu.compare(cpu.A, v)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteCMPZeroPageX() error {
	v, _, err := cpu.readOperand("zpx")
	if err != nil {
		return err
	}
	cpu.compare(cpu.A, v)
	return nil
}
func (cpu *CPU) ExecuteCMPAbsoluteY() error {
	v, crossed, err := cpu.readOperand("absy")
	if err != nil {
		return err
	}
	cpu.compare(cpu.A, v)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}
func (cpu *CPU) ExecuteCMPAbsoluteX() error {
	v, crossed, err := cpu.readOperand("absx")
	if err != nil {
		return err
	}
	cpu.compare(cpu.A, v)
	if crossed {
		cpu.extraCycles++
	}
	return nil
}

func (cpu *CPU) ExecuteCPYImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.compare(cpu.Y, v)
	return nil
}
func (cpu *CPU) ExecuteCPYZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.compare(cpu.Y, v)
	return nil
}
func (cpu *CPU) ExecuteCPYAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.compare(cpu.Y, v)
	return nil
}

func (cpu *CPU) ExecuteCPXImmediate() error {
	v, _, err := cpu.readOperand("imm")
	if err != nil {
		return err
	}
	cpu.compare(cpu.X, v)
	return nil
}
func (cpu *CPU) ExecuteCPXZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.compare(cpu.X, v)
	return nil
}
func (cpu *CPU) ExecuteCPXAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.compare(cpu.X, v)
	return nil
}

// -------------------------
// BIT
// -------------------------

func (cpu *CPU) ExecuteBITZeroPage() error {
	v, _, err := cpu.readOperand("zp")
	if err != nil {
		return err
	}
	cpu.bit(v)
	return nil
}
func (cpu *CPU) ExecuteBITAbsolute() error {
	v, _, err := cpu.readOperand("abs")
	if err != nil {
		return err
	}
	cpu.bit(v)
	return nil
}

func (cpu *CPU) bit(value byte) {
	if cpu.A&value == 0 {
		cpu.Status |= flagZero
	} else {
		cpu.Status &^= flagZero
	}
	cpu.Status = (cpu.Status &^ (flagNegative | flagOverflow)) |
		(value & (flagNegative | flagOverflow))
}

// -------------------------
// ASL / LSR / ROL / ROR
// -------------------------

func (cpu *CPU) ExecuteASLAccumulator() error {
	cpu.modifyAccumulator("ASL")
	return nil
}
func (cpu *CPU) ExecuteASLZeroPage() error {
	return cpu.modifyMemory("zp", "ASL")
}
func (cpu *CPU) ExecuteASLZeroPageX() error {
	return cpu.modifyMemory("zpx", "ASL")
}
func (cpu *CPU) ExecuteASLAbsolute() error {
	return cpu.modifyMemory("abs", "ASL")
}
func (cpu *CPU) ExecuteASLAbsoluteX() error {
	return cpu.modifyMemory("absx", "ASL")
}

func (cpu *CPU) ExecuteLSRAccumulator() error {
	cpu.modifyAccumulator("LSR")
	return nil
}
func (cpu *CPU) ExecuteLSRZeroPage() error {
	return cpu.modifyMemory("zp", "LSR")
}
func (cpu *CPU) ExecuteLSRZeroPageX() error {
	return cpu.modifyMemory("zpx", "LSR")
}
func (cpu *CPU) ExecuteLSRAbsolute() error {
	return cpu.modifyMemory("abs", "LSR")
}
func (cpu *CPU) ExecuteLSRAbsoluteX() error {
	return cpu.modifyMemory("absx", "LSR")
}

func (cpu *CPU) ExecuteROLAccumulator() error {
	cpu.modifyAccumulator("ROL")
	return nil
}
func (cpu *CPU) ExecuteROLZeroPage() error {
	return cpu.modifyMemory("zp", "ROL")
}
func (cpu *CPU) ExecuteROLZeroPageX() error {
	return cpu.modifyMemory("zpx", "ROL")
}
func (cpu *CPU) ExecuteROLAbsolute() error {
	return cpu.modifyMemory("abs", "ROL")
}
func (cpu *CPU) ExecuteROLAbsoluteX() error {
	return cpu.modifyMemory("absx", "ROL")
}

func (cpu *CPU) ExecuteRORAccumulator() error {
	cpu.modifyAccumulator("ROR")
	return nil
}
func (cpu *CPU) ExecuteRORZeroPage() error {
	return cpu.modifyMemory("zp", "ROR")
}
func (cpu *CPU) ExecuteRORZeroPageX() error {
	return cpu.modifyMemory("zpx", "ROR")
}
func (cpu *CPU) ExecuteRORAbsolute() error {
	return cpu.modifyMemory("abs", "ROR")
}
func (cpu *CPU) ExecuteRORAbsoluteX() error {
	return cpu.modifyMemory("absx", "ROR")
}

// -------------------------
// INC / DEC
// -------------------------

func (cpu *CPU) ExecuteINCZeroPage() error {
	return cpu.modifyMemory("zp", "INC")
}
func (cpu *CPU) ExecuteINCZeroPageX() error {
	return cpu.modifyMemory("zpx", "INC")
}
func (cpu *CPU) ExecuteINCAbsolute() error {
	return cpu.modifyMemory("abs", "INC")
}
func (cpu *CPU) ExecuteINCAbsoluteX() error {
	return cpu.modifyMemory("absx", "INC")
}

func (cpu *CPU) ExecuteDECZeroPage() error {
	return cpu.modifyMemory("zp", "DEC")
}
func (cpu *CPU) ExecuteDECZeroPageX() error {
	return cpu.modifyMemory("zpx", "DEC")
}
func (cpu *CPU) ExecuteDECAbsolute() error {
	return cpu.modifyMemory("abs", "DEC")
}
func (cpu *CPU) ExecuteDECAbsoluteX() error {
	return cpu.modifyMemory("absx", "DEC")
}

// -------------------------
// INX / INY / DEX / DEY
// -------------------------

func (cpu *CPU) ExecuteINXImplied() error {
	cpu.X++
	cpu.setZN(cpu.X)
	return nil
}
func (cpu *CPU) ExecuteINYImplied() error {
	cpu.Y++
	cpu.setZN(cpu.Y)
	return nil
}
func (cpu *CPU) ExecuteDEXImplied() error {
	cpu.X--
	cpu.setZN(cpu.X)
	return nil
}
func (cpu *CPU) ExecuteDEYImplied() error {
	cpu.Y--
	cpu.setZN(cpu.Y)
	return nil
}

// -------------------------
// Transfer instructions
// -------------------------

func (cpu *CPU) ExecuteTAXImplied() error {
	cpu.X = cpu.A
	cpu.setZN(cpu.X)
	return nil
}
func (cpu *CPU) ExecuteTAYImplied() error {
	cpu.Y = cpu.A
	cpu.setZN(cpu.Y)
	return nil
}
func (cpu *CPU) ExecuteTXAImplied() error {
	cpu.A = cpu.X
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteTYAImplied() error {
	cpu.A = cpu.Y
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecuteTSXImplied() error {
	cpu.X = cpu.SP
	cpu.setZN(cpu.X)
	return nil
}
func (cpu *CPU) ExecuteTXSImplied() error {
	cpu.SP = cpu.X
	return nil
}

// -------------------------
// Flag instructions
// -------------------------

func (cpu *CPU) ExecuteCLCImplied() error {
	cpu.Status &^= flagCarry
	return nil
}
func (cpu *CPU) ExecuteSECImplied() error {
	cpu.Status |= flagCarry
	return nil
}
func (cpu *CPU) ExecuteCLIImplied() error {
	cpu.Status &^= flagInterrupt
	return nil
}
func (cpu *CPU) ExecuteSEIImplied() error {
	cpu.Status |= flagInterrupt
	return nil
}
func (cpu *CPU) ExecuteCLVImplied() error {
	cpu.Status &^= flagOverflow
	return nil
}
func (cpu *CPU) ExecuteCLDImplied() error {
	cpu.Status &^= flagDecimal
	return nil
}
func (cpu *CPU) ExecuteSEDImplied() error {
	cpu.Status |= flagDecimal
	return nil
}

// -------------------------
// Branch instructions
// -------------------------

func (cpu *CPU) ExecuteBPLRelative() error {
	return cpu.branch(cpu.Status&flagNegative == 0)
}
func (cpu *CPU) ExecuteBMIRelative() error {
	return cpu.branch(cpu.Status&flagNegative != 0)
}
func (cpu *CPU) ExecuteBVCRelative() error {
	return cpu.branch(cpu.Status&flagOverflow == 0)
}
func (cpu *CPU) ExecuteBVSRelative() error {
	return cpu.branch(cpu.Status&flagOverflow != 0)
}
func (cpu *CPU) ExecuteBCCRelative() error {
	return cpu.branch(cpu.Status&flagCarry == 0)
}
func (cpu *CPU) ExecuteBCSRelative() error {
	return cpu.branch(cpu.Status&flagCarry != 0)
}
func (cpu *CPU) ExecuteBNERelative() error {
	return cpu.branch(cpu.Status&flagZero == 0)
}
func (cpu *CPU) ExecuteBEQRelative() error {
	return cpu.branch(cpu.Status&flagZero != 0)
}

// -------------------------
// Stack instructions
// -------------------------

func (cpu *CPU) ExecutePHAImplied() error {
	return cpu.push(cpu.A)
}
func (cpu *CPU) ExecutePHPImplied() error {
	return cpu.push(cpu.Status | flagBreak | flagUnused)
}
func (cpu *CPU) ExecutePLAImplied() error {
	v, err := cpu.pop()
	if err != nil {
		return err
	}
	cpu.A = v
	cpu.setZN(cpu.A)
	return nil
}
func (cpu *CPU) ExecutePLPImplied() error {
	v, err := cpu.pop()
	if err != nil {
		return err
	}
	cpu.Status = (v &^ flagBreak) | flagUnused
	return nil
}

// -------------------------
// JMP / JSR / RTS / RTI / BRK
// -------------------------

func (cpu *CPU) ExecuteJMPAbsolute() error {
	addr, err := cpu.fetchWord()
	if err != nil {
		return err
	}
	cpu.PC = addr
	return nil
}

func (cpu *CPU) ExecuteJMPIndirect() error {
	ptr, err := cpu.fetchWord()
	if err != nil {
		return err
	}

	low, err := cpu.Bus.CpuRead(ptr)
	if err != nil {
		return err
	}

	// Reproduce the original 6502 JMP indirect page-boundary bug.
	highAddr := (ptr & 0xFF00) | uint16(byte(ptr+1))
	high, err := cpu.Bus.CpuRead(highAddr)
	if err != nil {
		return err
	}

	cpu.PC = uint16(low) | uint16(high)<<8
	return nil
}

func (cpu *CPU) ExecuteJSRAbsolute() error {
	target, err := cpu.fetchWord()
	if err != nil {
		return err
	}

	returnAddress := cpu.PC - 1

	if err := cpu.push(byte(returnAddress >> 8)); err != nil {
		return err
	}
	if err := cpu.push(byte(returnAddress)); err != nil {
		return err
	}

	cpu.PC = target
	return nil
}

func (cpu *CPU) ExecuteRTSImplied() error {
	low, err := cpu.pop()
	if err != nil {
		return err
	}

	high, err := cpu.pop()
	if err != nil {
		return err
	}

	cpu.PC = (uint16(low) | uint16(high)<<8) + 1
	return nil
}

func (cpu *CPU) ExecuteRTIImplied() error {
	status, err := cpu.pop()
	if err != nil {
		return err
	}

	cpu.Status = (status &^ flagBreak) | flagUnused

	low, err := cpu.pop()
	if err != nil {
		return err
	}

	high, err := cpu.pop()
	if err != nil {
		return err
	}

	cpu.PC = uint16(low) | uint16(high)<<8
	return nil
}

func (cpu *CPU) ExecuteBRKImplied() error {
	// BRK has a padding byte and pushes the address after that byte.
	cpu.PC++

	if err := cpu.push(byte(cpu.PC >> 8)); err != nil {
		return err
	}
	if err := cpu.push(byte(cpu.PC)); err != nil {
		return err
	}
	if err := cpu.push(cpu.Status | flagBreak | flagUnused); err != nil {
		return err
	}

	cpu.Status |= flagInterrupt

	low, err := cpu.Bus.CpuRead(0xFFFE)
	if err != nil {
		return err
	}

	high, err := cpu.Bus.CpuRead(0xFFFF)
	if err != nil {
		return err
	}

	cpu.PC = uint16(low) | uint16(high)<<8
	return nil
}

// -------------------------
// NOP
// -------------------------

func (cpu *CPU) ExecuteNOPImplied() error {
	return nil
}
