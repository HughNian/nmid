package model

type ServerConfig struct {
	RpcServer      *RpcServer      `yaml:"RpcServer"`
	SideCar        *SideCar        `yaml:"SideCar"`
	Registry       *Registry       `yaml:"Registry"`
	Breaker        *Breaker        `yaml:"Breaker"`
	WhiteList      *WhiteList      `yaml:"WhiteList"`
	BlackList      *BlackList      `yaml:"BlackList"`
	BreakerConfig  *BreakerConfig  `yaml:"BreakerConfig"`
	LogConfig      *LogConfig      `yaml:"LogConfig"`
	TraceConfig    *TraceConfig    `yaml:"TraceConfig"`
	DingTalkConfig *DingTalkConfig `yaml:"DingTalkConfig"`
	Prometheus     *Prometheus     `yaml:"Prometheus"`
	Dashboard      *Dashboard      `yaml:"Dashboard"`
}

type RpcServer struct {
	NETWORK  string `yaml:"network"`
	HOST     string `yaml:"host"`
	PORT     string `yaml:"port"`
	HTTPPORT string `yaml:"http_port"`
}

type SideCar struct {
	InflowAddr  *ProxyServerOption `yaml:"InflowAddr"`
	OutflowAddr *ProxyServerOption `yaml:"OutflowAddr"`
}

type Service struct {
	Address string `yaml:"Address"`
}

type Registry struct {
	TYPE  string `yaml:"type"`
	HOST  string `yaml:"host"`
	PORT  string `yaml:"port"`
	RENEW int    `yaml:"renew"`
}

type LogConfig struct {
	Debug          bool   `yaml:"debug"`
	LogDir         string `yaml:"log_dir"`
	StdoutFilename string `yaml:"stdout_filename"`
	Encoding       string `yaml:"encoding"`
}

type WhiteList struct {
	Enable        bool            `yaml:"enable"`
	AllowList     map[string]bool `yaml:"allow_list"`
	AllowListMask []string        `yaml:"allow_list_mask"` //net.ParseCIDR("172.17.0.0/16") to get *net.IPNet
}

type BlackList struct {
	Enable          bool            `yaml:"enable"`
	NoAllowList     map[string]bool `yaml:"no_allow_list"`
	NoAllowListMask []string        `yaml:"no_allow_list_mask"`
}

type Breaker struct {
	SerialErrorNumbers uint32 `yaml:"serial_error_numbers"`
	ErrorPercent       uint8  `yaml:"error_percent"`
	MaxRequest         uint32 `yaml:"max_request"`
	Interval           uint32 `yaml:"interval"`
	Timeout            uint32 `yaml:"timeout"`
	RuleType           uint8  `yaml:"rule_type"`
	Btype              int8   `yaml:"btype"`
	RequestTimeout     uint32 `yaml:"request_timeout"`
	Cycle              uint32 `yaml:"cycle"`
}

// BreakerConfig 默认熔断规则类型
type BreakerConfig struct {
	MaxRequests    uint32 `yaml:"max_requests"`    // 熔断器半开时允许运行的请求数量 默认设置为：1，请求成功则断路器关闭
	Interval       uint32 `yaml:"interval"`        // 熔断器处于关闭状态时的清除周期，默认0，如果一直是关闭则不清除请求的次数信息
	ErrorNumbers   uint32 `yaml:"error_numbers"`   // 错误次数阈值
	OpenTimeout    uint32 `yaml:"open_timeout"`    // 熔断器处于打开状态时，经过多久触发为半开状态，单位：s
	RequestTimeout uint32 `yaml:"request_timeout"` // 请求超时时间，单位：s
	ErrorPercent   uint8  `yaml:"error_percent"`   // 错误率阈值，单位：%
	RuleType       uint8  `yaml:"rule_type"`       // 熔断类型：1连续错误达到阈值熔断，2错误率达到固定百分比熔断，3连续错误次数达到阈值或错误率达到阈值熔断，4连续错误次数达到阈值和错误率同时达到阈值熔断
	IsOpen         uint8  `yaml:"is_open"`         // 规则是否打开，1打开2关闭 修改时监控到值为0复位
	Btype          int8   `yaml:"btype"`           // 熔断粒度path单个实例下接口级别，host单个实例级别，默认接口级别
	Timestamp      int64  `yaml:"timestamp"`       // 最后更新时间
	WorkerName     string `yaml:"worker_name" binding:"required"`
}

// TraceConfig trace config
type TraceConfig struct {
	TraceType   string `yaml:"trace_type"`
	ReporterUrl string `yaml:"reporter_url"`
}

type DingTalkConfig struct {
	Enable bool   `yaml:"enable"`
	Tokens string `yaml:"tokens"`
	Secret string `yaml:"secret"`
}

type RetStruct struct {
	Code int
	Msg  string
	Data []byte
}

type Prometheus struct {
	Enable bool   `yaml:"enable"`
	Host   string `yaml:"host"`
	Port   string `yaml:"port" default:"9099"`
	Path   string `yaml:"path" default:"/metrics"`
}

type Dashboard struct {
	Enable      bool   `yaml:"enable"`
	Host        string `yaml:"host"`
	Port        string `yaml:"port" default:"9099"`
	DefaultPath string `yaml:"default_path" default:"/dashboard"`
}

func GetRetStruct() *RetStruct {
	return &RetStruct{
		Code: 0,
		Msg:  "",
		Data: make([]byte, 0),
	}
}
