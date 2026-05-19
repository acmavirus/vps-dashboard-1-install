<script lang="ts">
  import { onMount } from "svelte"
  import { ShieldAlert, ShieldCheck, Plus, Trash2, Shield, RefreshCw, X } from "lucide-svelte"

  export let token: string | null = null

  interface FirewallRule {
    index: int
    to: string
    action: string
    from: string
  }

  interface FirewallStatus {
    enabled: boolean
    rules: FirewallRule[]
  }

  let status: FirewallStatus = { enabled: false, rules: [] }
  let loading = true
  let error = ""
  let toggleLoading = false

  // Add rule modal / form state
  let showAddModal = false
  let addPort = ""
  let addProtocol = "all" // tcp, udp, all
  let addAction = "allow" // allow, deny
  let addLoading = false
  let addError = ""

  async function fetchStatus() {
    loading = true
    error = ""
    try {
      const response = await fetch("/api/firewall", {
        headers: {
          Authorization: token || "",
        },
      })
      if (response.ok) {
        status = await response.json()
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to load firewall status"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function handleToggle() {
    if (toggleLoading) return
    const nextState = !status.enabled
    const confirmMsg = nextState 
      ? "Are you sure you want to enable the UFW firewall? Ensure SSH port is allowed first!"
      : "Are you sure you want to disable the firewall? This will expose all ports."
    
    if (!confirm(confirmMsg)) return

    toggleLoading = true
    try {
      const response = await fetch("/api/firewall/toggle", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ enabled: nextState }),
      })
      if (response.ok) {
        await fetchStatus()
      } else {
        const errData = await response.json().catch(() => ({}))
        alert(errData.error || "Failed to toggle firewall state")
      }
    } catch {
      alert("Network error occurred")
    } finally {
      toggleLoading = false
    }
  }

  async function handleAddRule(e: Event) {
    e.preventDefault()
    addError = ""

    if (!addPort.trim()) {
      addError = "Port cannot be empty"
      return
    }

    // Clean port format validation
    const portRegex = /^[0-9:-]+$/
    if (!portRegex.test(addPort)) {
      addError = "Port must be a number or a range (e.g. 80, 8000:8010)"
      return
    }

    addLoading = true
    try {
      const response = await fetch("/api/firewall/rules", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          port: addPort.trim(),
          protocol: addProtocol,
          action: addAction,
        }),
      })

      if (response.ok) {
        showAddModal = false
        addPort = ""
        addProtocol = "all"
        addAction = "allow"
        await fetchStatus()
      } else {
        const errData = await response.json().catch(() => ({}))
        addError = errData.error || "Failed to add firewall rule"
      }
    } catch {
      addError = "Connection error"
    } finally {
      addLoading = false
    }
  }

  async function handleDeleteRule(index: number, toPort: string) {
    if (!confirm(`Are you sure you want to delete the firewall rule for '${toPort}' (Index ${index})?`)) {
      return
    }

    try {
      const response = await fetch(`/api/firewall/rules/${index}`, {
        method: "DELETE",
        headers: {
          Authorization: token || "",
        },
      })
      if (response.ok) {
        await fetchStatus()
      } else {
        const errData = await response.json().catch(() => ({}))
        alert(errData.error || "Failed to delete rule")
      }
    } catch {
      alert("Network error")
    }
  }

  onMount(() => {
    fetchStatus()
  })
</script>

