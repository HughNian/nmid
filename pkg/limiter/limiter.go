package limiter

import (
	"context"
	"nmid-v2/pkg/model"

	"github.com/joshbohde/codel"
	"github.com/juju/ratelimit"
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

//NewBucketLimiter token rate limiter, 令牌桶限流
func NewBucketLimiter() *ratelimit.Bucket {
	return ratelimit.NewBucket(model.FILLINTERVAL, model.CAPACITY)
}

func DoBucketLimiter(b *ratelimit.Bucket) bool {
	tokenGet := b.TakeAvailable(1)
	if tokenGet != 0 {
		return true
	} else {
		return false
	}
}
