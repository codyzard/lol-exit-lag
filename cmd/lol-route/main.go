package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/codyzard/lol-exit-lag/internal/discover"
	"github.com/codyzard/lol-exit-lag/internal/probe"
	"github.com/codyzard/lol-exit-lag/internal/trace"
	"github.com/codyzard/lol-exit-lag/internal/tunnel"
)

const version = "v0.2.0"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	subcommand := os.Args[1]

	switch subcommand {
	case "discover":
		runDiscover()
	case "probe":
		runProbe(os.Args[2:])
	case "trace":
		runTrace(os.Args[2:])
	case "benchmark":
		runBenchmark()
	case "tunnel":
		runTunnel(os.Args[2:])
	case "version":
		fmt.Printf("lol-exit-lag %s\n", version)
	default:
		fmt.Printf("Unknown command: %s\n\n", subcommand)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("=====================================================")
	fmt.Printf("   lol-exit-lag - Mini ExitLag Route Optimizer %s\n", version)
	fmt.Println("=====================================================")
	fmt.Println("Usage:")
	fmt.Println("  lol-exit-lag discover          Scan active League sockets & game server IP")
	fmt.Println("  lol-exit-lag probe <ip/host>  Measure Ping, Jitter & Packet Loss")
	fmt.Println("  lol-exit-lag trace <ip/host>  Traceroute path to target IP")
	fmt.Println("  lol-exit-lag benchmark         Run baseline test against candidate nodes")
	fmt.Println("  lol-exit-lag tunnel            Manage WireGuard split-tunnel config")
	fmt.Println("  lol-exit-lag version           Display version")
	fmt.Println("=====================================================")
}

func runDiscover() {
	fmt.Println("🔍 Scanning active League of Legends sockets & Game Logs...")

	// 1. Scan Game Logs for exact Match Server IP
	gameIP, gamePort, errLog := discover.FindGameServerFromLogs()
	fmt.Println("\n=====================================================")
	fmt.Println("🎮 MATCH GAME SERVER DETECTION")
	fmt.Println("=====================================================")
	if errLog == nil && gameIP != "" {
		fmt.Printf("🎯 Match Server IP:   %s\n", gameIP)
		if gamePort > 0 {
			fmt.Printf("🔌 Match UDP Port:    %d\n", gamePort)
		}
		fmt.Println("💡 Running auto probe test to match server...")

		opts := probe.DefaultOptions(gameIP)
		opts.Count = 10
		stats, err := probe.RunICMPProbe(opts)
		if err == nil && stats.Received > 0 {
			fmt.Printf("   ├─ Avg Latency:    %s\n", stats.AvgRTT)
			fmt.Printf("   ├─ Jitter:         %s\n", stats.Jitter)
			fmt.Printf("   └─ Packet Loss:    %.1f%%\n", stats.PacketLoss)
		}
	} else {
		fmt.Printf("⚠️  Could not detect active Game Server IP from logs: %v\n", errLog)
		fmt.Println("💡 Tip: Start a game or Practice Tool match to generate active GameLogs.")
	}

	// 2. Scan active socket endpoints
	fmt.Println("\n=====================================================")
	fmt.Println("🌐 ACTIVE SOCKET CONNECTIONS")
	fmt.Println("=====================================================")
	endpoints, err := discover.FindLeagueEndpoints()
	if err != nil {
		fmt.Printf("❌ Error discovering sockets: %v\n", err)
		return
	}

	if len(endpoints) == 0 {
		fmt.Println("⚠️  No active League processes or sockets found.")
		return
	}

	discover.SortEndpoints(endpoints)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "PID\tPROCESS\tPROTO\tLOCAL ADDR\tREMOTE ADDR\tSTATE")
	fmt.Fprintln(w, "---\t-------\t-----\t----------\t-----------\t-----")
	for _, ep := range endpoints {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n", ep.PID, ep.ProcessName, ep.Protocol, ep.LocalAddr, ep.RemoteAddr, ep.State)
	}
	w.Flush()
}

