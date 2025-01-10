package account

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"migpt-go/internal/common"
	internal_log "migpt-go/internal/log"
	internal_sync "migpt-go/internal/sync"
)

type Option func(*MIAccount)

type MiPass struct {
	Qs        *string `json:"qs,omitempty"`
	Sign      *string `json:"_sign,omitempty"`
	Callback  *string `json:"callback,omitempty"`
	Location  *string `json:"location,omitempty"`
	Ssecurity *string `json:"ssecurity,omitempty"`
	PassToken *string `json:"passToken,omitempty"`
	Nonce     *int64  `json:"nonce,omitempty"`
	UserID    *int64  `json:"userId,omitempty"`
	CUserID   *string `json:"cUserId,omitempty"`
	PSecurity *string `json:"psecurity,omitempty"`
}

type MIAccount struct {
	Sid          common.Sid `json:"sid,omitempty"`
	UserId       int64      `json:"userId,omitempty"`
	Pass         *MiPass    `json:"pass,omitempty"`
	ServiceToken string     `json:"service_token,omitempty"`
	Did          string     `json:"did,omitempty"`
	pubSub       *internal_sync.PubSub
}

type MiAccountResponse struct {
	MiPass

	ServiceParam    string  `json:"serviceParam,omitempty"`
	Code            int     `json:"code,omitempty"`
	Description     string  `json:"description,omitempty"`
	SecurityStatus  int     `json:"securityStatus,omitempty"`
	Sid             string  `json:"sid,omitempty"`
	Result          string  `json:"result,omitempty"`
	CaptchaUrl      *string `json:"captchaUrl,omitempty"` // Use *string to handle null values
	Pwd             int     `json:"pwd,omitempty"`
	Child           int     `json:"child,omitempty"`
	Desc            string  `json:"desc,omitempty"`
	NotificationUrl string  `json:"notificationUrl,omitempty"`
}

func (m *MIAccount) OnMessage(msg any) {
	m.GetLoginAccount(msg.(context.Context))
}

func WithSid(sid common.Sid) Option {
	return func(m *MIAccount) {
		m.Sid = sid
	}
}

func WithUserId(userId int64) Option {
	return func(m *MIAccount) {
		m.UserId = userId
	}
}

func WithDid(did string) Option {
	return func(m *MIAccount) {
		m.Did = did
	}
}

// TakeLoginToken implements IMIAccount.
func (m *MIAccount) TakeLoginToken(ctx context.Context) error {
	nonce := int64(0)
	location := ""
	ssecurity := ""
	if m.Pass.Nonce == nil || m.Pass.Location == nil || m.Pass.Ssecurity == nil {
		_, err := m.GetLoginAccount(ctx)
		if err != nil {
			panic(fmt.Errorf("登录失败 => %s", err))
		}
	}

	nonce = *m.Pass.Nonce
	location = *m.Pass.Location
	ssecurity = *m.Pass.Ssecurity

	data := fmt.Sprintf("nonce=%d&%s", nonce, ssecurity)
	hash := sha1.New()
	hash.Write([]byte(data))
	clientSign := base64.StdEncoding.EncodeToString(hash.Sum(nil))

	reqUrl, err := url.Parse(location)
	if err != nil {
		panic(fmt.Errorf("登录失败 => %s", err))
	}

	query := reqUrl.Query()
	query.Set("_userIdNeedEncrypt", "true")
	query.Set("clientSign", clientSign)
	reqUrl.RawQuery = query.Encode()

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", reqUrl.String(), nil)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "Error creating request: %s", err)
		return err
	}

	// 发送 HTTP 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "Error sending request: %s", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		internal_log.GetLogger().Errorf(ctx, "httpcode not 200 but %d", resp.StatusCode)
		return errors.New("take login token failed")
	}

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "serviceToken" {
			m.ServiceToken = cookie.Value
			break
		}
	}

	if m.ServiceToken == "" {
		internal_log.GetLogger().Errorf(ctx, "获取mi serviceToken 失败了")
		return errors.New("get mi serviceToken faield")
	}

	return nil
}

// RefreshLoginToken implements IMIAccount.
func (m *MIAccount) RefreshLoginToken(ctx context.Context) error {
	if m.Pass.PassToken == nil {
		return errors.New("请先调用 GetLoginAccount")
	}

	ticker := time.NewTicker(time.Second * 3)

	defer func() {
		ticker.Stop()
	}()

	b := true
	for b {
		select {
		case <-ticker.C:
			m.pubSub.Publish(ctx)
		case <-m.pubSub.Done():
			b = false
		}
	}

	return nil
}

