package main

func (cpu *CPU) ExecuteSEI() {
	// Set Interrupt Disable flag (bit 2).
	cpu.Status |= 0x04
}

func (cpu *CPU) ExecuteCLD() {
	// Clear Decimal flag (bit 3).
	cpu.Status &^= 0x08
}

func (cpu *CPU) ExecuteLDA() error {
	value, err := cpu.FetchByte()
	if err != nil {
		return err
	}
	cpu.A = value

	//update zero flag(bit 1)
	if cpu.A == 0 {
		cpu.Status |= 0x02
	} else {
		cpu.Status &^= 0x02
	}

	//update negative flag(bit 7)
	if cpu.A&0x80 != 0 {
		cpu.Status |= 0x80
	} else {
		cpu.Status &^= 0x80
	}
	return nil
}

func (cpu *CPU) ExecuteSTAAbsolute() error {
	// fetch 16bit addr
	low, err := cpu.FetchByte()
	if err != nil {
		return err
	}
	high, err := cpu.FetchByte()
	if err != nil {
		return err
	}
	addr := uint16(low) | uint16(high)<<8

	// Store accumulator A into memory.
	return cpu.Bus.CpuWrite(addr, cpu.A)
}

func (cpu *CPU) ExecuteLDX() error {
	value, err := cpu.FetchByte()
	if err != nil {
		return err
	}
	cpu.X = value
	// Update Zero flag (bit 1).
	if cpu.X == 0 {
		cpu.Status |= 0x02
	} else {
		cpu.Status &^= 0x02
	}

	//update negative flag(bit 7)
	if cpu.X&0x80 != 0 {
		cpu.Status |= 0x80
	} else {
		cpu.Status &^= 0x80
	}
	return nil
}

func (cpu *CPU) ExecuteLDAAbsolute() error {
	low, err := cpu.FetchByte()
	if err != nil {
		return err
	}
	high, err := cpu.FetchByte()
	if err != nil {
		return err
	}
	addr := uint16(low) | uint16(high)<<8
	// read the value stored at the addr
	value, err := cpu.Bus.CpuRead(addr)
	if err != nil {
		return err
	}
	cpu.A = value

	//update the zero flag
	if cpu.A == 0 {
		cpu.Status |= 0x02
	} else {
		cpu.Status &^= 0x02
	}

	//update negative flag
	if cpu.A&0x80 != 0 {
		cpu.Status |= 0x80
	} else {
		cpu.Status &^= 0x80
	}
	return nil
}

func (cpu *CPU) ExecuteBPL() error {
	offset, err := cpu.FetchByte()
	if err != nil {
		return err
	}
	if cpu.Status&0x80 == 0 {
		relativeOffset := int16(int8(offset))
		cpu.PC = uint16(int32(cpu.PC) + int32(relativeOffset))
	}
	return nil
}

func (cpu *CPU) ExecuteDEX() {
	cpu.X--

	//update zero flag
	if cpu.X == 0 {
		cpu.Status |= 0x02
	} else {
		cpu.Status &^= 0x02

	}

	//update negative flag
	if cpu.X&0x80 != 0 {
		cpu.Status |= 0x80
	} else {
		cpu.Status &^= 0x80

	}

}

func (cpu *CPU) ExecuteBNE() error {
	offset, err := cpu.FetchByte()
	if err != nil {
		return err
	}

	// Branch when Zero flag is clear.
	if cpu.Status&0x02 == 0 {
		cpu.PC += uint16(int16(int8(offset)))
	}

	return nil
}
func (cpu *CPU) ExecuteTXS() {
	cpu.SP = cpu.X
}
