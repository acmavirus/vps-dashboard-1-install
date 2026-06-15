<script lang="ts">
  import { onMount } from "svelte"
  import { toast } from "../../lib/toast"

  export let token: string | null

  interface PHPVersion {
    version: string
    status: string // running, stopped, not_installed
  }

  interface PHPSettings {
    memory_limit: string
    upload_max_filesize: string
    post_max_size: string
    max_execution_time: string
    display_errors: string
  }

  interface PHPExtension {
    name: string
    enabled: boolean
  }

  let versions: PHPVersion[] = []
  let selectedVersion = ""
  let activeSubTab: "settings" | "extensions" = "settings"

  let settings: PHPSettings = {
    memory_limit: "",
    upload_max_filesize: "",
    post_max_size: "",
    max_execution_time: "",
    display_errors: "",
  }

  let extensions: PHPExtension[] = []

  let loadingVersions = false
  let loadingSettings = false
  let loadingExtensions = false
  let savingSettings = false
  let togglingExtension = ""

  async function fetchVersions() {
    loadingVersions = true
    try {
      const response = await fetch("/api/php/versions", {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        versions = await response.json()
        const active = versions.find((v) => v.status === "running")
        if (active) {
          selectedVersion = active.version
          handleVersionChange()
        } else if (versions.length > 0) {
          selectedVersion = versions[0].version
          handleVersionChange()
        }
      }
    } catch (err) {
      console.error(err)
      toast.error("Error", "Failed to fetch PHP versions.")
    } finally {
      loadingVersions = false
    }
  }

  async function fetchSettings() {
    if (!selectedVersion) return
    loadingSettings = true
    try {
      const response = await fetch(`/api/php/settings?version=${selectedVersion}`, {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        settings = await response.json()
      }
    } catch (err) {
      console.error(err)
      toast.error("Error", "Failed to load PHP settings.")
    } finally {
      loadingSettings = false
    }
  }

  async function saveSettings() {
    savingSettings = true
    try {
      const response = await fetch("/api/php/settings", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          version: selectedVersion,
          settings,
        }),
      })

      if (response.ok) {
        toast.success("Success", `PHP ${selectedVersion} settings updated. PHP-FPM service restarted.`)
      } else {
        const data = await response.json()
        toast.error("Error", data.error || "Failed to save settings.")
      }
    } catch (err) {
      console.error(err)
      toast.error("Connection Error", "Failed to save settings.")
    } finally {
      savingSettings = false
    }
  }

  async function fetchExtensions() {
    if (!selectedVersion) return
    loadingExtensions = true
    try {
      const response = await fetch(`/api/php/extensions?version=${selectedVersion}`, {
        headers: { Authorization: token || "" },
      })
      if (response.ok) {
        extensions = await response.json()
      }
    } catch (err) {
      console.error(err)
      toast.error("Error", "Failed to load PHP extensions.")
    } finally {
      loadingExtensions = false
    }
  }

  async function toggleExtension(name: string, enable: boolean) {
    togglingExtension = name
    try {
      const response = await fetch("/api/php/extensions/toggle", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          version: selectedVersion,
          name,
          enable,
        }),
      })

      if (response.ok) {
        extensions = extensions.map((ext) =>
          ext.name === name ? { ...ext, enabled: enable } : ext
        )
        toast.success(
          "Success",
          `Extension "${name}" ${enable ? "enabled" : "disabled"}. PHP-FPM restarted.`
        )
      } else {
        const data = await response.json()
        toast.error("Error", data.error || "Failed to toggle extension.")
      }
    } catch (err) {
      console.error(err)
      toast.error("Connection Error", "Failed to toggle extension.")
    } finally {
      togglingExtension = ""
    }
  }

  function handleVersionChange() {
    if (activeSubTab === "settings") {
      fetchSettings()
    } else {
      fetchExtensions()
    }
  }

  onMount(() => {
    fetchVersions()
  })
</script>

