package main

import (
	"fmt"
	"os"
)

func main() {
	// Load the ROM.
	rom, err := os.ReadFile("./mario_rom.nes")
	if err != nil {
		panic(err)
	}

	// Parse the cartridge.
	cartridge, err := NewCartridge(rom)
	if err != nil {
		panic(err)
	}

	// initialize mmc3 mapper & bus
	mapper := NewMapper4(cartridge)

	bus := &Bus{
		Mapper: mapper,
	}

	cpu := &CPU{
		Bus: bus,
	}

	if err := cpu.Reset(); err != nil {
		panic(err)
	}

	// fmt.Println("===== NES CPU =====")
	// fmt.Println("Game: Super Mario Bros. 3")
	// // fmt.Println("Mapper:", cartridge.Mapper)
	// fmt.Printf("Starting PC: 0x%04X\n", cpu.PC)

	// if err := cpu.Step(); err != nil {
	// 	fmt.Println("CPU stopped:", err)
	// 	return
	// }
	// fmt.Println("Instruction executed successfully!")
	// fmt.Printf("PC:     0x%04X\n", cpu.PC)
	// fmt.Printf("A:      0x%02X\n", cpu.A)
	// fmt.Printf("Status: 0x%02X\n", cpu.Status)

	const maxInstructions = 100
	executed := 0

	for executed < maxInstructions {
		if err := cpu.Step(); err != nil {
			fmt.Println("CPU stopped: ", err)
			break
		}
		executed++
	}
	fmt.Println("\n===== CPU EXECUTION RESULT =====")
	fmt.Println("Instructions executed:", executed)
	fmt.Printf("PC:     0x%04X\n", cpu.PC)
	fmt.Printf("A:      0x%02X\n", cpu.A)
	fmt.Printf("X:      0x%02X\n", cpu.X)
	fmt.Printf("Y:      0x%02X\n", cpu.Y)
	fmt.Printf("SP:     0x%02X\n", cpu.SP)
	fmt.Println("CPU cycles:", cpu.Cycles)
	fmt.Printf("Status: 0x%02X\n", cpu.Status)

}