// ServiceLogin implements IMIAccount.
func (m *MIAccount) ServiceLogin(ctx context.Context) (*MiAccountResponse, error) {
	var accountResponse MiAccountResponse
	params := url.Values{}
	params.Add("sid", string(m.Sid))
	params.Add("_json", "1")
	params.Add("_locale", "zh_CN")
	requestUrl := fmt.Sprintf("%s/%s?%s", common.MiLoginApiUrl, common.MiLoginApiMethod, params.Encode())
	req, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "请求失败: %s", err)
		return &accountResponse, err
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "请求失败: %s", err)
		return &accountResponse, err
	}
	defer response.Body.Close()

	resp, err := io.ReadAll(response.Body)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "解包失败: %s", err)
		return &accountResponse, err
	}

	internal_log.GetLogger().Infof(ctx, "response: %s", string(resp))
	matchs := common.MiAccountResultRegexp.FindStringSubmatch(string(resp))
	if len(matchs) != 2 {
		internal_log.GetLogger().Errorf(ctx, "解包失败 => %s", string(resp))
		return &accountResponse, errors.New("解包失败")
	}

	err = json.Unmarshal([]byte(matchs[1]), &accountResponse)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "解包失败 => %s,原始数据 => %s", err, matchs[1])
		return &accountResponse, err
	}

	if accountResponse.Code != 0 {
		internal_log.GetLogger().Errorf(ctx, "登录失败 => %s ,原始数据 => %s", err, matchs[1])
		return &accountResponse, errors.New("登录失败")
	}

	return &accountResponse, nil

}

func (m *MIAccount) GetLoginAccount(ctx context.Context) (*MIAccount, error) {
	accountResponse, err := m.ServiceLogin(ctx)
	if err == nil {
		return m, nil
	}

	if accountResponse == nil {
		return nil, errors.New("登录失败")
	}

	data := url.Values{}
	data.Set("_json", "true")
	data.Set("qs", *accountResponse.Qs)
	data.Set("sid", m.Sid.ToString())
	data.Set("_sign", *accountResponse.Sign)
	data.Set("callback", *accountResponse.Callback)
	data.Set("user", strconv.Itoa(int(m.UserId)))
	data.Set("hash", func() string {
		hash := md5.New()
		hash.Write([]byte(os.Getenv("mi_password")))
		hashStr := hex.EncodeToString(hash.Sum(nil))
		return strings.ToUpper(hashStr)
	}())

	client := &http.Client{}

	loginURL := fmt.Sprintf("%s/%s", common.MiLoginApiUrl, common.MiLoginApiMethodAuth2)
	req, err := http.NewRequest("POST", loginURL, strings.NewReader(data.Encode()))
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "Error creating request: %s", err)
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
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

	internal_log.GetLogger().Infof(ctx, "resp.StatusCode = %d,response: %s", resp.StatusCode, string(respBody))

	matchs := common.MiAccountResultRegexp.FindStringSubmatch(string(respBody))
	if len(matchs) != 2 {
		internal_log.GetLogger().Errorf(ctx, "解包失败 => %s", string(respBody))
		return nil, errors.New("解包失败")
	}

	err = json.Unmarshal([]byte(matchs[1]), &accountResponse)
	if err != nil {
		internal_log.GetLogger().Errorf(ctx, "解包失败: %s,原始数据 => %s", err, matchs[1])
		return nil, err
	}

	if accountResponse.Code != 0 {
		internal_log.GetLogger().Errorf(ctx, "解包失败: %s", err)
		return nil, errors.New("解包失败")
	}

	verifyUrl := ""
	switch {
	case accountResponse.NotificationUrl != "":
		verifyUrl = accountResponse.NotificationUrl
	case accountResponse.CaptchaUrl != nil:
		verifyUrl = *accountResponse.CaptchaUrl
	default:
	}

	if verifyUrl != "" {
		internal_log.GetLogger().Infof(ctx, "请访问以下链接验证登录状态: %s", verifyUrl)
		return nil, errors.New("登录失败")
	}

	m.Pass = &accountResponse.MiPass

	err = m.TakeLoginToken(ctx)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func InitMIAccount(ctx context.Context, opts ...Option) IMIAccount {
	ac := &MIAccount{
		pubSub: internal_sync.NewPubSub(ctx),
	}

	for _, o := range opts {
		o(ac)
	}

	go ac.pubSub.Subseribe(ac, false)

	return ac
}
