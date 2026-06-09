<script lang="ts">
  import { onMount } from "svelte"
  import { 
    Clock, 
    Plus, 
    Trash2, 
    RefreshCw, 
    X, 
    Terminal, 
    FileText, 
    Check, 
    AlertTriangle,
    Eye,
    Info
  } from "lucide-svelte"

  export let token: string | null = null

  interface CronJob {
    id: string
    name: string
    schedule: string
    command: string
    status: string // "enabled" | "disabled"
    log_path: string
    is_system: boolean
  }

  let jobs: CronJob[] = []
  let loading = true
  let error = ""
  let successMsg = ""

  // Add modal
  let showAddModal = false
  let addName = ""
  let addSchedule = "0 0 * * *"
  let addCommand = ""
  let addLoading = false
  let addError = ""
  let schedulePreset = "daily"

  // Logs modal
  let showLogsModal = false
  let logContent = ""
  let logJob: CronJob | null = null
  let logLoading = false
  let logError = ""

  // Watch for schedule presets
  $: if (schedulePreset === "minute") {
    addSchedule = "*/1 * * * *"
  } else if (schedulePreset === "hour") {
    addSchedule = "0 * * * *"
  } else if (schedulePreset === "daily") {
    addSchedule = "0 0 * * *"
  } else if (schedulePreset === "weekly") {
    addSchedule = "0 0 * * 0"
  } else if (schedulePreset === "monthly") {
    addSchedule = "0 0 1 * *"
  }

  async function fetchCronJobs() {
    loading = true
    error = ""
    successMsg = ""
    try {
      const response = await fetch("/api/cron", {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        jobs = await response.json() || []
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to load cron tasks"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function handleAddCron(e: Event) {
    e.preventDefault()
    addError = ""
    if (!addName.trim()) {
      addError = "Task name cannot be empty"
      return
    }
    if (!addSchedule.trim()) {
      addError = "Schedule expression cannot be empty"
      return
    }
    if (!addCommand.trim()) {
      addError = "Command cannot be empty"
      return
    }

    addLoading = true
    try {
      const response = await fetch("/api/cron/add", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          name: addName.trim(),
          schedule: addSchedule.trim(),
          command: addCommand.trim(),
        }),
      })
      if (response.ok) {
        showAddModal = false
        addName = ""
        addCommand = ""
        schedulePreset = "daily"
        successMsg = "Scheduled task added successfully!"
        await fetchCronJobs()
      } else {
        const errData = await response.json().catch(() => ({}))
        addError = errData.error || "Failed to add scheduled task"
      }
    } catch {
      addError = "Connection error"
    } finally {
      addLoading = false
    }
  }

  async function handleDeleteCron(id: string) {
    if (!confirm("Are you sure you want to delete this scheduled task?")) return

    loading = true
    error = ""
    try {
      const response = await fetch("/api/cron/delete", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ id }),
      })
      if (response.ok) {
        successMsg = "Scheduled task deleted successfully."
        await fetchCronJobs()
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to delete task"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function handleToggleCron(id: string) {
    loading = true
    error = ""
    try {
      const response = await fetch("/api/cron/toggle", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ id }),
      })
      if (response.ok) {
        await fetchCronJobs()
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to toggle status"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function viewLogs(job: CronJob) {
    logJob = job
    showLogsModal = true
    await refreshLogs()
  }

  async function refreshLogs() {
    if (!logJob) return
    logLoading = true
    logError = ""
    try {
      const response = await fetch(`/api/cron/log?id=${logJob.id}`, {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        const data = await response.json()
        logContent = data.log || ""
      } else {
        logError = "Failed to load log file content."
      }
    } catch {
      logError = "Connection error"
    } finally {
      logLoading = false
    }
  }

  async function clearLogs() {
    if (!logJob) return
    if (!confirm("Are you sure you want to clear this task's log file?")) return
    logLoading = true
    try {
      const response = await fetch("/api/cron/log/clear", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ id: logJob.id }),
      })
      if (response.ok) {
        logContent = ""
        await refreshLogs()
      }
    } catch {
      logError = "Connection error"
    } finally {
      logLoading = false
    }
  }

  onMount(() => {
    fetchCronJobs()
  })
</script>

