package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	stdnet "net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var ruleRegex = regexp.MustCompile(`^\[\s*(\d+)\]\s+(.*?)\s+(ALLOW IN|DENY IN|ALLOW OUT|DENY OUT|ALLOW|DENY)\s+(.*)$`)

func registerSecurityRoutes(api *gin.RouterGroup) {
	api.GET("/security/settings", func(c *gin.Context) {
		c.JSON(200, loadSecuritySettings())
	})

	api.POST("/security/settings", func(c *gin.Context) {
		var settings SecuritySettings
		if err := c.BindJSON(&settings); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}
		if settings.BanThreshold <= 0 {
			settings.BanThreshold = 1
		}
		if err := saveSecuritySettings(settings); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok", "settings": settings})
	})

	api.GET("/security/logs", func(c *gin.Context) {
		c.JSON(200, loadSecurityLogs())
	})

	api.POST("/security/clear-logs", func(c *gin.Context) {
		if err := clearSecurityLogsSQL(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	api.POST("/security/ban", func(c *gin.Context) {
		var req struct {
			IP string `json:"ip"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		ip := strings.TrimSpace(req.IP)
		if stdnet.ParseIP(ip) == nil {
			c.JSON(400, gin.H{"error": "Invalid IP address"})
			return
		}

		if isPrivateOrLocalIP(ip) {
			c.JSON(400, gin.H{"error": "Cannot ban private or local IP addresses"})
			return
		}

		if runtime.GOOS == "windows" {
			_ = logSecurityEventSQL(SecurityLogEntry{
				IP:        ip,
				Timestamp: time.Now(),
				URI:       "Manual Ban",
				Domain:    "-",
				UserAgent: "-",
				Action:    "Simulation Banned Manually",
			})
			c.JSON(200, gin.H{"status": "ok", "message": "Simulation: Banned IP " + ip})
			return
		}

		cmd := exec.Command("ufw", "insert", "1", "deny", "from", ip, "to", "any")
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error(), "details": string(output)})
			return
		}

		_ = logSecurityEventSQL(SecurityLogEntry{
			IP:        ip,
			Timestamp: time.Now(),
			URI:       "Manual Ban",
			Domain:    "-",
			UserAgent: "-",
			Action:    "Banned Manually",
		})

		c.JSON(200, gin.H{"status": "ok", "message": string(output)})
	})

	api.POST("/security/unban", func(c *gin.Context) {
		var req struct {
			IP string `json:"ip"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		ip := strings.TrimSpace(req.IP)
		if stdnet.ParseIP(ip) == nil {
			c.JSON(400, gin.H{"error": "Invalid IP address"})
			return
		}

		if runtime.GOOS == "windows" {
			_ = logSecurityEventSQL(SecurityLogEntry{
				IP:        ip,
				Timestamp: time.Now(),
				URI:       "Manual Unban",
				Domain:    "-",
				UserAgent: "-",
				Action:    "Simulation Unbanned Manually",
			})
			c.JSON(200, gin.H{"status": "ok", "message": "Simulation: Unbanned IP " + ip})
			return
		}

		cmd := exec.Command("ufw", "delete", "deny", "from", ip)
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error(), "details": string(output)})
			return
		}

		_ = logSecurityEventSQL(SecurityLogEntry{
			IP:        ip,
			Timestamp: time.Now(),
			URI:       "Manual Unban",
			Domain:    "-",
			UserAgent: "-",
			Action:    "Unbanned Manually",
		})

		c.JSON(200, gin.H{"status": "ok", "message": string(output)})
	})

	// Firewall Endpoints
	api.GET("/firewall", func(c *gin.Context) {
		c.JSON(200, getFirewallStatus())
	})

	api.POST("/firewall/toggle", func(c *gin.Context) {
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		var cmd *exec.Cmd
		if req.Enabled {
			cmd = exec.Command("ufw", "--force", "enable")
		} else {
			cmd = exec.Command("ufw", "disable")
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error(), "details": string(output)})
			return
		}

		c.JSON(200, gin.H{"status": "ok", "message": string(output)})
	})

	api.POST("/firewall/rules", func(c *gin.Context) {
		var req struct {
			Port     string `json:"port"`
			Protocol string `json:"protocol"` // tcp, udp, all
			Action   string `json:"action"`   // allow, deny
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		portRegex := regexp.MustCompile(`^[0-9:-]+$`)
		if !portRegex.MatchString(req.Port) {
			c.JSON(400, gin.H{"error": "Invalid port format"})
			return
		}

		action := strings.ToLower(req.Action)
		if action != "allow" && action != "deny" {
			c.JSON(400, gin.H{"error": "Invalid action, must be allow or deny"})
			return
		}

		var arg string
		proto := strings.ToLower(req.Protocol)
		if proto == "all" || proto == "" {
			arg = req.Port
		} else if proto == "tcp" || proto == "udp" {
			arg = fmt.Sprintf("%s/%s", req.Port, proto)
		} else {
			c.JSON(400, gin.H{"error": "Invalid protocol, must be tcp, udp, or all"})
			return
		}

		cmd := exec.Command("ufw", action, arg)
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error(), "details": string(output)})
			return
		}

		c.JSON(200, gin.H{"status": "ok", "message": string(output)})
	})

	api.DELETE("/firewall/rules/:index", func(c *gin.Context) {
		indexStr := c.Param("index")
		var index int
		if _, err := fmt.Sscanf(indexStr, "%d", &index); err != nil || index <= 0 {
			c.JSON(400, gin.H{"error": "Invalid index parameter"})
			return
		}

		cmd := exec.Command("ufw", "--force", "delete", indexStr)
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error(), "details": string(output)})
			return
		}

		c.JSON(200, gin.H{"status": "ok", "message": string(output)})
	})
}

