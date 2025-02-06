package mina

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"migpt-go/mi/base/account"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"migpt-go/internal/common"
	internal_log "migpt-go/internal/log"

	"github.com/google/uuid"
)

type MinaAccount struct {
	account.MIAccount
	deviceList []*MinaDevice

	controls sync.Map
}

type MinaDevice struct {
	DeviceID        string          `json:"deviceID,omitempty"`
	SerialNumber    string          `json:"serialNumber,omitempty"`
	Name            string          `json:"name,omitempty"`
	Alias           string          `json:"alias,omitempty"`
	Current         bool            `json:"current,omitempty"`
	Presence        string          `json:"presence,omitempty"` // "offline" | "online"
	Address         string          `json:"address,omitempty"`
	MiotDID         string          `json:"miotDID,omitempty"`
	Hardware        string          `json:"hardware,omitempty"`
	RomVersion      string          `json:"romVersion,omitempty"`
	Capabilities    Capabilities    `json:"capabilities,omitempty"`
	RemoteCtrlType  string          `json:"remoteCtrlType,omitempty"`
	DeviceSNProfile DeviceSNProfile `json:"deviceSNProfile,omitempty"`
	DeviceProfile   DeviceProfile   `json:"deviceProfile,omitempty"`
	BrokerEndpoint  string          `json:"brokerEndpoint,omitempty"`
	BrokerIndex     int             `json:"brokerIndex,omitempty"`
	Mac             string          `json:"mac,omitempty"`
	Ssid            string          `json:"ssid,omitempty"`
}

type DeviceProfile struct {
	Sign     string `json:"sign,omitempty"`
	DeviceID string `json:"deviceId,omitempty"`
}

type DeviceSNProfile struct {
	Signature  string `json:"signature,omitempty"`
	RomVersion string `json:"romVersion,omitempty"`
	Sign       string `json:"sign,omitempty"`
	SN         string `json:"sn,omitempty"`
}

type Capabilities struct {
	ContentBlacklist    int `json:"content_blacklist,omitempty"`
	LanTvControl        int `json:"lan_tv_control,omitempty"`
	NightModeV2         int `json:"night_mode_v2,omitempty"`
	SchoolTimetable     int `json:"school_timetable,omitempty"`
	NightMode           int `json:"night_mode,omitempty"`
	UserNickName        int `json:"user_nick_name,omitempty"`
	PlayerPauseTimer    int `json:"player_pause_timer,omitempty"`
	DialogH5            int `json:"dialog_h5,omitempty"`
	ChildMode2          int `json:"child_mode_2,omitempty"`
	Dlna                int `json:"dlna,omitempty"`
	ReportTimes         int `json:"report_times,omitempty"`
	AiInstruction       int `json:"ai_instruction,omitempty"`
	AlarmVolume         int `json:"alarm_volume,omitempty"`
	ClassifiedAlarm     int `json:"classified_alarm,omitempty"`
	LoadmoreV2          int `json:"loadmore_v2,omitempty"`
	AiProtocol30        int `json:"ai_protocol_3_0,omitempty"`
	NightModeDetail     int `json:"night_mode_detail,omitempty"`
	ChildMode           int `json:"child_mode,omitempty"`
	BabySchedule        int `json:"baby_schedule,omitempty"`
	ToneSetting         int `json:"tone_setting,omitempty"`
	Earthquake          int `json:"earthquake,omitempty"`
	AlarmRepeatOptionV2 int `json:"alarm_repeat_option_v2,omitempty"`
	XiaomiVoip          int `json:"xiaomi_voip,omitempty"`
	NearbyWakeupCloud   int `json:"nearby_wakeup_cloud,omitempty"`
	FamilyVoice         int `json:"family_voice,omitempty"`
	BluetoothOptionV2   int `json:"bluetooth_option_v2,omitempty"`
	SkillTry            int `json:"skill_try,omitempty"`
	Yueyu               int `json:"yueyu,omitempty"`
	Yunduantts          int `json:"yunduantts,omitempty"`
	MicoCurrent         int `json:"mico_current,omitempty"`
	CpLevel             int `json:"cp_level,omitempty"`
	VoipUsedTime        int `json:"voip_used_time,omitempty"`
}

// AnswerLLM 结构体，表示 LLM 文本回应
type AnswerLLM struct {
	BitSet [4]int64 `json:"bitSet,omitempty"`
	Type   string   `json:"type"`
	LLM    struct {
		BitSet [2]int64 `json:"bitSet,omitempty"`
		Text   string   `json:"text,omitempty"`
	} `json:"llm,omitempty"`
}

