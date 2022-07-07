package conf

import "net"

type ServerConfig struct {
	Server        *ServerCon     `yaml:"Server"`
	Registry      *Registry      `yaml:"Registry"`
	Breaker       *Breaker       `yaml:"Breaker"`
	BreakerConfig *BreakerConfig `yaml:"BreakerConfig"`
	WhiteList     *WhiteList     `yaml:"WhiteList"`
	BlackList     *BlackList     `yaml:"BlackList"`
}

type ServerCon struct {
	NETWORK  string `yaml:"NETWORK"`
	HOST     string `yaml:"HOST"`
	PORT     string `yaml:"PORT"`
	HTTPPORT string `yaml:"HTTPPORT"`
}

//Registry register center config
type Registry struct {
	HOST  string `yaml:"HOST"`
	PORT  string `yaml:"PORT"`
	RENEW int    `yaml:"RENEW"`
}

type WhiteList struct {
	Enable        bool            `yaml:"ENABLE"`
	AllowList     map[string]bool `yaml:"ALLOWLIST"`
	AllowListMask []*net.IPNet    `yaml:"ALLOWLISTMASK"` //net.ParseCIDR("172.17.0.0/16") to get *net.IPNet
}

type BlackList struct {
	Enable          bool            `yaml:"ENABLE"`
	NoAllowList     map[string]bool `yaml:"NOALLOWLIST"`
	NoAllowListMask []*net.IPNet    `yaml:"NOALLOWLISTMASK"`
}

type Breaker struct {
	SerialErrorNumbers uint32 `yaml:"SERIAL_ERROR_NUMBERS"`
	ErrorPercent       uint8  `yaml:"ERROR_PERCENT"`
	MaxRequest         uint32 `yaml:"MAX_REQUEST"`
	Interval           uint32 `yaml:"INTERVAL"`
	Timeout            uint32 `yaml:"TIMEOUT"`
	RuleType           uint8  `yaml:"RULE_TYPE"`
	Btype              int8   `yaml:"BTYPE"`
	RequestTimeout     uint32 `yaml:"REQUEST_TIMEOUT"`
	Cycle              uint32 `yaml:"CYCLE"`
}

//BreakerConfig 默认熔断规则类型
type BreakerConfig struct {
	MaxRequests        uint32 `json:"max_requests"`         // 熔断器半开时允许运行的请求数量 默认设置为：1，请求成功则断路器关闭
	Interval           uint32 `json:"interval"`             // 熔断器处于关闭状态时的清除周期，默认0，如果一直是关闭则不清除请求的次数信息
	SerialErrorNumbers uint32 `json:"serial_error_numbers"` // 错误次数阈值
	OpenTimeout        uint32 `json:"open_timeout"`         // 熔断器处于打开状态时，经过多久触发为半开状态，单位：s
	RequestTimeout     uint32 `json:"request_timeout"`      // 请求超时时间，单位：s
	ErrorPercent       uint8  `json:"error_percent"`        // 错误率阈值，单位：%
	RuleType           uint8  `json:"rule_type"`            // 熔断类型：1连续错误达到阈值熔断，2错误率达到固定百分比熔断，3连续错误次数达到阈值或错误率达到阈值熔断，4连续错误次数达到阈值和错误率同时达到阈值熔断
	IsOpen             uint8  `json:"is_open"`              // 规则是否打开，1打开2关闭 修改时监控到值为0复位
	Btype              int8   `json:"btype"`                // 熔断粒度path单个实例下接口级别，host单个实例级别，默认接口级别
	Timestamp          int64  `json:"timestamp"`            // 最后更新时间
	WorkerName         string `json:"worker_name" binding:"required"`
}
