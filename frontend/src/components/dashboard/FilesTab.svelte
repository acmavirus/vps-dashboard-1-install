<script lang="ts">
  import { onMount } from "svelte"
  import { 
    Folder, 
    File, 
    FileText, 
    ChevronRight, 
    CornerUpLeft, 
    Plus, 
    Trash2, 
    Edit2, 
    RefreshCw, 
    X, 
    Save, 
    FolderPlus, 
    FilePlus, 
    ArrowRight 
  } from "lucide-svelte"
  import { toast } from "../../lib/toast"

  export let token: string | null = null

  interface FileItem {
    name: string
    size: number
    is_dir: boolean
    mod_time: string
    mode: string
  }

  let currentPath = "/home" // default to home root for convenience
  let files: FileItem[] = []
  let loading = true
  let error = ""

  // Editor modal state
  let showEditor = false
  let editorPath = ""
  let editorContent = ""
  let editorLoading = false
  let editorError = ""

  // Create modal state
  let showCreateModal = false
  let createType: "file" | "folder" = "file"
  let createName = ""
  let createError = ""
  let createLoading = false

  // Rename modal state
  let showRenameModal = false
  let renameOldPath = ""
  let renameOldName = ""
  let renameNewName = ""
  let renameError = ""
  let renameLoading = false

  // Path input jump state
  let pathInput = ""

  async function fetchFiles(path: string = currentPath) {
    loading = true
    error = ""
    try {
      const response = await fetch(`/api/files?path=${encodeURIComponent(path)}`, {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        const data = await response.json()
        currentPath = data.current_path
        files = data.files
        pathInput = currentPath
      } else {
        const errData = await response.json().catch(() => ({}))
        error = errData.error || "Failed to load directory"
      }
    } catch {
      error = "Connection error"
    } finally {
      loading = false
    }
  }

  function handleFolderClick(folderName: string) {
    let nextPath = currentPath
    if (nextPath === "/") {
      nextPath = "/" + folderName
    } else {
      nextPath = nextPath + "/" + folderName
    }
    fetchFiles(nextPath)
  }

  function handleGoUp() {
    if (currentPath === "/") return
    const parts = currentPath.split("/")
    parts.pop()
    const parentPath = parts.join("/") || "/"
    fetchFiles(parentPath)
  }

  function handleJumpPath() {
    if (!pathInput.trim()) return
    fetchFiles(pathInput.trim())
  }

  async function handleFileClick(file: FileItem) {
    // Open editor if it's not a directory
    editorPath = currentPath === "/" ? "/" + file.name : currentPath + "/" + file.name
    editorLoading = true
    editorError = ""
    showEditor = true
    editorContent = ""

    try {
      const response = await fetch(`/api/files/read?path=${encodeURIComponent(editorPath)}`, {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        const data = await response.json()
        editorContent = data.content
      } else {
        const errData = await response.json().catch(() => ({}))
        editorError = errData.error || "Cannot view binary or large file."
      }
    } catch {
      editorError = "Connection error"
    } finally {
      editorLoading = false
    }
  }

  async function handleSaveFile() {
    editorLoading = true
    editorError = ""
    try {
      const response = await fetch("/api/files/write", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          path: editorPath,
          content: editorContent,
        }),
      })
      if (response.ok) {
        toast.success("File saved", `${editorPath} has been saved successfully.`)
        showEditor = false
        fetchFiles()
      } else {
        const errData = await response.json().catch(() => ({}))
        editorError = errData.error || "Failed to save file"
      }
    } catch {
      editorError = "Connection error"
    } finally {
      editorLoading = false
    }
  }

  async function handleCreateItem(e: Event) {
    e.preventDefault()
    createError = ""
    if (!createName.trim()) {
      createError = "Name cannot be empty"
      return
    }

    createLoading = true
    const itemPath = currentPath === "/" ? "/" + createName.trim() : currentPath + "/" + createName.trim()

    try {
      const response = await fetch("/api/files/create", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          path: itemPath,
          is_dir: createType === "folder",
        }),
      })
      if (response.ok) {
        showCreateModal = false
        createName = ""
        fetchFiles()
      } else {
        const errData = await response.json().catch(() => ({}))
        createError = errData.error || "Failed to create item"
      }
    } catch {
      createError = "Connection error"
    } finally {
      createLoading = false
    }
  }

  async function handleDeleteItem(file: FileItem) {
    const itemPath = currentPath === "/" ? "/" + file.name : currentPath + "/" + file.name
    if (!confirm(`Are you sure you want to permanently delete '${file.name}'?`)) return

    try {
      const response = await fetch("/api/files/delete", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ path: itemPath }),
      })
      if (response.ok) {
        fetchFiles()
      } else {
        const errData = await response.json().catch(() => ({}))
        toast.error("Delete failed", errData.error || "Failed to delete item")
      }
    } catch {
      toast.error("Network error", "Could not reach the server.")
    }
  }

  function openRenameModal(file: FileItem) {
    renameOldName = file.name
    renameOldPath = currentPath === "/" ? "/" + file.name : currentPath + "/" + file.name
    renameNewName = file.name
    renameError = ""
    showRenameModal = true
  }

  async function handleRenameItem(e: Event) {
    e.preventDefault()
    renameError = ""
    if (!renameNewName.trim() || renameNewName.trim() === renameOldName) {
      showRenameModal = false
      return
    }

    renameLoading = true
    const newPath = currentPath === "/" ? "/" + renameNewName.trim() : currentPath + "/" + renameNewName.trim()

    try {
      const response = await fetch("/api/files/rename", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          old_path: renameOldPath,
          new_path: newPath,
        }),
      })
      if (response.ok) {
        showRenameModal = false
        fetchFiles()
      } else {
        const errData = await response.json().catch(() => ({}))
        renameError = errData.error || "Failed to rename item"
      }
    } catch {
      renameError = "Connection error"
    } finally {
      renameLoading = false
    }
  }

  function formatBytes(bytes: number) {
    if (bytes === 0) return "-"
    const k = 1024
    const sizes = ["Bytes", "KB", "MB", "GB"]
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
  }

  function formatTime(timeStr: string) {
    try {
      const d = new Date(timeStr)
      return d.toLocaleString()
    } catch {
      return timeStr
    }
  }

  onMount(() => {
    // Attempt starting at web root, if fails falls back to root '/'
    fetchFiles("/home").catch(() => fetchFiles("/"))
  })
