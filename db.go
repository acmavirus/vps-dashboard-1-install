package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func getDBPath() string {
	var dir string
	if runtime.GOOS == "windows" {
		dir = filepath.Join(".", "data")
	} else {
		dir = "/usr/local/bin/data"
	}
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "acmadash.db")
}

func initDB() {
	dbPath := getDBPath()
	log.Printf("📂 Opening SQLite database: %s\n", dbPath)
	
	var err error
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Fatal error opening database: %v", err)
	}

	// Set connection limits
	DB.SetMaxOpenConns(1) // SQLite works best with 1 writer

	createTables()
	migrateJSONToSQLite()
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,

		`CREATE TABLE IF NOT EXISTS domains (
			domain TEXT PRIMARY KEY,
			note TEXT,
			is_starred INTEGER DEFAULT 0
		);`,

		`CREATE TABLE IF NOT EXISTS apps_metadata (
			id TEXT PRIMARY KEY,
			app_id TEXT,
			domain TEXT,
			port TEXT,
			db_name TEXT,
			db_user TEXT,
			db_pass TEXT
		);`,

		`CREATE TABLE IF NOT EXISTS security_logs (
			ip TEXT,
			timestamp INTEGER,
			uri TEXT,
			domain TEXT,
			user_agent TEXT,
			action TEXT
		);`,

		`CREATE TABLE IF NOT EXISTS metrics_history (
			timestamp INTEGER PRIMARY KEY,
			cpu REAL,
			ram REAL,
			disk REAL,
			net_sent INTEGER,
			net_recv INTEGER
		);`,
	}

	for _, q := range queries {
		_, err := DB.Exec(q)
		if err != nil {
			log.Fatalf("Error creating tables: %v. Query: %s", err, q)
		}
	}
}

// Helper to check if JSON file exists and is not empty
func jsonFileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir() && info.Size() > 2
}

