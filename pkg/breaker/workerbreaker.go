package breaker

import (
	"github.com/sony/gobreaker"
	"nmid-v2/pkg/model"
	"sync"
	"time"
)

//ruleOne 连续错误达到阈值熔断
func ruleOne(bc *model.BreakerConfig) func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures >= bc.ErrorNumbers
	}
}

//ruleTwo 错误率达到固定百分比熔断
func ruleTwo(bc *model.BreakerConfig) func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		failureRatio := uint8(float64(counts.TotalFailures) / float64(counts.Requests) * 100)
		return counts.Requests >= 3 && failureRatio >= bc.ErrorPercent
	}
}

//ruleThree 连续错误次数达到阈值 或 错误率达到阈值熔断
func ruleThree(bc *model.BreakerConfig) func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		failureRatio := uint8(float64(counts.TotalFailures) / float64(counts.Requests) * 100)
		return counts.ConsecutiveFailures >= bc.ErrorNumbers || failureRatio >= bc.ErrorPercent
	}
}

//ruleFour 连续错误次数达到阈值 和 错误率同时达到阈值熔断
func ruleFour(bc *model.BreakerConfig) func(counts gobreaker.Counts) bool {
	return func(counts gobreaker.Counts) bool {
		failureRatio := uint8(float64(counts.TotalFailures) / float64(counts.Requests) * 100)
		return counts.ConsecutiveFailures >= bc.ErrorNumbers && failureRatio >= bc.ErrorPercent
	}
}

func switchOne(bc *model.BreakerConfig) func(counts gobreaker.Counts) bool {
	switch bc.RuleType {
	case 2:
		return ruleTwo(bc)
	case 3:
		return ruleThree(bc)
	case 4:
		return ruleFour(bc)
	default:
		return ruleOne(bc)
	}
}

//WorkerBreaker worker breaker 熔断
type WorkerBreaker struct {
	sync.RWMutex
	requestTimeout time.Duration
	config         *model.BreakerConfig
	btype          int8                                        // 1 path 代表接口粒度，2 host 代表单个实例级别
	hostname       map[string]*gobreaker.TwoStepCircuitBreaker // hostname: *gobreaker.TwoStepCircuitBreaker
}

func NewWorkerBreaker(sconf model.ServerConfig) *gobreaker.TwoStepCircuitBreaker {
	wb := &WorkerBreaker{
		config: sconf.BreakerConfig,
	}
	wb.requestTimeout = time.Duration(wb.config.RequestTimeout) * time.Second
	wb.btype = wb.config.Btype
	wb.hostname = make(map[string]*gobreaker.TwoStepCircuitBreaker)

	st := gobreaker.Settings{
		ReadyToTrip: switchOne(wb.config),
	}
	breaker := gobreaker.NewTwoStepCircuitBreaker(st)
	return breaker
}
