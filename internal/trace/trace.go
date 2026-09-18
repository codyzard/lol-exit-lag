package trace

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// HopInfo represents a single routing hop in path traceroute.
type HopInfo struct {
	Hop      int    `json:"hop"`
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	RTT1     string `json:"rtt1"`
	RTT2     string `json:"rtt2"`
	RTT3     string `json:"rtt3"`
}

// RunTraceroute executes Windows tracert to target IP.
func RunTraceroute(target string, maxHops int) ([]HopInfo, error) {
	if maxHops <= 0 {
		maxHops = 20
	}

	cmd := exec.Command("tracert", "-d", "-h", strconv.Itoa(maxHops), target)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil && out.Len() == 0 {
		return nil, fmt.Errorf("tracert command failed: %v", err)
	}

	lines := strings.Split(out.String(), "\n")
	hops := make([]HopInfo, 0)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Tracing route") || strings.HasPrefix(line, "over a maximum") || strings.HasPrefix(line, "Trace complete") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 {
			hopNum, err := strconv.Atoi(fields[0])
			if err == nil {
				hop := HopInfo{
					Hop: hopNum,
				}
				if len(fields) >= 5 {
					hop.RTT1 = fields[1]
					hop.RTT2 = fields[3]
					hop.RTT3 = fields[5]
					hop.IP = fields[len(fields)-1]
				} else {
					hop.IP = fields[len(fields)-1]
				}
				hops = append(hops, hop)
			}
		}
	}

	return hops, nil
}
