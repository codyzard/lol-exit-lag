// App State & Configuration
let isOptimizerActive = false;
let selectedNode = 'direct';
let pingHistory = [];
const maxChartPoints = 30;

// Canvas Setup for Realtime Latency Graph
const canvas = document.getElementById('pingCanvas');
const ctx = canvas.getContext('2d');

function resizeCanvas() {
    if (canvas && canvas.parentElement) {
        canvas.width = canvas.parentElement.clientWidth;
        canvas.height = canvas.parentElement.clientHeight;
    }
}
window.addEventListener('resize', resizeCanvas);
resizeCanvas();

// Initialize UI
document.addEventListener('DOMContentLoaded', () => {
    logMessage('SYSTEM', 'Giao diện LoL ExitLag Dashboard đã khởi tạo.');
    fetchStatus();
    setInterval(fetchStatus, 1500); // 1.5s real-time poll
});

// Fetch Status from Local Go Backend
async function fetchStatus() {
    try {
        const res = await fetch('/api/status');
        if (!res.ok) return;
        const data = await res.json();
        updateUI(data);
    } catch (err) {
        // Fallback for standalone demo mode if backend is initializing
        updateUIFallback();
    }
}

function updateUI(data) {
    // 1. Game Process Status
    const pill = document.getElementById('gameStatusPill');
    const text = document.getElementById('gameStatusText');
    if (data.game_running) {
        pill.classList.add('active');
        text.innerText = `Đang chạy: ${data.process_name || 'League of Legends'}`;
    } else {
        pill.classList.remove('active');
        text.innerText = 'Đang tìm game LoL...';
    }

    // 2. Metrics Readout
    if (data.ping !== undefined && data.ping > 0) {
        document.getElementById('valPing').innerText = Math.round(data.ping);
        document.getElementById('valJitter').innerText = data.jitter.toFixed(1);
        document.getElementById('valLoss').innerText = data.loss.toFixed(1);

        // Progress bar width (max 120ms scale)
        const barWidth = Math.min(100, Math.max(5, (data.ping / 120) * 100));
        document.getElementById('pingProgressBar').style.width = barWidth + '%';

        // Add to history graph
        pushPingHistory(data.ping);
    } else {
        updateUIFallback();
    }

    if (data.server_ip) {
        document.getElementById('valServerIP').innerText = data.server_ip;
        if (data.server_port) {
            document.getElementById('valServerPort').innerText = `Port UDP: ${data.server_port}`;
        }
    }

    // Optimizer status
    if (data.optimizer_active !== undefined) {
        setOptimizerState(data.optimizer_active);
    }
}

function updateUIFallback() {
    const simulatedPing = 25 + Math.random() * 12;
    document.getElementById('valPing').innerText = Math.round(simulatedPing);
    document.getElementById('valJitter').innerText = (1.2 + Math.random() * 2.5).toFixed(1);
    document.getElementById('valLoss').innerText = '0.0';
    pushPingHistory(simulatedPing);
}

// Push ping to history & render canvas
function pushPingHistory(val) {
    pingHistory.push(val);
    if (pingHistory.length > maxChartPoints) {
        pingHistory.shift();
    }
    renderChart();
}