// AnswerTTS 结构体，表示 TTS 文本回应
type AnswerTTS struct {
	BitSet [4]int64 `json:"bitSet,omitempty"`
	Type   string   `json:"type"`
	TTS    struct {
		BitSet [2]int64 `json:"bitSet,omitempty"`
		Text   string   `json:"text,omitempty"`
	} `json:"tts,omitempty"`
}

// AudioInfo 结构体，表示音频信息
type AudioInfo struct {
	BitSet [4]int64 `json:"bitSet,omitempty"`
	Title  string   `json:"title,omitempty"`
	Artist string   `json:"artist,omitempty"`
	CpName string   `json:"cpName,omitempty"`
}

// AnswerAudio 结构体，表示音乐播放列表
type AnswerAudio struct {
	BitSet [4]int64 `json:"bitSet,omitempty"`
	Type   string   `json:"type"`
	Audio  struct {
		BitSet        [2]int64    `json:"bitSet,omitempty"`
		AudioInfoList []AudioInfo `json:"audioInfoList,omitempty"`
	} `json:"audio,omitempty"`
}

type Answer interface {
	Allow()
}

func (llm AnswerLLM) Allow()   {}
func (llm AnswerTTS) Allow()   {}
func (llm AnswerAudio) Allow() {}

// Record 结构体，表示 records 数组中的每个元素
type Record struct {
	BitSet    [5]int64 `json:"bitSet,omitempty"`
	Answers   []Answer `json:"answers,omitempty"`
	Time      int64    `json:"time,omitempty"`
	Query     string   `json:"query,omitempty"`
	RequestID string   `json:"requestId,omitempty"`
}

// MiConversations 结构体
type MiConversation struct {
	BitSet      [3]int64 `json:"bitSet,omitempty"`
	Records     []Record `json:"records,omitempty"`
	NextEndTime int64    `json:"nextEndTime,omitempty"`
}

