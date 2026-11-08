// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/xmidt-org/skeleton"
)

func main() {
	os.Exit(run())
}

// run executes the program and returns the process exit code.  It is separate
// from main so deferred functions run before the process exits.
func run() int {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
		}
	}()

	err := skeleton.Main(os.Args[1:], true)
	if err == nil {
		return 0
	}

	fmt.Fprintln(os.Stderr, err)
	return -1
}
