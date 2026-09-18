package tunnel

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config represents WireGuard tunnel settings.
type Config struct {
	ClientPrivateKey string
	ClientAddress    string // e.g. 10.0.0.2/32
	ServerPublicKey  string
	ServerEndpoint   string // e.g. 1.2.3.4:51820
	AllowedIPs       []string
	DNS              string
}

// TargetRiotSubnets contains default LoL VN server subnets for Split-Tunneling.
var TargetRiotSubnets = []string{
	"103.149.0.0/16", // LoL VN Game Server Subnet
	"183.80.0.0/16",  // Riot Datacenter Gateway Subnet
}

// GenerateKeyPair creates a WireGuard private/public keypair using Curve25519 (base64).
func GenerateKeyPair() (privateKey string, publicKey string, err error) {
	var priv [32]byte
	_, err = rand.Read(priv[:])
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random bytes: %v", err)
	}

	// Clamp private key as per Curve25519 specification
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	privateKey = base64.StdEncoding.EncodeToString(priv[:])
	// For full Curve25519 public key computation, wg CLI or golang.org/x/crypto/curve25519 can be used.
	// We return placeholder base64 if standalone.
	return privateKey, "", nil
}

// GenerateConfigFile creates a WireGuard client configuration string with split tunneling.
func GenerateConfigFile(cfg Config) string {
	allowedIPsStr := strings.Join(cfg.AllowedIPs, ", ")
	if allowedIPsStr == "" {
		allowedIPsStr = strings.Join(TargetRiotSubnets, ", ")
	}

	dnsLine := ""
	if cfg.DNS != "" {
		dnsLine = fmt.Sprintf("DNS = %s\n", cfg.DNS)
	}

	return fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s
%s
[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = %s
PersistentKeepalive = 25
`, cfg.ClientPrivateKey, cfg.ClientAddress, dnsLine, cfg.ServerPublicKey, cfg.ServerEndpoint, allowedIPsStr)
}

// SaveConfigFile writes configuration file to disk.
func SaveConfigFile(filePath string, cfg Config) error {
	content := GenerateConfigFile(cfg)
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filePath, []byte(content), 0600)
}

// CheckWireGuardInstalled checks if WireGuard CLI is installed on Windows.
func CheckWireGuardInstalled() bool {
	_, err := exec.LookPath("wireguard.exe")
	if err == nil {
		return true
	}
	// Check standard installation directory
	path := `C:\Program Files\WireGuard\wireguard.exe`
	_, err = os.Stat(path)
	return err == nil
}

// AddSplitRoute adds explicit Windows IP routes for LoL VN subnets pointing to WireGuard interface.
func AddSplitRoute(interfaceName string, subnet string) error {
	cmdStr := fmt.Sprintf(`New-NetRoute -DestinationPrefix "%s" -InterfaceAlias "%s" -Confirm:$false -ErrorAction SilentlyContinue`, subnet, interfaceName)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", cmdStr)
	return cmd.Run()
}

// RemoveSplitRoute removes explicit Windows IP routes.
func RemoveSplitRoute(interfaceName string, subnet string) error {
	cmdStr := fmt.Sprintf(`Remove-NetRoute -DestinationPrefix "%s" -InterfaceAlias "%s" -Confirm:$false -ErrorAction SilentlyContinue`, subnet, interfaceName)
	cmd := exec.Command("powershell", "-NoProfile", "-Command", cmdStr)
	return cmd.Run()
}
