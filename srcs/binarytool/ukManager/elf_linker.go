package ukManager

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"tools/srcs/binarytool/elf64analyser"
	"tools/srcs/binarytool/elf64core"
)

type LinkerInfo struct {
	ldsString  string
	rodataAddr uint64
	dataAddr   uint64
	bssAddr    uint64
}

const (
	endtext_location   = "<END_TEXT_REPLACE_LOCATION>"
	rodata_location    = "<RODATA_REPLACE_LOCATION>"
	data_location      = "<DATA_REPLACE_LOCATION>"
	erodata_location   = "<ERODATA_REPLACE_LOCATION>"
	edata_location     = "<EDATA_REPLACE_LOCATION>"
	bss_location       = "<BSS_REPLACE_LOCATION>"
	tbss_location      = "<TBSS_REPLACE_LOCATION>"
	intrstack_location = "<INTRSTACK_REPLACE_LOCATION>"
	ukinit_location    = "<UK_INIT_REPLACE_LOCATION>"

	inner_rodata = "<INNER_RODATA>"
	inner_data   = "<INNER_DATA>"
	inner_bss    = "<INNER_BSS>"
)

func readLdsContent(filename string) string {

	strBuilder := strings.Builder{}
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// optionally, resize scanner's capacity for lines over 64K, see next example
	for scanner.Scan() {
		strBuilder.WriteString(scanner.Text() + "\n")
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return strBuilder.String()
}

func processLdsFile(locationCnt uint64, maxValSection map[string]uint64) LinkerInfo {

	type sectionLoc struct {
		sec string
		loc string
	}

	linkerInfo := LinkerInfo{}

	// Use an array to preserver order
	arrSection := []sectionLoc{
		{sec: elf64core.RodataSection, loc: rodata_location},
		{sec: elf64core.DataSection, loc: data_location},
		{sec: elf64core.BssSection, loc: bss_location},
		{sec: elf64core.TbssSection, loc: tbss_location}}

	ldsString := readLdsContent("lds/common.ld")
	// Update end of text
	ldsString = strings.Replace(ldsString, endtext_location, fmt.Sprintf(". = 0x%x;", locationCnt), -1)
	// Update ukinit
	ldsString = strings.Replace(ldsString, ukinit_location, fmt.Sprintf(". = 0x%x;", locationCnt+0x40), -1)

	locationCnt += elf64analyser.PageSize
	for _, sect := range arrSection {
		if sect.sec == elf64core.RodataSection {
			linkerInfo.rodataAddr = locationCnt
		} else if sect.sec == elf64core.DataSection {
			// Update erodata just before data
			ldsString = strings.Replace(ldsString, erodata_location, fmt.Sprintf(". = 0x%x;", locationCnt), -1)
			locationCnt += elf64analyser.PageSize
			linkerInfo.dataAddr = locationCnt
		} else if sect.sec == elf64core.BssSection {
			// Update edata just before bss
			ldsString = strings.Replace(ldsString, edata_location, fmt.Sprintf(". = 0x%x;", locationCnt), -1)
			locationCnt += elf64analyser.PageSize
			linkerInfo.bssAddr = locationCnt
		}
		// Update rodata, data, bss, tbss
		ldsString = strings.Replace(ldsString, sect.loc, fmt.Sprintf(". = 0x%x;", locationCnt), -1)
		locationCnt += maxValSection[sect.sec]
		locationCnt = roundAddr(locationCnt, elf64analyser.PageSize)
	}
	// Update intrstack
	linkerInfo.ldsString = strings.Replace(ldsString, intrstack_location, fmt.Sprintf(". = 0x%x;", locationCnt), -1)

	return linkerInfo
}
