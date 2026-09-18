# 📖 HƯỚNG DẪN SỬ DỤNG - LOL EXIT LAG (MANUAL GUIDE)

> **Hệ thống Tối ưu hóa Mạng & Giảm Ping League of Legends (LoL VN) dành cho người chơi từ Nhật Bản & Quốc tế.**

---

## 🌟 1. TỔNG QUAN HỆ THỐNG

Ứng dụng **`lol-exit-lag`** được thiết kế để giải quyết bài toán:
* **Vấn đề**: Khi ở Nhật Bản, đường truyền của nhà mạng quốc tế (ISP) thường đi vòng qua các nốt trung gian xa xôi (Hong Kong, Singapore...) dẫn đến Ping cao (80 - 120ms) và Jitter trồi sụt gây giật lag.
* **Giải pháp**:
  1. **Đầu cầu Việt Nam (Exit Node)**: Một chiếc điện thoại Android (như Xiaomi 14T Pro) đặt tại nhà ở Việt Nam, kết nối mạng Wi-Fi nội địa thông qua **Tailscale**.
  2. **Máy tính ở Nhật (Client)**: Ứng dụng **`lol-exit-lag.exe`** chạy trên Windows, kết nối về điện thoại ở Việt Nam và tự động bẻ gói tin game LoL qua đường truyền này với cơ chế **1-Click** siêu nhẹ.

---

## 🚀 2. QUY TRÌNH SỬ DỤNG THỰC TẾ HÀNG NGÀY

### Bước 1: Chuẩn bị thiết bị ở nhà tại Việt Nam (Làm 1 lần)
* Chiếc điện thoại Android (Xiaomi) đặt ở nhà, cắm sạc liên tục để không hết pin.
* Mở app **Tailscale** trên điện thoại, đảm bảo nút trạng thái luôn ở chế độ **Connected (Màu xanh)**.
* *(Gợi ý: Cài đặt màn hình chờ tự tắt và để điện thoại ở nơi thoáng mát).*

### Bước 2: Bật kết nối trên Máy tính (Khi bạn ở Nhật)
1. Mở ứng dụng **Tailscale** trên máy tính Windows.
2. Nhấp chuột phải vào biểu tượng **Tailscale** ở góc dưới cùng bên phải màn hình (gần chiếc đồng hồ).
3. Rê chuột vào mục **Exit Nodes** -> Bấm chọn vào tên chiếc điện thoại: **`xiaomi-14t-pro-1`**.
   *(Khi chọn đúng, cạnh tên điện thoại sẽ có dấu tích ✅).*

### Bước 3: Khởi chạy LoL Exit Lag Dashboard
1. Nhấp đúp chuột vào file **`lol-exit-lag.exe`** tại thư mục dự án:
   `C:\Users\mrahn\.gemini\antigravity-ide\scratch\lol-route\lol-exit-lag.exe`
2. Cửa sổ ứng dụng **Desktop Dashboard** với giao diện Gaming Dark Mode sẽ tự động bật lên.
3. Bấm vào nút: **`[ BẬT OPTIMIZER ]`**.

### Bước 4: Vào Game LoL VN
1. Bật Client **Riot Games / League of Legends (VN)** lên và vào trận đấu (hoặc Phòng Tập / Practice Tool).
2. Khi bạn vừa vào trận:
   * Ứng dụng `lol-exit-lag` sẽ tự động hiển thị: **`Đang chạy: League of Legends`**.
   * Ô **MATCH SERVER** sẽ tự động bắt đúng IP server trận đấu (dải `103.149.x.x`).
   * Biểu đồ **Latency Realtime** sẽ vẽ sóng ping và hiển thị chỉ số Jitter, Packet Loss trực quan từng giây.

---

## 📊 3. Ý NGHĨA CÁC CHỈ SỐ TRÊN GIAO DIỆN

| Chỉ số | Ý nghĩa | Trạng thái lý tưởng |
| :--- | :--- | :--- |
| **LATENCY (PING)** | Thời gian phản hồi tín hiệu mạng (mili-giây). | **Xanh (< 40ms)**: Tuyệt vời.<br>**Vàng (40 - 70ms)**: Ổn định.<br>**Đỏ (> 80ms)**: Cần kiểm tra mạng. |
| **JITTER** | Độ biến động trễ giữa các gói tin liên tiếp. | **Càng nhỏ càng tốt (< 5ms)**. Jitter thấp giúp combo chiêu mượt mà, không bị khựng hình. |
| **PACKET LOSS** | Tỷ lệ mất gói tin trên đường truyền. | **Bắt buộc 0.0%**. Mất gói tin sẽ gây hiện tượng "dịch chuyển tức thời" (teleport/ghosting). |
| **MATCH SERVER** | IP và cổng UDP server trận đấu mà LoL đang kết nối. | Dải IP chuẩn của máy chủ VNG / Riot VN2: `103.149.28.x`. |

---

## 🛠️ 4. XỬ LÝ SỰ CỐ THƯỜNG GẶP (TROUBLESHOOTING)

#### 1. Làm sao biết mạng máy tính đã thực sự đi qua điện thoại ở Việt Nam?
* Mở trình duyệt web trên máy tính vào trang [ipinfo.io](https://ipinfo.io).
* Nhìn dòng `"country"`: Nếu hiện **`"Vietnam"`** và tên nhà mạng là Viettel/FPT/VNPT tại nhà bạn -> **Thành công 100%**.

#### 2. Muốn tắt app hoàn toàn thì làm thế nào?
* Chỉ cần bấm nút **`[X]`** ở góc trên cửa sổ ứng dụng hoặc bấm tổ hợp phím **`Ctrl + Shift + Esc`** -> Chọn `lol-exit-lag.exe` -> Bấm **End task**.

#### 3. Khi không chơi game, lướt web bình thường thì sao?
* Bạn chỉ cần nhấp chuột phải vào icon **Tailscale** ở khay hệ thống -> **Exit Nodes** -> Chọn lại **`None`**.
* Mạng máy tính sẽ lập tức quay trở về dùng mạng gốc của Nhật Bản để tải file, xem YouTube độ phân giải cao nhanh nhất. Khi nào chơi LoL thì lại chọn sang Xiaomi!

---

## 🔗 Liên kết & Bản quyền
* Mã nguồn dự án: [github.com/codyzard/lol-exit-lag](https://github.com/codyzard/lol-exit-lag)
* Phát triển bằng: **Go (Golang)** + **HTML5 Canvas / Glassmorphism UI** + **Tailscale Engine**.
