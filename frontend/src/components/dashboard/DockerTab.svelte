<script lang="ts">
  import { Box, Play, Square, RotateCcw, Trash2, ShieldAlert } from "lucide-svelte"
  import type { ContainerInfo } from "./types"
  import { toast } from "../../lib/toast"
  import ConfirmModal from "../ConfirmModal.svelte"

  export let containers: ContainerInfo[] = []
  export let token: string | null = null
  export let onRefresh: () => void = () => {}

  let actionLoading = false
  let confirmOpen = false
  let selectedContainer: ContainerInfo | null = null

  function isRunning(status: string) {
    if (!status) return false
    const s = status.toLowerCase()
    return s.includes("up") || s.includes("running")
  }

  // Parse memory percent if available in mem string (e.g. "120MB / 16GB (0.75%)" -> 0.75)
  function getMemPercent(memStr: string): number {
    if (!memStr) return 0
    const match = memStr.match(/\(([\d.]+)%\)/)
    if (match && match[1]) {
      return parseFloat(match[1])
    }
    // Fallback if it's just raw number or percentage
    const clean = memStr.replace("%", "").trim()
    const parsed = parseFloat(clean)
    return isNaN(parsed) ? 0 : parsed
  }

  // Parse cpu percent (e.g. "0.5%" -> 0.5)
  function getCpuPercent(cpuStr: string): number {
    if (!cpuStr) return 0
    const clean = cpuStr.replace("%", "").trim()
    const parsed = parseFloat(clean)
    return isNaN(parsed) ? 0 : parsed
  }

  async function handleContainerControl(containerName: string, action: string) {
    if (!token) return
    actionLoading = true
    try {
      const response = await fetch("/api/docker/control", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Authorization": token
        },
        body: JSON.stringify({ id: containerName, action })
      })
      const data = await response.json()
      if (response.ok) {
        toast.success(
          `Container ${action}ed`, 
          `Container "${containerName}" has been successfully ${action}ed.`
        )
        onRefresh()
      } else {
        toast.error(`Action failed`, data.error || `Failed to ${action} container.`)
      }
    } catch (err) {
      toast.error("Connection error", "Could not connect to the server.")
    } finally {
      actionLoading = false
    }
  }

  function triggerRemove(container: ContainerInfo) {
    selectedContainer = container
    confirmOpen = true
  }

  async function confirmRemove() {
    if (!selectedContainer) return
    await handleContainerControl(selectedContainer.name, "remove")
    selectedContainer = null
  }
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <div>
      <h2 class="text-lg font-bold text-foreground">Docker Containers</h2>
      <p class="text-xs text-muted-foreground">Manage and monitor running container instances</p>
    </div>
  </div>

  {#if containers.length === 0}
    <div class="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/50 py-12 text-center">
      <Box size={40} class="text-muted-foreground/40 mb-3 animate-pulse" />
      <h3 class="text-sm font-semibold text-foreground">No containers found</h3>
      <p class="mt-1 text-xs text-muted-foreground max-w-xs">
        Ensure Docker is running on your server, or deploy apps from the App Store.
      </p>
    </div>
  {:else}
    <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      {#each containers as container, index (`${container.name}-${index}`)}
        {@const running = isRunning(container.status)}
        {@const cpuVal = getCpuPercent(container.cpu)}
        {@const memVal = getMemPercent(container.mem)}
        
        <div class="group relative overflow-hidden rounded-2xl border border-border bg-card p-5 space-y-4 transition-all duration-300 hover:shadow-lg hover:border-border/80">
          <div class="flex items-start justify-between">
            <div class="space-y-1">
              <h3 class="max-w-[200px] truncate text-sm font-bold text-foreground group-hover:text-primary transition-colors">
                {container.name}
              </h3>
              <p class="max-w-[200px] truncate text-[10px] text-muted-foreground font-mono bg-secondary/35 px-1.5 py-0.5 rounded w-fit">
                {container.image}
              </p>
            </div>
            <div class="flex h-8 w-8 items-center justify-center rounded-xl bg-indigo-500/10 text-indigo-400">
              <Box size={16} />
            </div>
          </div>

          <div class="flex items-center justify-between text-xs">
            <div class="flex items-center gap-2">
              <span
                class="h-2 w-2 rounded-full {running ? 'bg-emerald-500 animate-pulse' : 'bg-rose-500'}"
              />
              <span class="text-[11px] font-medium text-muted-foreground">{container.status}</span>
            </div>

            <!-- Action Controls -->
            <div class="flex items-center gap-1.5 opacity-90 lg:opacity-0 group-hover:opacity-100 transition-opacity duration-200">
              {#if !running}
                <button
                  type="button"
                  on:click={() => handleContainerControl(container.name, "start")}
                  disabled={actionLoading}
                  class="flex h-7 w-7 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-500 hover:bg-emerald-500 hover:text-white transition-all disabled:opacity-50"
                  title="Start Container"
                >
                  <Play size={12} />
                </button>
              {:else}
                <button
                  type="button"
                  on:click={() => handleContainerControl(container.name, "stop")}
                  disabled={actionLoading}
                  class="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-500/10 text-amber-500 hover:bg-amber-500 hover:text-white transition-all disabled:opacity-50"
                  title="Stop Container"
                >
                  <Square size={10} />
                </button>
                <button
                  type="button"
                  on:click={() => handleContainerControl(container.name, "restart")}
                  disabled={actionLoading}
                  class="flex h-7 w-7 items-center justify-center rounded-lg bg-blue-500/10 text-blue-500 hover:bg-blue-500 hover:text-white transition-all disabled:opacity-50"
                  title="Restart Container"
                >
                  <RotateCcw size={12} />
                </button>
              {/if}
              <button
                type="button"
                on:click={() => triggerRemove(container)}
                disabled={actionLoading}
                class="flex h-7 w-7 items-center justify-center rounded-lg bg-rose-500/10 text-rose-500 hover:bg-rose-500 hover:text-white transition-all disabled:opacity-50"
                title="Remove Container"
              >
                <Trash2 size={12} />
              </button>
            </div>
          </div>

          <div class="grid grid-cols-2 gap-4 border-t border-border pt-3">
            <div class="space-y-1">
              <div class="flex items-center justify-between text-[10px] text-muted-foreground">
                <span>CPU Usage</span>
                <span class="tabular-nums font-semibold">{container.cpu || '0%'}</span>
              </div>
              <div class="h-1.5 w-full rounded-full bg-secondary overflow-hidden">
                <div 
                  class="h-full rounded-full bg-indigo-500 transition-all duration-500" 
                  style="width: {Math.min(cpuVal, 100)}%"
                />
              </div>
            </div>
            <div class="space-y-1">
              <div class="flex items-center justify-between text-[10px] text-muted-foreground">
                <span>Memory</span>
                <span class="truncate max-w-[80px] tabular-nums font-semibold" title={container.mem}>{container.mem || '0MB'}</span>
              </div>
              <div class="h-1.5 w-full rounded-full bg-secondary overflow-hidden">
                <div 
                  class="h-full rounded-full bg-violet-500 transition-all duration-500" 
                  style="width: {Math.min(memVal || (running ? 5 : 0), 100)}%"
                />
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Confirm dialog for container removal -->
<ConfirmModal
  bind:isOpen={confirmOpen}
  title="Remove Container"
  message="Are you sure you want to permanently delete container '{selectedContainer?.name}'? This will also remove any associated non-persistent volumes."
  confirmLabel="Delete"
  cancelLabel="Cancel"
  variant="danger"
  onConfirm={confirmRemove}
/>
