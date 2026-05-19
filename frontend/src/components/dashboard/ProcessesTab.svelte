<script lang="ts">
  import type { ProcessInfo } from "./types"
  export let processes: ProcessInfo[] = []
</script>

<div class="rounded-xl border border-border bg-card overflow-hidden">
  <div class="overflow-x-auto">
    <table class="w-full text-left text-xs font-light">
      <thead class="border-b border-border bg-secondary/30 text-muted-foreground">
        <tr>
          <th class="px-6 py-3 font-medium">PID</th>
          <th class="px-6 py-3 font-medium">Name</th>
          <th class="px-6 py-3 font-medium">CPU %</th>
          <th class="px-6 py-3 font-medium">Mem %</th>
          <th class="px-6 py-3 font-medium">Command</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-border">
        {#each processes as process (process.pid)}
          <tr class="hover:bg-secondary/10">
            <td class="px-6 py-3 tabular-nums">{process.pid}</td>
            <td class="px-6 py-3 font-normal">{process.name}</td>
            <td class="px-6 py-3 tabular-nums">{process.cpu.toFixed(1)}</td>
            <td class="px-6 py-3 tabular-nums">{process.memory.toFixed(1)}</td>
            <td class="max-w-xs truncate px-6 py-3 text-muted-foreground" title={process.command}>
              {process.command}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>
