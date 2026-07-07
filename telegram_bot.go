package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// --- Telegram API Structures ---

type TelegramUpdate struct {
	UpdateID      int                    `json:"update_id"`
	Message       *TelegramMessage       `json:"message,omitempty"`
	CallbackQuery *TelegramCallbackQuery `json:"callback_query,omitempty"`
}

type TelegramMessage struct {
	MessageID int `json:"message_id"`
	Chat      struct {
		ID   int64  `json:"id"`
		Type string `json:"type"`
	} `json:"chat"`
	Text string `json:"text"`
	From struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
	} `json:"from"`
}

type TelegramCallbackQuery struct {
	ID      string           `json:"id"`
	Message *TelegramMessage `json:"message,omitempty"`
	Data    string           `json:"data"`
	From    struct {
		ID int64 `json:"id"`
	} `json:"from"`
}

type TelegramInlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type TelegramInlineKeyboardMarkup struct {
	InlineKeyboard [][]TelegramInlineKeyboardButton `json:"inline_keyboard"`
}

type TelegramKeyboardButton struct {
	Text string `json:"text"`
}

type TelegramReplyKeyboardMarkup struct {
	Keyboard        [][]TelegramKeyboardButton `json:"keyboard"`
	ResizeKeyboard  bool                     `json:"resize_keyboard"`
	OneTimeKeyboard bool                     `json:"one_time_keyboard"`
}

// --- Telegram Bot Main Loop ---

