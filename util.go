package hilink

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/clbanning/mxj"
)

var (
	// ErrBadStatusCode is the bad status code error.
	ErrBadStatusCode = errors.New("bad status code")

	// ErrInvalidResponse is the invalid response error.
	ErrInvalidResponse = errors.New("invalid response")

	// ErrInvalidError is the invalid error error.
	ErrInvalidError = errors.New("invalid error")

	// ErrInvalidValue is the invalid value error.
	ErrInvalidValue = errors.New("invalid value")

	// ErrInvalidXML is the invalid xml error.
	ErrInvalidXML = errors.New("invalid xml")

	// ErrMissingRootElement is the missing root element error.
	ErrMissingRootElement = errors.New("missing root element")

	// ErrMessageTooLong is the message too long error.
	ErrMessageTooLong = errors.New("message too long")
)

// SmsBoxType represents the different inbox types available on a hilink device.
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

// xmlPairs combines xml name/value pairs as a properly formatted XML buffer.
func xmlPairs(indent string, vals ...string) []byte {
	// make sure we have pairs
	if len(vals)%2 != 0 {
		panic(fmt.Errorf("xmlPairs can only accept pairs of strings, length: %d", len(vals)))
	}

	var buf bytes.Buffer

	// loop over pairs
	for i := 0; i < len(vals); i += 2 {
		buf.WriteString(fmt.Sprintf("%s<%s>%s</%s>\n", indent, vals[i], vals[i+1], vals[i]))
	}

	return buf.Bytes()
}

// xmlPairsString builds a string of XML pairs.
func xmlPairsString(indent string, vals ...string) string {
	return string(xmlPairs(indent, vals...))
}

// xmlNvp (ie, name value pair) builds a <Name>name</Name><Value>value</Value> XML pair.
func xmlNvp(name, value string) string {
	return xmlPairsString("", "Name", name, "Value", value)
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
	"-1":     "system not available",
	"100002": "not supported by firmware or incorrect API path",
	"100003": "access denied",
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
	"108007": "invalid username, password, or session timeout",
	"110024": "battery charge less than 50%",
	"111019": "no network response",
	"111020": "network timeout",
	"111022": "network not supported",
	"113018": "system busy",
	"114001": "file already exists",
	"114002": "file already exists",
	"114003": "SD card currently in use",
	"114004": "path does not exist",
	"114005": "path too long",
	"114006": "no permission for specified file or directory",
	"115001": "unknown error",
	"117001": "incorrect WiFi password",
	"117004": "incorrect WISPr password",
	"120001": "voice busy",
	"125001": "invalid token",
}

// encodeXML encodes a map to standard XML values.
func encodeXML(v interface{}) (io.Reader, error) {
	var err error
	var buf []byte

	switch x := v.(type) {
	case []byte:
		buf = x

	case XMLData:
		// wrap in request element
		m := mxj.Map(map[string]interface{}{
			"request": map[string]interface{}(x),
		})

		// encode xml
		buf, err = m.XmlIndent("", "  ")
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.New("unsupported type in encodeXML")
	}

	return bytes.NewReader(buf), nil
}

// decodeXML decodes buf into its simple xml values.
func decodeXML(buf []byte, takeFirstEl bool) (interface{}, error) {
	// decode xml
	m, err := mxj.NewMapXml(buf)
	if err != nil {
		return nil, err
	}

	// check if error was returned
	if e, ok := m["error"]; ok {
		z, ok := e.(map[string]interface{})
		if !ok {
			return nil, ErrInvalidError
		}

		// grab message if not passed by the api
		msg, _ := z["message"].(string)
		if msg == "" {
			c, _ := z["code"].(string)
			msg = ErrorCodeMessageMap[c]
		}

		return nil, fmt.Errorf("hilink error %v: %s", z["code"], msg)
	}

	// check there is only one element
	if len(m) != 1 {
		return nil, ErrMissingRootElement
	}

	// bail if not grabbing the first XML element
	if !takeFirstEl {
		return m, nil
	}

	// grab root element
	rootEl := ""
	for k := range m {
		rootEl = k
	}
	r, ok := m[rootEl]
	if !ok {
		return nil, ErrInvalidResponse
	}

	// convert
	t, ok := r.(map[string]interface{})
	if !ok {
		return nil, ErrInvalidXML
	}

	return t, nil
}
