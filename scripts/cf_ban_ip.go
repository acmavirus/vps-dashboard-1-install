package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type CFAccount struct {
	Email string
	Key   string
}

type CFZone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CFZoneResponse struct {
	Success bool     `json:"success"`
	Result  []CFZone `json:"result"`
}

type CFBlockResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

type CFAccessRule struct {
	ID            string `json:"id"`
	Configuration struct {
		Target string `json:"target"`
		Value  string `json:"value"`
	} `json:"configuration"`
}

type CFAccessRulesResponse struct {
	Success bool           `json:"success"`
	Result  []CFAccessRule `json:"result"`
}

const applicationJSON = "application/json"

// loadEnv đọc file .env thủ công để nạp biến môi trường
func loadEnv() {
	file, err := os.Open(".env")
	if err != nil {
		// Thử tìm ở thư mục cha nếu chạy từ thư mục scripts
		file, err = os.Open("../.env")
		if err != nil {
			return
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			// Cắt bỏ dấu ngoặc kép nếu có
			val = strings.Trim(val, `"'`)
			os.Setenv(key, val)
		}
	}
}

func getCFAccounts() []CFAccount {
	var accounts []CFAccount

	for i := 1; i <= 5; i++ {
		email := os.Getenv(fmt.Sprintf("CF_EMAIL_%d", i))
		key := os.Getenv(fmt.Sprintf("CF_KEY_%d", i))
		if email != "" && key != "" {
			accounts = append(accounts, CFAccount{
				Email: strings.TrimSpace(email),
				Key:   strings.TrimSpace(key),
			})
		}
	}

	email := os.Getenv("CF_EMAIL")
	key := os.Getenv("CF_KEY")
	if email != "" && key != "" {
		accounts = append(accounts, CFAccount{
			Email: strings.TrimSpace(email),
			Key:   strings.TrimSpace(key),
		})
	}

	return accounts
}

func getCFZones(accounts []CFAccount) ([]CFZone, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	var allZones []CFZone
	seenZones := make(map[string]bool)

	for _, acc := range accounts {
		reqURL := "https://api.cloudflare.com/client/v4/zones?per_page=50&status=active"
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("X-Auth-Email", acc.Email)
		req.Header.Set("X-Auth-Key", acc.Key)
		req.Header.Set("Content-Type", applicationJSON)

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var zonesResp CFZoneResponse
		if err := json.Unmarshal(body, &zonesResp); err != nil {
			continue
		}

		if zonesResp.Success {
			for _, z := range zonesResp.Result {
				if !seenZones[z.ID] {
					seenZones[z.ID] = true
					allZones = append(allZones, z)
				}
			}
		}
	}

	return allZones, nil
}

func banIPCloudflare(ip string, zoneID string, domain string, acc CFAccount) (bool, error) {
	reqURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/firewall/access_rules/rules", zoneID)
	
	payload := map[string]interface{}{
		"mode": "block",
		"configuration": map[string]string{
			"target": "ip",
			"value":  ip,
		},
		"notes": fmt.Sprintf("Auto-ban by AcmaDash IPS on domain %s", domain),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return false, err
	}

	req.Header.Set("X-Auth-Email", acc.Email)
	req.Header.Set("X-Auth-Key", acc.Key)
	req.Header.Set("Content-Type", applicationJSON)

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var cfResp CFBlockResponse
	if err := json.Unmarshal(body, &cfResp); err != nil {
		return false, fmt.Errorf("failed to parse response: %s", string(body))
	}

	if !cfResp.Success {
		if len(cfResp.Errors) > 0 {
			msg := cfResp.Errors[0].Message
			if strings.Contains(strings.ToLower(msg), "already exists") || cfResp.Errors[0].Code == 10001 {
				fmt.Printf("[Cloudflare] IP %s was already blocked on zone %s\n", ip, domain)
				return true, nil
			}
			return false, fmt.Errorf("cloudflare API error: %s", msg)
		}
		return false, fmt.Errorf("cloudflare block request failed")
	}

	fmt.Printf("[Cloudflare] Successfully blocked IP %s on zone %s\n", ip, domain)
	return true, nil
}

func unbanIPCloudflare(ip string, accounts []CFAccount) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	unbannedCount := 0

	for _, acc := range accounts {
		zones, err := getCFZones([]CFAccount{acc})
		if err != nil {
			continue
		}

		for _, zone := range zones {
			rulesURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/firewall/access_rules/rules?configuration.target=ip&configuration.value=%s", zone.ID, ip)
			rulesReq, err := http.NewRequest("GET", rulesURL, nil)
			if err != nil {
				continue
			}
			rulesReq.Header.Set("X-Auth-Email", acc.Email)
			rulesReq.Header.Set("X-Auth-Key", acc.Key)
			rulesReq.Header.Set("Content-Type", applicationJSON)

			rulesRespObj, err := client.Do(rulesReq)
			if err != nil {
				continue
			}
			rulesBody, err := io.ReadAll(rulesRespObj.Body)
			rulesRespObj.Body.Close()
			if err != nil {
				continue
			}

			var cfRulesResp CFAccessRulesResponse
			if err := json.Unmarshal(rulesBody, &cfRulesResp); err != nil {
				continue
			}

			if cfRulesResp.Success && len(cfRulesResp.Result) > 0 {
				for _, rule := range cfRulesResp.Result {
					delURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/firewall/access_rules/rules/%s", zone.ID, rule.ID)
					delReq, err := http.NewRequest("DELETE", delURL, nil)
					if err != nil {
						continue
					}
					delReq.Header.Set("X-Auth-Email", acc.Email)
					delReq.Header.Set("X-Auth-Key", acc.Key)
					delReq.Header.Set("Content-Type", applicationJSON)

					delResp, err := client.Do(delReq)
					if err == nil {
						delResp.Body.Close()
						fmt.Printf("[Cloudflare] Successfully unblocked IP %s on zone %s\n", ip, zone.Name)
						unbannedCount++
					}
				}
			}
		}
	}

	return unbannedCount, nil
}

