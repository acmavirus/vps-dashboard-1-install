import type { ReactNode, RefObject } from "react"
import { ChevronRight } from "lucide-react"

import type { LogData, LogTabItem } from "@/components/dashboard/types"
import { Card } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"

function highlightLog(text: string): ReactNode {
  if (!text) return text

  return text.split("\n").map((line, index) => {
    let color = "text-foreground/80"

    if (line.includes("ERROR") || line.includes("Failed") || line.includes("crit")) {
      color = "font-medium text-rose-400"
    } else if (line.includes("WARN") || line.includes("warning")) {
      color = "text-amber-400"
    } else if (line.includes(" 200 ") || line.includes("SUCCESS") || line.includes("active")) {
      color = "text-emerald-400"
    } else if (line.includes(" 404 ") || line.includes(" 500 ")) {
      color = "text-rose-500 underline"
    }

    return (
      <div key={index} className={color}>
        {line}
      </div>
    )
  })
}

export function LogsTab({
  logTabs,
  logTab,
  setLogTab,
  siteTab,
  setSiteTab,
  currentLog,
  live,
  autoScroll,
  setAutoScroll,
  logEndRef,
}: {
  logTabs: LogTabItem[]
  logTab: string
  setLogTab: (value: string) => void
  siteTab: "access" | "error"
  setSiteTab: (value: "access" | "error") => void
  currentLog: LogData | null | undefined
  live: boolean
  autoScroll: boolean
  setAutoScroll: (value: boolean) => void
  logEndRef: RefObject<HTMLDivElement>
}) {
  return (
    <div className="flex h-auto flex-col gap-4 lg:h-[640px] lg:flex-row">
      <div className="hidden w-52 flex-col gap-1 overflow-y-auto lg:flex">
        {logTabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setLogTab(tab.key)}
            className={`flex items-center gap-3 rounded-lg px-4 py-3 text-left text-sm font-light transition-colors ${
              logTab === tab.key
                ? "border border-border bg-card text-foreground"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            <tab.icon size={15} className={logTab === tab.key ? tab.color : ""} />
            <span className="truncate">{tab.label}</span>
            {logTab === tab.key && <ChevronRight size={14} className="ml-auto shrink-0 opacity-40" />}
          </button>
        ))}
      </div>

      <div className="flex gap-2 overflow-x-auto pb-2 lg:hidden">
        {logTabs.map((tab) => (
          <button
            key={tab.key}
            onClick={() => setLogTab(tab.key)}
            className={`whitespace-nowrap rounded-lg px-4 py-2 text-xs font-light transition-colors ${
              logTab === tab.key
                ? "border border-border bg-card text-foreground"
                : "text-muted-foreground"
            }`}
          >
            <span className="flex items-center gap-2">
              <tab.icon size={13} />
              {tab.label}
            </span>
          </button>
        ))}
      </div>

      <Card className="flex h-[480px] flex-1 flex-col overflow-hidden border-border bg-card lg:h-full">
        <div className="flex items-center justify-between border-b border-border bg-secondary/30 px-4 py-3 sm:px-5">
          <div className="flex items-center gap-3">
            <div className="flex gap-1.5">
              <span className="h-2.5 w-2.5 rounded-full bg-border" />
              <span className="h-2.5 w-2.5 rounded-full bg-border" />
              <span className="h-2.5 w-2.5 rounded-full bg-border" />
            </div>
            <span className="max-w-[300px] truncate text-[11px] font-light text-muted-foreground">
              {currentLog?.path ?? "loading..."}
            </span>
          </div>
          <div className="flex items-center gap-4">
            {logTab.startsWith("site:") && (
              <div className="rounded-md border border-border bg-background/50 p-0.5">
                <button
                  onClick={() => setSiteTab("access")}
                  className={`rounded-[4px] px-3 py-1 text-[10px] transition-all ${
                    siteTab === "access"
                      ? "bg-secondary text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  Access
                </button>
                <button
                  onClick={() => setSiteTab("error")}
                  className={`rounded-[4px] px-3 py-1 text-[10px] transition-all ${
                    siteTab === "error"
                      ? "bg-secondary text-foreground shadow-sm"
                      : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  Error
                </button>
              </div>
            )}
            <span
              className={`flex items-center gap-1.5 text-[10px] font-light ${
                live ? "text-emerald-400" : "text-muted-foreground"
              }`}
            >
              <span
                className={`h-1.5 w-1.5 rounded-full ${
                  live ? "bg-emerald-500" : "bg-muted-foreground"
                }`}
              />
              {live ? "live" : "offline"}
            </span>
            <div className="flex items-center gap-2 border-l border-border pl-4">
              <input
                type="checkbox"
                id="autoscroll"
                checked={autoScroll}
                onChange={(event) => setAutoScroll(event.target.checked)}
                className="h-3 w-3 rounded border-zinc-700 bg-zinc-800"
              />
              <label htmlFor="autoscroll" className="cursor-pointer select-none text-[10px] text-muted-foreground">
                Auto-scroll
              </label>
            </div>
          </div>
        </div>

        <ScrollArea className="flex-1">
          <div className="whitespace-pre-wrap p-4 font-mono text-[12px] font-light leading-relaxed sm:p-5 sm:text-[13px]">
            {currentLog ? highlightLog(currentLog.content) : "Waiting for data..."}
            <div ref={logEndRef} />
          </div>
          <div className="h-8" />
        </ScrollArea>
      </Card>
    </div>
  )
}