func getListeningPorts() []ListeningPort {
	var list []ListeningPort
	cmd := exec.Command("ss", "-tlnup")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return list
	}

	lines := strings.Split(string(output), "\n")
	processRegex := regexp.MustCompile(`users:\(\("([^"]+)",pid=(\d+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "tcp") && !strings.HasPrefix(line, "udp") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		netid := fields[0]    // tcp or udp
		localAddr := fields[4] // e.g. 0.0.0.0:3005 or [::]:8081 or *:6868

		addr := ""
		port := ""
		idxColon := strings.LastIndex(localAddr, ":")
		if idxColon != -1 {
			addr = localAddr[:idxColon]
			port = localAddr[idxColon+1:]
		}

		if idxPercent := strings.Index(addr, "%"); idxPercent != -1 {
			addr = addr[:idxPercent]
		}

		process := "unknown"
		pid := "-"
		processCol := fields[len(fields)-1]
		if match := processRegex.FindStringSubmatch(processCol); len(match) == 3 {
			process = match[1]
			pid = match[2]
		}

		found := false
		for i, item := range list {
			if item.Port == port && item.Protocol == netid && item.Address == addr {
				if item.Process == "unknown" && process != "unknown" {
					list[i].Process = process
					list[i].Pid = pid
				}
				found = true
				break
			}
		}

		if !found {
			list = append(list, ListeningPort{
				Port:     port,
				Protocol: netid,
				Address:  addr,
				Process:  process,
				Pid:      pid,
			})
		}
	}

	return list
}

func getFirewallStatus() FirewallStatus {
	cmd := exec.Command("ufw", "status", "numbered")
	output, err := cmd.CombinedOutput()
	
	enabled := false
	var rules []FirewallRule

	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Status: active") {
				enabled = true
				continue
			}
			if strings.HasPrefix(line, "Status: inactive") {
				enabled = false
				break
			}

			match := ruleRegex.FindStringSubmatch(line)
			if len(match) == 5 {
				idx := 0
				fmt.Sscanf(match[1], "%d", &idx)
				rules = append(rules, FirewallRule{
					Index:  idx,
					To:     strings.TrimSpace(match[2]),
					Action: strings.TrimSpace(match[3]),
					From:   strings.TrimSpace(match[4]),
				})
			}
		}
	}

	logging := "unknown"
	defaultIncoming := "deny"
	defaultOutgoing := "allow"
	defaultRouted := "deny"

	cmdVerbose := exec.Command("ufw", "status", "verbose")
	outputVerbose, errVerbose := cmdVerbose.CombinedOutput()
	if errVerbose == nil {
		linesVerbose := strings.Split(string(outputVerbose), "\n")
		for _, line := range linesVerbose {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Logging:") {
				logging = strings.TrimSpace(strings.TrimPrefix(line, "Logging:"))
			} else if strings.HasPrefix(line, "Default:") {
				val := strings.TrimSpace(strings.TrimPrefix(line, "Default:"))
				parts := strings.Split(val, ",")
				for _, part := range parts {
					part = strings.TrimSpace(part)
					if strings.Contains(part, "(incoming)") {
						defaultIncoming = strings.TrimSpace(strings.Split(part, " ")[0])
					} else if strings.Contains(part, "(outgoing)") {
						defaultOutgoing = strings.TrimSpace(strings.Split(part, " ")[0])
					} else if strings.Contains(part, "(routed)") {
						defaultRouted = strings.TrimSpace(strings.Split(part, " ")[0])
					}
				}
			}
		}
	}

	listeningPorts := getListeningPorts()

	return FirewallStatus{
		Enabled:         enabled,
		Logging:         logging,
		DefaultIncoming: defaultIncoming,
		DefaultOutgoing: defaultOutgoing,
		DefaultRouted:   defaultRouted,
		Rules:           rules,
		ListeningPorts:  listeningPorts,
	}
}

func loadSecuritySettings() SecuritySettings {
	defaultSettings := SecuritySettings{
		AutoBanEnabled: true,
		BanThreshold:   1,
		ProbePatterns:  []string{"/.env", ".env.", "/.git", "/wp-config.php"},
		TelegramAlerts: true,
	}

	val := getSetting("security_settings", "")
	if val == "" {
		return defaultSettings
	}

	var settings SecuritySettings
	if err := json.Unmarshal([]byte(val), &settings); err != nil {
		return defaultSettings
	}
	return settings
}

func saveSecuritySettings(settings SecuritySettings) error {
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	return saveSetting("security_settings", string(data))
}

func loadSecurityLogs() []SecurityLogEntry {
	logs, err := loadSecurityLogsSQL()
	if err != nil {
		return []SecurityLogEntry{}
	}
	return logs
}

func saveSecurityLogs(logs []SecurityLogEntry) error {
	// Dummy handler to satisfy any residual code
	return nil
}

func getBannedIPs() map[string]bool {
	banned := make(map[string]bool)
	if runtime.GOOS == "windows" {
		return banned
	}

	cmd := exec.Command("ufw", "status", "numbered")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return banned
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		match := ruleRegex.FindStringSubmatch(line)
		if len(match) == 5 {
			action := strings.ToUpper(match[3])
			from := strings.TrimSpace(match[4])
			if strings.Contains(action, "DENY") {
				ipParts := strings.Fields(from)
				if len(ipParts) > 0 {
					cleanIP := ipParts[0]
					if stdnet.ParseIP(cleanIP) != nil {
						banned[cleanIP] = true
					}
				}
			}
		}
	}
	return banned
}

func parseNginxAccessLogLine(line string) (ip, method, uri, ua string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", "", "", false
	}

	parts := strings.SplitN(line, " ", 2)
	if len(parts) < 2 {
		return "", "", "", "", false
	}
	ip = parts[0]

	firstQuote := strings.Index(line, "\"")
	secondQuote := -1
	if firstQuote != -1 {
		secondQuote = strings.Index(line[firstQuote+1:], "\"")
	}

	if firstQuote == -1 || secondQuote == -1 {
		return "", "", "", "", false
	}

	requestStr := line[firstQuote+1 : firstQuote+1+secondQuote]
	reqParts := strings.Split(requestStr, " ")
	if len(reqParts) >= 2 {
		method = reqParts[0]
		uri = reqParts[1]
	}

	lastQuoteOpen := strings.LastIndex(line, "\"")
	if lastQuoteOpen != -1 {
		matchingOpenQuote := strings.LastIndex(line[:lastQuoteOpen], "\"")
		if matchingOpenQuote != -1 && matchingOpenQuote > firstQuote+secondQuote+1 {
			ua = line[matchingOpenQuote+1 : lastQuoteOpen]
		}
	}

	return ip, method, uri, ua, true
}

func runIPSScan(offsets map[string]int64) {
	settings := loadSecuritySettings()
	if !settings.AutoBanEnabled {
		return
	}

	paths := getDomainPaths()
	nginxLogDir := paths.nginxLogDir

	var logFiles []string
	defaultAccessLog := filepath.Join(nginxLogDir, "access.log")
	if fileExists(defaultAccessLog) {
		logFiles = append(logFiles, defaultAccessLog)
	}

	entries, err := os.ReadDir(nginxLogDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, "_access.log") {
				logFiles = append(logFiles, filepath.Join(nginxLogDir, name))
			}
		}
	}

	bannedIPs := getBannedIPs()

	for _, logPath := range logFiles {
		fileInfo, err := os.Stat(logPath)
		if err != nil {
			continue
		}

		size := fileInfo.Size()
		offset, exists := offsets[logPath]

		if !exists {
			offsets[logPath] = size
			continue
		}
		if size < offset {
			offset = 0
		}
		if size == offset {
			continue
		}

		file, err := os.Open(logPath)
		if err != nil {
			continue
		}

		_, err = file.Seek(offset, 0)
		if err != nil {
			file.Close()
			continue
		}

		domain := "Default Server"
		baseName := filepath.Base(logPath)
		if strings.HasSuffix(baseName, "_access.log") {
			domain = strings.TrimSuffix(baseName, "_access.log")
		}

		scanner := bufio.NewScanner(file)
		ipAttempts := make(map[string]int)

		for scanner.Scan() {
			line := scanner.Text()
			ip, _, uri, ua, ok := parseNginxAccessLogLine(line)
			if !ok {
				continue
			}

			if stdnet.ParseIP(ip) == nil {
				continue
			}

			if isPrivateOrLocalIP(ip) {
				continue
			}

			isProbe := false
			for _, pattern := range settings.ProbePatterns {
				if strings.Contains(strings.ToLower(uri), strings.ToLower(pattern)) {
					isProbe = true
					break
				}
			}

			if isProbe {
				ipAttempts[ip]++

				if ipAttempts[ip] >= settings.BanThreshold {
					if bannedIPs[ip] {
						continue
					}

					actionResult := "Banned"
					if runtime.GOOS == "windows" {
						log.Printf("[IPS SIMULATION] Ban IP %s for probing %s on %s\n", ip, uri, domain)
						actionResult = "Simulation Blocked"
					} else {
						cmd := exec.Command("ufw", "insert", "1", "deny", "from", ip, "to", "any")
						output, cmdErr := cmd.CombinedOutput()
						if cmdErr != nil {
							log.Printf("[IPS ERROR] Failed to ban IP %s: %s\n", ip, string(output))
							actionResult = "Failed to Ban"
						} else {
							log.Printf("[IPS] Banned IP %s for probing %s on %s\n", ip, uri, domain)
							bannedIPs[ip] = true
						}
					}

					entry := SecurityLogEntry{
						IP:        ip,
						Timestamp: time.Now(),
						URI:       uri,
						Domain:    domain,
						UserAgent: ua,
						Action:    actionResult,
					}
					_ = logSecurityEventSQL(entry)

					if settings.TelegramAlerts {
						msg := fmt.Sprintf("🛡️ [Security Auto-Ban] VPS: %s\nIP: %s\nAction: %s\nProbed: %s\nDomain: %s\nUser Agent: %s",
							getHostname(), ip, actionResult, uri, domain, ua)
						go sendTelegram(msg)
					}
				}
			}
		}
		file.Close()
		offsets[logPath] = size
	}
}

func startIntrusionPreventionSystem() {
	log.Println("🛡️ Starting Intrusion Prevention System (IPS) worker...")
	offsets := make(map[string]int64)

	runIPSScan(offsets)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		runIPSScan(offsets)
	}
}

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "Unknown Host"
	}
	return h
}
