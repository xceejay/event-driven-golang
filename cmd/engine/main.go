package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"event-engine-starter/internal/adapter"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"event-engine-starter/config"
	"event-engine-starter/internal/job"
	mcpserver "event-engine-starter/internal/mcp"
	internalmigrate "event-engine-starter/internal/migrate"
	"event-engine-starter/internal/model"
	"event-engine-starter/internal/repository"
	"event-engine-starter/internal/service"
	"event-engine-starter/internal/web"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmsgprefix)
	log.SetPrefix("[engine] ")

	// Load configuration.
	cfgPath := "config.yaml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		cfgPath = p
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to MySQL.
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&loc=UTC",
		cfg.DB.User, cfg.DB.Password, cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("connect to MySQL: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	log.Printf("connected to MySQL at %s:%d/%s", cfg.DB.Host, cfg.DB.Port, cfg.DB.Name)

	// Apply SQL migrations (idempotent).
	if err := internalmigrate.Apply(db, "migrations"); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}
	log.Printf("migrations applied successfully")

	// Connect to NATS.
	nc, err := nats.Connect(cfg.Broker.URL)
	if err != nil {
		log.Fatalf("connect to NATS: %v", err)
	}
	defer nc.Close()
	log.Printf("connected to NATS at %s", cfg.Broker.URL)

	// Optionally start embedded adapter for demo flows.
	if os.Getenv("RUN_ADAPTER") == "1" {
		engineURL := fmt.Sprintf("http://localhost:%d", cfg.API.HttpPort)
		if err := adapter.Start(ctx, nc, engineURL); err != nil {
			log.Printf("failed to start adapter: %v", err)
		}
	}

	// Set up payload repository (Redis or in-memory).
	var payloadRepo repository.PayloadRepository
	if cfg.PayloadStore.Type == "redis" {
		rdb := redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%d", cfg.PayloadStore.Host, cfg.PayloadStore.Port),
		})
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("Redis unavailable, falling back to in-memory payload store: %v", err)
			payloadRepo = repository.NewInMemoryPayloadRepository()
		} else {
			payloadRepo = repository.NewRedisPayloadRepository(rdb)
			log.Printf("connected to Redis at %s:%d", cfg.PayloadStore.Host, cfg.PayloadStore.Port)
		}
	} else {
		payloadRepo = repository.NewInMemoryPayloadRepository()
		log.Println("using in-memory payload store")
	}

	// Build repositories.
	eventRepo := repository.NewMySQLEventRepository(db)
	configRepo := repository.NewMySQLConfigRepository(db)
	errorLogRepo := repository.NewMySQLErrorLogRepository(db)

	// Build services.
	processingSvc := service.NewEventProcessingService(eventRepo, payloadRepo, errorLogRepo)
	configSvc := service.NewEventConfigurationService(configRepo)
	dispatcher := service.NewEventDispatcher(nc)
	lifecycleSvc := service.NewEventLifecycleManagementService(processingSvc, configSvc, dispatcher, eventRepo)

	// WebSocket hub — wire up event state change broadcasts.
	hub := web.NewHub()
	lifecycleSvc.OnEventStateChange = func(event *model.Event) {
		hub.BroadcastEvent(event)
	}

	// Build MCP server.
	mcpSrv := mcpserver.NewServer(lifecycleSvc, configSvc, processingSvc)

	// Build HTTP router.
	router := web.NewRouter(mcpSrv, hub, lifecycleSvc, processingSvc)

	// Start scheduled jobs.
	flowTypes := []model.FlowType{model.FlowTypeA, model.FlowTypeB}
	var jobs []*job.Job

	for _, ft := range flowTypes {
		awaitingCfg := makeJobConfig(fmt.Sprintf("%s-awaiting", ft), cfg.Jobs, ft, "awaiting")
		scheduledCfg := makeJobConfig(fmt.Sprintf("%s-scheduled", ft), cfg.Jobs, ft, "scheduled")
		suspendedCfg := makeJobConfig(fmt.Sprintf("%s-suspended", ft), cfg.Jobs, ft, "suspended")

		j1 := job.NewAwaitingProcessingJob(ft, lifecycleSvc, awaitingCfg)
		j2 := job.NewScheduledEventsJob(ft, lifecycleSvc, scheduledCfg)
		j3 := job.NewExpiredSuspendedJob(ft, lifecycleSvc, suspendedCfg)

		j1.Start(ctx)
		j2.Start(ctx)
		j3.Start(ctx)

		jobs = append(jobs, j1, j2, j3)
	}

	// Start metrics job.
	metricsJob := job.NewMetricsJob(lifecycleSvc, flowTypes, 60*time.Second)
	metricsJob.Start(ctx)

	log.Printf("started %d scheduled jobs for %d flow types", len(jobs), len(flowTypes))

	// Start HTTP server.
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.API.HttpPort),
		Handler: router,
	}

	go func() {
		log.Printf("HTTP server listening on :%d", cfg.API.HttpPort)
		log.Printf("  Dashboard:  http://localhost:%d/", cfg.API.HttpPort)
		log.Printf("  Health:     http://localhost:%d/health", cfg.API.HttpPort)
		log.Printf("  Metrics:    http://localhost:%d/metrics", cfg.API.HttpPort)
		log.Printf("  MCP SSE:    http://localhost:%d/mcp", cfg.API.HttpPort)
		log.Printf("  WebSocket:  ws://localhost:%d/ws", cfg.API.HttpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	cancel()

	for _, j := range jobs {
		j.Stop()
	}
	metricsJob.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	server.Shutdown(shutdownCtx)

	log.Println("shutdown complete")
}

// makeJobConfig builds a JobConfig from the config map, with sensible defaults.
func makeJobConfig(name string, jobsCfg map[string]config.FlowJobsConfig, flowType model.FlowType, jobType string) job.JobConfig {
	defaults := job.JobConfig{
		Name:         name,
		Interval:     5 * time.Second,
		StartupDelay: 3 * time.Second,
		BatchSize:    100,
		MaxItems:     100000,
		MaxFails:     100,
		MaxRuntime:   30 * time.Second,
	}

	flowKey := strings.ToLower(string(flowType))
	flowCfg, ok := jobsCfg[flowKey]
	if !ok {
		return defaults
	}

	var jc *config.JobParams
	switch jobType {
	case "awaiting":
		jc = flowCfg.Awaiting
	case "scheduled":
		jc = flowCfg.Scheduled
	case "suspended":
		jc = flowCfg.Suspended
	}

	if jc == nil {
		return defaults
	}

	if jc.IntervalMs > 0 {
		defaults.Interval = time.Duration(jc.IntervalMs) * time.Millisecond
	}
	if jc.StartupDelayMs > 0 {
		defaults.StartupDelay = time.Duration(jc.StartupDelayMs) * time.Millisecond
	}
	if jc.BatchSize > 0 {
		defaults.BatchSize = jc.BatchSize
	}
	if jc.MaxItems > 0 {
		defaults.MaxItems = jc.MaxItems
	}
	if jc.MaxFails > 0 {
		defaults.MaxFails = jc.MaxFails
	}
	if jc.MaxRuntimeMs > 0 {
		defaults.MaxRuntime = time.Duration(jc.MaxRuntimeMs) * time.Millisecond
	}

	return defaults
}
