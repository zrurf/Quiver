package monitor

import (
	"sync/atomic"
)

var (
	activeConnections int64
)

func IncConnections() {
	atomic.AddInt64(&activeConnections, 1)
}

func DecConnections() {
	atomic.AddInt64(&activeConnections, -1)
}

func GetActiveConnections() int64 {
	return atomic.LoadInt64(&activeConnections)
}
