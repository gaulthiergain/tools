package dependtool

import (
	"debug/elf"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"os"
	"runtime"
	"strconv"
	"strings"
	u "tools/srcs/common"
)

// RunAnalyserTool allows to run the dependency analyser tool.
func RunAnalyserTool(homeDir string, data *u.Data) {

	// Support only Unix
	if strings.ToLower(runtime.GOOS) == "windows" {
		u.PrintErr("Windows platform is not supported")
	}

	// Init and parse local arguments
	args := new(u.Arguments)
	p, err := args.InitArguments()
	if err != nil {
		u.PrintErr(err)
	}
	if err := parseLocalArguments(p, args); err != nil {
		u.PrintErr(err)
	}

	// Get the kind of analysis (0: both; 1: static; 2: dynamic)
	typeAnalysis := *args.IntArg[typeAnalysis]
	if typeAnalysis < 0 || typeAnalysis > 2 {
		u.PrintErr(errors.New("analysis argument must be between [0,2]"))
	}

	// Get program path
	programPath, err := u.GetProgramPath(&*args.StringArg[programArg])
	if err != nil {
		u.PrintErr("Could not determine program path", err)
	}

	// Get program Name
	programName := *args.StringArg[programArg]

	// Create the folder 'output' if it does not exist
	outFolder := homeDir + u.SEP + programName + "_" + u.OUTFOLDER
	if _, err := u.CreateFolder(outFolder); err != nil {
		u.PrintErr(err)
	}

	// Display Minor Details
	displayProgramDetails(programName, programPath, args)

	var elfFile *elf.File
	isDynamic := false
	isLinux := strings.ToLower(runtime.GOOS) == "linux"

	// Check if the program is a binary
	if isLinux {
		elfFile, isDynamic = checkElf(&programPath)
	} else if strings.ToLower(runtime.GOOS) == "darwin" {
		u.PrintWarning("Static analysis is limited on macOS")
		checkMachOS(&programPath)
	}

	// Run static analyser
	if typeAnalysis == 0 || typeAnalysis == 1 {
		u.PrintHeader1("(1.1) RUN STATIC ANALYSIS")
		runStaticAnalyser(elfFile, isDynamic, isLinux, args, programName, programPath, outFolder, data)
	}

	// Run dynamic analyser
	if typeAnalysis == 0 || typeAnalysis == 2 {
		if isLinux {
			u.PrintHeader1("(1.2) RUN DYNAMIC ANALYSIS")
			runDynamicAnalyser(args, programName, programPath, outFolder, data)
		} else {
			// dtruss/dtrace on mac needs to disable system integrity protection
			u.PrintWarning("Dynamic analysis is not currently supported on macOS")
		}
	}

	// Use the trapper
	if *args.BoolArg[trapper] {
		u.PrintHeader2("Run syscalls trapper tool")
		if err := runTrapper(programName, data); err != nil {
			u.PrintWarning("Error during tracking RAX (results may be incomplete)" + err.Error())
		}
	}

	// Save Data to JSON
	if err = u.RecordDataJson(programName, data); err != nil {
		u.PrintErr(err)
	} else {
		u.PrintOk("All JSON Data saved into: " + programName +
			".json")
	}

	// Save syscall as txt
	if err = saveSystemCalls("syscalls_"+programName+".txt", data); err != nil {
		u.PrintErr(err)
	} else {
		u.PrintOk("Syscalls saved into: syscalls_" + programName + ".txt")
	}

	// Save graph if full dependencies option is set
	if *args.BoolArg[fullDepsArg] {
		saveGraph(programName, outFolder, data)
	}
}

func saveSystemCalls(filename string, data *u.Data) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	for key, element := range data.StaticData.SystemCalls {
		if _, err := file.WriteString(strconv.Itoa(element) + ":" + key + "\n"); err != nil {
			return err
		}
	}

	return nil
}

// displayProgramDetails display various information such path, background, ...
func displayProgramDetails(programName, programPath string, args *u.Arguments) {
	fmt.Println("----------------------------------------------")
	fmt.Println("Analyze Program: ", color.GreenString(programName))
	fmt.Println("Full Path: ", color.GreenString(programPath))
	if len(*args.StringArg[optionsArg]) > 0 {
		fmt.Println("Options: ", color.GreenString(*args.StringArg[optionsArg]))
	}

	if len(*args.StringArg[configFileArg]) > 0 {
		fmt.Println("Config file: ", color.GreenString(*args.StringArg[configFileArg]))
	}

	if len(*args.StringArg[testFileArg]) > 0 {
		fmt.Println("Test file: ", color.GreenString(*args.StringArg[testFileArg]))
	}

	fmt.Println("----------------------------------------------")
}

