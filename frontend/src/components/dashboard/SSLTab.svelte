<script lang="ts">
  import { onMount } from "svelte"
  import { Shield, RefreshCw, Clock, ShieldAlert, ShieldCheck } from "lucide-svelte"
  import { toast } from "../../lib/toast"

  export let token: string | null = null

  interface SSLCertInfo {
    domain: string
    issuer: string
    expiry_date: string
    days_left: number
    is_expired: boolean
  }

  let certificates: SSLCertInfo[] = []
  let loading = false
  let actionLoading = false
  let actionDomain = ""

  async function loadCertificates() {
    if (!token) return
    loading = true
    try {
      const response = await fetch("/api/ssl", {
        headers: { Authorization: token }
      })
      if (response.ok) {
        certificates = await response.json()
      } else {
        toast.error("Failed to load certificates", "Could not fetch SSL information.")
      }
    } catch {
      toast.error("Connection error", "Could not connect to the server.")
    } finally {
      loading = false
    }
  }

  async function handleRenew(domain: string) {
    if (!token) return
    actionDomain = domain
    actionLoading = true
    try {
      const response = await fetch("/api/ssl/renew", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token
        },
        body: JSON.stringify({ domain })
      })
      const data = await response.json()
      if (response.ok) {
        toast.success("SSL Renewed", `SSL certificate for "${domain}" has been successfully renewed.`)
        await loadCertificates()
      } else {
        toast.error("Renewal failed", data.error || "Could not renew certificate.")
      }
    } catch {
      toast.error("Connection error", "Could not connect to the server.")
    } finally {
      actionLoading = false
      actionDomain = ""
    }
  }

  function formatDate(dateStr: string) {
    if (!dateStr) return "-"
    const d = new Date(dateStr)
    return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
  }

  onMount(() => {
    loadCertificates()
  })
</script>

<div class="space-y-4">
  <div class="flex items-center justify-between">
    <div>
      <h2 class="text-lg font-bold text-foreground">SSL Certificates</h2>
      <p class="text-xs text-muted-foreground">Manage Let's Encrypt certificates and check expiration statuses</p>
    </div>
    <button
      type="button"
      on:click={loadCertificates}
      disabled={loading}
      class="flex h-9 items-center gap-2 rounded-lg border border-border bg-card px-4 text-xs font-semibold text-foreground hover:bg-secondary/40 transition-colors disabled:opacity-50"
    >
      <RefreshCw size={12} class={loading ? "animate-spin" : ""} />
      Refresh
    </button>
  </div>

  {#if loading && certificates.length === 0}
    <div class="flex flex-col items-center justify-center rounded-2xl border border-border bg-card py-16 text-center">
      <RefreshCw size={24} class="animate-spin text-muted-foreground mb-3" />
      <p class="text-xs text-muted-foreground">Scanning certificate directories...</p>
    </div>
  {:else if certificates.length === 0}
    <div class="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border bg-card/50 py-12 text-center">
      <Shield size={40} class="text-muted-foreground/40 mb-3" />
      <h3 class="text-sm font-semibold text-foreground">No SSL Certificates found</h3>
      <p class="mt-1 text-xs text-muted-foreground max-w-xs">
        SSL certificates created via Let's Encrypt will appear here. You can enable SSL when creating a domain.
      </p>
    </div>
  {:else}
    <div class="rounded-xl border border-border bg-card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs font-light">
          <thead class="border-b border-border bg-secondary/35 text-muted-foreground font-medium">
            <tr>
              <th class="px-5 py-3">Domain</th>
              <th class="px-5 py-3">Issuer</th>
              <th class="px-5 py-3">Expiration Date</th>
              <th class="px-5 py-3 w-36">Days Left</th>
              <th class="px-5 py-3 w-28">Status</th>
              <th class="px-5 py-3 w-28 text-center">Action</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border font-light">
            {#each certificates as cert}
              <tr class="hover:bg-secondary/15 transition-colors">
                <td class="px-5 py-4 font-bold text-foreground">{cert.domain}</td>
                <td class="px-5 py-4 text-muted-foreground">{cert.issuer || "Let's Encrypt"}</td>
                <td class="px-5 py-4 text-muted-foreground font-mono">{formatDate(cert.expiry_date)}</td>
                <td class="px-5 py-4 font-mono font-medium">
                  <span class={cert.days_left <= 15 ? "text-rose-500 font-bold" : cert.days_left <= 30 ? "text-amber-500" : "text-emerald-500"}>
                    {cert.days_left} days
                  </span>
                </td>
                <td class="px-5 py-4">
                  {#if cert.is_expired}
                    <span class="inline-flex items-center gap-1 rounded-full bg-rose-500/10 px-2 py-1 text-[10px] font-bold text-rose-500">
                      <ShieldAlert size={10} /> Expired
                    </span>
                  {:else if cert.days_left <= 30}
                    <span class="inline-flex items-center gap-1 rounded-full bg-amber-500/10 px-2 py-1 text-[10px] font-bold text-amber-500">
                      <Clock size={10} /> Expiring
                    </span>
                  {:else}
                    <span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 px-2 py-1 text-[10px] font-bold text-emerald-500">
                      <ShieldCheck size={10} /> Valid
                    </span>
                  {/if}
                </td>
                <td class="px-5 py-4 text-center">
                  <button
                    type="button"
                    on:click={() => handleRenew(cert.domain)}
                    disabled={actionLoading}
                    class="inline-flex h-8 items-center gap-1 rounded-lg border border-border bg-card px-2.5 text-[11px] font-bold text-foreground hover:bg-secondary/40 hover:text-primary transition-all disabled:opacity-50"
                  >
                    {#if actionLoading && actionDomain === cert.domain}
                      <RefreshCw size={10} class="animate-spin" /> Renewing...
                    {:else}
                      <RefreshCw size={10} /> Renew
                    {/if}
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}
</div>
