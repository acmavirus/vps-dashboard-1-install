<script lang="ts">
  import { ChevronRight } from "lucide-svelte"
  import type { LogData, LogTabItem } from "./types"

  export let logTabs: LogTabItem[] = []
  export let logTab: string
  export let setLogTab: (value: string) => void
  export let siteTab: "access" | "error"
  export let setSiteTab: (value: "access" | "error") => void
  export let currentLog: LogData | null | undefined
  export let live: boolean
  export let autoScroll: boolean
  export let setAutoScroll: (value: boolean) => void
  export let logEndRef: HTMLDivElement | null = null

  // Phân tích và highlight logs
  function getLineStyles(text: string) {
    if (!text) return []
    return text.split("\n").map(line => {
      let color = "text-foreground/80"
      if (line.includes("ERROR") || line.includes("Failed") || line.includes("crit")) {
        color = "font-medium text-rose-400"
      } else if (line.includes("WARN") || line.includes("warning")) {
        color = "text-amber-400"
      } else if (line.includes(" 200 ") || line.includes("SUCCESS") || line.includes("active")) {
        color = "text-emerald-400"
      } else if (line.includes(" 404 ") || line.includes(" 500 ")) {
        color = "text-rose-500 underline"
      }
      return { line, color }
    })
  }

  $: lines = currentLog ? getLineStyles(currentLog.content) : []
</script>

<div class="flex h-auto flex-col gap-4 lg:h-[640px] lg:flex-row">
  <!-- Desktop Sidebar -->
  <div class="hidden w-52 flex-col gap-1 overflow-y-auto lg:flex">
    {#each logTabs as tab (tab.key)}
      <button
        on:click={() => setLogTab(tab.key)}
        class="flex items-center gap-3 rounded-lg px-4 py-3 text-left text-sm font-light transition-colors {logTab === tab.key ? 'border border-border bg-card text-foreground' : 'text-muted-foreground hover:text-foreground'}"
      >
        <svelte:component this={tab.icon} size={15} class={logTab === tab.key ? tab.color : ""} />
        <span class="truncate">{tab.label}</span>
        {#if logTab === tab.key}
          <ChevronRight size={14} class="ml-auto shrink-0 opacity-40" />
        {/if}
      </button>
    {/each}
  </div>

  <!-- Mobile Scroll List -->
  <div class="flex gap-2 overflow-x-auto pb-2 lg:hidden">
    {#each logTabs as tab (tab.key)}
      <button
        on:click={() => setLogTab(tab.key)}
        class="whitespace-nowrap rounded-lg px-4 py-2 text-xs font-light transition-colors {logTab === tab.key ? 'border border-border bg-card text-foreground' : 'text-muted-foreground'}"
      >
        <span class="flex items-center gap-2">
          <svelte:component this={tab.icon} size={13} />
          {tab.label}
        </span>
      </button>
    {/each}
  </div>

  <!-- Log Content Card -->
  <div class="flex h-[480px] flex-1 flex-col overflow-hidden rounded-xl border border-border bg-card lg:h-full">
    <div class="flex items-center justify-between border-b border-border bg-secondary/30 px-4 py-3 sm:px-5">
      <div class="flex items-center gap-3">
        <div class="flex gap-1.5">
          <span class="h-2.5 w-2.5 rounded-full bg-border" />
          <span class="h-2.5 w-2.5 rounded-full bg-border" />
          <span class="h-2.5 w-2.5 rounded-full bg-border" />
        </div>
        <span class="max-w-[300px] truncate text-[11px] font-light text-muted-foreground">
          {currentLog?.path ?? "loading..."}
        </span>
      </div>
      <div class="flex items-center gap-4">
        {#if logTab.startsWith("site:")}
          <div class="rounded-md border border-border bg-background/50 p-0.5">
            <button
              on:click={() => setSiteTab("access")}
              class="rounded-[4px] px-3 py-1 text-[10px] transition-all {siteTab === 'access' ? 'bg-secondary text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
            >
              Access
            </button>
            <button
              on:click={() => setSiteTab("error")}
              class="rounded-[4px] px-3 py-1 text-[10px] transition-all {siteTab === 'error' ? 'bg-secondary text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
            >
              Error
            </button>
          </div>
        {/if}
        <span
          class="flex items-center gap-1.5 text-[10px] font-light {live ? 'text-emerald-400' : 'text-muted-foreground'}"
        >
          <span
            class="h-1.5 w-1.5 rounded-full {live ? 'bg-emerald-500' : 'bg-muted-foreground'}"
          />
          {live ? "live" : "offline"}
        </span>
        <div class="flex items-center gap-2 border-l border-border pl-4">
          <input
            type="checkbox"
            id="autoscroll"
            checked={autoScroll}
            on:change={(e) => setAutoScroll(e.currentTarget.checked)}
            class="h-3 w-3 rounded border-zinc-700 bg-zinc-800"
          />
          <label for="autoscroll" class="cursor-pointer select-none text-[10px] text-muted-foreground">
            Auto-scroll
          </label>
        </div>
      </div>
    </div>

    <!-- Scrolling area -->
    <div class="flex-1 overflow-y-auto p-4 sm:p-5">
      <div class="whitespace-pre-wrap font-mono text-[12px] font-light leading-relaxed sm:text-[13px]">
        {#if currentLog}
          {#each lines as item, idx (`line-${idx}`)}
            <div class={item.color}>{item.line}</div>
          {/each}
        {:else}
          <div class="text-muted-foreground">Waiting for data...</div>
        {/if}
        <div bind:this={logEndRef} />
      </div>
      <div class="h-8" />
    </div>
  </div>
</div>