func startTelegramBot() {
	log.Println("[TELEGRAM BOT] Initializing interactive Telegram Bot services...")
	offset := 0
	client := &http.Client{Timeout: 35 * time.Second}

	for {
		token := getSetting("telegram_bot_token", os.Getenv("TELEGRAM_BOT_TOKEN"))
		if token == "" {
			// Token not set yet, check again in 15 seconds
			time.Sleep(15 * time.Second)
			continue
		}

		urlStr := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=30", token, offset)
		resp, err := client.Get(urlStr)
		if err != nil {
			log.Printf("[TELEGRAM BOT ERROR] Connection failure: %v. Retrying in 10 seconds...\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		var updateResp struct {
			Ok     bool             `json:"ok"`
			Result []TelegramUpdate `json:"result"`
		}

		err = json.NewDecoder(resp.Body).Decode(&updateResp)
		resp.Body.Close()

		if err != nil {
			log.Printf("[TELEGRAM BOT ERROR] JSON decode failed: %v. Retrying in 10 seconds...\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		if !updateResp.Ok {
			log.Println("[TELEGRAM BOT ERROR] Telegram API returned ok=false (possibly invalid token). Retrying in 15 seconds...")
			time.Sleep(15 * time.Second)
			continue
		}

		for _, update := range updateResp.Result {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}

			if update.Message != nil {
				handleTelegramMessage(*update.Message)
			} else if update.CallbackQuery != nil {
				handleTelegramCallback(*update.CallbackQuery)
			}
		}
	}
}

// Check authorization based on Chat ID
func isAuthorized(chatID int64) bool {
	expectedChatIDStr := getSetting("telegram_chat_id", os.Getenv("TELEGRAM_CHAT_ID"))
	if expectedChatIDStr == "" {
		return false
	}
	expectedChatID, err := strconv.ParseInt(expectedChatIDStr, 10, 64)
	if err != nil {
		log.Printf("[TELEGRAM BOT ERROR] Invalid telegram_chat_id setting: %v\n", err)
		return false
	}
	return chatID == expectedChatID
}

// Send warning instruction if unauthorized
func sendUnauthorizedMessage(chatID int64) {
	text := fmt.Sprintf("⚠️ *CẢNH BÁO BẢO MẬT*\n\n"+
		"Tài khoản của bạn chưa được cấp quyền quản trị VPS.\n"+
		"👉 Chat ID của bạn là: `%d`\n\n"+
		"Để kích hoạt quyền quản trị, vui lòng cấu hình biến môi trường này vào file `.env` của AcmaDash:\n"+
		"`TELEGRAM_CHAT_ID=%d`\n\n"+
		"Sau đó bấm nút *Restart* trên panel để áp dụng cấu hình mới.", chatID, chatID)
	sendTelegramMessage(chatID, text, nil)
}

// --- Handler functions ---

func handleTelegramMessage(msg TelegramMessage) {
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	// Check if this chat is authorized
	if !isAuthorized(chatID) {
		sendUnauthorizedMessage(chatID)
		return
	}

	// Keyboard layout matching
	switch {
	case text == "/start" || text == "/help" || text == "💡 Help":
		welcomeText := "🤖 *CHÀO MỪNG BẠN ĐẾN VỚI ACMADASH BOT*\n\n" +
			"Bot quản trị VPS an toàn và bảo mật cao. Bạn có thể sử dụng các phím tắt nhanh bên dưới hoặc gõ lệnh trực tiếp.\n\n" +
			"📋 *Danh sách câu lệnh khả dụng:*\n" +
			"📊 /status - Xem tài nguyên VPS hiện tại\n" +
			"🌐 /domains - Trạng thái website & SSL\n" +
			"🐳 /docker - Quản lý Docker containers\n" +
			"⚙️ /services - Quản lý dịch vụ hệ thống\n" +
			"🛡️ /fw - Trạng thái tường lửa UFW\n" +
			"⚠️ /alerts - Bật/tắt cảnh báo quá tải\n" +
			"💻 `/cmd <lệnh>` - Thực thi lệnh Shell trực tiếp (Timeout 5s)\n\n" +
			"➕ *Quản lý Domain (cPanel Terminal):*\n" +
			"• `/adddomain <domain> static` - Thêm site tĩnh\n" +
			"• `/adddomain <domain> php [ver] [db:true|false] [ssl:true|false]` - Thêm site PHP (Ví dụ: `/adddomain test.com php 8.3 true true`)\n" +
			"• `/adddomain <domain> proxy <url>` - Thêm Reverse Proxy (Ví dụ: `/adddomain test.com proxy http://127.0.0.1:3000`)\n" +
			"• `/deldomain <domain>` - Xóa website & DB tương ứng"

		menuMarkup := TelegramReplyKeyboardMarkup{
			Keyboard: [][]TelegramKeyboardButton{
				{
					{Text: "📊 Status"},
					{Text: "🌐 Domains"},
					{Text: "🐳 Docker"},
				},
				{
					{Text: "🛡️ Firewall"},
					{Text: "⚙️ Services"},
					{Text: "💡 Help"},
				},
			},
			ResizeKeyboard:  true,
			OneTimeKeyboard: false,
		}
		sendTelegramMessage(chatID, welcomeText, menuMarkup)

	case text == "/status" || text == "📊 Status":
		sendTelegramMessage(chatID, "⏳ Đang thu thập dữ liệu hệ thống...", nil)
		statusText := getTelegramStatusText()
		sendTelegramMessage(chatID, statusText, nil)

	case text == "/domains" || text == "🌐 Domains":
		sendTelegramMessage(chatID, "⏳ Đang quét trạng thái các website...", nil)
		domainsText := getTelegramDomainsText()
		sendTelegramMessage(chatID, domainsText, nil)

	case text == "/docker" || text == "🐳 Docker":
		sendTelegramMessage(chatID, "⏳ Đang truy vấn Docker daemon...", nil)
		dockerText, markup := getTelegramDockerText()
		sendTelegramMessage(chatID, dockerText, markup)

	case text == "/services" || text == "⚙️ Services":
		sendTelegramMessage(chatID, "⏳ Đang kiểm tra trạng thái dịch vụ...", nil)
		servicesText, markup := getTelegramServicesText()
		sendTelegramMessage(chatID, servicesText, markup)

	case text == "/fw" || text == "🛡️ Firewall":
		sendTelegramMessage(chatID, "⏳ Đang truy cập cấu hình tường lửa UFW...", nil)
		fwText, markup := getTelegramFirewallText()
		sendTelegramMessage(chatID, fwText, markup)

	case text == "/alerts":
		alertText := toggleTelegramAlerts()
		sendTelegramMessage(chatID, alertText, nil)

	case strings.HasPrefix(text, "/adddomain"):
		args := strings.Fields(text)
		if len(args) < 3 {
			helpText := "⚠️ *Sai cú pháp!*\n" +
				"Sử dụng các mẫu sau:\n" +
				"1. Static HTML:\n" +
				"`/adddomain test.com static`\n\n" +
				"2. PHP Web (có/không tạo database và ssl):\n" +
				"`/adddomain test.com php 8.3 true true` (Domain | php | PHP Ver | Tạo DB? | SSL?)\n\n" +
				"3. Reverse Proxy:\n" +
				"`/adddomain test.com proxy http://127.0.0.1:3000` (Domain | proxy | URL đích)"
			sendTelegramMessage(chatID, helpText, nil)
			return
		}

		domain := args[1]
		domainType := args[2] // static, php, proxy
		phpVersion := "8.3"
		proxyPass := ""
		createDB := false
		ssl := false

		if domainType == "php" {
			if len(args) >= 4 {
				phpVersion = args[3]
			}
			if len(args) >= 5 {
				createDB = args[4] == "true"
			}
			if len(args) >= 6 {
				ssl = args[5] == "true"
			}
		} else if domainType == "proxy" {
			if len(args) < 4 {
				sendTelegramMessage(chatID, "⚠️ *Lỗi:* Vui lòng cung cấp địa chỉ URL proxy đích (ví dụ: `http://127.0.0.1:3000`).", nil)
				return
			}
			proxyPass = args[3]
		}

		sendTelegramMessage(chatID, fmt.Sprintf("⏳ Đang khởi tạo website *%s* (Kiểu: %s)...", domain, domainType), nil)
		msg, err := createDomainHelper(domain, domainType, phpVersion, proxyPass, createDB, ssl)
		if err != nil {
			sendTelegramMessage(chatID, fmt.Sprintf("❌ *Thất bại:* %s", err.Error()), nil)
			return
		}
		
		sendTelegramMessage(chatID, fmt.Sprintf("✅ *Thành công!*\n%s", msg), nil)

	case strings.HasPrefix(text, "/deldomain"):
		args := strings.Fields(text)
		if len(args) < 2 {
			sendTelegramMessage(chatID, "⚠️ *Cú pháp:* `/deldomain <domain>`", nil)
			return
		}
		domain := args[1]
		sendTelegramMessage(chatID, fmt.Sprintf("⏳ Đang xóa website *%s*...", domain), nil)
		
		result, err := deleteDomain(domain, true, true)
		if err != nil {
			sendTelegramMessage(chatID, fmt.Sprintf("❌ *Thất bại:* %s", err.Error()), nil)
			return
		}

		deletedInfo := "Không có thành phần nào bị xóa"
		if len(result.Deleted) > 0 {
			deletedInfo = strings.Join(result.Deleted, "\n• ")
		}
		sendTelegramMessage(chatID, fmt.Sprintf("✅ *Đã xóa thành công domain %s!*\n\n*Các thành phần đã dọn dẹp:*\n• %s", domain, deletedInfo), nil)

	case strings.HasPrefix(text, "/cmd "):
		cmdStr := strings.TrimPrefix(text, "/cmd ")
		sendTelegramMessage(chatID, fmt.Sprintf("⏳ Đang chạy lệnh: `%s`...", cmdStr), nil)
		resultText := runShellCommandTelegram(cmdStr)
		sendTelegramMessage(chatID, resultText, nil)

	default:
		sendTelegramMessage(chatID, "❓ Lệnh không hợp lệ. Vui lòng bấm *💡 Help* để xem danh sách lệnh hỗ trợ.", nil)
	}
}

func handleTelegramCallback(cb TelegramCallbackQuery) {
	chatID := cb.Message.Chat.ID
	messageID := cb.Message.MessageID
	data := cb.Data

	if !isAuthorized(chatID) {
		answerCallbackQuery(cb.ID, "Quyền truy cập bị từ chối")
		return
	}

	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		answerCallbackQuery(cb.ID, "Dữ liệu không hợp lệ")
		return
	}

	category := parts[0]
	action := parts[1]
	target := ""
	if len(parts) >= 3 {
		target = parts[2]
	}

	switch category {
	case "docker":
		if action == "refresh" {
			answerCallbackQuery(cb.ID, "Đang làm mới...")
			dockerText, markup := getTelegramDockerText()
			editTelegramMessage(chatID, messageID, dockerText, markup)
			return
		}

		// Ensure container ID matches allowed regex
		idPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
		if !idPattern.MatchString(target) {
			answerCallbackQuery(cb.ID, "Container ID không hợp lệ")
			return
		}

		var cmd *exec.Cmd
		if action == "restart" {
			answerCallbackQuery(cb.ID, "Đang khởi động lại container "+target)
			if runtime.GOOS == "windows" {
				time.Sleep(1 * time.Second)
			} else {
				cmd = exec.Command("docker", "restart", target)
				_ = cmd.Run()
			}
		} else if action == "stop" {
			answerCallbackQuery(cb.ID, "Đang dừng container "+target)
			if runtime.GOOS == "windows" {
				time.Sleep(1 * time.Second)
			} else {
				cmd = exec.Command("docker", "stop", target)
				_ = cmd.Run()
			}
		} else if action == "start" {
			answerCallbackQuery(cb.ID, "Đang khởi động container "+target)
			if runtime.GOOS == "windows" {
				time.Sleep(1 * time.Second)
			} else {
				cmd = exec.Command("docker", "start", target)
				_ = cmd.Run()
			}
		}

		time.Sleep(1 * time.Second) // Wait for state change
		dockerText, markup := getTelegramDockerText()
		editTelegramMessage(chatID, messageID, dockerText, markup)

	case "service":
		if action == "refresh" {
			answerCallbackQuery(cb.ID, "Đang làm mới...")
			servicesText, markup := getTelegramServicesText()
			editTelegramMessage(chatID, messageID, servicesText, markup)
			return
		}

		// Security: Restrict restart target
		validServices := map[string]bool{
			"nginx": true, "mysql": true, "mariadb": true, "redis": true,
			"php7.4-fpm": true, "php8.1-fpm": true, "php8.2-fpm": true, "php8.3-fpm": true,
			"vps-dashboard": true,
		}
		if !validServices[target] {
			answerCallbackQuery(cb.ID, "Dịch vụ không được hỗ trợ khởi động lại qua Telegram")
			return
		}

		answerCallbackQuery(cb.ID, "Đang khởi động lại "+target)

		if target == "vps-dashboard" {
			sendTelegramMessage(chatID, "🔄 *Đang khởi động lại AcmaDash...* Bot sẽ tạm thời offline trong 2-3 giây.", nil)
			go func() {
				time.Sleep(1 * time.Second)
				if runtime.GOOS != "windows" {
					_ = exec.Command("systemctl", "restart", "vps-dashboard").Run()
				}
				os.Exit(0)
			}()
			return
		}

		if runtime.GOOS != "windows" {
			_ = exec.Command("systemctl", "restart", target).Run()
		}
		time.Sleep(1500 * time.Millisecond) // Wait for restart

		servicesText, markup := getTelegramServicesText()
		editTelegramMessage(chatID, messageID, servicesText, markup)

	case "fw":
		if action == "refresh" {
			answerCallbackQuery(cb.ID, "Đang làm mới...")
			fwText, markup := getTelegramFirewallText()
			editTelegramMessage(chatID, messageID, fwText, markup)
			return
		}

		if action == "enable" {
			answerCallbackQuery(cb.ID, "Đang kích hoạt tường lửa...")
			if runtime.GOOS != "windows" {
				_ = exec.Command("ufw", "--force", "enable").Run()
			}
		} else if action == "disable" {
			answerCallbackQuery(cb.ID, "Đang vô hiệu hóa tường lửa...")
			if runtime.GOOS != "windows" {
				_ = exec.Command("ufw", "disable").Run()
			}
		}
		time.Sleep(1 * time.Second)

		fwText, markup := getTelegramFirewallText()
		editTelegramMessage(chatID, messageID, fwText, markup)

	case "alert":
		if action == "toggle" {
			answerCallbackQuery(cb.ID, "Đang cập nhật...")
			alertText := toggleTelegramAlerts()
			sendTelegramMessage(chatID, alertText, nil)
		}
	}
}

