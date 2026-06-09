<script lang="ts">
  import { onMount } from "svelte"
  import { 
    Key, 
    Plus, 
    Trash2, 
    RefreshCw, 
    X, 
    Folder, 
    Lock, 
    User, 
    Check, 
    AlertTriangle,
    Eye,
    EyeOff,
    Copy
  } from "lucide-svelte"

  export let token: string | null = null

  interface FtpUser {
    username: string
    path: string
    status: string // "active" | "disabled"
  }

  let users: FtpUser[] = []
  let loading = true
  let error = ""
  let successMsg = ""

  // Add user modal
  let showAddModal = false
  let addUsername = ""
  let addPassword = ""
  let addPath = "/home/"
  let addLoading = false
  let addError = ""

  // Password modal
  let showPasswordModal = false
  let selectedUser: FtpUser | null = null
  let newPassword = ""
  let passwordLoading = false
  let passwordError = ""

  // UI state
  let copiedField = ""

  function generateRandomPassword() {
    const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
    let pass = ""
    for (let i = 0; i < 14; i++) {
      pass += chars.charAt(Math.floor(Math.random() * chars.length))
    }
    return pass
  }

  async function fetchFtpUsers() {
    loading = true
    error = ""
    successMsg = ""
    try {
      const response = await fetch("/api/ftp", {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        users = await response.json() || []
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to load FTP accounts"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function handleAddFtp(e: Event) {
    e.preventDefault()
    addError = ""
    if (!addUsername.trim()) {
      addError = "Username cannot be empty"
      return
    }
    if (addPassword.length < 6) {
      addError = "Password must be at least 6 characters"
      return
    }
    if (!addPath.trim()) {
      addError = "Path cannot be empty"
      return
    }

    addLoading = true
    try {
      const response = await fetch("/api/ftp/add", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          username: addUsername.trim(),
          password: addPassword,
          path: addPath.trim(),
        }),
      })
      if (response.ok) {
        showAddModal = false
        addUsername = ""
        addPassword = ""
        addPath = "/home/"
        successMsg = "FTP account created successfully!"
        await fetchFtpUsers()
      } else {
        const errData = await response.json().catch(() => ({}))
        addError = errData.error || "Failed to create FTP account"
      }
    } catch {
      addError = "Connection error"
    } finally {
      addLoading = false
    }
  }

  async function handleDeleteFtp(username: string) {
    if (!confirm(`Are you sure you want to delete FTP user '${username}'?`)) return

    loading = true
    error = ""
    try {
      const response = await fetch("/api/ftp/delete", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ username }),
      })
      if (response.ok) {
        successMsg = `FTP user '${username}' deleted successfully.`
        await fetchFtpUsers()
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to delete FTP user"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function handleToggleFtp(username: string) {
    loading = true
    error = ""
    try {
      const response = await fetch("/api/ftp/toggle", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ username }),
      })
      if (response.ok) {
        await fetchFtpUsers()
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to toggle FTP status"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  async function handleChangePassword(e: Event) {
    e.preventDefault()
    passwordError = ""
    if (!selectedUser) return
    if (newPassword.length < 6) {
      passwordError = "Password must be at least 6 characters"
      return
    }

    passwordLoading = true
    try {
      const response = await fetch("/api/ftp/password", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          username: selectedUser.username,
          password: newPassword,
        }),
      })
      if (response.ok) {
        showPasswordModal = false
        newPassword = ""
        successMsg = `Password for '${selectedUser.username}' updated successfully!`
        await fetchFtpUsers()
      } else {
        const errData = await response.json().catch(() => ({}))
        passwordError = errData.error || "Failed to change password"
      }
    } catch {
      passwordError = "Connection error"
    } finally {
      passwordLoading = false
    }
  }

  function handleCopy(text: string, field: string) {
    navigator.clipboard.writeText(text)
    copiedField = field
    setTimeout(() => {
      copiedField = ""
    }, 2000)
  }

  onMount(() => {
    fetchFtpUsers()
  })
</script>

