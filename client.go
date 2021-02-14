package hilink

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/clbanning/mxj/v2"
	"github.com/kenshaw/httplog"
)

// see: https://blog.hqcodeshop.fi/archives/259-Huawei-E5186-AJAX-API.html
// also see: https://github.com/BlackyPanther/Huawei-HiLink/blob/master/hilink.class.php
// also see: http://www.bez-kabli.pl/viewtopic.php?t=42168

const (
	// DefaultURL is the default URL endpoint for the Hilink WebUI.
	DefaultURL = "http://192.168.8.1/"
	// DefaultTimeout is the default timeout.
	DefaultTimeout = 10 * time.Second
	// TokenHeader is the header used by the WebUI for CSRF tokens.
	TokenHeader = "__RequestVerificationToken"
)

// Client represents a Hilink client connection.
type Client struct {
	endpoint  string
	nostart   bool
	started   bool
	authID    string
	authPW    string
	cl        *http.Client
	token     string
	transport http.RoundTripper
	sync.Mutex
}

// NewClient creates a new client a Hilink device.
func NewClient(opts ...ClientOption) *Client {
	// create client
	c := &Client{
		endpoint: DefaultURL,
		cl: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
	// process options
	for _, o := range opts {
		o(c)
	}
	return c
}

// buildRequest creates a request for use with the Client.
func (cl *Client) buildRequest(urlstr string, v interface{}) (*http.Request, error) {
	if v == nil {
		return http.NewRequest("GET", urlstr, nil)
	}
	// encode xml
	body, err := xmlEncode(v)
	if err != nil {
		return nil, err
	}
	// build req
	req, err := http.NewRequest("POST", urlstr, body)
	if err != nil {
		return nil, err
	}
	// set content type and CSRF token
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set(TokenHeader, cl.token)
	return req, nil
}

func (cl *Client) start(ctx context.Context) error {
	cl.Lock()
	defer cl.Unlock()
	if !cl.nostart {
		return nil
	}
	if cl.started {
		return nil
	}
	// retrieve session id
	sessID, tokID, err := cl.NewSessionAndTokenID(ctx)
	if err != nil {
		return err
	}
	// set session id
	if err := cl.SetSessionAndTokenID(sessID, tokID); err != nil {
		return err
	}
	// try login
	if _, err := cl.login(ctx); err != nil {
		return err
	}
	cl.started = true
	return nil
}

// login authentifies the user using the user identifier and password given
// with the Auth option. Return nil if succeeded, or no Auth option
// was given, or the identifier is an empty string.
func (cl *Client) login(ctx context.Context) (bool, error) {
	if cl.authID == "" {
		return false, nil
	}
	// encode hashed password
	h := sha256.Sum256([]byte(cl.authPW + cl.token))
	tokenizedPW := base64.RawStdEncoding.EncodeToString([]byte(hex.EncodeToString(h[:])))
	return cl.doReqCheckOK(ctx, "api/user/login", XMLData{
		"Username":      cl.authID,
		"Password":      tokenizedPW,
		"password_type": 4,
	})
}

// doReq sends a request to the server with the provided path. If data is nil,
// then GET will be used as the HTTP method, otherwise POST will be used.
func (cl *Client) doReq(ctx context.Context, path string, v interface{}, takeFirstEl bool) (interface{}, error) {
	if err := cl.start(ctx); err != nil {
		return nil, err
	}
	cl.Lock()
	defer cl.Unlock()
	// build request
	req, err := cl.buildRequest(cl.endpoint+path, v)
	if err != nil {
		return nil, err
	}
	// do request
	res, err := cl.cl.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	// check status code
	if res.StatusCode != http.StatusOK {
		return nil, ErrBadStatusCode
	}
	// retrieve and save csrf token header
	if tok := res.Header.Get(TokenHeader); tok != "" {
		cl.token = tok
	}
	// read body
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	// decode
	return xmlDecode(body, takeFirstEl)
}

// doReqString wraps a request operation, returning the data of the specified
// child node named elName as a string.
func (cl *Client) doReqString(ctx context.Context, path string, v interface{}, elName string) (string, error) {
	// send request
	res, err := cl.doReq(ctx, path, v, true)
	if err != nil {
		return "", err
	}
	// convert
	d, ok := res.(map[string]interface{})
	if !ok {
		return "", ErrInvalidXML
	}
	l, ok := d[elName]
	if !ok {
		return "", ErrInvalidResponse
	}
	s, ok := l.(string)
	if !ok {
		return "", ErrInvalidValue
	}
	return s, nil
}

// doReqCheckOK wraps a request operation (ie, connect, disconnect, etc),
// checking success via the presence of 'OK' in the XML <response/>.
func (cl *Client) doReqCheckOK(ctx context.Context, path string, v interface{}) (bool, error) {
	res, err := cl.doReq(ctx, path, v, false)
	if err != nil {
		return false, err
	}
	// expect mxj.Map
	m, ok := res.(mxj.Map)
	if !ok {
		return false, ErrInvalidResponse
	}
	// check response present
	o := map[string]interface{}(m)
	r, ok := o["response"]
	if !ok {
		return false, ErrInvalidResponse
	}
	// convert
	s, ok := r.(string)
	if !ok {
		return false, ErrInvalidValue
	}
	return s == "OK", nil
}

// Do sends a request to the server with the provided path. If data is nil,
// then GET will be used as the HTTP method, otherwise POST will be used.
func (cl *Client) Do(ctx context.Context, path string, v interface{}) (XMLData, error) {
	// send request
	res, err := cl.doReq(ctx, path, v, true)
	if err != nil {
		return nil, err
	}
	// convert
	d, ok := res.(map[string]interface{})
	if !ok {
		return nil, ErrInvalidXML
	}
	return d, nil
}

// NewSessionAndTokenID starts a session with the server, and returns the
// session and token.
func (cl *Client) NewSessionAndTokenID(ctx context.Context) (string, string, error) {
	res, err := cl.doReq(ctx, "api/webserver/SesTokInfo", nil, true)
	if err != nil {
		return "", "", err
	}
	// convert
	vals, ok := res.(map[string]interface{})
	if !ok {
		return "", "", ErrInvalidResponse
	}
	// check ses/tokInfo present
	sesInfo, ok := vals["SesInfo"]
	if !ok {
		return "", "", ErrInvalidResponse
	}
	tokInfo, ok := vals["TokInfo"]
	if !ok {
		return "", "", ErrInvalidResponse
	}
	// convert to strings
	s, ok := sesInfo.(string)
	if !ok {
		return "", "", ErrInvalidResponse
	}
	t, ok := tokInfo.(string)
	if !ok {
		return "", "", ErrInvalidResponse
	}
	return strings.TrimPrefix(s, "SessionID="), t, nil
}

// SetSessionAndTokenID sets the sessionID and tokenID for the Client.
func (cl *Client) SetSessionAndTokenID(sessionID, tokenID string) error {
	cl.Lock()
	defer cl.Unlock()
	var err error
	// create cookie jar
	cl.cl.Jar, err = cookiejar.New(nil)
	if err != nil {
		return err
	}
	// set values on client
	u, err := url.Parse(cl.endpoint)
	if err != nil {
		return err
	}
	cl.cl.Jar.SetCookies(u, []*http.Cookie{&http.Cookie{
		Name:  "SessionID",
		Value: sessionID,
	}})
	cl.token = tokenID
	return nil
}

// GlobalConfig retrieves global Hilink configuration.
func (cl *Client) GlobalConfig(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "config/global/config.xml", nil)
}

