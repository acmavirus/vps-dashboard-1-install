<script lang="ts">
  import { Box } from "lucide-svelte"
  import type { ContainerInfo } from "./types"
  export let containers: ContainerInfo[] = []
</script>

<div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
  {#each containers as container, index (`${container.name}-${index}`)}
    <div class="rounded-xl border border-border bg-card p-5 space-y-4">
      <div class="flex items-start justify-between">
        <div class="space-y-1">
          <h3 class="max-w-[180px] truncate text-sm font-semibold">{container.name}</h3>
          <p class="max-w-[180px] truncate text-[10px] text-muted-foreground">
            {container.image}
          </p>
        </div>
        <Box size={16} class="text-indigo-400" />
      </div>
      <div class="flex items-center gap-2">
        <span
          class="h-1.5 w-1.5 rounded-full {container.status.toLowerCase().includes('up') ? 'bg-emerald-500' : 'bg-rose-500'}"
        />
        <span class="text-[11px] text-muted-foreground">{container.status}</span>
      </div>
      <div class="grid grid-cols-2 gap-4 border-t border-border pt-2">
        <div>
          <p class="mb-1 text-[10px] text-muted-foreground">CPU Usage</p>
          <p class="text-sm tabular-nums">{container.cpu}</p>
        </div>
        <div>
          <p class="mb-1 text-[10px] text-muted-foreground">Memory</p>
          <p class="text-sm tabular-nums">{container.mem}</p>
        </div>
      </div>
    </div>
  {/each}
</div>
