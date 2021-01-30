// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64core

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"text/tabwriter"
	u "tools/srcs/common"
)

type NotesTables struct {
	name     string
	dataNote dataNote
}

type dataNote struct {
	name        string
	description string
	elf64note   ELF64Note
}

type ELF64Note struct {
	Namesz   uint32
	Descsz   uint32
	TypeNote uint32
}

func (elfFile *ELF64File) parseNote(index int) error {

	var notesTable NotesTables
	notesTable.name = elfFile.SectionsTable.DataSect[index].Name

	content, err := elfFile.GetSectionContent(uint16(index))

	if err != nil {
		return fmt.Errorf("failed reading note section: %s", err)
	}

	data := bytes.NewReader(content)
	err = binary.Read(data, elfFile.Endianness, &notesTable.dataNote.elf64note)
	if err != nil {
		return fmt.Errorf("failed reading elf64note: %s", err)
	}

	return nil
}

func (elfFile *ELF64File) DisplayNotes() {

	if len(elfFile.NotesTables) == 0 {
		u.PrintWarning("Notes are empty")
		return
	}

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Println("-----------------------------------------------------------------------")
	for _, t := range elfFile.NotesTables {
		_, _ = fmt.Fprintf(w, "\nDisplaying notes found in: %s\n", t.name)
		_, _ = fmt.Fprintln(w, " Owner\tData size\tDescription")
		_, _ = fmt.Fprintf(w, " %s\t0x%.6x\t%x\n", t.dataNote.name,
			t.dataNote.elf64note.Descsz, t.dataNote.description)
	}

	_ = w.Flush()
}