func (d *MinaDevice) UnmarshalJSON(data []byte) error {
	type Alias MinaDevice
	aux := &struct {
		DeviceSNProfile string `json:"deviceSNProfile"`
		DeviceProfile   string `json:"deviceProfile"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		internal_log.GetLogger().Warnf(context.TODO(), "decode MinaDevice failed, %s", err)
		return err
	}

	// Decode Base64 field for DeviceSNProfile
	decodedSNProfile, err := base64.StdEncoding.DecodeString(aux.DeviceSNProfile)
	if err != nil {
		internal_log.GetLogger().Warnf(context.TODO(), "decode deviceSNProfile failed, %s", err)
		return err
	}

	// Unmarshal the decoded JSON into the DeviceSNProfile struct
	if err := json.Unmarshal(decodedSNProfile, &d.DeviceSNProfile); err != nil {
		internal_log.GetLogger().Warnf(context.TODO(), "decode deviceSNProfile failed, %s", err)
		return err
	}

	// Decode Base64 field for DeviceProfile
	decodedDeviceProfile, err := base64.StdEncoding.DecodeString(aux.DeviceProfile)
	if err != nil {
		internal_log.GetLogger().Warnf(context.TODO(), "decode DeviceProfile failed, %s", err)
		return err
	}

	// Unmarshal the decoded JSON into the DeviceInfo struct
	if err := json.Unmarshal(decodedDeviceProfile, &d.DeviceProfile); err != nil {
		internal_log.GetLogger().Warnf(context.TODO(), "decode DeviceProfile failed, %s", err)
		return err
	}

	return nil
}

// GetMiDeviceList implements IMina.
func (m *MinaAccount) GetMiDeviceList(ctx context.Context) ([]*MinaDevice, error) {
	if m.Sid != common.MiAccountSICOAPI {
		return nil, fmt.Errorf("only support %s but %s", common.MiAccountSICOAPI, m.Sid)
	}

	params := url.Values{}
	requestUrl := fmt.Sprintf("%s/%s?%s", common.MinaApiUrl, common.MinaApiMethodGetDeviceList, params.Encode())

	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "请求失败: %s", err)
		return nil, err
	}

	req.AddCookie(&http.Cookie{Name: "userId", Value: strconv.Itoa(int(m.UserId))})
	req.AddCookie(&http.Cookie{Name: "serviceToken", Value: m.ServiceToken})

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "请求失败: %s", err)
		return nil, err
	}
	defer response.Body.Close()

	resp, err := io.ReadAll(response.Body)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "解包失败: %s", err)
		return nil, err
	}

	var r common.Response[[]*MinaDevice]
	err = json.Unmarshal(resp, &r)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "解包失败: %s, 原始数据: %s", err, string(resp))
		return nil, err
	}

	m.deviceList = r.Data

	d, _ := json.MarshalIndent(r.Data, " ", "    ")
	internal_log.GetLogger().Debugf(ctx, "devicelist => %s", string(d))

	return r.Data, nil
}

// GetMiDeviceStatus implements IMina.
func (m *MinaAccount) GetMiDeviceStatus(ctx context.Context, deviceId string) (common.PlayStatus, error) {
	if m.Sid != common.MiAccountSICOAPI {
		return common.Unknown, fmt.Errorf("only support %s but %s", common.MiAccountSICOAPI, m.Sid)
	}

	respBody, err := m.requestMinaUbus(ctx, "mediaplayer", "player_get_play_status", "{}", deviceId)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "Error requestMina: %s", err)
		return common.Unknown, err
	}
	internal_log.GetLogger().Infof(ctx, string(respBody))

	return common.Unknown, nil
}

// GetUserConversations implements IMina.
func (m *MinaAccount) GetUserConversations(ctx context.Context, limit int, timestamp int64, hardware string, deviceId string) (*MiConversation, error) {
	if m.Sid != common.MiAccountSICOAPI {
		return nil, fmt.Errorf("only support %s but %s", common.MiAccountSICOAPI, m.Sid)
	}
	client := &http.Client{}

	data := url.Values{}
	data.Add("limit", strconv.Itoa(limit))
	data.Add("timestamp", fmt.Sprintf("%d", timestamp))
	data.Add("requestId", uuid.NewString())
	data.Add("hardware", hardware)

	requestUrl := fmt.Sprintf("%s?%s", common.MinaConversationUrl, data.Encode())
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "Error creating request: %s", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.AddCookie(&http.Cookie{Name: "userId", Value: strconv.Itoa(int(m.UserId))})
	req.AddCookie(&http.Cookie{Name: "serviceToken", Value: m.ServiceToken})
	req.AddCookie(&http.Cookie{Name: "deviceId", Value: deviceId})
	resp, err := client.Do(req)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "Error sending request: %s", err)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "解包失败: %s", err)
		return nil, err
	}

	var r common.Response[string]
	err = json.Unmarshal(respBody, &r)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "解包失败: %s, 原始数据: %s", err, string(respBody))
		return nil, err
	}

	conversation := &MiConversation{}
	internal_log.GetLogger().Debugf(ctx, "r.Data: %s \n", string(r.Data))
	err = json.Unmarshal([]byte(r.Data), &conversation)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "解包失败: %s, 原始数据: %s", err, string(respBody))
		return nil, err
	}

	d, _ := json.MarshalIndent(conversation, " ", "	")
	internal_log.GetLogger().Debugf(ctx, "conversations: %s \n", string(d))

	return conversation, nil
}

// SetMiDeviceVolume implements IMina.
func (m *MinaAccount) GetSystemBoard(ctx context.Context) error {
	//deviceId:  "7d6d4d83-734f-4490-94ed-5035cec3ffa8"
	respBody, err := m.requestMinaUbus(ctx, "system", "info", `{}`, "7d6d4d83-734f-4490-94ed-5035cec3ffa8")
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "Error requestMina: %s", err)
		return err
	}
	internal_log.GetLogger().Infof(ctx, string(respBody))
	return nil
}

// Control implements IMina.
func (m *MinaAccount) Control(ctx context.Context, c common.ControlFunc) common.ControlFunc {
	return c
}

// Controller implements IMina.
func (m *MinaAccount) Controller(ctx context.Context, op string, args ...interface{}) (string, error) {
	fn, ok := m.controls.Load(op)
	if !ok {
		return "", fmt.Errorf("%s must register first", op)
	}

	return m.Control(ctx, fn.(common.ControlFunc))(ctx, args)
}

// RegisterControl implements IMina.
func (m *MinaAccount) RegisterControl(name string, fn common.ControlFunc) {
	m.controls.Store(name, fn)
}

// https://openwrt.org/docs/techref/ubus
// 可以通过 sudo docker run -it openwrtorg/rootfs:x86-64 "ubus" 看 ubus的相关指令
// 此处猜测服务端调用了 ubus call
func (m *MinaAccount) requestMinaUbus(ctx context.Context, path, method, message, deviceId string) ([]byte, error) {
	if m.Sid != common.MiAccountSICOAPI {
		return nil, fmt.Errorf("only support %s but %s", common.MiAccountSICOAPI, m.Sid)
	}
	client := &http.Client{}

	data := url.Values{}
	data.Add("deviceId", deviceId)
	data.Add("path", path)
	data.Add("method", method)
	data.Add("message", message)

	requestUrl := fmt.Sprintf("%s%s?%s", common.MinaApiUrl, common.MinaApiUbus, data.Encode())
	internal_log.GetLogger().Infof(ctx, "creating request: %s", requestUrl)
	req, err := http.NewRequest("POST", requestUrl, nil)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "Error creating request: %s", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.AddCookie(&http.Cookie{Name: "userId", Value: strconv.Itoa(int(m.UserId))})
	req.AddCookie(&http.Cookie{Name: "serviceToken", Value: m.ServiceToken})

	resp, err := client.Do(req)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "Error sending request: %s", err)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "解包失败: %s", err)
		return nil, err
	}

	return respBody, nil
}

func (m *MinaAccount) pause(ctx context.Context, args ...interface{}) (string, error) {
	deviceId := args[0].(string)
	d, _ := json.Marshal(map[string]interface{}{"action": "pause"})
	resp, err := m.requestMinaUbus(ctx, "mediaplayer", "player_play_operation", string(d), deviceId)
	if err != nil {
		return "", err
	}

	return string(resp), nil
}

func (m *MinaAccount) play(ctx context.Context, args ...interface{}) (string, error) {
	if len(args) != 3 {
		panic("play args must 3")
	}

	var resp []byte
	var err error
	tts := args[0].(string)
	url := args[1].(string)
	deviceId := args[2].(string)
	switch {
	case tts != "":
		{
			d, _ := json.Marshal(map[string]interface{}{"text": tts, "save": 0})
			resp, err = m.requestMinaUbus(ctx, "mibrain", "text_to_speech", string(d), deviceId)
		}
	case url != "":
		{
			d, _ := json.Marshal(map[string]interface{}{"url": url, "type": 1})
			resp, err = m.requestMinaUbus(ctx, "mediaplayer", "player_play_url", string(d), deviceId)
		}
	default:
		{
			d, _ := json.Marshal(map[string]interface{}{"action": "play"})
			resp, err = m.requestMinaUbus(ctx, "mediaplayer", "player_play_operation", string(d), deviceId)
		}
	}

	if err != nil {
		return "", err
	}

	return string(resp), nil
}

func (m *MinaAccount) toggle(ctx context.Context, args ...interface{}) (string, error) {
	deviceId := args[0].(string)
	d, _ := json.Marshal(map[string]interface{}{"action": "toggle"})
	resp, err := m.requestMinaUbus(ctx, "mediaplayer", "player_play_operation", string(d), deviceId)
	if err != nil {
		return "", err
	}

	return string(resp), nil
}

func (m *MinaAccount) stop(ctx context.Context, args ...interface{}) (string, error) {
	deviceId := args[0].(string)
	d, _ := json.Marshal(map[string]interface{}{"action": "stop"})
	resp, err := m.requestMinaUbus(ctx, "mediaplayer", "player_play_operation", string(d), deviceId)
	if err != nil {
		return "", err
	}

	return string(resp), nil
}

func (m *MinaAccount) player_set_volume(ctx context.Context, args ...interface{}) (string, error) {
	deviceId := args[0].(string)
	volume := args[1].(int64)
	d, _ := json.Marshal(map[string]interface{}{"volume": volume})
	resp, err := m.requestMinaUbus(ctx, "mediaplayer", "player_set_volume", string(d), deviceId)
	if err != nil {
		return "", err
	}

	return string(resp), nil
}

func (m *MinaAccount) player_get_play_status(ctx context.Context, args ...interface{}) (string, error) {
	deviceId := args[0].(string)
	resp, err := m.requestMinaUbus(ctx, "mediaplayer", "player_get_play_status", `{}`, deviceId)
	if err != nil {
		return "", err
	}

	return string(resp), nil
}

func InitMina(ctx context.Context) IMina {
	user_id, err := strconv.Atoi(os.Getenv("mi_user_id"))
	if err != nil {
		panic("password required")
	}

	act, err := account.InitMIAccount(ctx,
		account.WithSid(common.MiAccountSICOAPI),
		account.WithUserId(int64(user_id)),
		account.WithDid(os.Getenv("mi_did")),
	).GetLoginAccount(ctx)
	if err != nil {
		panic("get login token failed")
	}

	mina := &MinaAccount{
		MIAccount: *act,
	}

	mina.RegisterControl("pause", mina.pause)
	mina.RegisterControl("play", mina.play)
	mina.RegisterControl("player_set_volume", mina.player_set_volume)
	mina.RegisterControl("toggle", mina.toggle)
	mina.RegisterControl("stop", mina.stop)
	mina.RegisterControl("player_get_play_status", mina.player_get_play_status)

	return mina
}
