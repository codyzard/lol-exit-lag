package hooker

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

var (
	mu             sync.Mutex
	activeRoutes   = make(map[string]string) // gameIP -> gatewayIP
	tailscaleIP    string
	defaultGateway string
)

// FindTailscaleAdapter finds the local Tailscale network adapter IP or virtual gateway
func FindTailscaleAdapter() string {
	mu.Lock()
	defer mu.Unlock()

	if tailscaleIP != "" {
		return tailscaleIP
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, iface := range ifaces {
		name := strings.ToLower(iface.Name)
		if strings.Contains(name, "tailscale") || strings.Contains(name, "wintun") || strings.Contains(name, "wireguard") {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
					tailscaleIP = ip.String()
					return tailscaleIP
				}
			}
		}
	}

	return ""
}

// GetDefaultGateway finds the default Windows gateway IP
func GetDefaultGateway() string {
	mu.Lock()
	defer mu.Unlock()

	if defaultGateway != "" {
		return defaultGateway
	}

	cmd := exec.Command("powershell", "-NoProfile", "-Command", `(Get-NetRoute -DestinationPrefix "0.0.0.0/0" -ErrorAction SilentlyContinue | Select-Object -First 1).NextHop`)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err == nil {
		defaultGateway = strings.TrimSpace(out.String())
		return defaultGateway
	}

	return "192.168.1.1" // Standard fallback
}

// ApplyGameRoute redirects only the specific LoL Game Server IP through the designated gateway
func ApplyGameRoute(gameIP string, gatewayIP string) error {
	mu.Lock()
	defer mu.Unlock()

	if gameIP == "" || gameIP == "0.0.0.0" {
		return fmt.Errorf("invalid game IP: %s", gameIP)
	}

	// Remove any existing route for this IP first
	removeRouteCommand(gameIP)

	// Add high priority route (metric 1) strictly for this Game Server IP
	// route add <gameIP> mask 255.255.255.255 <gatewayIP> metric 1
	cmd := exec.Command("route", "add", gameIP, "mask", "255.255.255.255", gatewayIP, "metric", "1")
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	var errOut bytes.Buffer
	cmd.Stderr = &errOut

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to add route: %v (details: %s)", err, errOut.String())
	}

	activeRoutes[gameIP] = gatewayIP
	return nil
}

// RestoreGameRoute removes the custom route and restores default Windows ISP path for the game IP
func RestoreGameRoute(gameIP string) error {
	mu.Lock()
	defer mu.Unlock()

	if gameIP == "" {
		return nil
	}

	removeRouteCommand(gameIP)
	delete(activeRoutes, gameIP)
	return nil
}

// RestoreAllRoutes cleans up all applied game routes upon app exit or toggle OFF
func RestoreAllRoutes() {
	mu.Lock()
	defer mu.Unlock()

	for gameIP := range activeRoutes {
		removeRouteCommand(gameIP)
	}
	activeRoutes = make(map[string]string)
}

func removeRouteCommand(ip string) {
	cmd := exec.Command("route", "delete", ip)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	_ = cmd.Run()
}
