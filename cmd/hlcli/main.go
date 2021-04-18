package main

//go:generate go run gen.go

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/kenshaw/hilink"
)

func main() {
	switch {
	case len(os.Args) < 2:
		doHelpMethods()
	case len(os.Args) == 2 && (os.Args[1] == "help" || os.Args[1] == "list" || os.Args[1] == "--help"):
		// short circuit for help
		doHelpMethods()
	case len(os.Args) == 3 && (os.Args[1] == "help" || os.Args[1] == "list"):
		doHelpMethodParams(os.Args[2])
	default:
		if err := run(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

func run(ctx context.Context) error {
	// find method
	typ := reflect.TypeOf(&hilink.Client{})
	method, err := findMethod(typ, os.Args[1])
	if err != nil {
		return err
	}
	// create flagset
	fs := flag.NewFlagSet(method.Name, flag.ExitOnError)
	debug := fs.Bool("v", false, "enable verbose")
	endpoint := fs.String("endpoint", "http://192.168.8.1/", "api endpoint")
	isVariadic := method.Type.IsVariadic()
	// add method params to flagset
	in := make([]reflect.Value, method.Type.NumIn())
	for i := 2; i < method.Type.NumIn(); i++ {
		p := method.Type.In(i)
		n := methodParamMap[method.Name][i-2]
		var v interface{}
		switch p.Kind() {
		case reflect.Bool:
			v = fs.Bool(n, false, "")
		case reflect.Int:
			v = fs.Int(n, 0, "")
		case reflect.Uint:
			v = fs.Uint(n, 0, "")
		case reflect.String:
			v = fs.String(n, "", "")
		}
		// special ...string case
		if p.Kind() == reflect.Slice && isVariadic &&
			i == method.Type.NumIn()-1 && reflect.String == p.Elem().Kind() {
			v = fs.String(n, "", "")
		}
		in[i] = reflect.ValueOf(v).Elem()
	}
	// parse flags
	fs.Parse(os.Args[2:])
	// hilink options
	opts := []hilink.ClientOption{
		hilink.WithURL(*endpoint),
	}
	if *debug {
		opts = append(opts, hilink.WithLogf(log.Printf))
	}
	// create client
	cl := hilink.NewClient(opts...)
	// retrieve session id
	sessID, tokID, err := cl.NewSessionAndTokenID(ctx)
	if err != nil {
		return err
	}
	// set session id
	if err := cl.SetSessionAndTokenID(sessID, tokID); err != nil {
		return err
	}
	// push client onto params and execute
	in[0] = reflect.ValueOf(cl)
	in[1] = reflect.ValueOf(ctx)
	out := method.Func.Call(in)
	if !out[1].IsNil() {
		return out[1].Interface().(error)
	}
	// special handling for bool
	if out[0].Type().Kind() == reflect.Bool {
		msg := "SUCCESS"
		if !out[0].Bool() {
			msg = "FAILURE"
		}
		fmt.Fprintln(os.Stdout, msg)
		return nil
	}
	// json encode and output
	buf, err := json.MarshalIndent(out[0].Interface(), "", "  ")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(append(buf, '\n'))
	return err
}

func max(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

func findMethod(typ reflect.Type, methodName string) (reflect.Method, error) {
	found := false
	methodNum := 0
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		// skip if results != 2 or if last result is not error
		if m.Type.NumOut() != 2 || !m.Type.Out(1).Implements(errorInterface) {
			continue
		}
		if strings.ToLower(methodName) == strings.ToLower(m.Name) {
			methodNum = i
			found = true
		}
	}
	if !found {
		return reflect.Method{}, errors.New("unknown method name")
	}
	return typ.Method(methodNum), nil
}

func doHelpMethods() {
	typ := reflect.TypeOf(&hilink.Client{})
	maxNameLength := 0
	// process methods
	var methods []reflect.Method
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		maxNameLength = max(maxNameLength, len(m.Name))
		methods = append(methods, m)
	}
	sort.Slice(methods, func(i, j int) bool {
		return strings.Compare(methods[i].Name, methods[j].Name) < 0
	})
	fmt.Fprintln(os.Stdout, `usage: `+os.Args[0]+` <method> [<params>]
Where <method> is one of the following:
`)
	for i := 0; i < len(methods); i++ {
		m := methods[i]
		// skip if results != 2 or if last result is not error
		if m.Type.NumOut() != 2 || !m.Type.Out(1).Implements(errorInterface) {
			continue
		}
		comment := strings.TrimSuffix(strings.TrimPrefix(methodCommentMap[m.Name], m.Name+" "), ".")
		if comment != "" {
			fmt.Fprintln(os.Stdout, "  "+m.Name+strings.Repeat(" ", maxNameLength-len(m.Name)+2)+comment)
		}
	}
	fmt.Fprintln(os.Stdout, ` 
Note that method names are case-insensitive.
For help regarding the available parameters for a method:
	`+os.Args[0]+` help <method>
`)
}

func doHelpMethodParams(methodName string) {
	typ := reflect.TypeOf(&hilink.Client{})
	method, err := findMethod(typ, methodName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: unknown method %q\n", methodName)
		os.Exit(1)
	}
	methodTyp := method.Func.Type()
	str := fmt.Sprintf("Parameters for %s:\n", method.Name)
	str += "  -v                  enable verbose\n  -endpoint=string    api endpoint\n"
	for i := 2; i < methodTyp.NumIn(); i++ {
		p := methodTyp.In(i)
		lastIsVariadic := methodTyp.IsVariadic() && i == methodTyp.NumIn()-1
		str += "  -" + methodParamMap[method.Name][i-2]
		if methodTyp.Kind() != reflect.Bool {
			str += "=" + strings.TrimPrefix(p.String(), "[]")
			if lastIsVariadic {
				str += "..."
			}
		}
		str += "\n"
	}
	fmt.Fprintf(os.Stdout, str)
}
