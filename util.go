package hilink

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/clbanning/mxj"
)

var (
	ErrBadStatusCode      = errors.New("bad status code")
	ErrInvalidResponse    = errors.New("invalid response")
	ErrInvalidError       = errors.New("invalid error")
	ErrInvalidValue       = errors.New("invalid value")
	ErrInvalidXML         = errors.New("invalid xml")
	ErrMissingRootElement = errors.New("missing root element")
	ErrMessageTooLong     = errors.New("message too long")
)

// SmsBoxType represents the different inbox types available on a hilink device.
type SmsBoxType uint

const (
	SmsBoxTypeInbox SmsBoxType = iota + 1
	SmsBoxTypeOutbox
	SmsBoxTypeDraft
)

// PinType are the PIN types for a PIN command.
type PinType int

const (
	PinTypeEnter PinType = iota
	PinTypeActivate
	PinTypeDeactivate
	PinTypeChange
	PinTypeEnterPuk
)

// UssdState represents the different USSD states.
type UssdState int

const (
	UssdStateNone UssdState = iota
	UssdStateActive
	UssdStateWaiting
)

// XMLData is a map of XML data to encode/decode.
type XMLData mxj.Map

func xmlPairs(indent string, vals ...string) []byte {
	var buf bytes.Buffer

	// loop over pairs
	for i := 0; i < len(vals); i += 2 {
		buf.WriteString(fmt.Sprintf("%s<%s>%s</%s>\n", indent, vals[i], vals[i+1], vals[i]))
	}

	return buf.Bytes()
}

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

	// make sure we have pairs
	if len(vals)%2 != 0 {
		panic(fmt.Errorf("SimpleRequestXML can only accept pairs of strings, length: %d", len(vals)))
	}

	// write header
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	buf.WriteString("\n<request>\n")

	// add pairs
	buf.Write(xmlPairs("  ", vals...))

	// end string
	buf.WriteString("</request>\n")

	return buf.Bytes()
}

// boolToString converts a bool to a "0" or "1".
func boolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// ErrorCodeMessageMap contains the known message strings for Hilink devices.
//
// see: http://www.bez-kabli.pl/viewtopic.php?t=42168
var ErrorCodeMessageMap = map[string]string{
	"100002": "not supported by firmware or incorrect API path",
	"100003": "no power",
	"100004": "system busy",
	"100005": "unknown error",
	"100006": "invalid parameter",
	"100009": "write error",
	"103002": "unknown error",
	"103015": "unknown error",
	"108001": "invalid username",
	"108002": "invalid password",
	"108003": "user already logged in",
	"108006": "invalid username or password",
	"108007": "invalid username, password, or session limit reached",
	"110024": "battery charge less than 50%",
	"111019": "no network response",
	"111020": "network timeout",
	"111022": "network is not supported",
	"113018": "system busy",
	"114001": "file already exists",
	"114002": "file already exists",
	"114003": "SD card currently in use",
	"114004": "shared path does not exist",
	"114005": "path too long",
	"114006": "no permission for specified file or directory",
	"115001": "unknown error",
	"117001": "incorrect WiFi password",
	"117004": "incorrect WISPr password",
	"120001": "voice busy",
	"125001": "invalid token",
}
