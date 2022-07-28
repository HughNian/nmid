package limiter

import (
	"context"
	"github.com/joshbohde/codel"
	"nmid-v2/pkg/model"
)

func NewCodelLimiter() *codel.Lock {
	return codel.New(codel.Options{
		// The maximum number of pending acquires
		MaxPending: model.MAXPENDING,
		// The maximum number of concurrent acquires
		MaxOutstanding: model.MAXOUTSTANDING,
		// The target latency to wait for an acquire.
		// Acquires that take longer than this can fail.
		TargetLatency: model.TARGETLATENCY,
	})
}

//DoCodelLimiter codel限流
func DoCodelLimiter(c *codel.Lock) bool {
	// Attempt to acquire the lock.
	err := c.Acquire(context.Background())

	// if err is not nil, acquisition failed.
	if err != nil {
		return false
	}

	// If acquisition succeeded, we need to release it.
	defer c.Release()

	return true
}
