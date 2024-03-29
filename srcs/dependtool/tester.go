// Copyright 2019 The UNICORE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file
//
// Author: Gaulthier Gain <gaulthier.gain@uliege.be>
package dependtool

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	u "tools/srcs/common"
)

const (
	stdinTest = iota
	execTest
	telnetTest
	externalTesting
)

const (
	stdinTestString  = "stdin"
	execTestString   = "exec"
	telnetTestString = "telnet"
)

const (
	timeOutMs  = 3000
	startupSec = 3
)

type Testing struct {
	TypeTest    string `json:"typeTest"`
	TimeOutTest int    `json:"timeOutMsTest"`

	// Only for telnet test
	AddressTelnet string `json:"addressTelnet"`
	PortTelnet    int    `json:"portTelnet"`

	TimeCommand  int32    `json:"timeMsCommand"`
	ListCommands []string `json:"listCommands"`
}

// checkTypeTest checks the type of testing: exec, stdin and telnet tests.
//
// It returns an integer which represents the type of tests
func checkTypeTest(testStruct *Testing) int {

	if testStruct == nil {
		return externalTesting
	}

	if strings.Compare(testStruct.TypeTest, stdinTestString) == 0 {
		return stdinTest
	} else if strings.Compare(testStruct.TypeTest, execTestString) == 0 {
		return execTest
	} else if strings.Compare(testStruct.TypeTest, telnetTestString) == 0 {
		return telnetTest
	}

	return externalTesting
}

// setDurationTimeOut sets the duration of the timeout by computing a reference
// value. In addition an extra margin value is added (3sec).
//
// It returns a duration either in milliseconds or in seconds.
func setDurationTimeOut(t *Testing, waitTime int) time.Duration {

	if checkTypeTest(t) != externalTesting {
		// Compute the number of commands + execution time (+ 3 seconds safe margin)
		totalMs := t.TimeCommand*int32(len(t.ListCommands)) + timeOutMs
		return time.Duration(totalMs) * time.Millisecond
	}

	return time.Duration(waitTime+startupSec) * time.Second
}

// runCommandTester run commands and captures stdout and stderr of a the
// executed command. It will also run the Tester to explore several execution
// paths of the given app.
//
// It returns to string which are respectively stdout and stderr.
func runCommandTester(programPath, programName, command, option string,
	testStruct *Testing, dArgs DynamicArgs, data *u.DynamicData) (string, string) {

	timeOut := setDurationTimeOut(testStruct, dArgs.waitTime)
	u.PrintInfo("Max testing duration of " + programName + " : " + timeOut.String())
	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()

	args := strings.Fields("-f " + programPath + " " + option)
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	bufOut, bufErr, bufIn := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout = bufOut // Add io.MultiWriter(os.Stdout) to record on stdout
	cmd.Stderr = bufErr // Add io.MultiWriter(os.Stderr) to record on stderr

	if checkTypeTest(testStruct) == stdinTest {
		cmd.Stdin = bufIn
		for _, cmds := range testStruct.ListCommands {
			time.Sleep(100 * time.Millisecond)
			bufIn.Write([]byte(cmds + "\n"))
		}
	}

	// Run the process (traced by strace/ltrace)
	if err := cmd.Start(); err != nil {
		u.PrintErr(err)
	}

	// Run a go routine to handle the tests
	go func() {
		if checkTypeTest(testStruct) != stdinTest {
			Tester(programName, cmd, data, testStruct, dArgs)

			// Kill the program after the tester has finished the job
			if err := u.PKill(programName, syscall.SIGINT); err != nil {
				u.PrintErr(err)
			}
		}
	}()

	// Ignore the error because the program is killed (waitTime)
	_ = cmd.Wait()

	if ctx.Err() == context.DeadlineExceeded {
		u.PrintInfo("Time out during executing: " + cmd.String())
		return bufOut.String(), bufErr.String()
	}

	return bufOut.String(), bufErr.String()
}

func RunVerifCommandTester(programPath, programName, option string, testStruct *Testing) (string, string) {

	timeOut := setDurationTimeOut(testStruct, 1)
	u.PrintInfo("Max testing duration of " + programName + " : " + timeOut.String())
	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()

	args := strings.Fields(option)
	cmd := exec.CommandContext(ctx, programPath, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	bufOut, bufErr, bufIn := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}
	cmd.Stdout = bufOut // Add io.MultiWriter(os.Stdout) to record on stdout
	cmd.Stderr = bufErr // Add io.MultiWriter(os.Stderr) to record on stderr

	if checkTypeTest(testStruct) == stdinTest {
		cmd.Stdin = bufIn
		bufIn.Write([]byte(" \n"))
		for _, cmds := range testStruct.ListCommands {
			if strings.Contains(programName, "qemu-system-x86_64") {
				time.Sleep(1000 * time.Millisecond)
			} else {
				time.Sleep(100 * time.Millisecond)
			}
			bufIn.Write([]byte(cmds + "\n"))
		}
	}

	// Run the process
	if err := cmd.Start(); err != nil {
		u.PrintErr(err)
	}

	// Run a go routine to handle the tests
	go func() {
		if checkTypeTest(testStruct) != stdinTest {
			VerifTester(testStruct)

			// Kill the program after the tester has finished the job
			if err := u.PKill(programName, syscall.SIGINT); err != nil {
				u.PrintErr(err)
			}
		}
	}()

	// Ignore the error because the program is killed (waitTime)
	_ = cmd.Wait()

	if ctx.Err() == context.DeadlineExceeded {
		u.PrintInfo("Time out during executing: " + cmd.String())
		return bufOut.String(), bufErr.String()
	}

	return bufOut.String(), bufErr.String()
}