func printUsage() {
	fmt.Println("AcmaDash Cloudflare IP Ban/Unban CLI Tool")
	fmt.Println("Usage:")
	fmt.Println("  Chặn IP trên tất cả các Zone:")
	fmt.Println("    go run scripts/cf_ban_ip.go <IP>")
	fmt.Println("  Chặn IP trên một Zone cụ thể:")
	fmt.Println("    go run scripts/cf_ban_ip.go <IP> <domain>")
	fmt.Println("  Gỡ chặn IP trên tất cả các Zone:")
	fmt.Println("    go run scripts/cf_ban_ip.go --unban <IP>   (hoặc -u <IP>)")
}

func main() {
	loadEnv()

	accounts := getCFAccounts()
	if len(accounts) == 0 {
		log.Fatal("Error: No Cloudflare accounts configured in .env")
	}

	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		return
	}

	// Xử lý UNBAN
	if args[0] == "--unban" || args[0] == "-u" {
		if len(args) < 2 {
			log.Fatal("Error: Please provide IP address to unban.")
		}
		ip := strings.TrimSpace(args[1])
		if net.ParseIP(ip) == nil {
			log.Fatalf("Error: Invalid IP address '%s'", ip)
		}

		fmt.Printf("Searching and unbanning IP %s on all Cloudflare zones...\n", ip)
		count, err := unbanIPCloudflare(ip, accounts)
		if err != nil {
			log.Fatalf("Error unbanning IP: %v", err)
		}
		fmt.Printf("Done! Removed %d rules from Cloudflare.\n", count)
		return
	}

	// Xử lý BAN
	ip := strings.TrimSpace(args[0])
	if net.ParseIP(ip) == nil {
		if strings.HasPrefix(ip, "-") {
			printUsage()
			return
		}
		log.Fatalf("Error: Invalid IP address '%s'", ip)
	}

	var targetDomain string
	if len(args) >= 2 {
		targetDomain = strings.ToLower(strings.TrimSpace(args[1]))
	}

	zones, err := getCFZones(accounts)
	if err != nil {
		log.Fatalf("Error fetching zones: %v", err)
	}
	if len(zones) == 0 {
		log.Fatal("Error: No active zones found on your Cloudflare accounts.")
	}

	if targetDomain != "" {
		// Chặn trên 1 domain cụ thể
		var targetZone CFZone
		var targetAcc CFAccount
		found := false

		for _, zone := range zones {
			if zone.Name == targetDomain || strings.HasSuffix(targetDomain, "."+zone.Name) {
				// Tìm tài khoản sở hữu zone này
				for _, acc := range accounts {
					checkURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", zone.Name)
					req, _ := http.NewRequest("GET", checkURL, nil)
					req.Header.Set("X-Auth-Email", acc.Email)
					req.Header.Set("X-Auth-Key", acc.Key)
					resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
					if err == nil {
						body, _ := io.ReadAll(resp.Body)
						resp.Body.Close()
						var zResp CFZoneResponse
						if json.Unmarshal(body, &zResp) == nil && zResp.Success && len(zResp.Result) > 0 {
							targetZone = zone
							targetAcc = acc
							found = true
							break
						}
					}
				}
				if found {
					break
				}
			}
		}

		if !found {
			log.Fatalf("Error: Domain '%s' not found on your Cloudflare accounts.", targetDomain)
		}

		fmt.Printf("Banning IP %s on domain %s...\n", ip, targetDomain)
		_, err = banIPCloudflare(ip, targetZone.ID, targetZone.Name, targetAcc)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}
	} else {
		// Chặn trên tất cả các domain
		fmt.Printf("Banning IP %s on ALL active Cloudflare zones (%d zones)...\n", ip, len(zones))
		successCount := 0
		for _, zone := range zones {
			for _, acc := range accounts {
				checkURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", zone.Name)
				req, _ := http.NewRequest("GET", checkURL, nil)
				req.Header.Set("X-Auth-Email", acc.Email)
				req.Header.Set("X-Auth-Key", acc.Key)
				resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
				if err == nil {
					body, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					var zResp CFZoneResponse
					if json.Unmarshal(body, &zResp) == nil && zResp.Success && len(zResp.Result) > 0 {
						ok, err := banIPCloudflare(ip, zone.ID, zone.Name, acc)
						if err == nil && ok {
							successCount++
						}
						break
					}
				}
			}
		}
		fmt.Printf("Done! Successfully banned IP on %d/%d zones.\n", successCount, len(zones))
	}
}
