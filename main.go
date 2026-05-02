package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

var Version = "v2.2.3"

//go:embed all:frontend/dist
var frontendFS embed.FS

var (
	lastCpuAlert  time.Time
	lastDdosAlert time.Time

	cachedDomains   []DomainInfo
	lastDomainCheck time.Time
)

type SystemStats struct {
	CPU         float64 `json:"cpu"`
	RAM         float64 `json:"ram"`
	RAMTotal    uint64  `json:"ram_total"`
	RAMUsed     uint64  `json:"ram_used"`
	Disk        float64 `json:"disk"`
	DiskTotal   uint64  `json:"disk_total"`
	DiskUsed    uint64  `json:"disk_used"`
	Uptime      uint64  `json:"uptime"`
	Hostname    string  `json:"hostname"`
	OS          string  `json:"os"`
	Platform    string  `json:"platform"`
	Kernel      string  `json:"kernel"`
	NetSent     uint64  `json:"net_sent"`
	NetRecv     uint64  `json:"net_recv"`
	Connections int     `json:"connections"`
	Timestamp   int64   `json:"timestamp"`
	Version     string  `json:"version"`
}

type ProcessInfo struct {
	PID     int32   `json:"pid"`
	Name    string  `json:"name"`
	CPU     float64 `json:"cpu"`
	Memory  float64 `json:"memory"`
	Command string  `json:"command"`
}

type DockerInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Image  string `json:"image"`
	CPU    string `json:"cpu"`
	MEM    string `json:"mem"`
}

type DomainInfo struct {
	Domain string `json:"domain"`
	Status string `json:"status"` // online, offline
	Code   int    `json:"code"`
	Note   string `json:"note,omitempty"`
}

type domainPaths struct {
	sitesEnabledDir   string
	sitesAvailableDir string
	nginxLogDir       string
}

type domainDeleteResult struct {
	Domain      string   `json:"domain"`
	Deleted     []string `json:"deleted"`
	Database    string   `json:"database,omitempty"`
	RootPath    string   `json:"root_path,omitempty"`
	DeleteDB    bool     `json:"delete_db"`
	DeleteRoot  bool     `json:"delete_root"`
	NginxReload bool     `json:"nginx_reload"`
}

var (
	domainNamePattern = regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
	dbNamePattern     = regexp.MustCompile(`^[a-zA-Z0-9_.$-]+$`)
)

func getDomainPaths() domainPaths {
	paths := domainPaths{
		sitesEnabledDir:   "/etc/nginx/sites-enabled",
		sitesAvailableDir: "/etc/nginx/sites-available",
		nginxLogDir:       "/var/log/nginx",
	}
	if runtime.GOOS == "windows" {
		paths.sitesEnabledDir = "./logs/sites-enabled"
		paths.sitesAvailableDir = "./logs/sites-available"
		paths.nginxLogDir = "./logs/nginx"
	}
	return paths
}

func clearDomainCache() {
	cachedDomains = nil
	lastDomainCheck = time.Time{}
}

func getDomainNotesPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(".", "data", "domain-notes.json")
	}
	return filepath.Join("/usr/local/bin", "data", "domain-notes.json")
}

func loadDomainNotes() map[string]string {
	path := getDomainNotesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}

	notes := map[string]string{}
	if err := json.Unmarshal(data, &notes); err != nil {
		return map[string]string{}
	}
	return notes
}

