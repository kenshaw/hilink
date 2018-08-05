package hilink

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// Option is an option used when creating a new Client.
type Option func(*Client) error

// URL is an option that handles setting the URL on the Client.
func URL(rawurl string) Option {
	return func(c *Client) error {
		var err error

		if !strings.HasSuffix(rawurl, "/") {
			rawurl += "/"
		}

		c.rawurl = rawurl
		c.url, err = url.Parse(rawurl)

		return err
	}
}

// Auth is an option specifying the identifier and password to use.
// The option is ignored if id is an empty string.
func Auth(id, pw string) Option {
	return func(c *Client) error {
		if id != "" {
			c.authID = id
			h := sha256.Sum256([]byte(pw))
			c.authPW = id + base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(h[:])))
		}
		return nil
	}
}

// HTTPClient is an option that allows setting the http.Client used by the
// Client.
func HTTPClient(client *http.Client) Option {
	return func(c *Client) error {
		c.client = client
		return nil
	}
}

// NoSessionStart is an option that prevents the automatic creation of a
// session with the Hilink device.
func NoSessionStart(c *Client) error {
	c.nostart = true
	return nil
}

// httpLogger handles logging http requests and responses.
type httpLogger struct {
	transport                 http.RoundTripper
	requestLogf, responseLogf func(string, ...interface{})
}

func (hl *httpLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	trans := hl.transport
	if trans == nil {
		trans = http.DefaultTransport
	}

	reqBody, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}
	res, err := trans.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resBody, err := httputil.DumpResponse(res, true)
	if err != nil {
		return nil, err
	}

	hl.requestLogf("%s", string(reqBody))
	hl.responseLogf("%s", string(resBody))

	return res, err
}

// Log is an option that writes all HTTP request and response data to the
// respective logger.
func Log(requestLogf, responseLogf func(string, ...interface{})) Option {
	return func(c *Client) error {
		hl := &httpLogger{
			requestLogf:  requestLogf,
			responseLogf: responseLogf,
		}

		// inject as client transport
		c.transport = hl
		if c.client != nil {
			hl.transport = c.client.Transport
			c.client.Transport = hl
		}

		return nil
	}
}
