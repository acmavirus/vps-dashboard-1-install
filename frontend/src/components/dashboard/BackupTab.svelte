<script lang="ts">
  import { onMount } from "svelte"
  import { toast } from "../../lib/toast"
  import { 
    Cloud, 
    Database, 
    Globe, 
    HardDrive, 
    RefreshCw, 
    Trash2, 
    Play, 
    Save, 
    Settings, 
    History,
    CheckCircle2,
    XCircle,
    Loader2
  } from "lucide-svelte"

  export let token: string | null

  interface BackupConfig {
    provider: string // "local", "s3", "gdrive"
    s3_access_key: string
    s3_secret_key: string
    s3_bucket: string
    s3_endpoint: string
    s3_region: string
    gdrive_folder: string
  }

  interface BackupHistoryEntry {
    id: string
    type: string // "site", "database"
    target: string
    file: string
    size: number
    timestamp: string
    status: string // "success", "failed"
    cloud_sync: string // "synced", "pending", "none"
  }

  interface DomainInfo {
    domain: string
  }

  let config: BackupConfig = {
    provider: "local",
    s3_access_key: "",
    s3_secret_key: "",
    s3_bucket: "",
    s3_endpoint: "",
    s3_region: "",
    gdrive_folder: ""
  }

  let history: BackupHistoryEntry[] = []
  let domains: DomainInfo[] = []
  let databases: string[] = []

  let backupType: "site" | "database" = "site"
  let backupTarget = ""

  let loadingConfig = false
  let loadingHistory = false
  let savingConfig = false
  let runningBackup = false
  let loadingTargets = false

  const formatSize = (bytes: number): string => {
    if (bytes === 0) return "0 Bytes"
    const k = 1024
    const sizes = ["Bytes", "KB", "MB", "GB"]
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
  }

  const formatDate = (dateStr: string): string => {
    try {
      const d = new Date(dateStr)
      return d.toLocaleString()
    } catch {
      return dateStr
    }
  }

  async function fetchConfig() {
    loadingConfig = true
    try {
      const res = await fetch("/api/backup/config", {
        headers: { Authorization: token || "" }
      })
      if (res.ok) {
        config = await res.json()
      }
    } catch (err) {
      console.error(err)
      toast.error("Error", "Failed to fetch backup configuration.")
    } finally {
      loadingConfig = false
    }
  }

  async function fetchHistory() {
    loadingHistory = true
    try {
      const res = await fetch("/api/backup/list", {
        headers: { Authorization: token || "" }
      })
      if (res.ok) {
        history = await res.json()
        // Sort history by date descending
        history.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      }
    } catch (err) {
      console.error(err)
      toast.error("Error", "Failed to fetch backup history.")
    } finally {
      loadingHistory = false
    }
  }

  async function fetchTargets() {
    loadingTargets = true
    try {
      // Fetch domains
      const domRes = await fetch("/api/domains", {
        headers: { Authorization: token || "" }
      })
      if (domRes.ok) {
        domains = await domRes.json()
      }

      // Fetch databases
      const dbRes = await fetch("/api/databases", {
        headers: { Authorization: token || "" }
      })
      if (dbRes.ok) {
        const dbData = await dbRes.json()
        databases = dbData.databases || []
      }

      // Set default target
      updateDefaultTarget()
    } catch (err) {
      console.error(err)
    } finally {
      loadingTargets = false
    }
  }

  function updateDefaultTarget() {
    if (backupType === "site") {
      backupTarget = domains.length > 0 ? domains[0].domain : ""
    } else {
      backupTarget = databases.length > 0 ? databases[0] : ""
    }
  }

  $: {
    if (backupType) {
      updateDefaultTarget()
    }
  }

  async function saveConfig() {
    savingConfig = true
    try {
      const res = await fetch("/api/backup/config", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || ""
        },
        body: JSON.stringify(config)
      })
      if (res.ok) {
        toast.success("Success", "Backup settings saved successfully.")
      } else {
        const data = await res.json()
        toast.error("Error", data.error || "Failed to save backup config.")
      }
    } catch (err) {
      console.error(err)
      toast.error("Error", "Connection failed while saving settings.")
    } finally {
      savingConfig = false
    }
  }

  async function triggerBackup() {
    if (!backupTarget) {
      toast.error("Validation Error", "Please select a backup target.")
      return
    }

    runningBackup = true
    try {
      const res = await fetch("/api/backup/run", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || ""
        },
        body: JSON.stringify({
          type: backupType,
          target: backupTarget
        })
      })

      const data = await res.json()
      if (res.ok) {
        toast.success(
          "Backup Complete",
          `Successfully backed up ${backupType} "${backupTarget}". File size: ${formatSize(data.size)}`
        )
        fetchHistory()
      } else {
        toast.error("Backup Failed", data.error || "An error occurred during backup.")
      }
    } catch (err) {
      console.error(err)
      toast.error("Error", "Failed to start backup process.")
    } finally {
      runningBackup = false
    }
  }

  onMount(() => {
    fetchConfig()
    fetchHistory()
    fetchTargets()
  })
