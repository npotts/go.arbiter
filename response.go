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
	"errors"
	"fmt"
	"time"
)

/*
Response is what is returns from Read commands  Bytes is the actual bytes read. In the case of
Read(), it is whatever bytes were ingested, and for the Control funcs, it is the contents of the buffer
when either the timeout occured, or the match was met. Error is any low-level error that may have occured from
the actual stream transport, or one of this packages's named Err* errors.
*/
type Response struct {
	Bytes    []byte        //Raw bytes read or received.  In Control funcs, this is the raw value that matched the 'match' clause
	Error    error         //any non-nil errors
	Duration time.Duration //how long did the request take
}

//String implements the Stringer interface
func (r Response) String() string {
	return fmt.Sprintf("Response> Rx Bytes: %q\tErrors: %v\tDuration: %v", r.Bytes, r.Error, r.Duration)
}

//ErrTimeout is the error returned when a command fails
var ErrTimeout = errors.New("Didnt get the required response in the duration specified")

//ErrBusy is returned if we are already busy with another command
var ErrBusy = errors.New("Busy - Another operation in progress")

//ErrNotConnected is returned if commands are sent on a closed connection
var ErrNotConnected = errors.New("Not connected")

//ErrMatch is returned if the provided error regex in command matches the bytes returned.
var ErrMatch = errors.New("Card returned error response")

//errUnformedResponse is the internal error when the system is still waiting for a timeout, positive or negative reply.
//this is not exported and only used internally
var errUnformedResponse = errors.New("Unformed Response: still waiting for timeout, error match, or positive match to occur")

/*
request is the corellary to Response.  This is passed to the go-routine for it to handle.
These are compiled by the function handlers and handeled by the go routine
*/
type request struct {
	Command Command //command to send in
	bytes   []byte  //result of Command.Bytes() with passed args
}
