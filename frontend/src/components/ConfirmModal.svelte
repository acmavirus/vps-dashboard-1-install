<script lang="ts">
  import { fade, scale } from "svelte/transition"
  import { AlertTriangle, Trash2, X } from "lucide-svelte"

  export let isOpen = false
  export let title = "Confirm Action"
  export let message = "Are you sure you want to proceed?"
  export let confirmLabel = "Confirm"
  export let cancelLabel = "Cancel"
  export let variant: "danger" | "warning" | "info" = "danger"
  export let onConfirm: (() => void) | (() => Promise<void>) = () => {}
  export let onCancel: () => void = () => { isOpen = false }

  let loading = false

  const variantStyles = {
    danger: {
      icon: Trash2,
      iconBg: "bg-rose-500/10",
      iconColor: "text-rose-500",
      btn: "bg-rose-600 hover:bg-rose-700 text-white",
    },
    warning: {
      icon: AlertTriangle,
      iconBg: "bg-amber-500/10",
      iconColor: "text-amber-500",
      btn: "bg-amber-600 hover:bg-amber-700 text-white",
    },
    info: {
      icon: AlertTriangle,
      iconBg: "bg-blue-500/10",
      iconColor: "text-blue-500",
      btn: "bg-blue-600 hover:bg-blue-700 text-white",
    },
  }

  $: v = variantStyles[variant]

  async function handleConfirm() {
    loading = true
    try {
      await onConfirm()
    } finally {
      loading = false
      isOpen = false
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (!isOpen) return
    if (e.key === "Escape") onCancel()
    if (e.key === "Enter") handleConfirm()
  }
</script>

<svelte:window on:keydown={handleKeydown} />

{#if isOpen}
  <!-- Backdrop -->
  <button
    type="button"
    class="fixed inset-0 z-[9990] bg-black/50 backdrop-blur-[2px] cursor-default border-none outline-none"
    transition:fade={{ duration: 150 }}
    on:click={onCancel}
    aria-label="Close dialog"
  />

  <!-- Dialog -->
  <div
    class="fixed inset-0 z-[9991] flex items-center justify-center p-4 pointer-events-none"
  >
    <div
      class="pointer-events-auto w-full max-w-sm rounded-2xl border border-border bg-card shadow-2xl overflow-hidden"
      transition:scale={{ duration: 200, start: 0.92 }}
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-title"
    >
      <!-- Header -->
      <div class="flex items-start justify-between p-6 pb-4">
        <div class="flex items-center gap-3">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl {v.iconBg}">
            <svelte:component this={v.icon} size={18} class={v.iconColor} />
          </div>
          <div>
            <h3 id="confirm-title" class="text-sm font-bold text-foreground">{title}</h3>
          </div>
        </div>
        <button
          type="button"
          on:click={onCancel}
          class="rounded-lg p-1 text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
        >
          <X size={14} />
        </button>
      </div>

      <!-- Body -->
      <div class="px-6 pb-6">
        <p class="text-sm text-muted-foreground leading-relaxed">{message}</p>

        <!-- Actions -->
        <div class="flex justify-end gap-2 mt-5">
          <button
            type="button"
            on:click={onCancel}
            disabled={loading}
            class="h-9 rounded-lg border border-border px-4 text-xs font-semibold text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors disabled:opacity-50"
          >
            {cancelLabel}
          </button>
          <button
            type="button"
            on:click={handleConfirm}
            disabled={loading}
            class="h-9 rounded-lg px-4 text-xs font-semibold transition-colors disabled:opacity-50 {v.btn}"
          >
            {loading ? "Processing..." : confirmLabel}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
