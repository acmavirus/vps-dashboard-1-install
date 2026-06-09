<script lang="ts">
  import { onMount } from "svelte"
  import { ArrowLeft, Globe, Terminal, FileText, Settings, ShieldCheck, Loader2, AlertTriangle, Save, Check } from "lucide-svelte"
  import type { DomainInfo } from "./types"

  export let token: string | null = null
  export let domain: DomainInfo
  export let logs: any = null
  export let onBack: () => void

  let activeTab: "overview" | "nginx-config" | "nginx-logs" = "overview"
  let activeLogTab: "access" | "error" = "access"

  // Note editor states
  let noteContent = domain.note || ""
  let isEditingNote = false
  let noteSaving = false
  let noteError = ""
  let noteSuccess = ""

  // Config editor states
  let configContent = ""
  let configPath = ""
  let configLoading = false
  let configSaving = false
  let configError = ""
  let configSuccess = ""

  onMount(() => {
    fetchConfig()
  })

  async function fetchConfig() {
    configLoading = true
    configError = ""
    try {
      const res = await fetch(`/api/domains/config?domain=${encodeURIComponent(domain.domain)}`, {
        headers: { Authorization: token || "" }
      })
      const data = await res.json()
      if (res.ok) {
        configContent = data.content
        configPath = data.path
      } else {
        configError = data.error || "Failed to load Nginx configuration"
      }
    } catch (err) {
      configError = "Connection error"
    } finally {
      configLoading = false
    }
  }

  async function saveConfig() {
    configSaving = true
    configError = ""
    configSuccess = ""
    try {
      const res = await fetch("/api/domains/config", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || ""
        },
        body: JSON.stringify({
          domain: domain.domain,
          content: configContent
        })
      })
      const data = await res.json()
      if (res.ok) {
        configSuccess = "Nginx configuration saved and reloaded successfully!"
        setTimeout(() => {
          configSuccess = ""
        }, 3000)
      } else {
        configError = data.error || "Failed to save Nginx configuration"
      }
    } catch (err) {
      configError = "Connection error"
    } finally {
      configSaving = false
    }
  }

  async function saveNote() {
    noteSaving = true
    noteError = ""
    noteSuccess = ""
    try {
      const res = await fetch("/api/domains/note", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || ""
        },
        body: JSON.stringify({
          domain: domain.domain,
          note: noteContent
        })
      })
      const data = await res.json()
      if (res.ok) {
        domain.note = noteContent
        noteSuccess = "Ghi chú đã được lưu thành công!"
        isEditingNote = false
        setTimeout(() => {
          noteSuccess = ""
        }, 3000)
      } else {
        noteError = data.error || "Không thể lưu ghi chú"
      }
    } catch (err) {
      noteError = "Lỗi kết nối mạng"
    } finally {
      noteSaving = false
    }
  }
</script>

