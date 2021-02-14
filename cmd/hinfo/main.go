package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kenshaw/hilink"
)

func main() {
	endpoint := flag.String("endpoint", "http://192.168.8.1/", "api endpoint")
	debug := flag.Bool("v", false, "enable verbose")
	flag.Parse()
	if err := run(context.Background(), *endpoint, *debug); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, endpoint string, debug bool) error {
	// options
	opts := []hilink.ClientOption{
		hilink.WithURL(endpoint),
	}
	if debug {
		opts = append(opts, hilink.WithLogf(log.Printf))
	}
	// create client
	cl := hilink.NewClient(opts...)
	// get device info
	d, err := cl.DeviceInfo(ctx)
	if err != nil {
		return err
	}
	// change to json
	buf, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(buf, '\n'))
	return err
}
