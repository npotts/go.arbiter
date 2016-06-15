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
	"fmt"
	"time"
)

/*
Arbiter provides a command and control interface to []byte streams. Original design intentions
were to provide a way to communicate to devices that respond to 'commands' sent over the wire. Functionally,
this can be seen as a socket or generic IO wrapper to provide a way to read and write commands and data.
As a sanity, there can only be one caller, as this is purposefully not safe from mutliple callers via the
standard "go <func>" syntax.  Any errors that are not ErrTimeout or ErrBusy are errors coming from the
underlying layers and are to be delt with
*/
type Arbiter interface {
	//Close and free any resources in use.
	Close() error

	// Opens the connection interface.  Using terminology borrowed from *net*, but
	// could be something other than a socket.  Connection must succeed by timeout
	Dial(addr string, timeout time.Duration, pingCmd Command) error

	/*Control forms a byte slice to write out on the wire by combining cmd with args, and sans error,
	will write the formed byte slice out on the wire.  It should block until either its internal buffer
	matches cmd.Response, cmd.Error, or the process takes longer than cmd.Timeout. The returned Response should
	be populated correctly as described in the Response docstring*/
	Control(cmd Command, args ...interface{}) Response
}

/*New returns a Arbiter for the requested type.  Currently, only "tcp" or "tcp4" types are implemented
and requesting anything other than "tcp" or "tcp4" will panic*/
func New(Type string) Arbiter {
	var rtn Arbiter
	switch Type {
	case "tcp", "tcp4":
		rtn = new(tcp)
	default:
		panic(fmt.Errorf("Unable to create an Arbiter of type %q", Type))
	}
	return rtn
}
