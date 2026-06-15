<script lang="ts">
  import { fly, fade } from "svelte/transition"
  import { flip } from "svelte/animate"
  import { CheckCircle, XCircle, AlertTriangle, Info, X } from "lucide-svelte"
  import { toast, type Toast } from "../lib/toast"

  const icons = {
    success: CheckCircle,
    error: XCircle,
    warning: AlertTriangle,
    info: Info,
  }

  const styles = {
    success: {
      border: "border-emerald-500/30",
      icon: "text-emerald-500",
      bg: "bg-emerald-500/8",
      bar: "bg-emerald-500",
    },
    error: {
      border: "border-rose-500/30",
      icon: "text-rose-500",
      bg: "bg-rose-500/8",
      bar: "bg-rose-500",
    },
    warning: {
      border: "border-amber-500/30",
      icon: "text-amber-500",
      bg: "bg-amber-500/8",
      bar: "bg-amber-500",
    },
    info: {
      border: "border-blue-500/30",
      icon: "text-blue-500",
      bg: "bg-blue-500/8",
      bar: "bg-blue-500",
    },
  }

  let toasts: Toast[] = []
  toast.subscribe((v) => (toasts = v))
</script>

<div
  class="fixed bottom-5 right-5 z-[9999] flex flex-col gap-2.5 pointer-events-none"
  aria-live="polite"
  aria-label="Notifications"
>
  {#each toasts as t (t.id)}
    <div
      animate:flip={{ duration: 250 }}
      in:fly={{ x: 60, duration: 300, opacity: 0 }}
      out:fade={{ duration: 200 }}
      class="pointer-events-auto w-[340px] max-w-[calc(100vw-2.5rem)] rounded-xl border backdrop-blur-sm shadow-lg overflow-hidden
        {styles[t.type].border} bg-card/95"
    >
      <div class="flex items-start gap-3 p-4 {styles[t.type].bg}">
        <div class="shrink-0 mt-0.5">
          <svelte:component
            this={icons[t.type]}
            size={16}
            class={styles[t.type].icon}
          />
        </div>
        <div class="flex-1 min-w-0">
          <p class="text-xs font-semibold text-foreground leading-snug">{t.title}</p>
          {#if t.message}
            <p class="text-[11px] text-muted-foreground mt-0.5 leading-relaxed break-words">{t.message}</p>
          {/if}
        </div>
        <button
          type="button"
          on:click={() => toast.remove(t.id)}
          class="shrink-0 rounded-md p-0.5 text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
          aria-label="Dismiss"
        >
          <X size={13} />
        </button>
      </div>
      <!-- Progress bar -->
      {#if (t.duration ?? 4000) > 0}
        <div class="h-0.5 {styles[t.type].bar} opacity-40"
          style="animation: toast-shrink {t.duration ?? 4000}ms linear forwards"
        />
      {/if}
    </div>
  {/each}
</div>

<style>
  @keyframes toast-shrink {
    from { width: 100%; }
    to { width: 0%; }
  }
</style>
