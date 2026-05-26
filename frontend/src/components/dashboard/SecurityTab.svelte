<script lang="ts">
  import { onMount } from "svelte"
  import { ShieldAlert, ShieldCheck, Plus, Trash2, Shield, RefreshCw, X, Lock, Unlock, Settings, Activity } from "lucide-svelte"

  export let token: string | null = null

  interface FirewallRule {
    index: number
    to: string
    action: string
    from: string
  }

  interface ListeningPort {
    port: string
    protocol: string
    address: string
    process: string
    pid: string
  }

  interface FirewallStatus {
    enabled: boolean
    logging: string
    default_incoming: string
    default_outgoing: string
    default_routed: string
    rules: FirewallRule[]
    listening_ports: ListeningPort[]
  }

  let status: FirewallStatus = {
    enabled: false,
    logging: "unknown",
    default_incoming: "deny",
    default_outgoing: "allow",
    default_routed: "deny",
    rules: [],
    listening_ports: []
  }

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

  function quickOpenPort(port: string, proto: string) {
    addPort = port
    const cleanProto = proto.toLowerCase()
    if (cleanProto === "tcp" || cleanProto === "udp") {
      addProtocol = cleanProto
    } else {
      addProtocol = "all"
    }
    addAction = "allow"
    showAddModal = true
  }

  function matchRuleToPort(ruleTo: string, portStr: string, protoStr: string): boolean {
    let cleanTo = ruleTo.toLowerCase()
    
    const idxParen = cleanTo.indexOf("(")
    if (idxParen !== -1) {
      cleanTo = cleanTo.substring(0, idxParen).trim()
    }
    
    const idxOn = cleanTo.indexOf(" on ")
    if (idxOn !== -1) {
      cleanTo = cleanTo.substring(0, idxOn).trim()
    }
    
    if (cleanTo === "anywhere") {
      return true
    }
    
    let ruleProto = ""
    const idxSlash = cleanTo.indexOf("/")
    if (idxSlash !== -1) {
      ruleProto = cleanTo.substring(idxSlash + 1).trim()
      cleanTo = cleanTo.substring(0, idxSlash).trim()
    }
    
    if (ruleProto && ruleProto !== "all" && ruleProto !== protoStr.toLowerCase()) {
      return false
    }
    
    const parts = cleanTo.split(",")
    for (let part of parts) {
      part = part.trim()
      if (part === portStr) {
        return true
      }
      
      if (part.includes(":")) {
        const [startStr, endStr] = part.split(":")
        const start = parseInt(startStr, 10)
        const end = parseInt(endStr, 10)
        const current = parseInt(portStr, 10)
        if (!isNaN(start) && !isNaN(end) && !isNaN(current)) {
          if (current >= start && current <= end) {
            return true
          }
        }
      }
    }
    
    return false
  }

  function getExposureStatus(lp: ListeningPort): { status: string; label: string; class: string; icon: any } {
    const isLocal = lp.address.startsWith("127.") || lp.address === "::1" || lp.address === "localhost" || lp.address.includes("127.0.0.53") || lp.address.includes("::1")
    if (isLocal) {
      return { 
        status: "local-only", 
        label: "Local-Only (Secure)", 
        class: "bg-emerald-500/10 text-emerald-500 border border-emerald-500/20", 
        icon: Lock 
      }
    }
    
    if (!status.enabled) {
      return { 
        status: "unprotected", 
        label: "Exposed (Firewall Inactive)", 
        class: "bg-rose-500/10 text-rose-500 border border-rose-500/20 animate-pulse", 
        icon: ShieldAlert 
      }
    }
    
    let hasAllow = false
    let hasDeny = false
    
    for (const rule of status.rules) {
      if (matchRuleToPort(rule.to, lp.port, lp.protocol)) {
        if (rule.action.toLowerCase().includes("allow")) {
          hasAllow = true
        } else if (rule.action.toLowerCase().includes("deny")) {
          hasDeny = true
        }
      }
    }
    
    if (hasAllow) {
      return { 
        status: "allowed", 
        label: "Exposed (Allowed)", 
        class: "bg-amber-500/10 text-amber-500 border border-amber-500/20", 
        icon: Unlock 
      }
    }
    
    if (hasDeny) {
      return { 
        status: "blocked", 
        label: "Blocked (Protected)", 
        class: "bg-emerald-500/10 text-emerald-500 border border-emerald-500/20", 
        icon: ShieldCheck 
      }
    }
    
    const defaultIncoming = status.default_incoming.toLowerCase()
    if (defaultIncoming === "allow") {
      return { 
        status: "allowed-default", 
        label: "Exposed (Default Allow)", 
        class: "bg-amber-500/10 text-amber-500 border border-amber-500/20", 
        icon: Unlock 
      }
    }
    
    return { 
      status: "protected-default", 
      label: "Blocked (Protected by default)", 
      class: "bg-emerald-500/10 text-emerald-500 border border-emerald-500/20", 
      icon: ShieldCheck 
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

  <!-- Stats Grid -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
    <!-- Card 1: Firewall Status -->
    <div class="rounded-2xl border border-border bg-card p-5 flex flex-col justify-between min-h-[130px]">
      <div class="flex items-center justify-between">
        <span class="text-xs font-semibold text-muted-foreground">Firewall Status</span>
        <Shield size={16} class={status.enabled ? "text-emerald-500" : "text-rose-500"} />
      </div>
      <div class="flex items-center gap-2 mt-2">
        <span class="text-xl font-bold text-foreground">{status.enabled ? "Active" : "Inactive"}</span>
        <span class="relative flex h-2 w-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75 {status.enabled ? 'bg-emerald-400' : 'bg-rose-400'}"></span>
          <span class="relative inline-flex rounded-full h-2 w-2 {status.enabled ? 'bg-emerald-500' : 'bg-rose-500'}"></span>
        </span>
      </div>
      <div class="mt-4">
        <button
          on:click={handleToggle}
          disabled={toggleLoading}
          class="w-full inline-flex h-8 items-center justify-center rounded-lg text-xs font-bold transition-all border {status.enabled ? 'border-rose-500/20 text-rose-500 hover:bg-rose-500/10' : 'border-emerald-500/20 text-emerald-500 hover:bg-emerald-500/10'} disabled:opacity-50"
        >
          {toggleLoading ? "Processing..." : status.enabled ? "Disable Firewall" : "Enable Firewall"}
        </button>
      </div>
    </div>

    <!-- Card 2: Default Policies -->
    <div class="rounded-2xl border border-border bg-card p-5 flex flex-col justify-between min-h-[130px]">
      <div class="flex items-center justify-between">
        <span class="text-xs font-semibold text-muted-foreground">Default Policies</span>
        <Settings size={16} class="text-muted-foreground" />
      </div>
      <div class="space-y-2 mt-2">
        <div class="flex items-center justify-between text-xs">
          <span class="text-muted-foreground">Incoming:</span>
          <span class="font-bold uppercase px-1.5 py-0.5 rounded text-[10px] {status.default_incoming === 'allow' ? 'text-amber-500 bg-amber-500/10 border border-amber-500/20' : 'text-emerald-500 bg-emerald-500/10 border border-emerald-500/20'}">{status.default_incoming}</span>
        </div>
        <div class="flex items-center justify-between text-xs">
          <span class="text-muted-foreground">Outgoing:</span>
          <span class="font-bold uppercase px-1.5 py-0.5 rounded text-[10px] {status.default_outgoing === 'allow' ? 'text-emerald-500 bg-emerald-500/10 border border-emerald-500/20' : 'text-rose-500 bg-rose-500/10 border border-rose-500/20'}">{status.default_outgoing}</span>
        </div>
      </div>
    </div>

    <!-- Card 3: Logging Level -->
    <div class="rounded-2xl border border-border bg-card p-5 flex flex-col justify-between min-h-[130px]">
      <div class="flex items-center justify-between">
        <span class="text-xs font-semibold text-muted-foreground">Logging Status</span>
        <Activity size={16} class="text-muted-foreground" />
      </div>
      <div class="mt-2">
        <span class="text-xl font-bold text-foreground capitalize">{status.logging}</span>
      </div>
      <p class="text-[10px] text-muted-foreground mt-2">Syslog level for firewall packet logs.</p>
    </div>

    <!-- Card 4: Summary Analyzer -->
    <div class="rounded-2xl border border-border bg-card p-5 flex flex-col justify-between min-h-[130px]">
      <div class="flex items-center justify-between">
        <span class="text-xs font-semibold text-muted-foreground">UFW Rules</span>
        <ShieldCheck size={16} class="text-muted-foreground" />
      </div>
      <div class="mt-2">
        <span class="text-xl font-bold text-foreground">{status.rules ? status.rules.length : 0} Rules</span>
      </div>
      <p class="text-[10px] text-muted-foreground mt-2">{status.listening_ports ? status.listening_ports.length : 0} network socket bindings analyzed.</p>
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
    {:else if !status.rules || status.rules.length === 0}
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
                  <span class="inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-wider {rule.action.includes('ALLOW') ? 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20' : 'bg-rose-500/10 text-rose-500 border border-rose-500/20'}">
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

  <!-- Listening Services Exposure Analyzer Block -->
  <div class="rounded-2xl border border-border bg-card overflow-hidden">
    <div class="flex items-center justify-between border-b border-border px-6 py-4">
      <div>
        <h3 class="text-sm font-bold text-foreground">Listening Services & Port Analyzer</h3>
        <p class="text-[10px] text-muted-foreground mt-0.5">Real-time list of listening sockets and their UFW exposure status.</p>
      </div>
    </div>

    {#if loading}
      <div class="flex flex-col items-center justify-center py-12 text-muted-foreground space-y-2">
        <RefreshCw size={24} class="animate-spin text-primary" />
        <span class="text-xs">Analyzing listening services...</span>
      </div>
    {:else if !status.listening_ports || status.listening_ports.length === 0}
      <div class="flex flex-col items-center justify-center py-12 text-center px-6 space-y-2">
        <ShieldCheck size={36} class="text-emerald-500/40" />
        <p class="text-sm font-semibold text-foreground">No active listeners detected</p>
        <p class="text-xs text-muted-foreground max-w-sm">
          No TCP or UDP sockets are bound to any interfaces on this system.
        </p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs border-collapse">
          <thead>
            <tr class="border-b border-border bg-secondary/20 text-muted-foreground font-semibold">
              <th class="px-6 py-3">Service / Process</th>
              <th class="px-6 py-3">PID</th>
              <th class="px-6 py-3">Protocol</th>
              <th class="px-6 py-3">Port</th>
              <th class="px-6 py-3">Bind Address</th>
              <th class="px-6 py-3">Firewall Exposure</th>
              <th class="px-6 py-3 text-right">Quick Action</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each status.listening_ports as lp}
              {@const exposure = getExposureStatus(lp)}
              <tr class="hover:bg-secondary/10 transition-colors">
                <td class="px-6 py-3.5 font-semibold text-foreground flex items-center gap-2">
                  <span class="inline-block h-1.5 w-1.5 rounded-full bg-primary"></span>
                  {lp.process}
                </td>
                <td class="px-6 py-3.5 font-mono text-muted-foreground">{lp.pid}</td>
                <td class="px-6 py-3.5 font-mono font-bold uppercase text-muted-foreground">{lp.protocol}</td>
                <td class="px-6 py-3.5 font-mono font-semibold text-foreground">{lp.port}</td>
                <td class="px-6 py-3.5 font-mono text-muted-foreground">{lp.address}</td>
                <td class="px-6 py-3.5">
                  <span class="inline-flex items-center gap-1 rounded px-2 py-0.5 text-[10px] font-semibold {exposure.class}">
                    <svelte:component this={exposure.icon} size={10} />
                    {exposure.label}
                  </span>
                </td>
                <td class="px-6 py-3.5 text-right">
                  {#if (exposure.status.includes('protected') || exposure.status === 'unprotected') && status.enabled}
                    <button 
                      on:click={() => quickOpenPort(lp.port, lp.protocol)}
                      class="inline-flex items-center gap-1 rounded-lg border border-primary/20 bg-primary/5 px-2.5 py-1 text-[10px] font-semibold text-primary hover:bg-primary hover:text-primary-foreground transition-all"
                      title="Open port in firewall"
                    >
                      <Plus size={10} />
                      Open Port
                    </button>
                  {:else}
                    <span class="text-muted-foreground/30 text-[10px]">-</span>
                  {/if}
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
