package miot

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"migpt-go/internal/common"
	internal "migpt-go/internal/log"
	"migpt-go/mi/base/account"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

type Result struct {
	List []*MiotDevice `json:"list,omitempty"`
}

type MiotDevice struct {
	Did         string `json:"did,omitempty"`
	Token       string `json:"token,omitempty"`
	Longitude   string `json:"longitude,omitempty"`
	Latitude    string `json:"latitude,omitempty"`
	Name        string `json:"name,omitempty"`
	Pid         string `json:"pid,omitempty"`
	Localip     string `json:"localip,omitempty"`
	Mac         string `json:"mac,omitempty"`
	Ssid        string `json:"ssid,omitempty"`
	Bssid       string `json:"bssid,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
	ParentModel string `json:"parent_model,omitempty"`
	ShowMode    int    `json:"show_mode,omitempty"`
	Model       string `json:"model,omitempty"`
	AdminFlag   int    `json:"adminFlag,omitempty"`
	ShareFlag   int    `json:"shareFlag,omitempty"`
	PermitLevel int    `json:"permitLevel,omitempty"`
	IsOnline    bool   `json:"isOnline,omitempty"`
	Desc        string `json:"desc,omitempty"`
	Extra       Extra  `json:"extra,omitempty"`
	Owner       *Owner `json:"owner,omitempty"`
	UID         int64  `json:"uid,omitempty"`
	PdID        int    `json:"pd_id,omitempty"`
	Password    string `json:"password,omitempty"`
	P2pID       string `json:"p2p_id,omitempty"`
	Rssi        int    `json:"rssi,omitempty"`
	FamilyID    int    `json:"family_id,omitempty"`
	ResetFlag   int    `json:"reset_flag,omitempty"`
}

type Extra struct {
	IsSetPincode      int    `json:"isSetPincode,omitempty"`
	PincodeType       int    `json:"pincodeType,omitempty"`
	FwVersion         string `json:"fw_version,omitempty"`
	NeedVerifyCode    int    `json:"needVerifyCode,omitempty"`
	IsPasswordEncrypt int    `json:"isPasswordEncrypt,omitempty"`
	McuVersion        string `json:"mcu_version,omitempty"`
}

type Owner struct {
	UserID   int64  `json:"userid,omitempty"`
	Nickname string `json:"nickname,omitempty"`
	Icon     string `json:"icon,omitempty"`
}

type MiotAccount struct {
	account.MIAccount
	deviceList []*MiotDevice

	controls sync.Map
}

// MiIOTRequest 结构体，用于存储编码后的请求数据
type MiIOTRequest struct {
	Data      map[string]any `json:"data"`
	Signature string         `json:"signature"`
	Nonce     string         `json:"_nonce"`
}

// MiIOTMiAccount 结构体，用于存储账户信息
type MiIOTMiAccount struct {
	UserId       string
	ServiceToken string
	DeviceId     string
	Pass         *MiIOTPass
}

// MiIOTPass 结构体，用于存储认证信息
type MiIOTPass struct {
	CUserId   string
	SSecurity string
}

// Response 结构体，用于解析 API 响应
type Response struct {
	Result any `json:"result"`
}

// signNonce 函数，签名 nonce
func signNonce(ssecurity string, nonce string) string {
	ssecurityBytes, err1 := base64.StdEncoding.DecodeString(ssecurity)
	nonceBytes, err2 := base64.StdEncoding.DecodeString(nonce)
	if err1 != nil || err2 != nil {
		panic("sign failed")
	}

	hash := sha256.New()
	hash.Write([]byte(ssecurityBytes))
	hash.Write([]byte(nonceBytes))

	hashed := hash.Sum(nil)
	return base64.StdEncoding.EncodeToString(hashed)
}

func signMiotParams(uri string, data map[string]any, ssecurity string) map[string]string {
	nonceBytes := make([]byte, 12)
	rand.Read(nonceBytes)
	nonce := base64.StdEncoding.EncodeToString(nonceBytes)
	snonce := signNonce(ssecurity, nonce)
	key, _ := base64.StdEncoding.DecodeString(snonce)
	jsonData, _ := json.Marshal(data)
	msg := strings.Join([]string{uri, snonce, nonce, "data=" + string(jsonData)}, "&")

	h := hmac.New(sha256.New, key)
	h.Write([]byte(msg))
	s := h.Sum(nil)

	return map[string]string{
		"_nonce":    nonce,
		"data":      string(jsonData),
		"signature": base64.StdEncoding.EncodeToString(s),
	}
}

// Control implements IMiot.
func (m *MiotAccount) Control(_ context.Context, c common.ControlFunc) common.ControlFunc {
	return c
}

// Controller implements IMiot.
func (m *MiotAccount) Controller(ctx context.Context, op string, args ...any) (string, error) {
	fn, ok := m.controls.Load(op)
	if !ok {
		return "", fmt.Errorf("%s must register first", op)
	}

	return m.Control(ctx, fn.(common.ControlFunc))(ctx, args...)
}

// GetMiotDevices implements IMiot.
func (m *MiotAccount) GetMiotDevices(ctx context.Context) ([]*MiotDevice, error) {
	resp, err := m.requestMiot(ctx, http.MethodPost, "/home/device_list", "", nil)
	if err != nil {
		internal.GetLogger().Errorf(ctx, "request device failed => %s", err)
		return nil, err
	}

	var r common.Response[*Result]
	err = json.Unmarshal(resp, &r)
	if err != nil {
		internal.GetLogger().Errorf(ctx, "解包失败: %s, 原始数据: %s", err, string(resp))
		return nil, err
	}

	m.deviceList = r.Result.List

	d, _ := json.MarshalIndent(m.deviceList, " ", "	")
	internal.GetLogger().Debugf(ctx, "devicelist: %s \n", string(d))
	return m.deviceList, nil
}

// RegisterControl implements IMiot.
func (m *MiotAccount) RegisterControl(name string, fn common.ControlFunc) {
	m.controls.Store(name, fn)
}

func (m *MiotAccount) requestMiot(ctx context.Context, method, path, device_id string, data map[string]any) ([]byte, error) {
	if m.Sid != common.MiAccountSIDIO {
		return nil, fmt.Errorf("only support %s but %s", common.MiAccountSICOAPI, m.Sid)
	}

	form := url.Values{}
	for k, v := range signMiotParams(path, data, *m.Pass.Ssecurity) {
		form.Add(k, v)
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", "https://api.io.mi.com/app", path), nil)
	if err != nil {
		return nil, err
	}

	req.Body = io.NopCloser(strings.NewReader(form.Encode()))

	req.AddCookie(&http.Cookie{
		Name:     "userId",
		Value:    fmt.Sprintf("%d", m.MIAccount.UserId),
		HttpOnly: true,
	})

	req.AddCookie(&http.Cookie{
		Name:     "serviceToken",
		Value:    m.MIAccount.ServiceToken,
		HttpOnly: true,
	})

	req.AddCookie(&http.Cookie{
		Name:     "PassportDeviceId",
		Value:    device_id,
		HttpOnly: true,
	})

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "iOS-14.4-6.0.103-iPhone12,3--D7744744F7AF32F0544445285880DD63E47D9BE9-8816080-84A3F44E137B71AE-iPhone")
	req.Header.Set("x-xiaomi-protocal-flag-cli", "PROTOCAL-HTTP2")

	/*
		proxyURL, err := url.Parse("http://127.0.0.1:8080") // 替换为你的代理地址
		if err != nil {
			internal.GetLogger().Errorf(ctx, "解析代理地址失败 => %s", err)
			return nil, err
		}
		transport := &http.Transport{
			Proxy:           http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
				client := &http.Client{Transport: transport}
	*/
	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		internal.GetLogger().Errorf(ctx, "解包失败: %s", err)
		return nil, err
	}

	return respBody, nil
}

func (m *MiotAccount) rpcControl(ctx context.Context, device_id, method string, params any) ([]byte, error) {
	return m.requestMiot(ctx, http.MethodPost, "/home/rpc/", device_id, map[string]any{
		"id":     1,
		"method": method,
		"params": params,
	})
}

func (m *MiotAccount) miotSpecControl(ctx context.Context, device_id, commond string, params any) ([]byte, error) {
	t, err := m.requestMiot(ctx, http.MethodPost, fmt.Sprintf("/miotspec/%s", commond), device_id, map[string]any{
		"id":         1,
		"datasource": 2,
		"params":     params,
	})

	return []byte(t), err
}

func (m *MiotAccount) getProperty(ctx context.Context, device_id string, siid, piid int) ([]byte, error) {
	return m.miotSpecControl(ctx, device_id, "prop/get", map[string]any{
		"did":  device_id,
		"siid": siid,
		"piid": piid,
	})
}

func (m *MiotAccount) setProperty(ctx context.Context, device_id string, siid, piid int, value any) ([]byte, error) {
	return m.miotSpecControl(ctx, device_id, "prop/set", map[string]any{
		"did":   device_id,
		"siid":  siid,
		"piid":  piid,
		"value": value,
	})
}

func (m *MiotAccount) action(ctx context.Context, args ...any) (string, error) {
	device_id := args[0].(string)
	for _, x := range args {
		switch x.(type) {
		case int, int64, uint, int8, int32, string:
		default:
			return "", fmt.Errorf("get args failed, got %v", reflect.TypeOf(x))
		}
	}

	t, err := m.miotSpecControl(ctx, device_id, "action", map[string]any{
		"did":  device_id,
		"siid": args[1],
		"aiid": args[2],
		"in":   args[2:],
	})

	return string(t), err
}

func InitMiot(ctx context.Context) IMiot {
	user_id, err := strconv.Atoi(os.Getenv("mi_user_id"))
	if err != nil {
		panic("password required")
	}

	act, err := account.InitMIAccount(ctx,
		account.WithSid(common.MiAccountSIDIO),
		account.WithUserId(int64(user_id)),
	).GetLoginAccount(ctx)
	if err != nil {
		panic("get login token failed")
	}
	iot := &MiotAccount{
		MIAccount: *act,
	}

	iot.RegisterControl("action", iot.action)

	return iot
}
