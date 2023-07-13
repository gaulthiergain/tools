package ukManager

import (
	"fmt"
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

func getLdsContent() string {
	return "SECTIONS\n{\n . = 0x100000;\n _text = .;\n .text :\n {\n  KEEP (*(.data.boot))\n  *(.text.boot)\n  *(.text)\n  *(.text.*) /* uncomment it to dissagregate functions */\n }\n\n<END_TEXT_REPLACE_LOCATION>\n _etext = .;\n . = ALIGN((1 << 12)); __eh_frame_start = .; .eh_frame : { *(.eh_frame) *(.eh_frame.*) } __eh_frame_end = .; __eh_frame_hdr_start = .; .eh_frame_hdr : { *(.eh_frame_hdr) *(.eh_frame_hdr.*) } __eh_frame_hdr_end = .;\n . = ALIGN((1 << 12)); uk_ctortab_start = .; .uk_ctortab : { KEEP(*(SORT_BY_NAME(.uk_ctortab[0-9]))) } uk_ctortab_end = .;\n<UK_INIT_REPLACE_LOCATION> uk_inittab_start = .; .uk_inittab : { KEEP(*(SORT_BY_NAME(.uk_inittab[1-6][0-9]))) } uk_inittab_end = .;\n\n<RODATA_REPLACE_LOCATION>\n . = ALIGN((1 << 12));\n  _rodata = .;\n .rodata :\n {\n<INNER_RODATA>\n   *(.rodata)\n   *(.rodata.*)\n }\n\n<ERODATA_REPLACE_LOCATION>\n _erodata = .;\n . = ALIGN(0x8);\n _ctors = .;\n .preinit_array : {\n  PROVIDE_HIDDEN (__preinit_array_start = .);\n  KEEP (*(.preinit_array))\n  PROVIDE_HIDDEN (__preinit_array_end = .);\n }\n . = ALIGN(0x8);\n .init_array : {\n  PROVIDE_HIDDEN (__init_array_start = .);\n  KEEP (*(SORT_BY_INIT_PRIORITY(.init_array.*) SORT_BY_INIT_PRIORITY(.ctors.*)))\n  KEEP (*(.init_array .ctors))\n  PROVIDE_HIDDEN (__init_array_end = .);\n }\n _ectors = .;\n . = ALIGN(0x8); _tls_start = .; .tdata : { *(.tdata) *(.tdata.*) *(.gnu.linkonce.td.*) } _etdata = .;\n\n<DATA_REPLACE_LOCATION>\n . = ALIGN((1 << 12));\n _data = .;\n .data :\n {\n<INNER_DATA>\n   *(.data)\n   *(.data.*)\n }\n\n<EDATA_REPLACE_LOCATION>\n _edata = .;\n . = ALIGN((1 << 12));\n\n<BSS_REPLACE_LOCATION>\n __bss_start = .;\n .bss :\n {\n<INNER_BSS>\n   *(.bss)\n   *(.bss.*)\n  *(COMMON)\n  . = ALIGN((1 << 12));\n }\n\n<TBSS_REPLACE_LOCATION>\n .tbss : { *(.tbss) *(.tbss.*) *(.gnu.linkonce.tb.*) . = ALIGN(0x8); } _tls_end = . + SIZEOF(.tbss);\n\n<INTRSTACK_REPLACE_LOCATION>\n .intrstack :\n {\n  *(.intrstack)\n  . = ALIGN((1 << 12));\n }\n _end = .;\n .comment 0 : { *(.comment) }\n .debug 0 : { *(.debug) } .line 0 : { *(.line) } .debug_srcinfo 0 : { *(.debug_srcinfo) } .debug_sfnames 0 : { *(.debug_sfnames) } .debug_aranges 0 : { *(.debug_aranges) } .debug_pubnames 0 : { *(.debug_pubnames) } .debug_info 0 : { *(.debug_info .gnu.linkonce.wi.*) } .debug_abbrev 0 : { *(.debug_abbrev) } .debug_line 0 : { *(.debug_line .debug_line.* .debug_line_end ) } .debug_frame 0 : { *(.debug_frame) } .debug_str 0 : { *(.debug_str) } .debug_loc 0 : { *(.debug_loc) } .debug_macinfo 0 : { *(.debug_macinfo) } .debug_weaknames 0 : { *(.debug_weaknames) } .debug_funcnames 0 : { *(.debug_funcnames) } .debug_typenames 0 : { *(.debug_typenames) } .debug_varnames 0 : { *(.debug_varnames) } .debug_pubtypes 0 : { *(.debug_pubtypes) } .debug_ranges 0 : { *(.debug_ranges) } .debug_macro 0 : { *(.debug_macro) } .gnu.attributes 0 : { KEEP (*(.gnu.attributes)) }\n /DISCARD/ : { *(.note.gnu.build-id) }\n}"
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

	ldsString := getLdsContent()
	// Update end of text
	ldsString = strings.Replace(ldsString, endtext_location, fmt.Sprintf(". = 0x%x;", locationCnt), -1)
	// Update ukinit
	ldsString = strings.Replace(ldsString, ukinit_location, fmt.Sprintf(". = 0x%x;", locationCnt+0x60), -1)

	locationCnt += elf64analyser.PageSize
	for _, sect := range arrSection {
		if sect.sec == elf64core.RodataSection {
			linkerInfo.rodataAddr = locationCnt
		} else if sect.sec == elf64core.DataSection {
			// Update erodata just before data
			ldsString = strings.Replace(ldsString, erodata_location, fmt.Sprintf(". = 0x%x;", locationCnt-elf64analyser.PageSize), -1)
			linkerInfo.dataAddr = locationCnt
		} else if sect.sec == elf64core.BssSection {
			// Update edata just before bss
			ldsString = strings.Replace(ldsString, edata_location, fmt.Sprintf(". = 0x%x;", locationCnt-elf64analyser.PageSize), -1)
			linkerInfo.bssAddr = locationCnt
		}
		// Update rodata, data, bss, tbss
		ldsString = strings.Replace(ldsString, sect.loc, fmt.Sprintf(". = 0x%x;", locationCnt), -1)
		locationCnt += maxValSection[sect.sec]
	}
	// Update intrstack
	linkerInfo.ldsString = strings.Replace(ldsString, intrstack_location, fmt.Sprintf(". = 0x%x;", locationCnt), -1)

	return linkerInfo
}
