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
	"bytes"
	"fmt"
	"net"
	"time"
)

//Internal Use only
const (
	idle           = iota //waitng for some incoming request
	waitingOnReply        //request issued, waiting on reply
	responseFormed        //Response was formed
)

/*tcp implements an Arbiter over a TCP socket.*/
type tcp struct {
	alive bool
	addr  string //listen / address string, something like "some.hostname.tld:20321"

	//The following are all used internally by the go-routine and should not be accessed outside of it
	conn net.Conn     //network connection
	ibuf bytes.Buffer //incomiong buffer from the network stack
	tick *time.Ticker //poll ticker
	stop chan error   //set running to false and read from this to verify runner has stopped

	//the following are used for communicating with the main routine
	request  request       //the request we are working from
	response Response      //the reponse
	reqTime  time.Time     //time request came in
	sreq     chan request  //incoming requests
	sresp    chan Response //outgoing responses
	state    int           // state machine for
	err      error         //error vars
}

/*
Close attempts to close down the tcp connections.  After Close is called,
tcp will no longer be functional, and any data in route will be dropped.
This does nothing if the underlying goroutine is not functional, and will block
until everything is closed.  If any errors are encountered, this will return a
non-nil error.
*/
func (t *tcp) Close() error {
	if t.alive {
		to := time.NewTicker(time.Duration(40) * time.Millisecond)
		defer func() { to.Stop() }()
		t.stop <- nil //lock step with goroutine
		select {      //block waiting for
		case <-to.C: //timeout.  Error out
			return ErrTimeout
		case err := <-t.stop:
			close(t.stop)
			return err //return any errors
		}
	}
	return nil
}

/*Dial opens the TCP socket and starts the internal structures buffering data comming off the
socket.  This does maintain a goroutine in the background.  Use Close to stop everthing and kill
off the goroutine*/
func (t *tcp) Dial(addr string, timeout time.Duration, pingCmd Command) error {
	t.addr = addr
	t.conn, t.err = net.DialTimeout("tcp", t.addr, timeout)
	if t.err != nil {
		return t.err
	}

	setup := make(chan bool)
	go t.runner(setup)
	<-setup //wait for go-routine to signal it started
	close(setup)

	_, err := pingCmd.Bytes()
	if err != nil {
		panic("tcp Ping command cannot require args")
	}

	//Make sure sock is alive by sending ping command a couple times
	for i := 0; i < 3; i++ {
		if resp := t.Control(pingCmd); resp.Error != nil {
			t.stop <- nil //lock step with goroutine
			<-t.stop
			return resp.Error
		}
	}
	return nil
}

/*Control is a Request-Reply patern. It sends out the request, and wait up to timeout for
a reply that matched the passed regexp.  The returned Response structure holds the bytes read, as
well as easy way to get to the reply data.  The command is considered "successful" if the reply
matches anywhere in the incoming byte stream from the remote host.  Any internal buffers are
flushed before the request is issued.  If an error is returned, Response.Bytes will be the contents
of whatever was on the incoming buffer.  If error is nil, Response.Bytes will be whatever byte slice
matched cmd.Response, with extra bytes removed.
*/
func (t *tcp) Control(cmd Command, args ...interface{}) Response {
	if !t.alive {
		return Response{Error: ErrNotConnected}
	}
	ireq := request{Command: cmd}
	//Check if the command can even be properly expanded with the args provided
	var err error
	ireq.bytes, err = cmd.Bytes(args...)
	if err != nil {
		return Response{Error: err}
	}
	t.sreq <- ireq //lock step, waiting for goroutine to respond
	r := <-t.sresp
	return r
}

/* sock2ibuf reads data off the socket and shovels them into our buffer.  This is only called
from within the go-routine to serialize access to the internal structures */
func (t *tcp) sock2ibuf() {
	b := make([]byte, 1024)
	t.conn.SetReadDeadline(time.Now().Add(time.Duration(1) * time.Millisecond)) //dont wait here
	n, err := t.conn.Read(b)                                                    //only reads up to the size of b
	//bytes to  buffer
	t.ibuf.Write(b[0:n])
	if toerr, ok := err.(net.Error); ok && toerr.Timeout() {
		t.err = nil
	} else if err != nil {
		t.err = err
	}
	t.err = nil
}

/*checkState checks the various pass and fail conditions*/
func (t *tcp) checkState() (Response, int) {
	if t.state == waitingOnReply {
		t.response.Error = errUnformedResponse
		//check if we need to send a response.  This happens by a timeout or a match
		alterResp := func(e error, by []byte) {
			t.response.Error = e
			t.response.Bytes = by
			t.response.Duration = time.Since(t.reqTime)
			t.state = responseFormed //tell goroutine we got a response they can handle
		}

		if time.Now().Sub(t.reqTime) > t.request.Command.Timeout { //timeout
			alterResp(ErrTimeout, t.ibuf.Bytes())
			return t.response, t.state
		}

		if t.request.Command.Error.Match(t.ibuf.Bytes()) { //Check for Failure Match
			alterResp(ErrMatch, t.ibuf.Bytes())
			return t.response, t.state
		}

		if t.request.Command.Response.Match(t.ibuf.Bytes()) { //Check for Success Match
			alterResp(nil, t.request.Command.Response.Find(t.ibuf.Bytes()))
			return t.response, t.state
		}

		fmt.Sprintf("")
	}
	return t.response, t.state
}

func (t *tcp) handleIncoming(r request) {
	if t.state != idle { //Busy
		resp := Response{Bytes: []byte(""), Error: ErrBusy}
		if t.err != nil { //f there was another error, (disconnected, etc) repeat that instead
			resp.Error = t.err
		}
		t.sresp <- resp
		return
	}
	t.ibuf.Truncate(0)                               //clear out internal buffer
	if _, err := t.conn.Write(r.bytes); err != nil { //write request onto the wire
		t.err = err //connection broken
		t.sresp <- Response{Bytes: []byte(""), Error: err}
		return
	}
	t.request = r
	t.reqTime = time.Now()
	t.state = waitingOnReply
}

/*runner is called as a go-routine internally*/
func (t *tcp) runner(setup chan<- bool) {
	t.alive = true
	//We are really up.  Start the background goroutine stuffs

	t.stop = make(chan error)
	t.tick = time.NewTicker(time.Duration(1) * time.Millisecond) //poll for crap every 20ms
	t.sreq = make(chan request)
	t.sresp = make(chan Response)

	//start background go routine to poll for data
	setup <- true

	defer func() {
		t.conn.Close() //kill network connection
		t.tick.Stop()  //GC stop ticker

		close(t.sreq)
		close(t.sresp)
		t.alive = false //done elsewhere as well, but just a failsafe
	}()

	for { //loop until we are told to stop
		select { //block
		case <-t.tick.C: //tick for checking for more data off the socket
			t.sock2ibuf()
		case r := <-t.sreq: //Incoming request or command.
			t.handleIncoming(r)
		case <-t.stop:
			t.alive = false //make sure we set this syncronously before we give up
			t.stop <- nil   //signal back we are done
			return
		}
		t.checkState() //force checking state (timeout, errors, or command data matches)
		//secondary, branched select for checking if we need to send a Response
		if t.state == responseFormed {
			select {
			case t.sresp <- t.response: //send response if requested
				t.state = idle //finished sending
			default:
			}

		}
	}
}