func runProbe(args []string) {
	probeFlags := flag.NewFlagSet("probe", flag.ExitOnError)
	count := probeFlags.Int("c", 20, "Number of probe packets to send")
	probeFlags.Parse(args)

	if probeFlags.NArg() < 1 {
		fmt.Println("Usage: lol-exit-lag probe <target_ip_or_domain> [-c 20]")
		os.Exit(1)
	}

	target := probeFlags.Arg(0)
	fmt.Printf("📡 Probing target %s (%d packets)...\n", target, *count)

	opts := probe.DefaultOptions(target)
	opts.Count = *count

	stats, err := probe.RunICMPProbe(opts)
	if err != nil {
		fmt.Printf("❌ Probe error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=====================================================")
	fmt.Printf("📊 PROBE REPORT: %s\n", stats.Target)
	fmt.Println("=====================================================")
	fmt.Printf("  Packets Sent:     %d\n", stats.Sent)
	fmt.Printf("  Packets Received: %d\n", stats.Received)
	fmt.Printf("  Packet Loss:      %.1f%%\n", stats.PacketLoss)
	fmt.Println("-----------------------------------------------------")
	fmt.Printf("  Min Latency:      %s\n", stats.MinRTT)
	fmt.Printf("  Max Latency:      %s\n", stats.MaxRTT)
	fmt.Printf("  Average Latency:  %s\n", stats.AvgRTT)
	fmt.Printf("  Jitter (Variance): %s\n", stats.Jitter)
	fmt.Println("=====================================================")
}

func runTrace(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: lol-exit-lag trace <target_ip_or_domain>")
		os.Exit(1)
	}
	target := args[0]

	fmt.Printf("🗺️  Tracerouting path to %s (max 20 hops)...\n\n", target)
	hops, err := trace.RunTraceroute(target, 20)
	if err != nil {
		fmt.Printf("❌ Traceroute error: %v\n", err)
		os.Exit(1)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "HOP\tRTT 1\tRTT 2\tRTT 3\tIP ADDRESS")
	fmt.Fprintln(w, "---\t-----\t-----\t-----\t----------")
	for _, h := range hops {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", h.Hop, h.RTT1, h.RTT2, h.RTT3, h.IP)
	}
	w.Flush()
}

func runBenchmark() {
	fmt.Println("⚡ Running Route Benchmark test against candidate locations...")
	candidates := map[string]string{
		"Cloudflare Global (1.1.1.1)": "1.1.1.1",
		"Google DNS (8.8.8.8)":        "8.8.8.8",
		"Tokyo, JP Node":              "13.112.63.213",
		"Singapore Node":              "13.228.0.251",
		"Hong Kong Node":              "18.162.0.251",
		"Taiwan Node":                 "34.81.0.1",
		"Vietnam Target (VN2)":        "103.149.28.1",
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NODE NAME\tTARGET IP\tAVG LATENCY\tJITTER\tLOSS %")
	fmt.Fprintln(w, "---------\t---------\t-----------\t------\t------")

	for name, ip := range candidates {
		opts := probe.DefaultOptions(ip)
		opts.Count = 5
		opts.Timeout = 800 * time.Millisecond
		stats, err := probe.RunICMPProbe(opts)
		if err != nil || stats.Received == 0 {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", name, ip, "TIMEOUT", "-", "100%")
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.1f%%\n", name, ip, stats.AvgRTT, stats.Jitter, stats.PacketLoss)
		}
	}
	w.Flush()
}

func runTunnel(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage:")
		fmt.Println("  lol-exit-lag tunnel status       Check WireGuard installation status")
		fmt.Println("  lol-exit-lag tunnel gen          Generate sample split-tunnel WireGuard config")
		return
	}

	switch args[0] {
	case "status":
		installed := tunnel.CheckWireGuardInstalled()
		fmt.Println("=====================================================")
		fmt.Println("🔒 WIREGUARD TUNNEL STATUS")
		fmt.Println("=====================================================")
		if installed {
			fmt.Println("✅ WireGuard is installed on this system.")
		} else {
			fmt.Println("⚠️  WireGuard CLI not detected on standard PATH.")
			fmt.Println("💡 Install WireGuard for Windows from https://www.wireguard.com/install/")
		}
		fmt.Printf("🎯 Split-Tunneling Subnets: %v\n", tunnel.TargetRiotSubnets)
		fmt.Println("=====================================================")

	case "gen":
		privKey, _, _ := tunnel.GenerateKeyPair()
		cfg := tunnel.Config{
			ClientPrivateKey: privKey,
			ClientAddress:    "10.0.0.2/32",
			ServerPublicKey:  "REPLACE_WITH_YOUR_VPS_PUBLIC_KEY",
			ServerEndpoint:   "YOUR_VPS_IP:51820",
			AllowedIPs:       tunnel.TargetRiotSubnets,
		}

		outFile := filepath.Join("config", "lol-tunnel.conf")
		err := tunnel.SaveConfigFile(outFile, cfg)
		if err != nil {
			fmt.Printf("❌ Failed to save config: %v\n", err)
			return
		}

		fmt.Println("=====================================================")
		fmt.Println("📄 WIREGUARD SPLIT-TUNNEL CONFIG GENERATED")
		fmt.Println("=====================================================")
		fmt.Printf("Saved to: %s\n\n", outFile)
		fmt.Println(tunnel.GenerateConfigFile(cfg))
		fmt.Println("=====================================================")

	default:
		fmt.Printf("Unknown tunnel command: %s\n", args[0])
	}
}
