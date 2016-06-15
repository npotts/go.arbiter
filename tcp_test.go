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
	"flag"
	"fmt"
	"net"
	"os"
	"regexp"
	"testing"
	"time"
)

var pingOk, pingWrong, pingBad, closeNice, closeEvil = Command{
	Name:          "ping",
	Timeout:       time.Duration(300) * time.Millisecond,
	Prototype:     "\r",
	CommandRegexp: regexp.MustCompile("\r"),
	Response:      regexp.MustCompile("\r"),
	Error:         regexp.MustCompile("a^"),
}, Command{
	Name:          "ping wrong",
	Timeout:       time.Duration(300) * time.Millisecond,
	Prototype:     "Req Arg, No Arg %d\r",
	CommandRegexp: regexp.MustCompile(".*"),
	Response:      regexp.MustCompile("\r"),
	Error:         regexp.MustCompile("a^"),
}, Command{
	Name:          "ping with no echo",
	Timeout:       time.Duration(150) * time.Millisecond,
	Prototype:     "DONT-ECHO",
	CommandRegexp: regexp.MustCompile("DONT-ECHO"),
	Response:      regexp.MustCompile("DONT-ECHO"),
	Error:         regexp.MustCompile("a^"),
}, Command{
	Name:          "close-nice",
	Timeout:       time.Duration(150) * time.Millisecond,
	Prototype:     "close-nice",
	CommandRegexp: regexp.MustCompile("close-nice"),
	Response:      regexp.MustCompile("ok"),
	Error:         regexp.MustCompile("a^"),
}, Command{
	Name:          "close-evil",
	Timeout:       time.Duration(150) * time.Millisecond,
	Prototype:     "close-evil",
	CommandRegexp: regexp.MustCompile("close-evil"),
	Response:      regexp.MustCompile("ok"),
	Error:         regexp.MustCompile("a^"),
}

func init() {
	flag.StringVar(&dial, "dial", "localhost:2001", "Listen & test on this host:port")
}

var dial string

/*HandleRequest incoming requests.*/
func HandleRequest(conn net.Conn) {
	for {
		buf := make([]byte, 1024)
		// var cooldown time.Duration
		reqLen, err := conn.Read(buf) // Read the incoming connection into the buffer.
		buf = buf[0:reqLen]
		if err != nil {
			// fmt.Println("Error reading:", err.Error())
			return
		}
		// buf = bytes.Trim(buf, "\r\n")
		switch string(buf) {
		case "\r":
			buf = []byte("\r")
		case "DONT-ECHO": //dont echo a response
			buf = buf[0:0]
		case "close-nice": //close connection nicely
			conn.Write([]byte("ok"))
			conn.Close() // Close the connection when you're done with it.
			return
		case "close-evil": //close connection nicely
			conn.Write([]byte("ok"))
			return
		}
		if _, err := conn.Write(buf); err != nil {
			fmt.Println("Simulator Write Error. ", err)
			return
		}
	}
}

func TcpServer() {
	l, err := net.Listen("tcp", dial) // Listen for incoming connections.
	if err != nil {
		panic(fmt.Errorf("Unable to create listening socked on  %q: %v", dial, err))
	}
	//Because this does not call l.Close, this leaks memory
	// fmt.Printf("Listening on %s\n", dial)
	for {
		conn, err := l.Accept() // Listen for an incoming connection.
		if err == nil {
			go HandleRequest(conn)
		} else {
			// fmt.Println("Error accepting: ", err.Error())
		}

	}
}

func TestMain(m *testing.M) {
	go TcpServer()
	os.Exit(m.Run())
}

func TestTcp_Close(t *testing.T) {
	tcp := new(tcp)
	tcp.stop = make(chan error, 0)
	if tcp.Close() != nil {
		t.Fatalf("Close should return immediately if not started")
	}

	tcp.alive = true
	//allow timeout
	go func() { <-tcp.stop }()
	if tcp.Close() != ErrTimeout {
		t.Fatalf("Should timeout")
	}

	go func() {
		<-tcp.stop
		tcp.stop <- ErrNotConnected
	}()

	if tcp.Close() != ErrNotConnected {
		t.Fatalf("Should not timeout")
	}
}

