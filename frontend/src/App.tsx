import { useState, useEffect, useRef } from 'react';
import {
    Cpu, HardDrive, Wifi, MemoryStick, Clock, Terminal,
    Globe, AlertTriangle, Menu, X, ChevronRight,
    Play, Square, RotateCcw, Box, Activity, Trash2
} from 'lucide-react';
import {
    AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer
} from 'recharts';

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ScrollArea } from "@/components/ui/scroll-area";

const VERSION = "2.1.1";

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
interface Process {
    pid: number;
    name: string;
    cpu: number;
    memory: number;
    command: string;
}
interface Container {
    name: string;
    status: string;
    image: string;
    cpu: string;
    mem: string;
}
interface Domain {
    domain: string;
    status: string;
    code: number;
    note?: string;
}

interface DomainDeleteState {
    domain: string;
    deleteDb: boolean;
    deleteRoot: boolean;
}

interface DomainNoteState {
    domain: string;
    note: string;
}

/* ─── Helpers ──────────────────────────────────── */
const gb = (b: number) => b ? (b / 1073741824).toFixed(1) + ' GB' : '0 GB';
const uptime = (s: number) => {
    const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600), m = Math.floor((s % 3600) / 60);
    return d > 0 ? `${d}d ${h}h ${m}m` : `${h}h ${m}m`;
};

