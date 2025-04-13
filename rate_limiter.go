package router

import (
	"net/http"
	"sync"
	"time"
)

type RequestCounter struct {
	lastRequest sync.Map
}

var requestCounter = &RequestCounter{}

func RateLimit(w http.ResponseWriter, r *http.Request, threshold int64) {
	now := time.Now()

	value, ok := requestCounter.lastRequest.Load(r.Host)
	if ok {
		if lastVisit, ok := value.(time.Time); ok {
			timeDiff := now.UnixNano() - lastVisit.UnixNano()
			if timeDiff < threshold {
				JSON(w, http.StatusOK, Msg{})
				RestrictAccess(10)
				return
			}
		}
	}

	requestCounter.lastRequest.Store(r.Host, now)
	cleanupOldRequests(now)
}

func cleanupOldRequests(currentTime time.Time) {
	oneSecondAgo := currentTime.Add(-time.Second)

	requestCounter.lastRequest.Range(func(key, value interface{}) bool {
		if visitTime, ok := value.(time.Time); ok && visitTime.Before(oneSecondAgo) {
			requestCounter.lastRequest.Delete(key)
		}
		return true
	})
}

func RestrictAccess(timeDuration int) {
	time.Sleep(time.Duration(timeDuration) * time.Millisecond)
}