</script>

<div class="space-y-6">
  <!-- Title Header -->
  <div class="flex items-center justify-between border-b border-border pb-4">
    <div>
      <h2 class="text-lg font-bold text-foreground flex items-center gap-2">
        <Folder size={18} class="text-primary" />
        Web File Manager
      </h2>
      <p class="text-xs text-muted-foreground mt-0.5">Browse directory tree, edit system configuration files directly.</p>
    </div>
    <div class="flex items-center gap-2">
      <!-- Quick navigation buttons -->
      <button 
        on:click={() => fetchFiles("/home")} 
        class="rounded-lg border border-border bg-card px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
      >
        /home
      </button>
      <button 
        on:click={() => fetchFiles("/etc/nginx")} 
        class="rounded-lg border border-border bg-card px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
      >
        /etc/nginx
      </button>
      <button 
        on:click={() => fetchFiles("/root")} 
        class="rounded-lg border border-border bg-card px-2.5 py-1.5 text-xs text-muted-foreground hover:text-foreground hover:bg-secondary transition-colors"
      >
        /root
      </button>
    </div>
  </div>

  <!-- Path Navigation Box & Actions Toolbar -->
  <div class="flex flex-col md:flex-row gap-3">
    <!-- Path input and jump button -->
    <div class="flex-1 flex items-center gap-1 bg-card border border-border rounded-xl px-3 py-1.5 focus-within:ring-1 focus-within:ring-primary">
      <span class="text-xs text-muted-foreground font-mono shrink-0">Path:</span>
      <input 
        type="text" 
        bind:value={pathInput} 
        on:keydown={(e) => e.key === 'Enter' && handleJumpPath()}
        class="flex-1 bg-transparent text-xs text-foreground focus:outline-none font-mono"
      />
      <button 
        on:click={handleJumpPath} 
        class="p-1 rounded text-muted-foreground hover:bg-secondary hover:text-foreground transition-colors"
      >
        <ArrowRight size={14} />
      </button>
    </div>

    <!-- Actions -->
    <div class="flex items-center gap-2">
      <button 
        on:click={handleGoUp} 
        disabled={currentPath === "/"}
        class="inline-flex h-9 items-center gap-1.5 rounded-xl border border-border bg-card px-3.5 text-xs font-semibold text-foreground hover:bg-secondary transition-colors disabled:opacity-40"
      >
        <CornerUpLeft size={13} />
        Up
      </button>
      <button 
        on:click={() => { createType = "file"; createName = ""; createError = ""; showCreateModal = true; }} 
        class="inline-flex h-9 items-center gap-1.5 rounded-xl border border-border bg-card px-3.5 text-xs font-semibold text-foreground hover:bg-secondary transition-colors"
      >
        <FilePlus size={13} />
        New File
      </button>
      <button 
        on:click={() => { createType = "folder"; createName = ""; createError = ""; showCreateModal = true; }} 
        class="inline-flex h-9 items-center gap-1.5 rounded-xl border border-border bg-card px-3.5 text-xs font-semibold text-foreground hover:bg-secondary transition-colors"
      >
        <FolderPlus size={13} />
        New Folder
      </button>
      <button 
        on:click={() => fetchFiles()}
        disabled={loading}
        class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-card text-muted-foreground hover:bg-secondary transition-colors"
      >
        <RefreshCw size={14} class={loading ? "animate-spin" : ""} />
      </button>
    </div>
  </div>

  <!-- Main Directory Listing Card -->
  <div class="rounded-2xl border border-border bg-card overflow-hidden">
    {#if loading}
      <div class="flex flex-col items-center justify-center py-20 text-muted-foreground space-y-2">
        <RefreshCw size={24} class="animate-spin text-primary" />
        <span class="text-xs">Loading directory contents...</span>
      </div>
    {:else if error}
      <div class="flex flex-col items-center justify-center py-20 text-center px-4">
        <p class="text-xs text-rose-500 font-semibold">{error}</p>
        <button on:click={() => fetchFiles()} class="mt-3 text-xs text-primary underline">Retry</button>
      </div>
    {:else if files.length === 0}
      <div class="flex flex-col items-center justify-center py-20 text-center px-6 space-y-2">
        <Folder size={36} class="text-muted-foreground/30" />
        <p class="text-sm font-semibold text-foreground">Empty Directory</p>
        <p class="text-xs text-muted-foreground">This folder contains no files or folders.</p>
      </div>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs border-collapse">
          <thead>
            <tr class="border-b border-border bg-secondary/20 text-muted-foreground font-semibold">
              <th class="px-6 py-3">Name</th>
              <th class="px-6 py-3 w-32">Size</th>
              <th class="px-6 py-3 w-40">Modified</th>
              <th class="px-6 py-3 w-28">Permissions</th>
              <th class="px-6 py-3 w-24 text-right">Operations</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            {#each files as file (file.name)}
              <tr class="hover:bg-secondary/10 transition-colors">
                <td class="px-6 py-3.5">
                  <div class="flex items-center gap-3">
                    {#if file.is_dir}
                      <Folder size={16} class="text-amber-500 fill-amber-500/10 shrink-0" />
                      <button 
                        on:click={() => handleFolderClick(file.name)} 
                        class="font-medium text-foreground hover:text-primary text-left focus:outline-none hover:underline"
                      >
                        {file.name}
                      </button>
                    {:else}
                      <FileText size={16} class="text-blue-500 shrink-0" />
                      <button 
                        on:click={() => handleFileClick(file)} 
                        class="font-normal text-foreground hover:text-primary text-left focus:outline-none hover:underline"
                      >
                        {file.name}
                      </button>
                    {/if}
                  </div>
                </td>
                <td class="px-6 py-3.5 text-muted-foreground font-mono">
                  {formatBytes(file.size)}
                </td>
                <td class="px-6 py-3.5 text-muted-foreground">
                  {formatTime(file.mod_time)}
                </td>
                <td class="px-6 py-3.5 text-muted-foreground font-mono">
                  {file.mode}
                </td>
                <td class="px-6 py-3.5 text-right">
                  <div class="flex items-center justify-end gap-1">
                    {#if !file.is_dir}
                      <button 
                        on:click={() => handleFileClick(file)}
                        class="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-primary transition-all"
                        title="Edit File"
                      >
                        <Edit2 size={13} />
                      </button>
                    {/if}
                    <button 
                      on:click={() => openRenameModal(file)}
                      class="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-primary transition-all"
                      title="Rename"
                    >
                      <ChevronRight size={13} class="rotate-45" />
                    </button>
                    <button 
                      on:click={() => handleDeleteItem(file)}
                      class="rounded p-1.5 text-muted-foreground hover:bg-secondary hover:text-rose-500 transition-all"
                      title="Delete"
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

  <!-- Monaco-like Web Editor Modal (Popup) -->
  {#if showEditor}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/75 p-4">
      <div class="w-full max-w-5xl h-[85vh] rounded-2xl border border-border bg-card shadow-2xl overflow-hidden flex flex-col">
        <!-- Editor Header -->
        <div class="flex items-center justify-between border-b border-border px-6 py-4 bg-secondary/10">
          <div>
            <h3 class="text-sm font-bold text-foreground flex items-center gap-2">
              <FileText size={16} class="text-primary" />
              File Editor
            </h3>
            <p class="text-[10px] text-muted-foreground font-mono mt-0.5">{editorPath}</p>
          </div>
          <button 
            on:click={() => showEditor = false}
            class="text-muted-foreground hover:text-foreground transition-colors"
          >
            <X size={18} />
          </button>
        </div>

        <!-- Editor Content -->
        <div class="flex-1 p-6 flex flex-col min-h-0 bg-zinc-950">
          {#if editorLoading}
            <div class="flex-1 flex flex-col items-center justify-center text-muted-foreground space-y-2">
              <RefreshCw size={24} class="animate-spin text-primary" />
              <span class="text-xs">Loading/Saving content...</span>
            </div>
          {:else}
            {#if editorError}
              <div class="mb-4 rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20">
                {editorError}
              </div>
            {/if}
            <textarea 
              bind:value={editorContent}
              class="flex-1 w-full bg-transparent text-xs text-zinc-100 font-mono focus:outline-none resize-none leading-relaxed select-text"
              placeholder="Start typing..."
              spellcheck="false"
            ></textarea>
          {/if}
        </div>

        <!-- Editor Footer -->
        <div class="flex items-center justify-between border-t border-border px-6 py-4 bg-secondary/10">
          <span class="text-xs text-muted-foreground font-mono">Lines: {editorContent.split('\n').length}</span>
          <div class="flex items-center gap-3">
            <button 
              type="button" 
              on:click={() => showEditor = false}
              class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors"
            >
              Close
            </button>
            <button 
              type="button" 
              on:click={handleSaveFile}
              disabled={editorLoading}
              class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-opacity disabled:opacity-50"
            >
              <Save size={12} />
              Save File
            </button>
          </div>
        </div>
      </div>
    </div>
  {/if}

  <!-- Create File/Folder Modal -->
  {#if showCreateModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl overflow-hidden">
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <h3 class="text-sm font-bold text-foreground">
            Create New {createType === "file" ? "File" : "Folder"}
          </h3>
          <button on:click={() => showCreateModal = false} class="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>

        <form on:submit={handleCreateItem} class="space-y-4 p-6">
          {#if createError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20">
              {createError}
            </div>
          {/if}

          <div class="space-y-2">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground">Name</label>
            <input 
              type="text" 
              bind:value={createName}
              placeholder={createType === "file" ? "e.g. index.html" : "e.g. public"}
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
              required
            />
          </div>

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
              {createLoading ? "Creating..." : "Create"}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- Rename Modal -->
  {#if showRenameModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
      <div class="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl overflow-hidden">
        <div class="flex items-center justify-between border-b border-border px-6 py-4">
          <h3 class="text-sm font-bold text-foreground">Rename Item</h3>
          <button on:click={() => showRenameModal = false} class="text-muted-foreground hover:text-foreground">
            <X size={16} />
          </button>
        </div>

        <form on:submit={handleRenameItem} class="space-y-4 p-6">
          {#if renameError}
            <div class="rounded-lg bg-rose-500/10 p-3 text-xs text-rose-500 border border-rose-500/20">
              {renameError}
            </div>
          {/if}

          <div class="space-y-2">
            <!-- svelte-ignore a11y-label-has-associated-control -->
            <label class="text-xs font-semibold text-muted-foreground">New Name</label>
            <input 
              type="text" 
              bind:value={renameNewName}
              class="w-full rounded-lg border border-border bg-secondary/20 px-3.5 py-2.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
              required
            />
          </div>

          <div class="flex items-center justify-end gap-3 pt-4 border-t border-border">
            <button 
              type="button" 
              on:click={() => showRenameModal = false}
              class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted-foreground hover:text-foreground transition-colors"
            >
              Cancel
            </button>
            <button 
              type="submit" 
              disabled={renameLoading}
              class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 disabled:opacity-50"
            >
              {renameLoading ? "Renaming..." : "Rename"}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</div>
