import { Activity, RotateCcw, Square, Play } from "lucide-react"

import type { Pm2Process } from "@/components/dashboard/types"
import { Card, CardContent } from "@/components/ui/card"

const gb = (bytes: number) => (bytes ? `${(bytes / 1073741824).toFixed(1)} GB` : "0 GB")

export function NodesTab({
  pm2,
  handlePM2Action,
  formatUptime,
}: {
  pm2: Pm2Process[]
  handlePM2Action: (name: string, action: string) => void
  formatUptime: (seconds: number) => string
}) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      {pm2.map((process, index) => (
        <Card key={`${process.name}-${index}`} className="border-border bg-card">
          <CardContent className="space-y-4 p-5">
            <div className="flex items-start justify-between">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <h3 className="max-w-[150px] truncate text-sm font-semibold">{process.name}</h3>
                  <span className="rounded bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground">
                    ID: {process.pm_id}
                  </span>
                </div>
                <p className="max-w-[180px] truncate text-[10px] text-muted-foreground">
                  {process.pm2_env?.pm_uptime
                    ? formatUptime(Math.floor((Date.now() - process.pm2_env.pm_uptime) / 1000))
                    : "N/A"}
                </p>
              </div>
              <div className="flex gap-1">
                <button
                  onClick={() => handlePM2Action(process.name, "restart")}
                  title="Restart"
                  className="rounded p-1 text-muted-foreground hover:bg-secondary hover:text-blue-400"
                >
                  <RotateCcw size={14} />
                </button>
                <button
                  onClick={() => handlePM2Action(process.name, "stop")}
                  title="Stop"
                  className="rounded p-1 text-muted-foreground hover:bg-secondary hover:text-rose-400"
                >
                  <Square size={14} />
                </button>
                <button
                  onClick={() => handlePM2Action(process.name, "start")}
                  title="Start"
                  className="rounded p-1 text-muted-foreground hover:bg-secondary hover:text-emerald-400"
                >
                  <Play size={14} />
                </button>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  process.status === "online" ? "bg-emerald-500" : "bg-rose-500"
                }`}
              />
              <span className="capitalize text-[11px] text-muted-foreground">{process.status}</span>
            </div>
            <div className="grid grid-cols-2 gap-4 border-t border-border pt-2">
              <div>
                <p className="mb-1 text-[10px] text-muted-foreground">CPU</p>
                <p className="text-sm tabular-nums">{process.monit?.cpu ?? 0}%</p>
              </div>
              <div>
                <p className="mb-1 text-[10px] text-muted-foreground">Memory</p>
                <p className="text-sm tabular-nums">{gb(process.monit?.memory ?? 0)}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
      {pm2.length === 0 && (
        <div className="col-span-full rounded-xl border border-dashed border-border py-12 text-center">
          <Activity className="mx-auto mb-3 opacity-20" size={32} />
          <p className="text-sm font-light text-muted-foreground">No PM2 processes found</p>
        </div>
      )}
    </div>
  )
}
