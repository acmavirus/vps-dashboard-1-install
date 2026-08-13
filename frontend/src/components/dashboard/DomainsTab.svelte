<script lang="ts">
  import { RefreshCw, Trash2, Plus, X, Globe, Shield, ShieldCheck, ShieldAlert, Clock, Database, ArrowRight, Loader2, Star, ChevronDown, ChevronRight, ExternalLink } from "lucide-svelte"
  import type { DomainInfo, DomainDeleteState, DomainNoteState } from "./types"
  import { toast } from "../../lib/toast"
  import ConfirmModal from "../ConfirmModal.svelte"

  export let token: string | null = null
  export let domains: DomainInfo[] = []
  export let setDomainDelete: (value: DomainDeleteState) => void
  export let setDomainNote: (value: DomainNoteState) => void
  export let onScan: () => void
  export let scanning: boolean
  export let onRefresh: () => void
  export let onToggleStar: (domainName: string, currentStarred: boolean) => void
  export let onSelectDomain: (domain: DomainInfo) => void

  // SSL Management states
  let sslActionLoading = false
  let showSSLModal = false
  let selectedSSLDomain: DomainInfo | null = null
  let showIssueConfirm = false
  let domainToIssueSSL: DomainInfo | null = null

  async function handleIssueSSL(domainName: string) {
    if (!token) return
    sslActionLoading = true
    try {
      const response = await fetch("/api/ssl/issue", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token
        },
        body: JSON.stringify({ domain: domainName })
      })
      const data = await response.json()
      if (response.ok) {
        toast.success("Thành công", `Đã kích hoạt SSL Let's Encrypt cho tên miền "${domainName}".`)
        onRefresh()
      } else {
        toast.error("Thất bại", data.error || "Không thể kích hoạt SSL.")
      }
    } catch {
      toast.error("Lỗi kết nối", "Không thể kết nối tới máy chủ.")
    } finally {
      sslActionLoading = false
      showIssueConfirm = false
      domainToIssueSSL = null
    }
  }

  async function handleRenewSSL(domainName: string) {
    if (!token) return
    sslActionLoading = true
    try {
      const response = await fetch("/api/ssl/renew", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token
        },
        body: JSON.stringify({ domain: domainName })
      })
      const data = await response.json()
      if (response.ok) {
        toast.success("Thành công", `Đã gia hạn chứng chỉ SSL cho tên miền "${domainName}".`)
        onRefresh()
        if (selectedSSLDomain && selectedSSLDomain.domain === domainName) {
          selectedSSLDomain = { ...selectedSSLDomain, ssl_days: 90 }
        }
      } else {
        toast.error("Thất bại", data.error || "Gia hạn thất bại.")
      }
    } catch {
      toast.error("Lỗi kết nối", "Không thể kết nối tới máy chủ.")
    } finally {
      sslActionLoading = false
    }
  }

  function triggerSSLAction(domain: DomainInfo) {
    if (domain.ssl_active) {
      selectedSSLDomain = domain
      showSSLModal = true
    } else {
      domainToIssueSSL = domain
      showIssueConfirm = true
    }
  }

  function formatDate(dateStr: string | undefined) {
    if (!dateStr) return "-"
    const d = new Date(dateStr)
    return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
  }

  // Create website states
  let showCreateModal = false
  let createLoading = false
  let createError = ""
  let createSuccess = ""

  let domainName = ""
  let websiteType = "php" // "static", "php", "proxy"
  let phpVersion = "8.3"
  let proxyPass = "http://127.0.0.1:3000"
  let createDb = false
  let enableSSL = false

  async function handleCreateWebsite(e: Event) {
    e.preventDefault()
    createError = ""
    createSuccess = ""
    const domainClean = domainName.trim().toLowerCase()
    if (!domainClean) {
      createError = "Domain name cannot be empty"
      return
    }
    createLoading = true
    try {
      const response = await fetch("/api/domains/create", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          domain: domainClean,
          type: websiteType,
          php_version: phpVersion,
          proxy_pass: proxyPass.trim(),
          create_db: createDb,
          ssl: enableSSL
        }),
      })
      const data = await response.json().catch(() => ({}))
      if (response.ok) {
        createSuccess = `Website ${domainClean} created successfully!`
        domainName = ""
        createDb = false
        enableSSL = false
        onRefresh()
        setTimeout(() => {
          showCreateModal = false
          createSuccess = ""
        }, 1500)
      } else {
        createError = data.error || "Failed to create website"
      }
    } catch {
      createError = "Connection error"
    } finally {
      createLoading = false
    }
  }

  // ─── Domain Grouping Logic ───────────────────────────────────────────────────

  interface DomainGroup {
    rootDomain: string
    subdomains: DomainInfo[]
    expanded: boolean
    totalSSL: number
    totalOnline: number
    totalOffline: number
  }

  /**
   * Extract the root domain (last 2 or 3 parts) from a FQDN.
   * Examples:
   *   "api.thuc.me"    → "thuc.me"
   *   "thuc.me"        → "thuc.me"
   *   "sub.igeara.com" → "igeara.com"
   *   "a.b.co.uk"      → "b.co.uk"  (simple 2-part TLD heuristic)
   */
  function extractRootDomain(fqdn: string): string {
    const parts = fqdn.split(".")
    if (parts.length <= 2) return fqdn
    // Check common 2-part TLDs (co.uk, com.vn, io.vn, etc.)
    const twoPartTLDs = ["co.uk", "com.au", "com.vn", "net.vn", "io.vn", "org.vn", "edu.vn", "gov.vn"]
    const lastTwo = parts.slice(-2).join(".")
    if (twoPartTLDs.includes(lastTwo)) {
      return parts.slice(-3).join(".")
    }
    return parts.slice(-2).join(".")
  }

  // Track expanded state per group key
  let expandedGroups: Set<string> = new Set()

  function toggleGroup(rootDomain: string) {
    if (expandedGroups.has(rootDomain)) {
      expandedGroups.delete(rootDomain)
    } else {
      expandedGroups.add(rootDomain)
    }
    expandedGroups = expandedGroups // trigger reactivity
  }

  // Sort starred domains first, then by name
  $: sortedDomains = [...domains].sort((a, b) => {
    if (a.is_starred && !b.is_starred) return -1
    if (!a.is_starred && b.is_starred) return 1
    return a.domain.localeCompare(b.domain)
  })

  $: groupedDomains = (() => {
    const map = new Map<string, DomainGroup>()

    for (const d of sortedDomains) {
      const root = extractRootDomain(d.domain)
      if (!map.has(root)) {
        map.set(root, {
          rootDomain: root,
          subdomains: [],
          expanded: expandedGroups.has(root),
          totalSSL: 0,
          totalOnline: 0,
          totalOffline: 0,
        })
      }
      const group = map.get(root)!
      group.subdomains.push(d)
      if (d.ssl_active) group.totalSSL++
      if (d.status === "online") group.totalOnline++
      else if (d.status === "offline") group.totalOffline++
      group.expanded = expandedGroups.has(root)
    }

    // Sort groups: starred groups first, then alphabetically
    return Array.from(map.values()).sort((a, b) => {
      const aHasStar = a.subdomains.some(d => d.is_starred)
      const bHasStar = b.subdomains.some(d => d.is_starred)
      if (aHasStar && !bHasStar) return -1
      if (!aHasStar && bHasStar) return 1
      return a.rootDomain.localeCompare(b.rootDomain)
    })
  })()