// NetworkTypes retrieves available network types.
func (cl *Client) NetworkTypes(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "config/global/net-type.xml", nil)
}

// PCAssistantConfig retrieves PC Assistant configuration.
func (cl *Client) PCAssistantConfig(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "config/pcassistant/config.xml", nil)
}

// DeviceConfig retrieves device configuration.
func (cl *Client) DeviceConfig(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "config/deviceinformation/config.xml", nil)
}

// WebUIConfig retrieves WebUI configuration.
func (cl *Client) WebUIConfig(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "config/webuicfg/config.xml", nil)
}

// SmsConfig retrieves device SMS configuration.
func (cl *Client) SmsConfig(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/sms/config", nil)
}

// WlanConfig retrieves basic WLAN settings.
func (cl *Client) WlanConfig(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/wlan/basic-settings", nil)
}

// DhcpConfig retrieves DHCP configuration.
func (cl *Client) DhcpConfig(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/dhcp/settings", nil)
}

// CradleStatusInfo retrieves cradle status information.
func (cl *Client) CradleStatusInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/cradle/status-info", nil)
}

// CradleMACSet sets the MAC address for the cradle.
func (cl *Client) CradleMACSet(ctx context.Context, addr string) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/cradle/current-mac", XMLData{
		"currentmac": addr,
	})
}