func migrateJSONToSQLite() {
	var dataDir string
	if runtime.GOOS == "windows" {
		dataDir = filepath.Join(".", "data")
	} else {
		dataDir = "/usr/local/bin/data"
	}

	// 1. Migrate Domain Notes
	notesPath := filepath.Join(dataDir, "domain-notes.json")
	if jsonFileExists(notesPath) {
		log.Println("Migrating domain notes to SQLite...")
		data, err := os.ReadFile(notesPath)
		if err == nil {
			var notes map[string]string
			if json.Unmarshal(data, &notes) == nil {
				for dom, note := range notes {
					_, _ = DB.Exec(`INSERT INTO domains (domain, note) VALUES (?, ?)
						ON CONFLICT(domain) DO UPDATE SET note = excluded.note`, dom, note)
				}
			}
		}
		_ = os.Rename(notesPath, notesPath+".bak")
	}

	// 2. Migrate Domain Stars
	starsPath := filepath.Join(dataDir, "domain-stars.json")
	if jsonFileExists(starsPath) {
		log.Println("Migrating domain stars to SQLite...")
		data, err := os.ReadFile(starsPath)
		if err == nil {
			var stars map[string]bool
			if json.Unmarshal(data, &stars) == nil {
				for dom, starred := range stars {
					starVal := 0
					if starred {
						starVal = 1
					}
					_, _ = DB.Exec(`INSERT INTO domains (domain, is_starred) VALUES (?, ?)
						ON CONFLICT(domain) DO UPDATE SET is_starred = excluded.is_starred`, dom, starVal)
				}
			}
		}
		_ = os.Rename(starsPath, starsPath+".bak")
	}

	// 3. Migrate Apps Metadata
	appsPath := filepath.Join(dataDir, "apps-metadata.json")
	if jsonFileExists(appsPath) {
		log.Println("Migrating apps metadata to SQLite...")
		data, err := os.ReadFile(appsPath)
		if err == nil {
			var apps []AppMetadata
			if json.Unmarshal(data, &apps) == nil {
				for _, app := range apps {
					_, _ = DB.Exec(`INSERT INTO apps_metadata (id, app_id, domain, port, db_name, db_user, db_pass)
						VALUES (?, ?, ?, ?, ?, ?, ?)
						ON CONFLICT(id) DO UPDATE SET app_id = excluded.app_id, domain = excluded.domain, port = excluded.port,
						db_name = excluded.db_name, db_user = excluded.db_user, db_pass = excluded.db_pass`,
						app.ID, app.AppID, app.Domain, app.Port, app.DBName, app.DBUser, app.DBPass)
				}
			}
		}
		_ = os.Rename(appsPath, appsPath+".bak")
	}

	// 4. Migrate Security Settings
	secSettingsPath := filepath.Join(dataDir, "security-settings.json")
	if jsonFileExists(secSettingsPath) {
		log.Println("Migrating security settings to SQLite...")
		data, err := os.ReadFile(secSettingsPath)
		if err == nil {
			_, _ = DB.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
				ON CONFLICT(key) DO UPDATE SET value = excluded.value`, "security_settings", string(data))
		}
		_ = os.Rename(secSettingsPath, secSettingsPath+".bak")
	}

	// 5. Migrate Security Logs
	secLogsPath := filepath.Join(dataDir, "security-logs.json")
	if jsonFileExists(secLogsPath) {
		log.Println("Migrating security logs to SQLite...")
		data, err := os.ReadFile(secLogsPath)
		if err == nil {
			var logs []SecurityLogEntry
			if json.Unmarshal(data, &logs) == nil {
				for _, entry := range logs {
					_, _ = DB.Exec(`INSERT INTO security_logs (ip, timestamp, uri, domain, user_agent, action)
						VALUES (?, ?, ?, ?, ?, ?)`,
						entry.IP, entry.Timestamp.Unix(), entry.URI, entry.Domain, entry.UserAgent, entry.Action)
				}
			}
		}
		_ = os.Rename(secLogsPath, secLogsPath+".bak")
	}
}

// --- Settings Helpers ---
func getSetting(key string, defaultVal string) string {
	var value string
	err := DB.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err != nil {
		return defaultVal
	}
	return value
}

func saveSetting(key string, value string) error {
	_, err := DB.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value", key, value)
	return err
}

// --- Domain Notes and Stars (SQLite) ---
func getDomainNoteSQL(domain string) string {
	var note sql.NullString
	err := DB.QueryRow("SELECT note FROM domains WHERE domain = ?", domain).Scan(&note)
	if err != nil || !note.Valid {
		return ""
	}
	return note.String
}

func getDomainStarredSQL(domain string) bool {
	var starred int
	err := DB.QueryRow("SELECT is_starred FROM domains WHERE domain = ?", domain).Scan(&starred)
	if err != nil {
		return false
	}
	return starred == 1
}

func updateDomainNoteSQL(domain string, note string) error {
	_, err := DB.Exec(`INSERT INTO domains (domain, note) VALUES (?, ?)
		ON CONFLICT(domain) DO UPDATE SET note = excluded.note`, domain, note)
	return err
}

func updateDomainStarSQL(domain string, starred bool) error {
	starVal := 0
	if starred {
		starVal = 1
	}
	_, err := DB.Exec(`INSERT INTO domains (domain, is_starred) VALUES (?, ?)
		ON CONFLICT(domain) DO UPDATE SET is_starred = excluded.is_starred`, domain, starVal)
	return err
}

func deleteDomainSQL(domain string) error {
	_, err := DB.Exec("DELETE FROM domains WHERE domain = ?", domain)
	return err
}

// --- Apps Metadata (SQLite) ---
func loadAppsMetadataSQL() ([]AppMetadata, error) {
	rows, err := DB.Query("SELECT id, app_id, domain, port, db_name, db_user, db_pass FROM apps_metadata")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []AppMetadata
	for rows.Next() {
		var app AppMetadata
		var id, appID, domain, port, dbName, dbUser, dbPass sql.NullString
		if err := rows.Scan(&id, &appID, &domain, &port, &dbName, &dbUser, &dbPass); err != nil {
			return nil, err
		}
		app.ID = id.String
		app.AppID = appID.String
		app.Domain = domain.String
		app.Port = port.String
		app.DBName = dbName.String
		app.DBUser = dbUser.String
		app.DBPass = dbPass.String
		list = append(list, app)
	}
	return list, nil
}

func saveAppMetadataSQL(app AppMetadata) error {
	_, err := DB.Exec(`INSERT INTO apps_metadata (id, app_id, domain, port, db_name, db_user, db_pass)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET app_id = excluded.app_id, domain = excluded.domain, port = excluded.port,
		db_name = excluded.db_name, db_user = excluded.db_user, db_pass = excluded.db_pass`,
		app.ID, app.AppID, app.Domain, app.Port, app.DBName, app.DBUser, app.DBPass)
	return err
}

func deleteAppMetadataSQL(id string) error {
	_, err := DB.Exec("DELETE FROM apps_metadata WHERE id = ?", id)
	return err
}

// --- Security Logs (SQLite) ---
func loadSecurityLogsSQL() ([]SecurityLogEntry, error) {
	rows, err := DB.Query("SELECT ip, timestamp, uri, domain, user_agent, action FROM security_logs ORDER BY timestamp DESC LIMIT 500")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []SecurityLogEntry
	for rows.Next() {
		var entry SecurityLogEntry
		var ip, uri, domain, userAgent, action sql.NullString
		var ts int64
		if err := rows.Scan(&ip, &ts, &uri, &domain, &userAgent, &action); err != nil {
			return nil, err
		}
		entry.IP = ip.String
		entry.Timestamp = time.Unix(ts, 0)
		entry.URI = uri.String
		entry.Domain = domain.String
		entry.UserAgent = userAgent.String
		entry.Action = action.String
		list = append(list, entry)
	}
	return list, nil
}

func logSecurityEventSQL(entry SecurityLogEntry) error {
	_, err := DB.Exec(`INSERT INTO security_logs (ip, timestamp, uri, domain, user_agent, action)
		VALUES (?, ?, ?, ?, ?, ?)`,
		entry.IP, entry.Timestamp.Unix(), entry.URI, entry.Domain, entry.UserAgent, entry.Action)
	return err
}

func clearSecurityLogsSQL() error {
	_, err := DB.Exec("DELETE FROM security_logs")
	return err
}

// --- Metrics History (SQLite) ---
func logSystemMetricsSQL(stats SystemStats) error {
	_, err := DB.Exec(`INSERT INTO metrics_history (timestamp, cpu, ram, disk, net_sent, net_recv)
		VALUES (?, ?, ?, ?, ?, ?)`,
		stats.Timestamp, stats.CPU, stats.RAM, stats.Disk, stats.NetSent, stats.NetRecv)
	return err
}

type MetricHistoryEntry struct {
	Timestamp int64   `json:"timestamp"`
	CPU       float64 `json:"cpu"`
	RAM       float64 `json:"ram"`
	Disk      float64 `json:"disk"`
	NetSent   uint64  `json:"net_sent"`
	NetRecv   uint64  `json:"net_recv"`
}

func getMetricsHistorySQL(limit int) ([]MetricHistoryEntry, error) {
	rows, err := DB.Query("SELECT timestamp, cpu, ram, disk, net_sent, net_recv FROM metrics_history ORDER BY timestamp DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []MetricHistoryEntry
	for rows.Next() {
		var entry MetricHistoryEntry
		if err := rows.Scan(&entry.Timestamp, &entry.CPU, &entry.RAM, &entry.Disk, &entry.NetSent, &entry.NetRecv); err != nil {
			return nil, err
		}
		list = append(list, entry)
	}
	
	// Reverse list to make it chronological
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	
	return list, nil
}

func cleanOldMetricsSQL() error {
	// Purge metrics older than 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Unix()
	_, err := DB.Exec("DELETE FROM metrics_history WHERE timestamp < ?", thirtyDaysAgo)
	return err
}