<div class="h-full overflow-y-auto p-6 space-y-6">
  <!-- Version Selector Header Card -->
  <div class="rounded-xl border border-border bg-card p-6 shadow-sm">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-4">
      <div>
        <h3 class="text-base font-semibold">PHP Manager</h3>
        <p class="text-xs font-light text-muted-foreground mt-1">
          Manage PHP configurations, ini settings, and active extensions.
        </p>
      </div>

      <div class="flex items-center gap-3">
        <label for="php-version" class="text-xs font-semibold text-muted-foreground whitespace-nowrap">Active Version:</label>
        {#if loadingVersions}
          <div class="h-8 w-24 rounded bg-secondary animate-pulse"></div>
        {:else}
          <select
            id="php-version"
            bind:value={selectedVersion}
            on:change={handleVersionChange}
            class="rounded-lg border border-border bg-background px-3 py-1.5 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            {#each versions as v}
              <option value={v.version}>
                PHP {v.version} ({v.status})
              </option>
            {/each}
          </select>
        {/if}
      </div>
    </div>
  </div>

  {#if selectedVersion}
    <!-- Tabs Selector -->
    <div class="flex border-b border-border">
      <button
        type="button"
        on:click={() => {
          activeSubTab = "settings"
          fetchSettings()
        }}
        class="border-b-2 px-5 py-2.5 text-xs font-medium transition-colors {activeSubTab === 'settings' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        PHP.INI Settings
      </button>
      <button
        type="button"
        on:click={() => {
          activeSubTab = "extensions"
          fetchExtensions()
        }}
        class="border-b-2 px-5 py-2.5 text-xs font-medium transition-colors {activeSubTab === 'extensions' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
      >
        PHP Extensions
      </button>
    </div>

    <!-- SubTab Content -->
    {#if activeSubTab === 'settings'}
      <div class="rounded-xl border border-border bg-card p-6 shadow-sm">
        {#if loadingSettings}
          <div class="space-y-4">
            <div class="h-6 w-32 rounded bg-secondary animate-pulse"></div>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              {#each Array(4) as _}
                <div class="space-y-2">
                  <div class="h-4 w-24 rounded bg-secondary animate-pulse"></div>
                  <div class="h-10 w-full rounded bg-secondary animate-pulse"></div>
                </div>
              {/each}
            </div>
          </div>
        {:else}
          <form on:submit|preventDefault={saveSettings} class="space-y-6">
            <h4 class="text-sm font-semibold">Common Settings (php.ini)</h4>
            
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
              <!-- Memory Limit -->
              <div class="space-y-1.5">
                <label for="mem-limit" class="text-xs font-semibold text-muted-foreground">memory_limit</label>
                <input
                  id="mem-limit"
                  type="text"
                  bind:value={settings.memory_limit}
                  class="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
                  placeholder="e.g. 128M"
                  required
                />
                <span class="text-[10px] text-muted-foreground">Maximum amount of memory a script may consume.</span>
              </div>

              <!-- Max Upload Filesize -->
              <div class="space-y-1.5">
                <label for="upload-limit" class="text-xs font-semibold text-muted-foreground">upload_max_filesize</label>
                <input
                  id="upload-limit"
                  type="text"
                  bind:value={settings.upload_max_filesize}
                  class="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
                  placeholder="e.g. 20M"
                  required
                />
                <span class="text-[10px] text-muted-foreground">Maximum allowed size for uploaded files.</span>
              </div>

              <!-- Post Max Size -->
              <div class="space-y-1.5">
                <label for="post-limit" class="text-xs font-semibold text-muted-foreground">post_max_size</label>
                <input
                  id="post-limit"
                  type="text"
                  bind:value={settings.post_max_size}
                  class="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
                  placeholder="e.g. 20M"
                  required
                />
                <span class="text-[10px] text-muted-foreground">Maximum size of POST data that PHP will accept.</span>
              </div>

              <!-- Max Execution Time -->
              <div class="space-y-1.5">
                <label for="exec-time" class="text-xs font-semibold text-muted-foreground">max_execution_time (seconds)</label>
                <input
                  id="exec-time"
                  type="number"
                  bind:value={settings.max_execution_time}
                  class="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500 font-mono"
                  placeholder="e.g. 30"
                  required
                />
                <span class="text-[10px] text-muted-foreground">Maximum execution time of each script, in seconds.</span>
              </div>

              <!-- Display Errors -->
              <div class="space-y-1.5">
                <label for="display-errors" class="text-xs font-semibold text-muted-foreground">display_errors</label>
                <select
                  id="display-errors"
                  bind:value={settings.display_errors}
                  class="w-full rounded-lg border border-border bg-background px-3 py-2 text-xs focus:outline-none focus:ring-1 focus:ring-blue-500"
                >
                  <option value="On">On (Development)</option>
                  <option value="Off">Off (Production)</option>
                </select>
                <span class="text-[10px] text-muted-foreground">Print out errors to screen as part of the output.</span>
              </div>
            </div>

            <div class="flex justify-end pt-4">
              <button
                type="submit"
                disabled={savingSettings}
                class="rounded-lg bg-blue-600 px-4 py-2 text-xs font-medium text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
              >
                {savingSettings ? "Saving Settings..." : "Save Changes"}
              </button>
            </div>
          </form>
        {/if}
      </div>
    {:else if activeSubTab === 'extensions'}
      <div class="rounded-xl border border-border bg-card p-6 shadow-sm">
        <h4 class="text-sm font-semibold mb-4">Extensions List</h4>

        {#if loadingExtensions}
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            {#each Array(6) as _}
              <div class="h-14 rounded-lg bg-secondary animate-pulse"></div>
            {/each}
          </div>
        {:else}
          <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
            {#each extensions as ext}
              <div class="flex items-center justify-between p-3.5 rounded-lg border border-border bg-secondary/20">
                <div>
                  <span class="text-xs font-semibold font-mono">{ext.name}</span>
                  <p class="text-[9px] text-muted-foreground">PHP Extension Mod</p>
                </div>

                <button
                  type="button"
                  on:click={() => toggleExtension(ext.name, !ext.enabled)}
                  disabled={togglingExtension === ext.name}
                  class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none {ext.enabled ? 'bg-primary' : 'bg-muted'} {togglingExtension === ext.name ? 'opacity-50 cursor-not-allowed' : ''}"
                >
                  <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-background shadow ring-0 transition duration-200 ease-in-out {ext.enabled ? 'translate-x-4' : 'translate-x-0'}"></span>
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {/if}
</div>