// checkMachOS checks if the program (from its path) is an Mach os file
func checkMachOS(programPath *string) {
	if err := getMachOS(*programPath); err != nil {
		u.PrintErr(err)
	}
}

// checkElf checks if the program (from its path) is an ELF file
func checkElf(programPath *string) (*elf.File, bool) {
	dynamicCompiled := false
	elfFile, err := getElf(*programPath)
	if err != nil {
		u.PrintErr(err)
	} else if elfFile == nil {
		*programPath = ""
		u.PrintWarning("Only ELF binaries are supported! Some analysis" +
			" procedures will be skipped")
	} else {

		// Get ELF architecture
		architecture, machine := GetElfArchitecture(elfFile)
		fmt.Println("ELF Class: ", architecture)
		fmt.Println("Machine: ", machine)
		fmt.Println("Entry Point: ", elfFile.Entry)
		for _, s := range elfFile.Sections {
			if strings.Contains(s.Name, ".dynamic") {
				fmt.Println("Type: Dynamically compiled")
				dynamicCompiled = true
				break
			}
		}
		if !dynamicCompiled {
			fmt.Println("Type: Statically compiled")
		}
		fmt.Println("----------------------------------------------")
	}
	return elfFile, dynamicCompiled
}

// runStaticAnalyser runs the static analyser
func runStaticAnalyser(elfFile *elf.File, isDynamic, isLinux bool, args *u.Arguments, programName, programPath,
	outFolder string, data *u.Data) {

	staticAnalyser(elfFile, isDynamic, isLinux, *args, data, programPath)

	// Save static Data into text file if display mode is set
	if *args.BoolArg[saveOutputArg] {

		// Create the folder 'output/static' if it does not exist
		outFolderStatic := outFolder + "static" + u.SEP
		if _, err := u.CreateFolder(outFolderStatic); err != nil {
			u.PrintErr(err)
		}

		fn := outFolderStatic + programName + ".txt"
		headersStr := []string{"Dependencies (from apt-cache show) list:",
			"Shared libraries list:", "System calls list:", "Symbols list:"}

		if err := u.RecordDataTxt(fn, headersStr, data.StaticData); err != nil {
			u.PrintWarning(err)
		} else {
			u.PrintOk("Data saved into " + fn)
		}
	}
}

// runDynamicAnalyser runs the dynamic analyser.
func runDynamicAnalyser(args *u.Arguments, programName, programPath,
	outFolder string, data *u.Data) {

	dynamicAnalyser(args, data, programPath)

	// Save dynamic Data into text file if display mode is set
	if *args.BoolArg[saveOutputArg] {

		// Create the folder 'output/dynamic' if it does not exist
		outFolderDynamic := outFolder + "dynamic" + u.SEP
		if _, err := u.CreateFolder(outFolderDynamic); err != nil {
			u.PrintErr(err)
		}

		fn := outFolderDynamic + programName + ".txt"
		headersStr := []string{"Shared libraries list:", "System calls list:",
			"Symbols list:"}

		if err := u.RecordDataTxt(fn, headersStr, data.DynamicData); err != nil {
			u.PrintWarning(err)
		} else {
			u.PrintOk("Data saved into " + fn)
		}
	}
}

// saveGraph saves dependency graphs of a given app into the output folder.
func saveGraph(programName, outFolder string, data *u.Data) {

	if len(data.StaticData.SharedLibs) > 0 {
		u.GenerateGraph(programName, outFolder+"static"+u.SEP+
			programName+"_shared_libs", data.StaticData.SharedLibs, nil)
	}

	if len(data.StaticData.Dependencies) > 0 {
		u.GenerateGraph(programName, outFolder+"static"+u.SEP+
			programName+"_dependencies", data.StaticData.Dependencies, nil)
	}

	if len(data.StaticData.SharedLibs) > 0 {
		u.GenerateGraph(programName, outFolder+"dynamic"+u.SEP+
			programName+"_shared_libs", data.DynamicData.SharedLibs, nil)
	}
}
