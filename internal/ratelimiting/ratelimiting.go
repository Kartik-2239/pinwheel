package ratelimiting

import (
	"math"
	"time"
)

func SimpleRateLimit(
	key_hash string,
	lastCheckTime int64,
	tokens int,
	capacity int,
	refillRate float64, // n per hour
	cost int,
) (int64, int, bool) {

	refillRate = refillRate / 3600.0 // convert to n per second
	currentTime := time.Now().Unix()
	elapsed := currentTime - lastCheckTime
	tokens = int(math.Min(float64(capacity), float64(tokens)+float64(elapsed)*refillRate))

	if tokens >= cost {
		tokens -= cost
		return currentTime, tokens, true
	} else {
		return currentTime, tokens, false
	}
}