<div class="space-y-6">
  <!-- Title Header -->
  <div class="flex items-center justify-between border-b border-border pb-4">
    <div>
      <h2 class="text-lg font-bold text-foreground flex items-center gap-2">
        <Shield size={18} class="text-primary" />
        Firewall & Port Security (UFW)
      </h2>
      <p class="text-xs text-muted-foreground mt-0.5">Manage server open ports and network security rules.</p>
    </div>
    <button 
      on:click={fetchStatus}
      disabled={loading}
      class="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-border bg-card text-muted-foreground hover:bg-secondary transition-colors"
      title="Refresh rules"
    >
      <RefreshCw size={14} class={loading ? "animate-spin" : ""} />
    </button>
  </div>

  <!-- Status Card & Toggle -->
  <div class="rounded-2xl border border-border bg-card p-6 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
    <div class="flex items-start gap-4">
      <div class="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl {status.enabled ? 'bg-emerald-500/10 text-emerald-500' : 'bg-rose-500/10 text-rose-500'}">
        {#if status.enabled}
          <ShieldCheck size={24} />
        {:else}
          <ShieldAlert size={24} />
        {/if}
      </div>
      <div>
        <div class="flex items-center gap-2">
          <h3 class="text-sm font-bold text-foreground">Firewall Status</h3>
          <span class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-semibold {status.enabled ? 'bg-emerald-500/10 text-emerald-500' : 'bg-rose-500/10 text-rose-500'}">
            {status.enabled ? "Active" : "Inactive"}
          </span>
        </div>
        <p class="text-xs text-muted-foreground mt-1">
          {status.enabled 
            ? "Your server firewall is active. Only allowed ports are accessible." 
            : "WARNING: Firewall is disabled. All ports are open to the internet."}
        </p>
      </div>
    </div>
    <div class="flex items-center gap-3">
      <button
        on:click={handleToggle}
        disabled={toggleLoading}
        class="inline-flex h-9 items-center justify-center rounded-xl px-4 text-xs font-semibold shadow-sm transition-colors border {status.enabled ? 'border-rose-500/20 text-rose-500 hover:bg-rose-500/10' : 'border-emerald-500/20 text-emerald-500 hover:bg-emerald-500/10'} disabled:opacity-50"
      >
        {#if toggleLoading}
          Processing...
        {:else}
          {status.enabled ? "Disable Firewall" : "Enable Firewall"}
        {/if}
      </button>
    </div>
  </div>

  <!-- Rules Table Block -->
  <div class="rounded-2xl border border-border bg-card overflow-hidden">
    <div class="flex items-center justify-between border-b border-border px-6 py-4">
      <h3 class="text-sm font-bold text-foreground">Active Firewall Rules</h3>
      <button 
        on:click={() => showAddModal = true}
        class="inline-flex h-8 items-center gap-1.5 rounded-lg bg-primary px-3 text-xs font-semibold text-primary-foreground shadow transition-colors hover:opacity-90"
      >
        <Plus size={12} />
        Add Rule
      </button>
    </div>

    {#if loading}
      <div class="flex flex-col items-center justify-center py-12 text-muted-foreground space-y-2">
        <RefreshCw size={24} class="animate-spin text-primary" />
        <span class="text-xs">Loading rules...</span>
      </div>
    {:else if error}
      <div class="flex flex-col items-center justify-center py-12 text-center px-4">
        <p class="text-xs text-rose-500 font-semibold">{error}</p>
        <button on:click={fetchStatus} class="mt-3 text-xs text-primary underline">Retry</button>
      </div>
    {:else if !status.enabled}
      <div class="flex flex-col items-center justify-center py-12 text-center px-6 space-y-2">
        <ShieldAlert size={36} class="text-muted-foreground/40" />
        <p class="text-sm font-semibold text-foreground">Firewall is not enabled</p>
        <p class="text-xs text-muted-foreground max-w-sm">
          Please enable the firewall using the toggle switch above to view and manage security rules.
        </p>
      </div>
    {:else if status.rules.length === 0}
      <div class="flex flex-col items-center justify-center py-12 text-center px-6 space-y-2">
        <ShieldCheck size={36} class="text-emerald-500/40" />
        <p class="text-sm font-semibold text-foreground">No custom rules added</p>
        <p class="text-xs text-muted-foreground max-w-sm">
          All traffic might be allowed/blocked by default settings. Add a rule to customize.
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs border-collapse">
          <thead>
            <tr class="border-b border-border bg-secondary/20 text-muted-foreground font-semibold">
              <th class="px-6 py-3">Index</th>
              <th class="px-6 py-3">To (Port/Proto)</th>
              <th class="px-6 py-3">Action</th>
              <th class="px-6 py-3">From</th>
              <th class="px-6 py-3 text-right">Operations</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each status.rules as rule (rule.index)}
              <tr class="hover:bg-secondary/10 transition-colors">
                <td class="px-6 py-3.5 font-mono text-muted-foreground">[{rule.index}]</td>
                <td class="px-6 py-3.5 font-semibold text-foreground">{rule.to}</td>
                <td class="px-6 py-3.5">
                  <span class="inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider {rule.action.includes('ALLOW') ? 'bg-emerald-500/10 text-emerald-500' : 'bg-rose-500/10 text-rose-500'}">
                    {rule.action}
                  </span>
                </td>
                <td class="px-6 py-3.5 text-muted-foreground">{rule.from}</td>
                <td class="px-6 py-3.5 text-right">
                  <button 
                    on:click={() => handleDeleteRule(rule.index, rule.to)}
                    class="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-rose-500 transition-all"
                    title="Delete rule"
                  >
                    <Trash2 size={13} />
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <!-- Add Rule Dialog Modal -->
  {#if showAddModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl overflow-hidden">
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <h3 class="text-sm font-bold text-foreground">Add Firewall Rule</h3>
          <button 
            on:click={() => showAddModal = false}
            class="text-muted-foreground hover:text-foreground transition-colors"
          >
            <X size={16} />
          </button>
        </div>

        <form on:submit={handleAddRule} class="space-y-4 p-6">
          {#if addError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 font-semibold border border-rose-500/20">
              {addError}
            </div>
          {/if}

          <!-- Port input -->
          <div class="space-y-2">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground block">Port / Port Range</label>
            <input 
              type="text" 
              bind:value={addPort}
              placeholder="e.g. 80, 8080, or 8000:8010"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
              required
            />
          </div>

          <!-- Protocol select -->
          <div class="space-y-2">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground block">Protocol</label>
            <select 
              bind:value={addProtocol}
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="all">ALL</option>
              <option value="tcp">TCP</option>
              <option value="udp">UDP</option>
            </select>
          </div>

          <!-- Action select -->
          <div class="space-y-2">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground block">Action</label>
            <select 
              bind:value={addAction}
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
            >
              <option value="allow">ALLOW</option>
              <option value="deny">DENY</option>
            </select>
          </div>

          <!-- Submit Buttons -->
          <div class="flex items-center justify-end gap-3 pt-4 border-t border-border">
            <button 
              type="button" 
              on:click={() => showAddModal = false}
              class="rounded-lg border border-border px-4 py-2.5 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors"
            >
              Cancel
            </button>
            <button 
              type="submit" 
              disabled={addLoading}
              class="rounded-lg bg-primary px-4 py-2.5 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-opacity disabled:opacity-50"
            >
              {addLoading ? "Adding..." : "Add Rule"}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</div>