<div class="space-y-6">
  <!-- Detail View Header -->
  <div class="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
    <div class="flex items-center gap-3">
      <button
        type="button"
        on:click={onBack}
        class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-card text-muted-foreground hover:bg-secondary hover:text-foreground transition-all shadow-xs"
      >
        <ArrowLeft size={16} />
      </button>
      <div>
        <div class="flex items-center gap-2 text-xs text-muted-foreground">
          <span>Website</span>
          <span>/</span>
          <span>Chi tiết</span>
          <span>/</span>
          <span class="font-medium text-foreground">{domain.domain}</span>
        </div>
        <h2 class="text-xl font-semibold tracking-tight text-foreground flex items-center gap-2 mt-1">
          <Globe size={18} class="text-primary" />
          {domain.domain}
        </h2>
      </div>
    </div>

    <!-- Quick Status Badge -->
    <div class="flex items-center gap-3">
      <div class="flex items-center gap-2 rounded-lg border border-border bg-card px-3 py-1.5 shadow-xs">
        <span class="h-2.5 w-2.5 rounded-full {domain.status === 'online' ? 'bg-emerald-500 animate-pulse' : 'bg-rose-500'}" />
        <span class="text-xs font-medium text-foreground capitalize">{domain.status}</span>
        <span class="text-xs text-muted-foreground px-1 border-l border-border tabular-nums">{domain.code || "--"}</span>
      </div>
    </div>
  </div>

  <!-- Domain Tabs Navigation -->
  <div class="flex border-b border-border bg-card rounded-xl p-1 border shadow-xs gap-1">
    <button
      type="button"
      on:click={() => activeTab = "overview"}
      class="flex items-center gap-2 rounded-lg px-4 py-2.5 text-xs font-medium transition-all {activeTab === 'overview' ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-secondary/50 hover:text-foreground'}"
    >
      <Settings size={14} />
      <span>Tổng quan & Ghi chú</span>
    </button>
    <button
      type="button"
      on:click={() => activeTab = "nginx-config"}
      class="flex items-center gap-2 rounded-lg px-4 py-2.5 text-xs font-medium transition-all {activeTab === 'nginx-config' ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-secondary/50 hover:text-foreground'}"
    >
      <Terminal size={14} />
      <span>Cấu hình Nginx</span>
    </button>
    <button
      type="button"
      on:click={() => activeTab = "nginx-logs"}
      class="flex items-center gap-2 rounded-lg px-4 py-2.5 text-xs font-medium transition-all {activeTab === 'nginx-logs' ? 'bg-primary/10 text-primary' : 'text-muted-foreground hover:bg-secondary/50 hover:text-foreground'}"
    >
      <FileText size={14} />
      <span>Nhật ký truy cập (Logs)</span>
    </button>
  </div>

  <!-- Tab Contents -->
  <div class="min-h-[450px]">
    {#if activeTab === "overview"}
      <div class="grid grid-cols-1 gap-6 md:grid-cols-3">
        <!-- Overview Stats Bento Grid -->
        <div class="md:col-span-2 space-y-6">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="rounded-xl border border-border bg-card p-5 shadow-xs flex flex-col justify-between h-28">
              <span class="text-[10px] text-muted-foreground uppercase font-semibold">Trạng thái</span>
              <div class="flex items-baseline gap-2 mt-2">
                <span class="text-2xl font-bold capitalize text-foreground">{domain.status}</span>
                <span class="h-2 w-2 rounded-full {domain.status === 'online' ? 'bg-emerald-500' : 'bg-rose-500'}" />
              </div>
            </div>
            <div class="rounded-xl border border-border bg-card p-5 shadow-xs flex flex-col justify-between h-28">
              <span class="text-[10px] text-muted-foreground uppercase font-semibold">HTTP Code</span>
              <span class="text-3xl font-extrabold text-foreground tracking-tight mt-2 tabular-nums">{domain.code || "--"}</span>
            </div>
            <div class="rounded-xl border border-border bg-card p-5 shadow-xs flex flex-col justify-between sm:col-span-2 h-28">
              <span class="text-[10px] text-muted-foreground uppercase font-semibold">Đường dẫn gốc (Web Root)</span>
              <div class="flex items-center justify-between mt-2">
                <span class="text-sm font-mono bg-secondary/35 px-2.5 py-1 rounded-md text-foreground select-all truncate">/home/{domain.domain}</span>
                <span class="text-[10px] text-muted-foreground flex items-center gap-1 font-medium"><ShieldCheck size={11} class="text-emerald-400" /> Hệ thống bảo vệ</span>
              </div>
            </div>
          </div>

          <!-- Note / DB Info Card -->
          <div class="rounded-xl border border-border bg-card p-5 shadow-xs space-y-4">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold text-foreground">Thông tin Database / Ghi chú</h3>
              {#if !isEditingNote}
                <button
                  type="button"
                  on:click={() => isEditingNote = true}
                  class="rounded-lg border border-border px-3 py-1.5 text-[11px] font-medium text-foreground hover:bg-secondary transition-colors"
                >
                  Sửa ghi chú
                </button>
              {/if}
            </div>

            {#if isEditingNote}
              <div class="space-y-3">
                <textarea
                  bind:value={noteContent}
                  class="w-full min-h-[120px] rounded-lg border border-border bg-secondary/15 p-3 text-xs focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                  placeholder="Điền thông tin kết nối DB, ghi chú tài khoản admin hoặc cấu hình đặc biệt của site..."
                />
                {#if noteError}
                  <p class="text-xs text-rose-500">{noteError}</p>
                {/if}
                <div class="flex gap-2">
                  <button
                    type="button"
                    on:click={saveNote}
                    disabled={noteSaving}
                    class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3.5 py-2 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
                  >
                    {#if noteSaving}
                      <Loader2 size={12} class="animate-spin" />
                    {:else}
                      <Save size={12} />
                    {/if}
                    <span>Lưu lại</span>
                  </button>
                  <button
                    type="button"
                    on:click={() => { isEditingNote = false; noteContent = domain.note || ""; }}
                    disabled={noteSaving}
                    class="rounded-lg border border-border px-3.5 py-2 text-xs font-medium text-foreground hover:bg-secondary disabled:opacity-50 transition-colors"
                  >
                    Hủy
                  </button>
                </div>
              </div>
            {:else}
              {#if domain.note}
                <div class="bg-secondary/20 border border-border/30 rounded-xl p-4 text-xs text-foreground font-mono whitespace-pre-wrap leading-relaxed select-text">
                  {domain.note}
                </div>
              {:else}
                <div class="flex flex-col items-center justify-center py-6 text-center border border-dashed border-border rounded-xl bg-secondary/5">
                  <span class="text-xs text-muted-foreground">Chưa có thông tin ghi chú hay database nào.</span>
                  <button
                    type="button"
                    on:click={() => isEditingNote = true}
                    class="mt-2 text-xs font-semibold text-primary hover:underline"
                  >
                    Tạo ghi chú ngay
                  </button>
                </div>
              {/if}
            {/if}

            {#if noteSuccess}
              <div class="flex items-center gap-2 text-xs text-emerald-500 bg-emerald-500/10 border border-emerald-500/20 px-3 py-2 rounded-lg">
                <Check size={12} />
                <span>{noteSuccess}</span>
              </div>
            {/if}
          </div>
        </div>

        <!-- Right Side Quick Info Bar -->
        <div class="space-y-6">
          <div class="rounded-xl border border-border bg-card p-5 shadow-xs space-y-4">
            <h3 class="text-sm font-semibold text-foreground border-b border-border pb-2">Thông tin máy chủ</h3>
            <div class="space-y-3 text-xs">
              <div class="flex justify-between">
                <span class="text-muted-foreground">Loại máy chủ</span>
                <span class="font-medium text-foreground">Nginx Server</span>
              </div>
              <div class="flex justify-between">
                <span class="text-muted-foreground">Quyền quản lý</span>
                <span class="font-medium text-foreground text-emerald-400">adm / root</span>
              </div>
              <div class="flex justify-between">
                <span class="text-muted-foreground">Nơi lưu cấu hình</span>
                <span class="font-mono text-[10px] text-foreground truncate max-w-[150px]" title={configPath}>{configPath || "Đang quét..."}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    {:else if activeTab === "nginx-config"}
      <!-- Nginx Config Tab -->
      <div class="rounded-xl border border-border bg-card p-5 shadow-xs flex flex-col h-[70vh] min-h-[500px]">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between border-b border-border pb-4 mb-4 shrink-0">
          <div>
            <h3 class="text-sm font-semibold text-foreground">Cấu hình Nginx Virtual Host</h3>
            <p class="text-xs text-muted-foreground font-mono mt-1">{configPath}</p>
          </div>
          <button
            type="button"
            on:click={saveConfig}
            disabled={configLoading || configSaving}
            class="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-4 py-2 text-xs font-medium text-white hover:bg-blue-700 disabled:opacity-50 transition-colors shadow-sm shrink-0"
          >
            {#if configSaving}
              <Loader2 size={13} class="animate-spin" />
              <span>Đang lưu...</span>
            {:else}
              <Save size={13} />
              <span>Lưu cấu hình</span>
            {/if}
          </button>
        </div>

        {#if configLoading}
          <div class="flex-1 flex flex-col items-center justify-center">
            <Loader2 size={24} class="animate-spin text-primary" />
            <span class="text-xs text-muted-foreground mt-2">Đang đọc cấu hình máy chủ...</span>
          </div>
        {:else}
          <div class="flex-1 flex flex-col min-h-0 space-y-4">
            <!-- Warnings / Errors -->
            {#if configError}
              <div class="bg-rose-500/10 border border-rose-500/25 text-rose-400 p-4 rounded-xl text-xs font-mono whitespace-pre-wrap overflow-y-auto max-h-[150px] leading-relaxed shrink-0">
                <div class="flex items-center gap-2 font-bold mb-1 text-rose-500">
                  <AlertTriangle size={13} />
                  Lỗi cú pháp Nginx (Đã khôi phục cấu hình cũ):
                </div>
                {configError}
              </div>
            {/if}
            {#if configSuccess}
              <div class="bg-emerald-500/10 border border-emerald-500/25 text-emerald-400 p-4 rounded-xl text-xs flex items-center gap-2 shrink-0">
                <Check size={14} class="text-emerald-500" />
                {configSuccess}
              </div>
            {/if}

            <!-- Text editor -->
            <div class="flex-1 bg-zinc-950 rounded-xl border border-zinc-800 p-1 flex min-h-0">
              <textarea
                bind:value={configContent}
                class="flex-1 w-full h-full bg-transparent p-4 border-0 focus:outline-none focus:ring-0 font-mono text-[11px] text-zinc-300 leading-relaxed overflow-y-auto resize-none select-text whitespace-pre"
                spellcheck="false"
              />
            </div>
          </div>
        {/if}
      </div>
    {:else if activeTab === "nginx-logs"}
      <!-- Nginx Logs Tab -->
      <div class="rounded-xl border border-border bg-card p-5 shadow-xs flex flex-col h-[70vh] min-h-[500px]">
        <div class="flex items-center justify-between border-b border-border pb-3 mb-4 shrink-0">
          <div>
            <h3 class="text-sm font-semibold text-foreground">Xem nhật ký Nginx thực tế</h3>
            <p class="text-xs text-muted-foreground mt-0.5">Tải trực tiếp 30 dòng nhật ký mới nhất từ VPS</p>
          </div>

          <!-- Log type selector -->
          <div class="rounded-md border border-border bg-background/60 p-0.5 flex">
            <button
              type="button"
              on:click={() => activeLogTab = "access"}
              class="rounded-[4px] px-3.5 py-1.5 text-[10px] transition-all {activeLogTab === 'access' ? 'bg-secondary text-foreground font-medium shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
            >
              Access Log
            </button>
            <button
              type="button"
              on:click={() => activeLogTab = "error"}
              class="rounded-[4px] px-3.5 py-1.5 text-[10px] transition-all {activeLogTab === 'error' ? 'bg-secondary text-foreground font-medium shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
            >
              Error Log
            </button>
          </div>
        </div>

        <!-- Logs Console -->
        <div class="flex-1 bg-zinc-950 rounded-xl border border-zinc-800 p-4 font-mono text-[11px] text-zinc-300 overflow-y-auto leading-relaxed select-text whitespace-pre-wrap min-h-0">
          {#if logs && logs.nginx_sites}
            {@const siteLog = logs.nginx_sites.find(item => item.domain === domain.domain)}
            {#if siteLog}
              {@const logContent = activeLogTab === "access" ? siteLog.access?.content : siteLog.error?.content}
              {#if logContent}
                {logContent}
              {:else}
                <span class="text-zinc-500">[Nhật ký trống hoặc chưa có dữ liệu ghi nhận]</span>
              {/if}
            {:else}
              <span class="text-zinc-500">[Không tìm thấy dữ liệu log của domain này]</span>
            {/if}
          {:else}
            <span class="text-zinc-500">Đang tải nhật ký truy cập từ hệ thống...</span>
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>
