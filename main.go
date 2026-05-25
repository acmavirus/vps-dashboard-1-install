package main

import (
	"bufio"
	"database/sql"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	stdnet "net"
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

	_ "github.com/go-sql-driver/mysql"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
)

var Version = "v2.2.4"

//go:embed all:frontend/dist
var frontendFS embed.FS

var (
	adminUser = "admin"
	adminPass = "h5jH7Gv|5m+0"
	authToken = "acmadash_secret_token_2024"

	lastCpuAlert  time.Time
	lastDdosAlert time.Time

	cachedDomains   []DomainInfo
	lastDomainCheck time.Time
	cachedCPUModel  string
	cachedCPUCores  int
)

type SystemStats struct {
	CPU         float64 `json:"cpu"`
	RAM         float64 `json:"ram"`
	RAMTotal    uint64  `json:"ram_total"`
	RAMUsed     uint64  `json:"ram_used"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapPercent float64 `json:"swap_percent"`
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
	Load1       float64 `json:"load_1"`
	Load5       float64 `json:"load_5"`
	Load15      float64 `json:"load_15"`
	CPUCores    int     `json:"cpu_cores"`
	CPUModel    string  `json:"cpu_model"`
	DiskRead    uint64  `json:"disk_read"`
	DiskWrite   uint64  `json:"disk_write"`
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

func provisionCMSDatabase(prefix string) (string, string, string, error) {
	dbConfig, err := loadDBConfig()
	if err != nil || dbConfig.Host == "" {
		return "", "", "", fmt.Errorf("Please configure database credentials in the Databases tab first.")
	}

	// Automatically fix firewall and MariaDB bind-address on Host to allow Docker connection
	fixHostDatabaseForDocker()

	suffix := strings.ToLower(generateRandomString(6))
	dbName := fmt.Sprintf("%s_%s", prefix, suffix)
	dbUser := fmt.Sprintf("%s_u_%s", prefix, suffix)
	dbPass := generateRandomString(14)

	// Create database
	_, err = runSQLCommand(fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", dbName))
	if err != nil {
		return "", "", "", fmt.Errorf("Failed to create database: %w", err)
	}

	// Create user
	_, err = runSQLCommand(fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s';", dbUser, dbPass))
	if err != nil {
		_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbName))
		return "", "", "", fmt.Errorf("Failed to create database user: %w", err)
	}

	// Grant privileges
	_, err = runSQLCommand(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';", dbName, dbUser))
	if err != nil {
		_, _ = runSQLCommand(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", dbUser))
		_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbName))
		return "", "", "", fmt.Errorf("Failed to grant privileges: %w", err)
	}

	_, _ = runSQLCommand("FLUSH PRIVILEGES;")
	return dbName, dbUser, dbPass, nil
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

	// Check if this domain belongs to a Docker App Store instance
	metaList, _ := loadAppsMetadata()
	var dockerApp *AppMetadata
	for _, m := range metaList {
		if m.Domain == domain {
			dockerApp = &m
			break
		}
	}

	if dockerApp != nil {
		// 1. Remove docker container
		_ = exec.Command("docker", "rm", "-f", dockerApp.ID).Run()
		// 2. Remove docker volume
		_ = exec.Command("docker", "volume", "rm", dockerApp.ID+"_data").Run()
		result.Deleted = append(result.Deleted, "docker-container:"+dockerApp.ID)
		result.Deleted = append(result.Deleted, "docker-volume:"+dockerApp.ID+"_data")

		// 3. Drop DB and DB user
		if dockerApp.DBName != "" {
			_, _ = runSQLCommand(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", dockerApp.DBUser))
			_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dockerApp.DBName))
			_, _ = runSQLCommand("FLUSH PRIVILEGES;")
			result.Database = dockerApp.DBName
			result.Deleted = append(result.Deleted, "database:"+dockerApp.DBName)
		}

		// 4. Remove Nginx configuration files
		for _, configPath := range getDomainConfigCandidates(domain) {
			existed := fileExists(configPath)
			if err := removeIfExists(configPath); err == nil && existed {
				result.Deleted = append(result.Deleted, configPath)
			}
		}

		// 5. Remove log files
		paths := getDomainPaths()
		logFiles := []string{
			filepath.Join(paths.nginxLogDir, domain+"_access.log"),
			filepath.Join(paths.nginxLogDir, domain+"_error.log"),
		}
		for _, logPath := range logFiles {
			existed := fileExists(logPath)
			if err := removeIfExists(logPath); err == nil && existed {
				result.Deleted = append(result.Deleted, logPath)
			}
		}

		// 6. Remove note & metadata entry
		_ = updateDomainNote(domain, "")
		var remainingMeta []AppMetadata
		for _, item := range metaList {
			if item.Domain != domain {
				remainingMeta = append(remainingMeta, item)
			}
		}
		_ = saveAppsMetadata(remainingMeta)

		// 7. Reload Nginx
		if runtime.GOOS != "windows" {
			_ = reloadNginx()
			result.NginxReload = true
		}
		clearDomainCache()
		return result, nil
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
		notes := loadDomainNotes()
		note := notes[domain]
		var noteDBName, noteDBUser string
		for _, l := range strings.Split(note, "\n") {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "Database:") {
				noteDBName = strings.TrimSpace(strings.TrimPrefix(l, "Database:"))
			} else if strings.HasPrefix(l, "User:") {
				noteDBUser = strings.TrimSpace(strings.TrimPrefix(l, "User:"))
			}
		}

		if noteDBName != "" {
			if dbNamePattern.MatchString(noteDBName) {
				if noteDBUser != "" {
					_, _ = runSQLCommand(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", noteDBUser))
				}
				_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", noteDBName))
				_, _ = runSQLCommand("FLUSH PRIVILEGES;")
				result.Database = noteDBName
				result.Deleted = append(result.Deleted, "database:"+noteDBName)
			}
		} else {
			if result.RootPath != "" {
				envPath := filepath.Join(result.RootPath, ".env")
				if fileExists(envPath) {
					dbName, dbErr := dropDatabaseFromEnv(result.RootPath)
					if dbErr != nil {
						return result, dbErr
					}
					result.Database = dbName
					result.Deleted = append(result.Deleted, "database:"+dbName)
				}
			}
		}
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
	swap, _ := mem.SwapMemory()

	var netSent, netRecv uint64
	if len(n) > 0 {
		netSent = n[0].BytesSent
		netRecv = n[0].BytesRecv
	}

	var load1, load5, load15 float64
	if runtime.GOOS != "windows" {
		if l, err := load.Avg(); err == nil {
			load1 = l.Load1
			load5 = l.Load5
			load15 = l.Load15
		}
	}

	dIO, _ := disk.IOCounters()
	var diskRead, diskWrite uint64
	for _, io := range dIO {
		diskRead += io.ReadBytes
		diskWrite += io.WriteBytes
	}

	stats := SystemStats{
		CPU:         cpuPercent[0],
		RAM:         vm.UsedPercent,
		RAMTotal:    vm.Total,
		RAMUsed:     vm.Used,
		SwapTotal:   swap.Total,
		SwapUsed:    swap.Used,
		SwapPercent: swap.UsedPercent,
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
		Load1:       load1,
		Load5:       load5,
		Load15:      load15,
		CPUCores:    cachedCPUCores,
		CPUModel:    cachedCPUModel,
		DiskRead:    diskRead,
		DiskWrite:   diskWrite,
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

	// Cache CPU hardware specs at startup
	if info, err := cpu.Info(); err == nil && len(info) > 0 {
		cachedCPUModel = info[0].ModelName
	} else {
		cachedCPUModel = "Unknown CPU"
	}
	cachedCPUCores, _ = cpu.Counts(true)
	if cachedCPUCores <= 0 {
		cachedCPUCores = 1
	}

	vFlag := flag.Bool("v", false, "Version")
	flag.Parse()
	if *vFlag {
		fmt.Printf("Version: %s\n", Version)
		return
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// --- Authentication Configuration ---
	if u := os.Getenv("ADMIN_USER"); u != "" {
		adminUser = u
	}
	if p := os.Getenv("ADMIN_PASS"); p != "" {
		adminPass = p
	}
	if t := os.Getenv("AUTH_TOKEN"); t != "" {
		authToken = t
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

		api.POST("/domains/create", func(c *gin.Context) {
			var req struct {
				Domain      string `json:"domain"`
				Type        string `json:"type"`          // "static", "php", "proxy"
				PHPVersion  string `json:"php_version"`   // "8.3", "7.4"
				ProxyPass   string `json:"proxy_pass"`    // e.g. http://127.0.0.1:8080
				CreateDB    bool   `json:"create_db"`
				SSL         bool   `json:"ssl"`
			}

			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			domain, err := sanitizeDomain(req.Domain)
			if err != nil {
				c.JSON(400, gin.H{"error": "Invalid domain format"})
				return
			}

			// Check if Nginx config already exists for this domain
			paths := getDomainPaths()
			availablePath := filepath.Join(paths.sitesAvailableDir, domain+".conf")
			enabledPath := filepath.Join(paths.sitesEnabledDir, domain+".conf")

			if fileExists(availablePath) || fileExists(enabledPath) {
				c.JSON(400, gin.H{"error": "Website / Domain configuration already exists"})
				return
			}

			var noteContent []string

			// Handle directories and sample files if static/php
			if req.Type == "static" || req.Type == "php" {
				webRoot := filepath.Join("/var/www", domain)
				if runtime.GOOS == "windows" {
					webRoot = filepath.Join(".", "logs", "www", domain)
				}

				if err := os.MkdirAll(webRoot, 0755); err != nil {
					c.JSON(500, gin.H{"error": "Failed to create web root directory: " + err.Error()})
					return
				}

				if req.Type == "static" {
					indexPath := filepath.Join(webRoot, "index.html")
					if !fileExists(indexPath) {
						defaultHTML := fmt.Sprintf("<!DOCTYPE html><html><head><title>Welcome to %s</title></head><body style='font-family: sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #0f172a; color: #f1f5f9;'><div><h1>%s has been successfully configured!</h1><p>Website is running under Static HTML mode.</p></div></body></html>", domain, domain)
						_ = os.WriteFile(indexPath, []byte(defaultHTML), 0644)
					}
				} else if req.Type == "php" {
					indexPath := filepath.Join(webRoot, "index.php")
					if !fileExists(indexPath) {
						defaultPHP := fmt.Sprintf("<?php\necho '<h1>Welcome to %s</h1>';\necho '<p>Website is running under PHP Mode (PHP Version: %s)</p>';\nphpinfo();", domain, req.PHPVersion)
						_ = os.WriteFile(indexPath, []byte(defaultPHP), 0644)
					}
				}
			}

			// Handle Database Provisioning
			var dbName, dbUser, dbPass string
			if req.CreateDB {
				prefix := strings.ReplaceAll(domain, ".", "_")
				if len(prefix) > 10 {
					prefix = prefix[:10]
				}
				var dbErr error
				dbName, dbUser, dbPass, dbErr = provisionCMSDatabase(prefix)
				if dbErr != nil {
					c.JSON(500, gin.H{"error": "Database provisioning failed: " + dbErr.Error()})
					return
				}
				noteContent = append(noteContent, fmt.Sprintf("Database: %s\nUser: %s\nPass: %s", dbName, dbUser, dbPass))
			}

			// Generate Nginx Configuration
			var nginxConfig string
			if req.Type == "static" {
				webRoot := filepath.Join("/var/www", domain)
				nginxConfig = fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s;
    index index.html index.htm;

    location / {
        try_files $uri $uri/ =404;
    }
}
`, domain, webRoot)
			} else if req.Type == "php" {
				webRoot := filepath.Join("/var/www", domain)
				sockPath := "/run/php/php8.3-fpm.sock"
				if req.PHPVersion == "7.4" {
					sockPath = "/run/php/php7.4-fpm.sock"
				}
				nginxConfig = fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s;
    index index.php index.html index.htm;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:%s;
    }
}
`, domain, webRoot, sockPath)
			} else if req.Type == "proxy" {
				nginxConfig = fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location / {
        proxy_pass %s;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
`, domain, req.ProxyPass)
			} else {
				c.JSON(400, gin.H{"error": "Invalid website type"})
				return
			}

			// Write configuration file
			if err := os.MkdirAll(filepath.Dir(availablePath), 0755); err != nil {
				c.JSON(500, gin.H{"error": "Failed to create nginx config directory: " + err.Error()})
				return
			}
			if err := os.WriteFile(availablePath, []byte(nginxConfig), 0644); err != nil {
				c.JSON(500, gin.H{"error": "Failed to write nginx available config: " + err.Error()})
				return
			}

			// Symlink config
			if err := os.MkdirAll(filepath.Dir(enabledPath), 0755); err != nil {
				c.JSON(500, gin.H{"error": "Failed to create nginx enabled directory: " + err.Error()})
				return
			}
			_ = os.Remove(enabledPath)
			if err := os.Symlink(availablePath, enabledPath); err != nil {
				if runtime.GOOS != "windows" {
					c.JSON(500, gin.H{"error": "Failed to symlink nginx configuration: " + err.Error()})
					return
				}
			}

			// Save DB Note
			if len(noteContent) > 0 {
				noteStr := strings.Join(noteContent, "\n")
				_ = updateDomainNote(domain, noteStr)
			}

			// Reload Nginx
			if runtime.GOOS != "windows" {
				if err := reloadNginx(); err != nil {
					c.JSON(500, gin.H{"error": "Failed to reload Nginx: " + err.Error()})
					return
				}
				if req.SSL {
					runCertbot(domain)
				}
			}

			clearDomainCache()

			c.JSON(200, gin.H{
				"status":  "ok",
				"message": "Website created successfully",
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

		// File Manager Endpoints
		api.GET("/files", func(c *gin.Context) {
			path := c.Query("path")
			if path == "" {
				path = "/"
			}
			path = filepath.Clean(path)

			// Ensure path exists and is a directory
			info, err := os.Stat(path)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			if !info.IsDir() {
				c.JSON(400, gin.H{"error": "Path is not a directory"})
				return
			}

			entries, err := os.ReadDir(path)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			var files []FileInfo
			for _, entry := range entries {
				info, err := entry.Info()
				if err != nil {
					continue
				}
				files = append(files, FileInfo{
					Name:    entry.Name(),
					Size:    info.Size(),
					IsDir:   entry.IsDir(),
					ModTime: info.ModTime(),
					Mode:    info.Mode().String(),
				})
			}

			// Sort directories first, then files
			sort.Slice(files, func(i, j int) bool {
				if files[i].IsDir && !files[j].IsDir {
					return true
				}
				if !files[i].IsDir && files[j].IsDir {
					return false
				}
				return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
			})

			c.JSON(200, gin.H{
				"current_path": path,
				"files":        files,
			})
		})

		api.GET("/files/read", func(c *gin.Context) {
			path := c.Query("path")
			if path == "" {
				c.JSON(400, gin.H{"error": "Path is required"})
				return
			}
			path = filepath.Clean(path)

			info, err := os.Stat(path)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			if info.IsDir() {
				c.JSON(400, gin.H{"error": "Cannot read directory as text file"})
				return
			}
			if info.Size() > 5*1024*1024 {
				c.JSON(400, gin.H{"error": "File size exceeds 5MB limit for editor"})
				return
			}

			content, err := os.ReadFile(path)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{
				"path":    path,
				"content": string(content),
			})
		})

		api.POST("/files/write", func(c *gin.Context) {
			var req struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}
			path := filepath.Clean(req.Path)

			err := os.WriteFile(path, []byte(req.Content), 0644)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/files/create", func(c *gin.Context) {
			var req struct {
				Path  string `json:"path"`
				IsDir bool   `json:"is_dir"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}
			path := filepath.Clean(req.Path)

			if req.IsDir {
				err := os.MkdirAll(path, 0755)
				if err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}
			} else {
				file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					c.JSON(500, gin.H{"error": err.Error()})
					return
				}
				file.Close()
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/files/delete", func(c *gin.Context) {
			var req struct {
				Path string `json:"path"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}
			path := filepath.Clean(req.Path)

			// Safety checks
			if path == "/" || path == "/root" || path == "/etc" || path == "/bin" || path == "/usr" || path == "/var" {
				c.JSON(400, gin.H{"error": "Deleting critical system directories is blocked for safety"})
				return
			}

			err := os.RemoveAll(path)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/files/rename", func(c *gin.Context) {
			var req struct {
				OldPath string `json:"old_path"`
				NewPath string `json:"new_path"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}
			oldPath := filepath.Clean(req.OldPath)
			newPath := filepath.Clean(req.NewPath)

			if oldPath == "/" || newPath == "/" {
				c.JSON(400, gin.H{"error": "Renaming root directory is blocked"})
				return
			}

			err := os.Rename(oldPath, newPath)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		// Database Endpoints
		api.GET("/databases", func(c *gin.Context) {
			config, err := loadDBConfig()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			configured := config.Host != ""
			
			var list []string
			if configured {
				out, err := runSQLCommand("SHOW DATABASES;")
				if err != nil {
					c.JSON(200, gin.H{
						"configured":   configured,
						"host":         config.Host,
						"port":         config.Port,
						"username":     config.Username,
						"has_password": config.Password != "",
						"error":        err.Error(),
						"databases":    []string{},
					})
					return
				}
				lines := strings.Split(out, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" || line == "Database" {
						continue
					}
					// Skip system databases
					if line == "information_schema" || line == "performance_schema" || line == "mysql" || line == "sys" {
						continue
					}
					list = append(list, line)
				}
			}

			c.JSON(200, gin.H{
				"configured":   configured,
				"host":         config.Host,
				"port":         config.Port,
				"username":     config.Username,
				"has_password": config.Password != "",
				"databases":    list,
			})
		})

		api.POST("/databases/config", func(c *gin.Context) {
			var req struct {
				Host     string `json:"host"`
				Port     string `json:"port"`
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			if req.Host == "" || req.Port == "" || req.Username == "" {
				c.JSON(400, gin.H{"error": "Host, Port, and Username are required"})
				return
			}

			config := DBConfig{
				Host:     req.Host,
				Port:     req.Port,
				Username: req.Username,
				Password: req.Password,
			}

			// Test connection
			args := []string{"-h", config.Host, "-P", config.Port, "-u", config.Username, "-e", "SELECT 1;"}
			cmd := exec.Command("mysql", args...)
			cmd.Env = os.Environ()
			if config.Password != "" {
				cmd.Env = append(cmd.Env, "MYSQL_PWD="+config.Password)
			}
			output, err := cmd.CombinedOutput()
			if err != nil {
				c.JSON(400, gin.H{"error": fmt.Sprintf("Connection test failed: %s", strings.TrimSpace(string(output)))})
				return
			}

			err = saveDBConfig(config)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/databases", func(c *gin.Context) {
			var req struct {
				Name       string `json:"name"`
				CreateUser bool   `json:"create_user"`
				Username   string `json:"username"`
				Password   string `json:"password"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			dbNamePattern := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
			if !dbNamePattern.MatchString(req.Name) {
				c.JSON(400, gin.H{"error": "Database name must be alphanumeric and underscore only"})
				return
			}

			// 1. Create database
			_, err := runSQLCommand(fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", req.Name))
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			// 2. Create user if requested
			if req.CreateUser {
				if !dbNamePattern.MatchString(req.Username) {
					_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE `%s`;", req.Name))
					c.JSON(400, gin.H{"error": "Username must be alphanumeric and underscore only"})
					return
				}
				if req.Password == "" {
					_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE `%s`;", req.Name))
					c.JSON(400, gin.H{"error": "Password cannot be empty"})
					return
				}

				escapedPassword := strings.ReplaceAll(req.Password, "'", "''")

				_, err = runSQLCommand(fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s';", req.Username, escapedPassword))
				if err != nil {
					_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE `%s`;", req.Name))
					c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to create database user: %s", err.Error())})
					return
				}

				_, err = runSQLCommand(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';", req.Name, req.Username))
				if err != nil {
					_, _ = runSQLCommand(fmt.Sprintf("DROP USER '%s'@'%%';", req.Username))
					_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE `%s`;", req.Name))
					c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to grant privileges: %s", err.Error())})
					return
				}

				_, err = runSQLCommand("FLUSH PRIVILEGES;")
				if err != nil {
					c.JSON(500, gin.H{"error": fmt.Sprintf("Flush privileges failed: %s", err.Error())})
					return
				}
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.DELETE("/databases/:name", func(c *gin.Context) {
			name := c.Param("name")
			dbNamePattern := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
			if !dbNamePattern.MatchString(name) {
				c.JSON(400, gin.H{"error": "Invalid database name"})
				return
			}

			_, err := runSQLCommand(fmt.Sprintf("DROP DATABASE `%s`;", name))
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/databases/backup", func(c *gin.Context) {
			var req struct {
				Name string `json:"name"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			dbNamePattern := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
			if !dbNamePattern.MatchString(req.Name) {
				c.JSON(400, gin.H{"error": "Invalid database name"})
				return
			}

			file, err := runBackup(req.Name)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{
				"status": "ok",
				"file":   file,
			})
		})

		// Database Explorer Endpoints
		api.GET("/databases/:name/tables", func(c *gin.Context) {
			dbName := c.Param("name")
			if !safeIdentifier(dbName) {
				c.JSON(400, gin.H{"error": "Invalid database name"})
				return
			}

			// We can query SHOW TABLE STATUS
			res, err := executeCustomSQL(dbName, "SHOW TABLE STATUS;")
			if err != nil {
				// Fallback to simple SHOW TABLES if status fails
				res2, err2 := executeCustomSQL(dbName, "SHOW TABLES;")
				if err2 != nil {
					c.JSON(500, gin.H{"error": err2.Error()})
					return
				}
				
				// Transform simple SHOW TABLES output into a standardized list
				rows := res2["rows"].([][]interface{})
				var tables []gin.H
				for _, row := range rows {
					if len(row) > 0 {
						tables = append(tables, gin.H{
							"name":       row[0],
							"engine":     "InnoDB",
							"rows":       0,
							"data_size":  0,
							"collation":  "utf8mb4_unicode_ci",
							"comment":    "",
						})
					}
				}
				c.JSON(200, tables)
				return
			}

			// Standard columns for SHOW TABLE STATUS
			columns := res["columns"].([]string)
			rows := res["rows"].([][]interface{})

			nameIdx, engineIdx, rowsIdx, dataIdx, collationIdx, commentIdx := -1, -1, -1, -1, -1, -1
			for i, col := range columns {
				colLower := strings.ToLower(col)
				switch colLower {
				case "name":
					nameIdx = i
				case "engine":
					engineIdx = i
				case "rows":
					rowsIdx = i
				case "data_length":
					dataIdx = i
				case "collation":
					collationIdx = i
				case "comment":
					commentIdx = i
				}
			}

			var tables []gin.H
			for _, row := range rows {
				name := ""
				engine := ""
				var rowCount int64 = 0
				var dataSize int64 = 0
				collation := ""
				comment := ""

				if nameIdx >= 0 && nameIdx < len(row) && row[nameIdx] != nil {
					name = fmt.Sprintf("%v", row[nameIdx])
				}
				if engineIdx >= 0 && engineIdx < len(row) && row[engineIdx] != nil {
					engine = fmt.Sprintf("%v", row[engineIdx])
				}
				if rowsIdx >= 0 && rowsIdx < len(row) && row[rowsIdx] != nil {
					if val, ok := row[rowsIdx].(int64); ok {
						rowCount = val
					} else {
						fmt.Sscanf(fmt.Sprintf("%v", row[rowsIdx]), "%d", &rowCount)
					}
				}
				if dataIdx >= 0 && dataIdx < len(row) && row[dataIdx] != nil {
					if val, ok := row[dataIdx].(int64); ok {
						dataSize = val
					} else {
						fmt.Sscanf(fmt.Sprintf("%v", row[dataIdx]), "%d", &dataSize)
					}
				}
				if collationIdx >= 0 && collationIdx < len(row) && row[collationIdx] != nil {
					collation = fmt.Sprintf("%v", row[collationIdx])
				}
				if commentIdx >= 0 && commentIdx < len(row) && row[commentIdx] != nil {
					comment = fmt.Sprintf("%v", row[commentIdx])
				}

				if name != "" {
					tables = append(tables, gin.H{
						"name":       name,
						"engine":     engine,
						"rows":       rowCount,
						"data_size":  dataSize,
						"collation":  collation,
						"comment":    comment,
					})
				}
			}

			c.JSON(200, tables)
		})

		api.GET("/databases/:name/tables/:table/columns", func(c *gin.Context) {
			dbName := c.Param("name")
			tableName := c.Param("table")
			if !safeIdentifier(dbName) || !safeIdentifier(tableName) {
				c.JSON(400, gin.H{"error": "Invalid database or table name"})
				return
			}

			// We query: SHOW FULL COLUMNS FROM `table`
			query := fmt.Sprintf("SHOW FULL COLUMNS FROM `%s`.`%s`;", dbName, tableName)
			res, err := executeCustomSQL(dbName, query)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			columns := res["columns"].([]string)
			rows := res["rows"].([][]interface{})

			fieldIdx, typeIdx, collationIdx, nullIdx, keyIdx, defaultIdx, extraIdx, commentIdx := -1, -1, -1, -1, -1, -1, -1, -1
			for i, col := range columns {
				switch strings.ToLower(col) {
				case "field":
					fieldIdx = i
				case "type":
					typeIdx = i
				case "collation":
					collationIdx = i
				case "null":
					nullIdx = i
				case "key":
					keyIdx = i
				case "default":
					defaultIdx = i
				case "extra":
					extraIdx = i
				case "comment":
					commentIdx = i
				}
			}

			var tableColumns []gin.H
			for _, row := range rows {
				field := ""
				colType := ""
				collation := ""
				null := ""
				key := ""
				defVal := ""
				extra := ""
				comment := ""

				if fieldIdx >= 0 && fieldIdx < len(row) && row[fieldIdx] != nil {
					field = fmt.Sprintf("%v", row[fieldIdx])
				}
				if typeIdx >= 0 && typeIdx < len(row) && row[typeIdx] != nil {
					colType = fmt.Sprintf("%v", row[typeIdx])
				}
				if collationIdx >= 0 && collationIdx < len(row) && row[collationIdx] != nil {
					collation = fmt.Sprintf("%v", row[collationIdx])
				}
				if nullIdx >= 0 && nullIdx < len(row) && row[nullIdx] != nil {
					null = fmt.Sprintf("%v", row[nullIdx])
				}
				if keyIdx >= 0 && keyIdx < len(row) && row[keyIdx] != nil {
					key = fmt.Sprintf("%v", row[keyIdx])
				}
				if defaultIdx >= 0 && defaultIdx < len(row) && row[defaultIdx] != nil {
					defVal = fmt.Sprintf("%v", row[defaultIdx])
				}
				if extraIdx >= 0 && extraIdx < len(row) && row[extraIdx] != nil {
					extra = fmt.Sprintf("%v", row[extraIdx])
				}
				if commentIdx >= 0 && commentIdx < len(row) && row[commentIdx] != nil {
					comment = fmt.Sprintf("%v", row[commentIdx])
				}

				tableColumns = append(tableColumns, gin.H{
					"field":     field,
					"type":      colType,
					"collation": collation,
					"null":      null,
					"key":       key,
					"default":   defVal,
					"extra":     extra,
					"comment":   comment,
				})
			}

			c.JSON(200, tableColumns)
		})

		api.GET("/databases/:name/tables/:table/data", func(c *gin.Context) {
			dbName := c.Param("name")
			tableName := c.Param("table")
			if !safeIdentifier(dbName) || !safeIdentifier(tableName) {
				c.JSON(400, gin.H{"error": "Invalid database or table name"})
				return
			}

			limitStr := c.DefaultQuery("limit", "50")
			offsetStr := c.DefaultQuery("offset", "0")

			var limit, offset int
			fmt.Sscanf(limitStr, "%d", &limit)
			fmt.Sscanf(offsetStr, "%d", &offset)

			if limit <= 0 {
				limit = 50
			}
			if offset < 0 {
				offset = 0
			}

			// Get total count
			countQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`.`%s`;", dbName, tableName)
			countRes, err := executeCustomSQL(dbName, countQuery)
			var total int64 = 0
			if err == nil {
				rows := countRes["rows"].([][]interface{})
				if len(rows) > 0 && len(rows[0]) > 0 && rows[0][0] != nil {
					if val, ok := rows[0][0].(int64); ok {
						total = val
					} else {
						fmt.Sscanf(fmt.Sprintf("%v", rows[0][0]), "%d", &total)
					}
				}
			}

			// Get data rows
			dataQuery := fmt.Sprintf("SELECT * FROM `%s`.`%s` LIMIT %d OFFSET %d;", dbName, tableName, limit, offset)
			res, err := executeCustomSQL(dbName, dataQuery)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{
				"total":   total,
				"limit":   limit,
				"offset":  offset,
				"columns": res["columns"],
				"rows":    res["rows"],
			})
		})

		api.POST("/databases/:name/query", func(c *gin.Context) {
			dbName := c.Param("name")
			if !safeIdentifier(dbName) {
				c.JSON(400, gin.H{"error": "Invalid database name"})
				return
			}

			var req struct {
				Query string `json:"query"`
			}
			if err := c.BindJSON(&req); err != nil || req.Query == "" {
				c.JSON(400, gin.H{"error": "Invalid request, query is required"})
				return
			}

			res, err := executeCustomSQL(dbName, req.Query)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, res)
		})

		// App Store Endpoints
		api.GET("/apps", func(c *gin.Context) {
			catalog := []StoreApp{
				{
					ID:          "nginx-proxy-manager",
					Name:        "Nginx Proxy Manager",
					Description: "Easy reverse proxy manager with automated SSL/TLS certificates via Let's Encrypt.",
					Category:    "Web Proxy",
					DefaultPort: "81",
					Image:       "jc21/nginx-proxy-manager:latest",
					Status:      "not_installed",
				},
				{
					ID:          "phpmyadmin",
					Name:        "phpMyAdmin",
					Description: "Web UI interface to manage MySQL and MariaDB databases easily.",
					Category:    "Database GUI",
					DefaultPort: "8080",
					Image:       "phpmyadmin:latest",
					Status:      "not_installed",
				},
				{
					ID:          "redis-cache",
					Name:        "Redis",
					Description: "High-performance in-memory key-value database and caching store.",
					Category:    "Database",
					DefaultPort: "6379",
					Image:       "redis:alpine",
					Status:      "not_installed",
				},
				{
					ID:          "postgres-db",
					Name:        "PostgreSQL",
					Description: "Robust open-source object-relational database management system.",
					Category:    "Database",
					DefaultPort: "5432",
					Image:       "postgres:alpine",
					Status:      "not_installed",
				},
				{
					ID:          "mongodb-db",
					Name:        "MongoDB",
					Description: "Popular document-oriented database for storing JSON-like documents.",
					Category:    "Database",
					DefaultPort: "27017",
					Image:       "mongo:latest",
					Status:      "not_installed",
				},
				{
					ID:          "wordpress-app",
					Name:        "WordPress",
					Description: "World's most popular blogging software and content management system.",
					Category:    "CMS",
					DefaultPort: "8081",
					Image:       "wordpress:latest",
					Status:      "not_installed",
				},
				{
					ID:          "joomla-app",
					Name:        "Joomla",
					Description: "A powerful, flexible, and feature-rich Content Management System (CMS).",
					Category:    "CMS",
					DefaultPort: "8082",
					Image:       "joomla:latest",
					Status:      "not_installed",
				},
				{
					ID:          "drupal-app",
					Name:        "Drupal",
					Description: "An open-source content management platform for high-performance websites.",
					Category:    "CMS",
					DefaultPort: "8083",
					Image:       "drupal:latest",
					Status:      "not_installed",
				},
				{
					ID:          "ghost-app",
					Name:        "Ghost",
					Description: "A professional headless Node.js blogging and publication platform.",
					Category:    "CMS",
					DefaultPort: "2368",
					Image:       "ghost:alpine",
					Status:      "not_installed",
				},
				{
					ID:          "prestashop-app",
					Name:        "PrestaShop",
					Description: "A popular, fully customizable open-source e-commerce solution.",
					Category:    "CMS",
					DefaultPort: "8084",
					Image:       "prestashop/prestashop:latest",
					Status:      "not_installed",
				},
			}

			// Get status of container names
			cmd := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}|{{.State}}")
			output, err := cmd.CombinedOutput()
			statuses := make(map[string]string)
			if err == nil {
				lines := strings.Split(string(output), "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					parts := strings.SplitN(line, "|", 2)
					if len(parts) == 2 {
						statuses[parts[0]] = parts[1]
					}
				}
			}

			metaList, _ := loadAppsMetadata()
			var finalApps []StoreApp
			installedIDs := make(map[string]bool)

			// 1. Add all installed instances from metadata
			for _, m := range metaList {
				installedIDs[m.ID] = true

				// Skip instances that have a domain configured (as requested by user)
				if m.Domain != "" {
					continue
				}

				baseAppID := m.AppID
				if baseAppID == "" {
					baseAppID = m.ID
				}

				var baseApp StoreApp
				for _, ca := range catalog {
					if ca.ID == baseAppID {
						baseApp = ca
						break
					}
				}

				status := "stopped"
				if state, found := statuses[m.ID]; found && state == "running" {
					status = "running"
				}

				displayName := baseApp.Name
				if m.Domain != "" {
					displayName = fmt.Sprintf("%s (%s)", baseApp.Name, m.Domain)
				} else if m.ID != baseAppID {
					displayName = fmt.Sprintf("%s (%s)", baseApp.Name, m.ID)
				}

				finalApps = append(finalApps, StoreApp{
					ID:          m.ID,
					Name:        displayName,
					Description: baseApp.Description,
					Category:    baseApp.Category,
					DefaultPort: m.Port,
					Image:       baseApp.Image,
					Status:      status,
					Domain:      m.Domain,
				})
			}

			// 2. Add base catalog items so user can install new ones
			for _, ca := range catalog {
				// Prevent multiple installs of Nginx Proxy Manager
				if ca.ID == "nginx-proxy-manager" && installedIDs[ca.ID] {
					continue
				}
				ca.Status = "not_installed"
				finalApps = append(finalApps, ca)
			}

			c.JSON(200, finalApps)
		})

		api.POST("/apps/install", func(c *gin.Context) {
			var req struct {
				ID       string `json:"id"`
				Port     string `json:"port"`
				Password string `json:"password"`
				Domain   string `json:"domain"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			idPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
			if !idPattern.MatchString(req.ID) {
				c.JSON(400, gin.H{"error": "Invalid application ID"})
				return
			}

			portPattern := regexp.MustCompile(`^[0-9]+$`)
			if req.Port != "" && !portPattern.MatchString(req.Port) {
				c.JSON(400, gin.H{"error": "Invalid port number"})
				return
			}

			domainPattern := regexp.MustCompile(`^[a-zA-Z0-9.-]+$`)
			if req.Domain != "" && !domainPattern.MatchString(req.Domain) {
				c.JSON(400, gin.H{"error": "Invalid domain name"})
				return
			}

			var wpDBName, wpDBUser, wpDBPass string
			var runCmd *exec.Cmd
			port := req.Port
			if port == "" {
				switch req.ID {
				case "nginx-proxy-manager":
					port = "81"
				case "phpmyadmin":
					port = "8080"
				case "redis-cache":
					port = "6379"
				case "postgres-db":
					port = "5432"
				case "mongodb-db":
					port = "27017"
				case "wordpress-app":
					port = "8081"
				case "joomla-app":
					port = "8082"
				case "drupal-app":
					port = "8083"
				case "ghost-app":
					port = "2368"
				case "prestashop-app":
					port = "8084"
				}
			}

			// Determine unique container ID
			containerID := req.ID
			if req.ID != "nginx-proxy-manager" {
				if req.Domain != "" {
					safeDomain := strings.ReplaceAll(req.Domain, ".", "-")
					containerID = fmt.Sprintf("%s-%s", req.ID, safeDomain)
				} else {
					containerID = fmt.Sprintf("%s-%s", req.ID, generateRandomString(6))
				}
			}

			// Check if containerID already exists in metadata
			metaList, _ := loadAppsMetadata()
			for _, m := range metaList {
				if m.ID == containerID {
					c.JSON(400, gin.H{"error": "An instance with this domain or name already exists."})
					return
				}
			}

			// Validate if the port is already allocated in metadata to another container
			for _, m := range metaList {
				if m.Port == port {
					c.JSON(400, gin.H{"error": fmt.Sprintf("Cổng %s đã được sử dụng bởi ứng dụng khác (%s). Vui lòng chọn cổng khác.", port, m.ID)})
					return
				}
			}

			// Check if the port is actively listening on the host (by another process or a running docker container)
			if port != "" {
				ln, err := stdnet.Listen("tcp", ":"+port)
				if err != nil {
					c.JSON(400, gin.H{"error": fmt.Sprintf("Cổng %s đang bị chiếm dụng trên hệ thống. Vui lòng chọn cổng khác.", port)})
					return
				}
				ln.Close()
			}

			// Pre-cleanup: if containerID exists in docker (e.g. from an orphaned failed run), force remove it first to avoid exit status 125 conflict
			_ = exec.Command("docker", "rm", "-f", containerID).Run()

			switch req.ID {
			case "nginx-proxy-manager":
				if port == "" {
					port = "81"
				}
				runCmd = exec.Command("docker", "run", "-d",
					"--name", containerID,
					"--restart", "unless-stopped",
					"-p", "80:80",
					"-p", "443:443",
					"-p", port+":81",
					"-v", "/var/lib/nginx-proxy-manager:/data",
					"-v", "/var/lib/nginx-proxy-manager/letsencrypt:/etc/letsencrypt",
					"jc21/nginx-proxy-manager:latest")
			case "phpmyadmin":
				if port == "" {
					port = "8080"
				}
				runCmd = exec.Command("docker", "run", "-d",
					"--name", containerID,
					"--restart", "unless-stopped",
					"-p", port+":80",
					"-e", "PMA_ARBITRARY=1",
					"phpmyadmin:latest")
			case "redis-cache":
				if port == "" {
					port = "6379"
				}
				runCmd = exec.Command("docker", "run", "-d",
					"--name", containerID,
					"--restart", "unless-stopped",
					"-p", port+":6379",
					"-v", containerID+"_data:/data",
					"redis:alpine")
			case "postgres-db":
				if port == "" {
					port = "5432"
				}
				pwd := req.Password
				if pwd == "" {
					pwd = "postgres_secure_pass_2026"
				}
				runCmd = exec.Command("docker", "run", "-d",
					"--name", containerID,
					"--restart", "unless-stopped",
					"-p", port+":5432",
					"-v", containerID+"_data:/var/lib/postgresql/data",
					"-e", "POSTGRES_PASSWORD="+pwd,
					"postgres:alpine")
			case "mongodb-db":
				if port == "" {
					port = "27017"
				}
				pwd := req.Password
				if pwd == "" {
					pwd = "mongo_secure_pass_2026"
				}
				runCmd = exec.Command("docker", "run", "-d",
					"--name", containerID,
					"--restart", "unless-stopped",
					"-p", port+":27017",
					"-v", containerID+"_data:/data/db",
					"-e", "MONGO_INITDB_ROOT_USERNAME=admin",
					"-e", "MONGO_INITDB_ROOT_PASSWORD="+pwd,
					"mongo:latest")
			case "wordpress-app":
				if port == "" {
					port = "8081"
				}

				var dbErr error
				wpDBName, wpDBUser, wpDBPass, dbErr = provisionCMSDatabase("wp")
				if dbErr != nil {
					c.JSON(500, gin.H{"error": dbErr.Error()})
					return
				}

				dbConfig, _ := loadDBConfig()
				dbHostForDocker := dbConfig.Host
				if dbHostForDocker == "127.0.0.1" || dbHostForDocker == "localhost" {
					dbHostForDocker = "host.docker.internal"
				}
				dbHostPortForDocker := fmt.Sprintf("%s:%s", dbHostForDocker, dbConfig.Port)

				runCmd = exec.Command("docker", "run", "-d",
					"--name", containerID,
					"--add-host", "host.docker.internal:host-gateway",
					"--restart", "unless-stopped",
					"-p", port+":80",
					"-v", containerID+"_data:/var/www/html",
					"-e", "WORDPRESS_DB_HOST="+dbHostPortForDocker,
					"-e", "WORDPRESS_DB_USER="+wpDBUser,
					"-e", "WORDPRESS_DB_PASSWORD="+wpDBPass,
					"-e", "WORDPRESS_DB_NAME="+wpDBName,
					"wordpress:latest")
			case "joomla-app":
				if port == "" {
					port = "8082"
				}

				var dbErr error
				wpDBName, wpDBUser, wpDBPass, dbErr = provisionCMSDatabase("joomla")
				if dbErr != nil {
					c.JSON(500, gin.H{"error": dbErr.Error()})
					return
				}

				dbConfig, _ := loadDBConfig()
				dbHostForDocker := dbConfig.Host
				if dbHostForDocker == "127.0.0.1" || dbHostForDocker == "localhost" {
					dbHostForDocker = "host.docker.internal"
				}
				dbHostPortForDocker := fmt.Sprintf("%s:%s", dbHostForDocker, dbConfig.Port)

				runCmd = exec.Command("docker", "run", "-d",
					"--name", containerID,
					"--add-host", "host.docker.internal:host-gateway",
					"--restart", "unless-stopped",
					"-p", port+":80",
					"-v", containerID+"_data:/var/www/html",
					"-e", "JOOMLA_DB_HOST="+dbHostPortForDocker,
					"-e", "JOOMLA_DB_USER="+wpDBUser,
					"-e", "JOOMLA_DB_PASSWORD="+wpDBPass,
					"-e", "JOOMLA_DB_NAME="+wpDBName,
					"joomla:latest")
			case "drupal-app":
				if port == "" {
					port = "8083"
				}

				var dbErr error
				wpDBName, wpDBUser, wpDBPass, dbErr = provisionCMSDatabase("drupal")
				if dbErr != nil {
					c.JSON(500, gin.H{"error": dbErr.Error()})
					return
				}

				runCmd = exec.Command("docker", "run", "-d",
					"--name", containerID,
					"--add-host", "host.docker.internal:host-gateway",
					"--restart", "unless-stopped",
					"-p", port+":80",
					"-v", containerID+"_data:/var/www/html",
					"drupal:latest")
			case "ghost-app":
				if port == "" {
					port = "2368"
				}

				var dbErr error
				wpDBName, wpDBUser, wpDBPass, dbErr = provisionCMSDatabase("ghost")
				if dbErr != nil {
					c.JSON(500, gin.H{"error": dbErr.Error()})
					return
				}

				dbConfig, _ := loadDBConfig()
				dbHostForDocker := dbConfig.Host
				if dbHostForDocker == "127.0.0.1" || dbHostForDocker == "localhost" {
					dbHostForDocker = "host.docker.internal"
				}

				ghostURL := "http://localhost:" + port
				if req.Domain != "" {
					ghostURL = "http://" + req.Domain
				}

				runCmd = exec.Command("docker", "run", "-d",
					"--name", containerID,
					"--add-host", "host.docker.internal:host-gateway",
					"--restart", "unless-stopped",
					"-p", port+":2368",
					"-v", containerID+"_data:/var/lib/ghost/content",
					"-e", "url="+ghostURL,
					"-e", "database__client=mysql",
					"-e", "database__connection__host="+dbHostForDocker,
					"-e", "database__connection__port="+dbConfig.Port,
					"-e", "database__connection__user="+wpDBUser,
					"-e", "database__connection__password="+wpDBPass,
					"-e", "database__connection__database="+wpDBName,
					"ghost:alpine")
			case "prestashop-app":
				if port == "" {
					port = "8084"
				}

				var dbErr error
				wpDBName, wpDBUser, wpDBPass, dbErr = provisionCMSDatabase("presta")
				if dbErr != nil {
					c.JSON(500, gin.H{"error": dbErr.Error()})
					return
				}

				dbConfig, _ := loadDBConfig()
				dbHostForDocker := dbConfig.Host
				if dbHostForDocker == "127.0.0.1" || dbHostForDocker == "localhost" {
					dbHostForDocker = "host.docker.internal"
				}
				dbHostPortForDocker := fmt.Sprintf("%s:%s", dbHostForDocker, dbConfig.Port)

				runCmd = exec.Command("docker", "run", "-d",
					"--name", containerID,
					"--add-host", "host.docker.internal:host-gateway",
					"--restart", "unless-stopped",
					"-p", port+":80",
					"-v", containerID+"_data:/var/www/html",
					"-e", "DB_SERVER="+dbHostPortForDocker,
					"-e", "DB_USER="+wpDBUser,
					"-e", "DB_PASSWD="+wpDBPass,
					"-e", "DB_NAME="+wpDBName,
					"prestashop/prestashop:latest")
			default:
				c.JSON(400, gin.H{"error": "Unsupported application ID"})
				return
			}

			output, err := runCmd.CombinedOutput()
			if err != nil {
				// Clean up DB if container start failed
				if wpDBName != "" {
					_, _ = runSQLCommand(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", wpDBUser))
					_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", wpDBName))
					_, _ = runSQLCommand("FLUSH PRIVILEGES;")
				}
				c.JSON(500, gin.H{"error": fmt.Sprintf("%s: %s", err.Error(), strings.TrimSpace(string(output)))})
				return
			}

			// Save to apps_metadata.json (save first so that even if Nginx proxy fails, the app is registered as installed)
			updated := false
			newMeta := AppMetadata{
				ID:     containerID,
				AppID:  req.ID,
				Domain: req.Domain,
				Port:   port,
				DBName: wpDBName,
				DBUser: wpDBUser,
				DBPass: wpDBPass,
			}
			for i, m := range metaList {
				if m.ID == containerID {
					metaList[i] = newMeta
					updated = true
					break
				}
			}
			if !updated {
				metaList = append(metaList, newMeta)
			}
			_ = saveAppsMetadata(metaList)

			// Configure Nginx Proxy if domain is provided
			if req.Domain != "" {
				err = createNginxProxy(req.Domain, port)
				if err != nil {
					c.JSON(200, gin.H{
						"status":       "ok",
						"container_id": strings.TrimSpace(string(output)),
						"warning":      "Container started, but Nginx proxy creation failed: " + err.Error(),
					})
					return
				}
				runCertbot(req.Domain)

				// Save DB info to Domain Note so user can copy-paste it
				if wpDBName != "" {
					note := fmt.Sprintf("CMS: %s | DB: %s | User: %s | Pass: %s", req.ID, wpDBName, wpDBUser, wpDBPass)
					_ = updateDomainNote(req.Domain, note)
				}
			}

			c.JSON(200, gin.H{"status": "ok", "container_id": strings.TrimSpace(string(output))})
		})

		api.POST("/apps/uninstall", func(c *gin.Context) {
			var req struct {
				ID string `json:"id"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			idPattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
			if !idPattern.MatchString(req.ID) {
				c.JSON(400, gin.H{"error": "Invalid application ID"})
				return
			}

			// Stop and remove docker container
			cmd := exec.Command("docker", "rm", "-f", req.ID)
			output, err := cmd.CombinedOutput()
			if err != nil && !strings.Contains(string(output), "No such container") {
				c.JSON(500, gin.H{"error": fmt.Sprintf("%s: %s", err.Error(), strings.TrimSpace(string(output)))})
				return
			}

			// Try to remove associated volume (ignore errors if it doesn't exist)
			_ = exec.Command("docker", "volume", "rm", req.ID+"_data").Run()

			// Clean up Nginx reverse proxy and DB database/user
			metaList, _ := loadAppsMetadata()
			var remainingMeta []AppMetadata
			for _, m := range metaList {
				if m.ID == req.ID {
					if m.Domain != "" {
						_ = deleteNginxProxy(m.Domain)
					}
					if m.DBName != "" {
						_, _ = runSQLCommand(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", m.DBUser))
						_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", m.DBName))
						_, _ = runSQLCommand("FLUSH PRIVILEGES;")
					}
				} else {
					remainingMeta = append(remainingMeta, m)
				}
			}
			_ = saveAppsMetadata(remainingMeta)

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

		// --- FTP API Endpoints ---
		api.GET("/ftp", func(c *gin.Context) {
			users, err := listFtpUsers()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, users)
		})

		api.POST("/ftp/add", func(c *gin.Context) {
			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
				Path     string `json:"path"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			usernamePattern := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
			if !usernamePattern.MatchString(req.Username) {
				c.JSON(400, gin.H{"error": "Invalid username format"})
				return
			}

			if len(req.Password) < 6 {
				c.JSON(400, gin.H{"error": "Password must be at least 6 characters"})
				return
			}

			if err := os.MkdirAll(req.Path, 0755); err != nil {
				c.JSON(500, gin.H{"error": "Failed to create directory: " + err.Error()})
				return
			}
			_ = exec.Command("chown", "-R", "www-data:www-data", req.Path).Run()

			cmd := exec.Command("pure-pw", "useradd", req.Username, "-u", "www-data", "-d", req.Path)
			stdin, err := cmd.StdinPipe()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			go func() {
				defer stdin.Close()
				_, _ = io.WriteString(stdin, req.Password+"\n"+req.Password+"\n")
			}()

			output, err := cmd.CombinedOutput()
			if err != nil {
				c.JSON(500, gin.H{"error": fmt.Sprintf("pure-pw failed: %s %s", err.Error(), string(output))})
				return
			}

			if err := exec.Command("pure-pw", "mkdb").Run(); err != nil {
				c.JSON(500, gin.H{"error": "Failed to update FTP database: " + err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/ftp/delete", func(c *gin.Context) {
			var req struct {
				Username string `json:"username"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			output, err := exec.Command("pure-pw", "userdel", req.Username).CombinedOutput()
			if err != nil {
				c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to delete user: %s %s", err.Error(), string(output))})
				return
			}

			if err := exec.Command("pure-pw", "mkdb").Run(); err != nil {
				c.JSON(500, gin.H{"error": "Failed to update FTP database: " + err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/ftp/password", func(c *gin.Context) {
			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			if len(req.Password) < 6 {
				c.JSON(400, gin.H{"error": "Password must be at least 6 characters"})
				return
			}

			cmd := exec.Command("pure-pw", "passwd", req.Username)
			stdin, err := cmd.StdinPipe()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			go func() {
				defer stdin.Close()
				_, _ = io.WriteString(stdin, req.Password+"\n"+req.Password+"\n")
			}()

			output, err := cmd.CombinedOutput()
			if err != nil {
				c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to change password: %s %s", err.Error(), string(output))})
				return
			}

			if err := exec.Command("pure-pw", "mkdb").Run(); err != nil {
				c.JSON(500, gin.H{"error": "Failed to update FTP database: " + err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/ftp/toggle", func(c *gin.Context) {
			var req struct {
				Username string `json:"username"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			data, err := os.ReadFile("/etc/pure-ftpd/pureftpd.passwd")
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			lines := strings.Split(string(data), "\n")
			found := false
			for i, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					continue
				}
				isCommented := strings.HasPrefix(trimmed, "#")
				normalized := trimmed
				if isCommented {
					normalized = strings.TrimPrefix(trimmed, "#")
				}
				parts := strings.Split(normalized, ":")
				if len(parts) > 0 && parts[0] == req.Username {
					found = true
					if isCommented {
						lines[i] = normalized
					} else {
						lines[i] = "#" + trimmed
					}
					break
				}
			}

			if !found {
				c.JSON(404, gin.H{"error": "User not found"})
				return
			}

			err = os.WriteFile("/etc/pure-ftpd/pureftpd.passwd", []byte(strings.Join(lines, "\n")), 0600)
			if err != nil {
				c.JSON(500, gin.H{"error": "Failed to write passwd file: " + err.Error()})
				return
			}

			if err := exec.Command("pure-pw", "mkdb").Run(); err != nil {
				c.JSON(500, gin.H{"error": "Failed to update FTP database: " + err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		// --- Cron API Endpoints ---
		api.GET("/cron", func(c *gin.Context) {
			jobs, err := listCronJobs()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, jobs)
		})

		api.POST("/cron/add", func(c *gin.Context) {
			var req struct {
				Name     string `json:"name"`
				Schedule string `json:"schedule"`
				Command  string `json:"command"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			if req.Name == "" || req.Schedule == "" || req.Command == "" {
				c.JSON(400, gin.H{"error": "All fields are required"})
				return
			}

			jobs, err := listCronJobs()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			logDir := "/var/log/cron_tasks"
			_ = os.MkdirAll(logDir, 0755)

			id := fmt.Sprintf("cron_%d", time.Now().UnixNano())
			logPath := fmt.Sprintf("%s/%s.log", logDir, id)

			newJob := CronJob{
				ID:       id,
				Name:     req.Name,
				Schedule: req.Schedule,
				Command:  req.Command,
				Status:   "enabled",
				LogPath:  logPath,
				IsSystem: false,
			}

			jobs = append(jobs, newJob)
			if err := saveCronJobs(jobs); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/cron/delete", func(c *gin.Context) {
			var req struct {
				ID string `json:"id"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			jobs, err := listCronJobs()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			var remainingJobs []CronJob
			found := false
			for _, job := range jobs {
				if job.ID == req.ID {
					found = true
					if job.LogPath != "" {
						_ = os.Remove(job.LogPath)
					}
					continue
				}
				remainingJobs = append(remainingJobs, job)
			}

			if !found {
				c.JSON(404, gin.H{"error": "Cronjob not found"})
				return
			}

			if err := saveCronJobs(remainingJobs); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/cron/toggle", func(c *gin.Context) {
			var req struct {
				ID string `json:"id"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			jobs, err := listCronJobs()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			found := false
			for i, job := range jobs {
				if job.ID == req.ID {
					found = true
					if job.Status == "enabled" {
						jobs[i].Status = "disabled"
					} else {
						jobs[i].Status = "enabled"
					}
					break
				}
			}

			if !found {
				c.JSON(404, gin.H{"error": "Cronjob not found"})
				return
			}

			if err := saveCronJobs(jobs); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.GET("/cron/log", func(c *gin.Context) {
			id := c.Query("id")
			if id == "" {
				c.JSON(400, gin.H{"error": "ID is required"})
				return
			}

			logPath := fmt.Sprintf("/var/log/cron_tasks/%s.log", id)
			data, err := os.ReadFile(logPath)
			if err != nil {
				if os.IsNotExist(err) {
					c.JSON(200, gin.H{"log": "[No logs generated yet. Executing task will write to this file.]"})
					return
				}
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"log": string(data)})
		})

		api.POST("/cron/log/clear", func(c *gin.Context) {
			var req struct {
				ID string `json:"id"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			logPath := fmt.Sprintf("/var/log/cron_tasks/%s.log", req.ID)
			_ = os.WriteFile(logPath, []byte(""), 0644)
			c.JSON(200, gin.H{"status": "ok"})
		})

		// --- Settings API Endpoints ---
		api.GET("/settings", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"username":     adminUser,
				"version":      Version,
				"go_version":   runtime.Version(),
				"os":           runtime.GOOS + "/" + runtime.GOARCH,
				"num_cpu":      runtime.NumCPU(),
				"goroutines":   runtime.NumGoroutine(),
			})
		})

		api.POST("/settings/update", func(c *gin.Context) {
			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			if err := c.BindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "Invalid request"})
				return
			}

			if req.Username == "" {
				c.JSON(400, gin.H{"error": "Username cannot be empty"})
				return
			}

			if req.Password != "" && len(req.Password) < 6 {
				c.JSON(400, gin.H{"error": "Password must be at least 6 characters"})
				return
			}

			adminUser = req.Username
			if req.Password != "" {
				adminPass = req.Password
			}

			if err := saveSettingsToEnv(adminUser, adminPass); err != nil {
				c.JSON(500, gin.H{"error": "Failed to save settings: " + err.Error()})
				return
			}

			c.JSON(200, gin.H{"status": "ok"})
		})

		api.POST("/settings/restart", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
			go func() {
				time.Sleep(1 * time.Second)
				_ = exec.Command("systemctl", "restart", "vps-dashboard").Run()
				os.Exit(0)
			}()
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

type FirewallRule struct {
	Index  int    `json:"index"`
	To     string `json:"to"`
	Action string `json:"action"`
	From   string `json:"from"`
}

type ListeningPort struct {
	Port     string `json:"port"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Process  string `json:"process"`
	Pid      string `json:"pid"`
}

type FirewallStatus struct {
	Enabled         bool            `json:"enabled"`
	Logging         string          `json:"logging"`
	DefaultIncoming string          `json:"default_incoming"`
	DefaultOutgoing string          `json:"default_outgoing"`
	DefaultRouted   string          `json:"default_routed"`
	Rules           []FirewallRule  `json:"rules"`
	ListeningPorts  []ListeningPort `json:"listening_ports"`
}

var ruleRegex = regexp.MustCompile(`^\[\s*(\d+)\]\s+(.*?)\s+(ALLOW IN|DENY IN|ALLOW OUT|DENY OUT|ALLOW|DENY)\s+(.*)$`)

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

		// Extract address and port
		addr := ""
		port := ""
		idxColon := strings.LastIndex(localAddr, ":")
		if idxColon != -1 {
			addr = localAddr[:idxColon]
			port = localAddr[idxColon+1:]
		}

		// Clean up address (remove %interface name if present, e.g. 127.0.0.53%lo)
		if idxPercent := strings.Index(addr, "%"); idxPercent != -1 {
			addr = addr[:idxPercent]
		}

		process := "unknown"
		pid := "-"
		// Try to extract process details from last column
		processCol := fields[len(fields)-1]
		if match := processRegex.FindStringSubmatch(processCol); len(match) == 3 {
			process = match[1]
			pid = match[2]
		}

		// Deduplicate: if same port, protocol and address are listed multiple times, group them
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
	// Parse status numbered
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

	// Parse status verbose for defaults and logging
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

	// Retrieve listening ports
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

type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	IsDir   bool      `json:"is_dir"`
	ModTime time.Time `json:"mod_time"`
	Mode    string    `json:"mode"`
}

type DBConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func getDBConfigPath() string {
	return "db_config.json"
}

func loadDBConfig() (DBConfig, error) {
	var config DBConfig
	path := getDBConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return config, nil // Return empty if not configured
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(data, &config)
	return config, err
}

func saveDBConfig(config DBConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getDBConfigPath(), data, 0600)
}

func safeIdentifier(s string) bool {
	pattern := regexp.MustCompile(`^[a-zA-Z0-9_$.-]+$`)
	return pattern.MatchString(s)
}

func getDBConnection(dbName string) (*sql.DB, error) {
	config, err := loadDBConfig()
	if err != nil {
		return nil, err
	}
	if config.Host == "" {
		return nil, fmt.Errorf("Database connection is not configured")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.Username, config.Password, config.Host, config.Port, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func executeCustomSQL(dbName string, query string) (gin.H, error) {
	db, err := getDBConnection(dbName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	trimmed := strings.TrimSpace(strings.ToLower(query))
	isSelect := strings.HasPrefix(trimmed, "select") ||
		strings.HasPrefix(trimmed, "show") ||
		strings.HasPrefix(trimmed, "desc") ||
		strings.HasPrefix(trimmed, "explain") ||
		strings.HasPrefix(trimmed, "help")

	if isSelect {
		sqlRows, err := db.Query(query)
		if err != nil {
			return nil, err
		}
		defer sqlRows.Close()

		columns, err := sqlRows.Columns()
		if err != nil {
			return nil, err
		}

		var rows [][]interface{}
		for sqlRows.Next() {
			rowValues := make([]interface{}, len(columns))
			rowValPointers := make([]interface{}, len(columns))
			for i := range rowValues {
				rowValPointers[i] = &rowValues[i]
			}

			if err := sqlRows.Scan(rowValPointers...); err != nil {
				return nil, err
			}

			formattedRow := make([]interface{}, len(columns))
			for i, val := range rowValues {
				if val == nil {
					formattedRow[i] = nil
				} else if b, ok := val.([]byte); ok {
					formattedRow[i] = string(b)
				} else {
					formattedRow[i] = val
				}
			}
			rows = append(rows, formattedRow)
		}

		if rows == nil {
			rows = [][]interface{}{}
		}

		return gin.H{
			"type":    "select",
			"columns": columns,
			"rows":    rows,
		}, nil
	} else {
		res, err := db.Exec(query)
		if err != nil {
			return nil, err
		}

		rowsAffected, _ := res.RowsAffected()
		lastInsertID, _ := res.LastInsertId()

		return gin.H{
			"type":           "exec",
			"rows_affected":  rowsAffected,
			"last_insert_id": lastInsertID,
		}, nil
	}
}

func runSQLCommand(sql string) (string, error) {
	config, err := loadDBConfig()
	if err != nil {
		return "", err
	}
	if config.Host == "" {
		return "", fmt.Errorf("Database connection is not configured")
	}

	args := []string{"-h", config.Host, "-P", config.Port, "-u", config.Username, "-e", sql}
	cmd := exec.Command("mysql", args...)
	cmd.Env = os.Environ()
	if config.Password != "" {
		cmd.Env = append(cmd.Env, "MYSQL_PWD="+config.Password)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func runBackup(dbName string) (string, error) {
	config, err := loadDBConfig()
	if err != nil {
		return "", err
	}
	if config.Host == "" {
		return "", fmt.Errorf("Database connection is not configured")
	}

	backupDir := "/var/www/backups"
	err = os.MkdirAll(backupDir, 0755)
	if err != nil {
		return "", err
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("%s_%s.sql", dbName, time.Now().Format("20060102_150405")))

	args := []string{"-h", config.Host, "-P", config.Port, "-u", config.Username, dbName}
	cmd := exec.Command("mysqldump", args...)
	cmd.Env = os.Environ()
	if config.Password != "" {
		cmd.Env = append(cmd.Env, "MYSQL_PWD="+config.Password)
	}

	output, err := cmd.Output()
	if err != nil {
		errCmd := exec.Command("mysqldump", args...)
		errCmd.Env = cmd.Env
		errOut, _ := errCmd.CombinedOutput()
		return "", fmt.Errorf("mysqldump failed: %s", strings.TrimSpace(string(errOut)))
	}

	err = os.WriteFile(backupFile, output, 0644)
	if err != nil {
		return "", err
	}

	return backupFile, nil
}

type StoreApp struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	DefaultPort string `json:"default_port"`
	Image       string `json:"image"`
	Status      string `json:"status"`
	Domain      string `json:"domain,omitempty"`
}

type AppMetadata struct {
	ID     string `json:"id"`
	AppID  string `json:"app_id"`
	Domain string `json:"domain"`
	Port   string `json:"port"`
	DBName string `json:"db_name"`
	DBUser string `json:"db_user"`
	DBPass string `json:"db_pass"`
}

func getAppsMetadataPath() string {
	return "apps_metadata.json"
}

func loadAppsMetadata() ([]AppMetadata, error) {
	var list []AppMetadata
	path := getAppsMetadataPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return list, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return list, err
	}
	err = json.Unmarshal(data, &list)
	return list, err
}

func saveAppsMetadata(list []AppMetadata) error {
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getAppsMetadataPath(), data, 0600)
}

func generateRandomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	bytes := make([]byte, n)
	seed := time.Now().UnixNano()
	for i := 0; i < n; i++ {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		bytes[i] = letters[seed%int64(len(letters))]
	}
	return string(bytes)
}

func createNginxProxy(domain string, port string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	configContent := fmt.Sprintf(`server {
    listen 80;
    server_name %s;

    location / {
        proxy_pass http://127.0.0.1:%s;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
`, domain, port)

	availablePath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", domain)
	enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", domain)

	err := os.WriteFile(availablePath, []byte(configContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write nginx sites-available: %w", err)
	}

	_ = os.Remove(enabledPath)
	err = os.Symlink(availablePath, enabledPath)
	if err != nil {
		return fmt.Errorf("failed to symlink nginx config: %w", err)
	}

	return reloadNginx()
}

func deleteNginxProxy(domain string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	availablePath := fmt.Sprintf("/etc/nginx/sites-available/%s.conf", domain)
	enabledPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s.conf", domain)

	_ = os.Remove(enabledPath)
	_ = os.Remove(availablePath)

	return reloadNginx()
}

func runCertbot(domain string) {
	if runtime.GOOS == "windows" {
		return
	}
	// Run certbot in background or separate routine so we don't block
	go func() {
		cmd := exec.Command("certbot", "--nginx", "-d", domain, "--non-interactive", "--agree-tos", "--register-unsafely-without-email")
		_ = cmd.Run()
	}()
}

func fixHostDatabaseForDocker() {
	if runtime.GOOS == "windows" {
		return
	}
	// 1. Allow UFW for docker0 bridge network
	_ = exec.Command("ufw", "allow", "in", "on", "docker0", "to", "any", "port", "3306").Run()
	_ = exec.Command("ufw", "allow", "from", "172.17.0.0/16", "to", "any", "port", "3306").Run()

	// 2. Fix MariaDB/MySQL bind-address to allow external connections (0.0.0.0)
	script := `
	CONFIG_FILES="/etc/mysql/mariadb.conf.d/50-server.cnf /etc/mysql/mysql.conf.d/mysqld.cnf /etc/mysql/my.cnf"
	CHANGED=0
	for file in $CONFIG_FILES; do
		if [ -f "$file" ]; then
			if grep -q "bind-address[[:space:]]*=[[:space:]]*127.0.0.1" "$file"; then
				sed -i 's/bind-address[[:space:]]*=[[:space:]]*127.0.0.1/bind-address            = 0.0.0.0/g' "$file"
				CHANGED=1
			fi
		fi
	done
	if [ $CHANGED -eq 1 ]; then
		systemctl restart mariadb || systemctl restart mysql
	fi
	`
	_ = exec.Command("bash", "-c", script).Run()
}

type FtpUser struct {
	Username string `json:"username"`
	Path     string `json:"path"`
	Status   string `json:"status"` // "active" or "disabled"
}

func listFtpUsers() ([]FtpUser, error) {
	data, err := os.ReadFile("/etc/pure-ftpd/pureftpd.passwd")
	if err != nil {
		if os.IsNotExist(err) {
			return []FtpUser{}, nil
		}
		return nil, err
	}

	users := []FtpUser{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		status := "active"
		if strings.HasPrefix(line, "#") {
			status = "disabled"
			line = strings.TrimPrefix(line, "#")
		}
		parts := strings.Split(line, ":")
		if len(parts) >= 6 {
			users = append(users, FtpUser{
				Username: parts[0],
				Path:     parts[5],
				Status:   status,
			})
		}
	}
	return users, nil
}

type CronJob struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
	Status   string `json:"status"` // "enabled" or "disabled"
	LogPath  string `json:"log_path"`
	IsSystem bool   `json:"is_system"`
}

func listCronJobs() ([]CronJob, error) {
	cmd := exec.Command("crontab", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "no crontab") {
		return nil, fmt.Errorf("failed to run crontab -l: %s %w", string(output), err)
	}

	jobs := []CronJob{}
	lines := strings.Split(string(output), "\n")
	var currentJob *CronJob

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if strings.HasPrefix(trimmed, "# PANELD_ID:") {
			if currentJob != nil {
				jobs = append(jobs, *currentJob)
			}
			currentJob = &CronJob{
				ID:       strings.TrimSpace(strings.TrimPrefix(trimmed, "# PANELD_ID:")),
				Status:   "enabled",
				IsSystem: false,
			}
			continue
		}

		if currentJob != nil {
			if strings.HasPrefix(trimmed, "# PANELD_NAME:") {
				currentJob.Name = strings.TrimSpace(strings.TrimPrefix(trimmed, "# PANELD_NAME:"))
				continue
			}
			if strings.HasPrefix(trimmed, "# PANELD_STATUS:") {
				currentJob.Status = strings.TrimSpace(strings.TrimPrefix(trimmed, "# PANELD_STATUS:"))
				continue
			}
			if strings.HasPrefix(trimmed, "# PANELD_LOG:") {
				currentJob.LogPath = strings.TrimSpace(strings.TrimPrefix(trimmed, "# PANELD_LOG:"))
				continue
			}
		}

		if currentJob != nil && !strings.HasPrefix(trimmed, "# PANELD_") {
			cronLine := trimmed
			if currentJob.Status == "disabled" && strings.HasPrefix(cronLine, "#") {
				cronLine = strings.TrimSpace(strings.TrimPrefix(cronLine, "#"))
			}

			parts := strings.Fields(cronLine)
			if len(parts) >= 6 {
				currentJob.Schedule = strings.Join(parts[0:5], " ")
				fullCommand := strings.Join(parts[5:], " ")
				redirIndex := strings.Index(fullCommand, " > ")
				if redirIndex != -1 {
					currentJob.Command = strings.TrimSpace(fullCommand[:redirIndex])
				} else {
					currentJob.Command = fullCommand
				}
			} else {
				currentJob.Command = cronLine
			}
			jobs = append(jobs, *currentJob)
			currentJob = nil
			continue
		}

		if currentJob == nil && !strings.HasPrefix(trimmed, "# PANELD_") {
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			parts := strings.Fields(trimmed)
			if len(parts) >= 6 {
				schedule := strings.Join(parts[0:5], " ")
				cmd := strings.Join(parts[5:], " ")
				jobs = append(jobs, CronJob{
					ID:       "system_" + fmt.Sprintf("%d", len(jobs)),
					Name:     "System Job",
					Schedule: schedule,
					Command:  cmd,
					Status:   "enabled",
					IsSystem: true,
				})
			}
		}
	}

	if currentJob != nil {
		jobs = append(jobs, *currentJob)
	}

	return jobs, nil
}

func saveCronJobs(jobs []CronJob) error {
	var lines []string
	for _, job := range jobs {
		if job.IsSystem {
			lines = append(lines, fmt.Sprintf("%s %s", job.Schedule, job.Command))
		} else {
			lines = append(lines, fmt.Sprintf("# PANELD_ID: %s", job.ID))
			lines = append(lines, fmt.Sprintf("# PANELD_NAME: %s", job.Name))
			lines = append(lines, fmt.Sprintf("# PANELD_STATUS: %s", job.Status))
			if job.LogPath != "" {
				lines = append(lines, fmt.Sprintf("# PANELD_LOG: %s", job.LogPath))
			}
			
			cmdLine := job.Command
			if job.LogPath != "" {
				cmdLine = fmt.Sprintf("%s > %s 2>&1", job.Command, job.LogPath)
			}
			
			if job.Status == "disabled" {
				lines = append(lines, fmt.Sprintf("# %s %s", job.Schedule, cmdLine))
			} else {
				lines = append(lines, fmt.Sprintf("%s %s", job.Schedule, cmdLine))
			}
		}
		lines = append(lines, "")
	}

	tmpFile := "/tmp/new_crontab"
	err := os.WriteFile(tmpFile, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile)

	output, err := exec.Command("crontab", tmpFile).CombinedOutput()
	if err != nil {
		return fmt.Errorf("crontab failed: %s %w", string(output), err)
	}

	return nil
}

func saveSettingsToEnv(username, password string) error {
	var lines []string
	if data, err := os.ReadFile(".env"); err == nil {
		oldLines := strings.Split(string(data), "\n")
		for _, line := range oldLines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				if key == "ADMIN_USER" || key == "ADMIN_PASS" || key == "AUTH_TOKEN" {
					continue
				}
				lines = append(lines, line)
			}
		}
	}

	lines = append(lines, fmt.Sprintf("ADMIN_USER=%s", username))
	lines = append(lines, fmt.Sprintf("ADMIN_PASS=%s", password))
	lines = append(lines, fmt.Sprintf("AUTH_TOKEN=%s", authToken))

	return os.WriteFile(".env", []byte(strings.Join(lines, "\n")), 0600)
}

