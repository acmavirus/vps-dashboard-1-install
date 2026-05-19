<script lang="ts">
  import { Activity, RotateCcw, Square, Play } from "lucide-svelte"
  import type { Pm2Process } from "./types"

  export let pm2: Pm2Process[] = []
  export let handlePM2Action: (name: string, action: string) => void
  export let formatUptime: (seconds: number) => string

  const gb = (bytes: number) => (bytes ? `${(bytes / 1073741824).toFixed(1)} GB` : "0 GB")
</script>

<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
  {#each pm2 as process, index (`${process.name}-${index}`)}
    <div class="rounded-xl border border-border bg-card p-5 space-y-4">
      <div class="flex items-start justify-between">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <h3 class="max-w-[150px] truncate text-sm font-semibold">{process.name}</h3>
            <span class="rounded bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground">
              ID: {process.pm_id}
            </span>
          </div>
          <p class="max-w-[180px] truncate text-[10px] text-muted-foreground">
            {process.pm2_env?.pm_uptime
              ? formatUptime(Math.floor((Date.now() - process.pm2_env.pm_uptime) / 1000))
              : "N/A"}
          </p>
        </div>
        <div class="flex gap-1">
          <button
            on:click={() => handlePM2Action(process.name, "restart")}
            title="Restart"
            class="rounded p-1 text-muted-foreground hover:bg-secondary hover:text-blue-400"
          >
            <RotateCcw size={14} />
          </button>
          <button
            on:click={() => handlePM2Action(process.name, "stop")}
            title="Stop"
            class="rounded p-1 text-muted-foreground hover:bg-secondary hover:text-rose-400"
          >
            <Square size={14} />
          </button>
          <button
            on:click={() => handlePM2Action(process.name, "start")}
            title="Start"
            class="rounded p-1 text-muted-foreground hover:bg-secondary hover:text-emerald-400"
          >
            <Play size={14} />
          </button>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <span
          class="h-1.5 w-1.5 rounded-full {process.status === 'online' ? 'bg-emerald-500' : 'bg-rose-500'}"
        />
        <span class="capitalize text-[11px] text-muted-foreground">{process.status}</span>
      </div>
      <div class="grid grid-cols-2 gap-4 border-t border-border pt-2">
        <div>
          <p class="mb-1 text-[10px] text-muted-foreground">CPU</p>
          <p class="text-sm tabular-nums">{process.monit?.cpu ?? 0}%</p>
        </div>
        <div>
          <p class="mb-1 text-[10px] text-muted-foreground">Memory</p>
          <p class="text-sm tabular-nums">{gb(process.monit?.memory ?? 0)}</p>
        </div>
      </div>
    </div>
  {:else}
    <div class="col-span-full rounded-xl border border-dashed border-border py-12 text-center">
      <Activity class="mx-auto mb-3 opacity-20" size={32} />
      <p class="text-sm font-light text-muted-foreground">No PM2 processes found</p>
    </div>
  {/each}
</div>
