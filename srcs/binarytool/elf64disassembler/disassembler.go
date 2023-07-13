package elf64disassembler

import (
	"fmt"
	"github.com/knightsc/gapstone"
	"log"
	"strconv"
	"strings"
	"tools/srcs/binarytool/elf64core"
)

//disassembler.Disass_section(elfFile, s)

func hex2int(hexStr string) uint64 {
	// remove 0x suffix if found in the input string
	cleaned := strings.Replace(hexStr, "0x", "", -1)

	// base 16 for hexadecimal
	result, _ := strconv.ParseUint(cleaned, 16, 64)
	return uint64(result)
}

func Disass_section(elfFile *elf64core.ELF64File, sections *elf64core.DataSections) {

	sect_data := elfFile.Raw[sections.Elf64section.FileOffset : sections.Elf64section.FileOffset+sections.Elf64section.Size]

	engine, err := gapstone.New(
		gapstone.CS_ARCH_X86,
		gapstone.CS_MODE_64,
	)
	if err != nil {
		log.Fatalf("Disassembly error: %v", err)
	}

	maj, min := engine.Version()
	log.Printf("Hello Capstone! Version: %v.%v\n", maj, min)

	insns, err := engine.Disasm(
		sect_data,                            // code buffer
		sections.Elf64section.VirtualAddress, // starting address
		0,
	)
	if err != nil {
		log.Fatalf("Disassembly error: %v", err)
	}

	log.Printf("Disasm: %d %d\n", len(sect_data), len(insns))
	for _, insn := range insns {

		callStr := ""
		if insn.Mnemonic == "call" {
			fmt.Printf("0x%x -> ", insn.Bytes)
			fmt.Printf("0x%x:\t[%s]\t\t{%s}%s\n", insn.Address, insn.Mnemonic, insn.OpStr, callStr)
			if val, ok := elfFile.MapFctAddrName[hex2int(insn.OpStr)]; ok {
				println("\t" + val)
			}

		}

	}

}
