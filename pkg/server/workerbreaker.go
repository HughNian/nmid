package server

import (
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

var (
	confstruct ServerConfig
	conf       = confstruct.GetConfig()
)

// 默认熔断规则类型
type breakerConfig struct {
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

/**
 * 策略1
 * - 连续错误达到阈值熔断
 */
func (bc *breakerConfig) readyToTripOne() func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures >= bc.SerialErrorNumbers
	}
}

/**
 * 策略2
 * - 错误率达到固定百分比熔断
 */
func (bc *breakerConfig) readyToTripTwo() func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		failureRatio := uint8(float64(counts.TotalFailures) / float64(counts.Requests) * 100)
		return counts.Requests >= 3 && failureRatio >= bc.ErrorPercent
	}
}

/**
 * 策略3
 * - 连续错误次数达到阈值 或 错误率达到阈值熔断
 */
func (bc *breakerConfig) readyToTripThree() func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		failureRatio := uint8(float64(counts.TotalFailures) / float64(counts.Requests) * 100)
		return counts.ConsecutiveFailures >= bc.SerialErrorNumbers || failureRatio >= bc.ErrorPercent
	}
}

/**
 * 策略4
 * - 连续错误次数达到阈值 和 错误率同时达到阈值熔断
 */
func (bc *breakerConfig) readyToTripFour() func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		failureRatio := uint8(float64(counts.TotalFailures) / float64(counts.Requests) * 100)
		return counts.ConsecutiveFailures >= bc.SerialErrorNumbers && failureRatio >= bc.ErrorPercent
	}
}

func (bc *breakerConfig) getReadyToTripOne() func(counts gobreaker.Counts) bool {
	switch bc.RuleType {
	case 2:
		return bc.readyToTripTwo()
	case 3:
		return bc.readyToTripThree()
	case 4:
		return bc.readyToTripFour()
	default:
		return bc.readyToTripOne()
	}
}

//worker breaker 熔断
type WorkerBreaker struct {
	sync.RWMutex
	requestTimeout time.Duration
	config         *breakerConfig
	btype          int8                                        // 1 path 代表接口粒度，2 host 代表单个实例级别
	hostname       map[string]*gobreaker.TwoStepCircuitBreaker // hostname: *gobreaker.TwoStepCircuitBreaker
}

func NewWorkerBreaker() *gobreaker.TwoStepCircuitBreaker {
	wb := &WorkerBreaker{
		config: &breakerConfig{
			MaxRequests:        conf.BREAKER.MaxRequest,
			Interval:           conf.BREAKER.Interval,
			SerialErrorNumbers: conf.BREAKER.SerialErrorNumbers,
			OpenTimeout:        conf.BREAKER.Timeout,
			RequestTimeout:     conf.BREAKER.RequestTimeout,
			ErrorPercent:       conf.BREAKER.ErrorPercent,
			RuleType:           conf.BREAKER.RuleType,
			Btype:              conf.BREAKER.Btype,
		},
	}
	wb.requestTimeout = time.Duration(wb.config.RequestTimeout) * time.Second
	wb.btype = wb.config.Btype
	wb.hostname = make(map[string]*gobreaker.TwoStepCircuitBreaker)

	st := gobreaker.Settings{
		ReadyToTrip: wb.config.getReadyToTripOne(),
	}
	breaker := gobreaker.NewTwoStepCircuitBreaker(st)
	return breaker
}
