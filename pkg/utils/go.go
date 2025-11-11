package utils

/*
 *  Copyright (c) 2024. Rapida
 *
 *  Permission is hereby granted, free of charge, to any person obtaining a copy
 *  of this software and associated documentation files (the "Software"), to deal
 *  in the Software without restriction, including without limitation the rights
 *  to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 *  copies of the Software, and to permit persons to whom the Software is
 *  furnished to do so, subject to the following conditions:
 *
 *  The above copyright notice and this permission notice shall be included in
 *  all copies or substantial portions of the Software.
 *
 *  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 *  IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 *  FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 *  AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 *  LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 *  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 *  THE SOFTWARE.
 *
 *  Author: Prashant <prashant@rapida.ai>
 *
 */

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

// Call executes the provided function and recovers from any panics that occur.
// If a panic is detected, it is reported and then re-panicked to maintain the error state.
//
// Use this when:
// - You need to execute a function that might panic
// - You want to ensure panics are properly logged before propagating
// - You need context information included in panic reporting
//
// Example:
//
//	utils.Call(ctx, func() {
//	    // code that might panic
//	})
func Call(ctx context.Context, fn func()) {
	defer func() { PanicIfNotNil(ctx, recover()) }()
	fn()
}

// CallSafe executes the provided function and recovers from any panics that occur.
// Unlike Call, this function will report the panic but not re-panic, allowing execution to continue.
//
// Use this when:
// - You need to execute a function that might panic
// - You want to handle the panic gracefully without terminating the current flow
// - You need the panic logged/reported but want to continue execution
//
// Example:
//
//	utils.CallSafe(ctx, func() {
//	    // potentially dangerous code
//	})
//	// execution continues here even if panic occurred
func CallSafe(ctx context.Context, fn func()) {
	defer func() { ReportPanicIfNotNil(ctx, recover()) }()
	fn()
}

// Go launches the provided function in a new goroutine with panic recovery.
// If the goroutine panics, the panic will be reported and re-panicked, which will
// terminate the goroutine but not the entire program.
//
// Use this when:
// - You need concurrent execution of a function
// - You want panic handling for the goroutine
// - You want the panic to be reported before the goroutine terminates
//
// Example:
//
//	utils.Go(ctx, func() {
//	    // concurrent code that might panic
//	})
func Go(ctx context.Context, fn func()) {
	go CallSafe(ctx, fn)
}

// PanicIfNotNil reports the provided recovered value if it's not nil,
// waits for 1 second (allowing logs to flush), then re-panics with formatted info.
//
// Use this when:
// - You've recovered from a panic and need to decide whether to continue
// - You want to ensure the panic is properly logged before propagating
// - You need a brief delay for log systems to process the error
//
// Example:
//
//	defer func() { utils.PanicIfNotNil(ctx, recover()) }()
func PanicIfNotNil(ctx context.Context, r any) {
	if r == nil {
		return
	}
	ReportPanicIfNotNil(ctx, r)
	time.Sleep(time.Second)
	panic(fmt.Sprintf("%#+v", r))
}

// ReportPanicIfNotNil reports the provided recovered value if it's not nil
// and returns true if reporting occurred, false otherwise.
//
// Use this when:
// - You've recovered from a panic and need to log it
// - You want to handle the panic without re-panicking
// - You need to know whether a panic occurred
//
// Example:
//
//	if utils.ReportPanicIfNotNil(ctx, recover()) {
//	    // handle the fact that a panic occurred
//	}
func ReportPanicIfNotNil(ctx context.Context, r any) bool {
	if r == nil {
		return false
	}
	panicMsg := fmt.Sprintf("%v", r)
	stackTrace := debug.Stack()
	fmt.Printf("Panic occurred: %s\n", panicMsg)
	fmt.Printf("Stack trace:\n%s", string(stackTrace))
	return true
}
