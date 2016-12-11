package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/nerk/hilink"
)

var (
	flagEndpoint = flag.String("endpoint", "http://192.168.8.1/", "api endpoint")
	flagDebug    = flag.Bool("v", false, "enable verbose")
)

func main() {
	var err error

	flag.Parse()

	// options
	opts := []hilink.Option{
		hilink.URL(*flagEndpoint),
	}
	if *flagDebug {
		opts = append(opts, hilink.Log(log.Printf, log.Printf))
	}

	// create client
	client, err := hilink.NewClient(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// get device info
	d, err := client.DeviceInfo()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// change to json
	v, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "%s\n", string(v))
}
