<script lang="ts">
  import { onMount, onDestroy, afterUpdate } from "svelte"
  import { fade, slide } from "svelte/transition"
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
    Plus,
    Search,
    Bell,
    ChevronDown,
    ChevronLeft,
    Sun,
    Moon,
    Palette
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

  let sidebarCollapsed = localStorage.getItem("sidebar_collapsed") === "true"
  let userMenuOpen = false
  let notificationsOpen = false
  let themeMenuOpen = false

  function toggleSidebar() {
    sidebarCollapsed = !sidebarCollapsed
    localStorage.setItem("sidebar_collapsed", String(sidebarCollapsed))
  }

  function closeAllMenus() {
    userMenuOpen = false
    notificationsOpen = false
    themeMenuOpen = false
  }

  const themesList = [
    { key: "aapanel", label: "aaPanel Green", color: "bg-emerald-500" },
    { key: "slate", label: "Slate Onyx", color: "bg-slate-700" },
    { key: "violet", label: "Aurora Violet", color: "bg-violet-600" },
    { key: "forest", label: "Nordic Forest", color: "bg-green-700" },
    { key: "abyss", label: "Oceanic Abyss", color: "bg-blue-800" },
    { key: "amber", label: "Sunset Amber", color: "bg-amber-600" },
  ]

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

  $: systemNotifications = [
    ...(stats && stats.cpu > 80 ? [{ id: "cpu", type: "error", title: "High CPU Usage", desc: `CPU usage is at ${stats.cpu.toFixed(1)}%`, time: "Just now" }] : []),
    ...(stats && (stats.mem_used / stats.mem_total) * 100 > 90 ? [{ id: "mem", type: "error", title: "High Memory Usage", desc: `Memory is at ${((stats.mem_used / stats.mem_total) * 100).toFixed(1)}%`, time: "Just now" }] : []),
    ...(stats && stats.disk_percent > 85 ? [{ id: "disk", type: "warning", title: "Disk Space Warning", desc: `Disk space used is at ${stats.disk_percent.toFixed(1)}%`, time: "Just now" }] : []),
    ...(containers.some(c => c.State === "exited") ? [{ id: "docker", type: "warning", title: "Container Exited", desc: "One or more Docker containers have exited", time: "1m ago" }] : []),
    ...(pm2.some(p => p.status !== "online") ? [{ id: "pm2", type: "error", title: "Node App Offline", desc: "One or more PM2 apps are offline", time: "2m ago" }] : []),
  ]
  
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

