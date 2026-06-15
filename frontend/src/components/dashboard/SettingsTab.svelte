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
    Shield,
    QrCode,
    CheckCircle2,
    XCircle,
    Loader2,
    X
  } from "lucide-svelte"
  import { toast } from "../../lib/toast"

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

  // 2FA state
  let twoFAEnabled = false
  let twoFALoading = false
  let showTwoFAModal = false
  let twoFASecret = ""
  let twoFAQRCode = ""
  let twoFACode = ""
  let twoFAError = ""
  let twoFAVerifyLoading = false

  let showDisable2FAModal = false
  let disable2FACode = ""
  let disable2FAError = ""
  let disable2FALoading = false

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

  async function fetch2FAStatus() {
    try {
      const response = await fetch("/api/settings/2fa/status", {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        const data = await response.json()
        twoFAEnabled = data.enabled
      }
    } catch (err) {
      console.error(err)
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

  async function handleInitEnable2FA() {
    twoFALoading = true
    twoFAError = ""
    twoFACode = ""
    try {
      const response = await fetch("/api/settings/2fa/generate", {
        method: "POST",
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        const data = await response.json()
        twoFASecret = data.secret
        twoFAQRCode = data.qr_code
        showTwoFAModal = true
      } else {
        toast.error("Error", "Failed to generate 2FA key.")
      }
    } catch {
      toast.error("Error", "Connection error")
    } finally {
      twoFALoading = false
    }
  }

  async function handleEnable2FA(e: Event) {
    e.preventDefault()
    if (!twoFACode.trim()) return
    twoFAVerifyLoading = true
    twoFAError = ""
    try {
      const response = await fetch("/api/settings/2fa/enable", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          secret: twoFASecret,
          code: twoFACode.trim(),
        }),
      })
      if (response.ok) {
        showTwoFAModal = false
        twoFAEnabled = true
        toast.success("2FA Enabled", "Two-Factor Authentication is now active.")
        fetch2FAStatus()
      } else {
        const data = await response.json().catch(() => ({}))
        twoFAError = data.error || "Mã xác thực không đúng."
      }
    } catch {
      twoFAError = "Connection error"
    } finally {
      twoFAVerifyLoading = false
    }
  }

  function openDisable2FAModal() {
    disable2FACode = ""
    disable2FAError = ""
    showDisable2FAModal = true
  }

  async function handleDisable2FA(e: Event) {
    e.preventDefault()
    if (!disable2FACode.trim()) return
    disable2FALoading = true
    disable2FAError = ""
    try {
      const response = await fetch("/api/settings/2fa/disable", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          code: disable2FACode.trim(),
        }),
      })
      if (response.ok) {
        showDisable2FAModal = false
        twoFAEnabled = false
        toast.success("2FA Disabled", "Two-Factor Authentication has been disabled.")
        fetch2FAStatus()
      } else {
        const data = await response.json().catch(() => ({}))
        disable2FAError = data.error || "Mã xác thực không đúng. Không thể tắt 2FA."
      }
    } catch {
      disable2FAError = "Connection error"
    } finally {
      disable2FALoading = false
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
    fetch2FAStatus()
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
      <p class="text-xs text-muted-foreground mt-0.5">Configure dashboard authentication settings, system profile, 2FA, and runtime configurations.</p>
    </div>
    <button 
      on:click={() => { fetchSettings(); fetch2FAStatus(); }}
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
    <!-- Left Column (Credentials & 2FA) -->
    <div class="lg:col-span-2 space-y-6">
      <!-- Card: Authentication Credentials -->
      <div class="rounded-2xl border border-border bg-card p-6 space-y-4">
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

      <!-- Card: 2FA Authentication -->
      <div class="rounded-2xl border border-border bg-card p-6 space-y-4">
        <div class="border-b border-border pb-3 flex items-center justify-between">
          <div class="flex items-center gap-2">
            <QrCode size={16} class="text-primary" />
            <h3 class="text-sm font-bold text-foreground">Two-Factor Authentication (2FA)</h3>
          </div>
          
          <span class="inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[10px] font-semibold tracking-wide uppercase {twoFAEnabled ? 'bg-emerald-500/10 text-emerald-500' : 'bg-secondary text-muted-foreground'}">
            {#if twoFAEnabled}
              Active
            {:else}
              Disabled
            {/if}
          </span>
        </div>

        <p class="text-xs text-muted-foreground leading-relaxed">
          Enhance your server security by adding an extra layer of protection. When enabled, you will be prompted to enter a 6-digit TOTP verification code from an authenticator app (such as Google Authenticator, Authy, or Microsoft Authenticator) upon signing in.
        </p>

        <div class="flex items-center gap-3 pt-2">
          {#if twoFAEnabled}
            <button
              type="button"
              on:click={openDisable2FAModal}
              class="rounded-lg border border-rose-500/20 text-rose-500 hover:bg-rose-500/10 px-4 py-2 text-xs font-semibold transition-colors"
            >
              Disable 2FA Protection
            </button>
          {:else}
            <button
              type="button"
              on:click={handleInitEnable2FA}
              disabled={twoFALoading}
              class="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-xs font-semibold text-white px-4 py-2 transition-colors disabled:opacity-50"
            >
              {#if twoFALoading}
                <Loader2 size={12} class="animate-spin" />
                Generating Key...
              {:else}
                <QrCode size={12} />
                Enable 2FA Protection
              {/if}
            </button>
          {/if}
        </div>
      </div>
    </div>

    <!-- Card: System & Panel Status -->
    <div class="rounded-2xl border border-border bg-card p-6 space-y-4 h-fit">
      <div class="border-b border-border pb-3 flex items-center gap-2">
        <Cpu size={16} class="text-primary" />
        <h3 class="text-sm font-bold text-foreground">System & Panel Profile</h3>
      </div>

      <div class="space-y-3.5 text-xs text-muted-foreground">
        <!-- Panel version -->
        <div class="flex justify-between items-center py-1 border-b border-border/50">
          <span>Panel Version</span>
          <span class="font-bold text-foreground font-mono bg-secondary/40 px-1.5 py-0.5 rounded">v{sysInfo.version || "4.0.0"}</span>
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

  <!-- Enable 2FA Modal -->
  {#if showTwoFAModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl overflow-hidden">
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <h3 class="text-sm font-bold text-foreground">Configure Authenticator 2FA</h3>
          <button on:click={() => showTwoFAModal = false} class="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>

        <form on:submit={handleEnable2FA} class="space-y-4 p-6">
          {#if twoFAError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20">
              {twoFAError}
            </div>
          {/if}

          <div class="space-y-3 flex flex-col items-center text-center">
            <p class="text-xs text-muted-foreground">
              Scan the QR code below with your authenticator app (Google Authenticator, Authy, Microsoft Authenticator) or enter the manual key.
            </p>
            
            <!-- QR Code Base64 Data URI -->
            {#if twoFAQRCode}
              <div class="p-2.5 bg-white rounded-xl border border-border">
                <img src={twoFAQRCode} alt="TOTP QR Code" class="w-48 h-48 select-none" />
              </div>
            {/if}

            <div class="bg-secondary/40 border border-border/60 rounded px-3 py-1.5 w-full text-center">
              <span class="text-[9px] uppercase tracking-wider text-muted-foreground block">Manual Secret Key</span>
              <span class="text-xs font-mono font-bold text-foreground select-all">{twoFASecret}</span>
            </div>
          </div>

          <div class="space-y-1.5 pt-2">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground">6-Digit Verification Code</label>
            <input 
              type="text" 
              bind:value={twoFACode}
              placeholder="e.g. 123456"
              maxlength="6"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-center font-mono tracking-widest text-lg font-bold"
              required
            />
          </div>

          <div class="flex items-center justify-end gap-3 pt-4 border-t border-border">
            <button 
              type="button" 
              on:click={() => showTwoFAModal = false}
              class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors"
            >
              Cancel
            </button>
            <button 
              type="submit" 
              disabled={twoFAVerifyLoading}
              class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50"
            >
              {#if twoFAVerifyLoading}
                <Loader2 size={12} class="animate-spin" />
                Verifying...
              {:else}
                Verify & Enable
              {/if}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- Disable 2FA Modal -->
  {#if showDisable2FAModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-sm rounded-2xl border border-border bg-card shadow-2xl overflow-hidden">
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <h3 class="text-sm font-bold text-foreground">Disable 2FA Protection</h3>
          <button on:click={() => showDisable2FAModal = false} class="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>

        <form on:submit={handleDisable2FA} class="space-y-4 p-6">
          {#if disable2FAError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20">
              {disable2FAError}
            </div>
          {/if}

          <p class="text-xs text-muted-foreground">
            Enter the 6-digit verification code from your authenticator app to confirm disabling Two-Factor Authentication.
          </p>

          <div class="space-y-1.5">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground">Verification Code</label>
            <input 
              type="text" 
              bind:value={disable2FACode}
              placeholder="e.g. 123456"
              maxlength="6"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-center font-mono tracking-widest text-lg font-bold"
              required
            />
          </div>

          <div class="flex items-center justify-end gap-3 pt-4 border-t border-border">
            <button 
              type="button" 
              on:click={() => showDisable2FAModal = false}
              class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors"
            >
              Cancel
            </button>
            <button 
              type="submit" 
              disabled={disable2FALoading}
              class="inline-flex items-center gap-1.5 rounded-lg bg-rose-600 hover:bg-rose-700 text-xs font-semibold text-white px-4 py-2 transition-colors disabled:opacity-50"
            >
              {#if disable2FALoading}
                <Loader2 size={12} class="animate-spin" />
                Disabling...
              {:else}
                Confirm Disable
              {/if}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

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
