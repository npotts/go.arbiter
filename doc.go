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

/*
Package arbiter provides a wrapper over a stream connections that provide a standard
command and control interface.

Arbiter Interface

The basic Arbiter interface requires only 3 functions:

	Stop - Stop the Arbiter and close underlaying stream connection(s)
	Dial - Opens initial connect over stream and verifies the connection is active via ping
	Control - Sends a command verb and waits for response, timeout, or error

Command Structure

As the purpose of this package is to provide a command and control structure around a stream,
the fundamental object that provides the needed information is a Command.  A command looks something like:

	type Command struct {
		Name string
		Timeout time.Duration
		Prototype string
		CommandRegexp *regexp.Regexp
		Response *regexp.Regexp
		Error *regexp.Regexp
		Description string
	}

Name is a shortened name for the command whereas Description is a full fledged description of what the
command ideally does. Prototype and CommandRegexp work together via Command.Bytes() to convert a Command
with arguments into a byte slice that can be pushed down a stream. Functionally, Prototype is a format
string fed into fmt.Sprintf() and the output of that is compared to CommandRegexp for validity. Response
is used to search for positive matches in the incoming byte stream where Errors is used to matches negative
responses in the byte stream. Timeout is the maximum type an Arbiter will wait before assuming the command
failed.

Response Structure

Control returns a 2-tuple of a Response (see below), and an error, which is identical to Response.Error.

	type Response struct {
    	Bytes    []byte
    	Error    error
    	Duration time.Duration
	}

Bytes is either the byte slice that matches the Command.CommandRegexp, or on any sort of error, the
contents of the internal buffer. Error is nil with no errors, or one of the Err* values from this,
or the underlying transport mechanism. Duration is simply the length of time the command took to complete.


Error Handling

It is not prefered that Arbiter try to maintain a constant connection, but rather that
when the connection dies / is killed / fails that the Arbiter passes these errors back
to the parent. This allows the parent to dispense with the error.


General Usage

A more detailed usage can be see in the testing routines.  Generally the following pattern is used:

	arbiter := New("tcp")
	if err := tt.Dial("localhost:2001",1*time.Second, PingCommand); err != nil {
		panic("Could not connect")
	}

	// ....

	response, err := arbiter.Control(SomeCommand)
	if err != nil {
		panic("Command failed")
	}
	fmt.Println("Command Succeeded ", response.Bytes)

	//....

*/
package arbiter
