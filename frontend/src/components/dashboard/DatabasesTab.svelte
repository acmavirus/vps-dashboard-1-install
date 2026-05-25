<script lang="ts">
  import { onMount } from "svelte"
  import { 
    Database, 
    Plus, 
    Trash2, 
    RefreshCw, 
    X, 
    Settings, 
    Check, 
    AlertTriangle, 
    Download, 
    DatabaseZap, 
    Info,
    Eye
  } from "lucide-svelte"
  import DatabaseExplorer from "./DatabaseExplorer.svelte"

  export let token: string | null = null

  let databases: string[] = []
  let configured = false
  let dbHost = "127.0.0.1"
  let dbPort = "3306"
  let dbUser = "root"
  let dbPassword = ""
  let hasPassword = false

  let loading = true
  let error = ""
  let successMsg = ""
  let exploringDb = ""

  // Toggles credentials card
  let showConfigCard = false

  // Create DB modal
  let showCreateModal = false
  let newDbName = ""
  let createUser = true
  let newDbUser = ""
  let newDbPassword = ""
  let customUserEdited = false
  let createError = ""
  let createLoading = false

  function generateRandomPassword() {
    const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    let pass = ""
    for (let i = 0; i < 14; i++) {
      pass += chars.charAt(Math.floor(Math.random() * chars.length))
    }
    return pass
  }

  $: if (createUser && newDbName && !customUserEdited) {
    newDbUser = newDbName
  }

  // Backup states
  let backupLoading = false
  let backupDbName = ""

  async function fetchDatabases() {
    loading = true
    error = ""
    successMsg = ""
    try {
      const response = await fetch("/api/databases", {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        const data = await response.json()
        configured = data.configured
        dbHost = data.host || "127.0.0.1"
        dbPort = data.port || "3306"
        dbUser = data.username || "root"
        hasPassword = data.has_password
        databases = data.databases || []
        
        // Show credentials config if DB is not configured yet
        if (!configured) {
          showConfigCard = true
        }

        if (data.error) {
          error = data.error
        }
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to load database status"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function handleSaveConfig(e: Event) {
    e.preventDefault()
    loading = true
    error = ""
    successMsg = ""
    try {
      const response = await fetch("/api/databases/config", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          host: dbHost,
          port: dbPort,
          username: dbUser,
          password: dbPassword,
        }),
      })
      if (response.ok) {
        successMsg = "Database credentials saved and verified successfully!"
        showConfigCard = false
        dbPassword = "" // clear input password
        await fetchDatabases()
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to verify connection parameters"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function handleCreateDatabase(e: Event) {
    e.preventDefault()
    createError = ""
    if (!newDbName.trim()) {
      createError = "Database name cannot be empty"
      return
    }

    if (createUser && !newDbUser.trim()) {
      createError = "Username cannot be empty"
      return
    }

    if (createUser && !newDbPassword.trim()) {
      createError = "Password cannot be empty"
      return
    }

    createLoading = true
    try {
      const response = await fetch("/api/databases", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ 
          name: newDbName.trim(),
          create_user: createUser,
          username: createUser ? newDbUser.trim() : "",
          password: createUser ? newDbPassword.trim() : "",
        }),
      })
      if (response.ok) {
        showCreateModal = false
        newDbName = ""
        newDbUser = ""
        newDbPassword = ""
        await fetchDatabases()
      } else {
        const errData = await response.json().catch(() => ({}))
        createError = errData.error || "Failed to create database"
      }
    } catch {
      createError = "Connection error"
    } finally {
      createLoading = false
    }
  }

  async function handleDeleteDatabase(name: string) {
    if (!confirm(`WARNING: Are you sure you want to permanently DELETE database '${name}'? All data will be lost forever.`)) return

    loading = true
    try {
      const response = await fetch(`/api/databases/${name}`, {
        method: "DELETE",
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        await fetchDatabases()
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to delete database"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function handleBackupDatabase(name: string) {
    backupDbName = name
    backupLoading = true
    try {
      const response = await fetch("/api/databases/backup", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ name }),
      })
      if (response.ok) {
        const data = await response.json()
        alert(`Backup completed successfully! Saved to: ${data.file}\n\nYou can access it in the 'Files' tab inside '/var/www/backups/'`)
      } else {
        const errData = await response.json().catch(() => ({}))
        alert(errData.error || "Backup failed")
      }
    } catch {
      alert("Network error during backup")
    } finally {
      backupLoading = false
      backupDbName = ""
    }
  }

  onMount(() => {
    fetchDatabases()
  })
