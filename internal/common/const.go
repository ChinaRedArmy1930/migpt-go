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

const (
	SpecUrl      = "https://miot-spec.org/miot-spec-v2/instance"
	SpecTypesUrl = "https://miot-spec.org/miot-spec-v2/spec/services"
	AllSpecUrl   = SpecUrl + "s?status=all"

	//referance: https://miot-spec.org/miot-spec-v2/spec/services
	IntelligentSpeakerService = "urn:miot-spec-v2:service:intelligent-speaker:0000789B"
	PlayControlService        = "urn:miot-spec-v2:service:play-control:0000781D"

	//referance: https://miot-spec.org/miot-spec-v2/spec/properties
	TextContentProperty = "urn:miot-spec-v2:property:text-content:000000FA"

	//referance: https://miot-spec.org/miot-spec-v2/spec/actions
	WakeUpAction               = "urn:miot-spec-v2:action:wake-up:0000283F"
	PauseAction                = "urn:miot-spec-v2:action:pause:0000280C"
	PlayTextAction             = "urn:miot-spec-v2:action:play-text:00002841"
	PlayRadioAction            = "urn:miot-spec-v2:action:play-radio:00002840"
	PlayMusicAction            = "urn:miot-spec-v2:action:play-music:00002846"
	ExecuteTextDirectiveAction = "urn:miot-spec-v2:action:execute-text-directive:00002842"

	WakeUpWord   = "Wake Up"
	PlayTextWord = "Play Text"
	PauseWord    = "Pause"
	PlayingWord  = "Playing State"
)
