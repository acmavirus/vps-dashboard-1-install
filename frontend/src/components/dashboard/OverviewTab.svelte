<script lang="ts">
  import { 
    Cpu, 
    MemoryStick, 
    HardDrive, 
    Wifi, 
    RotateCcw, 
    Square, 
    Play, 
    Globe, 
    ChevronRight, 
    Box, 
    Terminal, 
    ShieldAlert 
  } from "lucide-svelte"
  import type { Stats } from "./types"

  export let stats: Stats | null = null
  export let history: { t: string; v: number }[] = []
  export let handleAction: (service: string, action: string) => void
  export let domainsCount = 0
  export let containersCount = 0
  export let pm2Count = 0
  export let processesCount = 0
  export let switchTab: (tab: string) => void

  // Network speed states
  let lastSent = 0
  let lastRecv = 0
  let uploadSpeed = 0
  let downloadSpeed = 0

  $: {
    if (stats) {
      if (lastSent > 0) {
        uploadSpeed = Math.max((stats.net_sent - lastSent) / 3, 0)
      }
      if (lastRecv > 0) {
        downloadSpeed = Math.max((stats.net_recv - lastRecv) / 3, 0)
      }
      lastSent = stats.net_sent
      lastRecv = stats.net_recv
    }
  }

  function formatSpeed(bytesPerSec: number) {
    if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(1)} B/s`
    const kb = bytesPerSec / 1024
    if (kb < 1024) return `${kb.toFixed(2)} KB/s`
    const mb = kb / 1024
    return `${mb.toFixed(2)} MB/s`
  }

  function formatBytes(bytes: number) {
    if (!bytes) return "0 B"
    const gbVal = bytes / 1073741824
    if (gbVal < 1024) return `${gbVal.toFixed(2)} GB`
    const tbVal = gbVal / 1024
    return `${tbVal.toFixed(2)} TB`
  }

  const formatMB = (bytes: number) => {
    return bytes ? `${Math.floor(bytes / 1048576)}` : "0"
  }

  // Dimensions of SVG Area Chart
  const width = 800
  const height = 180
  const padding = { top: 10, right: 10, bottom: 25, left: 35 }

  $: points = history.map((d, i) => {
    const x = padding.left + (i / Math.max(history.length - 1, 1)) * (width - padding.left - padding.right)
    const y = height - padding.bottom - (d.v / 100) * (height - padding.top - padding.bottom)
    return { x, y, t: d.t, v: d.v }
  })

  $: pathData = points.length > 0 
    ? `M ${points[0].x} ${points[0].y} ` + points.slice(1).map(p => `L ${p.x} ${p.y}`).join(' ') 
    : ''

  $: areaData = points.length > 0 
    ? `${pathData} L ${points[points.length-1].x} ${height - padding.bottom} L ${points[0].x} ${height - padding.bottom} Z`
    : ''

  let activeTooltip: typeof points[0] | null = null
  let tooltipX = 0
  let tooltipY = 0

  function handleMouseMove(e: MouseEvent) {
    if (points.length === 0) return
    const rect = (e.currentTarget as SVGElement).getBoundingClientRect()
    const mouseX = ((e.clientX - rect.left) / rect.width) * width
    
    let closest = points[0]
    let minDiff = Math.abs(points[0].x - mouseX)
    
    for (const p of points) {
      const diff = Math.abs(p.x - mouseX)
      if (diff < minDiff) {
        minDiff = diff
        closest = p
      }
    }
    
    activeTooltip = closest
    tooltipX = (closest.x / width) * rect.width
    tooltipY = (closest.y / height) * rect.height
  }

  function handleMouseLeave() {
    activeTooltip = null
  }

  // Calculate Load status as cosmetic formula: (CPU% * 0.7) + 1.2
  $: loadStatus = stats ? Math.min((stats.cpu * 0.7) + 1.2, 100) : 0
</script>

<div class="space-y-6">
  <!-- Top Row: Sys Status & Disk -->
  <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
    <!-- Sys Status Card -->
    <div class="lg:col-span-2 rounded-2xl border border-border bg-card p-6 flex flex-col justify-between">
      <div class="mb-4">
        <h3 class="text-sm font-bold text-foreground">Sys Status</h3>
      </div>
      <div class="grid grid-cols-3 gap-4 py-2">
        <!-- Dial 1: Load Status -->
        <div class="flex flex-col items-center text-center space-y-3">
          <div class="relative flex items-center justify-center w-24 h-24 sm:w-28 sm:h-28">
            <svg class="w-full h-full transform -rotate-90">
              <circle
                cx="56"
                cy="56"
                r="42"
                stroke="hsl(var(--secondary))"
                stroke-width="6"
                fill="transparent"
              />
              <circle
                cx="56"
                cy="56"
                r="42"
                stroke="#10b981"
                stroke-width="6"
                stroke-dasharray={2 * Math.PI * 42}
                stroke-dashoffset={2 * Math.PI * 42 * (1 - loadStatus / 100)}
                stroke-linecap="round"
                fill="transparent"
              />
            </svg>
            <span class="absolute text-base font-bold tabular-nums text-foreground">{loadStatus.toFixed(1)}%</span>
          </div>
          <div>
            <p class="text-xs font-semibold text-foreground">Smooth operation</p>
            <p class="text-[10px] text-muted-foreground">Load status</p>
          </div>
        </div>

        <!-- Dial 2: CPU Usage -->
        <div class="flex flex-col items-center text-center space-y-3">
          <div class="relative flex items-center justify-center w-24 h-24 sm:w-28 sm:h-28">
            <svg class="w-full h-full transform -rotate-90">
              <circle
                cx="56"
                cy="56"
                r="42"
                stroke="hsl(var(--secondary))"
                stroke-width="6"
                fill="transparent"
              />
              <circle
                cx="56"
                cy="56"
                r="42"
                stroke="#3b82f6"
                stroke-width="6"
                stroke-dasharray={2 * Math.PI * 42}
                stroke-dashoffset={2 * Math.PI * 42 * (1 - (stats?.cpu ?? 0) / 100)}
                stroke-linecap="round"
                fill="transparent"
              />
            </svg>
            <span class="absolute text-base font-bold tabular-nums text-foreground">{stats?.cpu !== undefined ? stats.cpu.toFixed(1) : "--"}%</span>
          </div>
          <div>
            <p class="text-xs font-semibold text-foreground">1 Core(s)</p>
            <p class="text-[10px] text-muted-foreground">CPU usage</p>
          </div>
        </div>

        <!-- Dial 3: RAM Usage -->
        <div class="flex flex-col items-center text-center space-y-3">
          <div class="relative flex items-center justify-center w-24 h-24 sm:w-28 sm:h-28">
            <svg class="w-full h-full transform -rotate-90">
              <circle
                cx="56"
                cy="56"
                r="42"
                stroke="hsl(var(--secondary))"
                stroke-width="6"
                fill="transparent"
              />
              <circle
                cx="56"
                cy="56"
                r="42"
                stroke="#f59e0b"
                stroke-width="6"
                stroke-dasharray={2 * Math.PI * 42}
                stroke-dashoffset={2 * Math.PI * 42 * (1 - (stats?.ram ?? 0) / 100)}
                stroke-linecap="round"
                fill="transparent"
              />
            </svg>
            <span class="absolute text-base font-bold tabular-nums text-foreground">{stats?.ram !== undefined ? stats.ram.toFixed(1) : "--"}%</span>
          </div>
          <div>
            <p class="text-xs font-semibold text-foreground">
              {#if stats}
                {formatMB(stats.ram_used)} / {formatMB(stats.ram_total)}(MB)
              {:else}
                --
              {/if}
            </p>
            <p class="text-[10px] text-muted-foreground">RAM usage</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Disk Card -->
    <div class="rounded-2xl border border-border bg-card p-6 flex flex-col justify-between">
      <div class="mb-4">
        <h3 class="text-sm font-bold text-foreground">Disk</h3>
      </div>
      <div class="flex items-center justify-between gap-4 py-1">
        <!-- Disk Progress Bars -->
        <div class="flex-1 space-y-4">
          <div>
            <div class="flex items-center justify-between text-xs mb-1">
              <span class="font-semibold text-foreground">/</span>
              <span class="text-emerald-500 font-bold">{stats?.disk !== undefined ? stats.disk.toFixed(0) : "0"}%</span>
            </div>
            <div class="h-2 w-full rounded-full bg-secondary overflow-hidden">
              <div class="h-full rounded-full bg-emerald-500" style="width: {stats?.disk ?? 0}%" />
            </div>
            <p class="text-[10px] text-muted-foreground mt-1">
              {#if stats}
                {formatBytes(stats.disk_used)} / {formatBytes(stats.disk_total)}
              {:else}
                --
              {/if}
            </p>
          </div>
        </div>

        <!-- Disk Concentric SVG Ring -->
        <div class="shrink-0 relative w-24 h-24 flex items-center justify-center">
          <svg class="w-full h-full transform -rotate-90">
            <circle
              cx="48"
              cy="48"
              r="38"
              stroke="hsl(var(--secondary))"
              stroke-width="5"
              fill="transparent"
            />
            <circle
              cx="48"
              cy="48"
              r="38"
              stroke="#10b981"
              stroke-width="5"
              stroke-dasharray={2 * Math.PI * 38}
              stroke-dashoffset={2 * Math.PI * 38 * (1 - (stats?.disk ?? 0) / 100)}
              stroke-linecap="round"
              fill="transparent"
            />
            <circle
              cx="48"
              cy="48"
              r="28"
              stroke="hsl(var(--secondary))"
              stroke-width="3"
              fill="transparent"
            />
          </svg>
          <div class="absolute text-center">
            <span class="text-xs font-bold block text-emerald-500">{stats?.disk !== undefined ? stats.disk.toFixed(0) : "0"}%</span>
            <span class="text-[9px] text-muted-foreground block font-mono">Used</span>
          </div>
        </div>
      </div>
    </div>
  </div>

  <!-- Middle Row: Overview Stats -->
  <div class="space-y-3">
    <h3 class="text-sm font-bold text-foreground">Overview</h3>
    <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <!-- Sites -->
      <div class="flex items-center justify-between rounded-2xl border border-border bg-card p-5">
        <div class="space-y-1">
          <p class="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">Site</p>
          <p class="text-2xl font-bold tracking-tight text-foreground">{domainsCount}</p>
        </div>
        <button 
          type="button"
          on:click={() => switchTab("domains")}
          class="flex h-7 w-7 items-center justify-center rounded-lg bg-secondary text-muted-foreground hover:bg-primary hover:text-primary-foreground transition-colors"
        >
          <ChevronRight size={16} />
        </button>
      </div>

      <!-- Docker -->
      <div class="flex items-center justify-between rounded-2xl border border-border bg-card p-5">
        <div class="space-y-1">
          <p class="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">Docker</p>
          <p class="text-2xl font-bold tracking-tight text-foreground">{containersCount}</p>
        </div>
        <button 
          type="button"
          on:click={() => switchTab("docker")}
          class="flex h-7 w-7 items-center justify-center rounded-lg bg-secondary text-muted-foreground hover:bg-primary hover:text-primary-foreground transition-colors"
        >
          <ChevronRight size={16} />
        </button>
      </div>

      <!-- Nodes PM2 -->
      <div class="flex items-center justify-between rounded-2xl border border-border bg-card p-5">
        <div class="space-y-1">
          <p class="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">Nodes (PM2)</p>
          <p class="text-2xl font-bold tracking-tight text-foreground">{pm2Count}</p>
        </div>
        <button 
          type="button"
          on:click={() => switchTab("nodes")}
          class="flex h-7 w-7 items-center justify-center rounded-lg bg-secondary text-muted-foreground hover:bg-primary hover:text-primary-foreground transition-colors"
        >
          <ChevronRight size={16} />
        </button>
      </div>

      <!-- System Processes -->
      <div class="flex items-center justify-between rounded-2xl border border-border bg-card p-5">
        <div class="space-y-1">
          <p class="text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">Monitor</p>
          <p class="text-2xl font-bold tracking-tight text-foreground">{processesCount}</p>
        </div>
        <button 
          type="button"
          on:click={() => switchTab("processes")}
          class="flex h-7 w-7 items-center justify-center rounded-lg bg-secondary text-muted-foreground hover:bg-primary hover:text-primary-foreground transition-colors"
        >
          <ChevronRight size={16} />
        </button>
      </div>
    </div>
  </div>

  <!-- Bottom Row: Software & Network Chart -->
  <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
    <!-- Software Card (Left) -->
    <div class="rounded-2xl border border-border bg-card p-6 flex flex-col justify-between">
      <div class="mb-4">
        <h3 class="text-sm font-bold text-foreground">Software</h3>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <!-- Nginx -->
        <div class="flex items-center justify-between rounded-xl border border-border bg-secondary/10 p-4">
          <div class="flex items-center gap-3">
            <span class="flex h-9 w-9 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-500 text-sm font-bold">N</span>
            <div>
              <p class="text-xs font-semibold text-foreground">Nginx 1.22.1</p>
              <p class="text-[10px] text-muted-foreground">Web Server</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <span class="h-2.5 w-2.5 rounded-full bg-emerald-500" />
            <div class="flex gap-1">
              <button on:click={() => handleAction("nginx", "restart")} class="p-1 hover:text-blue-500 transition-colors" title="Restart">
                <RotateCcw size={12} />
              </button>
              <button on:click={() => handleAction("nginx", "stop")} class="p-1 hover:text-rose-500 transition-colors" title="Stop">
                <Square size={12} />
              </button>
            </div>
          </div>
        </div>

        <!-- PHP 8.3 -->
        <div class="flex items-center justify-between rounded-xl border border-border bg-secondary/10 p-4">
          <div class="flex items-center gap-3">
            <span class="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-400 text-sm font-bold">PHP</span>
            <div>
              <p class="text-xs font-semibold text-foreground">PHP 8.3</p>
              <p class="text-[10px] text-muted-foreground">FPM Service</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <span class="h-2.5 w-2.5 rounded-full bg-emerald-500" />
            <div class="flex gap-1">
              <button on:click={() => handleAction("php8.3", "restart")} class="p-1 hover:text-blue-500 transition-colors" title="Restart">
                <RotateCcw size={12} />
              </button>
              <button on:click={() => handleAction("php8.3", "stop")} class="p-1 hover:text-rose-500 transition-colors" title="Stop">
                <Square size={12} />
              </button>
            </div>
          </div>
        </div>

        <!-- PHP 7.4 -->
        <div class="flex items-center justify-between rounded-xl border border-border bg-secondary/10 p-4">
          <div class="flex items-center gap-3">
            <span class="flex h-9 w-9 items-center justify-center rounded-lg bg-indigo-500/10 text-indigo-400 text-sm font-bold">PHP</span>
            <div>
              <p class="text-xs font-semibold text-foreground">PHP 7.4</p>
              <p class="text-[10px] text-muted-foreground">Legacy FPM</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <span class="h-2.5 w-2.5 rounded-full bg-emerald-500" />
            <div class="flex gap-1">
              <button on:click={() => handleAction("php7.4", "restart")} class="p-1 hover:text-blue-500 transition-colors" title="Restart">
                <RotateCcw size={12} />
              </button>
              <button on:click={() => handleAction("php7.4", "stop")} class="p-1 hover:text-rose-500 transition-colors" title="Stop">
                <Square size={12} />
              </button>
            </div>
          </div>
        </div>

        <!-- MariaDB -->
        <div class="flex items-center justify-between rounded-xl border border-border bg-secondary/10 p-4">
          <div class="flex items-center gap-3">
            <span class="flex h-9 w-9 items-center justify-center rounded-lg bg-amber-500/10 text-amber-500 text-sm font-bold">DB</span>
            <div>
              <p class="text-xs font-semibold text-foreground">MariaDB</p>
              <p class="text-[10px] text-muted-foreground">MySQL Database</p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <span class="h-2.5 w-2.5 rounded-full bg-emerald-500" />
            <div class="flex gap-1">
              <button on:click={() => handleAction("mysql", "restart")} class="p-1 hover:text-blue-500 transition-colors" title="Restart">
                <RotateCcw size={12} />
              </button>
              <button on:click={() => handleAction("mysql", "stop")} class="p-1 hover:text-rose-500 transition-colors" title="Stop">
                <Square size={12} />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Traffic Card (Right) -->
    <div class="rounded-2xl border border-border bg-card p-6 flex flex-col justify-between space-y-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3 text-xs font-bold text-foreground">
          <span class="border-b-2 border-emerald-500 pb-1 cursor-pointer">Traffic</span>
          <span class="text-muted-foreground/60 cursor-not-allowed">Disk IO</span>
        </div>
        <select class="rounded border border-border bg-card px-2 py-0.5 text-[10px] text-foreground focus:outline-none focus:ring-1 focus:ring-emerald-500">
          <option>Net: All</option>
        </select>
      </div>

      <!-- Real-time network speed metrics -->
      <div class="grid grid-cols-4 gap-2 text-center py-1 bg-secondary/20 rounded-xl border border-border/50">
        <div>
          <p class="text-[9px] text-muted-foreground">● Upstream</p>
          <p class="text-xs font-bold text-foreground tabular-nums mt-0.5">{formatSpeed(uploadSpeed)}</p>
        </div>
        <div>
          <p class="text-[9px] text-muted-foreground">● Downstream</p>
          <p class="text-xs font-bold text-foreground tabular-nums mt-0.5">{formatSpeed(downloadSpeed)}</p>
        </div>
        <div>
          <p class="text-[9px] text-muted-foreground">Total sent</p>
          <p class="text-xs font-bold text-foreground tabular-nums mt-0.5">{stats ? formatBytes(stats.net_sent) : "0 GB"}</p>
        </div>
        <div>
          <p class="text-[9px] text-muted-foreground">Total received</p>
          <p class="text-xs font-bold text-foreground tabular-nums mt-0.5">{stats ? formatBytes(stats.net_recv) : "0 GB"}</p>
        </div>
      </div>

      <!-- Native SVG Area Chart styled like aaPanel -->
      <div class="relative h-[120px] pt-2">
        <svg 
          viewBox="0 0 {width} {height}" 
          class="w-full h-full select-none overflow-visible"
          role="img"
          aria-label="Network Traffic History"
          on:mousemove={handleMouseMove}
          on:mouseleave={handleMouseLeave}
        >
          <defs>
            <linearGradient id="trafficFill" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stop-color="#10b981" stop-opacity="0.15" />
              <stop offset="100%" stop-color="#10b981" stop-opacity="0" />
            </linearGradient>
          </defs>
          
          <!-- Y grid lines -->
          {#each [0, 25, 50, 75, 100] as tick}
            {@const y = height - padding.bottom - (tick / 100) * (height - padding.top - padding.bottom)}
            <line x1={padding.left} y1={y} x2={width - padding.right} y2={y} stroke="hsl(var(--border))" stroke-width="0.5" stroke-dasharray="2 3" />
          {/each}

          <!-- X grid line (floor) -->
          <line x1={padding.left} y1={height - padding.bottom} x2={width - padding.right} y2={height - padding.bottom} stroke="hsl(var(--border))" stroke-width="1" />

          <!-- X labels -->
          {#if points.length > 0}
            {#each points.filter((_, idx) => idx % 15 === 0 || idx === points.length - 1) as p}
              <text x={p.x} y={height - padding.bottom + 15} fill="hsl(var(--muted-foreground))" class="text-[9px] font-mono" text-anchor="middle">{p.t}</text>
            {/each}
          {/if}

          <!-- Area and Line graph -->
          {#if points.length > 0}
            <path d={areaData} fill="url(#trafficFill)" />
            <path d={pathData} fill="none" stroke="#10b981" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
            
            {#if activeTooltip}
              <line x1={activeTooltip.x} y1={padding.top} x2={activeTooltip.x} y2={height - padding.bottom} stroke="hsl(var(--border))" stroke-width="0.8" />
              <circle cx={activeTooltip.x} cy={activeTooltip.y} r="4.5" fill="#10b981" stroke="white" stroke-width="1.5" />
            {/if}
          {/if}
        </svg>

        {#if activeTooltip}
          <div 
            class="absolute z-10 pointer-events-none rounded-lg border border-border bg-card px-2.5 py-1.5 text-[10px] shadow-sm transition-all duration-75"
            style="left: {tooltipX}px; top: {tooltipY - 45}px; transform: translate(-50%, -50%);"
          >
            <p class="text-[8px] font-mono text-muted-foreground">{activeTooltip.t}</p>
            <p class="font-semibold text-foreground text-[10px]">Load: <span class="text-emerald-500 tabular-nums">{activeTooltip.v.toFixed(1)}%</span></p>
          </div>
        {/if}
      </div>

      <div class="flex items-center justify-between text-[9px] text-muted-foreground">
        <span>Unit: KB/s</span>
        <span>© aaPanel design emulation</span>
      </div>
    </div>
  </div>
</div>
