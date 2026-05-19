<script lang="ts">
  import { onMount } from "svelte"
  import { 
    ShoppingBag, 
    Download, 
    Trash2, 
    RefreshCw, 
    X, 
    CheckCircle2, 
    Play, 
    AlertTriangle, 
    Cpu, 
    Globe, 
    Database, 
    Layers, 
    Terminal, 
    ExternalLink, 
    Info
  } from "lucide-svelte"

  export let token: string | null = null

  interface StoreApp {
    id: string
    name: string
    description: string
    category: string
    default_port: string
    image: string
    status: string // "not_installed", "running", "stopped"
    domain?: string
  }

  let apps: StoreApp[] = []
  let loading = true
  let error = ""
  let activeCategory = "All"

  // Modal installation
  let showInstallModal = false
  let selectedApp: StoreApp | null = null
  let appPort = ""
  let appPassword = ""
  let appDomain = ""
  let installLoading = false
  let installError = ""

  // Uninstalling state
  let uninstallLoading = false
  let uninstallAppId = ""

  const categories = ["All", "Web Proxy", "Database", "Database GUI", "CMS"]

  async function fetchApps() {
    loading = true
    error = ""
    try {
      const response = await fetch("/api/apps", {
        headers: { Authorization: token || "" }
      })
      if (response.ok) {
        apps = await response.json()
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to load App Store"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  function openInstallModal(app: StoreApp) {
    selectedApp = app
    appPort = app.default_port
    appPassword = app.id.includes("db") ? "secure_pass_" + Math.random().toString(36).substring(2, 8) : ""
    appDomain = ""
    installError = ""
    showInstallModal = true
  }

  async function handleInstall() {
    if (!selectedApp) return
    installLoading = true
    installError = ""
    try {
      const response = await fetch("/api/apps/install", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || ""
        },
        body: JSON.stringify({
          id: selectedApp.id,
          port: appPort,
          password: appPassword,
          domain: appDomain.trim()
        })
      })

      if (response.ok) {
        showInstallModal = false
        await fetchApps()
      } else {
        const errData = await response.json().catch(() => ({}))
        installError = errData.error || "Installation failed"
      }
    } catch {
      installError = "Network connection failed"
    } finally {
      installLoading = false
    }
  }

  async function handleUninstall(appId: string, appName: string) {
    if (!confirm(`Are you sure you want to uninstall and completely remove container for '${appName}'? Any unsaved container data will be lost.`)) return

    uninstallAppId = appId
    uninstallLoading = true
    try {
      const response = await fetch("/api/apps/uninstall", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || ""
        },
        body: JSON.stringify({ id: appId })
      })

      if (response.ok) {
        await fetchApps()
      } else {
        const errData = await response.json().catch(() => ({}))
        alert(errData.error || "Failed to uninstall app")
      }
    } catch {
      alert("Network error")
    } finally {
      uninstallLoading = false
      uninstallAppId = ""
    }
  }

  // Get matching icon for category
  function getCategoryIcon(cat: string) {
    switch (cat) {
      case "Web Proxy": return Globe
      case "Database": return Database
      case "Database GUI": return Layers
      case "CMS": return Terminal
      default: return Cpu
    }
  }

  onMount(() => {
    fetchApps()
  })

  $: filteredApps = activeCategory === "All" 
    ? apps 
    : apps.filter(app => app.category === activeCategory)
</script>

