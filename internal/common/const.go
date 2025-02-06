package common

import (
	"regexp"
)

const (
	MiServiceMiIot = "miiot"
	MiServiceMina  = "mina"
)

type Sid string

const (
	MiAccountSIDIO   Sid = "xiaomiio"
	MiAccountSICOAPI Sid = "micoapi"
)

func (s Sid) ToString() string {
	return string(s)
}

type PlayStatus string

const (
	Idel    PlayStatus = "idle"
	Playing PlayStatus = "playing"
	Paused  PlayStatus = "paused"
	Stopped PlayStatus = "stopped"
	Unknown PlayStatus = "unknown"
)

const (
	MiLoginApiUrl       = "https://account.xiaomi.com/pass"
	MinaApiUrl          = "https://api2.mina.mi.com"
	MinaConversationUrl = "https://userprofile.mina.mi.com/device_profile/v2/conversation"
)

const (
	MiLoginApiMethod      = "serviceLogin"
	MiLoginApiMethodAuth2 = "serviceLoginAuth2"

	MinaApiMethodGetDeviceList = "/admin/v2/device_list"
	MinaApiUbus                = "/remote/ubus"
)

var (
	MiAccountResultRegexp = regexp.MustCompile(`^&&&START&&&({.*})$`)
)
