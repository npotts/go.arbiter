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
SOFTWARE.*/

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

/*Command represents a command that can be sent over an Arbitor*/
type Command struct {
	//Name is the basic form of the command, typically without any arugments.  EG, for an
	//on-the-wire command of `WTF403.00\r\n`, the name would be the base form:  WTF
	Name string

	//Timeout is the max time allowed before the command should return a response. If the command
	//take longer than this timeout, the command is to be understood to have failed
	Timeout time.Duration

	//Prototype is the command prototype that is populated by the config file.  It may contain
	//string tokens that will be send to fmt.Sprinft for final formatting to byes to send to device
	// interfaces.
	Prototype string

	//CommandRegexp is the regex that the final command must match before being returned by byes.
	//This works in conjunction with the .Prototype in the following way:
	//	c := fmt.Sprintf(.Prototype, v ... interface{}) #must not contain %!, a sign of too many/few/wrong parameters
	//	CommandRegexp.MatchString(c) #must be true, so values cannot be out of bounds, etc
	CommandRegexp *regexp.Regexp

	//Response is a regexp that should match good/positive/affirmative responses.
	Response *regexp.Regexp

	//Error is a regexp that should match bad/negative/failure responses
	Error *regexp.Regexp

	//Description is a human readable string of a brief explanaition of the commands purpose
	Description string
}

//String implements the Stringer interface
func (c Command) String() string {
	sanitize := func(i interface{}) string {
		var str string
		switch s := i.(type) {
		case *regexp.Regexp:
			if s == nil {
				return "nil"
			}
			str = s.String()
		case string:
			str = s
		}
		return strings.Replace(strings.Replace(str, "\r", "\\r", -1), "\n", "\\n", -1)
	}
	return fmt.Sprintf("%s: %v Prototype:%q CommandRegexp:%q Expect:%q Error:%q", c.Name, c.Timeout, sanitize(c.Prototype), sanitize(c.CommandRegexp), sanitize(c.Response), sanitize(c.Error))
}

//ErrBytesArgs is returned when calling Bytes if any of the following occur:
//	Wrong Number of args (too few / many)
//	Wrong order (ie Command.Prototype is "%s %d" and provided args are '24, "string"'')
//	Wrong types (ie Command.Prototype is "%s" and provided arg is '25')
var ErrBytesArgs = fmt.Errorf("Proper arguments not provided to expand command into bytes")

//ErrBytesFormat is returned when the args used to populate the command are forming an invalid command
var ErrBytesFormat = fmt.Errorf("Formed command does not match allowable format for outgoing commands")

/*Bytes returnes the raw bytes that should be sent to the interface based on the Command.Prototype and
any optional arguments passed to it. It will return a byte slice and one of the following errors:

	ErrBytesArgs if either too many, not enough, or the wrong type of args are provided
	ErrBytesFormat if the assembled byte slice does not match the required Command.CommandRegexp
	nil if a byte slice was successfully formed
*/
func (c Command) Bytes(v ...interface{}) ([]byte, error) {
	str := fmt.Sprintf(c.Prototype, v...)
	if strings.Contains(str, "%!") {
		// fmt.Printf("Arbiter: Malformed command: [%s] with args '%v'! I formed %q, which is incomplete", c, v, str)
		return []byte(str), ErrBytesArgs
	}
	//make sure whatever we stuffed matches the provided regexp
	if !c.CommandRegexp.MatchString(str) {
		// fmt.Printf("Malformed command: [%s] with args '%v'! I formed %q which does not match required regex %q", c, v, str, c.CommandRegexp.String())
		return []byte(str), ErrBytesFormat
	}
	return []byte(str), nil

}

//Commands is map of Command structure where the key should be Command.Name
type Commands map[string]Command

//String implements the Stringer() interface
func (c Commands) String() (r string) {
	for _, val := range c {
		r += fmt.Sprintf("%s\n", val.String())
	}
	return
}

//JSONLabels returns a json array of the stored commands
func (c Commands) JSONLabels() (r string) {
	r = "["
	i := 0
	for lab := range c {
		switch i {
		default:
			r += ","
		case 0:
		}
		i++
		r += fmt.Sprintf("%q", lab)
	}
	r += "]"
	return
}
