<script lang="ts">
  import { onMount } from "svelte"
  import {
    ArrowLeft,
    Database,
    Table,
    FileText,
    Terminal,
    Search,
    ChevronLeft,
    ChevronRight,
    Play,
    AlertTriangle,
    Check,
    RefreshCw,
    Info,
    Eye
  } from "lucide-svelte"

  export let token: string | null = null
  export let dbName: string = ""
  export let onBack: () => void = () => {}

  // Database metadata
  let tables: { name: string; engine: string; rows: number; data_size: number; collation: string; comment: string }[] = []
  let selectedTable: string = ""
  let searchTableQuery: string = ""
  
  // Loading states
  let tablesLoading = true
  let tablesError = ""
  let activeTab: "browse" | "structure" | "sql" = "browse"

  // Browse tab states
  let columns: string[] = []
  let rows: any[][] = []
  let totalRows = 0
  let limit = 50
  let offset = 0
  let dataLoading = false
  let dataError = ""

  // Structure tab states
  let structureColumns: { field: string; type: string; null: string; key: string; default: string; extra: string; comment: string }[] = []
  let structureLoading = false
  let structureError = ""

  // SQL tab states
  let sqlQuery = ""
  let sqlRunning = false
  let sqlResult: any = null
  let sqlError = ""
  let sqlSuccessMessage = ""

  // Filter tables
  $: filteredTables = tables.filter(t => t.name.toLowerCase().includes(searchTableQuery.toLowerCase()))

  async function fetchTables() {
    tablesLoading = true
    tablesError = ""
    try {
      const response = await fetch(`/api/databases/${dbName}/tables`, {
        headers: { Authorization: token || "" }
      })
      if (response.ok) {
        tables = await response.json()
        if (tables.length > 0 && !selectedTable) {
          selectTable(tables[0].name)
        }
      } else {
        const err = await response.json().catch(() => ({}))
        tablesError = err.error || "Failed to load database tables"
      }
    } catch {
      tablesError = "Network error loading tables"
    } finally {
      tablesLoading = false
    }
  }

  function formatBytes(bytes: number) {
    if (bytes === 0) return "0 Bytes"
    const k = 1024
    const sizes = ["Bytes", "KB", "MB", "GB"]
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
  }

  async function fetchTableData() {
    if (!selectedTable) return
    dataLoading = true
    dataError = ""
    try {
      const response = await fetch(`/api/databases/${dbName}/tables/${selectedTable}/data?limit=${limit}&offset=${offset}`, {
        headers: { Authorization: token || "" }
      })
      if (response.ok) {
        const result = await response.json()
        columns = result.columns || []
        rows = result.rows || []
        totalRows = result.total || 0
      } else {
        const err = await response.json().catch(() => ({}))
        dataError = err.error || "Failed to load table rows"
      }
    } catch {
      dataError = "Network error loading table data"
    } finally {
      dataLoading = false
    }
  }

  async function fetchTableStructure() {
    if (!selectedTable) return
    structureLoading = true
    structureError = ""
    try {
      const response = await fetch(`/api/databases/${dbName}/tables/${selectedTable}/columns`, {
        headers: { Authorization: token || "" }
      })
      if (response.ok) {
        structureColumns = await response.json()
      } else {
        const err = await response.json().catch(() => ({}))
        structureError = err.error || "Failed to load table structure"
      }
    } catch {
      structureError = "Network error loading structure"
    } finally {
      structureLoading = false
    }
  }

  function selectTable(tableName: string) {
    selectedTable = tableName
    offset = 0
    // Load whichever tab is active
    if (activeTab === "browse") {
      fetchTableData()
    } else if (activeTab === "structure") {
      fetchTableStructure()
    } else if (activeTab === "sql" && !sqlQuery) {
      sqlQuery = `SELECT * FROM \`${selectedTable}\` LIMIT 100;`
    }
  }

  function switchTab(tab: "browse" | "structure" | "sql") {
    activeTab = tab
    if (tab === "browse") {
      fetchTableData()
    } else if (tab === "structure") {
      fetchTableStructure()
    } else if (tab === "sql" && !sqlQuery && selectedTable) {
      sqlQuery = `SELECT * FROM \`${selectedTable}\` LIMIT 100;`
    }
  }

  // Handle SQL execution
  async function runSQLQuery() {
    if (!sqlQuery.trim()) return
    sqlRunning = true
    sqlError = ""
    sqlResult = null
    sqlSuccessMessage = ""

    try {
      const response = await fetch(`/api/databases/${dbName}/query`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || ""
        },
        body: JSON.stringify({ query: sqlQuery })
      })

      const data = await response.json()
      if (response.ok) {
        sqlResult = data
        if (data.type === "exec") {
          sqlSuccessMessage = `Query OK, ${data.rows_affected} rows affected. Last inserted ID: ${data.last_insert_id}.`
        }
      } else {
        sqlError = data.error || "Failed to execute SQL query"
      }
    } catch {
      sqlError = "Connection error while running SQL query"
    } finally {
      sqlRunning = false
    }
  }

  // Pagination controls
  function prevPage() {
    if (offset >= limit) {
      offset -= limit
      fetchTableData()
    }
  }

  function nextPage() {
    if (offset + limit < totalRows) {
      offset += limit
      fetchTableData()
    }
  }

  function appendSQLTemplate(template: string) {
    sqlQuery = template
    switchTab("sql")
  }

  onMount(() => {
    fetchTables()
  })
