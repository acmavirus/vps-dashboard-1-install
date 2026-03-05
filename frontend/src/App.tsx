import { useState, useEffect, useRef } from 'react';
import {
    Cpu, HardDrive, Wifi, MemoryStick, Clock, Terminal,
    Globe, AlertTriangle, Menu, X, ChevronRight
} from 'lucide-react';
import {
    AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer
} from 'recharts';

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ScrollArea } from "@/components/ui/scroll-area";

const VERSION = "2.0.0";

/* ─── Types ────────────────────────────────────── */
interface Stats {
    cpu: number; ram: number; ram_total: number; ram_used: number;
    disk: number; disk_total: number; disk_used: number;
    uptime: number; hostname: string; os: string; platform: string;
    kernel: string; net_sent: number; net_recv: number; connections: number;
}
interface LogData { content: string; path: string; }
interface AllLogs {
    system: LogData;
    nginx_access?: LogData;
    nginx_error?: LogData;
    nginx_sites?: {
        domain: string;
        access?: LogData;
        error?: LogData;
    }[];
}

/* ─── Helpers ──────────────────────────────────── */
const gb = (b: number) => b ? (b / 1073741824).toFixed(1) + ' GB' : '0 GB';
const uptime = (s: number) => {
    const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60);
    return d > 0 ? `${d}d ${h}h ${m}m` : `${h}h ${m}m`;
};

