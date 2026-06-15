<script lang="ts">
  import { onMount, onDestroy } from "svelte"
  import { Terminal } from "xterm"
  import { FitAddon } from "xterm-addon-fit"
  import "xterm/css/xterm.css"

  export let token: string | null

  let terminalContainer: HTMLDivElement
  let term: Terminal | null = null
  let fitAddon: FitAddon | null = null
  let ws: WebSocket | null = null
  let status: "connecting" | "connected" | "disconnected" = "connecting"
  let errorMsg = ""

  function initTerminal() {
    if (!terminalContainer || !token) return

    status = "connecting"
    errorMsg = ""

    // 1. Initialize xterm.js
    term = new Terminal({
      cursorBlink: true,
      fontFamily: "'JetBrains Mono', Consolas, Monaco, 'Andale Mono', monospace",
      fontSize: 13,
      lineHeight: 1.2,
      theme: {
        background: "#0b0f19", // Sleek dark tech background
        foreground: "#f8fafc",
        cursor: "#38bdf8",
        selectionBackground: "rgba(56, 189, 248, 0.3)",
        black: "#0f172a",
        red: "#ef4444",
        green: "#22c55e",
        yellow: "#eab308",
        blue: "#3b82f6",
        magenta: "#a855f7",
        cyan: "#06b6d4",
        white: "#cbd5e1",
      },
    })

    fitAddon = new FitAddon()
    term.loadAddon(fitAddon)

    // 2. Connect WebSocket
    const loc = window.location
    const protocol = loc.protocol === "https:" ? "wss:" : "ws:"
    const wsUrl = `${protocol}//${loc.host}/api/terminal/ws?token=${token}`

    ws = new WebSocket(wsUrl)
    ws.binaryType = "arraybuffer"

    ws.onopen = () => {
      status = "connected"
      if (term) {
        term.open(terminalContainer)
        fitAddon?.fit()
        term.focus()

        // Send initial size
        ws?.send(
          JSON.stringify({
            type: "resize",
            cols: term.cols,
            rows: term.rows,
          })
        )
      }
    }

    ws.onmessage = (event) => {
      if (term) {
        if (event.data instanceof ArrayBuffer) {
          term.write(new Uint8Array(event.data))
        } else {
          term.write(event.data)
        }
      }
    }

    ws.onclose = () => {
      status = "disconnected"
      term?.write("\r\n\r\n[SSH Connection Closed]\r\n")
    }

    ws.onerror = (err) => {
      status = "disconnected"
      errorMsg = "WebSocket connection failed."
      console.error(err)
    }

    // 3. Pipe Terminal data to WebSocket
    term.onData((data) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })

    // 4. Pipe Resize data to WebSocket
    term.onResize(({ cols, rows }) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols, rows }))
      }
    })

    // 5. Handle window resizing
    window.addEventListener("resize", handleWindowResize)
  }

  function handleWindowResize() {
    if (fitAddon && term) {
      fitAddon.fit()
    }
  }

  onMount(() => {
    initTerminal()
  })

  onDestroy(() => {
    window.removeEventListener("resize", handleWindowResize)
    if (ws) ws.close()
    if (term) term.dispose()
  })
</script>

<div class="flex flex-col h-full w-full bg-[#0b0f19] text-slate-100 overflow-hidden relative">
  <!-- Terminal Header Bar -->
  <div class="h-10 shrink-0 bg-slate-900 border-b border-slate-800 flex items-center justify-between px-4 select-none">
    <div class="flex items-center gap-2">
      <!-- Mac-style dots -->
      <span class="h-3 w-3 rounded-full bg-rose-500"></span>
      <span class="h-3 w-3 rounded-full bg-amber-500"></span>
      <span class="h-3 w-3 rounded-full bg-emerald-500"></span>
      <span class="text-xs font-semibold text-slate-400 ml-2 font-mono">web-ssh@localhost</span>
    </div>

    <!-- Status badge -->
    <div class="flex items-center gap-1.5">
      <span class="h-1.5 w-1.5 rounded-full {status === 'connected' ? 'bg-emerald-400 animate-pulse' : status === 'connecting' ? 'bg-amber-400 animate-pulse' : 'bg-rose-500'}"></span>
      <span class="text-[10px] uppercase font-semibold text-slate-400">
        {status}
      </span>
    </div>
  </div>

  <!-- Terminal Output -->
  <div class="flex-1 p-3 overflow-hidden" bind:this={terminalContainer}>
    {#if status === 'disconnected'}
      <div class="absolute inset-0 bg-slate-950/90 backdrop-blur-sm flex flex-col items-center justify-center gap-4 text-center z-10">
        <p class="text-sm text-slate-300 font-mono">SSH session disconnected or could not start</p>
        {#if errorMsg}
          <p class="text-xs text-rose-400 font-mono">{errorMsg}</p>
        {/if}
        <button 
          on:click={initTerminal} 
          class="rounded-lg bg-blue-600 px-4 py-2 text-xs font-medium text-white transition-colors hover:bg-blue-500"
        >
          Reconnect Terminal
        </button>
      </div>
    {/if}
  </div>
</div>

<style>
  :global(.xterm) {
    padding: 8px;
    height: 100%;
  }
</style>
