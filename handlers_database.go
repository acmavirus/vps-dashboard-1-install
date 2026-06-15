package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func registerDatabaseRoutes(api *gin.RouterGroup) {
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

		res, err := executeCustomSQL(dbName, "SHOW TABLE STATUS;")
		if err != nil {
			res2, err2 := executeCustomSQL(dbName, "SHOW TABLES;")
			if err2 != nil {
				c.JSON(500, gin.H{"error": err2.Error()})
				return
			}
			
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
}

func getDBConfigPath() string {
	return "db_config.json"
}

func loadDBConfig() (DBConfig, error) {
	var config DBConfig
	val := getSetting("db_config", "")
	if val == "" {
		path := "db_config.json"
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err == nil {
				_ = json.Unmarshal(data, &config)
				_ = saveDBConfig(config)
			}
		}
		return config, nil
	}
	err := json.Unmarshal([]byte(val), &config)
	return config, err
}

func saveDBConfig(config DBConfig) error {
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return saveSetting("db_config", string(data))
}

func getDBConnection(dbName string) (*sql.DB, error) {
	config, err := loadDBConfig()
	if err != nil {
		return nil, err
	}
	if config.Host == "" {
		return nil, fmt.Errorf("Database connection is not configured")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", config.Username, config.Password, config.Host, config.Port, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return db, nil
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

func safeIdentifier(val string) bool {
	reg := regexp.MustCompile(`^[a-zA-Z0-9_.-]+$`)
	return reg.MatchString(val)
}

func provisionCMSDatabase(prefix string) (string, string, string, error) {
	dbConfig, err := loadDBConfig()
	if err != nil || dbConfig.Host == "" {
		return "", "", "", fmt.Errorf("Please configure database credentials in the Databases tab first.")
	}

	fixHostDatabaseForDocker()

	suffix := strings.ToLower(generateRandomString(6))
	dbName := fmt.Sprintf("%s_%s", prefix, suffix)
	dbUser := fmt.Sprintf("%s_u_%s", prefix, suffix)
	dbPass := generateRandomString(14)

	_, err = runSQLCommand(fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", dbName))
	if err != nil {
		return "", "", "", fmt.Errorf("Failed to create database: %w", err)
	}

	_, err = runSQLCommand(fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s';", dbUser, dbPass))
	if err != nil {
		_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbName))
		return "", "", "", fmt.Errorf("Failed to create database user: %w", err)
	}

	_, err = runSQLCommand(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';", dbName, dbUser))
	if err != nil {
		_, _ = runSQLCommand(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", dbUser))
		_, _ = runSQLCommand(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", dbName))
		return "", "", "", fmt.Errorf("Failed to grant privileges: %w", err)
	}

	_, _ = runSQLCommand("FLUSH PRIVILEGES;")
	return dbName, dbUser, dbPass, nil
}

func fixHostDatabaseForDocker() {
	if runtime.GOOS == "windows" {
		return
	}
	script := `
	# Auto-detect bind-address in my.cnf and change it to 0.0.0.0
	CNF_FILES=("/etc/mysql/mariadb.conf.d/50-server.cnf" "/etc/mysql/my.cnf" "/etc/my.cnf")
	CHANGED=0
	for FILE in "${CNF_FILES[@]}"; do
		if [ -f "$FILE" ]; then
			if grep -q "bind-address" "$FILE"; then
				# If bind-address is 127.0.0.1, replace with 0.0.0.0
				if grep -q "bind-address\s*=\s*127.0.0.1" "$FILE"; then
					sed -i 's/bind-address\s*=\s*127.0.0.1/bind-address = 0.0.0.0/g' "$FILE"
					CHANGED=1
				fi
			fi
		fi
	done
	if [ $CHANGED -eq 1 ]; then
		systemctl restart mariadb || systemctl restart mysql
	fi
	`
	_ = exec.Command("bash", "-c", script).Run()
}