</script>

<div class="space-y-6">
  <!-- Header with Back Button -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between border-b border-border pb-4 gap-4">
    <div class="flex items-center gap-3">
      <button 
        on:click={onBack}
        class="inline-flex h-9 w-9 items-center justify-center rounded-xl border border-border bg-card text-muted-foreground hover:bg-secondary transition-colors"
      >
        <ArrowLeft size={16} />
      </button>
      <div>
        <h2 class="text-lg font-bold text-foreground flex items-center gap-2">
          <Database size={18} class="text-primary" />
          Database: <span class="font-mono text-primary">{dbName}</span>
        </h2>
        <p class="text-xs text-muted-foreground mt-0.5">Explore schema, run queries, and browse raw data.</p>
      </div>
    </div>
    <div class="flex items-center gap-2">
      <button 
        on:click={fetchTables}
        class="inline-flex h-9 items-center gap-1.5 rounded-xl border border-border bg-card px-3.5 text-xs font-semibold text-foreground hover:bg-secondary transition-colors"
      >
        <RefreshCw size={13} />
        Refresh Database
      </button>
    </div>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-4 gap-6">
    <!-- Left Sidebar: Tables List -->
    <div class="lg:col-span-1 flex flex-col gap-4">
      <div class="rounded-2xl border border-border bg-card p-4 space-y-3">
        <h3 class="text-xs font-bold text-foreground uppercase tracking-wider">Tables list</h3>
        <div class="relative">
          <Search size={14} class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <input 
            type="text" 
            bind:value={searchTableQuery}
            placeholder="Search tables..."
            class="w-full rounded-lg border border-border bg-secondary/20 pl-9 pr-3 py-2 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
          />
        </div>

        {#if tablesLoading}
          <div class="flex justify-center py-10">
            <RefreshCw size={18} class="animate-spin text-primary" />
          </div>
        {:else if tablesError}
          <div class="text-xs text-rose-500 bg-rose-500/10 p-3 rounded-xl border border-rose-500/20">
            {tablesError}
          </div>
        {:else if filteredTables.length === 0}
          <div class="text-xs text-muted-foreground text-center py-10">
            No tables found.
          </div>
        {:else}
          <div class="space-y-1 max-h-[500px] overflow-y-auto pr-1">
            {#each filteredTables as t}
              <button 
                on:click={() => selectTable(t.name)}
                class="w-full rounded-lg px-3 py-2 text-left transition-colors flex items-center justify-between text-xs {selectedTable === t.name ? 'bg-primary text-primary-foreground font-medium' : 'text-muted-foreground hover:bg-secondary/40 hover:text-foreground'}"
              >
                <span class="flex items-center gap-2 overflow-hidden truncate">
                  <Table size={13} class="shrink-0" />
                  <span class="truncate">{t.name}</span>
                </span>
                {#if t.rows > 0 || t.engine}
                  <span class="text-[9px] bg-secondary/80 dark:bg-secondary px-1.5 py-0.2 rounded font-mono text-muted-foreground shrink-0 {selectedTable === t.name ? 'text-primary-foreground/90' : ''}">
                    {t.rows.toLocaleString()}
                  </span>
                {/if}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    <!-- Right Panel: Data, Structure, and SQL Runner -->
    <div class="lg:col-span-3 flex flex-col gap-6">
      {#if !selectedTable && !tablesLoading}
        <div class="rounded-2xl border border-border bg-card p-12 text-center flex flex-col items-center justify-center space-y-3">
          <Table size={36} class="text-muted-foreground/30" />
          <h3 class="text-sm font-semibold text-foreground">Select a table</h3>
          <p class="text-xs text-muted-foreground max-w-sm">Please choose a table from the sidebar to browse its content, examine structure, or run queries.</p>
        </div>
      {:else}
        <!-- Tab Select Buttons -->
        <div class="flex items-center justify-between border-b border-border pb-1">
          <div class="flex items-center gap-2">
            <button
              on:click={() => switchTab("browse")}
              class="relative py-2.5 px-4 text-xs font-semibold transition-all border-b-2 {activeTab === "browse" ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground"}"
            >
              <span class="flex items-center gap-1.5">
                <Eye size={13} />
                Browse Data
              </span>
            </button>
            <button
              on:click={() => switchTab("structure")}
              class="relative py-2.5 px-4 text-xs font-semibold transition-all border-b-2 {activeTab === "structure" ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground"}"
            >
              <span class="flex items-center gap-1.5">
                <FileText size={13} />
                Structure
              </span>
            </button>
            <button
              on:click={() => switchTab("sql")}
              class="relative py-2.5 px-4 text-xs font-semibold transition-all border-b-2 {activeTab === "sql" ? "border-primary text-primary" : "border-transparent text-muted-foreground hover:text-foreground"}"
            >
              <span class="flex items-center gap-1.5">
                <Terminal size={13} />
                SQL Query
              </span>
            </button>
          </div>

          <!-- Quick statistics of active table -->
          {#if selectedTable}
            {@const activeT = tables.find(t => t.name === selectedTable)}
            {#if activeT}
              <div class="hidden sm:flex items-center gap-3 text-[10px] text-muted-foreground font-mono">
                <span>Engine: <b class="text-foreground">{activeT.engine}</b></span>
                <span>•</span>
                <span>Size: <b class="text-foreground">{formatBytes(activeT.data_size)}</b></span>
                {#if activeT.collation}
                  <span>•</span>
                  <span>Collation: <b class="text-foreground">{activeT.collation}</b></span>
                {/if}
              </div>
            {/if}
          {/if}
        </div>

        <!-- TAB CONTENT: BROWSE DATA -->
        {#if activeTab === "browse"}
          <div class="space-y-4">
            {#if dataLoading}
              <div class="flex flex-col items-center justify-center py-20 text-muted-foreground space-y-2">
                <RefreshCw size={24} class="animate-spin text-primary" />
                <span class="text-xs">Loading table rows...</span>
              </div>
            {:else if dataError}
              <div class="rounded-xl bg-rose-500/10 p-3.5 text-xs text-rose-500 border border-rose-500/20 flex items-start gap-2.5">
                <AlertTriangle size={15} class="shrink-0 mt-0.5" />
                <div>
                  <p class="font-semibold">Query error</p>
                  <p class="mt-0.5">{dataError}</p>
                </div>
              </div>
            {:else if columns.length === 0}
              <div class="rounded-2xl border border-border bg-card p-12 text-center text-muted-foreground text-xs">
                No column information returned for this table.
              </div>
            {:else}
              <!-- Table Data View -->
              <div class="rounded-2xl border border-border bg-card overflow-hidden">
                {#if rows.length === 0}
                  <div class="flex flex-col items-center justify-center py-16 text-center px-6 space-y-2 text-muted-foreground">
                    <Table size={24} class="opacity-45" />
                    <p class="text-xs font-semibold">Table is empty</p>
                    <p class="text-[11px]">No rows returned from this table schema.</p>
                  </div>
                {:else}
                  <div class="overflow-x-auto max-h-[500px]">
                    <table class="w-full text-left text-xs border-collapse">
                      <thead>
                        <tr class="border-b border-border bg-secondary/20 text-muted-foreground font-semibold">
                          {#each columns as col}
                            <th class="px-4 py-2.5 font-mono select-all truncate max-w-[200px]" title={col}>{col}</th>
                          {/each}
                        </tr>
                      </thead>
                      <tbody class="divide-y divide-border">
                        {#each rows as row}
                          <tr class="hover:bg-secondary/10 transition-colors">
                            {#each row as cell}
                              <td class="px-4 py-2.5 font-mono truncate max-w-[250px] text-foreground text-[11px]" title={cell === null ? "NULL" : String(cell)}>
                                {#if cell === null}
                                  <span class="italic text-muted-foreground/50">NULL</span>
                                {:else}
                                  {cell}
                                {/if}
                              </td>
                            {/each}
                          </tr>
                        {/each}
                      </tbody>
                    </table>
                  </div>
                {/if}
              </div>

              <!-- Pagination Footer -->
              <div class="flex flex-col sm:flex-row items-center justify-between gap-4 text-xs">
                <div class="text-muted-foreground">
                  {#if totalRows > 0}
                    Showing <b class="text-foreground">{offset + 1}</b> to <b class="text-foreground">{Math.min(offset + limit, totalRows)}</b> of <b class="text-foreground">{totalRows.toLocaleString()}</b> rows
                  {:else}
                    Showing 0 to 0 of 0 rows
                  {/if}
                </div>

                <div class="flex items-center gap-3">
                  <div class="flex items-center gap-1">
                    <span class="text-muted-foreground text-xs pr-1">Show:</span>
                    <select
                      bind:value={limit}
                      on:change={() => { offset = 0; fetchTableData(); }}
                      class="rounded-lg border border-border bg-card px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                    >
                      <option value={10}>10</option>
                      <option value={25}>25</option>
                      <option value={50}>50</option>
                      <option value={100}>100</option>
                    </select>
                  </div>

                  <div class="flex items-center gap-1.5">
                    <button 
                      on:click={prevPage}
                      disabled={offset === 0}
                      class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-card text-foreground hover:bg-secondary transition-colors disabled:opacity-30 disabled:pointer-events-none"
                    >
                      <ChevronLeft size={14} />
                    </button>
                    <button 
                      on:click={nextPage}
                      disabled={offset + limit >= totalRows}
                      class="inline-flex h-8 w-8 items-center justify-center rounded-lg border border-border bg-card text-foreground hover:bg-secondary transition-colors disabled:opacity-30 disabled:pointer-events-none"
                    >
                      <ChevronRight size={14} />
                    </button>
                  </div>
                </div>
              </div>
            {/if}
          </div>
        {/if}

        <!-- TAB CONTENT: STRUCTURE -->
        {#if activeTab === "structure"}
          <div class="space-y-4">
            {#if structureLoading}
              <div class="flex flex-col items-center justify-center py-20 text-muted-foreground space-y-2">
                <RefreshCw size={24} class="animate-spin text-primary" />
                <span class="text-xs">Loading columns definitions...</span>
              </div>
            {:else if structureError}
              <div class="rounded-xl bg-rose-500/10 p-3.5 text-xs text-rose-500 border border-rose-500/20 flex items-start gap-2.5">
                <AlertTriangle size={15} class="shrink-0 mt-0.5" />
                <div>
                  <p class="font-semibold">Structure error</p>
                  <p class="mt-0.5">{structureError}</p>
                </div>
              </div>
            {:else}
              <div class="rounded-2xl border border-border bg-card overflow-hidden">
                <div class="overflow-x-auto">
                  <table class="w-full text-left text-xs border-collapse">
                    <thead>
                      <tr class="border-b border-border bg-secondary/20 text-muted-foreground font-semibold">
                        <th class="px-6 py-3">Column Name</th>
                        <th class="px-6 py-3">Type</th>
                        <th class="px-6 py-3">Null</th>
                        <th class="px-6 py-3">Key</th>
                        <th class="px-6 py-3">Default Value</th>
                        <th class="px-6 py-3">Extra</th>
                        <th class="px-6 py-3">Comment</th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-border">
                      {#each structureColumns as col}
                        <tr class="hover:bg-secondary/10 transition-colors">
                          <td class="px-6 py-3 font-bold text-foreground font-mono">
                            {col.field}
                          </td>
                          <td class="px-6 py-3 text-muted-foreground font-mono">
                            {col.type}
                          </td>
                          <td class="px-6 py-3 text-muted-foreground">
                            {col.null}
                          </td>
                          <td class="px-6 py-3">
                            {#if col.key === "PRI"}
                              <span class="inline-flex items-center rounded bg-amber-500/10 px-1.5 py-0.5 text-[9px] font-bold text-amber-500 uppercase">
                                Primary Key
                              </span>
                            {:else if col.key === "MUL"}
                              <span class="inline-flex items-center rounded bg-blue-500/10 px-1.5 py-0.5 text-[9px] font-semibold text-blue-500 uppercase">
                                Indexed
                              </span>
                            {:else if col.key === "UNI"}
                              <span class="inline-flex items-center rounded bg-purple-500/10 px-1.5 py-0.5 text-[9px] font-semibold text-purple-500 uppercase">
                                Unique
                              </span>
                            {:else}
                              <span class="text-muted-foreground/30 font-mono">-</span>
                            {/if}
                          </td>
                          <td class="px-6 py-3 text-muted-foreground font-mono">
                            {#if col.default === "" || col.default === "NULL"}
                              <span class="italic text-muted-foreground/45">NULL</span>
                            {:else}
                              {col.default}
                            {/if}
                          </td>
                          <td class="px-6 py-3 text-muted-foreground font-mono">
                            {col.extra || "-"}
                          </td>
                          <td class="px-6 py-3 text-muted-foreground italic">
                            {col.comment || "-"}
                          </td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                </div>
              </div>
            {/if}
          </div>
        {/if}

        <!-- TAB CONTENT: SQL QUERY RUNNER -->
        {#if activeTab === "sql"}
          <div class="space-y-4">
            <!-- Text area and execution controls -->
            <div class="rounded-2xl border border-border bg-card overflow-hidden">
              <div class="flex items-center justify-between border-b border-border bg-secondary/15 px-4 py-2.5">
                <span class="text-xs font-bold text-foreground flex items-center gap-1.5">
                  <Terminal size={14} class="text-primary" />
                  SQL Statement Editor
                </span>
                
                <!-- Quick template links -->
                <div class="flex items-center gap-2">
                  <span class="text-[10px] text-muted-foreground pr-1 hidden sm:inline">Templates:</span>
                  <button 
                    on:click={() => appendSQLTemplate(`SELECT * FROM \`${selectedTable}\` LIMIT 50;`)}
                    class="text-[10px] text-primary hover:underline hover:text-primary/80"
                  >
                    SELECT
                  </button>
                  <span class="text-muted-foreground/20 text-[10px] hidden sm:inline">|</span>
                  <button 
                    on:click={() => appendSQLTemplate(`DESCRIBE \`${selectedTable}\`;`)}
                    class="text-[10px] text-primary hover:underline hover:text-primary/80"
                  >
                    DESCRIBE
                  </button>
                  <span class="text-muted-foreground/20 text-[10px] hidden sm:inline">|</span>
                  <button 
                    on:click={() => appendSQLTemplate(`SHOW TABLES;`)}
                    class="text-[10px] text-primary hover:underline hover:text-primary/80"
                  >
                    SHOW TABLES
                  </button>
                </div>
              </div>

              <div class="p-4 space-y-4">
                <textarea
                  bind:value={sqlQuery}
                  rows="6"
                  placeholder="SELECT * FROM users WHERE status = 'active';"
                  class="w-full rounded-lg border border-border bg-secondary/10 p-3.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary font-mono"
                />

                <div class="flex items-center justify-between">
                  <div class="text-[10px] text-muted-foreground flex items-center gap-1.5">
                    <Info size={12} class="text-primary" />
                    <span>Executes standard SQL queries directly on this database instance.</span>
                  </div>
                  <button 
                    on:click={runSQLQuery}
                    disabled={sqlRunning || !sqlQuery.trim()}
                    class="inline-flex h-9 items-center gap-1.5 rounded-xl bg-primary px-4 text-xs font-semibold text-primary-foreground shadow hover:opacity-90 transition-opacity disabled:opacity-50"
                  >
                    {#if sqlRunning}
                      <RefreshCw size={13} class="animate-spin" />
                      Running...
                    {:else}
                      <Play size={13} />
                      Run SQL Query
                    {/if}
                  </button>
                </div>
              </div>
            </div>

            <!-- Query results -->
            {#if sqlError}
              <div class="rounded-xl bg-rose-500/10 p-3.5 text-xs text-rose-500 border border-rose-500/20 flex items-start gap-2.5">
                <AlertTriangle size={15} class="shrink-0 mt-0.5" />
                <div>
                  <p class="font-semibold">Query execution error</p>
                  <p class="mt-0.5">{sqlError}</p>
                </div>
              </div>
            {/if}

            {#if sqlSuccessMessage}
              <div class="rounded-xl bg-emerald-500/10 p-3.5 text-xs text-emerald-500 border border-emerald-500/20 flex items-start gap-2.5">
                <Check size={15} class="shrink-0 mt-0.5" />
                <div>
                  <p class="font-semibold">Query OK</p>
                  <p class="mt-0.5">{sqlSuccessMessage}</p>
                </div>
              </div>
            {/if}

            {#if sqlResult && sqlResult.type === "select"}
              <div class="space-y-2">
                <h4 class="text-xs font-bold text-foreground">Result Table:</h4>
                <div class="rounded-2xl border border-border bg-card overflow-hidden">
                  {#if sqlResult.rows.length === 0}
                    <div class="p-8 text-center text-xs text-muted-foreground">
                      Query returned 0 rows.
                    </div>
                  {:else}
                    <div class="overflow-x-auto max-h-[500px]">
                      <table class="w-full text-left text-xs border-collapse">
                        <thead>
                          <tr class="border-b border-border bg-secondary/20 text-muted-foreground font-semibold">
                            {#each sqlResult.columns as col}
                              <th class="px-4 py-2.5 font-mono select-all truncate max-w-[200px]" title={col}>{col}</th>
                            {/each}
                          </tr>
                        </thead>
                        <tbody class="divide-y divide-border">
                          {#each sqlResult.rows as row}
                            <tr class="hover:bg-secondary/10 transition-colors">
                              {#each row as cell}
                                <td class="px-4 py-2.5 font-mono truncate max-w-[250px] text-foreground text-[11px]" title={cell === null ? "NULL" : String(cell)}>
                                  {#if cell === null}
                                    <span class="italic text-muted-foreground/50">NULL</span>
                                  {:else}
                                    {cell}
                                  {/if}
                                </td>
                              {/each}
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    </div>
                  {/if}
                </div>
                <p class="text-[10px] text-muted-foreground font-mono">Returned {sqlResult.rows.length} rows.</p>
              </div>
            {/if}
          </div>
        {/if}
      {/if}
    </div>
  </div>
</div>