</script>

<div class="h-full overflow-y-auto p-6 space-y-6">
  <!-- Header Card -->
  <div class="rounded-xl border border-border bg-card p-6 shadow-sm">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h3 class="text-base font-semibold flex items-center gap-2">
          <Cloud class="text-primary h-5 w-5" />
          Cloud Backup & Recovery
        </h3>
        <p class="text-xs font-light text-muted-foreground mt-1">
          Configure secure local or cloud backups for your websites and databases. Supports S3 and Google Drive.
        </p>
      </div>
      <button
        type="button"
        on:click={() => { fetchConfig(); fetchHistory(); fetchTargets(); }}
        class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-background px-3 py-1.5 text-xs font-medium text-foreground hover:bg-secondary transition-colors"
      >
        <RefreshCw size={13} class={loadingConfig || loadingHistory ? "animate-spin" : ""} />
        Refresh Data
      </button>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Trigger Backup Box -->
    <div class="rounded-xl border border-border bg-card p-6 shadow-sm space-y-6 h-fit">
      <div>
        <h4 class="text-sm font-semibold flex items-center gap-2">
          <Play size={14} class="text-primary" />
          Instant Backup
        </h4>
        <p class="text-[11px] text-muted-foreground mt-0.5">Run manual backups right now.</p>
      </div>

      <div class="space-y-4">
        <!-- Type Selection -->
        <div class="space-y-1.5">
          <span class="text-xs font-semibold text-muted-foreground">Backup Type</span>
          <div class="flex bg-secondary/50 rounded-lg p-0.5 border border-border">
            <button
              type="button"
              on:click={() => backupType = "site"}
              class="flex-1 py-1.5 rounded-md text-xs font-medium transition-colors {backupType === 'site' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
            >
              <span class="flex items-center justify-center gap-1.5">
                <Globe size={12} />
                Website Folder
              </span>
            </button>
            <button
              type="button"
              on:click={() => backupType = "database"}
              class="flex-1 py-1.5 rounded-md text-xs font-medium transition-colors {backupType === 'database' ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}"
            >
              <span class="flex items-center justify-center gap-1.5">
                <Database size={12} />
                Database
              </span>
            </button>
          </div>
        </div>

        <!-- Target Selection -->
        <div class="space-y-1.5">
          <label for="backup-target" class="text-xs font-semibold text-muted-foreground">Select Target</label>
          {#if loadingTargets}
            <div class="h-9 w-full bg-secondary/50 rounded-lg animate-pulse"></div>
          {:else}
            <select
              id="backup-target"
              bind:value={backupTarget}
              class="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              {#if backupType === "site"}
                {#each domains as dom}
                  <option value={dom.domain}>{dom.domain}</option>
                {:else}
                  <option value="">No websites found</option>
                {/each}
              {:else}
                {#each databases as db}
                  <option value={db}>{db}</option>
                {:else}
                  <option value="">No databases configured</option>
                {/each}
              {/if}
            </select>
          {/if}
        </div>

        <button
          type="button"
          on:click={triggerBackup}
          disabled={runningBackup || !backupTarget}
          class="w-full inline-flex items-center justify-center gap-2 rounded-lg bg-blue-600 hover:bg-blue-700 text-xs font-medium text-white py-2.5 transition-colors disabled:opacity-50"
        >
          {#if runningBackup}
            <Loader2 size={13} class="animate-spin" />
            Creating Backup...
          {:else}
            <Play size={13} />
            Backup Now
          {/if}
        </button>
      </div>
    </div>

    <!-- Configure Storage Box -->
    <div class="lg:col-span-2 rounded-xl border border-border bg-card p-6 shadow-sm space-y-6">
      <div>
        <h4 class="text-sm font-semibold flex items-center gap-2">
          <Settings size={14} class="text-primary" />
          Backup Destination Settings
        </h4>
        <p class="text-[11px] text-muted-foreground mt-0.5">Specify where backups should be archived (Local storage vs Cloud storage).</p>
      </div>

      {#if loadingConfig}
        <div class="space-y-4 animate-pulse">
          <div class="h-8 w-1/3 bg-secondary rounded"></div>
          <div class="grid grid-cols-2 gap-4">
            <div class="h-10 bg-secondary rounded"></div>
            <div class="h-10 bg-secondary rounded"></div>
          </div>
        </div>
      {:else}
        <form on:submit|preventDefault={saveConfig} class="space-y-5">
          <!-- Provider Picker -->
          <div class="space-y-1.5">
            <span class="text-xs font-semibold text-muted-foreground">Storage Provider</span>
            <div class="grid grid-cols-3 gap-3">
              <button
                type="button"
                on:click={() => config.provider = "local"}
                class="flex flex-col items-center justify-center gap-1.5 p-3 rounded-lg border text-xs font-medium transition-all {config.provider === 'local' ? 'border-primary bg-primary/5 text-primary' : 'border-border bg-secondary/10 text-muted-foreground hover:text-foreground'}"
              >
                <HardDrive size={16} />
                Local Storage
              </button>
              <button
                type="button"
                on:click={() => config.provider = "s3"}
                class="flex flex-col items-center justify-center gap-1.5 p-3 rounded-lg border text-xs font-medium transition-all {config.provider === 's3' ? 'border-primary bg-primary/5 text-primary' : 'border-border bg-secondary/10 text-muted-foreground hover:text-foreground'}"
              >
                <Cloud size={16} />
                Amazon S3
              </button>
              <button
                type="button"
                on:click={() => config.provider = "gdrive"}
                class="flex flex-col items-center justify-center gap-1.5 p-3 rounded-lg border text-xs font-medium transition-all {config.provider === 'gdrive' ? 'border-primary bg-primary/5 text-primary' : 'border-border bg-secondary/10 text-muted-foreground hover:text-foreground'}"
              >
                <Cloud size={16} />
                Google Drive
              </button>
            </div>
          </div>

          <!-- Conditional settings based on provider -->
          {#if config.provider === 's3'}
            <div class="grid grid-cols-1 md:grid-cols-2 gap-4 pt-2 border-t border-border">
              <div class="space-y-1">
                <label for="s3-key" class="text-xs font-medium text-muted-foreground">Access Key ID</label>
                <input
                  id="s3-key"
                  type="text"
                  bind:value={config.s3_access_key}
                  placeholder="e.g. AKIAIOSFODNN7EXAMPLE"
                  class="w-full rounded-lg border border-border bg-background px-3 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
                  required
                />
              </div>
              <div class="space-y-1">
                <label for="s3-secret" class="text-xs font-medium text-muted-foreground">Secret Access Key</label>
                <input
                  id="s3-secret"
                  type="password"
                  bind:value={config.s3_secret_key}
                  placeholder="••••••••••••••••"
                  class="w-full rounded-lg border border-border bg-background px-3 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
                  required
                />
              </div>
              <div class="space-y-1">
                <label for="s3-bucket" class="text-xs font-medium text-muted-foreground">S3 Bucket Name</label>
                <input
                  id="s3-bucket"
                  type="text"
                  bind:value={config.s3_bucket}
                  placeholder="e.g. acmadash-backup"
                  class="w-full rounded-lg border border-border bg-background px-3 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500"
                  required
                />
              </div>
              <div class="space-y-1">
                <label for="s3-region" class="text-xs font-medium text-muted-foreground">S3 Region</label>
                <input
                  id="s3-region"
                  type="text"
                  bind:value={config.s3_region}
                  placeholder="e.g. us-east-1"
                  class="w-full rounded-lg border border-border bg-background px-3 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              <div class="md:col-span-2 space-y-1">
                <label for="s3-endpoint" class="text-xs font-medium text-muted-foreground">S3 Custom Endpoint (Optional)</label>
                <input
                  id="s3-endpoint"
                  type="text"
                  bind:value={config.s3_endpoint}
                  placeholder="e.g. https://s3.us-west-004.backblazeb2.com"
                  class="w-full rounded-lg border border-border bg-background px-3 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
                />
              </div>
            </div>
          {:else if config.provider === 'gdrive'}
            <div class="space-y-1 pt-2 border-t border-border">
              <label for="gdrive-folder" class="text-xs font-medium text-muted-foreground">Google Drive Folder ID</label>
              <input
                id="gdrive-folder"
                type="text"
                bind:value={config.gdrive_folder}
                placeholder="e.g. 1a2b3c4d5e6f7g8h9i0j-FolderID"
                class="w-full rounded-lg border border-border bg-background px-3 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
                required
              />
              <span class="text-[10px] text-muted-foreground block mt-1">Requires standard rclone config for "acmadash_backup" remote to be configured on the server.</span>
            </div>
          {:else}
            <div class="p-3.5 rounded-lg border border-border bg-secondary/10 text-xs text-muted-foreground">
              Local backups will be securely stored under <code class="font-mono bg-secondary/50 px-1 py-0.5 rounded">/var/www/backups</code>. No credentials are required.
            </div>
          {/if}

          <div class="flex justify-end pt-2">
            <button
              type="submit"
              disabled={savingConfig}
              class="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 hover:bg-blue-700 text-xs font-medium text-white px-4 py-2 transition-colors disabled:opacity-50"
            >
              {#if savingConfig}
                <Loader2 size={13} class="animate-spin" />
                Saving Config...
              {:else}
                <Save size={13} />
                Save Settings
              {/if}
            </button>
          </div>
        </form>
      {/if}
    </div>
  </div>

  <!-- Backup History List -->
  <div class="rounded-xl border border-border bg-card p-6 shadow-sm space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h4 class="text-sm font-semibold flex items-center gap-2">
          <History size={14} class="text-primary" />
          Backup History
        </h4>
        <p class="text-[11px] text-muted-foreground mt-0.5">List of generated backups stored on your server or cloud.</p>
      </div>
    </div>

    {#if loadingHistory}
      <div class="space-y-3 py-4">
        {#each Array(3) as _}
          <div class="h-10 bg-secondary/50 rounded-lg animate-pulse"></div>
        {/each}
      </div>
    {:else if history.length === 0}
      <div class="flex flex-col items-center justify-center p-8 text-center border border-dashed border-border rounded-xl">
        <Cloud size={28} class="text-muted-foreground/30 mb-2" />
        <p class="text-xs font-medium text-foreground">No backups found</p>
        <p class="text-[10px] text-muted-foreground mt-0.5">Trigger an instant backup or configure regular automated backups.</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="border-b border-border text-[10px] font-semibold text-muted-foreground uppercase tracking-wider">
              <th class="pb-2">Type</th>
              <th class="pb-2">Target</th>
              <th class="pb-2">File Name</th>
              <th class="pb-2">Size</th>
              <th class="pb-2">Date Created</th>
              <th class="pb-2 text-center">Status</th>
              <th class="pb-2 text-right">Cloud Sync</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border text-xs">
            {#each history as item}
              <tr class="hover:bg-secondary/10 transition-colors">
                <td class="py-2.5 font-medium">
                  <span class="inline-flex items-center gap-1 rounded bg-secondary/60 px-1.5 py-0.5 text-[10px] font-semibold uppercase text-muted-foreground">
                    {#if item.type === 'site'}
                      <Globe size={10} />
                      site
                    {:else}
                      <Database size={10} />
                      db
                    {/if}
                  </span>
                </td>
                <td class="py-2.5 font-medium text-foreground">{item.target}</td>
                <td class="py-2.5 font-mono text-muted-foreground break-all max-w-[200px]" title={item.file}>{item.file}</td>
                <td class="py-2.5 font-medium tabular-nums text-foreground">{formatSize(item.size)}</td>
                <td class="py-2.5 text-muted-foreground">{formatDate(item.timestamp)}</td>
                <td class="py-2.5 text-center">
                  <span class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium {item.status === 'success' ? 'bg-emerald-500/10 text-emerald-500' : 'bg-rose-500/10 text-rose-500'}">
                    {#if item.status === 'success'}
                      <CheckCircle2 size={10} />
                      Success
                    {:else}
                      <XCircle size={10} />
                      Failed
                    {/if}
                  </span>
                </td>
                <td class="py-2.5 text-right font-medium">
                  {#if item.cloud_sync === 'synced'}
                    <span class="text-emerald-500 bg-emerald-500/10 px-1.5 py-0.5 rounded text-[10px]">Synced</span>
                  {:else}
                    <span class="text-muted-foreground bg-secondary/50 px-1.5 py-0.5 rounded text-[10px]">{item.cloud_sync}</span>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</div>
