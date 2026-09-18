package probe

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Statistics holds latency, jitter, and packet loss metrics.
type Statistics struct {
	Target      string        `json:"target"`
	Sent        int           `json:"sent"`
	Received    int           `json:"received"`
	PacketLoss  float64       `json:"packet_loss"`
	MinRTT      time.Duration `json:"min_rtt"`
	MaxRTT      time.Duration `json:"max_rtt"`
	AvgRTT      time.Duration `json:"avg_rtt"`
	P50RTT      time.Duration `json:"p50_rtt"`
	P95RTT      time.Duration `json:"p95_rtt"`
	Jitter      time.Duration `json:"jitter"`
	RTTList     []time.Duration
}

// ProbeOptions configures the ping test.
type ProbeOptions struct {
	Target   string
	Count    int
	Interval time.Duration
	Timeout  time.Duration
}

// DefaultOptions returns standard probing parameters.
func DefaultOptions(target string) ProbeOptions {
	return ProbeOptions{
		Target:   target,
		Count:    20,
		Interval: 200 * time.Millisecond,
		Timeout:  1000 * time.Millisecond,
	}
}

// RunICMPProbe runs system ping command on Windows to measure baseline metrics accurately.
func RunICMPProbe(opts ProbeOptions) (*Statistics, error) {
	if opts.Count <= 0 {
		opts.Count = 20
	}

	// Windows system ping command
	args := []string{"-n", strconv.Itoa(opts.Count), "-w", strconv.Itoa(int(opts.Timeout.Milliseconds())), opts.Target}
	cmd := exec.Command("ping", args...)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	output := out.String()

	stats := &Statistics{
		Target:  opts.Target,
		Sent:    opts.Count,
		RTTList: make([]time.Duration, 0),
	}

	// Parse ping stdout lines for round-trip times
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Windows ping format: "Reply from x.x.x.x: bytes=32 time=45ms TTL=54" or "time<1ms"
		if strings.Contains(line, "time=") || strings.Contains(line, "time<") {
			rtt := parsePingTime(line)
			if rtt >= 0 {
				stats.RTTList = append(stats.RTTList, rtt)
			}
		}
	}

	stats.Received = len(stats.RTTList)
	if stats.Sent > 0 {
		stats.PacketLoss = float64(stats.Sent-stats.Received) / float64(stats.Sent) * 100.0
	}

	if stats.Received == 0 {
		if err != nil {
			return nil, fmt.Errorf("ping target %s timed out or host unreachable: %v", opts.Target, err)
		}
		return stats, nil
	}

	calculateStats(stats)
	return stats, nil
}

// RunTCPProbe measures TCP handshake connection latency to a target IP:Port.
func RunTCPProbe(targetAddr string, count int, interval time.Duration) (*Statistics, error) {
	if count <= 0 {
		count = 10
	}

	stats := &Statistics{
		Target:  targetAddr,
		Sent:    count,
		RTTList: make([]time.Duration, 0),
	}

	for i := 0; i < count; i++ {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", targetAddr, 1500*time.Millisecond)
		if err == nil {
			rtt := time.Since(start)
			conn.Close()
			stats.RTTList = append(stats.RTTList, rtt)
		}
		if i < count-1 {
			time.Sleep(interval)
		}
	}

	stats.Received = len(stats.RTTList)
	if stats.Sent > 0 {
		stats.PacketLoss = float64(stats.Sent-stats.Received) / float64(stats.Sent) * 100.0
	}

	if stats.Received > 0 {
		calculateStats(stats)
	}

	return stats, nil
}

func parsePingTime(line string) time.Duration {
	idx := strings.Index(line, "time=")
	if idx != -1 {
		sub := line[idx+5:]
		spaceIdx := strings.Index(sub, "ms")
		if spaceIdx != -1 {
			valStr := strings.TrimSpace(sub[:spaceIdx])
			val, err := strconv.Atoi(valStr)
			if err == nil {
				return time.Duration(val) * time.Millisecond
			}
		}
	}
	if strings.Contains(line, "time<1ms") {
		return 1 * time.Millisecond
	}
	return -1
}

func calculateStats(stats *Statistics) {
	if len(stats.RTTList) == 0 {
		return
	}

	var sum time.Duration
	min := stats.RTTList[0]
	max := stats.RTTList[0]

	for _, rtt := range stats.RTTList {
		if rtt < min {
			min = rtt
		}
		if rtt > max {
			max = rtt
		}
		sum += rtt
	}

	stats.MinRTT = min
	stats.MaxRTT = max
	stats.AvgRTT = sum / time.Duration(len(stats.RTTList))

	// Calculate Jitter (Mean Absolute Difference between consecutive RTTs)
	if len(stats.RTTList) > 1 {
		var totalDiff float64
		for i := 1; i < len(stats.RTTList); i++ {
			diff := math.Abs(float64(stats.RTTList[i] - stats.RTTList[i-1]))
			totalDiff += diff
		}
		stats.Jitter = time.Duration(totalDiff / float64(len(stats.RTTList)-1))
	} else {
		stats.Jitter = 0
	}
}
