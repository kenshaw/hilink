package main

//go:generate go run gen.go

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"

	"../../../hilink"
)

func doExit(msg string, args ...interface{}) {
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}

func max(a, b int) int {
	if a >= b {
		return a
	}

	return b
}

type methodList []reflect.Method

func (m methodList) Len() int           { return len(m) }
func (m methodList) Swap(i, j int)      { m[i], m[j] = m[j], m[i] }
func (m methodList) Less(i, j int) bool { return strings.Compare(m[i].Name, m[j].Name) < 0 }

var (
	errorInterface = reflect.TypeOf((*error)(nil)).Elem()
	//textUnmarshalerInterface = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

func findMethodNum(typ reflect.Type, methodName string) int {
	found := false
	methodNum := 0

	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)

		// skip if results != 2 or if last result is not error
		if m.Type.NumOut() != 2 || !m.Type.Out(1).Implements(errorInterface) {
			continue
		}

		// skip "Do" command
		if m.Name == "Do" {
			continue
		}

		if strings.ToLower(methodName) == strings.ToLower(m.Name) {
			methodNum = i
			found = true
		}
	}

	if !found {
		doExit("error: unknown method name")
	}

	return methodNum
}

func doHelpMethodList() {
	typ := reflect.TypeOf(&hilink.Client{})

	maxNameLength := 0

	// process methods
	methods := methodList{}
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		maxNameLength = max(maxNameLength, len(m.Name))
		methods = append(methods, m)
	}

	// sort methods
	sort.Sort(methods)

	str := "usage: " + os.Args[0] + " <method> [<params>]\n\nThe following are the list of available methods:\n\n"
	for i := 0; i < len(methods); i++ {
		m := methods[i]

		// skip if results != 2 or if last result is not error
		if m.Type.NumOut() != 2 || !m.Type.Out(1).Implements(errorInterface) {
			continue
		}

		// skip "Do" command
		if m.Name == "Do" {
			continue
		}

		comment := strings.TrimPrefix(methodCommentMap[m.Name], m.Name+" ")
		comment = strings.TrimSuffix(comment, ".")
		if comment != "" {
			str += "  " + m.Name + strings.Repeat(" ", maxNameLength-len(m.Name)+2) + comment + "\n"
		}
	}

	fmt.Fprintf(os.Stdout, str)
}

func doHelpMethodParams(methodName string) {
	typ := reflect.TypeOf(&hilink.Client{})
	methodNum := findMethodNum(typ, methodName)
	method := typ.Method(methodNum)
	methodTyp := method.Func.Type()

	str := fmt.Sprintf("Params for %s:\n", method.Name)
	str += "  -v                  enable verbose\n  -endpoint=string    api endpoint\n"
	for i := 1; i < methodTyp.NumIn(); i++ {
		p := methodTyp.In(i)
		lastVd := methodTyp.IsVariadic() && i == methodTyp.NumIn()-1

		str += "  -" + methodParamMap[method.Name][i-1]
		if methodTyp.Kind() != reflect.Bool {
			str += "="
			if lastVd {
				str += p.String()[2:]
			} else {
				str += p.String()
			}
			if lastVd {
				str += "..."
			}
		}
		str += "\n"
	}

	fmt.Fprintf(os.Stdout, str)
}

func main() {
	// check argument length
	if len(os.Args) < 2 {
		doHelpMethodList()
		return
	}

	// short circuit for help
	if len(os.Args) == 2 && (os.Args[1] == "help" || os.Args[1] == "list") {
		doHelpMethodList()
		return
	}
	if len(os.Args) == 3 && (os.Args[1] == "help" || os.Args[1] == "list") {
		doHelpMethodParams(os.Args[2])
		return
	}

	// find method
	typ := reflect.TypeOf(&hilink.Client{})
	methodNum := findMethodNum(typ, os.Args[1])
	method := typ.Method(methodNum)

	// create flagset
	fs := flag.NewFlagSet(method.Name, flag.ExitOnError)
	flagDebug := fs.Bool("v", false, "enable verbose")
	flagEndpoint := fs.String("endpoint", "http://192.168.8.1/", "api endpoint")

	isVariadic := method.Type.IsVariadic()

	// add method params to flagset
	in := make([]reflect.Value, method.Type.NumIn())
	for i := 1; i < method.Type.NumIn(); i++ {
		p := method.Type.In(i)
		n := methodParamMap[method.Name][i-1]

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
	opts := []hilink.Option{
		hilink.URL(*flagEndpoint),
	}
	if *flagDebug {
		opts = append(opts, hilink.Log(log.Printf, log.Printf))
	}

	// create client
	client, err := hilink.NewClient(opts...)
	if err != nil {
		doExit("error: %v", err)
	}

	// push client onto params and execute
	in[0] = reflect.ValueOf(client)
	out := method.Func.Call(in)
	if !out[1].IsNil() {
		doExit("error: %v", out[1].Interface())
	}

	// special handling for bool
	if out[0].Type().Kind() == reflect.Bool {
		if out[0].Bool() {
			os.Stdout.WriteString("OK\n")
			return
		} else {
			doExit("failed")
		}
	}

	// json encode and output
	buf, err := json.MarshalIndent(out[0].Interface(), "", "  ")
	if err != nil {
		doExit("error: %v", err)
	}
	os.Stdout.Write(buf)
	os.Stdout.WriteString("\n")
}
