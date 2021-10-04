// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package dependtool

import (
	"encoding/hex"
	"fmt"
	"github.com/knightsc/gapstone"
	"strings"
	u "tools/srcs/common"
)


func runTrapper(programName string, data *u.Data) error{
	// Get the list of system calls
	syscalls_map := initSystemCalls()
	syscalls := make([]string, len(syscalls_map))
	for k, v := range syscalls_map {
		syscalls[v] = k
	}

	elfF, err := getElf(programName)
	if err != nil {
		return err
	}

	engine, err := gapstone.New(
		gapstone.CS_ARCH_X86,
		gapstone.CS_MODE_64,
	)
	if err != nil {
		return err
	}

	maj, min := engine.Version()
	fmt.Printf("Capstone version: %v.%v\n", maj, min)

	textSection := elfF.Section(".text")
	dataSection, err := textSection.Data()
	if err != nil {
		return err
	}
	insns, err := engine.Disasm(
		dataSection,             // code buffer
		textSection.Addr, // starting address
		0,                // insns to disassemble, 0 for all
	)
	if err != nil {
		return err
	}

	fmt.Printf("Disasm: %d %d\n", len(dataSection), len(insns))

	for i, insn := range insns {
		b := insn.Bytes
		strBuilder := strings.Builder{}
		strBuilder.WriteString(" 0x" + hex.EncodeToString(b))

		if b[0] == 0x0f && b[1] == 0x05 {
			// Direct syscall SYSCALL
			//log.Printf("0x%x:\t%s\t\t%s - %s\n", insn.Address, insn.Mnemonic, insn.OpStr, strBuilder.String())
			backtrack64(i, insns, syscalls, data)
		} else if b[0] == 0x0f && b[1] == 0x34 {
			// Direct syscall SYSENTER
			//log.Printf("0x%x:\t%s\t\t%s - %s\n", insn.Address, insn.Mnemonic, insn.OpStr, strBuilder.String())
			backtrack64(i, insns, syscalls, data)
		} else if b[0] == 0xcd && b[1] == 0x80 {
			// Direct syscall int 0x80
			//log.Printf("0x%x:\t%s\t\t%s - %s\n", insn.Address, insn.Mnemonic, insn.OpStr, strBuilder.String())
			backtracki386(i, insns, syscalls, data)
		}

	}

	if err := elfF.Close(); err != nil {
		return err
	}

	return nil
}

func backtrack64(index int, insns []gapstone.Instruction, syscalls []string, data *u.Data) {
	//log.Printf("current i: %d\n", index)

	for i := index - 1; i >= 0; i-- {
		insn := insns[i]
		b := insn.Bytes

		//log.Printf("\t->0x%x:\t%s\t\t%s\n", insn.Address, insn.Mnemonic, insn.OpStr)
		// MOV in EAX
		if b[0] == 0xb8 {
			data.StaticData.SystemCalls[syscalls[int(b[1])]] = int(b[1])
			//fmt.Printf("\"%s\": %d,\n", syscalls[int(b[1])], b[1])
			break
		}

		// Another syscall is called, break
		if b[0] == 0xcd && b[1] == 0x80 {
			break
		}
	}
}

func backtracki386(index int, insns []gapstone.Instruction, syscalls []string, data *u.Data) {
	//log.Printf("current i: %d\n", index)

	for i := index - 1; i >= 0; i-- {
		insn := insns[i]
		b := insn.Bytes

		//log.Printf("\t->0x%x:\t%s\t\t%s\n", insn.Address, insn.Mnemonic, insn.OpStr)
		// MOV in EAX
		if b[0] == 0xb8 {
			data.StaticData.SystemCalls[syscalls[int(b[1])]] = int(b[1])
			//fmt.Printf("\"%s\": %d,\n", syscalls[int(b[1])], b[1])
			break
		}

		// Another syscall is called, break
		if b[0] == 0xcd && b[1] == 0x80 {
			break
		}
	}
}
