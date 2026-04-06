package common

import (
	"log"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func LoggingMiddleware(next http.Handler, serviceName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)
		log.Printf("service=%s method=%s path=%s status=%d duration=%s remote=%s", serviceName, r.Method, r.URL.Path, recorder.statusCode, time.Since(startedAt).Round(time.Millisecond), r.RemoteAddr)
	})
}