// CradleMAC retrieves cradle MAC address.
func (cl *Client) CradleMAC(ctx context.Context) (string, error) {
	return cl.doReqString(ctx, "api/cradle/current-mac", nil, "currentmac")
}

// AutorunVersion retrieves device autorun version.
func (cl *Client) AutorunVersion(ctx context.Context) (string, error) {
	return cl.doReqString(ctx, "api/device/autorun-version", nil, "Version")
}

// DeviceBasicInfo retrieves basic device information.
func (cl *Client) DeviceBasicInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/device/basic_information", nil)
}

// PublicKey retrieves webserver public key.
func (cl *Client) PublicKey(ctx context.Context) (string, error) {
	return cl.doReqString(ctx, "api/webserver/publickey", nil, "encpubkeyn")
}

// DeviceControl sends a control code to the device.
func (cl *Client) DeviceControl(ctx context.Context, code uint) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/device/control", XMLData{
		"Control": fmt.Sprintf("%d", code),
	})
}

// DeviceReboot restarts the device.
func (cl *Client) DeviceReboot(ctx context.Context) (bool, error) {
	return cl.DeviceControl(ctx, 1)
}

// DeviceReset resets the device configuration.
func (cl *Client) DeviceReset(ctx context.Context) (bool, error) {
	return cl.DeviceControl(ctx, 2)
}

// DeviceBackup backups device configuration and retrieves backed up
// configuration data as a base64 encoded string.
func (cl *Client) DeviceBackup(ctx context.Context) (string, error) {
	// cause backup to be generated
	ok, err := cl.DeviceControl(ctx, 3)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("unable to backup device configuration")
	}
	// retrieve data
	// res, err := cl.doReq("nvram.bak")
	return " -- not implemented -- ", nil
}

// DeviceShutdown shuts down the device.
func (cl *Client) DeviceShutdown(ctx context.Context) (bool, error) {
	return cl.DeviceControl(ctx, 4)
}

// DeviceFeatures retrieves device feature information.
func (cl *Client) DeviceFeatures(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/device/device-feature-switch", nil)
}

// DeviceInfo retrieves general device information.
func (cl *Client) DeviceInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/device/information", nil)
}

// DeviceModeSet sets the device mode (0-project, 1-debug).
func (cl *Client) DeviceModeSet(ctx context.Context, mode uint) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/device/mode", XMLData{
		"mode": fmt.Sprintf("%d", mode),
	})
}

// FastbootFeatures retrieves fastboot feature information.
func (cl *Client) FastbootFeatures(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/device/fastbootswitch", nil)
}

// PowerFeatures retrieves power feature information.
func (cl *Client) PowerFeatures(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/device/powersaveswitch", nil)
}

// TetheringFeatures retrieves USB tethering feature information.
func (cl *Client) TetheringFeatures(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/device/usb-tethering-switch", nil)
}

// SignalInfo retrieves network signal information.
func (cl *Client) SignalInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/device/signal", nil)
}