/* ─── Component ────────────────────────────────── */
export default function App() {
    const [token, setToken] = useState<string | null>(localStorage.getItem('auth_token'));
    const [stats, setStats] = useState<Stats | null>(null);
    const [history, setHistory] = useState<{ t: string; v: number }[]>([]);
    const [logs, setLogs] = useState<AllLogs | null>(null);
    const [processes, setProcesses] = useState<Process[]>([]);
    const [containers, setContainers] = useState<Container[]>([]);
    const [pm2, setPm2] = useState<any[]>([]);
    const [domains, setDomains] = useState<Domain[]>([]);
    const [domainDelete, setDomainDelete] = useState<DomainDeleteState | null>(null);
    const [domainDeleteLoading, setDomainDeleteLoading] = useState(false);
    const [domainNote, setDomainNote] = useState<DomainNoteState | null>(null);
    const [domainNoteLoading, setDomainNoteLoading] = useState(false);
    const [live, setLive] = useState(false);
    const [logTab, setLogTab] = useState('system');
    const [siteTab, setSiteTab] = useState<'access' | 'error'>('access');
    const [autoScroll, setAutoScroll] = useState(true);
    const [nav, setNav] = useState(false);
    const es = useRef<EventSource | null>(null);
    const logEndRef = useRef<HTMLDivElement>(null);

    // Login State
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);

    const handleLogin = async (e: React.FormEvent) => {
        e.preventDefault();
        setLoading(true);
        setError('');
        try {
            const res = await fetch('/api/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });
            const data = await res.json();
            if (res.ok) {
                localStorage.setItem('auth_token', data.token);
                setToken(data.token);
            } else {
                setError(data.error || 'Login failed');
            }
        } catch {
            setError('Server error');
        } finally {
            setLoading(false);
        }
    };

    const handleLogout = () => {
        localStorage.removeItem('auth_token');
        setToken(null);
        if (es.current) es.current.close();
    };

    const push = (data: any) => {
        if (data.stats) {
            setStats(data.stats);
            const t = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
            setHistory(p => [...p.slice(-59), { t, v: data.stats.cpu }]);
        }
        if (data.logs) setLogs(data.logs);
    };

    useEffect(() => {
        if (!token) return;

        const connect = () => {
            es.current?.close();
            // Gửi token qua query string cho SSE
            const s = new EventSource(`/api/stream?token=${token}`);
            es.current = s;
            s.onopen = () => setLive(true);
            s.onerror = (e) => { 
                console.error("SSE Error:", e);
                setLive(false); 
                s.close(); 
                // Nếu lỗi 401 (unauthorized) thì logout
                setTimeout(connect, 3000); 
            };
            s.onmessage = e => { try { push(JSON.parse(e.data)); } catch { } };
        };
        connect();

        const poll = async () => {
            try {
                const headers = { 'Authorization': token };
                const fetchOpts = { headers };
                
                const responses = await Promise.all([
                    fetch('/api/stats', fetchOpts),
                    fetch('/api/logs', fetchOpts),
                    fetch('/api/processes', fetchOpts),
                    fetch('/api/docker', fetchOpts),
                    fetch('/api/pm2', fetchOpts),
                    fetch('/api/domains', fetchOpts)
                ]);

                // Check for 401
                if (responses.some(r => r.status === 401)) {
                    handleLogout();
                    return;
                }

                const [s, l, p, d, pm, doms] = await Promise.all(responses.map(r => r.json()));
                push({ stats: s, logs: l });
                setProcesses(p);
                setContainers(d);
                setPm2(pm);
                setDomains(doms);
            } catch (err) {
                console.error("Polling error:", err);
            }
        };
        poll();
        const id = setInterval(poll, 3000);
        return () => { es.current?.close(); clearInterval(id); };
    }, [token]);

    useEffect(() => {
        if (autoScroll && logEndRef.current) {
            logEndRef.current.scrollIntoView({ behavior: 'smooth' });
        }
    }, [logs, logTab, siteTab, autoScroll]);

    const highlightLog = (text: string) => {
        if (!text) return text;
        const lines = text.split('\n');
        return lines.map((line, i) => {
            let color = 'text-foreground/80';
            if (line.includes('ERROR') || line.includes('Failed') || line.includes('crit')) color = 'text-rose-400 font-medium';
            else if (line.includes('WARN') || line.includes('warning')) color = 'text-amber-400';
            else if (line.includes(' 200 ') || line.includes('SUCCESS') || line.includes('active')) color = 'text-emerald-400';
            else if (line.includes(' 404 ') || line.includes(' 500 ')) color = 'text-rose-500 underline';
            
            return <div key={i} className={color}>{line}</div>;
        });
    };

    const handleAction = async (service: string, action: string) => {
        if (!confirm(`Are you sure you want to ${action} ${service}?`)) return;
        try {
            const res = await fetch('/api/control', {
                method: 'POST',
                headers: { 
                    'Content-Type': 'application/json',
                    'Authorization': token || ''
                },
                body: JSON.stringify({ service, action })
            });
            if (res.ok) alert('Done!');
            else if (res.status === 401) handleLogout();
            else alert('Failed');
        } catch { alert('Error'); }
    };

    const handlePM2Action = async (name: string, action: string) => {
        if (!confirm(`Are you sure you want to ${action} ${name}?`)) return;
        try {
            const res = await fetch('/api/pm2/control', {
                method: 'POST',
                headers: { 
                    'Content-Type': 'application/json',
                    'Authorization': token || ''
                },
                body: JSON.stringify({ name, action })
            });
            if (res.ok) alert('Done!');
            else if (res.status === 401) handleLogout();
            else alert('Failed');
        } catch { alert('Error'); }
    };

    const handleDeleteDomain = async () => {
        if (!domainDelete) return;
        setDomainDeleteLoading(true);
        try {
            const res = await fetch('/api/domains/delete', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': token || ''
                },
                body: JSON.stringify({
                    domain: domainDelete.domain,
                    delete_db: domainDelete.deleteDb,
                    delete_root: domainDelete.deleteRoot
                })
            });

            const data = await res.json().catch(() => ({}));
            if (res.ok) {
                setDomains(prev => prev.filter(item => item.domain !== domainDelete.domain));
                setDomainDelete(null);
                alert(data.message || `Deleted ${domainDelete.domain}`);
            } else if (res.status === 401) {
                handleLogout();
            } else {
                alert(data.error || 'Delete failed');
            }
        } catch {
            alert('Error');
        } finally {
            setDomainDeleteLoading(false);
        }
    };

    const handleSaveDomainNote = async () => {
        if (!domainNote) return;
        setDomainNoteLoading(true);
        try {
            const res = await fetch('/api/domains/note', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': token || ''
                },
                body: JSON.stringify({
                    domain: domainNote.domain,
                    note: domainNote.note
                })
            });

            const data = await res.json().catch(() => ({}));
            if (res.ok) {
                setDomains(prev => prev.map(item => item.domain === domainNote.domain ? { ...item, note: domainNote.note.trim() } : item));
                setDomainNote(null);
                alert('Note saved');
            } else if (res.status === 401) {
                handleLogout();
            } else {
                alert(data.error || 'Save note failed');
            }
        } catch {
            alert('Error');
        } finally {
            setDomainNoteLoading(false);
        }
    };

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

    if (!token) {
        return (
            <div className="dark min-h-screen bg-background text-foreground flex items-center justify-center p-4">
                <Card className="w-full max-w-md bg-card border-border">
                    <CardHeader className="space-y-1 text-center">
                        <CardTitle className="text-2xl font-light tracking-tight">AcmaDash Login</CardTitle>
                        <p className="text-sm text-muted-foreground font-light">Enter your credentials to access the dashboard</p>
                    </CardHeader>
                    <CardContent>
                        <form onSubmit={handleLogin} className="space-y-4">
                            <div className="space-y-2">
                                <label className="text-xs font-light text-muted-foreground">Username</label>
                                <input 
                                    type="text" 
                                    value={username} 
                                    onChange={e => setUsername(e.target.value)}
                                    className="w-full bg-secondary/50 border border-border rounded-lg px-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
                                    placeholder="admin"
                                    required
                                />
                            </div>
                            <div className="space-y-2">
                                <label className="text-xs font-light text-muted-foreground">Password</label>
                                <input 
                                    type="password" 
                                    value={password} 
                                    onChange={e => setPassword(e.target.value)}
                                    className="w-full bg-secondary/50 border border-border rounded-lg px-4 py-2 text-sm focus:outline-none focus:ring-1 focus:ring-blue-500"
                                    placeholder="••••••••"
                                    required
                                />
                            </div>
                            {error && <p className="text-xs text-rose-400 text-center">{error}</p>}
                            <button 
                                type="submit" 
                                disabled={loading}
                                className="w-full bg-blue-600 hover:bg-blue-700 text-white rounded-lg py-2 text-sm font-medium transition-colors disabled:opacity-50"
                            >
                                {loading ? 'Logging in...' : 'Sign In'}
                            </button>
                        </form>
                    </CardContent>
                </Card>
            </div>
        );
    }

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
                    <div className="pt-4 mt-4 border-t border-border">
                        <button onClick={handleLogout} className="w-full flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-light text-rose-400 hover:bg-rose-400/10 transition-colors">
                             Logout
                        </button>
                    </div>
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
                        <button onClick={handleLogout} className="text-[10px] hover:text-foreground border border-border px-2 py-1 rounded transition-colors">Logout</button>
                        <span className="text-[10px] text-muted-foreground/60">v{VERSION}</span>
                    </div>
                </header>

                {/* Tabs */}
                <Tabs defaultValue="overview" className="space-y-6">
                    <TabsList className="bg-card border border-border h-9 p-0.5 rounded-lg">
                        <TabsTrigger value="overview" className="text-xs font-normal rounded-md px-4 h-full data-[state=active]:bg-secondary data-[state=active]:shadow-sm">Overview</TabsTrigger>
                        <TabsTrigger value="processes" className="text-xs font-normal rounded-md px-4 h-full data-[state=active]:bg-secondary data-[state=active]:shadow-sm">Processes</TabsTrigger>
                        <TabsTrigger value="docker" className="text-xs font-normal rounded-md px-4 h-full data-[state=active]:bg-secondary data-[state=active]:shadow-sm">Docker</TabsTrigger>
                        <TabsTrigger value="nodes" className="text-xs font-normal rounded-md px-4 h-full data-[state=active]:bg-secondary data-[state=active]:shadow-sm">Nodes</TabsTrigger>
                        <TabsTrigger value="domains" className="text-xs font-normal rounded-md px-4 h-full data-[state=active]:bg-secondary data-[state=active]:shadow-sm">Domains</TabsTrigger>
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

                        {/* Services Management */}
                        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                            <ServiceRow name="Nginx" id="nginx" onAction={handleAction} icon={Globe} />
                            <ServiceRow name="PHP 8.3" id="php8.3" onAction={handleAction} icon={Activity} />
                            <ServiceRow name="PHP 7.4" id="php7.4" onAction={handleAction} icon={Activity} />
                            <ServiceRow name="MariaDB" id="mysql" onAction={handleAction} icon={HardDrive} />
                        </div>
                    </TabsContent>

                    {/* ────── Processes ────── */}
                    <TabsContent value="processes" className="mt-0 space-y-4">
                        <Card className="bg-card border-border overflow-hidden">
                            <div className="overflow-x-auto">
                                <table className="w-full text-left text-xs font-light">
                                    <thead className="bg-secondary/30 text-muted-foreground border-b border-border">
                                        <tr>
                                            <th className="px-6 py-3 font-medium">PID</th>
                                            <th className="px-6 py-3 font-medium">Name</th>
                                            <th className="px-6 py-3 font-medium">CPU %</th>
                                            <th className="px-6 py-3 font-medium">Mem %</th>
                                            <th className="px-6 py-3 font-medium">Command</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-border">
                                        {processes.map(p => (
                                            <tr key={p.pid} className="hover:bg-secondary/10">
                                                <td className="px-6 py-3 tabular-nums">{p.pid}</td>
                                                <td className="px-6 py-3 font-normal">{p.name}</td>
                                                <td className="px-6 py-3 tabular-nums">{p.cpu.toFixed(1)}</td>
                                                <td className="px-6 py-3 tabular-nums">{p.memory.toFixed(1)}</td>
                                                <td className="px-6 py-3 text-muted-foreground truncate max-w-xs" title={p.command}>{p.command}</td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </Card>
                    </TabsContent>

                    {/* ────── Docker ────── */}
                    <TabsContent value="docker" className="mt-0 space-y-4">
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                            {containers.map((c, i) => (
                                <Card key={i} className="bg-card border-border">
                                    <CardContent className="p-5 space-y-4">
                                        <div className="flex items-start justify-between">
                                            <div className="space-y-1">
                                                <h3 className="text-sm font-semibold truncate max-w-[180px]">{c.name}</h3>
                                                <p className="text-[10px] text-muted-foreground truncate max-w-[180px]">{c.image}</p>
                                            </div>
                                            <Box size={16} className="text-indigo-400" />
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <span className={`w-1.5 h-1.5 rounded-full ${c.status.toLowerCase().includes('up') ? 'bg-emerald-500' : 'bg-rose-500'}`} />
                                            <span className="text-[11px] text-muted-foreground">{c.status}</span>
                                        </div>
                                        <div className="grid grid-cols-2 gap-4 pt-2 border-t border-border">
                                            <div>
                                                <p className="text-[10px] text-muted-foreground mb-1">CPU Usage</p>
                                                <p className="text-sm tabular-nums">{c.cpu}</p>
                                            </div>
                                            <div>
                                                <p className="text-[10px] text-muted-foreground mb-1">Memory</p>
                                                <p className="text-sm tabular-nums">{c.mem}</p>
                                            </div>
                                        </div>
                                    </CardContent>
                                </Card>
                            ))}
                        </div>
                    </TabsContent>

                    {/* ────── Nodes (PM2) ────── */}
                    <TabsContent value="nodes" className="mt-0 space-y-4">
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                            {pm2.map((p, i) => (
                                <Card key={i} className="bg-card border-border">
                                    <CardContent className="p-5 space-y-4">
                                        <div className="flex items-start justify-between">
                                            <div className="space-y-1">
                                                <div className="flex items-center gap-2">
                                                    <h3 className="text-sm font-semibold truncate max-w-[150px]">{p.name}</h3>
                                                    <span className="text-[10px] bg-secondary px-1.5 py-0.5 rounded text-muted-foreground">ID: {p.pm_id}</span>
                                                </div>
                                                <p className="text-[10px] text-muted-foreground truncate max-w-[180px]">
                                                    {p.pm2_env?.pm_uptime ? uptime(Math.floor((Date.now() - p.pm2_env.pm_uptime) / 1000)) : 'N/A'}
                                                </p>
                                            </div>
                                            <div className="flex gap-1">
                                                <button onClick={() => handlePM2Action(p.name, 'restart')} title="Restart" className="p-1 hover:bg-secondary rounded text-muted-foreground hover:text-blue-400">
                                                    <RotateCcw size={14} />
                                                </button>
                                                <button onClick={() => handlePM2Action(p.name, 'stop')} title="Stop" className="p-1 hover:bg-secondary rounded text-muted-foreground hover:text-rose-400">
                                                    <Square size={14} />
                                                </button>
                                                <button onClick={() => handlePM2Action(p.name, 'start')} title="Start" className="p-1 hover:bg-secondary rounded text-muted-foreground hover:text-emerald-400">
                                                    <Play size={14} />
                                                </button>
                                            </div>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <span className={`w-1.5 h-1.5 rounded-full ${p.status === 'online' ? 'bg-emerald-500' : 'bg-rose-500'}`} />
                                            <span className="text-[11px] text-muted-foreground capitalize">{p.status}</span>
                                        </div>
                                        <div className="grid grid-cols-2 gap-4 pt-2 border-t border-border">
                                            <div>
                                                <p className="text-[10px] text-muted-foreground mb-1">CPU</p>
                                                <p className="text-sm tabular-nums">{p.monit?.cpu ?? 0}%</p>
                                            </div>
                                            <div>
                                                <p className="text-[10px] text-muted-foreground mb-1">Memory</p>
                                                <p className="text-sm tabular-nums">{gb(p.monit?.memory ?? 0)}</p>
                                            </div>
                                        </div>
                                    </CardContent>
                                </Card>
                            ))}
                            {pm2.length === 0 && (
                                <div className="col-span-full py-12 text-center border border-dashed border-border rounded-xl">
                                    <Activity className="mx-auto mb-3 opacity-20" size={32} />
                                    <p className="text-sm text-muted-foreground font-light">No PM2 processes found</p>
                                </div>
                            )}
                        </div>
                    </TabsContent>

                    {/* ────── Domains ────── */}
                    <TabsContent value="domains" className="mt-0">
                        <Card className="bg-card border-border overflow-hidden">
                            <div className="overflow-x-auto">
                                <table className="w-full text-left text-xs font-light">
                                    <thead className="bg-secondary/30 text-muted-foreground border-b border-border">
                                        <tr>
                                            <th className="px-6 py-3 font-medium">Domain</th>
                                            <th className="px-6 py-3 font-medium">Note</th>
                                            <th className="px-6 py-3 font-medium">Status</th>
                                            <th className="px-6 py-3 font-medium">HTTP Code</th>
                                            <th className="px-6 py-3 font-medium">Action</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-border">
                                        {domains.map((d, i) => (
                                            <tr key={i} className="hover:bg-secondary/10">
                                                <td className="px-6 py-3 font-normal">{d.domain}</td>
                                                <td className="px-6 py-3 text-muted-foreground max-w-[260px]">
                                                    <div className="truncate">{d.note || '—'}</div>
                                                </td>
                                                <td className="px-6 py-3">
                                                    <div className="flex items-center gap-2">
                                                        <span className={`w-1.5 h-1.5 rounded-full ${d.status === 'online' ? 'bg-emerald-500' : 'bg-rose-500'}`} />
                                                        <span className="capitalize">{d.status}</span>
                                                    </div>
                                                </td>
                                                <td className="px-6 py-3 tabular-nums">
                                                    <span className={d.code >= 200 && d.code < 400 ? 'text-emerald-400' : 'text-rose-400'}>
                                                        {d.code || '—'}
                                                    </span>
                                                </td>
                                                <td className="px-6 py-3">
                                                    <div className="flex items-center gap-4">
                                                        <a href={`http://${d.domain}`} target="_blank" rel="noreferrer" className="text-blue-400 hover:underline">Visit</a>
                                                        <a
                                                            href={`https://www.google.com/search?q=${encodeURIComponent(`site:https://${d.domain}`)}`}
                                                            target="_blank"
                                                            rel="noreferrer"
                                                            className="text-amber-400 hover:underline"
                                                        >
                                                            Google
                                                        </a>
                                                        <button
                                                            type="button"
                                                            onClick={() => setDomainNote({ domain: d.domain, note: d.note || '' })}
                                                            className="text-cyan-400 hover:underline"
                                                        >
                                                            Edit note
                                                        </button>
                                                        <button
                                                            type="button"
                                                            onClick={() => setDomainDelete({ domain: d.domain, deleteDb: false, deleteRoot: false })}
                                                            className="inline-flex items-center gap-1.5 text-rose-400 hover:text-rose-300 transition-colors"
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
                                        <div className="flex items-center gap-2 border-l border-border pl-4">
                                            <input 
                                                type="checkbox" 
                                                id="autoscroll" 
                                                checked={autoScroll} 
                                                onChange={(e) => setAutoScroll(e.target.checked)}
                                                className="w-3 h-3 rounded bg-zinc-800 border-zinc-700"
                                            />
                                            <label htmlFor="autoscroll" className="text-[10px] text-muted-foreground select-none cursor-pointer">Auto-scroll</label>
                                        </div>
                                    </div>
                                </div>
                                <ScrollArea className="flex-1">
                                    <div className="p-4 sm:p-5 text-[12px] sm:text-[13px] font-mono font-light leading-relaxed whitespace-pre-wrap">
                                        {currentLog ? highlightLog(currentLog.content) : 'Waiting for data...'}
                                        <div ref={logEndRef} />
                                    </div>
                                    <div className="h-8" />
                                </ScrollArea>
                            </Card>
                        </div>
                    </TabsContent>
                </Tabs>
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
                                    className="text-muted-foreground hover:text-foreground transition-colors"
                                >
                                    <X size={16} />
                                </button>
                            </div>
                        </div>

                        <div className="space-y-4 px-6 py-5">
                            <p className="text-sm text-muted-foreground">
                                Domain config và nginx log sẽ bị xóa. Có thể bật thêm xóa database từ file <code>.env</code> và xóa thư mục root theo nginx config.
                            </p>

                            <label className="flex items-start gap-3 rounded-xl border border-border px-4 py-3">
                                <input
                                    type="checkbox"
                                    checked={domainDelete.deleteDb}
                                    onChange={(e) => setDomainDelete(prev => prev ? { ...prev, deleteDb: e.target.checked } : prev)}
                                    className="mt-0.5 h-4 w-4 rounded bg-zinc-800 border-zinc-700"
                                />
                                <span>
                                    <span className="block text-sm text-foreground">Delete database</span>
                                    <span className="block text-xs text-muted-foreground">Tự dò `DB_DATABASE` từ `.env` của site.</span>
                                </span>
                            </label>

                            <label className="flex items-start gap-3 rounded-xl border border-border px-4 py-3">
                                <input
                                    type="checkbox"
                                    checked={domainDelete.deleteRoot}
                                    onChange={(e) => setDomainDelete(prev => prev ? { ...prev, deleteRoot: e.target.checked } : prev)}
                                    className="mt-0.5 h-4 w-4 rounded bg-zinc-800 border-zinc-700"
                                />
                                <span>
                                    <span className="block text-sm text-foreground">Delete root folder</span>
                                    <span className="block text-xs text-muted-foreground">Tự dò root path từ file nginx config của domain.</span>
                                </span>
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
                                {domainDeleteLoading ? 'Deleting...' : 'Delete domain'}
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
                                    className="text-muted-foreground hover:text-foreground transition-colors"
                                >
                                    <X size={16} />
                                </button>
                            </div>
                        </div>

                        <div className="space-y-4 px-6 py-5">
                            <textarea
                                value={domainNote.note}
                                onChange={(e) => setDomainNote(prev => prev ? { ...prev, note: e.target.value.slice(0, 500) } : prev)}
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
                                {domainNoteLoading ? 'Saving...' : 'Save note'}
                            </button>
                        </div>
                    </div>
                </div>
            )}

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

function ServiceRow({ name, id, onAction, icon: Icon }: { name: string, id: string, onAction: any, icon: any }) {
    return (
        <div className="flex items-center justify-between bg-card border border-border rounded-lg px-5 py-4">
            <div className="flex items-center gap-3">
                <Icon size={16} className="text-muted-foreground" />
                <span className="text-xs font-medium">{name}</span>
            </div>
            <div className="flex gap-2">
                <button onClick={() => onAction(id, 'restart')} title="Restart" className="p-1.5 hover:bg-secondary rounded-md text-muted-foreground hover:text-blue-400 transition-colors">
                    <RotateCcw size={14} />
                </button>
                <button onClick={() => onAction(id, 'stop')} title="Stop" className="p-1.5 hover:bg-secondary rounded-md text-muted-foreground hover:text-rose-400 transition-colors">
                    <Square size={14} />
                </button>
                <button onClick={() => onAction(id, 'start')} title="Start" className="p-1.5 hover:bg-secondary rounded-md text-muted-foreground hover:text-emerald-400 transition-colors">
                    <Play size={14} />
                </button>
            </div>
        </div>
    );
}
