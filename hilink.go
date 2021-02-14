// Package hilink provides a Hilink WebUI client.
package hilink

import (
	"bytes"
	"strconv"

	"github.com/clbanning/mxj"
)

// Error is the error type.
type Error string

// Error values.
const (
	// ErrBadStatusCode is the bad status code error.
	ErrBadStatusCode Error = "bad status code"
	// ErrInvalidResponse is the invalid response error.
	ErrInvalidResponse Error = "invalid response"
	// ErrInvalidError is the invalid error error.
	ErrInvalidError Error = "invalid error"
	// ErrInvalidValue is the invalid value error.
	ErrInvalidValue Error = "invalid value"
	// ErrInvalidXML is the invalid xml error.
	ErrInvalidXML Error = "invalid xml"
	// ErrMissingRootElement is the missing root element error.
	ErrMissingRootElement Error = "missing root element"
	// ErrMessageTooLong is the message too long error.
	ErrMessageTooLong Error = "message too long"
)

// Error satisfies the error interface.
func (err Error) Error() string {
	return string(err)
}

// SmsBoxType represents the different inbox types available on a hilink
// device.
type SmsBoxType uint

// SmsBoxType values.
const (
	SmsBoxTypeInbox SmsBoxType = iota + 1
	SmsBoxTypeOutbox
	SmsBoxTypeDraft
)

// PinType are the PIN types for a PIN command.
type PinType int

// PinType values.
const (
	PinTypeEnter PinType = iota
	PinTypeActivate
	PinTypeDeactivate
	PinTypeChange
	PinTypeEnterPuk
)

// UssdState represents the different USSD states.
type UssdState int

// UssdState values.
const (
	UssdStateNone UssdState = iota
	UssdStateActive
	UssdStateWaiting
)

// XMLData is a map of XML data to encode/decode.
type XMLData mxj.Map

// SimpleRequestXML creates an XML string from value pairs.
//
// Unfortunately the XML parser (or whatever underyling code) included with the
// WebUI on Hilink devices expects parameters in a specific order. This makes
// packages like mxj or other map based solutions not feasible for use, as Go
// has random key ordering for maps.
//
// On another note, XML sucks.
func SimpleRequestXML(vals ...string) []byte {
	var buf bytes.Buffer
	// write header
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString("\n<request>\n")
	// add pairs
	buf.Write(xmlPairs("  ", vals...))
	// end string
	buf.WriteString("</request>\n")
	return buf.Bytes()
}

// ErrorCodeMap contains the known message strings for Hilink devices.
//
// see: http://www.bez-kabli.pl/viewtopic.php?t=42168
func ErrorCodeMap() map[int]string {
	return map[int]string{
		-1:     "system not available",
		100002: "not supported by firmware or incorrect API path",
		100003: "unauthorized",
		100004: "system busy",
		100005: "unknown error",
		100006: "invalid parameter",
		100009: "write error",
		103002: "unknown error",
		103015: "unknown error",
		108001: "invalid username",
		108002: "invalid password",
		108003: "user already logged in",
		108006: "invalid username or password",
		108007: "invalid username, password, or session timeout",
		110024: "battery charge less than 50%",
		111019: "no network response",
		111020: "network timeout",
		111022: "network not supported",
		113018: "system busy",
		114001: "file already exists",
		114002: "file already exists",
		114003: "SD card currently in use",
		114004: "path does not exist",
		114005: "path too long",
		114006: "no permission for specified file or directory",
		115001: "unknown error",
		117001: "incorrect WiFi password",
		117004: "incorrect WISPr password",
		120001: "voice busy",
		125001: "invalid token",
	}
}

// ErrorMessageFromString returns the error message from a string version of
// the error code.
func ErrorMessageFromString(code string) string {
	m := ErrorCodeMap()
	if c, err := strconv.Atoi(code); err == nil {
		if msg, ok := m[c]; ok {
			return msg
		}
	}
	return m[-1]
}
