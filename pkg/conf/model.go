package conf

type ServerConfig struct {
	Server   ServerCon `yaml:"Server"`
	Registry Registry  `yaml:"Registry"`
	Breaker  Breaker   `yaml:"Breaker"`
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
