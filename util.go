package hilink

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/clbanning/mxj"
)

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

// boolToString converts a bool to a "0" or "1".
func boolToString(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// xmlEncode encodes a map to standard XML values.
func xmlEncode(v interface{}) (io.Reader, error) {
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
		return nil, errors.New("unsupported type in xmlEncode")
	}
	return bytes.NewReader(buf), nil
}

// xmlDecode decodes buf into its simple xml values.
func xmlDecode(buf []byte, takeFirstEl bool) (interface{}, error) {
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
			msg = ErrorMessageFromString(c)
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
