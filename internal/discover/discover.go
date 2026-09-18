package discover

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ProcessEndpoint represents an active socket connection owned by a target process.
type ProcessEndpoint struct {
	PID         int    `json:"pid"`
	ProcessName string `json:"process_name"`
	Protocol    string `json:"protocol"`
	LocalAddr   string `json:"local_addr"`
	RemoteAddr  string `json:"remote_addr"`
	State       string `json:"state"`
}

// FindLeagueEndpoints scans Windows network connections for League processes.
func FindLeagueEndpoints() ([]ProcessEndpoint, error) {
	psScript := `
$targetNames = @('League of Legends', 'LeagueClient', 'LeagueClientUx', 'RiotClientServices', 'League of Legends.exe', 'LeagueClient.exe')

$procs = Get-Process | Where-Object { 
    $name = $_.ProcessName
    $targetNames | Where-Object { $name -like "*$_*" }
}

if (-not $procs) {
    Write-Output "[]"
    exit 0
}

$results = @()

foreach ($p in $procs) {
    $pidNum = $p.Id
    $pName = $p.ProcessName

    # TCP Connections
    $tcp = Get-NetTCPConnection -OwningProcess $pidNum -ErrorAction SilentlyContinue
    foreach ($c in $tcp) {
        if ($c.RemoteAddress -and $c.RemoteAddress -ne '0.0.0.0' -ne '::' -ne '127.0.0.1') {
            $results += [PSCustomObject]@{
                pid = $pidNum
                process_name = $pName
                protocol = "TCP"
                local_addr = "$($c.LocalAddress):$($c.LocalPort)"
                remote_addr = "$($c.RemoteAddress):$($c.RemotePort)"
                state = "$($c.State)"
            }
        }
    }

    # UDP Endpoints
    $udp = Get-NetUDPEndpoint -OwningProcess $pidNum -ErrorAction SilentlyContinue
    foreach ($u in $udp) {
        $results += [PSCustomObject]@{
            pid = $pidNum
            process_name = $pName
            protocol = "UDP"
            local_addr = "$($u.LocalAddress):$($u.LocalPort)"
            remote_addr = "0.0.0.0:0"
            state = "Listen/Active"
        }
    }
}

$results | ConvertTo-Json -Compress
`

	cmd := exec.Command("powershell", "-NoProfile", "-Command", psScript)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("powershell execution error: %v, stderr: %s", err, stderr.String())
	}

	raw := strings.TrimSpace(out.String())
	if raw == "" || raw == "[]" {
		return []ProcessEndpoint{}, nil
	}

	var endpoints []ProcessEndpoint
	if strings.HasPrefix(raw, "[") {
		if err := json.Unmarshal([]byte(raw), &endpoints); err != nil {
			return nil, fmt.Errorf("failed to parse JSON array: %v (raw: %s)", err, raw)
		}
	} else {
		var single ProcessEndpoint
		if err := json.Unmarshal([]byte(raw), &single); err != nil {
			return nil, fmt.Errorf("failed to parse single JSON object: %v", err)
		}
		endpoints = append(endpoints, single)
	}

	return endpoints, nil
}

// FindGameServerFromLogs searches League of Legends GameLogs for the active match server IP and Port.
func FindGameServerFromLogs() (string, int, error) {
	possiblePaths := []string{
		`C:\Riot Games\League of Legends\Logs\GameLogs`,
		`D:\Riot Games\League of Legends\Logs\GameLogs`,
		`E:\Riot Games\League of Legends\Logs\GameLogs`,
		`C:\Program Files\Riot Games\League of Legends\Logs\GameLogs`,
		`C:\Program Files (x86)\Riot Games\League of Legends\Logs\GameLogs`,
	}

	// Also check custom Riot install path from environment or running process
	psCmd := exec.Command("powershell", "-NoProfile", "-Command", `(Get-Process -Name "League of Legends" -ErrorAction SilentlyContinue).Path`)
	var pathOut bytes.Buffer
	psCmd.Stdout = &pathOut
	if psCmd.Run() == nil {
		exePath := strings.TrimSpace(pathOut.String())
		if exePath != "" {
			// e.g. C:\Riot Games\League of Legends\Game\League of Legends.exe -> Logs\GameLogs
			gameDir := filepath.Dir(filepath.Dir(exePath))
			logDir := filepath.Join(gameDir, "Logs", "GameLogs")
			possiblePaths = append([]string{logDir}, possiblePaths...)
		}
	}

	var latestLogFile string
	var latestTime int64

	for _, dir := range possiblePaths {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				subDir := filepath.Join(dir, entry.Name())
				files, _ := os.ReadDir(subDir)
				for _, f := range files {
					if strings.HasSuffix(f.Name(), "_r3dlog.txt") || strings.HasSuffix(f.Name(), ".log") || strings.HasSuffix(f.Name(), ".txt") {
						info, err := f.Info()
						if err == nil && info.ModTime().Unix() > latestTime {
							latestTime = info.ModTime().Unix()
							latestLogFile = filepath.Join(subDir, f.Name())
						}
					}
				}
			}
		}
	}

	if latestLogFile == "" {
		return "", 0, fmt.Errorf("no League of Legends GameLogs found in standard paths")
	}

	content, err := os.ReadFile(latestLogFile)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read log file %s: %v", latestLogFile, err)
	}

	// Regex to match server IP and Port in Riot game logs
	// Examples in log: "Connecting to 103.149.28.15:5120", "NetServer: 103.149.28.15", "Server IP: 103.149.28.15"
	ipRegex := regexp.MustCompile(`(?:Connecting to|NetServer|Server IP|ServerAddress|server ip)[:\s]+([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})(?:[:\s]+([0-9]{2,5}))?`)
	matches := ipRegex.FindAllStringSubmatch(string(content), -1)

	if len(matches) == 0 {
		// Generic IP:Port regex fallback
		genericRegex := regexp.MustCompile(`([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}):([0-9]{4,5})`)
		genericMatches := genericRegex.FindAllStringSubmatch(string(content), -1)
		if len(genericMatches) > 0 {
			last := genericMatches[len(genericMatches)-1]
			ip := last[1]
			port, _ := strconv.Atoi(last[2])
			return ip, port, nil
		}
		return "", 0, fmt.Errorf("no match server IP pattern found in %s", filepath.Base(latestLogFile))
	}

	lastMatch := matches[len(matches)-1]
	ip := lastMatch[1]
	port := 0
	if len(lastMatch) > 2 && lastMatch[2] != "" {
		port, _ = strconv.Atoi(lastMatch[2])
	}

	return ip, port, nil
}

// ExtractIP strips port number from IP:Port string
func ExtractIP(addr string) string {
	parts := strings.Split(addr, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return addr
}

// ParsePort extracts the port integer from IP:Port string
func ParsePort(addr string) int {
	parts := strings.Split(addr, ":")
	if len(parts) > 1 {
		p, _ := strconv.Atoi(parts[1])
		return p
	}
	return 0
}

// SortEndpoints sorts endpoints by ProcessName and Protocol
func SortEndpoints(endpoints []ProcessEndpoint) {
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].ProcessName == endpoints[j].ProcessName {
			return endpoints[i].Protocol < endpoints[j].Protocol
		}
		return endpoints[i].ProcessName < endpoints[j].ProcessName
	})
}