<div class="space-y-6">
  <!-- Header Title -->
  <div class="flex items-center justify-between border-b border-border pb-4">
    <div>
      <h2 class="text-lg font-bold text-foreground flex items-center gap-2">
        <ShoppingBag size={18} class="text-primary" />
        Docker App Store
      </h2>
      <p class="text-xs text-muted-foreground mt-0.5">Deploy popular database engines, reverse proxies, and CMS instances instantly via Docker.</p>
    </div>
    <div class="flex items-center gap-2">
      <button 
        on:click={fetchApps}
        disabled={loading}
        class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-card text-muted-foreground hover:bg-secondary transition-colors"
      >
        <RefreshCw size={14} class={loading ? "animate-spin" : ""} />
      </button>
    </div>
  </div>

  <!-- Categories Filter -->
  <div class="flex items-center gap-1.5 overflow-x-auto pb-1">
    {#each categories as category}
      <button
        on:click={() => activeCategory = category}
        class="h-8 px-4 rounded-xl text-xs font-semibold border transition-all whitespace-nowrap
          {activeCategory === category 
            ? 'bg-primary border-primary text-primary-foreground shadow-sm' 
            : 'bg-card border-border text-muted-foreground hover:bg-secondary hover:text-foreground'}"
      >
        {category}
      </button>
    {/each}
  </div>

  <!-- Loading / Error states -->
  {#if error}
    <div class="rounded-xl bg-rose-500/10 p-4 text-xs text-rose-500 border border-rose-500/20 flex items-start gap-2.5">
      <AlertTriangle size={15} class="shrink-0 mt-0.5" />
      <div>
        <p class="font-semibold">Failed to Load Store</p>
        <p class="mt-0.5">{error}</p>
      </div>
    </div>
  {/if}

  {#if loading && apps.length === 0}
    <div class="flex flex-col items-center justify-center py-20 text-muted-foreground space-y-2">
      <RefreshCw size={24} class="animate-spin text-primary" />
      <span class="text-xs">Fetching Docker registry data...</span>
    </div>
  {:else if filteredApps.length === 0}
    <div class="flex flex-col items-center justify-center py-20 text-center px-6 space-y-2">
      <ShoppingBag size={36} class="text-muted-foreground/30" />
      <p class="text-sm font-semibold text-foreground">No Apps Found</p>
      <p class="text-xs text-muted-foreground">There are no applications available in this category.</p>
    </div>
  {:else}
    <!-- Grid Apps -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
      {#each filteredApps as app (app.id)}
        <div class="rounded-2xl border border-border bg-card p-5 flex flex-col justify-between hover:shadow-md transition-shadow relative overflow-hidden">
          <div class="space-y-4">
            <!-- Header of card -->
            <div class="flex items-start justify-between">
              <div class="flex items-center gap-3">
                <div class="h-10 w-10 rounded-xl bg-primary/10 flex items-center justify-center text-primary">
                  <svelte:component this={getCategoryIcon(app.category)} size={20} />
                </div>
                <div>
                  <h3 class="text-xs font-bold text-foreground">{app.name}</h3>
                  <span class="inline-flex rounded-full bg-secondary/80 px-2 py-0.5 text-[9px] font-semibold text-muted-foreground mt-0.5">
                    {app.category}
                  </span>
                </div>
              </div>

              <!-- Status Badge -->
              {#if app.status === "running"}
                <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2.5 py-0.5 text-[9px] font-semibold text-emerald-500">
                  <span class="h-1 w-1 rounded-full bg-emerald-500 animate-pulse"></span>
                  Active
                </span>
              {:else if app.status === "stopped"}
                <span class="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2.5 py-0.5 text-[9px] font-semibold text-amber-500">
                  <span class="h-1 w-1 rounded-full bg-amber-500"></span>
                  Stopped
                </span>
              {/if}
            </div>

            <!-- Description -->
            <p class="text-xs text-muted-foreground leading-relaxed min-h-[40px]">
              {app.description}
            </p>

            <!-- Domain proxy badge if configured -->
            {#if app.status === "running" && app.domain}
              <div class="flex items-center justify-between gap-1.5 bg-primary/5 rounded-lg px-2.5 py-1.5 border border-primary/15">
                <div class="flex items-center gap-1.5 truncate">
                  <Globe size={11} class="text-primary animate-pulse shrink-0" />
                  <span class="text-[10px] font-bold text-primary truncate">{app.domain}</span>
                </div>
                <a 
                  href={`http://${app.domain}`} 
                  target="_blank" 
                  rel="noreferrer" 
                  class="text-[10px] font-semibold text-primary hover:underline flex items-center gap-0.5 shrink-0"
                >
                  Visit <ExternalLink size={10} />
                </a>
              </div>
            {/if}

            <!-- Image Tag Info -->
            <div class="flex items-center gap-1 bg-secondary/30 rounded-lg px-2.5 py-1.5 text-[10px] text-muted-foreground font-mono truncate">
              <Info size={11} class="shrink-0" />
              <span>{app.image}</span>
            </div>
          </div>

          <!-- Bottom Actions -->
          <div class="flex items-center justify-between border-t border-border pt-4 mt-5">
            <span class="text-[10px] text-muted-foreground">Default Port: <span class="font-bold text-foreground font-mono">{app.default_port}</span></span>
            
            <div class="flex items-center gap-1.5">
              {#if app.status === "not_installed"}
                <button
                  on:click={() => openInstallModal(app)}
                  class="inline-flex h-7 items-center gap-1 rounded-lg bg-primary px-3 text-[10px] font-bold text-primary-foreground hover:opacity-90 transition-opacity"
                >
                  <Download size={10} />
                  Install
                </button>
              {:else}
                <button
                  on:click={() => handleUninstall(app.id, app.name)}
                  disabled={uninstallLoading && uninstallAppId === app.id}
                  class="inline-flex h-7 items-center gap-1 rounded-lg border border-rose-500/20 bg-rose-500/5 px-3 text-[10px] font-bold text-rose-500 hover:bg-rose-500/10 transition-colors disabled:opacity-50"
                >
                  <Trash2 size={10} />
                  {uninstallLoading && uninstallAppId === app.id ? "Removing..." : "Uninstall"}
                </button>
              {/if}
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Synergy Docker Note -->
  <div class="rounded-xl border border-border bg-card p-4 flex items-start gap-3">
    <CheckCircle2 size={15} class="text-primary shrink-0 mt-0.5" />
    <div class="text-[11px] text-muted-foreground space-y-1">
      <p class="font-semibold text-foreground">Integrated Docker Ecosystem:</p>
      <p>Applications installed via the App Store run as official, lightweight Docker containers.</p>
      <p>Once deployed, you can monitor resource statistics (CPU/RAM usage), view logs, restart, or pause containers inside the main <strong>Docker</strong> tab.</p>
    </div>
  </div>

  <!-- Install App Modal -->
  {#if showInstallModal && selectedApp}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl overflow-hidden">
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <div>
            <h3 class="text-sm font-bold text-foreground">Install {selectedApp.name}</h3>
            <p class="text-[10px] text-muted-foreground mt-0.5">Configure deployment parameters for the container.</p>
          </div>
          <button on:click={() => showInstallModal = false} class="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>

        <form on:submit|preventDefault={handleInstall} class="space-y-4 p-6">
          {#if installError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20">
              {installError}
            </div>
          {/if}

          <!-- WordPress Auto Integration Notice -->
          {#if selectedApp.id === "wordpress-app"}
            <div class="rounded-xl border border-primary/20 bg-primary/5 p-3.5 space-y-1">
              <p class="text-[11px] font-bold text-foreground flex items-center gap-1.5">
                <Database size={12} class="text-primary" />
                Auto SQL DB & User Setup
              </p>
              <p class="text-[10px] text-muted-foreground leading-relaxed">
                WordPress requires a database. We will automatically create a secure database, username, and password on your MySQL host server, then pass them directly as environment variables to the container.
              </p>
            </div>
          {/if}

          <!-- Port mapping configuration -->
          <div class="space-y-1.5">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground">Host Port Mapping</label>
            <input 
              type="text" 
              bind:value={appPort}
              placeholder={selectedApp.default_port}
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
            <p class="text-[10px] text-muted-foreground">Specify the port on your VPS host that will map to this application.</p>
          </div>

          <!-- Domain Name configuration (Optional) -->
          <div class="space-y-1.5">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground flex items-center justify-between">
              <span>Domain Proxy (Optional)</span>
              {#if selectedApp.id === "wordpress-app"}
                <span class="text-[9px] text-primary font-bold">(Highly Recommended)</span>
              {/if}
            </label>
            <input 
              type="text" 
              bind:value={appDomain}
              placeholder="e.g. wordpress.yourdomain.com"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
            />
            <p class="text-[10px] text-muted-foreground">Map a custom domain name to this container port with automated Nginx reverse proxy and SSL setup.</p>
          </div>

          <!-- Password field (only for Databases) -->
          {#if selectedApp.id.includes("db") && selectedApp.id !== "wordpress-app"}
            <div class="space-y-1.5">
              <!-- svelte-ignore a11y-label-has-associated-control -->
              <label class="text-xs font-semibold text-muted-foreground flex items-center justify-between">
                <span>Root Password</span>
                <span class="text-[9px] text-rose-500 font-semibold">(Required)</span>
              </label>
              <input 
                type="text" 
                bind:value={appPassword}
                placeholder="Enter password"
                class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                required
              />
              <p class="text-[10px] text-muted-foreground">Set the root administrator password for the database.</p>
            </div>
          {/if}

          <!-- Spinner warning about pulls -->
          {#if installLoading}
            <div class="flex items-center gap-3 bg-secondary/40 rounded-xl p-3.5 text-xs text-muted-foreground">
              <RefreshCw size={16} class="animate-spin text-primary shrink-0" />
              <div>
                <p class="font-semibold text-foreground">Downloading image & deploying...</p>
                <p class="text-[10px] mt-0.5">This may take up to 60 seconds if Docker needs to pull the image from the registry.</p>
              </div>
            </div>
          {/if}

          <div class="flex items-center justify-end gap-3 pt-4 border-t border-border">
            <button 
              type="button" 
              on:click={() => showInstallModal = false}
              disabled={installLoading}
              class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button 
              type="submit" 
              disabled={installLoading}
              class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50 flex items-center gap-1.5"
            >
              <Play size={10} />
              Deploy Application
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</div>
