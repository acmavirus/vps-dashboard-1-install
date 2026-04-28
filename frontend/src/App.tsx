import { useEffect, useRef, useState } from "react"
import {
  Activity,
  AlertTriangle,
  Box,
  Clock,
  Cpu,
  Globe,
  Menu,
  Terminal,
  X,
} from "lucide-react"

import { DockerTab } from "@/components/dashboard/DockerTab"
import { DomainsTab } from "@/components/dashboard/DomainsTab"
import { LogsTab } from "@/components/dashboard/LogsTab"
import { NodesTab } from "@/components/dashboard/NodesTab"
import { OverviewTab } from "@/components/dashboard/OverviewTab"
import { ProcessesTab } from "@/components/dashboard/ProcessesTab"
import type {
  AllLogs,
  ContainerInfo,
  DomainDeleteState,
  DomainInfo,
  DomainNoteState,
  LogTabItem,
  Pm2Process,
  ProcessInfo,
  Stats,
} from "@/components/dashboard/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

const VERSION = "2.1.1"

const formatUptime = (seconds: number) => {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  return days > 0 ? `${days}d ${hours}h ${minutes}m` : `${hours}h ${minutes}m`
}

export default function App() {
  const [token, setToken] = useState<string | null>(localStorage.getItem("auth_token"))
  const [stats, setStats] = useState<Stats | null>(null)
  const [history, setHistory] = useState<{ t: string; v: number }[]>([])
  const [logs, setLogs] = useState<AllLogs | null>(null)
  const [processes, setProcesses] = useState<ProcessInfo[]>([])
  const [containers, setContainers] = useState<ContainerInfo[]>([])
  const [pm2, setPm2] = useState<Pm2Process[]>([])
  const [domains, setDomains] = useState<DomainInfo[]>([])
  const [domainDelete, setDomainDelete] = useState<DomainDeleteState | null>(null)
  const [domainDeleteLoading, setDomainDeleteLoading] = useState(false)
  const [domainNote, setDomainNote] = useState<DomainNoteState | null>(null)
  const [domainNoteLoading, setDomainNoteLoading] = useState(false)
  const [appTab, setAppTab] = useState("overview")
  const [live, setLive] = useState(false)
  const [logTab, setLogTab] = useState("system")
  const [siteTab, setSiteTab] = useState<"access" | "error">("access")
  const [autoScroll, setAutoScroll] = useState(true)
  const [nav, setNav] = useState(false)
  const [username, setUsername] = useState("")
  const [password, setPassword] = useState("")
  const [error, setError] = useState("")
  const [loading, setLoading] = useState(false)

  const es = useRef<EventSource | null>(null)
  const logEndRef = useRef<HTMLDivElement>(null)

  const push = (data: { stats?: Stats; logs?: AllLogs }) => {
    if (data.stats) {
      setStats(data.stats)
      const timeLabel = new Date().toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      })
      setHistory((prev) => [...prev.slice(-59), { t: timeLabel, v: data.stats!.cpu }])
    }

    if (data.logs) {
      setLogs(data.logs)
    }
  }

  const handleLogin = async (event: React.FormEvent) => {
    event.preventDefault()
    setLoading(true)
    setError("")

    try {
      const response = await fetch("/api/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      })
      const data = await response.json()

      if (response.ok) {
        localStorage.setItem("auth_token", data.token)
        setToken(data.token)
      } else {
        setError(data.error || "Login failed")
      }
    } catch {
      setError("Server error")
    } finally {
      setLoading(false)
    }
  }

  const handleLogout = () => {
    localStorage.removeItem("auth_token")
    setToken(null)
    es.current?.close()
  }

  useEffect(() => {
    if (!token) return

    const connect = () => {
      es.current?.close()
      const source = new EventSource(`/api/stream?token=${token}`)
      es.current = source

      source.onopen = () => setLive(true)
      source.onerror = (event) => {
        console.error("SSE Error:", event)
        setLive(false)
        source.close()
        setTimeout(connect, 3000)
      }
      source.onmessage = (event) => {
        try {
          push(JSON.parse(event.data))
        } catch {
          return
        }
      }
    }

    const poll = async () => {
      try {
        const headers = { Authorization: token }
        const options = { headers }

        const responses = await Promise.all([
          fetch("/api/stats", options),
          fetch("/api/logs", options),
          fetch("/api/processes", options),
          fetch("/api/docker", options),
          fetch("/api/pm2", options),
          fetch("/api/domains", options),
        ])

        if (responses.some((response) => response.status === 401)) {
          handleLogout()
          return
        }

        const [statsData, logsData, processData, dockerData, pm2Data, domainData] =
          await Promise.all(responses.map((response) => response.json()))

        push({ stats: statsData, logs: logsData })
        setProcesses(processData)
        setContainers(dockerData)
        setPm2(pm2Data)
        setDomains(domainData)
      } catch (pollError) {
        console.error("Polling error:", pollError)
      }
    }

    connect()
    poll()

    const intervalId = window.setInterval(poll, 3000)

    return () => {
      es.current?.close()
      window.clearInterval(intervalId)
    }
  }, [token])

  useEffect(() => {
    if (autoScroll && logEndRef.current) {
      logEndRef.current.scrollIntoView({ behavior: "smooth" })
    }
  }, [logs, logTab, siteTab, autoScroll])

  const handleAction = async (service: string, action: string) => {
    if (!confirm(`Are you sure you want to ${action} ${service}?`)) return

    try {
      const response = await fetch("/api/control", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ service, action }),
      })

      if (response.ok) {
        alert("Done!")
      } else if (response.status === 401) {
        handleLogout()
      } else {
        alert("Failed")
      }
    } catch {
      alert("Error")
    }
  }

  const handlePM2Action = async (name: string, action: string) => {
    if (!confirm(`Are you sure you want to ${action} ${name}?`)) return

    try {
      const response = await fetch("/api/pm2/control", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({ name, action }),
      })

      if (response.ok) {
        alert("Done!")
      } else if (response.status === 401) {
        handleLogout()
      } else {
        alert("Failed")
      }
    } catch {
      alert("Error")
    }
  }

  const handleDeleteDomain = async () => {
    if (!domainDelete) return

    setDomainDeleteLoading(true)

    try {
      const response = await fetch("/api/domains/delete", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          domain: domainDelete.domain,
          delete_db: domainDelete.deleteDb,
          delete_root: domainDelete.deleteRoot,
        }),
      })

      const data = await response.json().catch(() => ({}))

      if (response.ok) {
        setDomains((prev) => prev.filter((item) => item.domain !== domainDelete.domain))
        setDomainDelete(null)
        alert(data.message || `Deleted ${domainDelete.domain}`)
      } else if (response.status === 401) {
        handleLogout()
      } else {
        alert(data.error || "Delete failed")
      }
    } catch {
      alert("Error")
    } finally {
      setDomainDeleteLoading(false)
    }
  }

  const handleSaveDomainNote = async () => {
    if (!domainNote) return

    setDomainNoteLoading(true)

    try {
      const response = await fetch("/api/domains/note", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: token || "",
        },
        body: JSON.stringify({
          domain: domainNote.domain,
          note: domainNote.note,
        }),
      })

      const data = await response.json().catch(() => ({}))

      if (response.ok) {
        setDomains((prev) =>
          prev.map((item) =>
            item.domain === domainNote.domain
              ? { ...item, note: domainNote.note.trim() }
              : item,
          ),
        )
        setDomainNote(null)
        alert("Note saved")
      } else if (response.status === 401) {
        handleLogout()
      } else {
        alert(data.error || "Save note failed")
      }
    } catch {
      alert("Error")
    } finally {
      setDomainNoteLoading(false)
    }
  }

  const logTabs: LogTabItem[] = [
    { key: "system", label: "System", icon: Terminal, color: "text-blue-400" },
    ...(logs?.nginx_access || logs?.nginx_error
      ? [
          {
            key: "nginx_access",
            label: "Nginx Access",
            icon: Globe,
            color: "text-emerald-400",
          },
          {
            key: "nginx_error",
            label: "Nginx Error",
            icon: AlertTriangle,
            color: "text-rose-400",
          },
        ]
      : []),
    ...(logs?.nginx_sites?.map((site) => ({
      key: `site:${site.domain}`,
      label: site.domain,
      icon: Globe,
      color: "text-indigo-400",
    })) ?? []),
  ]

  const currentLog = (() => {
    if (!logs) return null
    if (logTab === "system") return logs.system
    if (logTab === "nginx_access") return logs.nginx_access
    if (logTab === "nginx_error") return logs.nginx_error

    if (logTab.startsWith("site:")) {
      const domain = logTab.replace("site:", "")
      const site = logs.nginx_sites?.find((item) => item.domain === domain)
      return siteTab === "access" ? site?.access : site?.error
    }

    return null
  })()

  const appTabs = [
    { key: "overview", label: "Overview", icon: Activity, description: "System summary and services" },
    { key: "processes", label: "Processes", icon: Cpu, description: "Top running processes" },
    { key: "docker", label: "Docker", icon: Box, description: "Containers and runtime status" },
    { key: "nodes", label: "Nodes", icon: Terminal, description: "PM2 applications and actions" },
    { key: "domains", label: "Domains", icon: Globe, description: "Sites, notes and domain actions" },
    { key: "logs", label: "Logs", icon: AlertTriangle, description: "System and nginx logs" },
  ]

  const activeAppTab = appTabs.find((tab) => tab.key === appTab) ?? appTabs[0]

  if (!token) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background p-4 text-foreground dark">
        <Card className="w-full max-w-md border-border bg-card">
          <CardHeader className="space-y-1 text-center">
            <CardTitle className="text-2xl font-light tracking-tight">AcmaDash Login</CardTitle>
            <p className="text-sm font-light text-muted-foreground">
              Enter your credentials to access the dashboard
            </p>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleLogin} className="space-y-4">
              <div className="space-y-2">
                <label className="text-xs font-light text-muted-foreground">Username</label>
                <input
                  type="text"
                  value={username}
                  onChange={(event) => setUsername(event.target.value)}
                  className="w-full rounded-lg border border-border bg-secondary/50 px-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="admin"
                  required
                />
              </div>
              <div className="space-y-2">
                <label className="text-xs font-light text-muted-foreground">Password</label>
                <input
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  className="w-full rounded-lg border border-border bg-secondary/50 px-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="••••••••"
                  required
                />
              </div>
              {error && <p className="text-center text-xs text-rose-400">{error}</p>}
              <button
                type="submit"
                disabled={loading}
                className="w-full rounded-lg bg-blue-600 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:opacity-50"
              >
                {loading ? "Logging in..." : "Sign In"}
              </button>
            </form>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-background text-foreground dark">
      {nav && (
        <div className="fixed inset-0 z-40 bg-black/50 lg:hidden" onClick={() => setNav(false)} />
      )}

      <aside
        className={`fixed inset-y-0 left-0 z-50 w-72 border-r border-border bg-card transition-transform duration-300 lg:hidden ${
          nav ? "" : "-translate-x-full"
        }`}
      >
        <div className="flex items-center justify-between border-b border-border p-6">
          <div>
            <p className="text-sm font-semibold tracking-wide">AcmaDash</p>
            <p className="text-[11px] text-muted-foreground">Dashboard shell</p>
          </div>
          <button type="button" onClick={() => setNav(false)}>
            <X size={18} className="text-muted-foreground" />
          </button>
        </div>
        <nav className="space-y-1 p-4">
          {appTabs.map((tab) => (
            <button
              key={tab.key}
              type="button"
              onClick={() => {
                setAppTab(tab.key)
                setNav(false)
              }}
              className={`flex w-full items-start gap-3 rounded-xl px-4 py-3 text-left transition-colors ${
                appTab === tab.key
                  ? "bg-secondary text-foreground"
                  : "text-muted-foreground hover:bg-secondary/40 hover:text-foreground"
              }`}
            >
              <tab.icon size={16} className="mt-0.5 shrink-0" />
              <span>
                <span className="block text-sm font-medium">{tab.label}</span>
                <span className="block text-[11px] text-muted-foreground">{tab.description}</span>
              </span>
            </button>
          ))}
          <div className="mt-4 border-t border-border pt-4">
            <button
              type="button"
              onClick={handleLogout}
              className="flex w-full items-center gap-3 rounded-lg px-4 py-3 text-sm font-light text-rose-400 transition-colors hover:bg-rose-400/10"
            >
              Logout
            </button>
          </div>
        </nav>
      </aside>

      <div className="min-h-screen bg-[radial-gradient(circle_at_top,_rgba(59,130,246,0.12),_transparent_35%),linear-gradient(180deg,rgba(255,255,255,0.02),transparent_20%)]">
        <div className="mx-auto max-w-[1600px] px-4 py-6 sm:px-6 sm:py-8 lg:flex lg:gap-6 lg:px-8">
          <aside className="hidden w-72 shrink-0 lg:block">
            <Card className="sticky top-6 overflow-hidden border-border bg-card/95 backdrop-blur">
              <CardHeader className="space-y-4 pb-4">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <CardTitle className="text-lg font-semibold tracking-tight">AcmaDash</CardTitle>
                    <p className="mt-1 text-xs text-muted-foreground">
                      {stats?.hostname ?? "..."} - {stats?.platform ?? "..."}
                    </p>
                  </div>
                  <Badge variant="secondary" className="rounded-full px-2 py-1 text-[10px]">
                    {live ? "Live" : "Syncing"}
                  </Badge>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="rounded-xl border border-border bg-secondary/30 p-3">
                    <p className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
                      Uptime
                    </p>
                    <p className="mt-2 text-sm font-medium">
                      {stats ? formatUptime(stats.uptime) : "--"}
                    </p>
                  </div>
                  <div className="rounded-xl border border-border bg-secondary/30 p-3">
                    <p className="text-[10px] uppercase tracking-[0.2em] text-muted-foreground">
                      Version
                    </p>
                    <p className="mt-2 text-sm font-medium">v{VERSION}</p>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <Separator />
                <nav className="space-y-1">
                  {appTabs.map((tab) => (
                    <button
                      key={tab.key}
                      type="button"
                      onClick={() => setAppTab(tab.key)}
                      className={`w-full rounded-xl px-4 py-3 text-left transition-colors ${
                        appTab === tab.key
                          ? "bg-secondary text-foreground shadow-sm"
                          : "text-muted-foreground hover:bg-secondary/40 hover:text-foreground"
                      }`}
                    >
                      <span className="flex items-start gap-3">
                        <tab.icon size={16} className="mt-0.5 shrink-0" />
                        <span>
                          <span className="block text-sm font-medium">{tab.label}</span>
                          <span className="mt-1 block text-[11px] text-muted-foreground">
                            {tab.description}
                          </span>
                        </span>
                      </span>
                    </button>
                  ))}
                </nav>
                <Separator />
                <Button variant="outline" className="w-full justify-start" onClick={handleLogout}>
                  Logout
                </Button>
              </CardContent>
            </Card>
          </aside>

          <main className="min-w-0 flex-1 space-y-8">
            <header className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div className="flex items-center gap-4">
                <Button
                  variant="ghost"
                  size="icon"
                  className="-ml-2 lg:hidden"
                  onClick={() => setNav(true)}
                >
                  <Menu size={18} />
                </Button>
                <div>
                  <div className="flex items-center gap-3">
                    <h1 className="text-xl font-semibold tracking-tight">{activeAppTab.label}</h1>
                    <Badge variant="outline" className="hidden rounded-full sm:inline-flex">
                      {stats?.os ?? "linux"}
                    </Badge>
                  </div>
                  <p className="text-xs font-light text-muted-foreground">
                    {stats?.hostname ?? "..."} - {stats?.platform ?? "..."}
                  </p>
                </div>
              </div>
              <div className="flex flex-wrap items-center gap-2 text-xs font-light text-muted-foreground">
                <Badge
                  variant="secondary"
                  className="hidden rounded-full px-3 py-1 text-[11px] sm:inline-flex"
                >
                  <span
                    className={`h-1.5 w-1.5 rounded-full ${
                      live ? "bg-emerald-500" : "animate-pulse bg-amber-500"
                    }`}
                  />
                  {live ? "Connected" : "Reconnecting"}
                </Badge>
                <Badge
                  variant="outline"
                  className="hidden rounded-full px-3 py-1 text-[11px] md:inline-flex"
                >
                  <Clock size={13} /> {stats ? formatUptime(stats.uptime) : "--"}
                </Badge>
                <Badge
                  variant="outline"
                  className="hidden rounded-full px-3 py-1 text-[11px] md:inline-flex"
                >
                  {stats ? `${stats.connections} conns` : "-- conns"}
                </Badge>
                <Button variant="outline" size="sm" onClick={handleLogout}>
                  Logout
                </Button>
                <span className="text-[10px] text-muted-foreground/60">v{VERSION}</span>
              </div>
            </header>

            <Tabs value={appTab} onValueChange={setAppTab} className="space-y-6">
              <TabsList className="h-9 rounded-lg border border-border bg-card p-0.5 lg:hidden">
                <TabsTrigger
                  value="overview"
                  className="h-full rounded-md px-4 text-xs font-normal data-[state=active]:bg-secondary data-[state=active]:shadow-sm"
                >
                  Overview
                </TabsTrigger>
                <TabsTrigger
                  value="processes"
                  className="h-full rounded-md px-4 text-xs font-normal data-[state=active]:bg-secondary data-[state=active]:shadow-sm"
                >
                  Processes
                </TabsTrigger>
                <TabsTrigger
                  value="docker"
                  className="h-full rounded-md px-4 text-xs font-normal data-[state=active]:bg-secondary data-[state=active]:shadow-sm"
                >
                  Docker
                </TabsTrigger>
                <TabsTrigger
                  value="nodes"
                  className="h-full rounded-md px-4 text-xs font-normal data-[state=active]:bg-secondary data-[state=active]:shadow-sm"
                >
                  Nodes
                </TabsTrigger>
                <TabsTrigger
                  value="domains"
                  className="h-full rounded-md px-4 text-xs font-normal data-[state=active]:bg-secondary data-[state=active]:shadow-sm"
                >
                  Domains
                </TabsTrigger>
                <TabsTrigger
                  value="logs"
                  className="h-full rounded-md px-4 text-xs font-normal data-[state=active]:bg-secondary data-[state=active]:shadow-sm"
                >
                  Logs
                </TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="mt-0 space-y-6">
                <OverviewTab stats={stats} history={history} handleAction={handleAction} />
              </TabsContent>

              <TabsContent value="processes" className="mt-0">
                <ProcessesTab processes={processes} />
              </TabsContent>

              <TabsContent value="docker" className="mt-0">
                <DockerTab containers={containers} />
              </TabsContent>

              <TabsContent value="nodes" className="mt-0">
                <NodesTab pm2={pm2} handlePM2Action={handlePM2Action} formatUptime={formatUptime} />
              </TabsContent>

              <TabsContent value="domains" className="mt-0">
                <DomainsTab
                  domains={domains}
                  setDomainDelete={setDomainDelete}
                  setDomainNote={setDomainNote}
                />
              </TabsContent>

              <TabsContent value="logs" className="mt-0">
                <LogsTab
                  logTabs={logTabs}
                  logTab={logTab}
                  setLogTab={setLogTab}
                  siteTab={siteTab}
                  setSiteTab={setSiteTab}
                  currentLog={currentLog}
                  live={live}
                  autoScroll={autoScroll}
                  setAutoScroll={setAutoScroll}
                  logEndRef={logEndRef}
                />
              </TabsContent>
            </Tabs>
          </main>
        </div>
      </div>

      {domainDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
          <div className="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl">
            <div className="border-b border-border px-6 py-4">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h3 className="text-base font-medium text-foreground">Delete domain</h3>
                  <p className="mt-1 text-sm text-muted-foreground">{domainDelete.domain}</p>
                </div>
                <button
                  type="button"
                  onClick={() => !domainDeleteLoading && setDomainDelete(null)}
                  className="text-muted-foreground transition-colors hover:text-foreground"
                >
                  <X size={16} />
                </button>
              </div>
            </div>

            <div className="space-y-4 px-6 py-5">
              <p className="text-sm text-muted-foreground">
                This action removes the domain config. Optional cleanup can also remove the
                database and root folder.
              </p>

              <label className="flex items-center gap-3 rounded-xl border border-border bg-secondary/20 px-4 py-3">
                <input
                  type="checkbox"
                  checked={domainDelete.deleteDb}
                  onChange={(event) =>
                    setDomainDelete((prev) =>
                      prev ? { ...prev, deleteDb: event.target.checked } : prev,
                    )
                  }
                  className="h-4 w-4 rounded border-zinc-700 bg-zinc-800"
                />
                <span className="text-sm text-foreground">Delete database</span>
              </label>

              <label className="flex items-center gap-3 rounded-xl border border-border bg-secondary/20 px-4 py-3">
                <input
                  type="checkbox"
                  checked={domainDelete.deleteRoot}
                  onChange={(event) =>
                    setDomainDelete((prev) =>
                      prev ? { ...prev, deleteRoot: event.target.checked } : prev,
                    )
                  }
                  className="h-4 w-4 rounded border-zinc-700 bg-zinc-800"
                />
                <span className="text-sm text-foreground">Delete root folder</span>
              </label>
            </div>

            <div className="flex items-center justify-end gap-3 border-t border-border px-6 py-4">
              <button
                type="button"
                onClick={() => setDomainDelete(null)}
                disabled={domainDeleteLoading}
                className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleDeleteDomain}
                disabled={domainDeleteLoading}
                className="rounded-lg bg-rose-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-rose-700 disabled:opacity-50"
              >
                {domainDeleteLoading ? "Deleting..." : "Delete domain"}
              </button>
            </div>
          </div>
        </div>
      )}

      {domainNote && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
          <div className="w-full max-w-md rounded-2xl border border-border bg-card shadow-2xl">
            <div className="border-b border-border px-6 py-4">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h3 className="text-base font-medium text-foreground">Edit note</h3>
                  <p className="mt-1 text-sm text-muted-foreground">{domainNote.domain}</p>
                </div>
                <button
                  type="button"
                  onClick={() => !domainNoteLoading && setDomainNote(null)}
                  className="text-muted-foreground transition-colors hover:text-foreground"
                >
                  <X size={16} />
                </button>
              </div>
            </div>

            <div className="space-y-4 px-6 py-5">
              <textarea
                value={domainNote.note}
                onChange={(event) =>
                  setDomainNote((prev) =>
                    prev ? { ...prev, note: event.target.value.slice(0, 500) } : prev,
                  )
                }
                rows={6}
                placeholder="Add a note for this domain..."
                className="w-full resize-none rounded-xl border border-border bg-secondary/30 px-4 py-3 text-sm text-foreground outline-none focus:ring-1 focus:ring-cyan-500"
              />
              <p className="text-right text-xs text-muted-foreground">{domainNote.note.length}/500</p>
            </div>

            <div className="flex items-center justify-end gap-3 border-t border-border px-6 py-4">
              <button
                type="button"
                onClick={() => setDomainNote(null)}
                disabled={domainNoteLoading}
                className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSaveDomainNote}
                disabled={domainNoteLoading}
                className="rounded-lg bg-cyan-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-cyan-700 disabled:opacity-50"
              >
                {domainNoteLoading ? "Saving..." : "Save note"}
              </button>
            </div>
          </div>
        </div>
      )}

      <footer className="mt-12 border-t border-border">
        <div className="mx-auto flex max-w-[1400px] flex-col items-center justify-between gap-3 px-4 py-6 text-xs font-light text-muted-foreground sm:flex-row sm:px-6 lg:px-8">
          <span>AcmaDash v{VERSION} - Built by AcmaTvirus</span>
          <span className="text-muted-foreground/50">&copy; 2024</span>
        </div>
      </footer>
    </div>
  )
}