// ConnectionInfo retrieves connection (dialup) information.
func (cl *Client) ConnectionInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/dialup/connection", nil)
}

// GlobalFeatures retrieves global feature information.
func (cl *Client) GlobalFeatures(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/global/module-switch", nil)
}

// Language retrieves current language.
func (cl *Client) Language(ctx context.Context) (string, error) {
	return cl.doReqString(ctx, "api/language/current-language", nil, "CurrentLanguage")
}

// LanguageSet sets the language.
func (cl *Client) LanguageSet(ctx context.Context, lang string) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/language/current-language", XMLData{
		"CurrentLanguage": lang,
	})
}

// NotificationInfo retrieves notification information.
func (cl *Client) NotificationInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/monitoring/check-notifications", nil)
}

// SimInfo retrieves SIM card information.
func (cl *Client) SimInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/monitoring/converged-status", nil)
}

// StatusInfo retrieves general device status information.
func (cl *Client) StatusInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/monitoring/status", nil)
}

// TrafficInfo retrieves traffic statistic information.
func (cl *Client) TrafficInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/monitoring/traffic-statistics", nil)
}

// TrafficClear clears the current traffic statistics.
func (cl *Client) TrafficClear(ctx context.Context) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/monitoring/clear-traffic", XMLData{
		"ClearTraffic": "1",
	})
}

// MonthInfo retrieves the month download statistic information.
func (cl *Client) MonthInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/monitoring/month_statistics", nil)
}

// WlanMonthInfo retrieves the WLAN month download statistic information.
func (cl *Client) WlanMonthInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/monitoring/month_statistics_wlan", nil)
}

// NetworkInfo retrieves network provider information.
func (cl *Client) NetworkInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/net/current-plmn", nil)
}

// WifiFeatures retrieves wifi feature information.
func (cl *Client) WifiFeatures(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/wlan/wifi-feature-switch", nil)
}

// ModeList retrieves available network modes.
func (cl *Client) ModeList(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/net/net-mode-list", nil)
}

// ModeInfo retrieves network mode settings information.
func (cl *Client) ModeInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/net/net-mode", nil)
}

// ModeNetworkInfo retrieves current network mode information.
func (cl *Client) ModeNetworkInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/net/network", nil)
}

// ModeSet sets the network mode.
func (cl *Client) ModeSet(ctx context.Context, netMode, netBand, lteBand string) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/net/net-mode", SimpleRequestXML(
		"NetworkMode", netMode,
		"NetworkBand", netBand,
		"LTEBand", lteBand,
	))
}

// PinInfo retrieves SIM PIN status information.
func (cl *Client) PinInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/pin/status", nil)
}

// doReqPin wraps a SIM PIN manipulation request.
func (cl *Client) doReqPin(ctx context.Context, pt PinType, cur, new, puk string) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/pin/operate", SimpleRequestXML(
		"OperateType", fmt.Sprintf("%d", pt),
		"CurrentPin", cur,
		"NewPin", new,
		"PukCode", puk,
	))
}

// PinEnter enters a SIM PIN.
func (cl *Client) PinEnter(ctx context.Context, pin string) (bool, error) {
	return cl.doReqPin(ctx, PinTypeEnter, pin, "", "")
}

// PinActivate activates a SIM PIN.
func (cl *Client) PinActivate(ctx context.Context, pin string) (bool, error) {
	return cl.doReqPin(ctx, PinTypeActivate, pin, "", "")
}

// PinDeactivate deactivates a SIM PIN.
func (cl *Client) PinDeactivate(ctx context.Context, pin string) (bool, error) {
	return cl.doReqPin(ctx, PinTypeDeactivate, pin, "", "")
}

// PinChange changes a SIM PIN.
func (cl *Client) PinChange(ctx context.Context, pin, new string) (bool, error) {
	return cl.doReqPin(ctx, PinTypeChange, pin, new, "")
}

