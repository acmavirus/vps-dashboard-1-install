export interface Stats {
  cpu: number
  ram: number
  ram_total: number
  ram_used: number
  swap_total: number
  swap_used: number
  swap_percent: number
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
  load_1: number
  load_5: number
  load_15: number
  cpu_cores: number
  cpu_model: string
  disk_read: number
  disk_write: number
  spam_alerts?: SpamAlert[]
}

export interface SpamAlert {
  domain: string
  request_count: number
  unique_ips: number
  detected_at: string
  severity: string
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
  is_starred?: boolean
  ssl_active?: boolean
  ssl_issuer?: string
  ssl_expiry?: string
  ssl_days?: number
  requests?: number
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
  icon: any
  color: string
}
