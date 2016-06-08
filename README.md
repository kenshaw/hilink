# About hilink

Package hilink is a Go package for working with Huawei Hilink devices (ie, modems).

# Installation

Install in the normal way:

```sh
go get -u github.com/knq/hilink
```

# Usage

Please see the godoc for api usage.

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
