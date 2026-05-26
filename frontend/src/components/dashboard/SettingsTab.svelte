<script lang="ts">
  import { onMount } from "svelte"
  import { 
    Settings, 
    User, 
    Lock, 
    RefreshCw, 
    Check, 
    AlertTriangle, 
    Cpu, 
    Layers, 
    Info,
    Shield
  } from "lucide-svelte"

  export let token: string | null = null

  let loading = true
  let saving = false
  let error = ""
  let successMsg = ""

  // Settings state
  let username = ""
  let newPassword = ""
  let confirmPassword = ""

  // System status state
  let sysInfo = {
    username: "",
    version: "",
    go_version: "",
    os: "",
    num_cpu: 0,
    goroutines: 0
  }

  async function fetchSettings() {
    loading = true
    error = ""
    successMsg = ""
    try {
      const response = await fetch("/api/settings", {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        sysInfo = await response.json()
        username = sysInfo.username
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to load dashboard settings"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function handleUpdateSettings(e: Event) {
    e.preventDefault()
    error = ""
    successMsg = ""

    if (!username.trim()) {
      error = "Username cannot be empty"
      return
    }

    if (newPassword) {
      if (newPassword.length < 6) {
        error = "Password must be at least 6 characters"
        return
      }
      if (newPassword !== confirmPassword) {
        error = "Passwords do not match"
        return
      }
    }

    saving = true
    try {
      const response = await fetch("/api/settings/update", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          username: username.trim(),
          password: newPassword || "",
        }),
      })
      if (response.ok) {
        successMsg = "Settings updated successfully! Changes saved to environment configuration."
        newPassword = ""
        confirmPassword = ""
        await fetchSettings()
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to save settings"
      }
    } catch {
      error = "Connection error"
    } finally {
      saving = false
    }
  }

  async function handleRestartPanel() {
    if (!confirm("WARNING: Are you sure you want to restart the dashboard panel? Current connections will temporarily drop for 2-3 seconds while the system service restarts.")) return

    loading = true
    try {
      const response = await fetch("/api/settings/restart", {
        method: "POST",
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        successMsg = "Restart command sent. The page will reload in 4 seconds..."
        setTimeout(() => {
          window.location.reload()
        }, 4000)
      } else {
        error = "Failed to restart panel"
        loading = false
      }
    } catch {
      // Net connection will drop, which is expected
      successMsg = "Panel is restarting. Reloading page..."
      setTimeout(() => {
        window.location.reload()
      }, 4000)
    }
  }

  onMount(() => {
    fetchSettings()
  })
</script>

<div class="space-y-6">
  <!-- Title Header -->
  <div class="flex items-center justify-between border-b border-border pb-4">
    <div>
      <h2 class="text-lg font-bold text-foreground flex items-center gap-2">
        <Settings size={18} class="text-primary" />
        Panel Settings
      </h2>
      <p class="text-xs text-muted-foreground mt-0.5">Configure dashboard authentication settings, system profile and runtime configurations.</p>
    </div>
    <button 
      on:click={fetchSettings}
      disabled={loading}
      class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-card text-muted-foreground hover:bg-secondary transition-colors"
    >
      <RefreshCw size={14} class={loading ? "animate-spin" : ""} />
    </button>
  </div>

  <!-- Alerts -->
  {#if error}
    <div class="rounded-xl bg-rose-500/10 p-3.5 text-xs text-rose-500 border border-rose-500/20 flex items-start gap-2.5">
      <AlertTriangle size={15} class="shrink-0 mt-0.5" />
      <div>
        <p class="font-semibold">Error</p>
        <p class="mt-0.5">{error}</p>
      </div>
    </div>
  {/if}

  {#if successMsg}
    <div class="rounded-xl bg-emerald-500/10 p-3.5 text-xs text-emerald-500 border border-emerald-500/20 flex items-start gap-2.5">
      <Check size={15} class="shrink-0 mt-0.5" />
      <div>
        <p class="font-semibold">Success</p>
        <p class="mt-0.5">{successMsg}</p>
      </div>
    </div>
  {/if}

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Card: Authentication Credentials -->
    <div class="lg:col-span-2 rounded-2xl border border-border bg-card p-6 space-y-4">
      <div class="border-b border-border pb-3 flex items-center gap-2">
        <Shield size={16} class="text-primary" />
        <h3 class="text-sm font-bold text-foreground">Admin Account Credentials</h3>
      </div>

      {#if loading && !username}
        <div class="flex items-center justify-center py-20 text-muted-foreground">
          <RefreshCw size={20} class="animate-spin text-primary mr-2" />
          <span class="text-xs">Loading admin details...</span>
        </div>
      {:else}
        <form on:submit={handleUpdateSettings} class="space-y-4">
          <!-- Username -->
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-muted-foreground flex items-center gap-1.5">
              <User size={13} />
              Admin Username
            </label>
            <input 
              type="text" 
              bind:value={username}
              placeholder="admin"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
          </div>

          <!-- New Password -->
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-muted-foreground flex items-center gap-1.5">
              <Lock size={13} />
              New Password (Leave blank to keep current)
            </label>
            <input 
              type="password" 
              bind:value={newPassword}
              placeholder="••••••••"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
            />
          </div>

          <!-- Confirm Password -->
          {#if newPassword}
            <div class="space-y-1.5">
              <label class="text-xs font-semibold text-muted-foreground flex items-center gap-1.5">
                <Check size={13} />
                Confirm New Password
              </label>
              <input 
                type="password" 
                bind:value={confirmPassword}
                placeholder="••••••••"
                class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                required
              />
            </div>
          {/if}

          <div class="flex justify-end pt-2">
            <button 
              type="submit" 
              disabled={saving}
              class="rounded-lg bg-primary px-4 py-2.5 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-opacity disabled:opacity-50"
            >
              {saving ? "Saving Changes..." : "Save Account Settings"}
            </button>
          </div>
        </form>
      {/if}
    </div>

    <!-- Card: System & Panel Status -->
    <div class="rounded-2xl border border-border bg-card p-6 space-y-4">
      <div class="border-b border-border pb-3 flex items-center gap-2">
        <Cpu size={16} class="text-primary" />
        <h3 class="text-sm font-bold text-foreground">System & Panel Profile</h3>
      </div>

      <div class="space-y-3.5 text-xs text-muted-foreground">
        <!-- Panel version -->
        <div class="flex justify-between items-center py-1 border-b border-border/50">
          <span>Panel Version</span>
          <span class="font-bold text-foreground font-mono bg-secondary/40 px-1.5 py-0.5 rounded">v{sysInfo.version || "2.2.3"}</span>
        </div>

        <!-- Port -->
        <div class="flex justify-between items-center py-1 border-b border-border/50">
          <span>Active Port</span>
          <span class="font-bold text-foreground font-mono">8900</span>
        </div>

        <!-- OS -->
        <div class="flex justify-between items-center py-1 border-b border-border/50">
          <span>OS Architecture</span>
          <span class="font-bold text-foreground font-mono">{sysInfo.os || "linux/amd64"}</span>
        </div>

        <!-- CPU Count -->
        <div class="flex justify-between items-center py-1 border-b border-border/50">
          <span>Available CPU Cores</span>
          <span class="font-bold text-foreground font-mono">{sysInfo.num_cpu || "4"} cores</span>
        </div>

        <!-- Go version -->
        <div class="flex justify-between items-center py-1 border-b border-border/50">
          <span>Go Runtime version</span>
          <span class="font-bold text-foreground font-mono">{sysInfo.go_version || "go1.22"}</span>
        </div>

        <!-- Goroutines -->
        <div class="flex justify-between items-center py-1 border-b border-border/50">
          <span>Active Goroutines</span>
          <span class="font-bold text-foreground font-mono">{sysInfo.goroutines || "0"}</span>
        </div>

        <!-- Restart action -->
        <div class="pt-4 space-y-2">
          <p class="text-[10px] text-muted-foreground leading-normal">
            Restarting the dashboard terminates the current active backend instance and prompts the supervisor/systemd manager to spin up a fresh load.
          </p>
          <button 
            on:click={handleRestartPanel}
            disabled={loading}
            class="w-full rounded-lg border border-rose-500/20 text-rose-500 hover:bg-rose-500/10 py-2.5 text-xs font-semibold transition-colors disabled:opacity-50 flex items-center justify-center gap-1.5"
          >
            <RefreshCw size={12} class={loading ? "animate-spin" : ""} />
            Restart Dashboard Panel
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Info Box -->
  <div class="rounded-xl border border-border bg-card p-4 flex items-start gap-3">
    <Info size={15} class="text-primary shrink-0 mt-0.5" />
    <div class="text-[11px] text-muted-foreground space-y-1">
      <p class="font-semibold text-foreground">Aesthetic & Security Notice:</p>
      <p>Updating credentials triggers direct encryption writes to <span class="font-mono bg-secondary/40 px-1 py-0.5 rounded text-foreground">.env</span> file. Make sure backups are active if managing custom config files.</p>
      <p>This panel uses secure authentication tokens. Changing your username or password in Settings will write the environment variables, keeping your current login session active.</p>
    </div>
  </div>
</div>