// Tester runs the executable file of a given application to perform tests to
// get program dependencies.
//
func Tester(programName string, cmd *exec.Cmd, data *u.DynamicData,
	testStruct *Testing, dArgs DynamicArgs) {

	if len(dArgs.testFile) > 0 {
		// Wait until the program has started
		time.Sleep(time.Second * startupSec)
		u.PrintInfo("Run internal tests from file " + dArgs.testFile)

		// Launch execution tests
		if checkTypeTest(testStruct) == execTest {
			launchTestsExternal(testStruct)
		} else if checkTypeTest(testStruct) == telnetTest {
			if len(testStruct.AddressTelnet) == 0 || testStruct.PortTelnet == 0 {
				u.PrintWarning("Cannot find Address and port for telnet " +
					"within json file. Skip tests")
			} else {
				launchTelnetTest(testStruct)
			}
		}
	} else {
		u.PrintInfo("Waiting for external tests for " + strconv.Itoa(
			dArgs.waitTime) + " sec")
		ticker := time.Tick(time.Second)
		for i := 1; i <= dArgs.waitTime; i++ {
			<-ticker
			fmt.Printf("-")
		}
		fmt.Printf("\n")
	}

	// Gather shared libs
	u.PrintHeader2("(*) Gathering shared libs")
	if err := gatherDynamicSharedLibs(programName, cmd.Process.Pid, data,
		dArgs.fullDeps); err != nil {
		u.PrintWarning(err)
	}
}

func VerifTester(testStruct *Testing) {
	// Wait until the program has started
	time.Sleep(time.Second * startupSec)
	u.PrintInfo("Run internal tests from test file")

	// Launch execution tests
	if checkTypeTest(testStruct) == execTest {
		launchTestsExternal(testStruct)
	} else if checkTypeTest(testStruct) == telnetTest {
		if len(testStruct.AddressTelnet) == 0 || testStruct.PortTelnet == 0 {
			u.PrintWarning("Cannot find Address and port for telnet " +
				"within json file. Skip tests")
		} else {
			launchTelnetTest(testStruct)
		}
	}
}

//----------------------------------Tests---------------------------------------

// launchTestsExternal runs external tests written in the 'test.json' file.
//
func launchTestsExternal(testStruct *Testing) {

	for _, cmd := range testStruct.ListCommands {
		if len(cmd) > 0 {

			// Perform a sleep between command if specified
			if testStruct.TimeCommand > 0 {
				timeMs := rand.Int31n(testStruct.TimeCommand)
				time.Sleep(time.Duration(timeMs) * time.Millisecond)
			}

			// Execute each line as a command
			if _, err := u.ExecutePipeCommand(cmd); err != nil {
				u.PrintWarning("Impossible to execute test: " + cmd)
			} else {
				u.PrintInfo("Test executed: " + cmd)
			}
		}
	}
}

// launchTelnetTest runs telnet tests written in the 'test.json' file.
//
func launchTelnetTest(testStruct *Testing) {

	addr := testStruct.AddressTelnet + ":" + strconv.Itoa(testStruct.PortTelnet)
	conn, _ := net.Dial("tcp", addr)

	for _, cmd := range testStruct.ListCommands {
		if len(cmd) > 0 {

			// Perform a sleep between command if specified
			if testStruct.TimeCommand > 0 {
				timeMs := rand.Int31n(testStruct.TimeCommand)
				time.Sleep(time.Duration(timeMs) * time.Millisecond)
			}

			// Set a timeout to avoid blocking
			if err := conn.SetReadDeadline(
				time.Now().Add(time.Duration(timeOutMs) * time.Millisecond)); err != nil {
				u.PrintWarning("Impossible to set a timeout to TCP command")
			}

			// Send commands (test)
			if _, err := fmt.Fprintf(conn, cmd+"\n"); err != nil {
				u.PrintWarning("Impossible to execute test: " + cmd)
			} else {
				u.PrintInfo("Test executed: " + cmd)
			}

			// Read response
			message := readerTelnet(conn)
			fmt.Println("----->Message from server: " + message)
		}
	}
}

// readerTelnet reads data from the telnet connection.
//
func readerTelnet(conn net.Conn) (out string) {
	var buffer [1]byte
	recvData := buffer[:]
	var n int
	var err error

	for {
		n, err = conn.Read(recvData)
		if n <= 0 || err != nil {
			break
		} else {
			out += string(recvData)
		}
	}
	return out
}