// --- Report Generators ---

func getTelegramStatusText() string {
	stats := getStats()
	uptimeStr := formatUptime(stats.Uptime)

	cpuModel := stats.CPUModel
	if cpuModel == "" {
		cpuModel = cachedCPUModel
	}
	cores := stats.CPUCores
	if cores <= 0 {
		cores = cachedCPUCores
	}

	return fmt.Sprintf("📊 *TRẠNG THÁI VPS THỜI GIAN THỰC*\n\n"+
		"🖥️ *Hệ thống:*\n"+
		"• Hostname: `%s`\n"+
		"• OS: `%s (%s)`\n"+
		"• Kernel: `%s`\n"+
		"• Uptime: `%s`\n\n"+
		"🔥 *Tài nguyên:*\n"+
		"• CPU Model: `%s` (%d cores)\n"+
		"• CPU Usage: `%s`\n"+
		"• RAM Usage: `%s` (`%.2f GB` / `%.2f GB`)\n"+
		"• Disk Usage: `%s` (`%.1f GB` / `%.1f GB`)\n"+
		"• Load Average: `%.2f, %.2f, %.2f`\n"+
		"• TCP Connections: `%d`\n\n"+
		"⚡ *Tốc độ Mạng:*\n"+
		"• Đã gửi: `%s` | Đã nhận: `%s`\n",
		stats.Hostname, stats.OS, stats.Platform, stats.Kernel, uptimeStr,
		cpuModel, cores,
		makeProgressBar(stats.CPU),
		makeProgressBar(stats.RAM), float64(stats.RAMUsed)/1024/1024/1024, float64(stats.RAMTotal)/1024/1024/1024,
		makeProgressBar(stats.Disk), float64(stats.DiskUsed)/1024/1024/1024, float64(stats.DiskTotal)/1024/1024/1024,
		stats.Load1, stats.Load5, stats.Load15,
		stats.Connections,
		formatBytes(stats.NetSent), formatBytes(stats.NetRecv),
	)
}

