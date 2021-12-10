package ukManager

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"tools/srcs/binarytool/elf64analyser"
	"tools/srcs/binarytool/elf64core"
	u "tools/srcs/common"
)

const ukbootMain = "libukboot_main.ld.o"

type Manager struct {
	Unikernels      []*Unikernel
	MicroLibs       map[string]*MicroLib
	SortedMicroLibs []*MicroLib //Used for the display
}

type MicroLib struct {
	name        string
	startAddr   uint64
	size        uint64
	instance    int
	sectionSize *SectionMicroLibs
}

type SectionMicroLibs struct {
	rodataSize uint64
	dataSize   uint64
	bssSize    uint64

	rodataAddr uint64
	dataAddr   uint64
	bssAddr    uint64
}

func (manager *Manager) ComputeAlignment(unikernel Unikernel) {

	for _, libs := range unikernel.Analyser.ElfLibs {

		// Ignore ukbootMain to place it to specific position
		if libs.Name == ukbootMain {
			continue
		}

		// Add microlib to a global map per instance
		if val, ok := manager.MicroLibs[libs.Name]; ok {
			if manager.MicroLibs[libs.Name].size != libs.Size {
				//u.PrintWarning(fmt.Sprintf("Different size between %s (0x%x) (0x%x)", libs.Name, libs.Size, val.size))
				if val.size < libs.Size {
					u.PrintWarning(fmt.Sprintf("Bigger size found %s (0x%x) > (0x%x)", libs.Name, libs.Size, val.size))
					manager.MicroLibs[libs.Name].size = libs.Size
					manager.MicroLibs[libs.Name].sectionSize.rodataSize = libs.RodataSize
					manager.MicroLibs[libs.Name].sectionSize.dataSize = libs.DataSize
					manager.MicroLibs[libs.Name].sectionSize.bssSize = libs.BssSize
				}
			}
			manager.MicroLibs[libs.Name].instance += 1
		} else {
			mlib := &MicroLib{
				name:      libs.Name,
				startAddr: libs.StartAddr,
				size:      libs.Size,
				instance:  1,
				sectionSize: &SectionMicroLibs{
					rodataSize: libs.RodataSize,
					dataSize:   libs.DataSize,
					bssSize:    libs.BssSize,
				},
			}
			manager.MicroLibs[libs.Name] = mlib
		}
	}
}

func (manager *Manager) sortMicroLibs() {
	type kv struct {
		key      string
		instance int
		addr     uint64
	}
	i := 0
	var kvSlice = make([]kv, len(manager.MicroLibs))
	for k, v := range manager.MicroLibs {
		kvSlice[i] = kv{k, v.instance, v.startAddr}
		i++
	}

	sort.Slice(kvSlice, func(i, j int) bool {
		if kvSlice[i].instance != kvSlice[j].instance {
			return kvSlice[i].instance > kvSlice[j].instance
		}
		return kvSlice[i].addr < kvSlice[j].addr
	})

	manager.SortedMicroLibs = make([]*MicroLib, len(manager.MicroLibs))
	for k, lib := range kvSlice {
		manager.SortedMicroLibs[k] = manager.MicroLibs[lib.key]
	}
}

func (manager *Manager) DisplayMicroLibs() {
	if manager.SortedMicroLibs == nil {
		manager.sortMicroLibs()
	}

	for i, lib := range manager.SortedMicroLibs {
		fmt.Printf("%d %s: %x - %x - %d\n", i, lib.name, lib.startAddr, lib.size, lib.instance)
	}
}

