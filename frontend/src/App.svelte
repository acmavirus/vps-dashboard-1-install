<script lang="ts">
  import { onMount, onDestroy, afterUpdate } from "svelte"
  import {
    LayoutDashboard,
    Globe,
    Server,
    Cpu,
    Database,
    HardDrive,
    Terminal,
    FileText,
    Key,
    ShieldAlert,
    ShoppingBag,
    Settings,
    LogOut,
    RefreshCw,
    Wrench,
    Power,
    Play,
    Square,
    RotateCcw,
    AlertTriangle,
    Box,
    Menu,
    X,
    Clock,
    Activity,
    ChevronRight,
    Plus
  } from "lucide-svelte"

  import OverviewTab from "./components/dashboard/OverviewTab.svelte"
  import ProcessesTab from "./components/dashboard/ProcessesTab.svelte"
  import DockerTab from "./components/dashboard/DockerTab.svelte"
  import NodesTab from "./components/dashboard/NodesTab.svelte"
  import DomainsTab from "./components/dashboard/DomainsTab.svelte"
  import LogsTab from "./components/dashboard/LogsTab.svelte"
  import SecurityTab from "./components/dashboard/SecurityTab.svelte"
  import FilesTab from "./components/dashboard/FilesTab.svelte"
  import DatabasesTab from "./components/dashboard/DatabasesTab.svelte"
  import AppStoreTab from "./components/dashboard/AppStoreTab.svelte"
  import FtpTab from "./components/dashboard/FtpTab.svelte"
  import CronTab from "./components/dashboard/CronTab.svelte"
  import SettingsTab from "./components/dashboard/SettingsTab.svelte"

  import type {
    AllLogs,
    ContainerInfo,
    DomainDeleteState,
    DomainInfo,
    DomainNoteState,
    LogTabItem,
    Pm2Process,
    ProcessInfo,
    Stats,
  } from "./components/dashboard/types"

  const VERSION = "2.1.1"

  let currentTheme = localStorage.getItem("selected_theme") || "aapanel"
  let isDark = localStorage.getItem("selected_dark") !== "false"

  function applyTheme() {
    if (typeof document === 'undefined') return
    const root = document.documentElement
    root.classList.remove("theme-aapanel", "theme-violet", "theme-forest", "theme-abyss", "theme-amber", "dark")
    if (currentTheme !== "slate") {
      root.classList.add(`theme-${currentTheme}`)
    }
    if (isDark) {
      root.classList.add("dark")
    }
    localStorage.setItem("selected_theme", currentTheme)
    localStorage.setItem("selected_dark", String(isDark))
  }

  $: {
    if (currentTheme || isDark !== undefined) {
      applyTheme()
    }
  }

  let token = localStorage.getItem("auth_token")
  let stats: Stats | null = null
  let history: { t: string; v: number }[] = []
  let logs: AllLogs | null = null
  let processes: ProcessInfo[] = []
  let containers: ContainerInfo[] = []
  let pm2: Pm2Process[] = []
  let domains: DomainInfo[] = []
  
  let domainDelete: DomainDeleteState | null = null
  let domainDeleteLoading = false
  let domainNote: DomainNoteState | null = null
  let domainNoteLoading = false
  let domainScanning = false
  
  let appTab = "overview"
  let live = false
  let logTab = "system"
  let siteTab: "access" | "error" = "access"
  let autoScroll = true
  let nav = false
  
  let username = ""
  let password = ""
  let error = ""
  let loading = false

  let es: EventSource | null = null
  let logEndRef: HTMLDivElement | null = null
  let pollIntervalId: number | undefined

  const formatUptime = (seconds: number) => {
    const days = Math.floor(seconds / 86400)
    const hours = Math.floor((seconds % 86400) / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)

    return days > 0 ? `${days}d ${hours}h ${minutes}m` : `${hours}h ${minutes}m`
  }

  function push(data: { stats?: Stats; logs?: AllLogs }) {
    if (data.stats) {
      stats = data.stats
      const timeLabel = new Date().toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      })
      history = [...history.slice(-59), { t: timeLabel, v: data.stats.cpu }]
    }

    if (data.logs) {
      logs = data.logs
    }
  }

  async function handleLogin(event: Event) {
    event.preventDefault()
    loading = true
    error = ""

    try {
      const response = await fetch("/api/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      })
      const data = await response.json()

      if (response.ok) {
        localStorage.setItem("auth_token", data.token)
        token = data.token
        initDashboard()
      } else {
        error = data.error || "Login failed"
      }
    } catch {
      error = "Server error"
    } finally {
      loading = false
    }
  }

  function handleLogout() {
    localStorage.removeItem("auth_token")
    token = null
    if (es) {
      es.close()
      es = null
    }
    if (pollIntervalId) {
      clearInterval(pollIntervalId)
      pollIntervalId = undefined
    }
    stats = null
    history = []
    logs = null
    processes = []
    containers = []
    pm2 = []
    domains = []
  }

  async function poll() {
    if (!token) return
    try {
      const headers = { Authorization: token }
      const options = { headers }

      const responses = await Promise.all([
        fetch("/api/stats", options),
        fetch("/api/logs", options),
        fetch("/api/processes", options),
        fetch("/api/docker", options),
        fetch("/api/pm2", options),
        fetch("/api/domains", options),
      ])

      if (responses.some((response) => response.status === 401)) {
        handleLogout()
        return
      }

      const [statsData, logsData, processData, dockerData, pm2Data, domainData] =
        await Promise.all(responses.map((response) => response.json()))

      push({ stats: statsData, logs: logsData })
      processes = processData
      containers = dockerData
      pm2 = pm2Data
      domains = domainData
    } catch (pollError) {
      console.error("Polling error:", pollError)
    }
  }

  function connect() {
    if (!token) return
    if (es) {
      es.close()
    }
    const source = new EventSource(`/api/stream?token=${token}`)
    es = source

    source.onopen = () => {
      live = true
    }
    source.onerror = (event) => {
      console.error("SSE Error:", event)
      live = false
      source.close()
      setTimeout(connect, 3000)
    }
    source.onmessage = (event) => {
      try {
        push(JSON.parse(event.data))
      } catch {
        return
      }
    }
  }

  function initDashboard() {
    if (!token) return
    connect()
    poll()
    if (pollIntervalId) clearInterval(pollIntervalId)
    pollIntervalId = window.setInterval(poll, 3000)
  }

  onMount(() => {
    applyTheme()
    if (token) {
      initDashboard()
    }
  })

  onDestroy(() => {
    if (es) {
      es.close()
    }
    if (pollIntervalId) {
      clearInterval(pollIntervalId)
    }
  })

  afterUpdate(() => {
    if (autoScroll && logEndRef) {
      logEndRef.scrollIntoView({ behavior: "smooth" })
    }
  })

  async function handleAction(service: string, action: string) {
    if (!confirm(`Are you sure you want to ${action} ${service}?`)) return

    try {
      const response = await fetch("/api/control", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ service, action }),
      })

      if (response.ok) {
        alert("Done!")
      } else if (response.status === 401) {
        handleLogout()
      } else {
        alert("Failed")
      }
    } catch {
      alert("Error")
    }
  }

  async function handlePM2Action(name: string, action: string) {
    if (!confirm(`Are you sure you want to ${action} ${name}?`)) return

    try {
      const response = await fetch("/api/pm2/control", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ name, action }),
      })

      if (response.ok) {
        alert("Done!")
      } else if (response.status === 401) {
        handleLogout()
      } else {
        alert("Failed")
      }
    } catch {
      alert("Error")
    }
  }

  async function handleDeleteDomain() {
    if (!domainDelete) return

    domainDeleteLoading = true

    try {
      const response = await fetch("/api/domains/delete", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          domain: domainDelete.domain,
          delete_db: domainDelete.deleteDb,
          delete_root: domainDelete.deleteRoot,
        }),
      })

      const data = await response.json().catch(() => ({}))

      if (response.ok) {
        domains = domains.filter((item) => item.domain !== domainDelete!.domain)
        const deletedDomainName = domainDelete.domain
        domainDelete = null
        alert(data.message || `Deleted ${deletedDomainName}`)
      } else if (response.status === 401) {
        handleLogout()
      } else {
        alert(data.error || "Delete failed")
      }
    } catch {
      alert("Error")
    } finally {
      domainDeleteLoading = false
    }
  }

  async function handleSaveDomainNote() {
    if (!domainNote) return

    domainNoteLoading = true

    try {
      const response = await fetch("/api/domains/note", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          domain: domainNote.domain,
          note: domainNote.note,
        }),
      })

      const data = await response.json().catch(() => ({}))

      if (response.ok) {
        const savedDomainName = domainNote.domain
        const savedNoteContent = domainNote.note.trim()
        domains = domains.map((item) =>
          item.domain === savedDomainName
            ? { ...item, note: savedNoteContent }
            : item
        )
        domainNote = null
        alert("Note saved")
      } else if (response.status === 401) {
        handleLogout()
      } else {
        alert(data.error || "Save note failed")
      }
    } catch {
      alert("Error")
    } finally {
      domainNoteLoading = false
    }
  }

  async function handleScanDomains() {
    domainScanning = true
    try {
      const response = await fetch("/api/domains?scan=true", {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        const data = await response.json()
        domains = data
      } else if (response.status === 401) {
        handleLogout()
      }
    } catch (error) {
      console.error("Scan error:", error)
    } finally {
      domainScanning = false
    }
  }

  // Khai báo Tabs chính mở rộng theo phong cách aaPanel
  const appTabsExtended = [
    { key: "overview", label: "Home", icon: LayoutDashboard, disabled: false, description: "Sys status & system parameters" },
    { key: "domains", label: "Website", icon: Globe, disabled: false, description: "Sites, notes and domain actions" },
    { key: "ftp", label: "FTP", icon: Key, disabled: false, description: "FTP accounts & parameters" },
    { key: "databases", label: "Databases", icon: Database, disabled: false, description: "SQL database management" },
    { key: "docker", label: "Docker", icon: Box, disabled: false, description: "Containers and runtime status" },
    { key: "processes", label: "Monitor", icon: Cpu, disabled: false, description: "Top running processes" },
    { key: "nodes", label: "Nodes (PM2)", icon: Terminal, disabled: false, description: "PM2 applications and actions" },
    { key: "security", label: "Security", icon: ShieldAlert, disabled: false, description: "Firewall & intrusion protection" },
    { key: "files", label: "Files", icon: HardDrive, disabled: false, description: "Web-based file explorer" },
    { key: "logs", label: "Logs", icon: FileText, disabled: false, description: "System and nginx logs" },
    { key: "cron", label: "Cron", icon: Clock, disabled: false, description: "Scheduled task execution" },
    { key: "app-store", label: "App Store", icon: ShoppingBag, disabled: false, description: "Install software & modules" },
    { key: "settings", label: "Settings", icon: Settings, disabled: false, description: "Panel configurations" },
  ]

  $: activeAppTab = appTabsExtended.find((tab) => tab.key === appTab) ?? appTabsExtended[0]

  // Khai báo Log Tabs động cho LogsTab
  $: logTabs = [
    { key: "system", label: "System", icon: Terminal, color: "text-blue-400" },
    ...(logs?.nginx_access || logs?.nginx_error
      ? [
          {
            key: "nginx_access",
            label: "Nginx Access",
            icon: Globe,
            color: "text-emerald-400",
          },
          {
            key: "nginx_error",
            label: "Nginx Error",
            icon: AlertTriangle,
            color: "text-rose-400",
          },
        ]
      : []),
    ...(logs?.nginx_sites?.map((site) => ({
      key: `site:${site.domain}`,
      label: site.domain,
      icon: Globe,
      color: "text-indigo-400",
    })) ?? []),
  ] as LogTabItem[]

  $: currentLog = (() => {
    if (!logs) return null
    if (logTab === "system") return logs.system
    if (logTab === "nginx_access") return logs.nginx_access
    if (logTab === "nginx_error") return logs.nginx_error

    if (logTab.startsWith("site:")) {
      const domain = logTab.replace("site:", "")
      const site = logs.nginx_sites?.find((item) => item.domain === domain)
      return siteTab === "access" ? site?.access : site?.error
    }

    return null
  })()

  // helper setter functions for child components in Svelte
  const setLogTab = (val: string) => { logTab = val }
  const setSiteTab = (val: "access" | "error") => { siteTab = val }
  const setAutoScroll = (val: boolean) => { autoScroll = val }
  
  const setDomainDelete = (val: DomainDeleteState) => { domainDelete = val }
  const setDomainNote = (val: DomainNoteState) => { domainNote = val }
</script>

{#if !token}
  <div class="flex min-h-screen items-center justify-center bg-background p-4 text-foreground">
    <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl">
      <div class="space-y-1 text-center p-6 pb-4">
        <h2 class="text-2xl font-light tracking-tight">AcmaDash Login</h2>
        <p class="text-sm font-light text-muted-foreground">
          Enter your credentials to access the dashboard
        </p>
      </div>
      <div class="px-6 pb-6">
        <form on:submit={handleLogin} class="space-y-4">
          <div class="space-y-2">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-light text-muted-foreground">Username</label>
            <input
              type="text"
              bind:value={username}
              class="w-full rounded-lg border border-border bg-secondary/50 px-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
              placeholder="admin"
              required
            />
          </div>
          <div class="space-y-2">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-light text-muted-foreground">Password</label>
            <input
              type="password"
              bind:value={password}
              class="w-full rounded-lg border border-border bg-secondary/50 px-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
              placeholder="••••••••"
              required
            />
          </div>
          {#if error}
            <p class="text-center text-xs text-rose-400">{error}</p>
          {/if}
          <button
            type="submit"
            disabled={loading}
            class="w-full rounded-lg bg-blue-600 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? "Logging in..." : "Sign In"}
          </button>
        </form>
      </div>
    </div>
  </div>
{:else}
  <div class="min-h-screen bg-background text-foreground">
    <!-- Mobile Sidebar overlay -->
    {#if nav}
      <button 
        type="button" 
        class="fixed inset-0 z-40 bg-black/50 lg:hidden cursor-default border-none outline-none" 
        on:click={() => nav = false} 
      />
    {/if}

    <!-- Mobile Sidebar Drawer -->
    <aside
      class="fixed inset-y-0 left-0 z-50 w-72 border-r border-border bg-card transition-transform duration-300 lg:hidden {nav ? '' : '-translate-x-full'}"
    >
      <div class="flex items-center justify-between border-b border-border p-6">
        <div class="flex items-center gap-2">
          <span class="h-6 w-6 rounded bg-emerald-500 flex items-center justify-center text-white text-xs font-bold font-mono">aa</span>
          <span class="text-sm font-bold text-foreground">AcmaPanel</span>
        </div>
        <button type="button" on:click={() => nav = false}>
          <X size={18} class="text-muted-foreground" />
        </button>
      </div>
      <nav class="space-y-1 p-4 overflow-y-auto max-h-[calc(100vh-80px)]">
        {#each appTabsExtended as tab (tab.key)}
          {#if tab.disabled}
            <div class="flex items-center justify-between rounded-xl px-4 py-2.5 text-xs text-muted-foreground/40 cursor-not-allowed">
              <span class="flex items-center gap-3">
                <svelte:component this={tab.icon} size={15} />
                <span>{tab.label}</span>
              </span>
              <span class="text-[8px] bg-secondary px-1 py-0.2 rounded text-muted-foreground/30 font-mono">SOON</span>
            </div>
          {:else}
            <button
              type="button"
              on:click={() => {
                appTab = tab.key
                nav = false
              }}
              class="flex w-full items-start gap-3 rounded-xl px-4 py-2.5 text-left transition-colors {appTab === tab.key ? 'bg-primary text-primary-foreground shadow-sm' : 'text-muted-foreground hover:bg-secondary/40 hover:text-foreground'}"
            >
              <svelte:component this={tab.icon} size={15} class="mt-0.5 shrink-0" />
              <span>
                <span class="block text-xs font-medium">{tab.label}</span>
              </span>
            </button>
          {/if}
        {/each}
        <div class="mt-4 border-t border-border pt-4">
          <button
            type="button"
            on:click={handleLogout}
            class="flex w-full items-center gap-3 rounded-lg px-4 py-2.5 text-xs font-light text-rose-400 transition-colors hover:bg-rose-400/10"
          >
            Logout
          </button>
        </div>
      </nav>
    </aside>

    <!-- Main Outer Container -->
    <div class="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(16,185,129,0.06),_transparent_35%),linear-gradient(180deg,rgba(255,255,255,0.01),transparent_20%)]">
      <div class="mx-auto max-w-[1600px] px-4 py-6 sm:px-6 sm:py-8 lg:flex lg:gap-6 lg:px-8">
        
        <!-- Desktop Sidebar -->
        <aside class="hidden w-64 shrink-0 lg:block">
          <div class="sticky top-6 rounded-2xl border border-border bg-card/95 backdrop-blur overflow-hidden">
            <div class="p-6 pb-4">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <span class="h-6 w-6 rounded bg-emerald-500 flex items-center justify-center text-white text-xs font-bold font-mono">aa</span>
                  <span class="text-sm font-bold text-foreground">AcmaPanel</span>
                </div>
                <span class="rounded bg-emerald-500/10 text-emerald-500 px-2 py-0.5 text-[10px] font-semibold">0</span>
              </div>
            </div>
            
            <div class="px-4 pb-6 space-y-4">
              <hr class="border-border mx-2" />
              <nav class="space-y-0.5 max-h-[calc(100vh-220px)] overflow-y-auto">
                {#each appTabsExtended as tab (tab.key)}
                  {#if tab.disabled}
                    <div class="flex items-center justify-between rounded-xl px-4 py-2 text-xs text-muted-foreground/40 cursor-not-allowed">
                      <span class="flex items-center gap-3">
                        <svelte:component this={tab.icon} size={15} />
                        <span>{tab.label}</span>
                      </span>
                      <span class="text-[8px] bg-secondary px-1.5 py-0.2 rounded text-muted-foreground/30 font-mono">SOON</span>
                    </div>
                  {:else}
                    <button
                      type="button"
                      on:click={() => appTab = tab.key}
                      class="w-full rounded-xl px-4 py-2 text-left transition-colors {appTab === tab.key ? 'bg-primary text-primary-foreground shadow-sm font-medium' : 'text-muted-foreground hover:bg-secondary/40 hover:text-foreground'}"
                    >
                      <span class="flex items-center gap-3">
                        <svelte:component this={tab.icon} size={15} class="shrink-0" />
                        <span class="text-xs">{tab.label}</span>
                      </span>
                    </button>
                  {/if}
                {/each}
              </nav>
              <hr class="border-border mx-2" />
              <button 
                type="button"
                class="w-full inline-flex h-9 items-center justify-center rounded-lg border border-border bg-transparent px-4 text-xs font-light hover:bg-secondary transition-colors" 
                on:click={handleLogout}
              >
                Logout
              </button>
            </div>
        </aside>

        <!-- Main Content Area -->
        <main class="min-w-0 flex-1 space-y-8">
          <header class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between border-b border-border pb-4">
            <!-- Left Side of Header -->
            <div class="flex items-center gap-4">
              <!-- Mobile menu button -->
              <button
                type="button"
                class="inline-flex h-9 w-9 items-center justify-center rounded-lg hover:bg-secondary lg:hidden"
                on:click={() => nav = true}
              >
                <Menu size={18} />
              </button>
              
              <!-- User Profile & OS pill -->
              <div class="flex items-center gap-3">
                <div class="flex items-center gap-2">
                  <div class="h-7 w-7 rounded-full bg-secondary flex items-center justify-center text-xs font-semibold text-foreground border border-border">
                    U
                  </div>
                  <span class="text-xs font-semibold text-foreground">admin</span>
                </div>
                
                <!-- Divider -->
                <span class="text-muted-foreground/30 text-xs">|</span>
                
                <!-- OS Pill -->
                <div class="flex items-center gap-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 px-3 py-1 text-[11px] font-medium text-emerald-500">
                  <span>{stats?.platform ?? "linux"}</span>
                  <span class="opacity-55 mx-1">Up Time:</span>
                  <span class="tabular-nums font-semibold">{stats ? formatUptime(stats.uptime) : "--"}</span>
                </div>
              </div>
            </div>
            
            <!-- Right Side of Header -->
            <div class="flex flex-wrap items-center gap-2 text-xs font-light text-muted-foreground">
              <!-- PRO Badge -->
              <span class="inline-flex items-center rounded-md bg-amber-500 px-2 py-0.5 text-[10px] font-bold text-white uppercase tracking-wider">
                PRO
              </span>
              
              <!-- Version -->
              <span class="text-xs text-muted-foreground/80 font-mono pr-2">v{VERSION}</span>

              <!-- Theme Dropdown & Dark/Light Toggle -->
              <select 
                bind:value={currentTheme}
                class="rounded-lg border border-border bg-card px-2.5 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-emerald-500"
              >
                <option value="aapanel">aaPanel Green</option>
                <option value="slate">Slate Onyx</option>
                <option value="violet">Aurora Violet</option>
                <option value="forest">Nordic Forest</option>
                <option value="abyss">Oceanic Abyss</option>
                <option value="amber">Sunset Amber</option>
              </select>

              <button 
                type="button"
                on:click={() => isDark = !isDark}
                class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-card text-foreground hover:bg-secondary"
                title="Toggle Light/Dark Mode"
              >
                {#if isDark}
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-sun"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>
                {:else}
                  <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-moon"><path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/></svg>
                {/if}
              </button>

              <!-- aaPanel Action Buttons -->
              <button 
                type="button" 
                class="inline-flex h-8 items-center gap-1.5 rounded-lg border border-border bg-card px-3 text-xs text-rose-500 hover:bg-rose-500/10 transition-colors" 
                on:click={() => {
                  if (confirm("Are you sure you want to restart the dashboard backend process?")) {
                    handleAction("vps-dashboard", "restart")
                  }
                }}
              >
                <Power size={12} />
                <span>Restart</span>
              </button>
            </div>
          </header>

          <!-- Mobile Tabs Select -->
          <div class="flex items-center rounded-lg border border-border bg-card p-0.5 lg:hidden overflow-x-auto">
            {#each appTabsExtended as tab}
              {#if !tab.disabled}
                <button
                  type="button"
                  on:click={() => appTab = tab.key}
                  class="flex-1 min-w-[70px] rounded-md py-1.5 text-center text-xs font-normal transition-all {appTab === tab.key ? 'bg-primary text-primary-foreground shadow-sm' : 'text-muted-foreground'}"
                >
                  {tab.label}
                </button>
              {/if}
            {/each}
          </div>

          <!-- Active tab container -->
          <div>
            {#if appTab === "overview"}
              <OverviewTab 
                {stats} 
                {history} 
                {handleAction} 
                domainsCount={domains.length}
                containersCount={containers.length}
                pm2Count={pm2.length}
                processesCount={processes.length}
                switchTab={(tabKey) => appTab = tabKey}
              />
            {:else}
              {#if appTab === "processes"}
                <ProcessesTab {processes} />
              {:else if appTab === "docker"}
                <DockerTab {containers} />
              {:else if appTab === "nodes"}
                <NodesTab {pm2} {handlePM2Action} {formatUptime} />
              {:else if appTab === "domains"}
                <DomainsTab
                  {token}
                  {domains}
                  {setDomainDelete}
                  {setDomainNote}
                  onScan={handleScanDomains}
                  scanning={domainScanning}
                  onRefresh={poll}
                />
              {:else if appTab === "logs"}
                <LogsTab
                  {logTabs}
                  {logTab}
                  {setLogTab}
                  {siteTab}
                  {setSiteTab}
                  {currentLog}
                  {live}
                  {autoScroll}
                  {setAutoScroll}
                  bind:logEndRef
                />
              {:else if appTab === "security"}
                <SecurityTab {token} />
              {:else if appTab === "files"}
                <FilesTab {token} />
              {:else if appTab === "databases"}
                <DatabasesTab {token} />
              {:else if appTab === "app-store"}
                <AppStoreTab {token} />
              {:else if appTab === "ftp"}
                <FtpTab {token} />
              {:else if appTab === "cron"}
                <CronTab {token} />
              {:else if appTab === "settings"}
                <SettingsTab {token} />
              {/if}
            {/if}
          </div>
        </main>
      </div>
    </div>

    <!-- Modals: Domain Delete Confirmation -->
    {#if domainDelete}
      <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
        <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl">
          <div class="border-b border-border px-6 py-4">
            <div class="flex items-start justify-between gap-4">
              <div>
                <h3 class="text-base font-medium text-foreground">Delete domain</h3>
                <p class="mt-1 text-sm text-muted-foreground">{domainDelete.domain}</p>
              </div>
              <button
                type="button"
                on:click={() => !domainDeleteLoading && (domainDelete = null)}
                class="text-muted-foreground transition-colors hover:text-foreground"
              >
                <X size={16} />
              </button>
            </div>
          </div>

          <div class="space-y-4 px-6 py-5">
            <p class="text-sm text-muted-foreground">
              This action removes the domain config. Optional cleanup can also remove the
              database and root folder.
            </p>

            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="flex items-center gap-3 rounded-xl border border-border bg-secondary/20 px-4 py-3 cursor-pointer">
              <input
                type="checkbox"
                bind:checked={domainDelete.deleteDb}
                class="h-4 w-4 rounded border-zinc-700 bg-zinc-800"
              />
              <span class="text-sm text-foreground">Delete database</span>
            </label>

            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="flex items-center gap-3 rounded-xl border border-border bg-secondary/20 px-4 py-3 cursor-pointer">
              <input
                type="checkbox"
                bind:checked={domainDelete.deleteRoot}
                class="h-4 w-4 rounded border-zinc-700 bg-zinc-800"
              />
              <span class="text-sm text-foreground">Delete root folder</span>
            </label>
          </div>

          <div class="flex items-center justify-end gap-3 border-t border-border px-6 py-4">
            <button
              type="button"
              on:click={() => domainDelete = null}
              disabled={domainDeleteLoading}
              class="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="button"
              on:click={handleDeleteDomain}
              disabled={domainDeleteLoading}
              class="rounded-lg bg-rose-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-rose-700 disabled:opacity-50"
            >
              {domainDeleteLoading ? "Deleting..." : "Delete domain"}
            </button>
          </div>
        </div>
      </div>
    {/if}

    <!-- Modals: Domain Edit Note -->
    {#if domainNote}
      <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
        <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl">
          <div class="border-b border-border px-6 py-4">
            <div class="flex items-start justify-between gap-4">
              <div>
                <h3 class="text-base font-medium text-foreground">Edit note</h3>
                <p class="mt-1 text-sm text-muted-foreground">{domainNote.domain}</p>
              </div>
              <button
                type="button"
                on:click={() => !domainNoteLoading && (domainNote = null)}
                class="text-muted-foreground transition-colors hover:text-foreground"
              >
                <X size={16} />
              </button>
            </div>
          </div>

          <div class="space-y-4 px-6 py-5">
            <div class="space-y-2">
              <!-- svelte-ignore a11y-label-has-associated-control -->
              <label class="text-xs font-light text-muted-foreground">Note details</label>
              <textarea
                bind:value={domainNote.note}
                rows={4}
                maxlength={500}
                class="w-full rounded-lg border border-border bg-secondary/50 px-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
                placeholder="Enter domain note/annotation here..."
              />
            </div>
          </div>

          <div class="flex items-center justify-end gap-3 border-t border-border px-6 py-4">
            <button
              type="button"
              on:click={() => domainNote = null}
              disabled={domainNoteLoading}
              class="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              type="button"
              on:click={handleSaveDomainNote}
              disabled={domainNoteLoading}
              class="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
            >
              {domainNoteLoading ? "Saving..." : "Save note"}
            </button>
          </div>
        </div>
      </div>
    {/if}
  </div>
{/if}