</script>

<div class="space-y-4">
  <!-- Header Controls -->
  <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
    <div>
      <h2 class="text-lg font-medium text-foreground">Quản lý Website</h2>
      <p class="text-xs text-muted-foreground">
        {domains.length} domain · {groupedDomains.length} root domain · Cấu hình Nginx, PHP, Reverse Proxy và SSL
      </p>
    </div>

    <div class="flex items-center gap-2 w-full sm:w-auto">
      <button
        on:click={onScan}
        disabled={scanning}
        class="inline-flex h-9 items-center justify-center gap-2 rounded-lg border border-border bg-card px-4 text-xs font-medium text-muted-foreground hover:bg-secondary/40 hover:text-foreground disabled:opacity-50"
      >
        <RefreshCw size={14} class={scanning ? "animate-spin" : ""} />
        <span>{scanning ? "Đang quét..." : "Quét trạng thái"}</span>
      </button>

      <button
        on:click={() => showCreateModal = true}
        class="inline-flex h-9 items-center justify-center gap-2 rounded-lg bg-blue-600 px-4 text-xs font-medium text-white transition-colors hover:bg-blue-700 w-full sm:w-auto"
      >
        <Plus size={15} />
        <span>Thêm Website</span>
      </button>
    </div>
  </div>

  <!-- Domain Groups -->
  {#if domains.length === 0}
    <div class="rounded-xl border border-border bg-card px-6 py-12 text-center text-sm text-muted-foreground">
      Chưa có website nào được thêm. Bấm "Thêm Website" để bắt đầu.
    </div>
  {:else}
    <div class="space-y-2">
      {#each groupedDomains as group (group.rootDomain)}
        <div class="rounded-xl border border-border bg-card overflow-hidden shadow-sm transition-all">
          <!-- Group Header (Root Domain Row) -->
          <button
            type="button"
            on:click={() => toggleGroup(group.rootDomain)}
            class="w-full flex items-center gap-3 px-4 py-3 hover:bg-secondary/20 transition-colors text-left select-none group"
          >
            <!-- Chevron -->
            <span class="text-muted-foreground transition-transform duration-200 {group.expanded ? 'rotate-0' : '-rotate-90'}">
              {#if group.expanded}
                <ChevronDown size={16} />
              {:else}
                <ChevronRight size={16} />
              {/if}
            </span>

            <!-- Root domain name -->
            <Globe size={15} class="text-blue-400 shrink-0" />
            <span class="font-semibold text-sm text-foreground font-sans flex-1 truncate">
              {group.rootDomain}
            </span>

            <!-- Stats badges -->
            <div class="flex items-center gap-2 shrink-0">
              <!-- Domain count -->
              <span class="text-xs text-muted-foreground bg-secondary/50 px-2 py-0.5 rounded-full">
                {group.subdomains.length} site{group.subdomains.length > 1 ? 's' : ''}
              </span>

              <!-- Online indicator -->
              {#if group.totalOnline > 0}
                <span class="inline-flex items-center gap-1 text-[10px] text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full">
                  <span class="h-1.5 w-1.5 rounded-full bg-emerald-500 inline-block"></span>
                  {group.totalOnline}
                </span>
              {/if}

              <!-- Offline indicator -->
              {#if group.totalOffline > 0}
                <span class="inline-flex items-center gap-1 text-[10px] text-rose-400 bg-rose-500/10 px-2 py-0.5 rounded-full">
                  <span class="h-1.5 w-1.5 rounded-full bg-rose-500 inline-block"></span>
                  {group.totalOffline}
                </span>
              {/if}

              <!-- SSL summary -->
              {#if group.totalSSL === group.subdomains.length}
                <span class="inline-flex items-center gap-1 text-[10px] text-emerald-400 bg-emerald-500/10 px-2 py-0.5 rounded-full">
                  <ShieldCheck size={11} />
                  SSL ✓
                </span>
              {:else if group.totalSSL > 0}
                <span class="inline-flex items-center gap-1 text-[10px] text-amber-400 bg-amber-500/10 px-2 py-0.5 rounded-full">
                  <ShieldAlert size={11} />
                  SSL {group.totalSSL}/{group.subdomains.length}
                </span>
              {:else}
                <span class="inline-flex items-center gap-1 text-[10px] text-zinc-500 bg-zinc-500/10 px-2 py-0.5 rounded-full">
                  <Shield size={11} />
                  No SSL
                </span>
              {/if}
            </div>
          </button>

          <!-- Subdomain List (Expanded) -->
          {#if group.expanded}
            <div class="border-t border-border/60">
              {#each group.subdomains as domain, idx (domain.domain)}
                <div
                  class="flex items-center gap-3 px-4 py-3 text-xs transition-colors hover:bg-secondary/10
                    {idx < group.subdomains.length - 1 ? 'border-b border-border/30' : ''}
                    {domain.domain === group.rootDomain ? 'bg-secondary/5' : ''}
                  "
                >
                  <!-- Indent line + Star -->
                  <div class="flex items-center gap-2 shrink-0 pl-5">
                    <button
                      type="button"
                      on:click={() => onToggleStar(domain.domain, !!domain.is_starred)}
                      class="focus:outline-none transition-transform hover:scale-110 active:scale-95"
                      title={domain.is_starred ? "Bỏ nổi bật" : "Đánh dấu nổi bật"}
                    >
                      {#if domain.is_starred}
                        <Star size={12} class="text-amber-400 fill-amber-400" />
                      {:else}
                        <Star size={12} class="text-muted-foreground/30 hover:text-amber-400/80 transition-colors" />
                      {/if}
                    </button>
                  </div>

                  <!-- Domain name (clickable → detail) -->
                  <button
                    type="button"
                    on:click={() => onSelectDomain(domain)}
                    class="font-semibold text-foreground hover:text-primary hover:underline text-left transition-colors font-sans flex-1 truncate min-w-0"
                    title={domain.domain}
                  >
                    {#if domain.domain === group.rootDomain}
                      <span class="text-blue-400">{domain.domain}</span>
                    {:else}
                      {domain.domain}
                    {/if}
                  </button>

                  <!-- Note (truncated) -->
                  {#if domain.note}
                    <span class="text-muted-foreground truncate max-w-[160px] hidden md:block" title={domain.note}>
                      {domain.note}
                    </span>
                  {/if}

                  <!-- Status dot + HTTP code -->
                  <div class="flex items-center gap-2 shrink-0">
                    <span class="flex items-center gap-1.5">
                      <span class="h-1.5 w-1.5 rounded-full {domain.status === 'online' ? 'bg-emerald-500' : domain.status === 'offline' ? 'bg-rose-500' : 'bg-zinc-500'}"></span>
                      <span class="text-muted-foreground capitalize hidden sm:inline">{domain.status}</span>
                    </span>

                    <span class="tabular-nums font-mono w-10 text-right {domain.code >= 200 && domain.code < 400 ? 'text-emerald-400' : 'text-rose-400'}">
                      {domain.code || "--"}
                    </span>
                  </div>

                  <!-- SSL badge -->
                  <div class="shrink-0 w-28 text-right">
                    {#if domain.ssl_active}
                      <button
                        type="button"
                        on:click={() => triggerSSLAction(domain)}
                        class="inline-flex items-center gap-1 text-emerald-500 hover:text-emerald-400 transition-colors focus:outline-none"
                        title="Xem chi tiết hoặc gia hạn SSL"
                      >
                        <ShieldCheck size={12} />
                        <span>{domain.ssl_days}d</span>
                      </button>
                    {:else}
                      <button
                        type="button"
                        on:click={() => triggerSSLAction(domain)}
                        class="inline-flex items-center gap-1 text-amber-500 hover:text-amber-400 transition-colors focus:outline-none"
                        title="Kích hoạt Let's Encrypt SSL"
                      >
                        <ShieldAlert size={12} />
                        <span class="underline decoration-dotted">No SSL</span>
                      </button>
                    {/if}
                  </div>

                  <!-- Actions -->
                  <div class="flex items-center gap-3 shrink-0">
                    <a
                      href={`https://${domain.domain}`}
                      target="_blank"
                      rel="noreferrer"
                      class="inline-flex items-center gap-1 text-blue-400 hover:text-blue-300 font-medium transition-colors"
                      title="Truy cập website"
                    >
                      <ExternalLink size={12} />
                      <span class="hidden lg:inline">Truy cập</span>
                    </a>

                    <button
                      type="button"
                      on:click={() => setDomainNote({ domain: domain.domain, note: domain.note || "" })}
                      class="text-cyan-400 hover:text-cyan-300 font-medium transition-colors"
                    >
                      Ghi chú
                    </button>

                    <button
                      type="button"
                      on:click={() => setDomainDelete({ domain: domain.domain, deleteDb: false, deleteRoot: false })}
                      class="inline-flex items-center gap-1 text-rose-400 hover:text-rose-300 transition-colors font-medium"
                    >
                      <Trash2 size={12} />
                      <span class="hidden sm:inline">Xóa</span>
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}

  <!-- Create Website Modal -->
  {#if showCreateModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-xs">
      <div class="w-full max-w-lg rounded-2xl border border-border bg-card shadow-2xl overflow-hidden transform transition-all">
        <!-- Modal Header -->
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <div>
            <h3 class="text-base font-medium text-foreground">Thêm Website mới</h3>
            <p class="text-xs text-muted-foreground">Tạo cấu hình Nginx, cấp phát root folder & DB</p>
          </div>
          <button
            type="button"
            on:click={() => !createLoading && (showCreateModal = false)}
            class="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <X size={16} />
          </button>
        </div>

        <!-- Modal Form -->
        <form on:submit={handleCreateWebsite}>
          <div class="space-y-4 p-6">
            <!-- Domain Name Input -->
            <div class="space-y-2">
              <label for="domainName" class="text-xs font-light text-muted-foreground">Tên Domain</label>
              <input
                id="domainName"
                type="text"
                bind:value={domainName}
                class="w-full rounded-lg border border-border bg-secondary/30 px-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
                placeholder="example.com hoặc shop.thuc.me"
                required
                disabled={createLoading}
              />
            </div>

            <!-- Website Type Select -->
            <div class="space-y-2">
              <!-- svelte-ignore a11y-label-has-associated-control -->
              <label class="text-xs font-light text-muted-foreground">Loại Website</label>
              <div class="grid grid-cols-3 gap-2">
                <button
                  type="button"
                  on:click={() => { websiteType = "static"; createDb = false; }}
                  class="flex flex-col items-center gap-1 rounded-xl border p-3 text-center transition-all {websiteType === 'static' ? 'border-blue-500 bg-blue-500/10 text-blue-400' : 'border-border bg-secondary/10 hover:bg-secondary/20 text-muted-foreground'}"
                  disabled={createLoading}
                >
                  <Globe size={18} />
                  <span class="text-[10px] font-medium">Static (HTML)</span>
                </button>

                <button
                  type="button"
                  on:click={() => websiteType = "php"}
                  class="flex flex-col items-center gap-1 rounded-xl border p-3 text-center transition-all {websiteType === 'php' ? 'border-indigo-500 bg-indigo-500/10 text-indigo-400' : 'border-border bg-secondary/10 hover:bg-secondary/20 text-muted-foreground'}"
                  disabled={createLoading}
                >
                  <Database size={18} />
                  <span class="text-[10px] font-medium">PHP Application</span>
                </button>

                <button
                  type="button"
                  on:click={() => { websiteType = "proxy"; createDb = false; }}
                  class="flex flex-col items-center gap-1 rounded-xl border p-3 text-center transition-all {websiteType === 'proxy' ? 'border-orange-500 bg-orange-500/10 text-orange-400' : 'border-border bg-secondary/10 hover:bg-secondary/20 text-muted-foreground'}"
                  disabled={createLoading}
                >
                  <ArrowRight size={18} />
                  <span class="text-[10px] font-medium">Reverse Proxy</span>
                </button>
              </div>
            </div>

            <!-- PHP Version Select -->
            {#if websiteType === "php"}
              <div class="space-y-2">
                <!-- svelte-ignore a11y-label-has-associated-control -->
                <label class="text-xs font-light text-muted-foreground">Phiên bản PHP</label>
                <div class="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    on:click={() => phpVersion = "8.3"}
                    class="rounded-lg border py-2 text-center text-xs transition-all {phpVersion === '8.3' ? 'border-blue-500 bg-blue-500/10 text-blue-400' : 'border-border bg-secondary/10 text-muted-foreground'}"
                    disabled={createLoading}
                  >
                    PHP 8.3 (Mặc định)
                  </button>
                  <button
                    type="button"
                    on:click={() => phpVersion = "7.4"}
                    class="rounded-lg border py-2 text-center text-xs transition-all {phpVersion === '7.4' ? 'border-blue-500 bg-blue-500/10 text-blue-400' : 'border-border bg-secondary/10 text-muted-foreground'}"
                    disabled={createLoading}
                  >
                    PHP 7.4
                  </button>
                </div>
              </div>
            {/if}

            <!-- Proxy Pass Target -->
            {#if websiteType === "proxy"}
              <div class="space-y-2">
                <label for="proxyPass" class="text-xs font-light text-muted-foreground">Proxy Destination (Nơi chuyển tiếp)</label>
                <input
                  id="proxyPass"
                  type="text"
                  bind:value={proxyPass}
                  class="w-full rounded-lg border border-border bg-secondary/30 px-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="http://127.0.0.1:3000"
                  required
                  disabled={createLoading}
                />
              </div>
            {/if}

            <!-- Checkboxes -->
            <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-2">
              {#if websiteType === "php"}
                <!-- svelte-ignore a11y-label-has-associated-control -->
                <label class="flex items-center gap-3 rounded-xl border border-border bg-secondary/10 px-4 py-3 cursor-pointer select-none">
                  <input
                    type="checkbox"
                    bind:checked={createDb}
                    class="h-4 w-4 rounded border-zinc-700 bg-zinc-800 focus:ring-1 focus:ring-blue-500"
                    disabled={createLoading}
                  />
                  <div class="flex flex-col">
                    <span class="text-xs font-medium text-foreground">Tạo Database</span>
                    <span class="text-[9px] text-muted-foreground">Sinh DB & User tự động</span>
                  </div>
                </label>
              {/if}

              <!-- svelte-ignore a11y-label-has-associated-control -->
              <label class="flex items-center gap-3 rounded-xl border border-border bg-secondary/10 px-4 py-3 cursor-pointer select-none">
                <input
                  type="checkbox"
                  bind:checked={enableSSL}
                  class="h-4 w-4 rounded border-zinc-700 bg-zinc-800 focus:ring-1 focus:ring-blue-500"
                  disabled={createLoading}
                />
                <div class="flex flex-col">
                  <span class="text-xs font-medium text-foreground flex items-center gap-1">
                    SSL Let's Encrypt <ShieldCheck size={11} class="text-emerald-400" />
                  </span>
                  <span class="text-[9px] text-muted-foreground">Tự động cấu hình HTTPS</span>
                </div>
              </label>
            </div>

            <!-- Messages -->
            {#if createError}
              <p class="text-xs text-rose-400 text-center bg-rose-500/10 border border-rose-500/20 py-2 rounded-lg">{createError}</p>
            {/if}
            {#if createSuccess}
              <p class="text-xs text-emerald-400 text-center bg-emerald-500/10 border border-emerald-500/20 py-2 rounded-lg">{createSuccess}</p>
            {/if}
          </div>

          <!-- Modal Footer -->
          <div class="flex items-center justify-end gap-3 border-t border-border px-6 py-4">
            <button
              type="button"
              on:click={() => showCreateModal = false}
              disabled={createLoading}
              class="rounded-lg border border-border px-4 py-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground disabled:opacity-50"
            >
              Hủy bỏ
            </button>
            <button
              type="submit"
              disabled={createLoading || !!createSuccess}
              class="inline-flex h-9 items-center justify-center gap-2 rounded-lg bg-blue-600 px-5 text-xs font-medium text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
            >
              {#if createLoading}
                <Loader2 size={13} class="animate-spin" />
                <span>Đang xử lý...</span>
              {:else}
                <span>Tạo Website</span>
              {/if}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- SSL Detail Modal -->
  {#if showSSLModal && selectedSSLDomain}
    <div class="fixed inset-0 z-[9900] flex items-center justify-center bg-black/60 p-4 backdrop-blur-xs">
      <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl overflow-hidden transform transition-all">
        <!-- Modal Header -->
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <div class="flex items-center gap-2">
            <Shield class="text-emerald-500" size={18} />
            <h3 class="text-base font-semibold text-foreground">Chi tiết SSL Certificate</h3>
          </div>
          <button
            type="button"
            on:click={() => showSSLModal = false}
            class="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <X size={16} />
          </button>
        </div>

        <!-- Modal Body -->
        <div class="p-6 space-y-4">
          <div class="grid grid-cols-3 gap-2 py-1.5 border-b border-border/50 text-xs">
            <span class="text-muted-foreground font-light">Tên miền</span>
            <span class="col-span-2 font-semibold text-foreground">{selectedSSLDomain.domain}</span>
          </div>
          <div class="grid grid-cols-3 gap-2 py-1.5 border-b border-border/50 text-xs">
            <span class="text-muted-foreground font-light">Nhà phát hành</span>
            <span class="col-span-2 text-foreground font-mono">{selectedSSLDomain.ssl_issuer || "Let's Encrypt"}</span>
          </div>
          <div class="grid grid-cols-3 gap-2 py-1.5 border-b border-border/50 text-xs">
            <span class="text-muted-foreground font-light">Ngày hết hạn</span>
            <span class="col-span-2 text-foreground font-mono">{formatDate(selectedSSLDomain.ssl_expiry)}</span>
          </div>
          <div class="grid grid-cols-3 gap-2 py-1.5 text-xs">
            <span class="text-muted-foreground font-light">Thời hạn còn lại</span>
            <span class="col-span-2 font-semibold font-mono">
              <span class={selectedSSLDomain.ssl_days && selectedSSLDomain.ssl_days <= 15 ? "text-rose-500 font-bold" : selectedSSLDomain.ssl_days && selectedSSLDomain.ssl_days <= 30 ? "text-amber-500" : "text-emerald-500"}>
                {selectedSSLDomain.ssl_days} ngày
              </span>
            </span>
          </div>
        </div>

        <!-- Modal Footer -->
        <div class="flex items-center justify-end gap-3 border-t border-border px-6 py-4 bg-secondary/10">
          <button
            type="button"
            on:click={() => showSSLModal = false}
            class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            Đóng
          </button>
          <button
            type="button"
            on:click={() => handleRenewSSL(selectedSSLDomain?.domain || "")}
            disabled={sslActionLoading}
            class="inline-flex h-9 items-center justify-center gap-2 rounded-lg bg-blue-600 px-5 text-xs font-semibold text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
          >
            {#if sslActionLoading}
              <Loader2 size={13} class="animate-spin" />
              <span>Đang gia hạn...</span>
            {:else}
              <RefreshCw size={13} />
              <span>Gia hạn SSL (Renew)</span>
            {/if}
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Confirm Issue SSL Modal -->
  <ConfirmModal
    bind:isOpen={showIssueConfirm}
    title="Kích hoạt Let's Encrypt SSL"
    message="Bạn có muốn kích hoạt và cấp chứng chỉ Let's Encrypt SSL miễn phí cho tên miền '{domainToIssueSSL?.domain}' không? Tên miền cần phải được cấu hình DNS trỏ đúng về địa chỉ IP của máy chủ này để quá trình cấp chứng chỉ thành công."
    confirmLabel="Kích hoạt (Issue)"
    cancelLabel="Hủy"
    variant="info"
    onConfirm={() => handleIssueSSL(domainToIssueSSL?.domain || "")}
  />

</div>