func TestTcp_Dial(t *testing.T) {
	tcp_ := new(tcp)
	if e := tcp_.Dial("host-does-not-exist:65537", 100*time.Millisecond, pingOk); e == nil {
		t.Fatalf("Invalid Hostname and port should fail dial")
	}
	tcp_.Close()

	var e interface{}
	cpanic := func() {
		defer func() { e = recover() }()
		tcp_.Dial(dial, 100*time.Millisecond, pingWrong)
	}

	cpanic()
	if e == nil {
		t.Fatalf("Invalid dial should panic")
	}
	tcp_.Close()

	//now dial with a bad and sucessful attempts

	tcp_ = new(tcp)
	if e := tcp_.Dial(dial, 100*time.Millisecond, pingOk); e != nil {
		t.Fatalf("Dial should be successful with good ping ", e)
	}
	tcp_.Close()

	tcp_ = new(tcp)
	if e := tcp_.Dial(dial, 100*time.Millisecond, pingBad); e == nil {
		t.Fatalf("Dial should  fail with bad ping")
	}
	tcp_.Close()
}

func TestTcp_Control(t *testing.T) {
	tcp_ := new(tcp)
	tcp_.sreq = make(chan request)
	tcp_.sresp = make(chan Response)

	if resp := tcp_.Control(pingOk); resp.Error != ErrNotConnected {
		t.Fatalf("When in unstarted state, should fail")
	}

	tcp_.alive = true

	if resp := tcp_.Control(pingWrong); resp.Error != ErrBytesArgs {
		t.Fatalf("Not feeding requied arg should produce an error")
	}

	//now check, but manually control .sreq ??
	go func() { //
		<-tcp_.sreq //dont care about their request.  Send some response
		tcp_.sresp <- Response{Error: ErrNotConnected}
	}()
	resp := tcp_.Control(pingWrong)
	if resp.Error == nil {
		t.Fatalf("Manually interveining for runner() failed")
	}
}

func mustGetError(tc *tcp, ti time.Duration) (b bool) {
	then := time.Now()
	for {
		if time.Now().Sub(then) > ti {
			return
		}
		tc.sock2ibuf()
		if tc.err != nil {
			b = true
			return
		}
	}
	return
}

func TestTcp_sock2ibuf(t *testing.T) {
	tcp_ := new(tcp)
	if e := tcp_.Dial(dial, 100*time.Millisecond, pingOk); e != nil {
		t.Fatalf("Need initial connection to be setup, and couldnt setup")
		t.FailNow()
	}
	defer tcp_.conn.Close()

	//emulate no-data read == net.Error timeout
	tcp_.sock2ibuf() //should timeout
	//check some flags
	if tcp_.err != nil {
		t.Fatalf("timeout should not set an error")
	}

	//manually send some data over conn
	tcp_.conn.Write([]byte(pingOk.Prototype))
	time.Sleep(20 * time.Millisecond)
	tcp_.sock2ibuf()
	//check some flags
	if tcp_.err != nil {
		t.Fatalf("result should be available, but itsnt")
	}

	tcp_.conn.Write([]byte(closeNice.Prototype))

	if !mustGetError(tcp_, 1000*time.Millisecond) {
		// t.Fatalf("Should error out with multiple requests")
	}

	//do the same thing, but close malignantly
	tcp_ = new(tcp)
	if e := tcp_.Dial(dial, 1000*time.Millisecond, pingOk); e != nil {
		t.Fatalf("Need initial connection to be setup, and couldnt setup")
		t.FailNow()
	}
	defer tcp_.conn.Close()
	tcp_.conn.Write([]byte(closeEvil.Prototype))

	if !mustGetError(tcp_, 100*time.Millisecond) {
		// t.Fatalf("Should error out with multiple requests")
	}
}

