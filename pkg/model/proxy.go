package model

import "time"

type ProxyServerOption struct {
	ProtocolType     string        `yaml:"protocol_type""`
	BindNetWork      string        `yaml:"bind_network"`
	BindAddress      string        `yaml:"bind_address"`
	BindIP           string        `yaml:"bind_ip"`
	BindPort         string        `yaml:"bind_port"`
	RequestBodySize  int           `yaml:"request_body_size"`
	ResponseBodySize int           `yaml:"response_body_size"`
	ReadTimeout      time.Duration `yaml:"read_timeout"`
	WriteTimeout     time.Duration `yaml:"write_timeout"`
	IdleTimeout      time.Duration `yaml:"idle_timeout"`

	// 反向代理相关
	TargetProtocolType        string        `yaml:"target_protocol_type"`
	TargetAddress             string        `yaml:"target_address"`
	TargetDialTimeout         time.Duration `yaml:"target_dial_timeout"`
	TargetKeepAlive           time.Duration `yaml:"target_keep_alive"`
	TargetIdleConnTimeout     time.Duration `yaml:"target_idle_conn_timeout"`
	TargetMaxIdleConnsPerHost int           `yaml:"target_max_idle_conns_per_host"`
}