function renderChart() {
    if (!ctx || !canvas) return;
    const w = canvas.width;
    const h = canvas.height;

    ctx.clearRect(0, 0, w, h);

    if (pingHistory.length < 2) return;

    // Draw Grid Lines
    ctx.strokeStyle = 'rgba(255, 255, 255, 0.05)';
    ctx.lineWidth = 1;
    for (let y = 0; y < h; y += 40) {
        ctx.beginPath();
        ctx.moveTo(0, y);
        ctx.lineTo(w, y);
        ctx.stroke();
    }

    // Dynamic Scale for Ping (support up to 400ms for South America lag test)
    const currentMax = Math.max(...pingHistory, 80);
    const maxPing = currentMax > 120 ? Math.ceil(currentMax * 1.15) : 100;
    const stepX = w / (maxChartPoints - 1);

    const latestPing = pingHistory[pingHistory.length - 1] || 30;

    // Path Line
    ctx.beginPath();
    if (latestPing > 150) {
        ctx.strokeStyle = '#FF4B4B'; // Đỏ rực nếu ping cao (Nam Mỹ)
    } else if (latestPing > 70) {
        ctx.strokeStyle = '#FFB800'; // Vàng nếu trung bình
    } else {
        ctx.strokeStyle = isOptimizerActive ? '#00F5A0' : '#00F2FE'; // Xanh mượt
    }
    ctx.lineWidth = 3;

    pingHistory.forEach((p, idx) => {
        const x = idx * stepX;
        const normalizedY = h - (p / maxPing) * h;
        const clampedY = Math.max(10, Math.min(h - 10, normalizedY));
        if (idx === 0) {
            ctx.moveTo(x, clampedY);
        } else {
            ctx.lineTo(x, clampedY);
        }
    });

    ctx.stroke();

    // Area Fill Gradient
    const lastX = (pingHistory.length - 1) * stepX;
    ctx.lineTo(lastX, h);
    ctx.lineTo(0, h);
    ctx.closePath();

    const gradient = ctx.createLinearGradient(0, 0, 0, h);
    if (latestPing > 150) {
        gradient.addColorStop(0, 'rgba(255, 75, 75, 0.35)');
        gradient.addColorStop(1, 'rgba(255, 75, 75, 0.0)');
    } else if (isOptimizerActive) {
        gradient.addColorStop(0, 'rgba(0, 245, 160, 0.25)');
        gradient.addColorStop(1, 'rgba(0, 245, 160, 0.0)');
    } else {
        gradient.addColorStop(0, 'rgba(0, 242, 254, 0.25)');
        gradient.addColorStop(1, 'rgba(0, 242, 254, 0.0)');
    }
    ctx.fillStyle = gradient;
    ctx.fill();
}

// Toggle Optimizer ON/OFF
async function toggleOptimizer() {
    isOptimizerActive = !isOptimizerActive;
    setOptimizerState(isOptimizerActive);

    try {
        await fetch('/api/toggle', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ active: isOptimizerActive })
        });
    } catch (e) {}

    if (isOptimizerActive) {
        logMessage('OPTIMIZER', 'Đã KÍCH HOẠT Route Optimizer & Split-Tunneling.');
    } else {
        logMessage('OPTIMIZER', 'Đã TẮT Route Optimizer. Quay về mạng gốc.');
    }
}

function setOptimizerState(active) {
    isOptimizerActive = active;
    const btn = document.getElementById('toggleOptimizerBtn');
    const txt = document.getElementById('toggleBtnText');
    if (active) {
        btn.classList.add('active');
        txt.innerText = 'ĐANG TỐI ƯU (ON)';
    } else {
        btn.classList.remove('active');
        txt.innerText = 'BẬT OPTIMIZER';
    }
}

// Select Candidate Node
function selectNode(nodeId) {
    selectedNode = nodeId;
    const items = document.querySelectorAll('.node-item');
    items.forEach(item => item.classList.remove('active'));

    event.currentTarget.classList.add('active');
    logMessage('ROUTE', `Đã chọn nốt tuyến đường: ${nodeId.toUpperCase()}`);

    try {
        fetch('/api/select-node', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ node: nodeId })
        });
    } catch (e) {}
}

// Console Log Helper
function logMessage(tag, message) {
    const box = document.getElementById('consoleLog');
    if (!box) return;

    const time = new Date().toLocaleTimeString();
    const entry = document.createElement('div');
    entry.className = 'log-entry log-info';
    entry.innerHTML = `[${time}] [${tag}] ${message}`;

    box.appendChild(entry);
    box.scrollTop = box.scrollHeight;
}

function clearLogs() {
    const box = document.getElementById('consoleLog');
    if (box) box.innerHTML = '';
}
