<script lang="ts">
  import { Search, Trash2, ShieldAlert, Cpu, HardDrive } from "lucide-svelte"
  import type { ProcessInfo } from "./types"
  import { toast } from "../../lib/toast"
  import ConfirmModal from "../ConfirmModal.svelte"

  export let processes: ProcessInfo[] = []
  export let token: string | null = null
  export let onRefresh: () => void = () => {}

  let searchQuery = ""
  let selectedProcess: ProcessInfo | null = null
  let confirmOpen = false
  let actionLoading = false

  // Filter processes based on search query
  $: filteredProcesses = processes.filter(p => {
    const q = searchQuery.toLowerCase()
    return p.name.toLowerCase().includes(q) || (p.command && p.command.toLowerCase().includes(q))
  })

  async function handleKillProcess(pid: number, processName: string) {
    if (!token) return
    actionLoading = true
    try {
      const response = await fetch("/api/processes/kill", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": token
        },
        body: JSON.stringify({ pid })
      })
      const data = await response.json()
      if (response.ok) {
        toast.success("Process terminated", `Successfully sent SIGKILL to process ${processName} (PID: ${pid}).`)
        onRefresh()
      } else {
        toast.error("Failed to kill process", data.error || "An error occurred.")
      }
    } catch (err) {
      toast.error("Connection error", "Could not connect to the server.")
    } finally {
      actionLoading = false
    }
  }

  function triggerKill(p: ProcessInfo) {
    selectedProcess = p
    confirmOpen = true
  }

  async function confirmKill() {
    if (!selectedProcess) return
    await handleKillProcess(selectedProcess.pid, selectedProcess.name)
    selectedProcess = null
  }
</script>

<div class="space-y-4">
  <!-- Header / Search -->
  <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
    <div>
      <h2 class="text-lg font-bold text-foreground">Processes Monitor</h2>
      <p class="text-xs text-muted-foreground">Monitor system resources and terminate processes</p>
    </div>

    <!-- Search Input -->
    <div class="relative w-full sm:w-64">
      <Search size={14} class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Filter by name or command..."
        class="w-full h-9 rounded-lg border border-border bg-card pl-9 pr-4 text-xs placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-primary focus:border-primary"
      />
    </div>
  </div>

  <!-- Table -->
  <div class="rounded-xl border border-border bg-card overflow-hidden">
    <div class="overflow-x-auto">
      <table class="w-full text-left text-xs font-light">
        <thead class="border-b border-border bg-secondary/35 text-muted-foreground font-medium">
          <tr>
            <th class="px-5 py-3 w-20">PID</th>
            <th class="px-5 py-3">Name</th>
            <th class="px-5 py-3 w-36">CPU %</th>
            <th class="px-5 py-3 w-36">Memory %</th>
            <th class="px-5 py-3">Command</th>
            <th class="px-5 py-3 w-16 text-center">Action</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border font-light">
          {#if filteredProcesses.length === 0}
            <tr>
              <td colspan="6" class="px-5 py-8 text-center text-muted-foreground">
                No matching processes found
              </td>
            </tr>
          {:else}
            {#each filteredProcesses as process (process.pid)}
              <tr class="hover:bg-secondary/15 transition-colors">
                <td class="px-5 py-3.5 tabular-nums text-muted-foreground font-mono">{process.pid}</td>
                <td class="px-5 py-3.5 font-bold text-foreground">{process.name}</td>
                
                <!-- CPU with mini bar -->
                <td class="px-5 py-3.5">
                  <div class="flex flex-col gap-1 w-28">
                    <span class="tabular-nums font-mono font-medium">{process.cpu.toFixed(1)}%</span>
                    <div class="h-1 w-full rounded-full bg-secondary overflow-hidden">
                      <div 
                        class="h-full rounded-full bg-indigo-500" 
                        style="width: {Math.min(process.cpu, 100)}%"
                      />
                    </div>
                  </div>
                </td>

                <!-- Memory with mini bar -->
                <td class="px-5 py-3.5">
                  <div class="flex flex-col gap-1 w-28">
                    <span class="tabular-nums font-mono font-medium">{process.memory.toFixed(1)}%</span>
                    <div class="h-1 w-full rounded-full bg-secondary overflow-hidden">
                      <div 
                        class="h-full rounded-full bg-violet-500" 
                        style="width: {Math.min(process.memory, 100)}%"
                      />
                    </div>
                  </div>
                </td>

                <!-- Command -->
                <td class="px-5 py-3.5 max-w-xs truncate text-muted-foreground font-mono text-[11px]" title={process.command}>
                  {process.command || '-'}
                </td>

                <!-- Action Kill -->
                <td class="px-5 py-3.5 text-center">
                  <button
                    type="button"
                    on:click={() => triggerKill(process)}
                    disabled={actionLoading}
                    class="rounded-lg p-1.5 text-rose-500/80 hover:bg-rose-500/10 hover:text-rose-500 transition-colors disabled:opacity-50"
                    title="Terminate Process"
                  >
                    <Trash2 size={13} />
                  </button>
                </td>
              </tr>
            {/each}
          {/if}
        </tbody>
      </table>
    </div>
  </div>
</div>

<!-- Confirm dialog for process termination -->
<ConfirmModal
  bind:isOpen={confirmOpen}
  title="Kill Process"
  message="Are you sure you want to send SIGKILL to process '{selectedProcess?.name}' (PID: {selectedProcess?.pid})? Killing critical system processes can cause OS instability."
  confirmLabel="Kill Process"
  cancelLabel="Cancel"
  variant="danger"
  onConfirm={confirmKill}
/>
