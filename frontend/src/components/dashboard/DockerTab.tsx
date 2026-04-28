import { Box } from "lucide-react"

import type { ContainerInfo } from "@/components/dashboard/types"
import { Card, CardContent } from "@/components/ui/card"

export function DockerTab({ containers }: { containers: ContainerInfo[] }) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
      {containers.map((container, index) => (
        <Card key={`${container.name}-${index}`} className="border-border bg-card">
          <CardContent className="space-y-4 p-5">
            <div className="flex items-start justify-between">
              <div className="space-y-1">
                <h3 className="max-w-[180px] truncate text-sm font-semibold">{container.name}</h3>
                <p className="max-w-[180px] truncate text-[10px] text-muted-foreground">
                  {container.image}
                </p>
              </div>
              <Box size={16} className="text-indigo-400" />
            </div>
            <div className="flex items-center gap-2">
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  container.status.toLowerCase().includes("up")
                    ? "bg-emerald-500"
                    : "bg-rose-500"
                }`}
              />
              <span className="text-[11px] text-muted-foreground">{container.status}</span>
            </div>
            <div className="grid grid-cols-2 gap-4 border-t border-border pt-2">
              <div>
                <p className="mb-1 text-[10px] text-muted-foreground">CPU Usage</p>
                <p className="text-sm tabular-nums">{container.cpu}</p>
              </div>
              <div>
                <p className="mb-1 text-[10px] text-muted-foreground">Memory</p>
                <p className="text-sm tabular-nums">{container.mem}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}
