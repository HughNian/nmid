package trace

import (
	"context"
	"github.com/HughNian/nmid/pkg/logger"
	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/reporter"
	"time"
)

const (
	ComponentIDGOHttpServer = 5004
	ComponentIDGOHttpClient = 5005
	SpanContextKey          = "span"
)

type SkySpan struct {
	entryCtx context.Context
}

func NewReporter(reporterUrl string) (tracer *go2sky.Tracer) {
	rp, err := reporter.NewGRPCReporter(reporterUrl, reporter.WithCheckInterval(time.Second))
	if nil != err {
		logger.Error("create gosky reporter failed!", err)
		return nil
	}
	defer rp.Close()

	tracer, err = go2sky.NewTracer("test-demo1", go2sky.WithReporter(rp))
	if nil != err {
		logger.Error("new gosky tracer failed!", err)
		return nil
	}

	return tracer
}
