import { RefreshCw, Trash2 } from "lucide-react"

import type {
  DomainDeleteState,
  DomainInfo,
  DomainNoteState,
} from "@/components/dashboard/types"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"

export function DomainsTab({
  domains,
  setDomainDelete,
  setDomainNote,
  onScan,
  scanning,
}: {
  domains: DomainInfo[]
  setDomainDelete: (value: DomainDeleteState) => void
  setDomainNote: (value: DomainNoteState) => void
  onScan: () => void
  scanning: boolean
}) {
  return (
    <Card className="overflow-hidden border-border bg-card">
      <div className="flex items-center justify-between border-b border-border bg-secondary/10 px-6 py-4">
        <h3 className="text-sm font-medium">Danh sách Domain</h3>
        <Button
          variant="outline"
          size="sm"
          onClick={onScan}
          disabled={scanning}
          className="h-8 gap-2 border-cyan-500/50 text-cyan-400 hover:bg-cyan-500/10 hover:text-cyan-300"
        >
          <RefreshCw size={14} className={scanning ? "animate-spin" : ""} />
          {scanning ? "Đang quét..." : "Quét trạng thái"}
        </Button>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs font-light">
          <thead className="border-b border-border bg-secondary/30 text-muted-foreground">
            <tr>
              <th className="px-6 py-3 font-medium">Domain</th>
              <th className="px-6 py-3 font-medium">Note</th>
              <th className="px-6 py-3 font-medium">Status</th>
              <th className="px-6 py-3 font-medium">HTTP Code</th>
              <th className="px-6 py-3 font-medium">Action</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {domains.map((domain, index) => (
              <tr key={`${domain.domain}-${index}`} className="hover:bg-secondary/10">
                <td className="px-6 py-3 font-normal">{domain.domain}</td>
                <td className="max-w-[260px] px-6 py-3 text-muted-foreground">
                  <div className="truncate">{domain.note || "--"}</div>
                </td>
                <td className="px-6 py-3">
                  <div className="flex items-center gap-2">
                    <span
                      className={`h-1.5 w-1.5 rounded-full ${
                        domain.status === "online" ? "bg-emerald-500" : "bg-rose-500"
                      }`}
                    />
                    <span className="capitalize">{domain.status}</span>
                  </div>
                </td>
                <td className="px-6 py-3 tabular-nums">
                  <span
                    className={
                      domain.code >= 200 && domain.code < 400
                        ? "text-emerald-400"
                        : "text-rose-400"
                    }
                  >
                    {domain.code || "--"}
                  </span>
                </td>
                <td className="px-6 py-3">
                  <div className="flex items-center gap-4">
                    <a
                      href={`http://${domain.domain}`}
                      target="_blank"
                      rel="noreferrer"
                      className="text-blue-400 hover:underline"
                    >
                      Visit
                    </a>
                    <a
                      href={`https://www.google.com/search?q=${encodeURIComponent(
                        `site:https://${domain.domain}`,
                      )}`}
                      target="_blank"
                      rel="noreferrer"
                      className="text-amber-400 hover:underline"
                    >
                      Google
                    </a>
                    <button
                      type="button"
                      onClick={() =>
                        setDomainNote({ domain: domain.domain, note: domain.note || "" })
                      }
                      className="text-cyan-400 hover:underline"
                    >
                      Edit note
                    </button>
                    <button
                      type="button"
                      onClick={() =>
                        setDomainDelete({
                          domain: domain.domain,
                          deleteDb: false,
                          deleteRoot: false,
                        })
                      }
                      className="inline-flex items-center gap-1.5 text-rose-400 transition-colors hover:text-rose-300"
                    >
                      <Trash2 size={13} />
                      Delete
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}