// PinEnterPuk enters a SIM PIN puk.
func (cl *Client) PinEnterPuk(ctx context.Context, puk, new string) (bool, error) {
	return cl.doReqPin(ctx, PinTypeEnterPuk, new, new, puk)
}

// PinSaveInfo retrieves SIM PIN save information.
func (cl *Client) PinSaveInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/pin/save-pin", nil)
}

// PinSimlockInfo retrieves SIM lock information.
func (cl *Client) PinSimlockInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/pin/simlock", nil)
}

// Connect connects the Hilink device to the network provider.
func (cl *Client) Connect(ctx context.Context) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/dialup/dial", XMLData{
		"Action": "1",
	})
}

// Disconnect disconnects the Hilink device from the network provider.
func (cl *Client) Disconnect(ctx context.Context) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/dialup/dial", XMLData{
		"Action": "0",
	})
}

// ProfileInfo retrieves profile information (ie, APN).
func (cl *Client) ProfileInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/dialup/profiles", nil)
}

// SmsFeatures retrieves SMS feature information.
func (cl *Client) SmsFeatures(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/sms/sms-feature-switch", nil)
}

// SmsList retrieves list of SMS in an inbox.
func (cl *Client) SmsList(ctx context.Context, boxType, page, count uint, sortByName, ascending, unreadPreferred bool) (XMLData, error) {
	// execute request -- note: the order is important!
	return cl.Do(ctx, "api/sms/sms-list", SimpleRequestXML(
		"PageIndex", fmt.Sprintf("%d", page),
		"ReadCount", fmt.Sprintf("%d", count),
		"BoxType", fmt.Sprintf("%d", boxType),
		"SortType", boolToString(sortByName),
		"Ascending", boolToString(ascending),
		"UnreadPreferred", boolToString(unreadPreferred),
	))
}

// SmsCount retrieves count of SMS per inbox type.
func (cl *Client) SmsCount(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/sms/sms-count", nil)
}

// SmsSend sends an SMS.
func (cl *Client) SmsSend(ctx context.Context, msg string, to ...string) (bool, error) {
	if len(msg) >= 160 {
		return false, ErrMessageTooLong
	}
	// build phones
	phones := []string{}
	for _, t := range to {
		phones = append(phones, "Phone", t)
	}
	// send request (order matters below!)
	return cl.doReqCheckOK(ctx, "api/sms/send-sms", SimpleRequestXML(
		"Index", "-1",
		"Phones", "\n"+string(xmlPairs("    ", phones...)),
		"Sca", "",
		"Content", msg,
		"Length", fmt.Sprintf("%d", len(msg)),
		"Reserved", "1",
		"Date", time.Now().Format("2006-01-02 15:04:05"),
	))
}

// SmsSendStatus retrieves SMS send status information.
func (cl *Client) SmsSendStatus(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/sms/send-status", nil)
}

// SmsReadSet sets the read status of a SMS.
func (cl *Client) SmsReadSet(ctx context.Context, id string) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/sms/set-read", SimpleRequestXML(
		"Index", id,
	))
}

// SmsDelete deletes a specified SMS.
func (cl *Client) SmsDelete(ctx context.Context, id uint) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/sms/delete-sms", SimpleRequestXML(
		"Index", fmt.Sprintf("%d", id),
	))
}

// doReqConn wraps a connection manipulation request.
/*func (cl *Client) doReqConn(
	ctx context.Context,
	cm ConnectMode,
	autoReconnect, roamAutoConnect, roamAutoReconnect bool,
	interval, idle int,
) (bool, error) {
	boolToString()
	return cl.doReqCheckOK(ctx, "api/dialup/connection", SimpleRequestXML(
		"RoamAutoConnectEnable", boolToString(roamAutoConnect),
		"AutoReconnect", boolToString(autoReconnect),
		"RoamAutoReconnectEnable", boolToString(roamAutoReconnect),
		"ReconnectInterval", fmt.Sprintf("%d", interval),
		"MaxIdleTime", fmt.Sprintf("%d", idle),
		"ConnectMode", cm.String(),
	))
}*/

