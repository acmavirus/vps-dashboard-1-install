#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import json
import urllib.request
import urllib.error
import os
import sys

# Cloudflare API Base URL
CF_API_BASE = "https://api.cloudflare.com/client/v4"
VPS_IP = "15.235.199.163"  # IP VPS cua ban

def load_env():
    """Tu dong doc file .env thu cong de nap bien moi truong"""
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
    """Thuc hien HTTP Request bang urllib de khong can cai thu vien ngoai"""
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

def get_all_zones(headers):
    """Lay danh sach tat ca cac Zone (Domain) dang hoat dong tren tai khoan"""
    url = f"{CF_API_BASE}/zones?per_page=50&status=active"
    res = make_request(url, "GET", headers)
    if res.get("success"):
        return res.get("result", [])
    errors = res.get("errors", [{"message": "Unknown error"}])
    raise Exception(f"Khong the lay danh sach cac zones: {errors[0]['message']}")

def check_domain_points_to_vps(zone_id, headers):
    """Kiem tra xem domain co ban ghi A nao tro ve IP VPS cua ban hay khong"""
    url = f"{CF_API_BASE}/zones/{zone_id}/dns_records?type=A"
    res = make_request(url, "GET", headers)
    if res.get("success"):
        for record in res.get("result", []):
            if record.get("content") == VPS_IP:
                return True
    return False

def get_or_create_ruleset(zone_id, phase, headers):
    """Tim hoac tao moi ruleset cho mot phase cu the (custom rules)"""
    url = f"{CF_API_BASE}/zones/{zone_id}/rulesets"
    res = make_request(url, "GET", headers)
    if not res.get("success"):
        errors = res.get("errors", [{"message": "Unknown error"}])
        raise Exception(f"Khong the doc danh sach rulesets: {errors[0]['message']}")
    
    # Tim xem ruleset cho phase nay da ton tai chua
    for ruleset in res.get("result", []):
        if ruleset.get("phase") == phase:
            return ruleset["id"]
            
    # Neu chua ton tai ruleset cho phase do, tao moi
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
    raise Exception(f"Khong the tao ruleset cho phase {phase}: {create_errors[0]['message']}")

def check_rule_exists(zone_id, ruleset_id, description, headers):
    """Kiem tra xem rule co mo ta (description) do da duoc tao trong ruleset chua"""
    url = f"{CF_API_BASE}/zones/{zone_id}/rulesets/{ruleset_id}"
    res = make_request(url, "GET", headers)
    if res.get("success") and res.get("result"):
        rules = res["result"].get("rules", [])
        for rule in rules:
            if rule.get("description") == description:
                return True, rule["id"]
    return False, None

def setup_probe_block_rule(zone_id, headers):
    """Cau hinh WAF Custom Rule: Block cac IP quet file nhay cam"""
    phase = "http_request_firewall_custom"
    ruleset_id = get_or_create_ruleset(zone_id, phase, headers)
    
    description = "WAF Rule: Block Malicious Path Probing (.env, .git, config.php...)"
    exists, rule_id = check_rule_exists(zone_id, ruleset_id, description, headers)
    
    # Danh sach cac chuoi nhay cam can chan
    keywords = [
        "/.env", ".env.", "/.git", "/.svn", "/.htaccess", 
        "/wp-config.php", "/config.php", "/database.sql"
    ]
    
    expr_parts = [f'(http.request.uri.path contains "{kw}")' for kw in keywords]
    expression = " or ".join(expr_parts)
    
    payload = {
        "action": "block",
        "expression": expression,
        "description": description,
        "enabled": True
    }
    
    if exists:
        print(f"  [-] WAF rule '{description}' da ton tai (ID: {rule_id}). Dang cap nhat...")
        update_url = f"{CF_API_BASE}/zones/{zone_id}/rulesets/{ruleset_id}/rules/{rule_id}"
        res = make_request(update_url, "PATCH", headers, payload)
    else:
        print(f"  [+] Dang tao moi WAF rule: {description}")
        create_url = f"{CF_API_BASE}/zones/{zone_id}/rulesets/{ruleset_id}/rules"
        res = make_request(create_url, "POST", headers, payload)
        
    if res.get("success"):
        print("  [OK] Cau hinh WAF Path Block Rule thanh cong!")
    else:
        errors = res.get("errors", [{"message": "Unknown error"}])
        print(f"  [ERROR] Loi cau hinh WAF Path Block Rule: {errors[0]['message']}")

