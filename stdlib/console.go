// Copyright 2018 Alexey Krivonogov. All rights reserved.
// Use of this source code is governed by a MIT license
// that can be found in the LICENSE file.

package stdlib

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/gentee/gentee/core"
)

// InitConsole appends stdlib console functions to the virtual machine
func InitConsole(vm *core.VirtualMachine) {
	for _, item := range []interface{}{
		PrintShiftºStr, // unary bitwise OR
		Print,          // Print()
		Println,        // Println()
		ReadString,     // ReadString(str) str
	} {
		vm.StdLib().NewEmbed(item)
	}
}

// Print writes to standard output.
func Print(pars ...interface{}) (int64, error) {
	n, err := fmt.Print(pars...)
	return int64(n), err
}

// Println writes to standard output.
func Println(pars ...interface{}) (int64, error) {
	n, err := fmt.Println(pars...)
	return int64(n), err
}

// PrintShiftºStr writes to standard output with trim spaces characters in the each line.
func PrintShiftºStr(par string) (int64, error) {
	lines := strings.Split(par, "\n")
	for i, v := range lines {
		lines[i] = strings.TrimSpace(v)
	}
	return Print(strings.Join(lines, "\n"))
}

// ReadString reads a string from standard input.
func ReadString(text string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	if len(text) > 0 {
		fmt.Print(text)
	}
	ret, err := reader.ReadString('\n')
	return strings.TrimSpace(ret), err
}
