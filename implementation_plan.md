# Kế Hoạch: Xây Dựng Lại UI Quản Lý Website

## Mục Tiêu

Thay thế danh sách flat (bảng table phẳng) thành **cấu trúc nhóm Domain → Subdomain**:
- Hiển thị danh sách **Root Domain** (expandable card)
- Khi click → mở rộng và hiển thị tất cả **Subdomain** thuộc root domain đó
- Subdomain hiển thị thông tin đầy đủ (status, SSL, HTTP code, actions)

---

## Thiết Kế UI (Không thay đổi API backend)

### Layout tổng quan:
```
┌─────────────────────────────────────────────────────┐
│  Quản lý Website          [Quét trạng thái] [+ Thêm]│
├─────────────────────────────────────────────────────┤
│                                                     │
│  ▼ thuc.me                          12 domains  SSL ✓│
│  ├─ api.thuc.me       online  200  8903  SSL 82 ngày │
│  ├─ socket.thuc.me    online  200  3031  SSL 75 ngày │
│  ├─ chatai.thuc.me    online  200  3050  SSL 90 ngày │
│  └─ statuslive.thuc.me online 200  --    Chưa SSL    │
│                                                     │
│  ▶ igeara.com                        3 domains  SSL ✓│
│                                                     │
│  ▶ Không có Root Domain              5 domains      │
│    (domains không match root nào)                   │
└─────────────────────────────────────────────────────┘
```

---

## Thuật Toán Nhóm Domain (Frontend-only)

Logic grouping thực hiện hoàn toàn ở frontend — **không cần thay đổi backend**:

```typescript
// Từ flat list domains[], tạo cấu trúc nhóm:
// 1. Tìm tất cả root domain (2 phần: example.com)
// 2. Nhóm subdomain vào root tương ứng
// 3. Subdomain không khớp root nào → nhóm "Khác"

interface DomainGroup {
  rootDomain: string        // "thuc.me", "igeara.com"
  subdomains: DomainInfo[]  // Tất cả domains thuộc root này
  expanded: boolean         // State collapse/expand
  hasSSL: boolean           // Tất cả SSL?
  anyOffline: boolean       // Có domain offline?
}
```

---

## Proposed Changes

### Component: [DomainsTab.svelte](file:///d:/laragon/www_thuc/vps-dashboard-1-install/frontend/src/components/dashboard/DomainsTab.svelte)

#### Thay đổi:
1. **Thêm grouping logic** (`$: groupedDomains = groupDomains(sortedDomains)`)
2. **Thay thế `<table>`** bằng cấu trúc **Card expandable per group**
3. **Root Domain Row**: tên domain, số lượng subdomain, tổng trạng thái SSL, badge
4. **Subdomain Rows** (khi expanded): giữ nguyên toàn bộ actions hiện có
5. **Thêm state**: `expandedGroups: Set<string>` để track group nào đang mở

#### Giữ nguyên:
- Tất cả logic SSL (issue, renew modal)
- Modal tạo website
- Modal xác nhận xóa
- Star/Note/Delete actions
- `onSelectDomain` callback

---

## Mockup UI Chi Tiết

**Root Domain Card (collapsed)**:
```
┌──────────────────────────────────────────────────┐
│ ▶ thuc.me        [12 sites] [●online x10 ●off x2] [SSL ✓]  │
└──────────────────────────────────────────────────┘
```

**Root Domain Card (expanded)**:
```
┌──────────────────────────────────────────────────┐
│ ▼ thuc.me        [12 sites] [●online x10] [SSL ✓]          │
├──────────────────────────────────────────────────┤
│   ☆ 🌐 api.thuc.me     ● online  200  SSL ✓ 82d  Truy cập Google Ghi chú Xóa │
│   ☆ 🌐 socket.thuc.me  ● online  200  SSL ✓ 75d  ...       │
│   ★ 🌐 chatai.thuc.me  ● online  200  SSL ✓ 90d  ...       │
│   ☆ 🌐 thuc.me         ● online  200  ⚠ Chưa SSL ...       │
└──────────────────────────────────────────────────┘
```

---

## Open Questions

> [!IMPORTANT]
> **Root domain itself**: `thuc.me` vừa là root group vừa là 1 domain trong list (có Nginx config). Nên đặt nó vào nhóm của chính nó luôn hay nhóm riêng? → **Đề xuất**: Đặt vào nhóm `thuc.me`, nó sẽ là mục đầu tiên trong expanded list.

> [!IMPORTANT]
> **Expand mặc định**: Mặc định tất cả group đóng hay mở? → **Đề xuất**: Đóng tất cả, group có starred domain thì mở sẵn.

> [!NOTE]
> **Tìm kiếm / Filter**: Hiện tại không có search. Khi chuyển sang dạng group, có cần thêm search box không? → Không bắt buộc cho lần này, có thể thêm sau.

---

## Verification Plan

- Build frontend: `npm run build` trong `/frontend`  
- Kiểm tra tất cả actions (SSL, delete, note, star) vẫn hoạt động
- Test với domain list thực tế từ VPS