<div class="space-y-6">
  <!-- Title Header -->
  <div class="flex items-center justify-between border-b border-border pb-4">
    <div>
      <h2 class="text-lg font-bold text-foreground flex items-center gap-2">
        <Key size={18} class="text-primary" />
        FTP Accounts
      </h2>
      <p class="text-xs text-muted-foreground mt-0.5">Manage FTP credentials for isolated folder access. Ideal for web developers.</p>
    </div>
    <div class="flex items-center gap-2">
      <button 
        on:click={() => { 
          showAddModal = true; 
          addError = ""; 
          addUsername = ""; 
          addPassword = generateRandomPassword(); 
          addPath = "/home/"; 
        }}
        class="inline-flex h-9 items-center gap-1.5 rounded-xl bg-primary px-3.5 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-opacity"
      >
        <Plus size={13} />
        Add FTP Account
      </button>
      <button 
        on:click={fetchFtpUsers}
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
        <p class="font-semibold">Error</p>
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

  <!-- FTP Users Table -->
  <div class="rounded-2xl border border-border bg-card overflow-hidden">
    {#if loading && users.length === 0}
      <div class="flex flex-col items-center justify-center py-20 text-muted-foreground space-y-2">
        <RefreshCw size={24} class="animate-spin text-primary" />
        <span class="text-xs">Loading FTP accounts...</span>
      </div>
    {:else if users.length === 0}
      <div class="flex flex-col items-center justify-center py-20 text-center px-6 space-y-2">
        <Key size={36} class="text-muted-foreground/30" />
        <p class="text-sm font-semibold text-foreground">No FTP Accounts</p>
        <p class="text-xs text-muted-foreground">Click 'Add FTP Account' to configure virtual FTP directories.</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs border-collapse">
          <thead>
            <tr class="border-b border-border bg-secondary/20 text-muted-foreground font-semibold">
              <th class="px-6 py-3">Username</th>
              <th class="px-6 py-3">Directory Path</th>
              <th class="px-6 py-3 w-32">Status</th>
              <th class="px-6 py-3 w-64 text-right">Operations</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each users as user (user.username)}
              <tr class="hover:bg-secondary/10 transition-colors {user.status === 'disabled' ? 'opacity-65' : ''}">
                <td class="px-6 py-3.5 font-bold text-foreground">
                  <div class="flex items-center gap-2">
                    <User size={15} class="text-primary shrink-0" />
                    <span class="font-mono">{user.username}</span>
                    <button 
                      on:click={() => handleCopy(user.username, 'u_' + user.username)}
                      class="text-muted-foreground hover:text-foreground transition-colors p-1"
                      title="Copy Username"
                    >
                      {#if copiedField === 'u_' + user.username}
                        <Check size={11} class="text-emerald-500" />
                      {:else}
                        <Copy size={11} />
                      {/if}
                    </button>
                  </div>
                </td>
                <td class="px-6 py-3.5 text-muted-foreground font-mono">
                  <div class="flex items-center gap-2">
                    <Folder size={14} class="text-amber-500/70 shrink-0" />
                    <span>{user.path}</span>
                    <button 
                      on:click={() => handleCopy(user.path, 'p_' + user.username)}
                      class="text-muted-foreground hover:text-foreground transition-colors p-1"
                      title="Copy Path"
                    >
                      {#if copiedField === 'p_' + user.username}
                        <Check size={11} class="text-emerald-500" />
                      {:else}
                        <Copy size={11} />
                      {/if}
                    </button>
                  </div>
                </td>
                <td class="px-6 py-3.5">
                  {#if user.status === "active"}
                    <span class="inline-flex items-center gap-1 rounded bg-emerald-500/10 px-2 py-0.5 text-[10px] font-semibold text-emerald-500">
                      Active
                    </span>
                  {:else}
                    <span class="inline-flex items-center gap-1 rounded bg-zinc-500/10 px-2 py-0.5 text-[10px] font-semibold text-zinc-400">
                      Disabled
                    </span>
                  {/if}
                </td>
                <td class="px-6 py-3.5 text-right">
                  <div class="flex items-center justify-end gap-2">
                    <button 
                      on:click={() => handleToggleFtp(user.username)}
                      class="inline-flex items-center rounded-lg border border-border bg-card px-2.5 py-1.5 text-[10px] font-semibold text-foreground hover:bg-secondary transition-all"
                    >
                      {user.status === "active" ? "Disable" : "Enable"}
                    </button>
                    <button 
                      on:click={() => {
                        selectedUser = user;
                        newPassword = generateRandomPassword();
                        showPasswordModal = true;
                        passwordError = "";
                      }}
                      class="inline-flex items-center rounded-lg border border-border bg-card px-2.5 py-1.5 text-[10px] font-semibold text-foreground hover:bg-secondary transition-all"
                    >
                      Password
                    </button>
                    <button 
                      on:click={() => handleDeleteFtp(user.username)}
                      class="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-rose-500 transition-all"
                      title="Delete User"
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

  <!-- Add User Modal -->
  {#if showAddModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl overflow-hidden">
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <h3 class="text-sm font-bold text-foreground flex items-center gap-1.5">
            <Key size={14} class="text-primary" />
            Add FTP Account
          </h3>
          <button on:click={() => showAddModal = false} class="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>

        <form on:submit={handleAddFtp} class="space-y-4 p-6">
          {#if addError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20">
              {addError}
            </div>
          {/if}

          <!-- Username -->
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-muted-foreground flex items-center gap-1">
              <User size={12} />
              Username
            </label>
            <input 
              type="text" 
              bind:value={addUsername}
              placeholder="e.g. ftp_user"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
            <p class="text-[10px] text-muted-foreground">Only alphanumeric, dashes, and underscores allowed.</p>
          </div>

          <!-- Password -->
          <div class="space-y-1.5">
            <div class="flex items-center justify-between">
              <label class="text-xs font-semibold text-muted-foreground flex items-center gap-1">
                <Lock size={12} />
                Password
              </label>
              <button 
                type="button" 
                on:click={() => addPassword = generateRandomPassword()}
                class="text-[10px] text-primary hover:underline font-semibold"
              >
                Generate
              </button>
            </div>
            <input 
              type="text" 
              bind:value={addPassword}
              placeholder="Password"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
          </div>

          <!-- Directory Path -->
          <div class="space-y-1.5">
            <label class="text-xs font-semibold text-muted-foreground flex items-center gap-1">
              <Folder size={12} />
              Directory Path
            </label>
            <input 
              type="text" 
              bind:value={addPath}
              placeholder="/home/my-domain.com"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
            <p class="text-[10px] text-muted-foreground">Absolute path. Users are locked (chrooted) inside this directory.</p>
          </div>

          <div class="flex items-center justify-end gap-3 pt-4 border-t border-border">
            <button 
              type="button" 
              on:click={() => showAddModal = false}
              class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors"
            >
              Cancel
            </button>
            <button 
              type="submit" 
              disabled={addLoading}
              class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50"
            >
              {addLoading ? "Creating..." : "Create Account"}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- Password Change Modal -->
  {#if showPasswordModal && selectedUser}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl overflow-hidden">
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <h3 class="text-sm font-bold text-foreground">Change Password</h3>
          <button on:click={() => showPasswordModal = false} class="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>

        <form on:submit={handleChangePassword} class="space-y-4 p-6">
          <p class="text-xs text-muted-foreground">Changing password for virtual FTP user: <span class="font-bold text-foreground font-mono">{selectedUser.username}</span></p>

          {#if passwordError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20">
              {passwordError}
            </div>
          {/if}

          <!-- New Password -->
          <div class="space-y-1.5">
            <div class="flex items-center justify-between">
              <label class="text-xs font-semibold text-muted-foreground flex items-center gap-1">
                <Lock size={12} />
                New Password
              </label>
              <button 
                type="button" 
                on:click={() => newPassword = generateRandomPassword()}
                class="text-[10px] text-primary hover:underline font-semibold"
              >
                Generate
              </button>
            </div>
            <input 
              type="text" 
              bind:value={newPassword}
              placeholder="New Password"
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
              required
            />
          </div>

          <div class="flex items-center justify-end gap-3 pt-4 border-t border-border">
            <button 
              type="button" 
              on:click={() => showPasswordModal = false}
              class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors"
            >
              Cancel
            </button>
            <button 
              type="submit" 
              disabled={passwordLoading}
              class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50"
            >
              {passwordLoading ? "Saving..." : "Change Password"}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</div>