</script>

<div class="space-y-6">
  {#if exploringDb}
    <DatabaseExplorer 
      {token} 
      dbName={exploringDb} 
      onBack={() => { exploringDb = ""; fetchDatabases(); }} 
    />
  {:else}
    <!-- Title Header -->
    <div class="flex items-center justify-between border-b border-border pb-4">
      <div>
        <h2 class="text-lg font-bold text-foreground flex items-center gap-2">
          <Database size={18} class="text-primary" />
          SQL Databases
        </h2>
        <p class="text-xs text-muted-foreground mt-0.5">Manage MySQL / MariaDB databases, execute backups, and edit connection settings.</p>
      </div>
      <div class="flex items-center gap-2">
        <button 
          on:click={() => showConfigCard = !showConfigCard}
          class="inline-flex h-9 items-center gap-1.5 rounded-xl border border-border bg-card px-3.5 text-xs font-semibold text-foreground hover:bg-secondary transition-colors"
        >
          <Settings size={13} />
          Credentials
        </button>
        <button 
          on:click={() => { 
            showCreateModal = true; 
            createError = ""; 
            newDbName = ""; 
            createUser = true; 
            newDbUser = ""; 
            newDbPassword = generateRandomPassword(); 
            customUserEdited = false; 
          }} 
          disabled={!configured}
          class="inline-flex h-9 items-center gap-1.5 rounded-xl bg-primary px-3.5 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-opacity disabled:opacity-50"
        >
          <Plus size={13} />
          Add Database
        </button>
        <button 
          on:click={fetchDatabases}
          disabled={loading}
          class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-card text-muted-foreground hover:bg-secondary transition-colors"
        >
          <RefreshCw size={14} class={loading ? "animate-spin" : ""} />
        </button>
      </div>
    </div>

    <!-- Error / Success Alert Box -->
    {#if error}
      <div class="rounded-xl bg-rose-500/10 p-3.5 text-xs text-rose-500 border border-rose-500/20 flex items-start gap-2.5">
        <AlertTriangle size={15} class="shrink-0 mt-0.5" />
        <div>
          <p class="font-semibold">Error / Warning</p>
          <p class="mt-0.5">{error}</p>
        </div>
      </div>
    {/if}

    {#if successMsg}
      <div class="rounded-xl bg-emerald-500/10 p-3.5 text-xs text-emerald-500 border border-emerald-500/20 flex items-start gap-2.5">
        <Check size={15} class="shrink-0 mt-0.5" />
        <div>
          <p class="font-semibold">Success</p>
          <p class="mt-0.5">{successMsg}</p>
        </div>
      </div>
    {/if}

    <!-- Connection Credentials Card -->
    {#if showConfigCard}
      <div class="rounded-2xl border border-border bg-card p-6 space-y-4">
        <div class="flex items-center justify-between border-b border-border pb-3">
          <h3 class="text-sm font-bold text-foreground flex items-center gap-2">
            <DatabaseZap size={15} class="text-primary" />
            MySQL Server Connection Settings
          </h3>
          {#if configured}
            <button 
              on:click={() => showConfigCard = false} 
              class="text-xs text-muted-foreground hover:text-foreground"
            >
              Hide
            </button>
          {/if}
        </div>

        <form on:submit={handleSaveConfig} class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <div class="space-y-1.5">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground">Host</label>
            <input 
              type="text" 
              bind:value={dbHost} 
              placeholder="127.0.0.1"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3 py-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
          </div>
          <div class="space-y-1.5">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground">Port</label>
            <input 
              type="text" 
              bind:value={dbPort} 
              placeholder="3306"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3 py-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
          </div>
          <div class="space-y-1.5">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground">Username</label>
            <input 
              type="text" 
              bind:value={dbUser} 
              placeholder="root"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3 py-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
          </div>
          <div class="space-y-1.5">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground">Password {hasPassword ? "(Saved: Yes)" : "(Saved: No)"}</label>
            <input 
              type="password" 
              bind:value={dbPassword} 
              placeholder="******"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3 py-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
            />
          </div>

          <div class="md:col-span-2 lg:col-span-4 flex items-center justify-between border-t border-border pt-4 mt-2">
            <p class="text-[10px] text-muted-foreground">Testing database connectivity will execute a test query on saving credentials.</p>
            <button 
              type="submit" 
              disabled={loading}
              class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-opacity disabled:opacity-50"
            >
              {loading ? "Verifying..." : "Verify & Save Credentials"}
            </button>
          </div>
        </form>
      </div>
    {/if}

    <!-- Databases Table -->
    <div class="rounded-2xl border border-border bg-card overflow-hidden">
      {#if !configured}
        <div class="flex flex-col items-center justify-center py-20 text-center px-6 space-y-3">
          <Database size={36} class="text-muted-foreground/30 animate-pulse" />
          <p class="text-sm font-semibold text-foreground">MySQL Credentials Required</p>
          <p class="text-xs text-muted-foreground max-w-sm">
            Please configure your MySQL/MariaDB server Host, User and Password in the settings panel above to fetch the database list.
          </p>
        </div>
      {:else if loading && databases.length === 0}
        <div class="flex flex-col items-center justify-center py-20 text-muted-foreground space-y-2">
          <RefreshCw size={24} class="animate-spin text-primary" />
          <span class="text-xs">Connecting to SQL server...</span>
        </div>
      {:else if databases.length === 0}
        <div class="flex flex-col items-center justify-center py-20 text-center px-6 space-y-2">
          <Database size={36} class="text-muted-foreground/30" />
          <p class="text-sm font-semibold text-foreground">No Custom Databases</p>
          <p class="text-xs text-muted-foreground">Click 'Add Database' to create a new database schema on your MySQL server.</p>
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-left text-xs border-collapse">
            <thead>
              <tr class="border-b border-border bg-secondary/20 text-muted-foreground font-semibold">
                <th class="px-6 py-3">Database Name</th>
                <th class="px-6 py-3 w-44">Charset / Collation</th>
                <th class="px-6 py-3 w-40">Server Address</th>
                <th class="px-6 py-3 w-32">Status</th>
                <th class="px-6 py-3 w-64 text-right">Operations</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              {#each databases as db (db)}
                <tr class="hover:bg-secondary/10 transition-colors">
                  <td class="px-6 py-3.5 font-bold text-foreground flex items-center gap-2">
                    <Database size={15} class="text-primary shrink-0" />
                    {db}
                  </td>
                  <td class="px-6 py-3.5 text-muted-foreground font-mono">
                    utf8mb4_unicode_ci
                  </td>
                  <td class="px-6 py-3.5 text-muted-foreground font-mono">
                    {dbHost}:{dbPort}
                  </td>
                  <td class="px-6 py-3.5">
                    <span class="inline-flex items-center gap-1 rounded bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-500">
                      Online
                    </span>
                  </td>
                  <td class="px-6 py-3.5 text-right">
                    <div class="flex items-center justify-end gap-2">
                      <button 
                        on:click={() => exploringDb = db}
                        class="inline-flex items-center gap-1 rounded-lg border border-border bg-card px-2.5 py-1.5 text-[10px] font-semibold text-foreground hover:bg-secondary transition-all"
                      >
                        <Eye size={10} />
                        Explore
                      </button>
                      <button 
                        on:click={() => handleBackupDatabase(db)}
                        disabled={backupLoading}
                        class="inline-flex items-center gap-1 rounded-lg border border-border bg-card px-2.5 py-1.5 text-[10px] font-semibold text-foreground hover:bg-secondary transition-all disabled:opacity-50"
                      >
                        <Download size={10} class={backupLoading && backupDbName === db ? "animate-bounce" : ""} />
                        {backupLoading && backupDbName === db ? "Backing up..." : "Backup DB"}
                      </button>
                      <button 
                        on:click={() => handleDeleteDatabase(db)}
                        class="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-rose-500 transition-all"
                        title="Drop Database"
                      >
                        <Trash2 size={13} />
                      </button>
                    </div>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>

    <!-- Info Alert Backup -->
    {#if configured}
      <div class="rounded-xl border border-border bg-card p-4 flex items-start gap-3">
        <Info size={15} class="text-primary shrink-0 mt-0.5" />
        <div class="text-[11px] text-muted-foreground space-y-1">
          <p class="font-semibold text-foreground">Synergistic Backup System Information:</p>
          <p>Database backups are stored as pure SQL text files inside the directory: <span class="font-mono bg-secondary/40 px-1 py-0.5 rounded text-foreground">/var/www/backups/</span>.</p>
          <p>You can instantly view, download, rename, or restore these backup files inside the <strong>Files</strong> explorer tab.</p>
        </div>
      </div>
    {/if}
  {/if}

  <!-- Create Database Modal -->
  {#if showCreateModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl overflow-hidden">
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <h3 class="text-sm font-bold text-foreground">Create New SQL Database</h3>
          <button on:click={() => showCreateModal = false} class="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>

        <form on:submit={handleCreateDatabase} class="space-y-4 p-6">
          {#if createError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20">
              {createError}
            </div>
          {/if}

          <!-- DB Name -->
          <div class="space-y-1.5">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground">Database Name</label>
            <input 
              type="text" 
              bind:value={newDbName}
              placeholder="e.g. blog_db"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
            <p class="text-[10px] text-muted-foreground">Only alphanumeric characters and underscores are allowed.</p>
          </div>

          <!-- Create User Toggle Checkbox -->
          <div class="flex items-center gap-2 py-1">
            <input 
              type="checkbox" 
              id="createUserCheckbox" 
              bind:checked={createUser}
              class="rounded border-border bg-secondary text-primary focus:ring-primary h-4 w-4"
            />
            <label for="createUserCheckbox" class="text-xs font-semibold text-foreground select-none cursor-pointer">
              Tự động tạo User cho Database (DB User)
            </label>
          </div>

          <!-- DB User & Password Fields -->
          {#if createUser}
            <div class="space-y-3 border-t border-border pt-3">
              <div class="space-y-1.5">
                <!-- svelte-ignore a11y-label-has-associated-control -->
                <label class="text-xs font-semibold text-muted-foreground">Username</label>
                <input 
                  type="text" 
                  bind:value={newDbUser}
                  on:input={() => customUserEdited = true}
                  placeholder="e.g. blog_db"
                  class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                  required
                />
              </div>

              <div class="space-y-1.5">
                <!-- svelte-ignore a11y-label-has-associated-control -->
                <label class="text-xs font-semibold text-muted-foreground flex items-center justify-between">
                  <span>Password</span>
                  <button 
                    type="button" 
                    on:click={() => newDbPassword = generateRandomPassword()}
                    class="text-[10px] text-primary hover:underline"
                  >
                    Tạo ngẫu nhiên
                  </button>
                </label>
                <input 
                  type="text" 
                  bind:value={newDbPassword}
                  placeholder="Password"
                  class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                  required
                />
              </div>
            </div>
          {/if}

          <div class="flex items-center justify-end gap-3 pt-4 border-t border-border">
            <button 
              type="button" 
              on:click={() => showCreateModal = false}
              class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors"
            >
              Cancel
            </button>
            <button 
              type="submit" 
              disabled={createLoading}
              class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50"
            >
              {createLoading ? "Creating..." : "Create Database"}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</div>
