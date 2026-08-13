package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	Errors  []interface{} `json:"errors"`
}

type CFBlockResponse struct {
	Success bool `json:"success"`
	Errors  []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
}

func getCFAccounts() []CFAccount {
	var accounts []CFAccount

	// Nạp từ môi trường các cấu hình dạng mảng CF_EMAIL_X và CF_KEY_X
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

	// Fallback nếu chỉ cấu hình 1 tài khoản đơn lẻ
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

// Trích xuất Apex Domain hoặc cắt tỉa subdomain để dò tìm zone
func getPotentialDomains(domain string) []string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" || domain == "-" || domain == "default server" {
		return nil
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return []string{domain}
	}

	var list []string
	// Thêm chính domain gốc trước
	list = append(list, domain)

	// Thử cắt subdomain dần dần từ trái qua phải
	// Ví dụ: shop.thuc.me -> thuc.me
	// a.b.c.com -> b.c.com -> c.com
	for i := 1; i < len(parts)-1; i++ {
		list = append(list, strings.Join(parts[i:], "."))
	}

	return list
}

// Dò tìm Zone ID và tài khoản quản lý domain tương ứng
func findCFZone(domain string, accounts []CFAccount) (string, CFAccount, bool) {
	potentials := getPotentialDomains(domain)
	if len(potentials) == 0 {
		return "", CFAccount{}, false
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, d := range potentials {
		for _, acc := range accounts {
			reqURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones?name=%s", d)
			req, err := http.NewRequest("GET", reqURL, nil)
			if err != nil {
				continue
			}

			req.Header.Set("X-Auth-Email", acc.Email)
			req.Header.Set("X-Auth-Key", acc.Key)
			req.Header.Set("Content-Type", applicationJSON) // sài hằng số hoặc định nghĩa trực tiếp

			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			var cfResp CFZoneResponse
			if err := json.Unmarshal(body, &cfResp); err != nil {
				continue
			}

			if cfResp.Success && len(cfResp.Result) > 0 {
				log.Printf("[Cloudflare] Found Zone ID %s for domain %s (managed by %s)\n", cfResp.Result[0].ID, d, acc.Email)
				return cfResp.Result[0].ID, acc, true
			}
		}
	}

	return "", CFAccount{}, false
}

const applicationJSON = "application/json"

// Ban IP trên Cloudflare (Tạo IP Access Rule cấp độ Zone)
func banIPCloudflare(ip string, domain string) (bool, error) {
	accounts := getCFAccounts()
	if len(accounts) == 0 {
		return false, fmt.Errorf("no cloudflare accounts configured")
	}

	zoneID, acc, found := findCFZone(domain, accounts)
	if !found {
		return false, fmt.Errorf("zone not found on configured cloudflare accounts for domain %s", domain)
	}

	// Tạo API Request
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
		// Kiểm tra nếu lỗi do IP đã bị block từ trước (Mã lỗi Cloudflare: 10001 hoặc tương tự)
		if len(cfResp.Errors) > 0 {
			msg := cfResp.Errors[0].Message
			if strings.Contains(strings.ToLower(msg), "already exists") || cfResp.Errors[0].Code == 10001 {
				log.Printf("[Cloudflare] IP %s was already blocked on Cloudflare for zone %s\n", ip, zoneID)
				return true, nil
			}
			return false, fmt.Errorf("cloudflare API error: %s", msg)
		}
		return false, fmt.Errorf("cloudflare block request failed")
	}

	return true, nil
}

// CFAccessRule định nghĩa cấu trúc của một rule chặn IP trên Cloudflare
type CFAccessRule struct {
	ID            string `json:"id"`
	Configuration struct {
		Target string `json:"target"`
		Value  string `json:"value"`
	} `json:"configuration"`
}

// CFAccessRulesResponse định nghĩa phản hồi khi tìm kiếm IP Access Rules
type CFAccessRulesResponse struct {
	Success bool           `json:"success"`
	Result  []CFAccessRule `json:"result"`
}

// unbanIPCloudflare tìm và gỡ chặn địa chỉ IP trên mọi active zone của tất cả accounts cấu hình
func unbanIPCloudflare(ip string) (int, error) {
	accounts := getCFAccounts()
	if len(accounts) == 0 {
		return 0, fmt.Errorf("no cloudflare accounts configured")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	unbannedCount := 0

	for _, acc := range accounts {
		// 1. Lấy danh sách zones của account này
		reqURL := "https://api.cloudflare.com/client/v4/zones?per_page=50"
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			log.Printf("[Cloudflare Error] Failed to create request to list zones for %s: %v\n", acc.Email, err)
			continue
		}
		req.Header.Set("X-Auth-Email", acc.Email)
		req.Header.Set("X-Auth-Key", acc.Key)
		req.Header.Set("Content-Type", applicationJSON)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[Cloudflare Error] Failed to fetch zones for %s: %v\n", acc.Email, err)
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

		if !zonesResp.Success {
			continue
		}

		// 2. Với mỗi zone, kiểm tra xem IP có bị block không
		for _, zone := range zonesResp.Result {
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
				// 3. Delete từng rule tìm thấy
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
						log.Printf("[Cloudflare] Successfully unblocked IP %s on zone %s (%s)\n", ip, zone.Name, zone.ID)
						unbannedCount++
					} else {
						log.Printf("[Cloudflare Error] Failed to delete rule %s on zone %s: %v\n", rule.ID, zone.Name, err)
					}
				}
			}
		}
	}

	return unbannedCount, nil
}

// getCFZones kết nối Cloudflare và lấy danh sách tất cả các Zone (Domain) hoạt động
func getCFZones() ([]CFZone, error) {
	accounts := getCFAccounts()
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no cloudflare accounts configured")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var allZones []CFZone
	seenZones := make(map[string]bool)

	for _, acc := range accounts {
		reqURL := "https://api.cloudflare.com/client/v4/zones?per_page=50&status=active"
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			log.Printf("[Cloudflare Error] Failed to create zones request for %s: %v\n", acc.Email, err)
			continue
		}
		req.Header.Set("X-Auth-Email", acc.Email)
		req.Header.Set("X-Auth-Key", acc.Key)
		req.Header.Set("Content-Type", applicationJSON)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[Cloudflare Error] Failed to fetch zones for %s: %v\n", acc.Email, err)
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