def delete_probe_block_rule_if_exists(zone_id, headers):
    """Xoa WAF Custom Rule khoi domain lien ket neu co"""
    phase = "http_request_firewall_custom"
    try:
        ruleset_id = get_or_create_ruleset(zone_id, phase, headers)
    except Exception:
        return
        
    description = "WAF Rule: Block Malicious Path Probing (.env, .git, config.php...)"
    exists, rule_id = check_rule_exists(zone_id, ruleset_id, description, headers)
    
    if exists:
        print(f"  [-] Phat hien rule tren domain lien ket. Tien hanh xoa (Rule ID: {rule_id})...")
        url = f"{CF_API_BASE}/zones/{zone_id}/rulesets/{ruleset_id}/rules/{rule_id}"
        res = make_request(url, "DELETE", headers)
        if res.get("success"):
            print("  [OK] Da xoa rule khoi domain lien ket thanh cong!")
        else:
            errors = res.get("errors", [{"message": "Unknown error"}])
            print(f"  [ERROR] Khong the xoa rule: {errors[0]['message']}")

def main():
    load_env()
    
    # Tap hop danh sach cac credentials (email, key)
    accounts = []
    
    # 1. Kiem tra cap chinh CF_EMAIL / CF_KEY
    email_main = os.environ.get("CF_EMAIL")
    key_main = os.environ.get("CF_KEY")
    if email_main and key_main:
        accounts.append((email_main, key_main))
        
    # 2. Duyet qua CF_EMAIL_1 -> CF_EMAIL_5
    for i in range(1, 6):
        email_i = os.environ.get(f"CF_EMAIL_{i}")
        key_i = os.environ.get(f"CF_KEY_{i}")
        if email_i and key_i:
            if (email_i, key_i) not in accounts:
                accounts.append((email_i, key_i))
                
    if not accounts:
        print("[ERROR] Loi: Vui long cau hinh credentials Cloudflare trong file .env!")
        sys.exit(1)
        
    print("==========================================================")
    print("        AUTO-SETUP CLOUDFLARE WAF PROBE PATH BLOCKER       ")
    print("==========================================================")
    
    for email, key in accounts:
        headers = {
            "X-Auth-Email": email,
            "X-Auth-Key": key
        }
        print(f"\n[+] Bat dau thiet lap cho tai khoan: {email}")
        try:
            print("  [*] Dang quet cac ten mien (zones) dang hoat dong...")
            zones = get_all_zones(headers)
            
            if not zones:
                print("  [-] Khong tim thay ten mien hoat dong nao.")
                continue
                
            print(f"  [OK] Tim thay {len(zones)} ten mien hoat dong.")
            
            for zone in zones:
                domain_name = zone["name"]
                zone_id = zone["id"]
                
                # Kiem tra xem domain co tro ve VPS nay khong
                is_vps_domain = check_domain_points_to_vps(zone_id, headers)
                
                if is_vps_domain:
                    print(f"  -> [A RECORD MATCH] Cau hinh bao mat cho: {domain_name} (ID: {zone_id})")
                    setup_probe_block_rule(zone_id, headers)
                else:
                    # Neu truoc do da lo tao, thi bay gio xoa di
                    delete_probe_block_rule_if_exists(zone_id, headers)
                
        except Exception as e:
            print(f"  [ERROR] Loi khi xu ly tai khoan {email}: {str(e)}")

if __name__ == "__main__":
    main()
