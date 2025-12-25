package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/makalin/arcron/internal/alerts"
	"github.com/makalin/arcron/internal/config"
	"github.com/makalin/arcron/internal/jobs"
	"github.com/makalin/arcron/internal/ml"
	"github.com/makalin/arcron/internal/monitoring"
	"github.com/makalin/arcron/internal/scheduler"
	"github.com/makalin/arcron/internal/storage"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// Server represents the API server
type Server struct {
	config       *config.Config
	store        *storage.Storage
	jobManager   *jobs.Manager
	scheduler    *scheduler.Scheduler
	monitor      *monitoring.Monitor
	mlEngine     *ml.Engine
	alertManager *alerts.Manager
	router       *mux.Router
	httpServer   *http.Server
	upgrader     websocket.Upgrader
}

// New creates a new API server instance
func New(cfg *config.Config, store *storage.Storage, jobManager *jobs.Manager, 
	sched *scheduler.Scheduler, monitor *monitoring.Monitor, mlEngine *ml.Engine,
	alertManager *alerts.Manager) (*Server, error) {
	
	router := mux.NewRouter()
	
	server := &Server{
		config:       cfg,
		store:        store,
		jobManager:   jobManager,
		scheduler:    sched,
		monitor:      monitor,
		mlEngine:     mlEngine,
		alertManager: alertManager,
		router:       router,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
	}

	server.setupRoutes()

	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	server.httpServer = httpServer
	return server, nil
}

// setupRoutes sets up all API routes
func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()
	
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	
	// Metrics endpoints
	api.HandleFunc("/metrics", s.handleGetMetrics).Methods("GET")
	api.HandleFunc("/metrics/realtime", s.handleRealtimeMetrics).Methods("GET")
	
	// Job endpoints
	api.HandleFunc("/jobs", s.handleListJobs).Methods("GET")
	api.HandleFunc("/jobs/{name}", s.handleGetJob).Methods("GET")
	api.HandleFunc("/jobs/{name}/execute", s.handleExecuteJob).Methods("POST")
	api.HandleFunc("/jobs/{name}/executions", s.handleGetJobExecutions).Methods("GET")
	api.HandleFunc("/jobs/{name}/statistics", s.handleGetJobStatistics).Methods("GET")
	
	// Scheduler endpoints
	api.HandleFunc("/scheduler/status", s.handleSchedulerStatus).Methods("GET")
	api.HandleFunc("/scheduler/jobs/{name}/status", s.handleGetJobStatus).Methods("GET")
	
	// ML endpoints
	api.HandleFunc("/ml/status", s.handleMLStatus).Methods("GET")
	api.HandleFunc("/ml/predict/{jobName}", s.handleMLPredict).Methods("GET")
	
	// System endpoints
	api.HandleFunc("/system/status", s.handleSystemStatus).Methods("GET")
	
	// WebSocket for real-time updates
	s.router.HandleFunc("/ws", s.handleWebSocket)
	
	// Serve static files for dashboard
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/dist/")))
}

// Start starts the API server
func (s *Server) Start(ctx context.Context) error {
	logrus.Infof("Starting API server on %s", s.httpServer.Addr)
	
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.httpServer.Shutdown(shutdownCtx)
	}()
	
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %v", err)
	}
	
	return nil
}

// Response represents a standard API response
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) writeError(w http.ResponseWriter, status int, err error) {
	s.writeJSON(w, status, Response{
		Success: false,
		Error:   err.Error(),
	})
}

func (s *Server) writeSuccess(w http.ResponseWriter, data interface{}) {
	s.writeJSON(w, http.StatusOK, Response{
		Success: true,
		Data:    data,
	})
}

// Health check handler
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeSuccess(w, map[string]interface{}{
		"status":  "healthy",
		"version": "1.0.0",
		"uptime":  time.Since(time.Now()).String(), // Placeholder
	})
}

// Metrics handlers
func (s *Server) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	startStr := query.Get("start")
	endStr := query.Get("end")
	limit := 1000
	
	var start, end time.Time
	var err error
	
	if startStr != "" {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid start time: %v", err))
			return
		}
	} else {
		start = time.Now().Add(-24 * time.Hour)
	}
	
	if endStr != "" {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			s.writeError(w, http.StatusBadRequest, fmt.Errorf("invalid end time: %v", err))
			return
		}
	} else {
		end = time.Now()
	}
	
	metrics, err := s.store.GetSystemMetrics(start, end, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	
	s.writeSuccess(w, metrics)
}

func (s *Server) handleRealtimeMetrics(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			metrics := s.monitor.GetLastMetrics()
			if metrics != nil {
				if err := conn.WriteJSON(metrics); err != nil {
					logrus.Errorf("WebSocket write error: %v", err)
					return
				}
			}
		}
	}
}

