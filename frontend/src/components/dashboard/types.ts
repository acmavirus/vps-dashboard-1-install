import type { LucideIcon } from "lucide-react"

export interface Stats {
  cpu: number
  ram: number
  ram_total: number
  ram_used: number
  disk: number
  disk_total: number
  disk_used: number
  uptime: number
  hostname: string
  os: string
  platform: string
  kernel: string
  net_sent: number
  net_recv: number
  connections: number
}

export interface LogData {
  content: string
  path: string
}

export interface AllLogs {
  system: LogData
  nginx_access?: LogData
  nginx_error?: LogData
  nginx_sites?: {
    domain: string
    access?: LogData
    error?: LogData
  }[]
}

export interface ProcessInfo {
  pid: number
  name: string
  cpu: number
  memory: number
  command: string
}

export interface ContainerInfo {
  name: string
  status: string
  image: string
  cpu: string
  mem: string
}

export interface Pm2Process {
  name: string
  pm_id: number | string
  status: string
  monit?: {
    cpu?: number
    memory?: number
  }
  pm2_env?: {
    pm_uptime?: number
  }
}

export interface DomainInfo {
  domain: string
  status: string
  code: number
  note?: string
}

export interface DomainDeleteState {
  domain: string
  deleteDb: boolean
  deleteRoot: boolean
}

export interface DomainNoteState {
  domain: string
  note: string
}

export interface LogTabItem {
  key: string
  label: string
  icon: LucideIcon
  color: string
}
