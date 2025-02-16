package spec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"migpt-go/internal/common"
	"net/http"
)

type DeviceInstancesData struct {
	Instances []DeviceInstance `json:"instances,omitempty"`
}

type DeviceInstance struct {
	Status  string `json:"status,omitempty"`
	Model   string `json:"model,omitempty"`
	Version int    `json:"version,omitempty"`
	Type    string `json:"type,omitempty"`
	TS      int64  `json:"ts,omitempty"`
}

// 顶层设备协议结构
type DeviceSpec struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Services    []Service `json:"services"`
}

// 服务定义
type Service struct {
	IID         int        `json:"iid"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	Properties  []Property `json:"properties,omitempty"`
	Actions     []Action   `json:"actions,omitempty"`
}

// 属性定义
type Property struct {
	IID         int             `json:"iid"`
	Type        string          `json:"type"`
	Description string          `json:"description"`
	Format      string          `json:"format"`
	Access      []string        `json:"access"`
	Unit        string          `json:"unit,omitempty"`
	ValueRange  []int           `json:"value-range,omitempty"`
	ValueList   []ValueListItem `json:"value-list,omitempty"`
}

// 属性值列表项
type ValueListItem struct {
	Value       int    `json:"value"`
	Description string `json:"description"`
}

// 动作定义
type Action struct {
	IID         int    `json:"iid"`
	Type        string `json:"type"`
	Description string `json:"description"`
	In          []int  `json:"in"`
	Out         []int  `json:"out"`
}

func GetAllSpec(ctx context.Context) DeviceInstancesData {
	resp, err := http.DefaultClient.Get(common.AllSpecUrl)
	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var data DeviceInstancesData
	err = json.Unmarshal(respBody, &data)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

func GetDeviceSpecCommmond(_type string) *DeviceSpec {
	resp, err := http.DefaultClient.Get(fmt.Sprintf("%s?type=%s", common.SpecUrl, _type))
	if err != nil {
		log.Fatal(err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var data *DeviceSpec
	err = json.Unmarshal(respBody, &data)
	if err != nil {
		log.Fatal(err)
	}

	return data
}
