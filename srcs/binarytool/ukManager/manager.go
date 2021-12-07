package ukManager

import (
	"fmt"
	"math"
	"sort"
	"tools/srcs/binarytool/elf64analyser"
	u "tools/srcs/common"
)

type Manager struct {
	Unikernels      []Unikernel
	MicroLibs       map[string]*MicroLib
	SortedMicroLibs []*MicroLib //Used for the display
}

type MicroLib struct {
	name        string
	startAddr   uint64
	size        uint64
	instance    int
	sectionSize SectionMicroLibs
}

type SectionMicroLibs struct {
	rodataSize uint64
	dataSize   uint64
	BssSize    uint64
}

func (manager *Manager) ComputeAlignment(unikernel Unikernel) {

	for _, libs := range unikernel.Analyser.ElfLibs {
		if val, ok := manager.MicroLibs[libs.Name]; ok {
			if manager.MicroLibs[libs.Name].size != libs.Size {
				//u.PrintWarning(fmt.Sprintf("Different size between %s (0x%x) (0x%x)", libs.Name, libs.Size, val.size))
				if val.size < libs.Size {
					u.PrintWarning(fmt.Sprintf("Bigger size found %s (0x%x) > (0x%x)", libs.Name, libs.Size, val.size))
					manager.MicroLibs[libs.Name].size = libs.Size
				}
			}
			manager.MicroLibs[libs.Name].instance += 1
		} else {
			mlib := &MicroLib{
				name:      libs.Name,
				startAddr: libs.StartAddr,
				size:      libs.Size,
				instance:  1,
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
	for i, lib := range kvSlice {
		manager.SortedMicroLibs[i] = manager.MicroLibs[lib.key]
	}
}

func (manager *Manager) DisplayMicroLibs() {
	manager.sortMicroLibs()
	for i, lib := range manager.SortedMicroLibs {

		if lib.instance < len(manager.Unikernels) {
			fmt.Printf("%d %s: %x - %x - %d\n", i, lib.name, lib.startAddr, lib.size, lib.instance)
		}
	}
}

func (manager *Manager) PerformAlignement() {
	var startValue uint64 = 0x106000
	//manager.Unikernels[0].Analyser.ElfLibs

	manager.sortMicroLibs()
	commonMicroLibs := make([]*MicroLib, 0)
	for _, lib := range manager.SortedMicroLibs {
		if lib.instance == len(manager.Unikernels) {
			lib.startAddr = startValue
			commonMicroLibs = append(commonMicroLibs, lib)
			//fmt.Printf("%s -> 0x%x (0x%x) - 0x%x\n", lib.name, startValue, startValue, startValue+lib.size)
			startValue += lib.size
		} else if lib.instance > 1 {
			//fmt.Printf("---%s -> 0x%x (0x%x) - 0x%x\n", lib.name, startValue, startValue, startValue+lib.size)
			startValue = roundAddr(startValue)
			for i, _ := range manager.Unikernels {
				if manager.Unikernels[i].alignedLibs == nil {
					// Init structure
					manager.Unikernels[i].InitAlignment()
				}
				manager.Unikernels[i].AddAlignedMicroLibs(startValue, lib)
			}
			startValue += lib.size
			//fmt.Printf("%s -> 0x%x (0x%x) - 0x%x\n", lib.name, startValue, roundAddr(startValue), startValue+lib.size)
		} else if lib.instance == 1 {
			for i, _ := range manager.Unikernels {
				manager.Unikernels[i].AddSingleMicroLibs(roundAddr(startValue), lib)
			}
			//fmt.Printf("%s -> 0x%x (0x%x) - 0x%x\n", lib.name, startValue, roundAddr(startValue), startValue+lib.size)
		}
	}

	// Find max value through unikernels
	for i, _ := range manager.Unikernels {
		manager.Unikernels[i].alignedLibs.AllCommonMicroLibs = commonMicroLibs
		if startValue < manager.Unikernels[i].alignedLibs.startValueUk {
			startValue = manager.Unikernels[i].alignedLibs.startValueUk
		}
	}

	fmt.Printf("END SECTIONS: 0x%x\n", roundAddr(startValue))

	/*println(commonMicroLibs.String())
	for _, uk := range manager.Unikernels {
		println(uk.microLibsInstance.String())
	}*/
}

func roundAddr(value uint64) uint64 {
	x := float64(value)
	unit := float64(elf64analyser.PageSize)
	return uint64(math.Round(x/unit+0.5) * unit)
}