// UssdStatus retrieves current USSD session status information.
func (cl *Client) UssdStatus(ctx context.Context) (UssdState, error) {
	s, err := cl.doReqString(ctx, "api/ussd/status", nil, "result")
	if err != nil {
		return UssdStateNone, err
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return UssdStateNone, ErrInvalidResponse
	}
	return UssdState(i), nil
}

// UssdCode sends a USSD code to the Hilink device.
func (cl *Client) UssdCode(ctx context.Context, code string) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/ussd/send", SimpleRequestXML(
		"content", code,
		"codeType", "CodeType",
		"timeout", "",
	))
}

// UssdContent retrieves content buffer of the active USSD session.
func (cl *Client) UssdContent(ctx context.Context) (string, error) {
	return cl.doReqString(ctx, "api/ussd/get", nil, "content")
}

// UssdRelease releases the active USSD session.
func (cl *Client) UssdRelease(ctx context.Context) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/ussd/release", nil)
}

// DdnsList retrieves list of DDNS providers.
func (cl *Client) DdnsList(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/ddns/ddns-list", nil)
}

// LogPath retrieves device log path (URL).
func (cl *Client) LogPath(ctx context.Context) (string, error) {
	return cl.doReqString(ctx, "api/device/compresslogfile", nil, "LogPath")
}

// LogInfo retrieves current log setting information.
func (cl *Client) LogInfo(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/device/logsetting", nil)
}

// PhonebookGroupList retrieves list of the phonebook groups.
func (cl *Client) PhonebookGroupList(ctx context.Context, page, count uint, sortByName, ascending bool) (XMLData, error) {
	return cl.Do(ctx, "api/pb/group-list", SimpleRequestXML(
		"PageIndex", fmt.Sprintf("%d", page),
		"ReadCount", fmt.Sprintf("%d", count),
		"SortType", boolToString(sortByName),
		"Ascending", boolToString(ascending),
	))
}

// PhonebookCount retrieves count of phonebook entries per group.
func (cl *Client) PhonebookCount(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/pb/pb-count", nil)
}

// PhonebookImport imports SIM contacts into specified phonebook group.
func (cl *Client) PhonebookImport(ctx context.Context, group uint) (XMLData, error) {
	return cl.Do(ctx, "api/pb/pb-copySIM", XMLData{
		"GroupID": fmt.Sprintf("%d", group),
	})
}

// PhonebookDelete deletes a specified phonebook entry.
func (cl *Client) PhonebookDelete(ctx context.Context, id uint) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/pb/delete-pb", SimpleRequestXML(
		"Index", fmt.Sprintf("%d", id),
	))
}

// PhonebookList retrieves list of phonebook entries from a specified group.
func (cl *Client) PhonebookList(ctx context.Context, group, page, count uint, sim, sortByName, ascending bool, keyword string) (XMLData, error) {
	// execute request -- note: the order is important!
	return cl.Do(ctx, "api/pb/pb-list", SimpleRequestXML(
		"GroupID", fmt.Sprintf("%d", group),
		"PageIndex", fmt.Sprintf("%d", page),
		"ReadCount", fmt.Sprintf("%d", count),
		"SaveType", boolToString(sim),
		"SortType", boolToString(sortByName),
		"Ascending", boolToString(ascending),
		"KeyWord", keyword,
	))
}

// PhonebookCreate creates a new phonebook entry.
func (cl *Client) PhonebookCreate(ctx context.Context, group uint, name, phone string, sim bool) (XMLData, error) {
	return cl.Do(ctx, "api/pb/pb-new", SimpleRequestXML(
		"GroupID", fmt.Sprintf("%d", group),
		"SaveType", boolToString(sim),
		"Field", xmlNvp("FormattedName", name),
		"Field", xmlNvp("MobilePhone", phone),
		"Field", xmlNvp("HomePhone", ""),
		"Field", xmlNvp("WorkPhone", ""),
		"Field", xmlNvp("WorkEmail", ""),
	))
}

