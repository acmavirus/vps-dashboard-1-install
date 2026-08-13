#!/bin/bash
# Tải dải IP mới nhất của Cloudflare
CF_IPV4=$(curl -s https://www.cloudflare.com/ips-v4)
CF_IPV6=$(curl -s https://www.cloudflare.com/ips-v6)

echo "Resetting web ports rules in UFW..."
# Xóa các rules cũ liên quan đến port 80 và 443
while ufw status numbered | grep -E "80|443" > /dev/null; do
    INDEX=$(ufw status numbered | grep -E "80|443" | head -n 1 | awk -F"[][]" '{print $2}')
    if [ ! -z "$INDEX" ]; then
        ufw --force delete $INDEX
    else
        break
    fi
done

echo "Adding Cloudflare IPv4 rules..."
for ip in $CF_IPV4; do
    ufw allow from $ip to any port 80 proto tcp comment 'Cloudflare IP'
    ufw allow from $ip to any port 443 proto tcp comment 'Cloudflare IP'
done

echo "Adding Cloudflare IPv6 rules..."
for ip in $CF_IPV6; do
    ufw allow from $ip to any port 80 proto tcp comment 'Cloudflare IP'
    ufw allow from $ip to any port 443 proto tcp comment 'Cloudflare IP'
done

# Đảm bảo block truy cập direct
ufw deny 80/tcp
ufw deny 443/tcp

echo "Reloading UFW..."
ufw reload
echo "Done!"
