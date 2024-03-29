// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>

package elf64core

const (
	TextSection      = ".text"
	BssSection       = ".bss"
	TbssSection      = ".tbss"
	DataSection      = ".data"
	RodataSection    = ".rodata"
	IntrstackSection = ".intrstack"

	BootTextSection  = ".text.boot"
	BootDataSection  = ".data.boot"
	UkCtorTabSection = ".uk_ctortab"
	UkInitTabSection = ".uk_inittab"
	UnlikelySection  = ".unlikely"
	StartupSection   = ".startup"
)

var rx86_64Strings = map[uint32]string{
	0:  "R_X86_64_NONE",
	1:  "R_X86_64_64",
	2:  "R_X86_64_PC32",
	3:  "R_X86_64_GOT32",
	4:  "R_X86_64_PLT32",
	5:  "R_X86_64_COPY",
	6:  "R_X86_64_GLOB_DAT",
	7:  "R_X86_64_JMP_SLOT",
	8:  "R_X86_64_RELATIVE",
	9:  "R_X86_64_GOTPCREL",
	10: "R_X86_64_32",
	11: "R_X86_64_32S",
	12: "R_X86_64_16",
	13: "R_X86_64_PC16",
	14: "R_X86_64_8",
	15: "R_X86_64_PC8",
	16: "R_X86_64_DTPMOD64",
	17: "R_X86_64_DTPOFF64",
	18: "R_X86_64_TPOFF64",
	19: "R_X86_64_TLSGD",
	20: "R_X86_64_TLSLD",
	21: "R_X86_64_DTPOFF32",
	22: "R_X86_64_GOTTPOFF",
	23: "R_X86_64_TPOFF32",
	24: "R_X86_64_PC64",
	25: "R_X86_64_GOTOFF64",
	26: "R_X86_64_GOTPC32",
	27: "R_X86_64_GOT64",
	28: "R_X86_64_GOTPCREL64",
	29: "R_X86_64_GOTPC64",
	30: "R_X86_64_GOTPLT64",
	31: "R_X86_64_PLTOFF64",
	32: "R_X86_64_SIZE32",
	33: "R_X86_64_SIZE64",
	34: "R_X86_64_GOTPC32_TLSDESC",
	35: "R_X86_64_TLSDESC_CALL",
	36: "R_X86_64_TLSDESC",
	37: "R_X86_64_IRELATIVE",
	38: "R_X86_64_RELATIVE64",
	39: "R_X86_64_PC32_BND",
	40: "R_X86_64_PLT32_BND",
	41: "R_X86_64_GOTPCRELX",
	42: "R_X86_64_REX_GOTPCRELX",
}

var shtStrings = map[uint32]string{
	0:          "SHT_NULL",
	1:          "SHT_PROGBITS",
	2:          "SHT_SYMTAB",
	3:          "SHT_STRTAB",
	4:          "SHT_RELA",
	5:          "SHT_HASH",
	6:          "SHT_DYNAMIC",
	7:          "SHT_NOTE",
	8:          "SHT_NOBITS",
	9:          "SHT_REL",
	10:         "SHT_SHLIB",
	11:         "SHT_DYNSYM",
	14:         "SHT_INIT_ARRAY",
	15:         "SHT_FINI_ARRAY",
	16:         "SHT_PREINIT_ARRAY",
	17:         "SHT_GROUP",
	18:         "SHT_SYMTAB_SHNDX",
	0x60000000: "SHT_LOOS",
	0x6ffffff5: "SHT_GNU_ATTRIBUTES",
	0x6ffffff6: "SHT_GNU_HASH",
	0x6ffffff7: "SHT_GNU_LIBLIST",
	0x6ffffffd: "SHT_GNU_VERDEF",
	0x6ffffffe: "SHT_GNU_VERNEED",
	0x6fffffff: "SHT_GNU_VERSYM",
	0x70000000: "SHT_LOPROC",
	0x7fffffff: "SHT_HIPROC",
	0x80000000: "SHT_LOUSER",
	0xffffffff: "SHT_HIUSER",
}

var sttStrings = map[byte]string{
	0:  "NOTYPE",
	1:  "OBJECT",
	2:  "FUNC",
	3:  "SECTION",
	4:  "FILE",
	5:  "COMMON",
	6:  "TLS",
	10: "LOOS",
	12: "HIOS",
	13: "LOPROC",
	15: "HIPROC",
}

var ptStrings = map[uint32]string{
	0:          "PT_NULL",
	1:          "PT_LOAD",
	2:          "PT_DYNAMIC",
	3:          "PT_INTERP",
	4:          "PT_NOTE",
	5:          "PT_SHLIB",
	6:          "PT_PHDR",
	7:          "PT_TLS",
	0x60000000: "PT_LOOS",
	0x6fffffff: "PT_HIOS",
	0x70000000: "PT_LOPROC",
	0x7fffffff: "PT_HIPROC",
	0x6474e550: "GNU_EH_FRAME",
	0x6474e551: "GNU_STACK",
	0x6474e552: "GNU_RELRO",
}

var dtStrings = map[uint64]string{
	0:          "DT_NULL",
	1:          "DT_NEEDED",
	2:          "DT_PLTRELSZ",
	3:          "DT_PLTGOT",
	4:          "DT_HASH",
	5:          "DT_STRTAB",
	6:          "DT_SYMTAB",
	7:          "DT_RELA",
	8:          "DT_RELASZ",
	9:          "DT_RELAENT",
	10:         "DT_STRSZ",
	11:         "DT_SYMENT",
	12:         "DT_INIT",
	13:         "DT_FINI",
	14:         "DT_SONAME",
	15:         "DT_RPATH",
	16:         "DT_SYMBOLIC",
	17:         "DT_REL",
	18:         "DT_RELSZ",
	19:         "DT_RELENT",
	20:         "DT_PLTREL",
	21:         "DT_DEBUG",
	22:         "DT_TEXTREL",
	23:         "DT_JMPREL",
	24:         "DT_BIND_NOW",
	25:         "DT_INIT_ARRAY",
	26:         "DT_FINI_ARRAY",
	27:         "DT_INIT_ARRAYSZ",
	28:         "DT_FINI_ARRAYSZ",
	29:         "DT_RUNPATH",
	30:         "DT_FLAGS",
	32:         "DT_PREINIT_ARRAY",
	33:         "DT_PREINIT_ARRAYSZ",
	0x6000000d: "DT_LOOS",
	0x6ffff000: "DT_HIOS",
	0x6ffffff0: "DT_VERSYM",
	0x6ffffffe: "DT_VERNEED",
	0x6fffffff: "DT_VERNEEDNUM",
	0x70000000: "DT_LOPROC",
	0x7fffffff: "DT_HIPROC",
}
