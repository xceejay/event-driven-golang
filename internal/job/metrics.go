package job

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"event-engine-starter/internal/model"
)

var queueSizeGauge = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "event_engine_queue_size",
		Help: "Number of events by flow type and status",
	},
	[]string{"flow_type", "status"},
)

func init() {
	prometheus.MustRegister(queueSizeGauge)
}

// allStatuses lists the non-final statuses to track.
var allStatuses = []model.EventStatus{
	model.StatusAwaitingProcessing,
	model.StatusDispatched,
	model.StatusBeingProcessed,
	model.StatusSuspended,
}

// MetricsJob periodically records queue sizes as Prometheus gauges.
type MetricsJob struct {
	svc       LifecycleService
	flowTypes []model.FlowType
	interval  time.Duration
	stopCh    chan struct{}
	logger    *log.Logger
}

// NewMetricsJob creates a new queue size metrics job.
func NewMetricsJob(svc LifecycleService, flowTypes []model.FlowType, interval time.Duration) *MetricsJob {
	return &MetricsJob{
		svc:       svc,
		flowTypes: flowTypes,
		interval:  interval,
		stopCh:    make(chan struct{}),
		logger:    log.New(os.Stdout, "[metrics] ", log.LstdFlags|log.Lmsgprefix),
	}
}

// Start launches the metrics job in a background goroutine.
func (m *MetricsJob) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.collect(ctx)
			case <-m.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Stop signals the metrics job to stop.
func (m *MetricsJob) Stop() {
	select {
	case <-m.stopCh:
	default:
		close(m.stopCh)
	}
}

func (m *MetricsJob) collect(ctx context.Context) {
	for _, ft := range m.flowTypes {
		for _, status := range allStatuses {
			size, err := m.svc.GetQueueSize(ctx, ft, status)
			if err != nil {
				m.logger.Printf("error getting queue size for %s/%s: %v", ft, status, err)
				continue
			}
			queueSizeGauge.WithLabelValues(string(ft), string(status)).Set(float64(size))
		}
	}
}
