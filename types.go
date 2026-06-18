package main

import "time"

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
	Domain      string    `json:"domain"`
	Status      string    `json:"status"` // online, offline
	Code        int       `json:"code"`
	Note        string    `json:"note,omitempty"`
	IsStarred   bool      `json:"is_starred"`
	SSLActive   bool      `json:"ssl_active"`
	SSLIssuer   string    `json:"ssl_issuer"`
	SSLExpiry   time.Time `json:"ssl_expiry"`
	SSLDays     int       `json:"ssl_days"`
}

type SSLCertInfo struct {
	Domain     string    `json:"domain"`
	Issuer     string    `json:"issuer"`
	ExpiryDate time.Time `json:"expiry_date"`
	DaysLeft   int       `json:"days_left"`
	IsExpired  bool      `json:"is_expired"`
}

type SoftwareInfo struct {
	Nginx string `json:"nginx"`
	PHP83 string `json:"php83"`
	PHP74 string `json:"php74"`
	MySQL string `json:"mysql"`
	Redis string `json:"redis"`
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

type SecuritySettings struct {
	AutoBanEnabled bool     `json:"auto_ban_enabled"`
	BanThreshold   int      `json:"ban_threshold"`
	ProbePatterns  []string `json:"probe_patterns"`
	TelegramAlerts bool     `json:"telegram_alerts"`
}

type SecurityLogEntry struct {
	IP        string    `json:"ip"`
	Timestamp time.Time `json:"timestamp"`
	URI       string    `json:"uri"`
	Domain    string    `json:"domain"`
	UserAgent string    `json:"user_agent"`
	Action    string    `json:"action"`
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

type FtpUser struct {
	Username string `json:"username"`
	Path     string `json:"path"`
	Status   string `json:"status"` // "active" or "disabled"
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
