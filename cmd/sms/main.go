package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kenshaw/hilink"
)

func main() {
	endpoint := flag.String("endpoint", "http://192.168.8.1/", "api endpoint")
	debug := flag.Bool("v", false, "enable verbose")
	to := flag.String("to", "", "to")
	msg := flag.String("msg", "", "message")
	list := flag.Bool("list", false, "list sms messages in inbox")
	count := flag.Uint("c", 50, "message count for -list")
	flag.Parse()
	if err := run(context.Background(), *endpoint, *debug, *to, *msg, *list, *count); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, endpoint string, debug bool, to, msg string, list bool, count uint) error {
	// options
	opts := []hilink.ClientOption{
		hilink.WithURL(endpoint),
	}
	if debug {
		opts = append(opts, hilink.WithLogf(log.Printf))
	}
	// create client
	cl := hilink.NewClient(opts...)
	// handle list
	if list {
		return doList(ctx, cl, hilink.SmsBoxTypeInbox, count)
	}
	// check flags
	if msg == "" {
		return errors.New("must specify msg")
	}
	if to == "" {
		return errors.New("must specify to")
	}
	// send sms
	ok, err := cl.SmsSend(ctx, msg, to)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("could not send message")
	}
	fmt.Fprintln(os.Stdout, "message sent")
	return nil
}

// doList lists the sms in the inbox in json format.
func doList(ctx context.Context, cl *hilink.Client, bt hilink.SmsBoxType, count uint) error {
	// get sms counts
	l, err := cl.SmsList(ctx, uint(bt), 1, count, false, false, true)
	if err != nil {
		return err
	}
	// convert to json
	buf, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(buf, '\n'))
	return err
}
