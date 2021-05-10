# About hilink

Package hilink is a Go package for working with Huawei Hilink devices (ie,
3G/4G modems and WiFi access devices).

# Installation

Install in the normal way:

```sh
$ go get -u github.com/kenshaw/hilink
```

# Usage

To use the Go API, please see the full API information on
[Go ref](http://pkg.go.dev/github.com/kenshaw/hilink).

There is a convenient command line tool, [`hlcli`](cmd/hlcli) that makes
working with the API extremely easy:

```sh
# install hlcli tool
$ go get -u github.com/kenshaw/hilink/cmd/hlcli

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
$ cd $GOPATH/src/github.com/kenshaw/hilink
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

# Information on Huawei Stick and Hilink devices

Because the documentation available for the E3372h-153 is so poor, especially
for the version that I have (M150-2, identified by the "HardwareVersion" of
CL2E3372HM), and very little is available in English, I have decided for
posterity to record my notes on installing modified firmware here. These notes
should also work for other Huawei devices, but make sure you are using the
correct firmware for your device! Incorrectly flashing your Hilink device, or
using the wrong firmware will likely brick your device.

## About Stick and Hilink firmware versions

Huawei modems with older firmware (circa 2014 and older) are commonly referred
to as "stick" devices. These devices can sometimes be flashed with the newer
"hilink" firmware versions, however as I only have one hardware revision
available to me, I cannot definitely say which can or cannot be flashed. My
rudimentary investigation leads me to believe that the vast majority of the
Huawei modems on the market can be updated to a Hilink version of the firmware.

For reference, all "stick" firmware starts with "21." and all hilink firmware
starts with "22." While some people have had success with getting the stick
firmware to work properly with Linux, I personally have not been successful
doing that. Additionally, those who have reported getting the stick firmware to
work, have reported significantly lower than 4G speeds.

The primary difference between "stick" and "hilink" firmware, is that the stick
firmware exposes standard serial ports to the host that work as standard
modems. These also have a standard tty interface that one can access and
control the internals. The "hilink" firmware instead provides a standard USB
ethernet device, with its own subnett'd network and exposes a web interface for
controlling the software over the ethernet device. By default, that network
provides standard DHCP and is configured to issue addresses in the range of
192.168.8.100-200.

Note that a hilink device can be put into "debug" mode which then exposes a
number of standard `/dev/ttyUSB*` that can then be used to send AT commands.
This can be done by using this API or by using the `usb_modeswitch` command
line tool. Please see [Go ref](http://pkg.go.dev/github.com/kenshaw/hilink) or
the `usb_modeswitch` manpage.

## USB Modeswitching

Occassionally when a hilink device is plugged in, the Linux kernel/udev does
not recognize it, and does not properly put the device into hilink mode. The
latest versions of `usb_modeswitch` are able to deal with this:
```sh
$ sudo usb_modeswitch -v 12d1 -p 1f01 -V 12d1 -P 14dc -J
```

## Megafon M150-2 (aka E3372h-153, aka hw rev. CL2E3372HM)

The Huawei E3372h-153 device is a 4G LTE USB modem that works with with all
international 4G frequencies. The same hardware is sold as a "stick" model and
a "hilink" model -- please see the comments above. Other companies may
distribute this same hardware version under other names, and will usually be
labeled a E3372h-153 device. I am not sure if the -153 is specific only to
Megafon or not, as I have seen others listed (but have not bought) with that
model number.

If it is sold as a "stick" model (sometimes listed on the packaging as a
E3372s) then it will have the older firmware, and will need to be flashed /
updated. Additionally, the stock firmware is missing many useful features, and
should be flashed to a modified firmware version.

### Flashing a CL2E3372HM with modified firmware

1. Write down the IMEI number listed on the modem (needed if it is running the 'stick' firmware)
2. Boot into Windows and plug the CL2E3372HM into the system
3. Download all the files [here](https://monster.xe-xe.org/files/e3372h/)
4. Extract `MobileBrServ.rar` and as an Administrator, execute the extracted
   `mbbServiceSetup.exe`. Follow the prompts to install (if any)
5. Extract `E3372h-607_Update_21.110.99.02.00.rar` and as an Administrator,
   execute the extracted `E3372h-607_Update_21.110.99.02.00.exe`. Follow the
   prompts. If there is an error, unplug the CL2E3372HM and plug it in again.
   Wait until Windows recognizes the device and try again
6. If the existing version of the firmware is a 'stick' firmware, you will be
   prompted for a flash unlock code. You can use [this tool](https://github.com/knq/huaweihash/tree/master/cmd/huaweicalc)
   to generate an unlock code. Enter the unlock code and continue following the prompts
7. Extract `E3372h-153_Update_22.200.09.01.161_M_AT_01.rar` and as an
   Administrator, execute the extracted `E3372h-153_Update_22.200.09.01.161_M_AT_01.exe`.
   Follow the prompts
8. Extract `Update_WEBUI_17.100.11.00.03_forE3372_Mod1.0.rar` and as an
   Administrator, execute the extracted
   `Update_WEBUI_17.100.11.00.03_forE3372_Mod1.0.exe`. Follow the prompts
