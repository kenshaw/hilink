package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/knq/hilink"
)

var (
	flagSleep    = flag.Duration("t", 1*time.Second, "sleep duration between ussd api calls")
	flagEndpoint = flag.String("endpoint", "http://192.168.8.1/", "api endpoint")
	flagDebug    = flag.Bool("v", false, "enable verbose")
	flagCheck    = flag.Bool("check", false, "check ussd status")
	flagCode     = flag.String("code", "", "ussd code to send")
	flagNoWait   = flag.Bool("nowait", false, "exit immediately after sending ussd code")
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
		log.Fatal(err)
	}

	if *flagCheck {
		var v hilink.UssdState
		v, err = doCheck(client)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stdout, "received: %v\n", v)
		return
	}

	if *flagCode == "" {
		fmt.Fprintf(os.Stderr, "error: no code provided\n")
		os.Exit(1)
	}

	// send ussd code
	ok, err := client.UssdCode(*flagCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "error: could not send ussd code\n")
		os.Exit(1)
	}

	// bail if not waiting
	if !*flagNoWait {
		time.Sleep(*flagSleep)
	}

	time.Sleep(*flagSleep)

	// grab content
	content, err := client.UssdContent()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "%s\n", content)
}

func doCheck(client *hilink.Client) (hilink.UssdState, error) {
	return client.UssdStatus()
}
