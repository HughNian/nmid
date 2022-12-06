package trace

import (
	"context"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	"time"
)

const (
	ComponentIDGOHttpServer  = 5004
	ComponentIDGOHttpClient  = 5005
	ComponentIDGoMicroClient = 5008
	ComponentIDGoMicroServer = 5009
	SpanContextKey           = "span"
)

type SkySpan struct {
	entryCtx context.Context
}

func NewReporter(reporterUrl, serviceName string) (rp go2sky.Reporter, tracer *go2sky.Tracer) {
	rp, err := reporter.NewGRPCReporter(reporterUrl, reporter.WithCheckInterval(time.Second))
	if nil != err {
		logger.Error("create gosky reporter failed!", err)
		return nil, nil
	}
	//defer rp.Close()

	tracer, err = go2sky.NewTracer(serviceName, go2sky.WithReporter(rp))
	if nil != err {
		logger.Error("new gosky tracer failed!", err)
		return nil, nil
	}

	return rp, tracer
}
