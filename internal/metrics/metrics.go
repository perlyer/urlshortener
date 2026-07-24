// Package metrics - Prometheus-метрики и HTTP-middleware для сервисов.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Всего HTTP-запросов по сервису, маршруту и статусу.",
	}, []string{"service", "method", "route", "status"})

	httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Длительность обработки HTTP-запроса, секунды.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "route"})

	cacheOps = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cache_operations_total",
		Help: "Обращения к кэшу ссылок по результату.",
	}, []string{"result"})
)

// Handler отдаёт метрики в формате Prometheus (для эндпоинта /metrics).
func Handler() http.Handler {
	return promhttp.Handler()
}

// CacheHit и CacheMiss фиксируют результат обращения к кэшу.
func CacheHit()  { cacheOps.WithLabelValues("hit").Inc() }
func CacheMiss() { cacheOps.WithLabelValues("miss").Inc() }

// statusRecorder запоминает статус ответа, чтобы положить его в метку.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Middleware считает количество и длительность запросов. Маршрут для метки
// берётся из chi-паттерна ("/{code}"), а НЕ из сырого пути: иначе каждый
// уникальный код стал бы отдельной серией и кардинальность метрик взорвалась бы
// (классическая ошибка инструментирования под нагрузкой).
func Middleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r)

			route := chi.RouteContext(r.Context()).RoutePattern()
			if route == "" {
				route = "unknown"
			}
			httpRequests.WithLabelValues(service, r.Method, route, strconv.Itoa(rec.status)).Inc()
			httpDuration.WithLabelValues(service, route).Observe(time.Since(start).Seconds())
		})
	}
}
