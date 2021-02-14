package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kenshaw/hilink"
)

func main() {
	sleep := flag.Duration("t", 1*time.Second, "sleep duration between ussd api calls")
	endpoint := flag.String("endpoint", "http://192.168.8.1/", "api endpoint")
	debug := flag.Bool("v", false, "enable verbose")
	check := flag.Bool("check", false, "check ussd status")
	code := flag.String("code", "", "ussd code to send")
	noWait := flag.Bool("nowait", false, "exit immediately after sending ussd code")
	release := flag.Bool("r", false, "release ussd session")
	flag.Parse()
	if err := run(context.Background(), *sleep, *endpoint, *debug, *check, *code, *noWait, *release); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, sleep time.Duration, endpoint string, debug, check bool, code string, noWait, release bool) error {
	// options
	opts := []hilink.ClientOption{
		hilink.WithURL(endpoint),
	}
	if debug {
		opts = append(opts, hilink.WithLogf(log.Printf))
	}
	// create client
	cl := hilink.NewClient(opts...)
	if release {
		ok, err := cl.UssdRelease(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if !ok {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "ussd session released\n")
		return nil
	}
	if check {
		v, err := cl.UssdStatus(ctx)
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "received: %v\n", v)
		return nil
	}
	if code == "" {
		return errors.New("no code provided")
	}
	// send ussd code
	ok, err := cl.UssdCode(ctx, code)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("could not send ussd code")
	}
	// bail if not waiting
	if !noWait {
		time.Sleep(sleep)
	}
	time.Sleep(sleep)
	// grab content
	content, err := cl.UssdContent(ctx)
	if err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(content + "\n")
	return err
}
