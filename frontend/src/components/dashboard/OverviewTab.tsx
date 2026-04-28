import {
  Activity,
  Cpu,
  Globe,
  HardDrive,
  MemoryStick,
  Play,
  RotateCcw,
  Square,
  Wifi,
} from "lucide-react"
import { Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts"

import type { Stats } from "@/components/dashboard/types"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Progress } from "@/components/ui/progress"

const COLORS: Record<string, string> = {
  blue: "bg-blue-500",
  violet: "bg-violet-500",
  amber: "bg-amber-500",
  emerald: "bg-emerald-500",
}

const gb = (bytes: number) => (bytes ? `${(bytes / 1073741824).toFixed(1)} GB` : "0 GB")

function MetricCard({
  label,
  value,
  unit,
  icon: Icon,
  color,
  sub,
}: {
  label: string
  value?: number
  unit: string
  icon: typeof Cpu
  color: string
  sub?: string
}) {
  return (
    <Card className="border-border bg-card">
      <CardContent className="space-y-3 p-4 sm:p-5">
        <div className="flex items-center justify-between">
          <span className="text-xs font-light text-muted-foreground">{label}</span>
          <Icon size={15} className="text-muted-foreground/60" />
        </div>
        <div className="flex items-baseline gap-1.5">
          <span className="text-2xl font-light tracking-tight tabular-nums sm:text-3xl">
            {value?.toFixed(1) ?? "--"}
          </span>
          <span className="text-xs font-light text-muted-foreground">{unit}</span>
        </div>
        <Progress
          value={value ?? 0}
          className="h-1 rounded-full bg-secondary"
          indicatorClassName={`${COLORS[color]} rounded-full`}
        />
        {sub && <p className="text-[11px] font-light text-muted-foreground/70">{sub}</p>}
      </CardContent>
    </Card>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between rounded-lg border border-border bg-card px-4 py-3">
      <span className="text-xs font-light text-muted-foreground">{label}</span>
      <span className="max-w-[180px] truncate text-xs font-normal text-foreground">{value}</span>
    </div>
  )
}

function ServiceRow({
  name,
  id,
  icon: Icon,
  onAction,
}: {
  name: string
  id: string
  icon: typeof Globe
  onAction: (service: string, action: string) => void
}) {
  return (
    <div className="flex items-center justify-between rounded-lg border border-border bg-card px-5 py-4">
      <div className="flex items-center gap-3">
        <Icon size={16} className="text-muted-foreground" />
        <span className="text-xs font-medium">{name}</span>
      </div>
      <div className="flex gap-2">
        <button
          onClick={() => onAction(id, "restart")}
          title="Restart"
          className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-blue-400"
        >
          <RotateCcw size={14} />
        </button>
        <button
          onClick={() => onAction(id, "stop")}
          title="Stop"
          className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-rose-400"
        >
          <Square size={14} />
        </button>
        <button
          onClick={() => onAction(id, "start")}
          title="Start"
          className="rounded-md p-1.5 text-muted-foreground transition-colors hover:bg-secondary hover:text-emerald-400"
        >
          <Play size={14} />
        </button>
      </div>
    </div>
  )
}

export function OverviewTab({
  stats,
  history,
  handleAction,
}: {
  stats: Stats | null
  history: { t: string; v: number }[]
  handleAction: (service: string, action: string) => void
}) {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4">
        <MetricCard label="CPU" value={stats?.cpu} unit="%" icon={Cpu} color="blue" />
        <MetricCard
          label="Memory"
          value={stats?.ram}
          unit="%"
          icon={MemoryStick}
          color="violet"
          sub={stats ? `${gb(stats.ram_used)} / ${gb(stats.ram_total)}` : undefined}
        />
        <MetricCard
          label="Disk"
          value={stats?.disk}
          unit="%"
          icon={HardDrive}
          color="amber"
          sub={stats ? `${gb(stats.disk_used)} / ${gb(stats.disk_total)}` : undefined}
        />
        <MetricCard
          label="Network"
          value={stats ? parseFloat((stats.net_recv / 1073741824).toFixed(1)) : undefined}
          unit="GB down"
          icon={Wifi}
          color="emerald"
          sub={stats ? `up ${gb(stats.net_sent)} - ${stats.connections} conns` : undefined}
        />
      </div>

      <Card className="border-border bg-card">
        <CardHeader className="flex flex-row items-center justify-between px-5 pb-2 pt-5 sm:px-6 sm:pt-6">
          <div>
            <CardTitle className="text-sm font-medium">CPU Usage</CardTitle>
            <p className="mt-0.5 text-xs font-light text-muted-foreground">Last 60 data points</p>
          </div>
          {stats && <span className="text-2xl font-light tabular-nums">{stats.cpu.toFixed(1)}%</span>}
        </CardHeader>
        <CardContent className="h-[260px] p-0 sm:h-[320px]">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={history} margin={{ left: 0, right: 12, top: 8, bottom: 0 }}>
              <defs>
                <linearGradient id="cpuFill" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.12} />
                  <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
                </linearGradient>
              </defs>
              <XAxis
                dataKey="t"
                fontSize={10}
                tickLine={false}
                axisLine={false}
                stroke="hsl(var(--muted-foreground))"
                minTickGap={60}
                dy={8}
                tick={{ fontWeight: 300 }}
              />
              <YAxis
                fontSize={10}
                tickLine={false}
                axisLine={false}
                domain={[0, 100]}
                stroke="hsl(var(--muted-foreground))"
                width={32}
                tick={{ fontWeight: 300 }}
              />
              <Tooltip
                contentStyle={{
                  background: "hsl(var(--card))",
                  border: "1px solid hsl(var(--border))",
                  borderRadius: "8px",
                  fontSize: "12px",
                  fontWeight: 300,
                }}
                labelStyle={{ color: "hsl(var(--muted-foreground))", fontSize: "11px" }}
                itemStyle={{ color: "hsl(var(--foreground))" }}
              />
              <Area
                type="monotone"
                dataKey="v"
                name="CPU"
                stroke="#3b82f6"
                strokeWidth={1.5}
                fill="url(#cpuFill)"
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3 sm:gap-4">
        <InfoRow label="Platform" value={stats?.platform ?? "--"} />
        <InfoRow label="Kernel" value={stats?.kernel ?? "--"} />
        <InfoRow label="OS" value={stats?.os ?? "--"} />
      </div>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <ServiceRow name="Nginx" id="nginx" onAction={handleAction} icon={Globe} />
        <ServiceRow name="PHP 8.3" id="php8.3" onAction={handleAction} icon={Activity} />
        <ServiceRow name="PHP 7.4" id="php7.4" onAction={handleAction} icon={Activity} />
        <ServiceRow name="MariaDB" id="mysql" onAction={handleAction} icon={HardDrive} />
      </div>
    </div>
  )
}
