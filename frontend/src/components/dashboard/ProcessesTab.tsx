import type { ProcessInfo } from "@/components/dashboard/types"
import { Card } from "@/components/ui/card"

export function ProcessesTab({ processes }: { processes: ProcessInfo[] }) {
  return (
    <Card className="overflow-hidden border-border bg-card">
      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs font-light">
          <thead className="border-b border-border bg-secondary/30 text-muted-foreground">
            <tr>
              <th className="px-6 py-3 font-medium">PID</th>
              <th className="px-6 py-3 font-medium">Name</th>
              <th className="px-6 py-3 font-medium">CPU %</th>
              <th className="px-6 py-3 font-medium">Mem %</th>
              <th className="px-6 py-3 font-medium">Command</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {processes.map((process) => (
              <tr key={process.pid} className="hover:bg-secondary/10">
                <td className="px-6 py-3 tabular-nums">{process.pid}</td>
                <td className="px-6 py-3 font-normal">{process.name}</td>
                <td className="px-6 py-3 tabular-nums">{process.cpu.toFixed(1)}</td>
                <td className="px-6 py-3 tabular-nums">{process.memory.toFixed(1)}</td>
                <td className="max-w-xs truncate px-6 py-3 text-muted-foreground" title={process.command}>
                  {process.command}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}
