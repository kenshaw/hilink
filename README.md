# About hilink

Package hilink is a Go package for working with Huawei Hilink devices (ie,
3G/4G modems and WiFi access devices).

# Installation

Install in the normal way:

```sh
go get -u github.com/knq/hilink
```

# Usage

To use the Go API, please see the full API information on
[GoDoc](http://godoc.org/github.com/knq/hilink).

There is a convenient command line tool, [`hlcli`](cmd/hlcli) that makes
working with the API extremely easy:

```sh
# install hlcli tool
$ go get -u github.com/knq/hilink/hlcli

# display available commands
$ hlcli help

# get help for a subcommand 'smslist'
$ hlcli help smslist
$ hlcli smslist --help

# get network connection information from non-standard API endpoint
$ hlcli networkinfo -endpoint http://192.168.245.1/

# send sms with verbose output
$ hlcli smssend -to='+62....' -msg='your message' -v

# send ussd code with verbose output
$ hlcli ussdcode -code -v
```

# Notes

This was built for interfacing with a Huawei E3370h-153 (specifically a Megafon
M150-2) device, popular in Europe and Asia. It was flashed using a custom
firmware and WebUI that enables the extra features.

Here is the relevant information taken from the API using the
[hinfo](cmd/hinfo) tool:
```sh
$ cd $GOPATH/src/github.com/knq/hilink
$ go build ./cmd/hinfo/ && ./hinfo
{
  "Classify": "hilink",
  "DeviceName": "E3370",
  "HardwareVersion": "CL2E3372HM",
  <<sensitive information omitted>>
  "Msisdn": "",
  "ProductFamily": "LTE",
  "SoftwareVersion": "22.200.09.01.161",
  "WebUIVersion": "17.100.11.00.03-Mod1.0",
  "supportmode": "LTE|WCDMA|GSM",
  "workmode": "LTE"
}
```

# TODO

This API is currently incomplete, as I only have one type of Hilink device to
test with and because I have not attempted to do an exhaustive list of all API
calls. That said, it should be fairly easy to write a new API call by following
the existing code. Pull requests are greatly appreciated, and encouraged!

## Hilink API Resources Available Online
* [Huawei E5186 AJAX API](https://blog.hqcodeshop.fi/archives/259-Huawei-E5186-AJAX-API.html)
* [hilink PHP implementation](https://github.com/BlackyPanther/Huawei-HiLink/blob/master/hilink.class.php)
* [Modemy GSM forum](http://www.bez-kabli.pl/viewtopic.php?t=42168) (in Polish)
