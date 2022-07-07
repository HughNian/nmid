package server

import (
	"nmid-v2/pkg/conf"
	"sync"
	"time"

	"github.com/sony/gobreaker"
)

/**
 * 策略1
 * - 连续错误达到阈值熔断
 */
func readyToTripOne(bc *conf.BreakerConfig) func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures >= bc.SerialErrorNumbers
	}
}

/**
 * 策略2
 * - 错误率达到固定百分比熔断
 */
func readyToTripTwo(bc *conf.BreakerConfig) func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		failureRatio := uint8(float64(counts.TotalFailures) / float64(counts.Requests) * 100)
		return counts.Requests >= 3 && failureRatio >= bc.ErrorPercent
	}
}

/**
 * 策略3
 * - 连续错误次数达到阈值 或 错误率达到阈值熔断
 */
func readyToTripThree(bc *conf.BreakerConfig) func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		failureRatio := uint8(float64(counts.TotalFailures) / float64(counts.Requests) * 100)
		return counts.ConsecutiveFailures >= bc.SerialErrorNumbers || failureRatio >= bc.ErrorPercent
	}
}

/**
 * 策略4
 * - 连续错误次数达到阈值 和 错误率同时达到阈值熔断
 */
func readyToTripFour(bc *conf.BreakerConfig) func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		failureRatio := uint8(float64(counts.TotalFailures) / float64(counts.Requests) * 100)
		return counts.ConsecutiveFailures >= bc.SerialErrorNumbers && failureRatio >= bc.ErrorPercent
	}
}

func getReadyToTripOne(bc *conf.BreakerConfig) func(counts gobreaker.Counts) bool {
	switch bc.RuleType {
	case 2:
		return readyToTripTwo(bc)
	case 3:
		return readyToTripThree(bc)
	case 4:
		return readyToTripFour(bc)
	default:
		return readyToTripOne(bc)
	}
}

//WorkerBreaker worker breaker 熔断
type WorkerBreaker struct {
	sync.RWMutex
	requestTimeout time.Duration
	config         *conf.BreakerConfig
	btype          int8                                        // 1 path 代表接口粒度，2 host 代表单个实例级别
	hostname       map[string]*gobreaker.TwoStepCircuitBreaker // hostname: *gobreaker.TwoStepCircuitBreaker
}

func NewWorkerBreaker(sconf conf.ServerConfig) *gobreaker.TwoStepCircuitBreaker {
	wb := &WorkerBreaker{
		config: sconf.BreakerConfig,
	}
	wb.requestTimeout = time.Duration(wb.config.RequestTimeout) * time.Second
	wb.btype = wb.config.Btype
	wb.hostname = make(map[string]*gobreaker.TwoStepCircuitBreaker)

	st := gobreaker.Settings{
		ReadyToTrip: getReadyToTripOne(wb.config),
	}
	breaker := gobreaker.NewTwoStepCircuitBreaker(st)
	return breaker
}
