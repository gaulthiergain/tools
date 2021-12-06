// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64analyser

import (
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"tools/srcs/binarytool/elf64core"
	u "tools/srcs/common"
)

type ComparisonElf struct {
	GroupFileSegment []*ElfFileSegment
	dictSamePage     map[string]int
	dictFile         map[string]map[string]int
}

func (comparison *ComparisonElf) processDictName(filename, hash string) {
	m := comparison.dictFile[hash]
	if _, ok := m[filename]; !ok {
		m[filename] = 1
	} else {
		t := m[filename]
		t += 1
		m[filename] = t
	}
	comparison.dictFile[hash] = m
}

func (comparison *ComparisonElf) ComparePageTables() {

	comparison.dictSamePage = make(map[string]int)
	comparison.dictFile = make(map[string]map[string]int)

	for _, file := range comparison.GroupFileSegment {

		for _, p := range file.Pages {
			if _, ok := comparison.dictSamePage[p.hash]; !ok {
				comparison.dictSamePage[p.hash] = 1
				comparison.dictFile[p.hash] = make(map[string]int)
			} else {
				comparison.dictSamePage[p.hash] += 1
			}
			comparison.processDictName(file.Filename, p.hash)
		}

	}
}

func (comparison *ComparisonElf) DisplayComparison() {

	fmt.Println("\n\nHash comparison:")
	for key, value := range comparison.dictFile {
		fmt.Println(key, ";", value)
	}

	fmt.Println("---------------------------")
	fmt.Println("\n\nStats:")
	countSamePage := 0
	singlePage := 0
	for _, value := range comparison.dictSamePage {
		if value > 1 {
			countSamePage += value
		} else {
			singlePage++
		}
	}

	totalNbPages := 0
	for i, _ := range comparison.GroupFileSegment {
		totalNbPages += comparison.GroupFileSegment[i].NbPages
	}
	ratio := (float64(countSamePage) / float64(totalNbPages)) * 100

	fmt.Printf("- Total Nb of pages: %d\n", totalNbPages)
	fmt.Printf("- Nb page(s) sharing: %d\n", countSamePage)
	fmt.Printf("- Page alone: %d\n", singlePage)
	fmt.Printf("- Ratio: %f\n", ratio)
}

func filterOutput(text1, text2 []string) string {
	header := "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Diff Pages</title></head><body style=\"font-family:Menlo\">"
	footer := "</body></html>"

	maxArray := text1
	minArray := text2
	if len(text1) < len(text2) {
		maxArray = text2
		minArray = text1
	}

	var builder strings.Builder
	for i := 0; i < len(maxArray); i++ {
		builder.WriteString("<span>" + maxArray[i] + "</span><br>")
		if i < len(minArray)-1 && maxArray[i] != minArray[i] {
			builder.WriteString("<p><del style=\"background:#ffe6e6;\">" + minArray[i] + "</del><br>")
			builder.WriteString("<ins style=\"background:#e6ffe6;\">" + maxArray[i] + "</ins></p>")
		}
	}

	return header + builder.String() + footer
}

func (comparison *ComparisonElf) DiffComparison(path string) error {

	if len(comparison.GroupFileSegment) != 2 {
		return errors.New("multi-comparison (more than 2) is still not supported")
	}

	minPage := comparison.GroupFileSegment[0].Pages
	for _, file := range comparison.GroupFileSegment {
		if len(minPage) > len(file.Pages) {
			minPage = file.Pages
		}
	}

	println(len(comparison.GroupFileSegment[0].Pages))
	println(len(comparison.GroupFileSegment[1].Pages))

	for i := 0; i < len(minPage); i++ {

		page1 := comparison.GroupFileSegment[0].Pages[i]
		page2 := comparison.GroupFileSegment[1].Pages[i]
		if page1.hash != page2.hash {

			text1String := comparison.GroupFileSegment[0].Pages[i].pageContentToString()
			text2String := comparison.GroupFileSegment[1].Pages[i].pageContentToString()
			text1 := strings.Split(text1String, "\n")
			text2 := strings.Split(text2String, "\n")
			str := filterOutput(text1, text2)

			file, err := os.Create(path + "page_" + strconv.Itoa(page1.number) + "_diff.html")
			if err != nil {
				return err
			}
			if _, err := file.WriteString(str); err != nil {
				return err
			}
			file.Close()

		}
	}

	return nil
}