// FirewallFeatures retrieves firewall security feature information.
func (cl *Client) FirewallFeatures(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/security/firewall-switch", nil)
}

// DmzConfig retrieves DMZ status and IP address of DMZ host.
func (cl *Client) DmzConfig(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/security/dmz", nil)
}

// DmzConfigSet enables or disables the DMZ and the DMZ IP address of the
// device.
func (cl *Client) DmzConfigSet(ctx context.Context, enabled bool, dmzIPAddress string) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/security/dmz", SimpleRequestXML(
		"DmzIPAddress", dmzIPAddress,
		"DmzStatus", boolToString(enabled),
	))
}

// SipAlg retrieves status and port of the SIP application-level gateway.
func (cl *Client) SipAlg(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/security/sip", nil)
}

// SipAlgSet enables/disables SIP application-level gateway and sets SIP port.
func (cl *Client) SipAlgSet(ctx context.Context, port uint, enabled bool) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/security/sip", SimpleRequestXML(
		"SipPort", fmt.Sprintf("%d", port),
		"SipStatus", boolToString(enabled),
	))
}

// NatType retrieves NAT type.
func (cl *Client) NatType(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/security/nat", nil)
}

// NatTypeSet sets NAT type (values: 0, 1).
func (cl *Client) NatTypeSet(ctx context.Context, ntype uint) (bool, error) {
	return cl.doReqCheckOK(ctx, "api/security/nat", SimpleRequestXML(
		"NATType", fmt.Sprintf("%d", ntype),
	))
}

// Upnp retrieves the status of UPNP.
func (cl *Client) Upnp(ctx context.Context) (XMLData, error) {
	return cl.Do(ctx, "api/security/upnp", nil)
}

// UpnpSet enables/disables UPNP.
func (cl *Client) UpnpSet(ctx context.Context, enabled bool) (bool, error) {
	return cl.doReqCheckOK(
		ctx,
		"api/security/upnp",
		SimpleRequestXML(
			"UpnpStatus", boolToString(enabled),
		),
	)
}

// TODO:
// UserLogin/UserLogout/UserPasswordChange
//
// WLAN management
// firewall ("security") configuration
// wifi profile management

// CLientOption is a client option.
type ClientOption func(*Client)

// WithURL is a client option to set the URL endpoint.
func WithURL(endpoint string) ClientOption {
	return func(cl *Client) {
		for strings.HasSuffix(endpoint, "/") {
			endpoint = strings.TrimSuffix(endpoint, "/")
		}
		cl.endpoint = endpoint + "/"
	}
}

// WithAuth is a client option specifying the identifier and password to use.
// The option is ignored if id is an empty string.
func WithAuth(id, pw string) ClientOption {
	return func(cl *Client) {
		if id != "" {
			cl.authID = id
			h := sha256.Sum256([]byte(pw))
			cl.authPW = id + base64.StdEncoding.EncodeToString([]byte(hex.EncodeToString(h[:])))
		}
	}
}

// WithNoStart is a client option to disable automatic start.
func WithNoStart(nostart bool) ClientOption {
	return func(cl *Client) {
		cl.nostart = nostart
	}
}

// WithHTTPClient is a client option that sets the underlying http client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(cl *Client) {
		cl.cl = client
	}
}

// WithTransport is a client option that sets the http transport used.
func WithTransport(transport http.RoundTripper) ClientOption {
	return func(cl *Client) {
		cl.cl.Transport = transport
	}
}

// WithLogf is a client option that writes all http request and response data
// to the specified log func.
func WithLogf(logf func(string, ...interface{})) ClientOption {
	return func(cl *Client) {
		cl.cl.Transport = httplog.NewPrefixedRoundTripLogger(cl.cl.Transport, logf)
	}
}

// WithTimeout is a client option that sets the request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(cl *Client) {
		cl.cl.Timeout = timeout
	}
}
