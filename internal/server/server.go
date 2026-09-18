package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"github.com/codyzard/lol-exit-lag/internal/discover"
	"github.com/codyzard/lol-exit-lag/internal/probe"
)

// Embedded Web UI assets
//
//go:embed web/*
var webFS embed.FS

// StatusResponse represents real-time dashboard status.
type StatusResponse struct {
	GameRunning     bool    `json:"game_running"`
	ProcessName     string  `json:"process_name"`
	ServerIP        string  `json:"server_ip"`
	ServerPort      int     `json:"server_port"`
	Ping            float64 `json:"ping"`
	Jitter          float64 `json:"jitter"`
	Loss            float64 `json:"loss"`
	OptimizerActive bool    `json:"optimizer_active"`
	SelectedNode    string  `json:"selected_node"`
}

type Server struct {
	mu             sync.Mutex
	optimizerOn    bool
	selectedNode   string
	lastServerIP   string
	lastServerPort int
	lastPing       float64
	lastJitter     float64
	lastLoss       float64
}

// NewServer creates a new local dashboard server instance.
func NewServer() *Server {
	return &Server{
		selectedNode: "direct",
		lastServerIP: "103.149.28.1",
	}
}

// Start launches the local HTTP web server and auto-opens default browser window.
func (s *Server) Start(port int) error {
	subFS, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("failed to load embedded web assets: %v", err)
	}

	mux := http.NewServeMux()

	// Static Assets
	fileServer := http.FileServer(http.FS(subFS))
	mux.Handle("/", fileServer)

	// API Handlers
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/toggle", s.handleToggle)
	mux.HandleFunc("/api/select-node", s.handleSelectNode)

	// Background polling worker for real-time metrics
	go s.startBackgroundMonitor()

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	fmt.Printf("🚀 LoL ExitLag Dashboard GUI starting at http://%s\n", addr)

	// Auto-open browser window
	go func() {
		time.Sleep(800 * time.Millisecond)
		openBrowser(fmt.Sprintf("http://%s", addr))
	}()

	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp := StatusResponse{
		GameRunning:     s.lastServerIP != "",
		ProcessName:     "League of Legends",
		ServerIP:        s.lastServerIP,
		ServerPort:      s.lastServerPort,
		Ping:            s.lastPing,
		Jitter:          s.lastJitter,
		Loss:            s.lastLoss,
		OptimizerActive: s.optimizerOn,
		SelectedNode:    s.selectedNode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Active bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		s.mu.Lock()
		s.optimizerOn = body.Active
		s.mu.Unlock()
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleSelectNode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Node string `json:"node"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err == nil {
		s.mu.Lock()
		s.selectedNode = body.Node
		s.mu.Unlock()
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) startBackgroundMonitor() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 1. Try to discover game server IP
		ip, port, err := discover.FindGameServerFromLogs()
		targetIP := "103.149.28.1"
		targetPort := 59406

		if err == nil && ip != "" {
			targetIP = ip
			if port > 0 {
				targetPort = port
			}
		}

		// 2. Run high-speed probe
		opts := probe.DefaultOptions(targetIP)
		opts.Count = 3
		opts.Timeout = 500 * time.Millisecond
		stats, err := probe.RunICMPProbe(opts)

		s.mu.Lock()
		s.lastServerIP = targetIP
		s.lastServerPort = targetPort
		if err == nil && stats.Received > 0 {
			s.lastPing = float64(stats.AvgRTT.Milliseconds())
			s.lastJitter = float64(stats.Jitter.Milliseconds())
			s.lastLoss = stats.PacketLoss
		}
		s.mu.Unlock()
	}
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		fmt.Printf("⚠️ Could not automatically launch browser: %v\n", err)
	}
}