<svelte:window on:click={closeAllMenus} />

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
  <div class="flex h-screen w-screen overflow-hidden bg-background text-foreground">
    <!-- Mobile Sidebar overlay -->
    {#if nav}
      <button 
        type="button" 
        class="fixed inset-0 z-40 bg-black/60 lg:hidden cursor-default border-none outline-none transition-opacity" 
        on:click={() => nav = false} 
      />
    {/if}

    <!-- Mobile Sidebar Drawer -->
    <aside
      class="fixed inset-y-0 left-0 z-50 flex flex-col w-64 border-r border-border bg-card transition-transform duration-300 lg:hidden {nav ? 'translate-x-0' : '-translate-x-full'}"
    >
      <div class="flex items-center justify-between border-b border-border px-5 h-16 shrink-0">
        <div class="flex items-center gap-2">
          <span class="h-8 w-8 rounded bg-primary text-primary-foreground flex items-center justify-center font-bold font-mono text-sm shadow-sm">ap</span>
          <span class="text-sm font-bold tracking-tight text-foreground">AcmaPanel</span>
        </div>
        <button type="button" class="p-1.5 rounded-lg hover:bg-secondary text-muted-foreground transition-colors" on:click={() => nav = false}>
          <X size={16} />
        </button>
      </div>
      <nav class="flex-1 space-y-1 p-3 overflow-y-auto">
        {#each appTabsExtended as tab (tab.key)}
          {#if tab.disabled}
            <div class="flex items-center justify-between rounded-lg px-3 py-2 text-xs text-muted-foreground/40 cursor-not-allowed select-none">
              <span class="flex items-center gap-3">
                <svelte:component this={tab.icon} size={16} />
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
              class="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors {appTab === tab.key ? 'bg-primary/10 text-primary font-medium' : 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'}"
            >
              <svelte:component this={tab.icon} size={16} class="shrink-0" />
              <span class="text-xs">{tab.label}</span>
            </button>
          {/if}
        {/each}
      </nav>
      <div class="p-3 border-t border-border shrink-0">
        <button
          type="button"
          on:click={handleLogout}
          class="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-xs font-medium text-rose-500 hover:bg-rose-500/10 transition-colors"
        >
          <LogOut size={16} />
          <span>Sign Out</span>
        </button>
      </div>
    </aside>

    <!-- Desktop Sidebar -->
    <aside class="hidden lg:flex flex-col h-screen bg-card border-r border-border transition-all duration-300 {sidebarCollapsed ? 'w-16' : 'w-64'} shrink-0 z-30 relative">
      <!-- Brand Area -->
      <div class="flex items-center border-b border-border px-4 h-16 shrink-0 {sidebarCollapsed ? 'justify-center' : 'justify-between'}">
        {#if !sidebarCollapsed}
          <div class="flex items-center gap-2.5" transition:fade={{ duration: 150 }}>
            <span class="h-8 w-8 rounded bg-primary text-primary-foreground flex items-center justify-center font-bold font-mono text-sm shadow-sm shrink-0">ap</span>
            <span class="text-sm font-semibold tracking-tight text-foreground truncate">AcmaPanel</span>
          </div>
        {:else}
          <span class="h-8 w-8 rounded bg-primary text-primary-foreground flex items-center justify-center font-bold font-mono text-sm shadow-sm shrink-0">ap</span>
        {/if}
      </div>

      <!-- Navigation Links -->
      <nav class="flex-1 py-4 px-3 space-y-1 overflow-y-auto">
        {#each appTabsExtended as tab (tab.key)}
          {#if tab.disabled}
            {#if !sidebarCollapsed}
              <div class="flex items-center justify-between rounded-lg px-3 py-2 text-xs text-muted-foreground/40 cursor-not-allowed select-none">
                <span class="flex items-center gap-3">
                  <svelte:component this={tab.icon} size={16} />
                  <span>{tab.label}</span>
                </span>
                <span class="text-[8px] bg-secondary px-1.5 py-0.2 rounded text-muted-foreground/30 font-mono">SOON</span>
              </div>
            {/if}
          {:else}
            <div class="relative group">
              <button
                type="button"
                on:click={() => appTab = tab.key}
                class="flex w-full items-center rounded-lg px-3 py-2 text-left transition-colors {sidebarCollapsed ? 'justify-center' : 'gap-3'} {appTab === tab.key ? 'bg-primary/10 text-primary font-medium' : 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'}"
              >
                <svelte:component this={tab.icon} size={16} class="shrink-0 {appTab === tab.key ? 'text-primary' : ''}" />
                {#if !sidebarCollapsed}
                  <span class="text-xs truncate">{tab.label}</span>
                {/if}
              </button>
              
              <!-- Collapsed Sidebar Tooltip -->
              {#if sidebarCollapsed}
                <div class="absolute left-full top-1/2 -translate-y-1/2 ml-3 px-2.5 py-1.5 bg-popover text-popover-foreground border border-border text-[11px] font-medium rounded shadow-md pointer-events-none opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-50">
                  {tab.label}
                </div>
              {/if}
            </div>
          {/if}
        {/each}
      </nav>

      <!-- Sidebar Collapse Toggler / Footer -->
      <div class="p-3 border-t border-border shrink-0 flex {sidebarCollapsed ? 'justify-center' : 'justify-end'}">
        <button
          type="button"
          on:click={toggleSidebar}
          class="p-2 rounded-lg bg-secondary/60 hover:bg-secondary border border-border text-muted-foreground hover:text-foreground transition-all duration-200"
          title={sidebarCollapsed ? "Expand Sidebar" : "Collapse Sidebar"}
        >
          {#if sidebarCollapsed}
            <ChevronRight size={15} />
          {:else}
            <ChevronLeft size={15} />
          {/if}
        </button>
      </div>
    </aside>

    <!-- Main Content Container -->
    <div class="flex flex-col flex-1 min-w-0 overflow-hidden">
      <!-- Top Header -->
      <header class="h-16 border-b border-border bg-card/95 backdrop-blur flex items-center justify-between px-6 z-20 shrink-0 select-none">
        
        <!-- Left Side: Mobile Menu Toggler + Breadcrumbs -->
        <div class="flex items-center gap-4">
          <button
            type="button"
            class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border hover:bg-secondary lg:hidden text-foreground transition-colors shrink-0"
            on:click|stopPropagation={() => nav = true}
          >
            <Menu size={16} />
          </button>
          
          <div class="flex items-center gap-2 text-xs font-light text-muted-foreground">
            <span class="font-semibold text-foreground text-sm tracking-tight">{activeAppTab.label}</span>
            <span class="opacity-30">/</span>
            <span class="text-[11px] truncate max-w-[180px] hidden sm:inline">{activeAppTab.description}</span>
          </div>
        </div>
        
        <!-- Right Side: Action Badges & Popovers -->
        <div class="flex items-center gap-3">
          
          <!-- Unified Server Status Pill -->
          <div class="hidden md:flex items-center gap-3 bg-secondary/40 border border-border rounded-full px-3 py-1 text-[11px] font-medium text-muted-foreground">
            <div class="flex items-center gap-1.5">
              <span class="capitalize text-foreground font-semibold">{stats?.platform ?? "linux"}</span>
            </div>
            <span class="opacity-30">|</span>
            <div class="flex items-center gap-1">
              <span>Uptime:</span>
              <span class="tabular-nums font-semibold text-foreground">{stats ? formatUptime(stats.uptime) : "--"}</span>
            </div>
            <span class="opacity-30">|</span>
            <div class="flex items-center gap-1.5">
              <span class="relative flex h-2 w-2">
                {#if live}
                  <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                  <span class="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                {:else}
                  <span class="relative inline-flex rounded-full h-2 w-2 bg-amber-500"></span>
                {/if}
              </span>
              <span class="text-[10px] font-semibold uppercase {live ? 'text-emerald-500' : 'text-amber-500'}">
                {live ? 'SSE Live' : 'Polling'}
              </span>
            </div>
          </div>

          <!-- SpotLight mock search input -->
          <div class="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-lg bg-secondary/50 border border-border text-muted-foreground w-44 hover:w-56 transition-all duration-300 text-[11px] cursor-not-allowed select-none">
            <Search size={13} class="shrink-0" />
            <span class="flex-1">Search settings...</span>
            <kbd class="pointer-events-none inline-flex h-4.5 select-none items-center gap-0.5 rounded border border-border bg-muted px-1 font-mono text-[9px] font-medium text-muted-foreground/75 opacity-100">
              <span>⌘</span>K
            </kbd>
          </div>

          <!-- Notifications Bell Dropdown -->
          <div class="relative">
            <button 
              type="button"
              on:click|stopPropagation={() => {
                const prev = notificationsOpen;
                closeAllMenus();
                notificationsOpen = !prev;
              }}
              class="relative inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-card text-foreground hover:bg-secondary transition-colors animate-none"
              title="Notifications"
            >
              <Bell size={15} />
              {#if systemNotifications.length > 0}
                <span class="absolute -top-1 -right-1 flex h-4 w-4 items-center justify-center rounded-full bg-rose-500 text-[8px] font-bold text-white ring-2 ring-card">
                  {systemNotifications.length}
                </span>
              {/if}
            </button>
            
            {#if notificationsOpen}
              <div class="absolute right-0 top-full mt-2 w-80 rounded-lg border border-border bg-card shadow-lg py-1 z-50" transition:fade={{ duration: 100 }}>
                <div class="flex items-center justify-between px-3.5 py-2 border-b border-border">
                  <span class="text-xs font-semibold text-foreground">System Warnings</span>
                  {#if systemNotifications.length > 0}
                    <span class="rounded-full bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
                      {systemNotifications.length} Active
                    </span>
                  {/if}
                </div>
                <div class="max-h-64 overflow-y-auto divide-y divide-border">
                  {#if systemNotifications.length === 0}
                    <div class="flex flex-col items-center justify-center p-6 text-center">
                      <Activity size={24} class="text-muted-foreground/30 mb-2 animate-pulse" />
                      <p class="text-xs font-medium text-foreground">All systems normal</p>
                      <p class="text-[10px] text-muted-foreground mt-0.5">Live monitoring is active.</p>
                    </div>
                  {:else}
                    {#each systemNotifications as notif}
                      <div class="p-3 text-xs flex gap-2.5 items-start hover:bg-secondary/30 transition-colors">
                        <span class="mt-1 flex h-2 w-2 shrink-0 rounded-full {notif.type === 'error' ? 'bg-rose-500' : 'bg-amber-500'}"></span>
                        <div class="space-y-0.5 flex-1 min-w-0">
                          <p class="font-medium text-foreground truncate">{notif.title}</p>
                          <p class="text-[11px] text-muted-foreground leading-normal">{notif.desc}</p>
                          <p class="text-[9px] text-muted-foreground/60 mt-1">{notif.time}</p>
                        </div>
                      </div>
                    {/each}
                  {/if}
                </div>
              </div>
            {/if}
          </div>

          <!-- Custom Theme Selector Popover -->
          <div class="relative">
            <button 
              type="button"
              on:click|stopPropagation={() => {
                const prev = themeMenuOpen;
                closeAllMenus();
                themeMenuOpen = !prev;
              }}
              class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-card text-foreground hover:bg-secondary transition-colors"
              title="Change Theme & Appearance"
            >
              <Palette size={15} />
            </button>
            
            {#if themeMenuOpen}
              <div class="absolute right-0 top-full mt-2 w-52 rounded-lg border border-border bg-card shadow-lg py-1 z-50" transition:fade={{ duration: 100 }}>
                <div class="px-3.5 py-2 border-b border-border text-xs font-semibold text-muted-foreground">Select Theme</div>
                <div class="p-1.5 space-y-0.5">
                  {#each themesList as t}
                    <button
                      type="button"
                      on:click|stopPropagation={() => { currentTheme = t.key; themeMenuOpen = false; }}
                      class="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors hover:bg-secondary {currentTheme === t.key ? 'bg-primary/10 text-primary font-medium' : 'text-foreground'}"
                    >
                      <span class="h-3.5 w-3.5 rounded-full {t.color} border border-border/20 shrink-0"></span>
                      <span class="flex-1">{t.label}</span>
                      {#if currentTheme === t.key}
                        <span class="h-1.5 w-1.5 rounded-full bg-primary shrink-0"></span>
                      {/if}
                    </button>
                  {/each}
                </div>
                
                <div class="border-t border-border p-1.5">
                  <button
                    type="button"
                    on:click|stopPropagation={() => { isDark = !isDark; }}
                    class="flex w-full items-center justify-between rounded-md px-2.5 py-1.5 text-left text-xs text-foreground transition-colors hover:bg-secondary"
                  >
                    <span class="flex items-center gap-2">
                      {#if isDark}
                        <Moon size={13} class="text-muted-foreground" />
                      {:else}
                        <Sun size={13} class="text-muted-foreground" />
                      {/if}
                      <span>Dark Mode</span>
                    </span>
                    <div class="relative inline-flex h-4.5 w-8 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none {isDark ? 'bg-primary' : 'bg-muted'}">
                      <span class="pointer-events-none inline-block h-3.5 w-3.5 transform rounded-full bg-background shadow ring-0 transition duration-200 ease-in-out {isDark ? 'translate-x-3.5' : 'translate-x-0'}"></span>
                    </div>
                  </button>
                </div>
              </div>
            {/if}
          </div>

          <!-- User Profile Dropdown -->
          <div class="relative">
            <button 
              type="button"
              on:click|stopPropagation={() => {
                const prev = userMenuOpen;
                closeAllMenus();
                userMenuOpen = !prev;
              }}
              class="flex items-center gap-2 p-1.5 rounded-lg hover:bg-secondary transition-colors"
              title="User Account"
            >
              <div class="h-7 w-7 rounded-full bg-primary/10 border border-primary/20 text-primary flex items-center justify-center text-xs font-bold shadow-sm shrink-0">
                A
              </div>
              <ChevronDown size={12} class="text-muted-foreground shrink-0 transition-transform duration-200 {userMenuOpen ? 'rotate-180' : ''}" />
            </button>
            
            {#if userMenuOpen}
              <div class="absolute right-0 top-full mt-2 w-56 rounded-lg border border-border bg-card shadow-lg py-1 z-50" transition:fade={{ duration: 100 }}>
                <div class="flex items-center gap-2.5 px-4 py-3 border-b border-border">
                  <div class="h-9 w-9 rounded-full bg-primary/10 border border-primary/20 text-primary flex items-center justify-center font-bold text-sm shadow-sm shrink-0">
                    A
                  </div>
                  <div class="min-w-0">
                    <p class="text-xs font-semibold text-foreground truncate">admin</p>
                    <p class="text-[10px] text-muted-foreground truncate">Administrator</p>
                  </div>
                </div>
                
                <div class="p-1">
                  <!-- PRO Badge label inside dropdown -->
                  <div class="flex items-center justify-between px-3 py-1.5 text-[10px] text-muted-foreground">
                    <span>License Type</span>
                    <span class="rounded bg-amber-500 px-1.5 py-0.2 text-[8px] font-bold text-white tracking-wider">PRO</span>
                  </div>

                  <hr class="border-border my-1" />

                  <button
                    type="button"
                    on:click|stopPropagation={() => {
                      userMenuOpen = false;
                      if (confirm("Are you sure you want to restart the dashboard backend process?")) {
                        handleAction("vps-dashboard", "restart");
                      }
                    }}
                    class="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-xs text-rose-500 hover:bg-rose-500/10 transition-colors font-medium"
                  >
                    <Power size={13} />
                    <span>Restart Panel</span>
                  </button>
                  
                  <button
                    type="button"
                    on:click|stopPropagation={() => { userMenuOpen = false; handleLogout(); }}
                    class="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-xs text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
                  >
                    <LogOut size={13} />
                    <span>Sign Out</span>
                  </button>
                </div>
              </div>
            {/if}
          </div>

        </div>
      </header>

      <!-- Main Layout Content Section -->
      <main class="flex-1 overflow-y-auto bg-secondary/15 p-6 space-y-6 relative">
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
