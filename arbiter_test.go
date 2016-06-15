package arbiter

/*
The MIT License (MIT)

Copyright (c) 2016 Nick Potts

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

import (
	"testing"
)

func Test_New(t *testing.T) {
	var e interface{}
	catchPanic := func(t string) Arbiter {
		defer func() {
			e = recover()
		}()
		return New(t)
	}
	catchPanic("bad")
	if e == nil {
		t.Fatalf("Bad arbiter was returned rather than panic")
	}
	e = nil
	good := New("tcp")
	switch good.(type) {
	case *tcp:
	default:
		t.Fatalf("Type is not of type tcp")
	}

	e = nil
	good = New("tcp4")
	switch good.(type) {
	case *tcp:
	default:
		t.Fatalf("Type is not of type tcp")
	}
}
