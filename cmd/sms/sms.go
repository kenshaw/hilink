package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/knq/hilink"
)

var (
	flagEndpoint = flag.String("endpoint", "http://192.168.8.1/", "api endpoint")
	flagDebug    = flag.Bool("v", false, "enable verbose")
	flagTo       = flag.String("to", "", "to")
	flagMsg      = flag.String("msg", "", "message")
	flagList     = flag.Bool("list", false, "list sms messages in inbox")
)

type httpLogger struct {
	RoundTripper http.RoundTripper
}

func (hl *httpLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	reqBody, _ := httputil.DumpRequestOut(req, true)

	res, err := hl.RoundTripper.RoundTrip(req)
	resBody, _ := httputil.DumpResponse(res, true)

	fmt.Println("------------------------------")
	fmt.Printf("%s\n\n", reqBody)
	fmt.Printf("%s", resBody)
	fmt.Println("------------------------------\n\n")

	return res, err
}

func main() {
	flag.Parse()

	// setup logger
	if *flagDebug {
		http.DefaultTransport = &httpLogger{
			RoundTripper: http.DefaultTransport,
		}
	}

	// create client
	client, err := hilink.NewClient(*flagEndpoint)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// handle list
	if *flagList {
		doList(client, hilink.SmsBoxTypeInbox)
		return
	}

	// check flags
	if *flagMsg == "" {
		fmt.Fprintf(os.Stderr, "error: must specify msg\n")
		os.Exit(1)
	}
	if *flagTo == "" {
		fmt.Fprintf(os.Stderr, "error: must specify to\n")
		os.Exit(1)
	}

	// send sms
	b, err := client.SmsSend(*flagMsg, *flagTo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !b {
		fmt.Fprintf(os.Stderr, "could not send message\n")
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "message sent\n")
}

// doList lists the sms in the inbox in json format.
func doList(client *hilink.Client, bt hilink.SmsBoxType) {
	// get sms counts
	c, err := client.SmsCount()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	buf, err := json.Marshal(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(">>> stuff: %s", string(buf))

}
