#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import json
import urllib.request
import urllib.error
import os
import sys

# Cloudflare API Base URL
CF_API_BASE = "https://api.cloudflare.com/client/v4"

def load_env():
    """Tự động đọc file .env thủ công để nạp biến môi trường"""
    env_paths = [".env", "../.env"]
    for path in env_paths:
        if os.path.exists(path):
            with open(path, "r", encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line or line.startswith("#"):
                        continue
                    parts = line.split("=", 1)
                    if len(parts) == 2:
                        key = parts[0].strip()
                        val = parts[1].strip().strip("\"'")
                        os.environ[key] = val
            break

def make_request(url, method="GET", headers=None, data=None):
    """Thực hiện HTTP Request bằng urllib để không cần cài thư viện ngoài"""
    if headers is None:
        headers = {}
    
    req_data = None
    if data is not None:
        req_data = json.dumps(data).encode("utf-8")
        headers["Content-Type"] = "application/json"
        
    req = urllib.request.Request(url, data=req_data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req) as response:
            res_body = response.read().decode("utf-8")
            return json.loads(res_body)
    except urllib.error.HTTPError as e:
        err_body = e.read().decode("utf-8")
        try:
            return json.loads(err_body)
        except Exception:
            return {"success": False, "errors": [{"message": f"HTTP Error {e.code}: {e.reason}"}]}
    except Exception as e:
        return {"success": False, "errors": [{"message": str(e)}]}

def get_zone_id(domain, headers):
    """Lấy Zone ID từ tên miền"""
    url = f"{CF_API_BASE}/zones?name={domain}"
    res = make_request(url, "GET", headers)
    if res.get("success") and res.get("result"):
        return res["result"][0]["id"]
    errors = res.get("errors", [{"message": "Unknown error"}])
    raise Exception(f"Không thể lấy Zone ID cho {domain}: {errors[0]['message']}")

def get_all_zones(headers):
    """Lấy danh sách tất cả các Zone (Domain) đang hoạt động trên tài khoản"""
    url = f"{CF_API_BASE}/zones?per_page=50&status=active"
    res = make_request(url, "GET", headers)
    if res.get("success"):
        return res.get("result", [])
    errors = res.get("errors", [{"message": "Unknown error"}])
    raise Exception(f"Không thể lấy danh sách zones: {errors[0]['message']}")

def get_or_create_ruleset(zone_id, phase, headers):
    """Tìm hoặc tạo mới ruleset cho một phase cụ thể (custom rules hoặc ratelimit)"""
    # 1. Lấy danh sách rulesets hiện có của Zone
    url = f"{CF_API_BASE}/zones/{zone_id}/rulesets"
    res = make_request(url, "GET", headers)
    if not res.get("success"):
        errors = res.get("errors", [{"message": "Unknown error"}])
        raise Exception(f"Không thể đọc danh sách rulesets: {errors[0]['message']}")
    
    # Tìm xem ruleset cho phase này đã tồn tại chưa
    for ruleset in res.get("result", []):
        if ruleset.get("phase") == phase:
            return ruleset["id"]
            
    # 2. Nếu chưa tồn tại ruleset cho phase đó, tạo mới
    create_url = f"{CF_API_BASE}/zones/{zone_id}/rulesets"
    payload = {
        "name": f"Ruleset for {phase}",
        "description": f"Auto-created ruleset for phase {phase}",
        "kind": "zone",
        "phase": phase
    }
    create_res = make_request(create_url, "POST", headers, payload)
    if create_res.get("success") and create_res.get("result"):
        return create_res["result"]["id"]
        
    create_errors = create_res.get("errors", [{"message": "Unknown error"}])
    raise Exception(f"Không thể tạo ruleset cho phase {phase}: {create_errors[0]['message']}")

def check_rule_exists(zone_id, ruleset_id, description, headers):
    """Kiểm tra xem rule có mô tả (description) đó đã được tạo trong ruleset chưa"""
    url = f"{CF_API_BASE}/zones/{zone_id}/rulesets/{ruleset_id}"
    res = make_request(url, "GET", headers)
    if res.get("success") and res.get("result"):
        rules = res["result"].get("rules", [])
        for rule in rules:
            if rule.get("description") == description:
                return True, rule["id"]
    return False, None

def setup_waf_custom_rule(zone_id, headers):
    """Cấu hình WAF Custom Rule: Block /api/v1 không có x-api-key"""
    phase = "http_request_firewall_custom"
    ruleset_id = get_or_create_ruleset(zone_id, phase, headers)
    
    description = "WAF Rule 1: Block /api/v1 without x-api-key"
    exists, rule_id = check_rule_exists(zone_id, ruleset_id, description, headers)
    
    # Biểu thức WAF kiểm tra đường dẫn bắt đầu bằng /api/v1 và không chứa tiêu đề x-api-key
    expression = '(starts_with(http.request.uri.path, "/api/v1") and not any(http.request.headers.names[*] == "x-api-key"))'
    
    payload = {
        "action": "block",
        "expression": expression,
        "description": description,
        "enabled": True
    }
    
    if exists:
        print(f"[-] WAF rule '{description}' đã tồn tại (ID: {rule_id}). Đang cập nhật...")
        update_url = f"{CF_API_BASE}/zones/{zone_id}/rulesets/{ruleset_id}/rules/{rule_id}"
        res = make_request(update_url, "PATCH", headers, payload)
    else:
        print(f"[+] Đang tạo mới WAF rule: {description}")
        create_url = f"{CF_API_BASE}/zones/{zone_id}/rulesets/{ruleset_id}/rules"
        res = make_request(create_url, "POST", headers, payload)
        
    if res.get("success"):
        print("[✓] Cấu hình WAF Rule thành công!")
    else:
        errors = res.get("errors", [{"message": "Unknown error"}])
        print(f"[X] Lỗi cấu hình WAF Rule: {errors[0]['message']}")

def setup_rate_limit_rule(zone_id, headers):
    """Cấu hình Rate Limiting Rule: 200r/10s cho /api/v1"""
    phase = "http_ratelimit"
    ruleset_id = get_or_create_ruleset(zone_id, phase, headers)
    
    description = "Rate Limit: 200 req/10s per IP on /api/v1"
    exists, rule_id = check_rule_exists(zone_id, ruleset_id, description, headers)
    
    expression = 'starts_with(http.request.uri.path, "/api/v1")'
    
    payload = {
        "action": "block",
        "expression": expression,
        "description": description,
        "enabled": True,
        "ratelimit": {
            "characteristics": ["cf.colo.id", "ip.src"],
            "period": 10,
            "requests_per_period": 200,
            "mitigation_timeout": 10
        }
    }
    
    if exists:
        print(f"[-] Rate limit rule '{description}' đã tồn tại (ID: {rule_id}). Đang cập nhật...")
        update_url = f"{CF_API_BASE}/zones/{zone_id}/rulesets/{ruleset_id}/rules/{rule_id}"
        res = make_request(update_url, "PATCH", headers, payload)
    else:
        print(f"[+] Đang tạo mới Rate Limit rule: {description}")
        create_url = f"{CF_API_BASE}/zones/{zone_id}/rulesets/{ruleset_id}/rules"
        res = make_request(create_url, "POST", headers, payload)
        
    if res.get("success"):
        print("[✓] Cấu hình Rate Limit Rule thành công!")
    else:
        errors = res.get("errors", [{"message": "Unknown error"}])
        print(f"[X] Lỗi cấu hình Rate Limit Rule: {errors[0]['message']}")

def enable_bot_fight_mode(zone_id, headers):
    """Bật Bot Fight Mode (Block headless browsers)"""
    print("[+] Đang kích hoạt Bot Fight Mode...")
    url = f"{CF_API_BASE}/zones/{zone_id}/settings/bot_fight_mode"
    payload = {
        "value": "on"
    }
    res = make_request(url, "PATCH", headers, payload)
    if res.get("success"):
        print("[✓] Kích hoạt Bot Fight Mode thành công!")
    else:
        errors = res.get("errors", [{"message": "Unknown error"}])
        # Một số gói trả phí hoặc API của zone cũ có thể không hỗ trợ PATCH trực tiếp qua endpoint này
        print(f"[i] Thông báo Bot Fight Mode: {errors[0]['message']} (Nếu zone của bạn dùng gói Free, Bot Fight Mode có thể cần được kích hoạt thủ công trên giao diện Web Cloudflare -> Security -> Bots).")

def main():
    load_env()
    
    # 1. Nhận thông tin xác thực từ biến môi trường hoặc nhập tay
    cf_email = os.environ.get("CF_EMAIL")
    cf_key = os.environ.get("CF_KEY")
    cf_domain = os.environ.get("CF_DOMAIN")
    
    print("==================================================")
    print("      TỰ ĐỘNG CẤU HÌNH BẢO MẬT CLOUDFLARE EDGE    ")
    print("==================================================")
    
    if not cf_email:
        cf_email = input("Nhập Email tài khoản Cloudflare: ").strip()
    if not cf_key:
        cf_key = input("Nhập Cloudflare Global API Key: ").strip()
    if not cf_domain:
        cf_domain = input("Nhập tên miền cấu hình (ví dụ: thuc.me) [Bỏ trống để cấu hình TẤT CẢ các domain]: ").strip()
        
    if not cf_email or not cf_key:
        print("[X] Lỗi: Các thông tin Email và Key là bắt buộc!")
        sys.exit(1)
        
    headers = {
        "X-Auth-Email": cf_email,
        "X-Auth-Key": cf_key
    }
    
    try:
        zones = []
        if cf_domain:
            # Lấy thông tin cho 1 domain cụ thể
            print(f"[*] Đang dò tìm Zone ID cho tên miền: {cf_domain}...")
            zone_id = get_zone_id(cf_domain, headers)
            print(f"[✓] Tìm thấy Zone ID: {zone_id}")
            zones.append({"id": zone_id, "name": cf_domain})
        else:
            # Tự động quét tất cả các domain hoạt động trên tài khoản
            print("[*] Đang quét toàn bộ các tên miền hoạt động trên tài khoản Cloudflare...")
            zones_list = get_all_zones(headers)
            if not zones_list:
                print("[X] Không tìm thấy tên miền nào hoạt động trên tài khoản Cloudflare của bạn.")
                sys.exit(0)
            for z in zones_list:
                zones.append({"id": z["id"], "name": z["name"]})
                
        print(f"[✓] Tìm thấy {len(zones)} tên miền mục tiêu.")
        print("==================================================")
        
        # 3. Lần lượt cấu hình cho các domain mục tiêu
        for z in zones:
            print(f"\n🚀 Bắt đầu cấu hình cho tên miền: {z['name']} (ID: {z['id']})...")
            print("--------------------------------------------------")
            try:
                # Cấu hình WAF Custom Rule
                setup_waf_custom_rule(z['id'], headers)
                print("--------------------------------------------------")
                
                # Cấu hình Rate Limiting Rule
                setup_rate_limit_rule(z['id'], headers)
                print("--------------------------------------------------")
                
                # Bật Bot Fight Mode
                enable_bot_fight_mode(z['id'], headers)
                print("--------------------------------------------------")
                print(f"[✓] Đã cấu hình xong bảo mật Cloudflare cho tên miền: {z['name']}")
            except Exception as e:
                print(f"[X] Lỗi cấu hình cho {z['name']}: {str(e)}")
                
        print("\n[✓] Hoàn tất quá trình cấu hình bảo mật Cloudflare Edge tự động!")
        
    except Exception as e:
        print(f"\n[X] Lỗi nghiêm trọng: {str(e)}")
        sys.exit(1)

if __name__ == "__main__":
    main()