func getTelegramDomainsText() string {
	domains := getDomains(false)
	if len(domains) == 0 {
		return "🌐 *DANH SÁCH DOMAIN*\n\nKhông tìm thấy cấu hình website nào."
	}

	var sb strings.Builder
	sb.WriteString("🌐 *DANH SÁCH DOMAIN & SSL*\n\n")
	for _, d := range domains {
		statusEmoji := "🟢"
		if d.Status == "offline" {
			statusEmoji = "🔴"
		} else if d.Code >= 400 {
			statusEmoji = "🟡"
		}

		sslText := "Không hoạt động"
		if d.SSLActive {
			if d.SSLDays <= 0 {
				sslText = "⚠️ Hết hạn"
			} else {
				sslText = fmt.Sprintf("Còn %d ngày", d.SSLDays)
			}
		}

		starredMark := ""
		if d.IsStarred {
			starredMark = " ⭐"
		}

		sb.WriteString(fmt.Sprintf("%s *%s*%s\n", statusEmoji, d.Domain, starredMark))
		sb.WriteString(fmt.Sprintf("  • HTTP Status: `%d %s`\n", d.Code, strings.ToUpper(d.Status)))
		sb.WriteString(fmt.Sprintf("  • SSL Cert: `%s` (%s)\n", sslText, d.SSLIssuer))
		if d.Note != "" {
			sb.WriteString(fmt.Sprintf("  • Ghi chú: _%s_\n", d.Note))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func getTelegramDockerText() (string, TelegramInlineKeyboardMarkup) {
	containers := getDockerStats()
	var markup TelegramInlineKeyboardMarkup

	if len(containers) == 0 {
		return "🐳 *DOCKER CONTAINERS*\n\nKhông tìm thấy container Docker nào.", markup
	}

	var sb strings.Builder
	sb.WriteString("🐳 *DOCKER CONTAINERS*\n\n")

	var buttons [][]TelegramInlineKeyboardButton

	for _, c := range containers {
		statusEmoji := "🔴"
		if strings.Contains(strings.ToLower(c.Status), "up") || strings.Contains(strings.ToLower(c.Status), "running") {
			statusEmoji = "🟢"
		}

		sb.WriteString(fmt.Sprintf("%s *%s*\n", statusEmoji, c.Name))
		sb.WriteString(fmt.Sprintf("  • Trạng thái: `%s`\n", c.Status))
		sb.WriteString(fmt.Sprintf("  • Image: `%s`\n", c.Image))
		sb.WriteString(fmt.Sprintf("  • CPU / MEM: `%s` / `%s`\n\n", c.CPU, c.MEM))

		if statusEmoji == "🟢" {
			buttons = append(buttons, []TelegramInlineKeyboardButton{
				{Text: "🔄 Restart " + c.Name, CallbackData: "docker:restart:" + c.Name},
				{Text: "🛑 Stop " + c.Name, CallbackData: "docker:stop:" + c.Name},
			})
		} else {
			buttons = append(buttons, []TelegramInlineKeyboardButton{
				{Text: "▶️ Start " + c.Name, CallbackData: "docker:start:" + c.Name},
			})
		}
	}

	buttons = append(buttons, []TelegramInlineKeyboardButton{
		{Text: "🔄 Làm mới danh sách", CallbackData: "docker:refresh:"},
	})

	markup.InlineKeyboard = buttons
	return sb.String(), markup
}

func getTelegramServicesText() (string, TelegramInlineKeyboardMarkup) {
	baseServices := []string{"nginx", "mysql", "redis", "vps-dashboard"}
	
	// Dynamically scan for installed php-fpm versions
	phpVersions := []string{"php7.4-fpm", "php8.1-fpm", "php8.2-fpm", "php8.3-fpm"}
	var services []string
	for _, bs := range baseServices {
		if serviceExists(bs) {
			services = append(services, bs)
		}
	}
	for _, php := range phpVersions {
		if serviceExists(php) {
			services = append(services, php)
		}
	}

	var sb strings.Builder
	sb.WriteString("⚙️ *DỊCH VỤ HỆ THỐNG*\n\n")

	var buttons [][]TelegramInlineKeyboardButton

	for _, s := range services {
		isActive := isServiceActive(s)
		statusEmoji := "🔴"
		statusText := "Không hoạt động"
		if isActive {
			statusEmoji = "🟢"
			statusText = "Đang chạy"
		}

		sb.WriteString(fmt.Sprintf("%s *%s*: `%s`\n", statusEmoji, s, statusText))

		buttons = append(buttons, []TelegramInlineKeyboardButton{
			{Text: "🔄 Restart " + s, CallbackData: "service:restart:" + s},
		})
	}

	buttons = append(buttons, []TelegramInlineKeyboardButton{
		{Text: "🔄 Làm mới trạng thái", CallbackData: "service:refresh:"},
	})

	var markup TelegramInlineKeyboardMarkup
	markup.InlineKeyboard = buttons
	return sb.String(), markup
}

func getTelegramFirewallText() (string, TelegramInlineKeyboardMarkup) {
	fw := getFirewallStatus()
	var markup TelegramInlineKeyboardMarkup

	statusEmoji := "🔴"
	statusText := "Đã tắt"
	if fw.Enabled {
		statusEmoji = "🟢"
		statusText = "Đang hoạt động"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🛡️ *TƯỜNG LỬA UFW*\n\n• Trạng thái: %s *%s*\n", statusEmoji, statusText))
	sb.WriteString(fmt.Sprintf("• Mặc định vào: `%s`\n", fw.DefaultIncoming))
	sb.WriteString(fmt.Sprintf("• Mặc định ra: `%s`\n\n", fw.DefaultOutgoing))

	if len(fw.Rules) > 0 {
		sb.WriteString("📋 *Luật tường lửa (UFW Rules):*\n")
		// Limit rule count to prevent giant lists
		maxRules := 15
		for i, r := range fw.Rules {
			if i >= maxRules {
				sb.WriteString(fmt.Sprintf("  • ... và %d luật khác.\n", len(fw.Rules)-maxRules))
				break
			}
			sb.WriteString(fmt.Sprintf("  • `[%d]` %s từ `%s` -> %s\n", r.Index, r.Action, r.From, r.To))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("⚠️ Chưa cấu hình luật tường lửa nào.\n\n")
	}

	if len(fw.ListeningPorts) > 0 {
		sb.WriteString("🔌 *Cổng đang mở (Listening Ports - Top 10):*\n")
		count := 0
		for _, p := range fw.ListeningPorts {
			if count >= 10 {
				sb.WriteString("  • ... và một số cổng khác.\n")
				break
			}
			sb.WriteString(fmt.Sprintf("  • Cổng `%s` (%s) - %s (PID: %s)\n", p.Port, p.Protocol, p.Process, p.Pid))
			count++
		}
	}

	var buttons [][]TelegramInlineKeyboardButton
	if fw.Enabled {
		buttons = append(buttons, []TelegramInlineKeyboardButton{
			{Text: "🔴 Tắt Tường lửa", CallbackData: "fw:disable:"},
		})
	} else {
		buttons = append(buttons, []TelegramInlineKeyboardButton{
			{Text: "🟢 Bật Tường lửa", CallbackData: "fw:enable:"},
		})
	}
	buttons = append(buttons, []TelegramInlineKeyboardButton{
		{Text: "🔄 Làm mới", CallbackData: "fw:refresh:"},
	})
	markup.InlineKeyboard = buttons

	return sb.String(), markup
}

func toggleTelegramAlerts() string {
	settings := loadSecuritySettings()
	settings.TelegramAlerts = !settings.TelegramAlerts
	_ = saveSecuritySettings(settings)

	status := "ĐÃ BẬT"
	if !settings.TelegramAlerts {
		status = "ĐÃ TẮT"
	}
	return fmt.Sprintf("🔔 Cấu hình gửi thông báo cảnh báo qua Telegram: *%s*", status)
}

func runShellCommandTelegram(cmdStr string) string {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", cmdStr)
	} else {
		cmd = exec.Command("/bin/sh", "-c", cmdStr)
	}

	// 5-second context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return "⚠️ *LỖI:* Câu lệnh chạy vượt quá thời gian tối đa cho phép (5 giây)."
	}

	outText := string(output)
	if outText == "" {
		if err != nil {
			return fmt.Sprintf("❌ *Thất bại:* %s", err.Error())
		}
		return "✅ Lệnh được thực thi thành công và không trả về output."
	}

	// Trim content to fit telegram limitations
	if len(outText) > 3500 {
		outText = outText[:3500] + "\n... (Bị cắt bớt do dữ liệu quá dài)"
	}

	return fmt.Sprintf("💻 *KẾT QUẢ THỰC THI:*\n```\n%s\n```", outText)
}

// --- Utilities ---

func makeProgressBar(percent float64) string {
	width := 10
	completed := int(percent / 10.0)
	if completed > width {
		completed = width
	} else if completed < 0 {
		completed = 0
	}

	bar := ""
	for i := 0; i < completed; i++ {
		bar += "█"
	}
	for i := completed; i < width; i++ {
		bar += "░"
	}
	return fmt.Sprintf("[%s] %.1f%%", bar, percent)
}

func formatUptime(uptimeSeconds uint64) string {
	days := uptimeSeconds / (24 * 3600)
	hours := (uptimeSeconds % (24 * 3600)) / 3600
	minutes := (uptimeSeconds % 3600) / 60

	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d ngày", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d giờ", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d phút", minutes))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d giây", uptimeSeconds)
	}
	return strings.Join(parts, ", ")
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func serviceExists(name string) bool {
	if runtime.GOOS == "windows" {
		return name == "vps-dashboard" || name == "nginx" || name == "mysql" || name == "php8.3-fpm"
	}
	cmd := exec.Command("systemctl", "list-unit-files", name+".service")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), name+".service")
}

