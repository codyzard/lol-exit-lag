# 🎮 lol-exit-lag

> **Mini ExitLag Route Optimizer & Latency Analyzer for League of Legends (LoL VN)**

`lol-exit-lag` là một công cụ tối ưu hóa mạng (Mini ExitLag CLI) được viết bằng **Go**, giúp đo đạc, phân tích độ trễ (latency), jitter, packet loss và điều hướng traffic game LoL VN2 qua các nốt VPN / WireGuard VPS tối ưu nhất bằng cơ chế **Split-Tunneling**.

---

## ⚡ Tính Năng (Features)

- 🔍 **`discover`**: Tự động phát hiện tiến trình `League of Legends.exe` và trích xuất chính xác IP / Port của Game Server trận đấu từ Riot Logs & Active Sockets.
- 📡 **`probe <IP>`**: Gửi gói tin đo đạc chỉ số RTT Min/Max/Avg, Jitter (biến động trễ) và Packet Loss %.
- 🗺️ **`trace <IP>`**: Traceroute lộ trình nốt hop từ máy client tới server game.
- ⚡ **`benchmark`**: Đo đạc đối chiếu nhanh baseline với các nốt VPS candidate (Tokyo, Singapore, Taiwan, Vietnam).
- 🔒 **`tunnel`**: Tự động tạo file cấu hình WireGuard Split-Tunneling (`AllowedIPs = 103.149.0.0/16, 183.80.0.0/16`), chỉ điều hướng traffic LoL qua VPN.

---

## 🛠 Cài Đặt & Biên Dịch (Build & Install)

Yêu cầu: **Go 1.22+**

```bash
git clone https://github.com/codyzard/lol-exit-lag.git
cd lol-exit-lag

# Biên dịch binary
go build -o lol-exit-lag.exe ./cmd/lol-route
```

---

## 📖 Hướng Dẫn Sử Dụng (Usage)

### 1. Quét IP Server Trận Đấu
Khi đang trong trận đấu League of Legends (hoặc Phòng Tập / Practice Tool):
```powershell
.\lol-exit-lag.exe discover
```

### 2. Đo Chi Tiết Latency & Jitter
```powershell
.\lol-exit-lag.exe probe 103.149.28.1 -c 30
```

### 3. Phân Tích Hop Route
```powershell
.\lol-exit-lag.exe trace 103.149.28.1
```

### 4. Chạy Benchmark Nốt Candidate
```powershell
.\lol-exit-lag.exe benchmark
```

### 5. Tạo Config WireGuard Split-Tunneling
```powershell
.\lol-exit-lag.exe tunnel gen
```

---

## 📐 Kiến Trúc Hệ Thống (Architecture)

```
                    ┌────────────────────────┐
                    │     Windows PC 🇯🇵/🇻🇳    │
                    │   League of Legends    │
                    └───────────┬────────────┘
                                │
                    (UDP Game Packets - Process Level)
                                │
                                ▼
                    ┌────────────────────────┐
                    │    lol-exit-lag CLI    │
                    │   (Go Route Selector)  │
                    └───────────┬────────────┘
                                │
                                │ Split-Tunneling (WireGuard)
                                ▼
                ┌───────────────┼───────────────┐
                ▼               ▼               ▼
           Tokyo VPS      Singapore VPS     Taiwan VPS
             (🇯🇵)            (🇸🇬)              (🇹🇼)
                │               │               │
                └───────────────┼───────────────┘
                                ▼
                    ┌────────────────────────┐
                    │ LoL VN Server (VN2 🇻🇳) │
                    └────────────────────────┘
```

---

## 📄 License

MIT License