// Job handlers
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	allJobs := s.jobManager.GetAllJobs()
	jobsList := make([]map[string]interface{}, 0, len(allJobs))
	
	for name, job := range allJobs {
		scheduledJob, _ := s.scheduler.GetJobStatus(name)
		jobData := map[string]interface{}{
			"name":     name,
			"type":    job.GetType(),
			"schedule": job.GetSchedule(),
			"status":   job.GetStatus(),
		}
		
		if scheduledJob != nil {
			jobData["next_run"] = scheduledJob.NextRun
			jobData["last_run"] = scheduledJob.LastRun
			jobData["run_count"] = scheduledJob.RunCount
		}
		
		jobsList = append(jobsList, jobData)
	}
	
	s.writeSuccess(w, jobsList)
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobName := vars["name"]
	
	job, exists := s.jobManager.GetJob(jobName)
	if !exists {
		s.writeError(w, http.StatusNotFound, fmt.Errorf("job not found: %s", jobName))
		return
	}
	
	scheduledJob, _ := s.scheduler.GetJobStatus(jobName)
	jobData := map[string]interface{}{
		"name":     job.GetName(),
		"type":     job.GetType(),
		"schedule": job.GetSchedule(),
		"status":   job.GetStatus(),
		"config":   job.GetConfig(),
	}
	
	if scheduledJob != nil {
		jobData["next_run"] = scheduledJob.NextRun
		jobData["last_run"] = scheduledJob.LastRun
		jobData["run_count"] = scheduledJob.RunCount
		if scheduledJob.Prediction != nil {
			jobData["prediction"] = scheduledJob.Prediction
		}
	}
	
	s.writeSuccess(w, jobData)
}

func (s *Server) handleExecuteJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobName := vars["name"]
	
	job, exists := s.jobManager.GetJob(jobName)
	if !exists {
		s.writeError(w, http.StatusNotFound, fmt.Errorf("job not found: %s", jobName))
		return
	}
	
	go func() {
		if err := s.jobManager.ExecuteJob(job); err != nil {
			logrus.Errorf("Failed to execute job %s: %v", jobName, err)
		}
	}()
	
	s.writeSuccess(w, map[string]string{
		"message": fmt.Sprintf("Job %s execution started", jobName),
	})
}

func (s *Server) handleGetJobExecutions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobName := vars["name"]
	
	limit := 100
	executions, err := s.jobManager.GetJobExecutions(jobName, limit)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	
	s.writeSuccess(w, executions)
}

func (s *Server) handleGetJobStatistics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobName := vars["name"]
	
	stats, err := s.store.GetJobStatistics(jobName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	
	s.writeSuccess(w, stats)
}

// Scheduler handlers
func (s *Server) handleSchedulerStatus(w http.ResponseWriter, r *http.Request) {
	status := s.scheduler.GetStatus()
	s.writeSuccess(w, status)
}

func (s *Server) handleGetJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobName := vars["name"]
	
	scheduledJob, exists := s.scheduler.GetJobStatus(jobName)
	if !exists {
		s.writeError(w, http.StatusNotFound, fmt.Errorf("job not found: %s", jobName))
		return
	}
	
	status := map[string]interface{}{
		"status":    scheduledJob.Status,
		"next_run":  scheduledJob.NextRun,
		"last_run":  scheduledJob.LastRun,
		"run_count": scheduledJob.RunCount,
	}
	
	if scheduledJob.Prediction != nil {
		status["prediction"] = scheduledJob.Prediction
	}
	
	s.writeSuccess(w, status)
}

// ML handlers
func (s *Server) handleMLStatus(w http.ResponseWriter, r *http.Request) {
	status := s.mlEngine.GetStatus()
	s.writeSuccess(w, status)
}

func (s *Server) handleMLPredict(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobName := vars["name"]
	
	job, exists := s.jobManager.GetJob(jobName)
	if !exists {
		s.writeError(w, http.StatusNotFound, fmt.Errorf("job not found: %s", jobName))
		return
	}
	
	metrics := s.monitor.GetLastMetrics()
	if metrics == nil {
		s.writeError(w, http.StatusServiceUnavailable, fmt.Errorf("no metrics available"))
		return
	}
	
	prediction, err := s.mlEngine.PredictOptimalTime(jobName, job.GetType(), *metrics)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err)
		return
	}
	
	s.writeSuccess(w, prediction)
}

// System status handler
func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"monitor":   s.monitor.GetStatus(),
		"ml_engine": s.mlEngine.GetStatus(),
		"scheduler": s.scheduler.GetStatus(),
	}
	
	s.writeSuccess(w, status)
}

// WebSocket handler
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			update := map[string]interface{}{
				"timestamp": time.Now(),
				"metrics":   s.monitor.GetLastMetrics(),
				"scheduler": s.scheduler.GetStatus(),
			}
			
			if err := conn.WriteJSON(update); err != nil {
				logrus.Errorf("WebSocket write error: %v", err)
				return
			}
		}
	}
}