func saveDomainNotes(notes map[string]string) error {
	path := getDomainNotesPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func updateDomainNote(domain string, note string) error {
	notes := loadDomainNotes()
	note = strings.TrimSpace(note)
	if note == "" {
		delete(notes, domain)
	} else {
		notes[domain] = note
	}

	if err := saveDomainNotes(notes); err != nil {
		return err
	}

	clearDomainCache()
	return nil
}

func sanitizeDomain(domain string) (string, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" || !domainNamePattern.MatchString(domain) || strings.Contains(domain, "..") {
		return "", fmt.Errorf("invalid domain")
	}
	return domain, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func getDomainConfigCandidates(domain string) []string {
	paths := getDomainPaths()
	return []string{
		filepath.Join(paths.sitesEnabledDir, domain),
		filepath.Join(paths.sitesEnabledDir, domain+".conf"),
		filepath.Join(paths.sitesAvailableDir, domain),
		filepath.Join(paths.sitesAvailableDir, domain+".conf"),
	}
}

func findDomainConfigPath(domain string) (string, error) {
	for _, path := range getDomainConfigCandidates(domain) {
		if fileExists(path) {
			return path, nil
		}
	}
	return "", fmt.Errorf("domain config not found")
}

func parseNginxRoot(configPath string) (string, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if !strings.HasPrefix(line, "root ") {
			continue
		}
		root := strings.TrimSpace(strings.TrimPrefix(line, "root"))
		root = strings.TrimSuffix(root, ";")
		root = strings.Trim(root, `"'`)
		if root != "" {
			return filepath.Clean(root), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("root directive not found")
}

func getAppRoot(rootPath string) string {
	cleanRoot := filepath.Clean(rootPath)
	if strings.EqualFold(filepath.Base(cleanRoot), "public") {
		return filepath.Dir(cleanRoot)
	}
	return cleanRoot
}

func isAllowedRootDeletePath(path string) bool {
	cleanPath := filepath.Clean(path)
	if cleanPath == "" || cleanPath == "." || cleanPath == string(filepath.Separator) {
		return false
	}

	allowedPrefixes := []string{
		filepath.Clean("/var/www"),
		filepath.Clean("/home"),
		filepath.Clean("/srv/www"),
		filepath.Clean("/opt"),
	}
	if runtime.GOOS == "windows" {
		allowedPrefixes = []string{
			filepath.Clean("./logs/www"),
			filepath.Clean("./www"),
		}
	}

	for _, prefix := range allowedPrefixes {
		if cleanPath == prefix || strings.HasPrefix(cleanPath, prefix+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"'`)
		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func removeIfExists(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

func removeAllIfExists(path string) error {
	if !fileExists(path) {
		return nil
	}
	return os.RemoveAll(path)
}

func dropDatabaseFromEnv(appRoot string) (string, error) {
	envPath := filepath.Join(appRoot, ".env")
	envValues, err := parseEnvFile(envPath)
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %w", envPath, err)
	}

	dbName := envValues["DB_DATABASE"]
	if dbName == "" {
		return "", fmt.Errorf("DB_DATABASE not found in %s", envPath)
	}
	if !dbNamePattern.MatchString(dbName) {
		return "", fmt.Errorf("database name is not allowed")
	}

	dbConn := strings.ToLower(envValues["DB_CONNECTION"])
	if dbConn != "" && dbConn != "mysql" && dbConn != "mariadb" {
		return "", fmt.Errorf("unsupported DB_CONNECTION: %s", dbConn)
	}

	dbUser := envValues["DB_USERNAME"]
	if dbUser == "" {
		return "", fmt.Errorf("DB_USERNAME not found in %s", envPath)
	}

	args := []string{"-u", dbUser}
	if host := envValues["DB_HOST"]; host != "" {
		args = append(args, "-h", host)
	}
	if port := envValues["DB_PORT"]; port != "" {
		args = append(args, "-P", port)
	}
	args = append(args, "-e", fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))

	cmd := exec.Command("mysql", args...)
	cmd.Env = os.Environ()
	if password := envValues["DB_PASSWORD"]; password != "" {
		cmd.Env = append(cmd.Env, "MYSQL_PWD="+password)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("drop database failed: %s", strings.TrimSpace(string(output)))
	}

	return dbName, nil
}

func reloadNginx() error {
	cmd := exec.Command("systemctl", "reload", "nginx")
	if output, err := cmd.CombinedOutput(); err == nil {
		return nil
	} else if fallbackOutput, fallbackErr := exec.Command("nginx", "-s", "reload").CombinedOutput(); fallbackErr != nil {
		return fmt.Errorf("systemctl reload nginx failed: %s | nginx -s reload failed: %s", strings.TrimSpace(string(output)), strings.TrimSpace(string(fallbackOutput)))
	}
	return nil
}

func deleteDomain(domain string, deleteDB bool, deleteRoot bool) (domainDeleteResult, error) {
	result := domainDeleteResult{
		Domain:     domain,
		DeleteDB:   deleteDB,
		DeleteRoot: deleteRoot,
	}

	configPath, err := findDomainConfigPath(domain)
	if err != nil {
		return result, err
	}

	rootPath, err := parseNginxRoot(configPath)
	if err == nil {
		result.RootPath = getAppRoot(rootPath)
	}

	if deleteRoot {
		if result.RootPath == "" {
			return result, fmt.Errorf("cannot detect root path from nginx config")
		}
		if !isAllowedRootDeletePath(result.RootPath) {
			return result, fmt.Errorf("root path is outside allowed delete scope: %s", result.RootPath)
		}
	}

	if deleteDB {
		if result.RootPath == "" {
			return result, fmt.Errorf("cannot detect app root for database lookup")
		}
		dbName, dbErr := dropDatabaseFromEnv(result.RootPath)
		if dbErr != nil {
			return result, dbErr
		}
		result.Database = dbName
		result.Deleted = append(result.Deleted, "database:"+dbName)
	}

	for _, configPath := range getDomainConfigCandidates(domain) {
		existed := fileExists(configPath)
		if err := removeIfExists(configPath); err != nil {
			return result, err
		}
		if existed {
			result.Deleted = append(result.Deleted, configPath)
		}
	}

	paths := getDomainPaths()
	logFiles := []string{
		filepath.Join(paths.nginxLogDir, domain+"_access.log"),
		filepath.Join(paths.nginxLogDir, domain+"_error.log"),
	}
	for _, logPath := range logFiles {
		existed := fileExists(logPath)
		if err := removeIfExists(logPath); err != nil {
			return result, err
		}
		if existed {
			result.Deleted = append(result.Deleted, logPath)
		}
	}

	if err := updateDomainNote(domain, ""); err != nil {
		return result, err
	}

	if deleteRoot {
		if err := removeAllIfExists(result.RootPath); err != nil {
			return result, err
		}
		result.Deleted = append(result.Deleted, result.RootPath)
	}

	if runtime.GOOS != "windows" {
		if err := reloadNginx(); err != nil {
			return result, err
		}
		result.NginxReload = true
	}

	clearDomainCache()
	return result, nil
}

func sendTelegram(message string) {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	if token == "" || chatID == "" {
		return
	}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	_, _ = http.PostForm(apiURL, url.Values{
		"chat_id": {chatID},
		"text":    {message},
	})
}

func getStats() SystemStats {
	vm, _ := mem.VirtualMemory()
	cpuPercent, _ := cpu.Percent(0, false)
	d, _ := disk.Usage("/")
	h, _ := host.Info()
	n, _ := net.IOCounters(false)
	c, _ := net.Connections("tcp")

	var netSent, netRecv uint64
	if len(n) > 0 {
		netSent = n[0].BytesSent
		netRecv = n[0].BytesRecv
	}

	stats := SystemStats{
		CPU:         cpuPercent[0],
		RAM:         vm.UsedPercent,
		RAMTotal:    vm.Total,
		RAMUsed:     vm.Used,
		Disk:        d.UsedPercent,
		DiskTotal:   d.Total,
		DiskUsed:    d.Used,
		Uptime:      h.Uptime,
		Hostname:    h.Hostname,
		OS:          runtime.GOOS,
		Platform:    h.Platform,
		Kernel:      h.KernelVersion,
		NetSent:     netSent,
		NetRecv:     netRecv,
		Connections: len(c),
		Timestamp:   time.Now().Unix(),
		Version:     Version,
	}

	if stats.CPU > 90.0 && time.Since(lastCpuAlert) > 5*time.Minute {
		msg := fmt.Sprintf("🚨 [CPU ALERT] VPS: %s\nLoad: %.1f%%", stats.Hostname, stats.CPU)
		go sendTelegram(msg)
		lastCpuAlert = time.Now()
	}

	if stats.Connections > 2000 && time.Since(lastDdosAlert) > 10*time.Minute {
		msg := fmt.Sprintf("⚠️ [DDoS ALERT] VPS: %s\nConnections: %d", stats.Hostname, stats.Connections)
		go sendTelegram(msg)
		lastDdosAlert = time.Now()
	}

	return stats
}

func getTopProcesses() []ProcessInfo {
	processes, err := process.Processes()
	if err != nil {
		return nil
	}

	var results []ProcessInfo
	for _, p := range processes {
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		name, _ := p.Name()
		cmd, _ := p.Cmdline()
		if cpu > 0.1 || mem > 0.1 {
			results = append(results, ProcessInfo{
				PID:     p.Pid,
				Name:    name,
				CPU:     cpu,
				Memory:  float64(mem),
				Command: cmd,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CPU > results[j].CPU
	})

	if len(results) > 10 {
		return results[:10]
	}
	return results
}

func getDockerStats() []DockerInfo {
	if runtime.GOOS == "windows" {
		return []DockerInfo{{Name: "demo-container", Status: "Running", Image: "nginx:latest", CPU: "0.5%", MEM: "120MB"}}
	}
	// Use docker stats command for simplicity
	cmd := exec.Command("docker", "stats", "--no-stream", "--format", "{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}")
	output, _ := cmd.CombinedOutput()
	lines := strings.Split(string(output), "\n")

	// Also get statuses
	cmdStat := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}|{{.Status}}|{{.Image}}")
	outStat, _ := cmdStat.CombinedOutput()
	statLines := strings.Split(string(outStat), "\n")
	statusMap := make(map[string]DockerInfo)
	for _, l := range statLines {
		parts := strings.Split(l, "|")
		if len(parts) >= 3 {
			statusMap[parts[0]] = DockerInfo{Name: parts[0], Status: parts[1], Image: parts[2]}
		}
	}

	var results []DockerInfo
	for _, l := range lines {
		parts := strings.Split(l, "|")
		if len(parts) >= 3 {
			name := parts[0]
			if info, ok := statusMap[name]; ok {
				info.CPU = parts[1]
				info.MEM = parts[2]
				results = append(results, info)
			}
		}
	}
	return results
}

func getPM2Stats() interface{} {
	if runtime.GOOS == "windows" {
		return []map[string]interface{}{
			{"name": "demo-api", "pm_id": 0, "status": "online", "monit": map[string]interface{}{"cpu": 1.2, "memory": 45000000}, "pm2_env": map[string]interface{}{"pm_uptime": time.Now().Unix() * 1000}},
		}
	}
	cmd := exec.Command("pm2", "jlist")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return []interface{}{}
	}
	var data interface{}
	_ = json.Unmarshal(output, &data)
	return data
}

func getDomains(scan bool) []DomainInfo {
	notes := loadDomainNotes()
	
	// Nếu không yêu cầu quét và đã có cache, trả về cache
	if !scan && len(cachedDomains) > 0 {
		for i := range cachedDomains {
			cachedDomains[i].Note = notes[cachedDomains[i].Domain]
		}
		return cachedDomains
	}

	sitesEnabledDir := getDomainPaths().sitesEnabledDir
	files, err := os.ReadDir(sitesEnabledDir)
	if err != nil {
		return []DomainInfo{}
	}

	var domains []string
	for _, f := range files {
		if f.IsDir() || f.Name() == "default" || f.Name() == "phpmyadmin" {
			continue
		}
		domains = append(domains, strings.TrimSuffix(f.Name(), ".conf"))
	}

	results := make([]DomainInfo, len(domains))
	if scan {
		type resChan struct {
			index int
			info  DomainInfo
		}
		ch := make(chan resChan, len(domains))

		for i, d := range domains {
			go func(index int, domain string) {
				client := http.Client{Timeout: 3 * time.Second}
				resp, err := client.Head("http://" + domain)
				status := "online"
				code := 0
				if err != nil {
					status = "offline"
				} else {
					code = resp.StatusCode
					resp.Body.Close()
				}
				ch <- resChan{index, DomainInfo{Domain: domain, Status: status, Code: code, Note: notes[domain]}}
			}(i, d)
		}

		for i := 0; i < len(domains); i++ {
			r := <-ch
			results[r.index] = r.info
		}
		lastDomainCheck = time.Now()
		cachedDomains = results
	} else {
		// Chỉ liệt kê, không quét
		for i, d := range domains {
			status := "unknown"
			code := 0
			// Thử lấy lại trạng thái từ cache cũ nếu có
			for _, c := range cachedDomains {
				if c.Domain == d {
					status = c.Status
					code = c.Code
					break
				}
			}
			results[i] = DomainInfo{Domain: d, Status: status, Code: code, Note: notes[d]}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Domain < results[j].Domain
	})

	return results
}

func getTail(path string, lines int) string {
	if runtime.GOOS == "windows" {
		return "Log viewer only supports Linux (Simulation Mode)."
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Sprintf("File %s not found.", path)
	}
	cmd := exec.Command("tail", "-n", fmt.Sprintf("%d", lines), path)
	output, _ := cmd.CombinedOutput()
	return string(output)
}

func getAllLogs() map[string]interface{} {
	logs := map[string]interface{}{
		"system": gin.H{
			"content": getTail("/var/log/syslog", 30),
			"path":    "/var/log/syslog",
		},
	}

	paths := getDomainPaths()
	nginxDir := paths.nginxLogDir + string(filepath.Separator)
	sitesEnabledDir := paths.sitesEnabledDir

	if runtime.GOOS == "windows" {
		_ = os.MkdirAll(nginxDir, 0755)
		_ = os.MkdirAll(sitesEnabledDir, 0755)
	}

	// 1. Get domains from sites-enabled (the source of truth)
	domains := []string{}
	siteFiles, err := os.ReadDir(sitesEnabledDir)
	if err == nil {
		for _, f := range siteFiles {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			if name == "default" || name == "phpmyadmin" {
				continue
			}
			domain := strings.TrimSuffix(name, ".conf")
			domains = append(domains, domain)
		}
	}

	// 2. Also check log directory for other potential logs (fallback)
	logFiles, _ := os.ReadDir(nginxDir)
	sitesMap := make(map[string]map[string]gin.H)

	// Pre-fill from sites-enabled
	for _, d := range domains {
		sitesMap[d] = make(map[string]gin.H)
		accPath := nginxDir + d + "_access.log"
		errPath := nginxDir + d + "_error.log"
		sitesMap[d]["access"] = gin.H{"content": getTail(accPath, 30), "path": accPath}
		sitesMap[d]["error"] = gin.H{"content": getTail(errPath, 30), "path": errPath}
	}

	// Scan log directory to catch any missed or differently named logs
	if logFiles != nil {
		for _, f := range logFiles {
			if f.IsDir() {
				continue
			}
			name := f.Name()
			path := nginxDir + name

			if name == "access.log" || name == "error.log" {
				key := "nginx_access"
				if name == "error.log" {
					key = "nginx_error"
				}
				logs[key] = gin.H{"content": getTail(path, 30), "path": path}
				continue
			}

			if strings.HasSuffix(name, "_access.log") {
				domain := strings.TrimSuffix(name, "_access.log")
				if _, ok := sitesMap[domain]; !ok {
					sitesMap[domain] = make(map[string]gin.H)
				}
				sitesMap[domain]["access"] = gin.H{"content": getTail(path, 30), "path": path}
			} else if strings.HasSuffix(name, "_error.log") {
				domain := strings.TrimSuffix(name, "_error.log")
				if _, ok := sitesMap[domain]; !ok {
					sitesMap[domain] = make(map[string]gin.H)
				}
				sitesMap[domain]["error"] = gin.H{"content": getTail(path, 30), "path": path}
			}
		}
	}

	var nginxSites []gin.H
	for domain, data := range sitesMap {
		site := gin.H{"domain": domain}
		if acc, ok := data["access"]; ok {
			site["access"] = acc
		}
		if err, ok := data["error"]; ok {
			site["error"] = err
		}
		nginxSites = append(nginxSites, site)
	}

	if len(nginxSites) > 0 {
		sort.Slice(nginxSites, func(i, j int) bool {
			return nginxSites[i]["domain"].(string) < nginxSites[j]["domain"].(string)
		})
		logs["nginx_sites"] = nginxSites
	}

	return logs
}

func main() {
	_ = godotenv.Load(".env")
	vFlag := flag.Bool("v", false, "Version")
	flag.Parse()
	if *vFlag {
		fmt.Printf("Version: %s\n", Version)
		return
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// --- Authentication Configuration ---
	adminUser := os.Getenv("ADMIN_USER")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPass := os.Getenv("ADMIN_PASS")
	if adminPass == "" {
		adminPass = "h5jH7Gv|5m+0" // Mật khẩu cố định theo yêu cầu
	}
	authToken := os.Getenv("AUTH_TOKEN")
	if authToken == "" {
		authToken = "acmadash_secret_token_2024"
	}

	authMiddleware := func(c *gin.Context) {
		// Kiểm tra token từ Header hoặc Query (cho SSE)
		token := c.GetHeader("Authorization")
		if token == "" {
			token = c.Query("token")
		}

		if token != authToken {
			c.JSON(401, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}

	r.POST("/api/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		if req.Username == adminUser && req.Password == adminPass {
			c.JSON(200, gin.H{
				"status": "ok",
				"token":  authToken,
			})
		} else {
			c.JSON(401, gin.H{"error": "Sai tài khoản hoặc mật khẩu"})
		}
	})

	// 1. API - Protected Group
	api := r.Group("/api")
	api.Use(authMiddleware)
	{
		api.GET("/stats", func(c *gin.Context) {
			c.JSON(200, getStats())
		})

		api.GET("/logs", func(c *gin.Context) {
			c.JSON(200, getAllLogs())
		})

		api.GET("/processes", func(c *gin.Context) {
			c.JSON(200, getTopProcesses())
		})

		api.GET("/docker", func(c *gin.Context) {
			c.JSON(200, getDockerStats())
		})

		api.POST("/control", func(c *gin.Context) {
			var req struct {
				Service string `json:"service"`
				Action  string `json:"action"` // start, stop, restart
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			services := map[string]string{
				"nginx":  "nginx",
				"php8.3": "php8.3-fpm",
				"php7.4": "php7.4-fpm",
				"mysql":  "mariadb",
			}

			target, ok := services[req.Service]
			if !ok {
				c.JSON(400, gin.H{"error": "Service not allowed"})
				return
			}

			cmd := exec.Command("systemctl", req.Action, target)
			err := cmd.Run()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"status": "ok"})
		})

		api.GET("/pm2", func(c *gin.Context) {
			c.JSON(200, getPM2Stats())
		})

		api.GET("/domains", func(c *gin.Context) {
			scan := c.Query("scan") == "true"
			c.JSON(200, getDomains(scan))
		})

		api.POST("/domains/delete", func(c *gin.Context) {
			var req struct {
				Domain     string `json:"domain"`
				DeleteDB   bool   `json:"delete_db"`
				DeleteRoot bool   `json:"delete_root"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			domain, err := sanitizeDomain(req.Domain)
			if err != nil {
				c.JSON(400, gin.H{"error": "Domain is not allowed"})
				return
			}

			result, err := deleteDomain(domain, req.DeleteDB, req.DeleteRoot)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{
				"status":  "ok",
				"message": fmt.Sprintf("Deleted domain %s", domain),
				"result":  result,
			})
		})

		api.POST("/domains/note", func(c *gin.Context) {
			var req struct {
				Domain string `json:"domain"`
				Note   string `json:"note"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			domain, err := sanitizeDomain(req.Domain)
			if err != nil {
				c.JSON(400, gin.H{"error": "Domain is not allowed"})
				return
			}

			note := strings.TrimSpace(req.Note)
			if len(note) > 500 {
				c.JSON(400, gin.H{"error": "Note is too long"})
				return
			}

			if err := updateDomainNote(domain, note); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{
				"status": "ok",
				"domain": domain,
				"note":   note,
			})
		})

		api.POST("/pm2/control", func(c *gin.Context) {
			var req struct {
				Name   string `json:"name"`
				Action string `json:"action"` // restart, stop, start, delete
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			cmd := exec.Command("pm2", req.Action, req.Name)
			err := cmd.Run()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"status": "ok"})
		})

		// SSE - Real-time Streaming
		api.GET("/stream", func(c *gin.Context) {
			c.Writer.Header().Set("Content-Type", "text/event-stream")
			c.Writer.Header().Set("Cache-Control", "no-cache")
			c.Writer.Header().Set("Connection", "keep-alive")
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			c.Stream(func(w io.Writer) bool {
				select {
				case <-ticker.C:
					stats := getStats()
					logs := getAllLogs()
					data, _ := json.Marshal(gin.H{
						"stats": stats,
						"logs":  logs,
					})
					c.SSEvent("message", string(data))
					return true
				case <-c.Request.Context().Done():
					return false
				}
			})
		})
	}

	// 3. Static Files
	publicFS, _ := fs.Sub(frontendFS, "frontend/dist")
	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api") {
			c.JSON(404, gin.H{"error": "Not Found"})
			return
		}
		trimPath := strings.TrimPrefix(path, "/")
		if trimPath == "" || trimPath == "/" {
			trimPath = "index.html"
		}
		data, err := fs.ReadFile(publicFS, trimPath)
		if err != nil {
			data, _ = fs.ReadFile(publicFS, "index.html")
			trimPath = "index.html"
		}
		contentType := "text/plain"
		switch {
		case strings.HasSuffix(trimPath, ".html"):
			contentType = "text/html"
		case strings.HasSuffix(trimPath, ".js"):
			contentType = "application/javascript"
		case strings.HasSuffix(trimPath, ".css"):
			contentType = "text/css"
		case strings.HasSuffix(trimPath, ".svg"):
			contentType = "image/svg+xml"
		}
		c.Data(200, contentType, data)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8900"
	}
	log.Printf("🚀 AcmaDash %s running on :%s\n", Version, port)
	r.Run(":" + port)
}
