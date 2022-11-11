package limiter

import (
	"github.com/HughNian/nmid/pkg/model"

	"github.com/juju/ratelimit"
)

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