func isServiceActive(name string) bool {
	if runtime.GOOS == "windows" {
		return true // Mock running on Windows
	}
	cmd := exec.Command("systemctl", "is-active", name)
	err := cmd.Run()
	return err == nil
}

// --- Telegram HTTP Helpers ---

func sendTelegramMessage(chatID int64, text string, replyMarkup interface{}) {
	token := getSetting("telegram_bot_token", os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	formData := url.Values{
		"chat_id":    {strconv.FormatInt(chatID, 10)},
		"text":       {text},
		"parse_mode": {"Markdown"},
	}

	if replyMarkup != nil {
		markupBytes, err := json.Marshal(replyMarkup)
		if err == nil {
			formData.Set("reply_markup", string(markupBytes))
		}
	}

	resp, err := http.PostForm(apiURL, formData)
	if err != nil {
		log.Printf("[TELEGRAM BOT ERROR] Failed to send message: %v\n", err)
		return
	}
	resp.Body.Close()
}

func editTelegramMessage(chatID int64, messageID int, text string, replyMarkup interface{}) {
	token := getSetting("telegram_bot_token", os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", token)
	formData := url.Values{
		"chat_id":    {strconv.FormatInt(chatID, 10)},
		"message_id": {strconv.Itoa(messageID)},
		"text":       {text},
		"parse_mode": {"Markdown"},
	}

	if replyMarkup != nil {
		markupBytes, err := json.Marshal(replyMarkup)
		if err == nil {
			formData.Set("reply_markup", string(markupBytes))
		}
	}

	resp, err := http.PostForm(apiURL, formData)
	if err != nil {
		log.Printf("[TELEGRAM BOT ERROR] Failed to edit message: %v\n", err)
		return
	}
	resp.Body.Close()
}

func answerCallbackQuery(callbackQueryID string, text string) {
	token := getSetting("telegram_bot_token", os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		return
	}

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/answerCallbackQuery", token)
	formData := url.Values{
		"callback_query_id": {callbackQueryID},
		"text":              {text},
	}

	resp, err := http.PostForm(apiURL, formData)
	if err != nil {
		log.Printf("[TELEGRAM BOT ERROR] Failed to answer callback query: %v\n", err)
		return
	}
	resp.Body.Close()
}