func (manager *Manager) PerformAlignement() {

	var startValue uint64 = 0x106000
	var locationCnt = startValue
	commonMicroLibs := make([]*MicroLib, 0)

	// Sort micro-libs per instance (usage) and per addresses (descending order)
	if manager.SortedMicroLibs == nil {
		manager.sortMicroLibs()
	}

	// Update micro-libs mapping globally and per unikernels
	for i, lib := range manager.SortedMicroLibs {
		if lib.instance == len(manager.Unikernels) {
			// micro-libs common to all instances
			lib.startAddr = locationCnt
			commonMicroLibs = append(commonMicroLibs, lib)
			locationCnt += lib.size
		} else if lib.instance > 1 {
			// micro-libs common to particular instances
			if manager.SortedMicroLibs[i-1].instance == len(manager.Unikernels) {
				// Add ukboot main after all common micro-libs
				locationCnt = roundAddr(locationCnt, elf64analyser.PageSize)
				ukbootMainLib := &MicroLib{
					name:        ukbootMain,
					startAddr:   locationCnt,
					size:        0,
					instance:    len(manager.Unikernels),
					sectionSize: &SectionMicroLibs{},
				}
				commonMicroLibs = append(commonMicroLibs, ukbootMainLib)
			}

			locationCnt = roundAddr(locationCnt, elf64analyser.PageSize)
			for _, uk := range manager.Unikernels {
				if uk.alignedLibs == nil {
					uk.InitAlignment()
				}
				uk.AddAlignedMicroLibs(locationCnt, lib)
			}
			locationCnt += lib.size
		} else if lib.instance == 1 {
			// micro-libs to only single instance
			for _, uk := range manager.Unikernels {
				uk.AddSingleMicroLibs(roundAddr(locationCnt, elf64analyser.PageSize), lib)
			}
		}
	}

	maxValSection := make(map[string]uint64, 3)
	sections := []string{elf64core.RodataSection, elf64core.DataSection, elf64core.TbssSection, elf64core.BssSection}

	// Find max location counter value through unikernels
	for _, uk := range manager.Unikernels {

		// Update the locationCnt by finding the maximum one from unikernel (the biggest size)
		uk.alignedLibs.AllCommonMicroLibs = commonMicroLibs
		if locationCnt < uk.alignedLibs.startValueUk {
			locationCnt = uk.alignedLibs.startValueUk
		}

		// Analyse sections to find the biggest section sizes (rodata, data, tbss, bss)
		for _, section := range sections {
			findMaxValue(section, uk, maxValSection)
		}
	}

	// Update the common lds file with new location counter
	locationCnt = roundAddr(locationCnt, elf64analyser.PageSize)
	linkerInfo := processLdsFile(locationCnt, maxValSection)

	// Use temporary variable to keep linkerInfo unchanged
	linkerInfoGlobal := &LinkerInfo{
		ldsString:  "",
		rodataAddr: linkerInfo.rodataAddr,
		dataAddr:   linkerInfo.dataAddr,
		bssAddr:    linkerInfo.bssAddr,
	}
	// Redo a pass on micro-libs to align inner rodata, data and bss
	for _, lib := range manager.SortedMicroLibs {

		if lib.instance > 1 {
			// Update inner rodata location counter
			lib.sectionSize.rodataAddr = linkerInfoGlobal.rodataAddr
			linkerInfoGlobal.rodataAddr += roundAddr(lib.sectionSize.rodataSize, 32)

			// Update inner dataAddr location counter
			lib.sectionSize.dataAddr = linkerInfoGlobal.dataAddr
			linkerInfoGlobal.dataAddr += roundAddr(lib.sectionSize.dataSize, 32)

			// Update inner bssAddr location counter
			lib.sectionSize.bssAddr = linkerInfoGlobal.bssAddr
			linkerInfoGlobal.bssAddr += roundAddr(lib.sectionSize.bssSize, 32)
		}
	}

	// Update per unikernel
	for _, uk := range manager.Unikernels {
		uk.writeTextAlignment(startValue)

		// todo remove and replace per uk.buildpath
		lib := strings.Replace(strings.Split(uk.BuildPath, "/")[5], "lib-", "", -1)
		dst := "/Users/gaulthiergain/Desktop/memory_dedup/gcc/lds/common_optimized_app_dce_size/link64_" + lib + ".lds"

		uk.writeLdsToFile(dst, linkerInfo)
	}

}

func findMaxValue(section string, uk *Unikernel, maxValSection map[string]uint64) {
	index := uk.ElfFile.IndexSections[section]
	size := uk.ElfFile.SectionsTable.DataSect[index].Elf64section.Size
	if val, ok := maxValSection[section]; ok {
		if val < size {
			maxValSection[section] = size
		}
	} else {
		maxValSection[section] = size
	}
}

func roundAddr(v uint64, round uint64) uint64 {
	return uint64(math.Round(float64(v/round))*float64(round)) + round
}
