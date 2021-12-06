// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64analyser

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const PageSize = 0x1000

type ElfFileSegment struct {
	Filename string
	NbPages  int
	Pages    []*ElfPage
}

type ElfPage struct {
	number           int
	startAddress     uint64
	contentByteArray []byte
	hash             string
	libName          string
	sectionName      string
	noNullValues     int
	cap              int
}

func (p *ElfPage) pageContentToString() string {

	var builder strings.Builder
	for i, entry := range p.contentByteArray {

		if i > 0 && i%4 == 0 {
			builder.WriteString(" ")
		}

		if i > 0 && i%16 == 0 {
			builder.WriteString("\n")
		}

		_, _ = builder.WriteString(fmt.Sprintf("%02x", entry))

	}
	_, _ = builder.WriteString("")
	return builder.String()
}

func (p *ElfPage) displayPageContent(mw io.Writer) {

	/*
		hexStartAddr, err := strconv.ParseInt(p.startAddress, 16, 64);
		if err != nil {
			panic(err)
		}
	*/
	for i, entry := range p.contentByteArray {

		if i > 0 && i%4 == 0 {
			_, _ = fmt.Fprintf(mw, " ")
		}

		if i > 0 && i%16 == 0 {
			_, _ = fmt.Fprintf(mw, "\n")
		}

		_, _ = fmt.Fprintf(mw, "%02x", entry)

	}
	_, _ = fmt.Fprintln(mw, "")
}

func (p *ElfPage) displayPageContentShort(mw io.Writer) {

	entryLine := 0
	for i, entry := range p.contentByteArray {

		if entry > 0 {
			_, _ = fmt.Fprintf(mw, "[%d] %02x ", i, entry)
			if entryLine > 0 && entryLine%16 == 0 {
				_, _ = fmt.Fprintf(mw, "\n")
			}
			entryLine++
		}
	}
	_, _ = fmt.Fprintln(mw, "")
}

func SavePagesToFile(pageTables []*ElfPage, filename string, shortView bool) error {

	mw := io.MultiWriter(os.Stdout)
	if len(filename) > 0 {
		file, err := os.Create(filename)

		if err != nil {
			return err
		}
		mw = io.MultiWriter(file)
	}

	for i, p := range pageTables {
		_, _ = fmt.Fprintln(mw, "----------------------------------------------------")
		_, _ = fmt.Fprintf(mw, "Page: %d\n", i+1)
		_, _ = fmt.Fprintf(mw, "LibName: %s\n", p.libName)
		_, _ = fmt.Fprintf(mw, "Section: %s\n", p.sectionName)
		_, _ = fmt.Fprintf(mw, "StartAddr: %x (%d)\n", p.startAddress, p.startAddress)
		_, _ = fmt.Fprintf(mw, "Non-Null value: %d\n", p.noNullValues)
		_, _ = fmt.Fprintf(mw, "Hash: %s\n", p.hash)

		if shortView {
			p.displayPageContentShort(mw)
		} else {
			p.displayPageContent(mw)
		}
		_, _ = fmt.Fprintln(mw, "----------------------------------------------------")
	}
	return nil
}