func (analyser *ElfAnalyser) DisplaySectionInfo(elfFile *elf64core.ELF64File, info []string) {

	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 10, 8, 0, '\t', 0)
	_, _ = fmt.Fprintln(w, "Name\tAddress\tOffset\tSize")
	for _, sectionName := range info {
		if indexSection, ok := elfFile.IndexSections[sectionName]; ok {
			section := elfFile.SectionsTable.DataSect[indexSection].Elf64section

			_, _ = fmt.Fprintf(w, "- %s\t0x%.6x\t0x%.6x\t%d\n",
				sectionName, section.VirtualAddress, section.FileOffset,
				section.Size)

		} else {
			u.PrintWarning("Wrong section name " + sectionName)
		}
	}
	_ = w.Flush()
}

func (analyser *ElfAnalyser) FindSectionByAddress(elfFile *elf64core.ELF64File, addresses []string) {
	if len(elfFile.SectionsTable.DataSect) == 0 {
		u.PrintWarning("Sections table is empty")
		return
	}
	for _, addr := range addresses {
		hexStr := strings.Replace(addr, "0x", "", -1)
		intAddr, err := strconv.ParseUint(hexStr, 16, 64)
		if err != nil {
			u.PrintWarning(fmt.Sprintf("Error %s: Cannot convert %s to integer. Skip.", err, addr))
		} else {
			found := false
			for _, s := range elfFile.SectionsTable.DataSect {
				if s.Elf64section.VirtualAddress <= intAddr && intAddr < s.Elf64section.VirtualAddress+s.Elf64section.Size {
					fmt.Printf("Address %s is in section %s\n", addr, s.Name)
					found = true
				}
			}
			if !found {
				u.PrintWarning(fmt.Sprintf("Cannot find a section for address: %s", addr))
			}
		}
	}
}

func (analyser *ElfAnalyser) DisplayStatSize(elfFile *elf64core.ELF64File) {
	if len(elfFile.SectionsTable.DataSect) == 0 {
		u.PrintWarning("Sections table is empty")
		return
	}
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, 0, '\t', 0)

	var totalSizeText uint64
	var totalSizeElf uint64
	_, _ = fmt.Fprintf(w, "-----------------------------------------------------------------------\n")
	_, _ = fmt.Fprintf(w, "Name\tVirtual Size (Bytes/Hex) \t#pages\tInfos:\n")

	// Sort by addresses
	dataSec := elfFile.SectionsTable.DataSect
	sort.Slice(dataSec, func(i, j int) bool {
		return dataSec[i].Elf64section.VirtualAddress < dataSec[j].Elf64section.VirtualAddress
	})

	for i, s := range elfFile.SectionsTable.DataSect { //&uint64(elf.SHF_WRITE)
		if s.Elf64section.VirtualAddress > 0 {

			var size uint64
			var currNext = ""
			if strings.Contains(s.Name, elf64core.IntrstackSection) || strings.Contains(s.Name, elf64core.TbssSection) {
				size = s.Elf64section.Size
				totalSizeElf += size

			} else {
				size = dataSec[i+1].Elf64section.VirtualAddress -
					dataSec[i].Elf64section.VirtualAddress
				totalSizeElf += size
				currNext = fmt.Sprintf("0x%x -> 0x%x : (%s)-> (%s)", dataSec[i].Elf64section.VirtualAddress,
					dataSec[i+1].Elf64section.VirtualAddress, dataSec[i].Name, dataSec[i+1].Name)
			}

			if strings.Contains(s.Name, elf64core.TextSection) && strings.Contains(dataSec[i+1].Name, elf64core.TextSection) {
				totalSizeText += size
			} else if strings.Contains(s.Name, elf64core.TextSection) {
				// Main application code
				size = s.Elf64section.Size
				totalSizeText += size
			}
			_, _ = fmt.Fprintf(w, "%s\t%d (0x%x)\t%.2f\t%s\n", s.Name, size, size, float32(size)/float32(PageSize), currNext)

		}
	}
	_, _ = fmt.Fprintf(w, "----------------------\t----------------------\t------\t----------------------------\n")
	_, _ = fmt.Fprintf(w, "Total Size:\n")
	_, _ = fmt.Fprintf(w, "Section .text:\t%d (0x%x)\n", totalSizeText, totalSizeText)
	_, _ = fmt.Fprintf(w, "All sections:\t%d (0x%x)\n", totalSizeElf, totalSizeElf)
	_, _ = fmt.Fprintf(w, "#Pages (.text):\t%d\n", roundPage(float64(totalSizeText)/float64(PageSize)))
	_, _ = fmt.Fprintf(w, "#Pages (all sections):\t%d\n", roundPage(float64(totalSizeElf)/float64(PageSize)))
	_ = w.Flush()
}

func roundPage(x float64) uint64 {
	return uint64(math.Round(x + 0.5))
}