func TestTcp_checkState(t *testing.T) {
	tc := new(tcp)
	//now the harder checks
	tc.request.Command = Command{
		Name:          "test",
		Timeout:       5 * time.Second,
		Prototype:     "test",
		CommandRegexp: regexp.MustCompile("test"),
		Response:      regexp.MustCompile("[a-z]{1,}\n"),
		Error:         regexp.MustCompile("[0-9]{1,}\n"),
	}

	type ttt struct {
		N          string
		T1, T2     time.Time
		E1, E2     error
		B          []byte
		S0, S1, S2 int //start state, final after test 1, final after test2
	}

	tc.response.Error = errUnformedResponse
	tests := []ttt{
		ttt{N: "Idle Case", S0: idle, S1: idle, S2: idle, E1: errUnformedResponse, E2: errUnformedResponse},
		ttt{N: "response formed Case", S0: responseFormed, S1: responseFormed, S2: responseFormed, E1: errUnformedResponse, E2: errUnformedResponse},
		ttt{N: "Timeout Cases", T1: time.Now().Add(10 * time.Second), T2: time.Now().Add(-10 * time.Second), E1: errUnformedResponse, E2: ErrTimeout, S0: waitingOnReply, S1: waitingOnReply, S2: responseFormed},
		ttt{N: "Error Match", T1: time.Now(), T2: time.Now(), E1: errUnformedResponse, E2: ErrMatch, S0: waitingOnReply, S1: waitingOnReply, S2: responseFormed, B: []byte("1234567890\n")},
		ttt{N: "Good Match", T1: time.Now(), T2: time.Now(), E1: errUnformedResponse, E2: nil, S0: waitingOnReply, S1: waitingOnReply, S2: responseFormed, B: []byte("abcdefg\n")},
	}

	runTest := func(c ttt) {
		//setup
		tc.ibuf.Reset()
		tc.reqTime = c.T1
		tc.state = c.S0
		// fmt.Println("||A>", tc.response)
		if resp, state := tc.checkState(); resp.Error != c.E1 || state != c.S1 {
			t.Errorf("checkState() failed test #1 %q:\n\tError  '%v' != '%v'\n\t'%d' != '%d'", c.N, resp.Error, c.E1, state, c.S1)
		}

		//setup for positive test
		tc.ibuf.Reset()
		tc.ibuf.Write(c.B)
		if !bytes.Equal(tc.ibuf.Bytes(), c.B) {
			panic("adsada")
		}
		tc.reqTime = c.T2
		tc.state = c.S0
		if resp, state := tc.checkState(); !bytes.Equal(c.B, resp.Bytes) ||
			resp.Error != c.E2 ||
			state != c.S2 {
			fmt.Println("@@@@@@@", resp)
			t.Errorf("checkState() failed test #2 %q:\n\tError  '%v' != '%v'\n\t'%d' != '%d'\n\t%v != %v", c.N, resp.Error, c.E2, state, c.S2, resp.Bytes, c.B)
		}
	}

	for _, test := range tests {
		runTest(test)
	}
}

func TestTcp_handleIncoming(t *testing.T) {
	tc := new(tcp)
	tc.sresp = make(chan Response, 0)
	var resp Response
	req := request{}

	tc.err = nil
	tc.state = idle - 1
	go tc.handleIncoming(req)
	select {
	case resp = <-tc.sresp:
	}
	if resp.Error != ErrBusy {
		t.Errorf("Should get busy signal")
	}

	tc.err = errUnformedResponse
	tc.state = idle - 1
	go tc.handleIncoming(req)
	select {
	case resp = <-tc.sresp:
	}
	if resp.Error != errUnformedResponse {
		t.Errorf("Underlying errors should override busy signal")
	}

	var err error
	if tc.conn, err = net.DialTimeout("tcp", dial, 1*time.Second); err != nil {
		t.Fatalf("Unable to perform needed dial")
	}
	// defer tc.conn.Close()

	// go read()
	tc.state = idle
	tc.handleIncoming(req)
	if tc.state != waitingOnReply {
		t.Errorf("Should be setting waitinOnReply bit")
	}
	//locally kill connection so write will fail
	tc.conn.Close()

	tc.state = idle
	go tc.handleIncoming(req) //should error out here
	select {
	case resp = <-tc.sresp:
	}
	if resp.Error == nil {
		t.Errorf("Should not be able to write to closed socket")
	}
}

// func Test_tcp(t *testing.T) {
// 	fmt.Println("Testing tcp Arbiter")
// 	tt := new(tcp)
// 	time.Sleep(2 * time.Second)

// 	err := tt.Dial(dial, time.Duration(2)*time.Second, pingOk)

// 	if err != nil {
// 		t.Fatalf("Unable to start: %v", err)
// 	}

// 	for i := 0; i < 10; i++ {
// 		rstring := uuid.NewV4().String()
// 		cmd := Command{
// 			Timeout:       time.Duration(500) * time.Millisecond,
// 			Prototype:     rstring,
// 			CommandRegexp: regexp.MustCompile(".*"),
// 			Response:      regexp.MustCompile(rstring),
// 			Error:         regexp.MustCompile("a^"),
// 		}
// 		resp := tt.Control(cmd)
// 		if resp.Error != nil {
// 			t.Fatalf("Echo did not properly return: %v", err)
// 		} else {
// 			// fmt.Printf("Got positive response on #%d %s\n", i, resp)
// 		}
// 	}
// 	//send wrongPing
// 	if resp := tt.Control(pingWrong); resp.Error != ErrBytesArgs {
// 		t.Fatalf("Response should have errored with wrong format, but didnt %v", resp)
// 	}

// 	//send pingBad
// 	if resp := tt.Control(pingBad); resp.Error != ErrTimeout {
// 		t.Fatalf("Response should have errored out, but didnt %v", resp)
// 	}

// 	fmt.Println("Starting to Close Socket")
// 	tt.Close()
// 	fmt.Println("Closing Socket")
// }