<div class="space-y-6">
  <!-- Title Header -->
  <div class="flex items-center justify-between border-b border-border pb-4">
    <div>
      <h2 class="text-lg font-bold text-foreground flex items-center gap-2">
        <Clock size={18} class="text-primary" />
        Cron Scheduled Tasks
      </h2>
      <p class="text-xs text-muted-foreground mt-0.5">Automate system scripts, backups, or recurring operations with crontab.</p>
    </div>
    <div class="flex items-center gap-2">
      <button 
        on:click={() => { 
          showAddModal = true; 
          addError = ""; 
          addName = ""; 
          addCommand = ""; 
          schedulePreset = "daily"; 
          addSchedule = "0 0 * * *"; 
        }}
        class="inline-flex h-9 items-center gap-1.5 rounded-xl bg-primary px-3.5 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-opacity"
      >
        <Plus size={13} />
        Add Scheduled Task
      </button>
      <button 
        on:click={fetchCronJobs}
        disabled={loading}
        class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-card text-muted-foreground hover:bg-secondary transition-colors"
      >
        <RefreshCw size={14} class={loading ? "animate-spin" : ""} />
      </button>
    </div>
  </div>

  <!-- Error / Success Alert Box -->
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

  <!-- Cron Jobs Table -->
  <div class="rounded-2xl border border-border bg-card overflow-hidden">
    {#if loading && jobs.length === 0}
      <div class="flex flex-col items-center justify-center py-20 text-muted-foreground space-y-2">
        <RefreshCw size={24} class="animate-spin text-primary" />
        <span class="text-xs">Loading scheduled tasks...</span>
      </div>
    {:else if jobs.length === 0}
      <div class="flex flex-col items-center justify-center py-20 text-center px-6 space-y-2">
        <Clock size={36} class="text-muted-foreground/30" />
        <p class="text-sm font-semibold text-foreground">No Cron Tasks</p>
        <p class="text-xs text-muted-foreground">Click 'Add Scheduled Task' to create a new background automation.</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs border-collapse">
          <thead>
            <tr class="border-b border-border bg-secondary/20 text-muted-foreground font-semibold">
              <th class="px-6 py-3">Task Name</th>
              <th class="px-6 py-3 w-40">Schedule</th>
              <th class="px-6 py-3">Shell Command</th>
              <th class="px-6 py-3 w-28">Status</th>
              <th class="px-6 py-3 w-48 text-right">Operations</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border font-mono">
            {#each jobs as job (job.id)}
              <tr class="hover:bg-secondary/10 transition-colors {job.status === 'disabled' ? 'opacity-60' : ''}">
                <td class="px-6 py-3.5 font-bold text-foreground">
                  <div class="flex items-center gap-1.5 font-sans">
                    <Terminal size={14} class="text-primary shrink-0" />
                    <span>{job.name}</span>
                    {#if job.is_system}
                      <span class="inline-flex rounded bg-blue-500/10 px-1.5 py-0.5 text-[8px] font-semibold text-blue-400 font-mono">
                        SYSTEM
                      </span>
                    {/if}
                  </div>
                </td>
                <td class="px-6 py-3.5 text-foreground font-semibold">
                  {job.schedule}
                </td>
                <td class="px-6 py-3.5 text-muted-foreground truncate max-w-xs" title={job.command}>
                  {job.command}
                </td>
                <td class="px-6 py-3.5 font-sans">
                  {#if job.status === "enabled"}
                    <span class="inline-flex items-center gap-1 rounded bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-500">
                      Enabled
                    </span>
                  {:else}
                    <span class="inline-flex items-center gap-1 rounded bg-zinc-500/10 px-2 py-0.5 text-[10px] font-semibold text-zinc-400">
                      Disabled
                    </span>
                  {/if}
                </td>
                <td class="px-6 py-3.5 text-right font-sans">
                  <div class="flex items-center justify-end gap-2">
                    {#if !job.is_system}
                      <button 
                        on:click={() => handleToggleCron(job.id)}
                        class="inline-flex items-center rounded-lg border border-border bg-card px-2 py-1 text-[10px] font-semibold text-foreground hover:bg-secondary transition-all"
                      >
                        {job.status === "enabled" ? "Disable" : "Enable"}
                      </button>
                      <button 
                        on:click={() => viewLogs(job)}
                        class="inline-flex items-center gap-1 rounded-lg border border-border bg-card px-2 py-1 text-[10px] font-semibold text-foreground hover:bg-secondary transition-all"
                      >
                        <FileText size={10} />
                        Logs
                      </button>
                      <button 
                        on:click={() => handleDeleteCron(job.id)}
                        class="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-rose-500 transition-all"
                        title="Delete Task"
                      >
                        <Trash2 size={13} />
                      </button>
                    {:else}
                      <span class="text-[10px] text-muted-foreground italic">Read-only</span>
                    {/if}
                  </div>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>

  <!-- Info Box -->
  <div class="rounded-xl border border-border bg-card p-4 flex items-start gap-3">
    <Info size={15} class="text-primary shrink-0 mt-0.5" />
    <div class="text-[11px] text-muted-foreground space-y-1">
      <p class="font-semibold text-foreground">Important System Guidelines:</p>
      <p>Custom tasks redirect logs to <span class="font-mono bg-secondary/40 px-1 py-0.5 rounded text-foreground">/var/log/cron_tasks/</span>.</p>
      <p>System crontabs are parsed securely. Tasks defined outside this dashboard will appear labeled as <span class="font-mono text-blue-400">SYSTEM</span> and are protected from modification.</p>
    </div>
  </div>

  <!-- Add Task Modal -->
  {#if showAddModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl overflow-hidden">
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <h3 class="text-sm font-bold text-foreground flex items-center gap-1.5">
            <Clock size={14} class="text-primary" />
            Add Scheduled Task
          </h3>
          <button on:click={() => showAddModal = false} class="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>

        <form on:submit={handleAddCron} class="space-y-4 p-6 font-sans">
          {#if addError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20">
              {addError}
            </div>
          {/if}

          <!-- Name -->
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-muted-foreground">
              Task Name
            </label>
            <input 
              type="text" 
              bind:value={addName}
              placeholder="e.g. Daily Database Backup"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
              required
            />
          </div>

          <!-- Schedule Presets -->
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-muted-foreground">
              Schedule Type
            </label>
            <div class="grid grid-cols-5 gap-1.5">
              {#each [
                { key: "minute", label: "Minutely" },
                { key: "hour", label: "Hourly" },
                { key: "daily", label: "Daily" },
                { key: "weekly", label: "Weekly" },
                { key: "monthly", label: "Monthly" }
              ] as opt}
                <button
                  type="button"
                  on:click={() => schedulePreset = opt.key}
                  class="rounded-lg border px-1.5 py-2 text-[10px] font-semibold transition-all text-center {schedulePreset === opt.key ? 'border-primary bg-primary/10 text-primary' : 'border-border bg-card text-muted-foreground hover:bg-secondary'}"
                >
                  {opt.label}
                </button>
              {/each}
            </div>
          </div>

          <!-- Schedule Expression -->
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-muted-foreground">
              Cron Expression
            </label>
            <input 
              type="text" 
              bind:value={addSchedule}
              placeholder="* * * * *"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono font-bold"
              required
            />
            <p class="text-[10px] text-muted-foreground font-mono">Format: minute hour day_of_month month day_of_week</p>
          </div>

          <!-- Shell Command -->
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-muted-foreground flex items-center justify-between">
              <span>Shell Command</span>
            </label>
            <textarea 
              bind:value={addCommand}
              rows={4}
              placeholder="e.g. bash /home/my-script.sh"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
          </div>

          <div class="flex items-center justify-end gap-3 pt-4 border-t border-border">
            <button 
              type="button" 
              on:click={() => showAddModal = false}
              class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors"
            >
              Cancel
            </button>
            <button 
              type="submit" 
              disabled={addLoading}
              class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50"
            >
              {addLoading ? "Adding..." : "Add Task"}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- View Logs Modal -->
  {#if showLogsModal && logJob}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-2xl rounded-2xl border border-border bg-card shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        <div class="flex items-center justify-between border-b border-border px-6 py-4 shrink-0">
          <div>
            <h3 class="text-sm font-bold text-foreground">Task Execution Logs</h3>
            <p class="text-[10px] text-muted-foreground mt-0.5">{logJob.name}</p>
          </div>
          <button on:click={() => showLogsModal = false} class="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>

        <div class="p-6 overflow-y-auto grow flex flex-col min-h-[300px]">
          {#if logError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20 mb-4 shrink-0">
              {logError}
            </div>
          {/if}

          <!-- Logger terminal view -->
          <div class="bg-black/95 rounded-xl border border-zinc-800 p-4 text-[11px] font-mono text-zinc-300 overflow-auto grow select-text whitespace-pre-wrap flex-1 max-h-[400px]">
            {#if logLoading && !logContent}
              <span class="text-zinc-500">Connecting to log stream...</span>
            {:else if !logContent}
              <span class="text-zinc-500">[No logs recorded yet. Execution output will stream here.]</span>
            {:else}
              {logContent}
            {/if}
          </div>
        </div>

        <div class="flex items-center justify-between gap-3 px-6 py-4 border-t border-border bg-secondary/10 shrink-0">
          <button 
            on:click={clearLogs}
            disabled={logLoading}
            class="rounded-lg border border-rose-500/20 text-rose-500 hover:bg-rose-500/10 px-4 py-2 text-xs font-semibold transition-colors disabled:opacity-50"
          >
            Clear Log
          </button>
          <div class="flex gap-2">
            <button 
              on:click={refreshLogs}
              disabled={logLoading}
              class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-card px-4 py-2 text-xs font-semibold text-foreground hover:bg-secondary transition-all disabled:opacity-50"
            >
              <RefreshCw size={12} class={logLoading ? "animate-spin" : ""} />
              Refresh
            </button>
            <button 
              type="button" 
              on:click={() => showLogsModal = false}
              class="rounded-lg bg-zinc-800 hover:bg-zinc-700 px-4 py-2 text-xs font-semibold text-white transition-colors"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>