/* ─── Component ────────────────────────────────── */
export default function App() {
    const [stats, setStats] = useState<Stats | null>(null);
    const [history, setHistory] = useState<{ t: string; v: number }[]>([]);
    const [logs, setLogs] = useState<AllLogs | null>(null);
    const [live, setLive] = useState(false);
    const [logTab, setLogTab] = useState('system');
    const [siteTab, setSiteTab] = useState<'access' | 'error'>('access');
    const [nav, setNav] = useState(false);
    const es = useRef<EventSource | null>(null);

    const push = (data: any) => {
        if (data.stats) {
            setStats(data.stats);
            const t = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
            setHistory(p => [...p.slice(-59), { t, v: data.stats.cpu }]);
        }
        if (data.logs) setLogs(data.logs);
    };

    useEffect(() => {
        const connect = () => {
            es.current?.close();
            const s = new EventSource('/api/stream');
            es.current = s;
            s.onopen = () => setLive(true);
            s.onerror = () => { setLive(false); s.close(); setTimeout(connect, 3000); };
            s.onmessage = e => { try { push(JSON.parse(e.data)); } catch { } };
        };
        connect();

        const poll = async () => {
            try {
                const [s, l] = await Promise.all([fetch('/api/stats').then(r => r.json()), fetch('/api/logs').then(r => r.json())]);
                push({ stats: s, logs: l });
            } catch { }
        };
        poll();
        const id = setInterval(poll, 3000);
        return () => { es.current?.close(); clearInterval(id); };
    }, []);

    const logTabs = [
        { key: 'system', label: 'System', icon: Terminal, color: 'text-blue-400' },
        ...(logs?.nginx_access || logs?.nginx_error ? [
            { key: 'nginx_access', label: 'Nginx Access', icon: Globe, color: 'text-emerald-400' },
            { key: 'nginx_error', label: 'Nginx Error', icon: AlertTriangle, color: 'text-rose-400' },
        ] : []),
        ...(logs?.nginx_sites?.map(s => ({
            key: `site:${s.domain}`,
            label: s.domain,
            icon: Globe,
            color: 'text-indigo-400'
        })) ?? []),
    ];

    const getCurrentLog = () => {
        if (!logs) return null;
        if (logTab === 'system') return logs.system;
        if (logTab === 'nginx_access') return logs.nginx_access;
        if (logTab === 'nginx_error') return logs.nginx_error;
        if (logTab.startsWith('site:')) {
            const domain = logTab.replace('site:', '');
            const site = logs.nginx_sites?.find(s => s.domain === domain);
            return siteTab === 'access' ? site?.access : site?.error;
        }
        return null;
    };

    const currentLog = getCurrentLog();

    return (
        <div className="dark min-h-screen bg-background text-foreground">
            {/* Mobile nav overlay */}
            {nav && <div className="fixed inset-0 bg-black/50 z-40 lg:hidden" onClick={() => setNav(false)} />}

            {/* Mobile sidebar */}
            <aside className={`fixed inset-y-0 left-0 w-64 bg-card border-r border-border z-50 lg:hidden transition-transform duration-300 ${nav ? '' : '-translate-x-full'}`}>
                <div className="p-6 flex items-center justify-between border-b border-border">
                    <span className="text-sm font-semibold tracking-wide">AcmaDash</span>
                    <button onClick={() => setNav(false)}><X size={18} className="text-muted-foreground" /></button>
                </div>
                <nav className="p-4 space-y-1">
                    {logTabs.map(t => (
                        <button key={t.key} onClick={() => { setLogTab(t.key); setNav(false); }}
                            className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-light transition-colors ${logTab === t.key ? 'bg-secondary text-foreground' : 'text-muted-foreground hover:text-foreground'}`}>
                            <t.icon size={16} /> {t.label}
                        </button>
                    ))}
                </nav>
            </aside>

            {/* Main layout */}
            <div className="max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 py-6 sm:py-8 space-y-8">

                {/* Header */}
                <header className="flex items-center justify-between">
                    <div className="flex items-center gap-4">
                        <button className="lg:hidden p-2 -ml-2 text-muted-foreground hover:text-foreground" onClick={() => setNav(true)}>
                            <Menu size={20} />
                        </button>
                        <div>
                            <h1 className="text-lg font-semibold tracking-tight">AcmaDash</h1>
                            <p className="text-xs text-muted-foreground font-light">{stats?.hostname ?? '...'} · {stats?.platform ?? '...'}</p>
                        </div>
                    </div>
                    <div className="flex items-center gap-4 text-xs text-muted-foreground font-light">
                        <span className="hidden sm:flex items-center gap-2">
                            <span className={`w-1.5 h-1.5 rounded-full ${live ? 'bg-emerald-500' : 'bg-amber-500 animate-pulse'}`} />
                            {live ? 'Connected' : 'Reconnecting'}
                        </span>
                        <span className="hidden md:flex items-center gap-1.5">
                            <Clock size={13} /> {stats ? uptime(stats.uptime) : '--'}
                        </span>
                        <span className="text-[10px] text-muted-foreground/60">v{VERSION}</span>
                    </div>
                </header>

                {/* Tabs */}
                <Tabs defaultValue="overview" className="space-y-6">
                    <TabsList className="bg-card border border-border h-9 p-0.5 rounded-lg">
                        <TabsTrigger value="overview" className="text-xs font-normal rounded-md px-4 h-full data-[state=active]:bg-secondary data-[state=active]:shadow-sm">Overview</TabsTrigger>
                        <TabsTrigger value="logs" className="text-xs font-normal rounded-md px-4 h-full data-[state=active]:bg-secondary data-[state=active]:shadow-sm">Logs</TabsTrigger>
                    </TabsList>

                    {/* ────── Overview ────── */}
                    <TabsContent value="overview" className="space-y-6 mt-0">

                        {/* Metric cards */}
                        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
                            <MetricCard label="CPU" value={stats?.cpu} unit="%" icon={Cpu} color="blue" />
                            <MetricCard label="Memory" value={stats?.ram} unit="%" icon={MemoryStick} color="violet"
                                sub={stats ? `${gb(stats.ram_used)} / ${gb(stats.ram_total)}` : undefined} />
                            <MetricCard label="Disk" value={stats?.disk} unit="%" icon={HardDrive} color="amber"
                                sub={stats ? `${gb(stats.disk_used)} / ${gb(stats.disk_total)}` : undefined} />
                            <MetricCard label="Network" value={stats ? parseFloat((stats.net_recv / 1073741824).toFixed(1)) : undefined} unit="GB ↓" icon={Wifi} color="emerald"
                                sub={stats ? `↑ ${gb(stats.net_sent)} · ${stats.connections} conns` : undefined} />
                        </div>

                        {/* Chart */}
                        <Card className="bg-card border-border">
                            <CardHeader className="pb-2 px-5 pt-5 sm:px-6 sm:pt-6 flex flex-row items-center justify-between">
                                <div>
                                    <CardTitle className="text-sm font-medium">CPU Usage</CardTitle>
                                    <p className="text-xs text-muted-foreground font-light mt-0.5">Last 60 data points</p>
                                </div>
                                {stats && <span className="text-2xl font-light tabular-nums">{stats.cpu.toFixed(1)}%</span>}
                            </CardHeader>
                            <CardContent className="p-0 h-[260px] sm:h-[320px]">
                                <ResponsiveContainer width="100%" height="100%">
                                    <AreaChart data={history} margin={{ left: 0, right: 12, top: 8, bottom: 0 }}>
                                        <defs>
                                            <linearGradient id="cpuFill" x1="0" y1="0" x2="0" y2="1">
                                                <stop offset="0%" stopColor="#3b82f6" stopOpacity={0.12} />
                                                <stop offset="100%" stopColor="#3b82f6" stopOpacity={0} />
                                            </linearGradient>
                                        </defs>
                                        <XAxis dataKey="t" fontSize={10} tickLine={false} axisLine={false}
                                            stroke="hsl(var(--muted-foreground))" minTickGap={60} dy={8}
                                            tick={{ fontWeight: 300 }} />
                                        <YAxis fontSize={10} tickLine={false} axisLine={false} domain={[0, 100]}
                                            stroke="hsl(var(--muted-foreground))" width={32}
                                            tick={{ fontWeight: 300 }} />
                                        <Tooltip
                                            contentStyle={{ background: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '8px', fontSize: '12px', fontWeight: 300 }}
                                            labelStyle={{ color: 'hsl(var(--muted-foreground))', fontSize: '11px' }}
                                            itemStyle={{ color: 'hsl(var(--foreground))' }}
                                        />
                                        <Area type="monotone" dataKey="v" name="CPU" stroke="#3b82f6" strokeWidth={1.5}
                                            fill="url(#cpuFill)" isAnimationActive={false} />
                                    </AreaChart>
                                </ResponsiveContainer>
                            </CardContent>
                        </Card>

                        {/* System info row */}
                        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4">
                            <InfoRow label="Platform" value={stats?.platform ?? '—'} />
                            <InfoRow label="Kernel" value={stats?.kernel ?? '—'} />
                            <InfoRow label="OS" value={stats?.os ?? '—'} />
                        </div>
                    </TabsContent>

                    {/* ────── Logs ────── */}
                    <TabsContent value="logs" className="mt-0">
                        <div className="flex flex-col lg:flex-row gap-4 h-auto lg:h-[640px]">

                            {/* Desktop side nav */}
                            <div className="hidden lg:flex flex-col w-52 gap-1 overflow-y-auto">
                                {logTabs.map(t => (
                                    <button key={t.key} onClick={() => setLogTab(t.key)}
                                        className={`flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-light transition-colors text-left ${logTab === t.key ? 'bg-card border border-border text-foreground' : 'text-muted-foreground hover:text-foreground'}`}>
                                        <t.icon size={15} className={logTab === t.key ? t.color : ''} />
                                        <span className="truncate">{t.label}</span>
                                        {logTab === t.key && <ChevronRight size={14} className="ml-auto opacity-40 shrink-0" />}
                                    </button>
                                ))}
                            </div>

                            {/* Mobile log tabs */}
                            <div className="flex lg:hidden gap-2 overflow-x-auto pb-2">
                                {logTabs.map(t => (
                                    <button key={t.key} onClick={() => setLogTab(t.key)}
                                        className={`flex items-center gap-2 px-4 py-2 rounded-lg text-xs font-light whitespace-nowrap transition-colors ${logTab === t.key ? 'bg-card border border-border text-foreground' : 'text-muted-foreground'}`}>
                                        <t.icon size={13} /> {t.label}
                                    </button>
                                ))}
                            </div>

                            {/* Terminal */}
                            <Card className="flex-1 bg-card border-border overflow-hidden flex flex-col h-[480px] lg:h-full">
                                <div className="flex items-center justify-between px-4 sm:px-5 py-3 border-b border-border bg-secondary/30">
                                    <div className="flex items-center gap-3">
                                        <div className="flex gap-1.5">
                                            <span className="w-2.5 h-2.5 rounded-full bg-border" />
                                            <span className="w-2.5 h-2.5 rounded-full bg-border" />
                                            <span className="w-2.5 h-2.5 rounded-full bg-border" />
                                        </div>
                                        <span className="text-[11px] text-muted-foreground font-light truncate max-w-[300px]">
                                            {currentLog?.path ?? 'loading...'}
                                        </span>
                                    </div>
                                    <div className="flex items-center gap-4">
                                        {logTab.startsWith('site:') && (
                                            <div className="flex bg-background/50 rounded-md p-0.5 border border-border">
                                                <button onClick={() => setSiteTab('access')}
                                                    className={`px-3 py-1 text-[10px] rounded-[4px] transition-all ${siteTab === 'access' ? 'bg-secondary text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}`}>
                                                    Access
                                                </button>
                                                <button onClick={() => setSiteTab('error')}
                                                    className={`px-3 py-1 text-[10px] rounded-[4px] transition-all ${siteTab === 'error' ? 'bg-secondary text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'}`}>
                                                    Error
                                                </button>
                                            </div>
                                        )}
                                        <span className={`text-[10px] font-light flex items-center gap-1.5 ${live ? 'text-emerald-400' : 'text-muted-foreground'}`}>
                                            <span className={`w-1.5 h-1.5 rounded-full ${live ? 'bg-emerald-500' : 'bg-muted-foreground'}`} /> {live ? 'live' : 'offline'}
                                        </span>
                                    </div>
                                </div>
                                <ScrollArea className="flex-1">
                                    <pre className="p-4 sm:p-5 text-[12px] sm:text-[13px] font-mono font-light leading-relaxed text-foreground/80 whitespace-pre-wrap">
                                        {currentLog ? (currentLog.content || 'Log file is empty.') : 'Waiting for data...'}
                                    </pre>
                                    <div className="h-8" />
                                </ScrollArea>
                            </Card>
                        </div>
                    </TabsContent>
                </Tabs>
            </div>

            {/* Footer */}
            <footer className="border-t border-border mt-12">
                <div className="max-w-[1400px] mx-auto px-4 sm:px-6 lg:px-8 py-6 flex flex-col sm:flex-row items-center justify-between gap-3 text-xs text-muted-foreground font-light">
                    <span>AcmaDash v{VERSION} · Built by AcmaTvirus</span>
                    <span className="text-muted-foreground/50">&copy; 2024</span>
                </div>
            </footer>
        </div>
    );
}

/* ─── Sub-components ───────────────────────────── */

const COLORS: Record<string, string> = {
    blue: 'bg-blue-500', violet: 'bg-violet-500', amber: 'bg-amber-500', emerald: 'bg-emerald-500',
};

function MetricCard({ label, value, unit, icon: Icon, color, sub }: {
    label: string; value?: number; unit: string; icon: any; color: string; sub?: string;
}) {
    return (
        <Card className="bg-card border-border">
            <CardContent className="p-4 sm:p-5 space-y-3">
                <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground font-light">{label}</span>
                    <Icon size={15} className="text-muted-foreground/60" />
                </div>
                <div className="flex items-baseline gap-1.5">
                    <span className="text-2xl sm:text-3xl font-light tabular-nums tracking-tight">
                        {value?.toFixed(1) ?? '—'}
                    </span>
                    <span className="text-xs text-muted-foreground font-light">{unit}</span>
                </div>
                <Progress value={value ?? 0} className="h-1 bg-secondary rounded-full"
                    indicatorClassName={`${COLORS[color]} rounded-full`} />
                {sub && <p className="text-[11px] text-muted-foreground/70 font-light">{sub}</p>}
            </CardContent>
        </Card>
    );
}

function InfoRow({ label, value }: { label: string; value: string }) {
    return (
        <div className="flex items-center justify-between bg-card border border-border rounded-lg px-4 py-3">
            <span className="text-xs text-muted-foreground font-light">{label}</span>
            <span className="text-xs font-normal text-foreground truncate max-w-[180px]">{value}</span>
        </div>
    );
}
