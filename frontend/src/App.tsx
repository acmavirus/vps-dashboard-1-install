import React, { useState, useEffect, useRef } from 'react';
import {
    Activity,
    Cpu,
    Database,
    HardDrive,
    RefreshCcw,
    ShieldCheck,
    Server,
    CloudLightning,
    Terminal,
    Globe,
    Monitor,
    Hash,
    Link as LinkIcon,
    AlertTriangle,
    LayoutDashboard,
    FileText,
    Menu,
    X,
    CpuIcon,
    ArrowUpRight,
    CircleDot
} from 'lucide-react';
import {
    AreaChart,
    Area,
    XAxis,
    YAxis,
    CartesianGrid,
    Tooltip,
    ResponsiveContainer
} from 'recharts';

import { Card, CardContent, CardDescription, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";

const Version = "1.1.4-Slate";

interface Stats {
    cpu: number;
    ram: number;
    ram_total: number;
    ram_used: number;
    disk: number;
    disk_total: number;
    disk_used: number;
    uptime: number;
    hostname: string;
    os: string;
    platform: string;
    kernel: string;
    net_sent: number;
    net_recv: number;
    connections: number;
}

interface LogData {
    content: string;
    path: string;
}

interface AllLogs {
    system: LogData;
    nginx_access: LogData;
    nginx_error: LogData;
}

const App: React.FC = () => {
    const [stats, setStats] = useState<Stats | null>(null);
    const [history, setHistory] = useState<{ time: string; cpu: number }[]>([]);
    const [logs, setLogs] = useState<AllLogs | null>(null);
    const [connected, setConnected] = useState(false);
    const [logTab, setLogTab] = useState<'system' | 'nginx_access' | 'nginx_error'>('system');
    const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
    const eventSourceRef = useRef<EventSource | null>(null);

    const formatBytes = (bytes: number) => {
        if (!bytes) return '0 GB';
        return (bytes / (1024 * 1024 * 1024)).toFixed(2) + ' GB';
    };

    const formatUptime = (seconds: number) => {
        const d = Math.floor(seconds / (3600 * 24));
        const h = Math.floor((seconds % (3600 * 24)) / 3600);
        const m = Math.floor((seconds % 3600) / 60);
        return `${d}d ${h}h ${m}m`;
    };

    const updateUI = (data: any) => {
        const { stats: newStat, logs: newLogs } = data;
        if (newStat) {
            setStats(newStat);
            const now = new Date();
            const timeStr = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
            setHistory(prev => {
                const updated = [...prev, { time: timeStr, cpu: newStat.cpu }];
                if (updated.length > 30) return updated.slice(1);
                return updated;
            });
        }
        if (newLogs) {
            setLogs(newLogs);
        }
    };

    const fetchFallback = async () => {
        try {
            const res = await fetch('/api/stats');
            const data = await res.json();
            const logRes = await fetch('/api/logs');
            const logData = await logRes.json();
            updateUI({ stats: data, logs: logData });
        } catch (e) {
            console.error("Fallback error", e);
        }
    };

    useEffect(() => {
        const setupSSE = () => {
            if (eventSourceRef.current) eventSourceRef.current.close();
            const es = new EventSource('/api/stream');
            eventSourceRef.current = es;
            es.onopen = () => setConnected(true);
            es.onerror = () => {
                setConnected(false);
                es.close();
                setTimeout(setupSSE, 5000);
            };
            es.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    updateUI(data);
                } catch (e) {
                    console.error("SSE parse error", e);
                }
            };
        };
        setupSSE();
        fetchFallback();
        const interval = setInterval(fetchFallback, 4000);
        return () => {
            if (eventSourceRef.current) eventSourceRef.current.close();
            clearInterval(interval);
        };
    }, []);

    return (
        <div className="min-h-screen bg-background text-foreground flex flex-col font-sans dark selection:bg-primary/20">
            {/* Soft Ambient Glows */}
            <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10 bg-grid-white/[0.02] bg-[size:40px_40px]"></div>
            <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10">
                <div className="absolute top-0 left-1/4 w-[500px] h-[500px] bg-blue-500/5 blur-[120px] rounded-full animate-pulse-slow"></div>
                <div className="absolute bottom-0 right-1/4 w-[500px] h-[500px] bg-indigo-500/5 blur-[120px] rounded-full animate-pulse-slow"></div>
            </div>

            <div className="flex flex-1 overflow-hidden relative">
                {/* Desktop Sidebar - "Modern Slate" Darker accent */}
                <aside className="hidden lg:flex flex-col w-20 bg-slate-950/40 border-r border-border items-center py-8 gap-12 backdrop-blur-xl">
                    <div className="w-12 h-12 bg-white/5 border border-white/10 rounded-2xl flex items-center justify-center shadow-sm">
                        <Server size={22} className="text-white" />
                    </div>
                    <nav className="flex flex-col gap-8">
                        <Button variant="ghost" size="icon" className="hover:bg-slate-800/50 rounded-2xl transition-all">
                            <LayoutDashboard size={20} className="text-blue-400" />
                        </Button>
                        <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-white rounded-2xl transition-all">
                            <Activity size={20} />
                        </Button>
                        <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-white rounded-2xl transition-all">
                            <Terminal size={20} />
                        </Button>
                    </nav>
                </aside>

                {/* Mobile Menu Backdrop */}
                {isMobileMenuOpen && (
                    <div className="lg:hidden fixed inset-0 bg-black/40 backdrop-blur-md z-40 transition-all duration-300" onClick={() => setIsMobileMenuOpen(false)}></div>
                )}

                {/* Mobile Sidebar */}
                <aside className={`lg:hidden fixed left-0 top-0 h-full w-72 bg-card z-50 transform transition-transform duration-500 border-r border-border ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}`}>
                    <div className="p-8 flex flex-col h-full">
                        <div className="flex items-center justify-between mb-12">
                            <div className="flex items-center gap-3">
                                <div className="w-8 h-8 bg-blue-500/10 border border-blue-500/20 rounded-lg flex items-center justify-center">
                                    <Server className="text-blue-400" size={16} />
                                </div>
                                <span className="font-bold uppercase tracking-tight text-xl">Acma<span className="text-blue-500 font-black">Dash</span></span>
                            </div>
                            <Button variant="ghost" size="icon" className="text-muted-foreground" onClick={() => setIsMobileMenuOpen(false)}>
                                <X size={20} />
                            </Button>
                        </div>
                        <nav className="flex flex-col gap-3 flex-1">
                            <Button variant="secondary" className="justify-start gap-4 text-xs font-bold uppercase tracking-widest h-14 rounded-2xl px-6">
                                <LayoutDashboard size={18} /> Overview
                            </Button>
                            <Button variant="ghost" className="justify-start gap-4 text-xs font-bold uppercase tracking-widest text-muted-foreground hover:text-white h-14 rounded-2xl px-6">
                                <Activity size={18} /> Monitors
                            </Button>
                            <Button variant="ghost" className="justify-start gap-4 text-xs font-bold uppercase tracking-widest text-muted-foreground hover:text-white h-14 rounded-2xl px-6">
                                <Terminal size={18} /> Terminal
                            </Button>
                        </nav>
                    </div>
                </aside>

                <main className="flex-1 overflow-y-auto p-4 sm:p-6 md:p-10 lg:p-12 space-y-10">
                    <header className="flex flex-col md:flex-row items-center justify-between gap-6">
                        <div className="flex items-center justify-between w-full md:w-auto">
                            <div className="flex items-center gap-4 sm:gap-6">
                                <Button variant="outline" size="icon" className="lg:hidden bg-card/40 border-border rounded-xl w-10 h-10 shadow-sm" onClick={() => setIsMobileMenuOpen(true)}>
                                    <Menu size={20} />
                                </Button>
                                <div>
                                    <h1 className="text-2xl sm:text-3xl font-black tracking-tight flex items-center gap-3 uppercase text-white">
                                        Acma<span className="text-blue-500">Dash</span>
                                        <Badge variant="outline" className="hidden xs:inline-flex border-blue-500/20 text-blue-400 bg-blue-500/5 h-6 px-2 text-[10px] font-bold">SLATE EDITION</Badge>
                                    </h1>
                                    <p className="text-[10px] sm:text-xs text-muted-foreground font-bold uppercase tracking-[0.2em] mt-1 italic">Real-time Infrastructure View</p>
                                </div>
                            </div>
                        </div>

                        <div className="flex items-center gap-3 w-full md:w-auto">
                            <div className="hidden md:flex items-center gap-3 bg-secondary/50 px-5 py-2.5 rounded-2xl border border-border shadow-sm">
                                <div className={`w-2 h-2 rounded-full ${connected ? "bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.3)]" : "bg-amber-500"}`}></div>
                                <span className="text-[10px] font-bold uppercase tracking-[0.2em] text-foreground">
                                    {connected ? "Live Operation" : "Connecting..."}
                                </span>
                            </div>
                            <Button size="sm" variant="outline" className="rounded-2xl border-border bg-secondary/30 hover:bg-secondary flex-1 md:flex-initial text-[10px] sm:text-xs h-10 px-6 font-bold uppercase">
                                <RefreshCcw size={14} className={`mr-2 ${connected ? "animate-spin-slow text-blue-400" : ""}`} /> Up: {stats ? formatUptime(stats.uptime) : 'SYNCING'}
                            </Button>
                        </div>
                    </header>

                    <Tabs defaultValue="overview" className="space-y-8 animate-in fade-in duration-700">
                        <div className="flex flex-col sm:flex-row items-center justify-between gap-6">
                            <TabsList className="bg-secondary/40 border border-border p-1 rounded-2xl w-full sm:w-auto">
                                <TabsTrigger value="overview" className="flex-1 sm:flex-initial data-[state=active]:bg-card rounded-xl px-8 font-bold text-[10px] tracking-widest uppercase h-9 transition-all">Overview</TabsTrigger>
                                <TabsTrigger value="logs" className="flex-1 sm:flex-initial data-[state=active]:bg-card rounded-xl px-8 font-bold text-[10px] tracking-widest uppercase h-9 transition-all">Terminal</TabsTrigger>
                            </TabsList>
                            <div className="flex items-center gap-3 bg-secondary/30 px-4 py-2 rounded-xl text-muted-foreground text-[11px] font-bold uppercase border border-border shadow-sm">
                                <Monitor size={14} /> {stats?.hostname || 'Unknown Host'}
                            </div>
                        </div>

                        <TabsContent value="overview" className="space-y-8">
                            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
                                <Card className="bg-card/50 border-border hover:border-blue-500/30 transition-all duration-300 shadow-sm relative overflow-hidden group">
                                    <CardHeader className="pb-3 p-6 pt-5 flex flex-row items-center justify-between">
                                        <CardTitle className="text-[10px] font-bold text-muted-foreground uppercase tracking-[0.2em]">Processor</CardTitle>
                                        <Cpu size={16} className="text-blue-500/70" />
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0 space-y-3">
                                        <div className="text-3xl font-bold tracking-tighter text-foreground font-mono">{stats?.cpu.toFixed(1) || '0.0'}<span className="text-xs text-muted-foreground ml-1">%</span></div>
                                        <Progress value={stats?.cpu || 0} className="h-2 bg-secondary rounded-full"
                                            indicatorClassName={stats && stats.cpu > 80 ? "bg-destructive shadow-[0_0_8px_rgba(239,68,68,0.3)]" : "bg-blue-500 shadow-[0_0_8px_rgba(59,130,246,0.2)]"}
                                        />
                                    </CardContent>
                                    <CardFooter className="p-4 py-3 bg-secondary/[0.3] text-[9px] text-muted-foreground font-bold uppercase tracking-widest flex justify-between">
                                        System Load <ArrowUpRight size={10} />
                                    </CardFooter>
                                </Card>

                                <Card className="bg-card/50 border-border hover:border-emerald-500/30 transition-all duration-300 shadow-sm relative overflow-hidden group">
                                    <CardHeader className="pb-3 p-6 pt-5 flex flex-row items-center justify-between">
                                        <CardTitle className="text-[10px] font-bold text-muted-foreground uppercase tracking-[0.2em]">Data RAM</CardTitle>
                                        <Database size={16} className="text-emerald-500/70" />
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0 space-y-3">
                                        <div className="text-3xl font-bold tracking-tighter text-foreground font-mono">{stats?.ram.toFixed(1) || '0.0'}<span className="text-xs text-muted-foreground ml-1">%</span></div>
                                        <Progress value={stats?.ram || 0} className="h-2 bg-secondary rounded-full"
                                            indicatorClassName="bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.2)]"
                                        />
                                    </CardContent>
                                    <CardFooter className="p-4 py-3 bg-secondary/[0.3] text-[9px] text-muted-foreground font-bold uppercase tracking-widest flex justify-between w-full">
                                        <span>{stats ? formatBytes(stats.ram_used) : '0.00'}</span>
                                        <span className="opacity-50">/ {stats ? formatBytes(stats.ram_total).split(' ')[0] : '0.00'}</span>
                                    </CardFooter>
                                </Card>

                                <Card className="bg-card/50 border-border hover:border-amber-500/30 transition-all duration-300 shadow-sm relative overflow-hidden group">
                                    <CardHeader className="pb-3 p-6 pt-5 flex flex-row items-center justify-between">
                                        <CardTitle className="text-[10px] font-bold text-muted-foreground uppercase tracking-[0.2em]">Storage</CardTitle>
                                        <HardDrive size={16} className="text-amber-500/70" />
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0 space-y-3">
                                        <div className="text-3xl font-bold tracking-tighter text-foreground font-mono">{stats?.disk.toFixed(1) || '0.0'}<span className="text-xs text-muted-foreground ml-1">%</span></div>
                                        <Progress value={stats?.disk || 0} className="h-2 bg-secondary rounded-full"
                                            indicatorClassName="bg-amber-500"
                                        />
                                    </CardContent>
                                    <CardFooter className="p-4 py-3 bg-secondary/[0.3] text-[9px] text-muted-foreground font-bold uppercase tracking-widest">
                                        Free: {stats ? formatBytes(stats.disk_total - stats.disk_used) : '---'}
                                    </CardFooter>
                                </Card>

                                <Card className="bg-card/50 border-border hover:border-indigo-500/30 transition-all duration-300 shadow-sm relative overflow-hidden group">
                                    <CardHeader className="pb-3 p-6 pt-5 flex flex-row items-center justify-between">
                                        <CardTitle className="text-[10px] font-bold text-muted-foreground uppercase tracking-[0.2em]">Networking</CardTitle>
                                        <CloudLightning size={16} className="text-indigo-500/70" />
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0 space-y-3">
                                        <div className="text-3xl font-bold tracking-tighter text-foreground truncate font-mono">
                                            {stats ? formatBytes(stats.net_recv) : '0.00 GB'}
                                        </div>
                                        <div className="text-[10px] text-muted-foreground font-bold uppercase tracking-widest opacity-60">Inbound Payload</div>
                                    </CardContent>
                                    <CardFooter className="p-4 py-3 bg-secondary/[0.3] text-[9px] text-muted-foreground font-bold uppercase tracking-widest flex justify-between w-full">
                                        <span>SENT: {stats ? formatBytes(stats.net_sent).split(' ')[0] : '0.00'}</span>
                                        <span className="flex items-center gap-1 text-indigo-400 font-mono">{stats?.connections || 0} CONNS</span>
                                    </CardFooter>
                                </Card>
                            </div>

                            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                                <Card className="lg:col-span-2 bg-card/60 border-border shadow-sm">
                                    <CardHeader className="flex flex-row items-center justify-between p-6 sm:p-8">
                                        <div>
                                            <CardTitle className="text-base font-bold uppercase tracking-widest text-foreground">Live Telemetry</CardTitle>
                                            <CardDescription className="text-[11px] uppercase font-bold text-muted-foreground mt-1">CPU Oscillation / Sec</CardDescription>
                                        </div>
                                        <div className="flex items-center gap-2">
                                            <div className="w-2 h-2 rounded-full bg-blue-500 animate-pulse"></div>
                                            <span className="text-[10px] font-bold uppercase text-muted-foreground tracking-widest">Live Flow</span>
                                        </div>
                                    </CardHeader>
                                    <CardContent className="h-[300px] sm:h-[400px] p-0 overflow-hidden">
                                        <ResponsiveContainer width="100%" height="100%">
                                            <AreaChart data={history} margin={{ left: -10, right: 10, top: 10, bottom: 0 }}>
                                                <defs>
                                                    <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                                                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.2} />
                                                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                                                    </linearGradient>
                                                </defs>
                                                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#ffffff05" />
                                                <XAxis dataKey="time" stroke="#475569" fontSize={10} tickLine={false} axisLine={false} minTickGap={50} tick={{ fontWeight: '700' }} />
                                                <YAxis stroke="#475569" fontSize={10} tickLine={false} axisLine={false} domain={[0, 100]} tick={{ fontWeight: '700' }} />
                                                <Tooltip
                                                    contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '12px', boxShadow: '0 10px 15px -3px rgba(0,0,0,0.1)' }}
                                                    itemStyle={{ color: 'hsl(var(--foreground))', fontWeight: 'bold' }}
                                                />
                                                <Area type="monotone" dataKey="cpu" stroke="#3b82f6" strokeWidth={3} fillOpacity={1} fill="url(#colorCpu)" animationDuration={300} isAnimationActive={history.length < 2} />
                                            </AreaChart>
                                        </ResponsiveContainer>
                                    </CardContent>
                                </Card>

                                <div className="space-y-6">
                                    <Card className="bg-card/40 border-border">
                                        <CardHeader className="pb-4 p-6">
                                            <CardTitle className="text-[11px] font-bold uppercase tracking-[0.2em] text-muted-foreground">System Manifest</CardTitle>
                                        </CardHeader>
                                        <CardContent className="space-y-5 p-6 pt-0">
                                            <div className="flex items-center justify-between">
                                                <span className="text-[10px] font-bold text-muted-foreground uppercase">OS Platform</span>
                                                <span className="text-xs font-bold uppercase italic text-foreground">{stats?.platform || 'Checking...'}</span>
                                            </div>
                                            <Separator className="bg-border/50" />
                                            <div className="flex items-center justify-between">
                                                <span className="text-[10px] font-bold text-muted-foreground uppercase">Kernel</span>
                                                <span className="text-[11px] font-bold text-foreground truncate max-w-[120px]">{stats?.kernel || '---'}</span>
                                            </div>
                                            <Separator className="bg-border/50" />
                                            <div className="flex items-center justify-between">
                                                <span className="text-[10px] font-bold text-muted-foreground uppercase">Protection</span>
                                                <Badge className="bg-emerald-500/10 text-emerald-500 border-none text-[9px] font-bold uppercase px-3 h-6">Active</Badge>
                                            </div>
                                        </CardContent>
                                    </Card>

                                    <Card className="bg-blue-500/[0.03] border-border border-dashed">
                                        <CardContent className="p-8 flex flex-col items-center text-center gap-4">
                                            <div className="p-4 bg-blue-500/10 rounded-full">
                                                <ShieldCheck size={32} className="text-blue-500" />
                                            </div>
                                            <div>
                                                <h4 className="text-[11px] font-bold uppercase tracking-widest text-foreground mb-2">Automated Sentinel</h4>
                                                <p className="text-[10px] text-muted-foreground font-medium leading-relaxed uppercase">Threat monitoring and protocol alerts are operational in the dedicated background thread.</p>
                                            </div>
                                        </CardContent>
                                    </Card>
                                </div>
                            </div>
                        </TabsContent>

                        <TabsContent value="logs" className="space-y-6">
                            <div className="flex flex-col lg:flex-row gap-8 h-auto lg:h-[750px] items-stretch">
                                {/* Side Nav - Soft Darker than main */}
                                <Card className="hidden lg:flex flex-col w-72 bg-secondary/20 border-border shadow-sm h-full">
                                    <CardHeader className="p-6 pb-2">
                                        <CardTitle className="text-[10px] font-bold uppercase tracking-[0.4em] text-muted-foreground">Data Streams</CardTitle>
                                    </CardHeader>
                                    <CardContent className="p-3 space-y-1">
                                        <Button
                                            onClick={() => setLogTab('system')}
                                            variant={logTab === 'system' ? 'secondary' : 'ghost'}
                                            className={`w-full justify-start rounded-xl font-bold text-xs uppercase h-12 transition-all gap-3 ${logTab === 'system' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground'}`}
                                        >
                                            <Terminal size={18} className={logTab === 'system' ? 'text-blue-500' : 'text-slate-600'} /> System Logs
                                        </Button>
                                        <Button
                                            onClick={() => setLogTab('nginx_access')}
                                            variant={logTab === 'nginx_access' ? 'secondary' : 'ghost'}
                                            className={`w-full justify-start rounded-xl font-bold text-xs uppercase h-12 transition-all gap-3 ${logTab === 'nginx_access' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground'}`}
                                        >
                                            <Globe size={18} className={logTab === 'nginx_access' ? 'text-emerald-500' : 'text-slate-600'} /> Nginx Access
                                        </Button>
                                        <Button
                                            onClick={() => setLogTab('nginx_error')}
                                            variant={logTab === 'nginx_error' ? 'secondary' : 'ghost'}
                                            className={`w-full justify-start rounded-xl font-bold text-xs uppercase h-12 transition-all gap-3 ${logTab === 'nginx_error' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground'}`}
                                        >
                                            <AlertTriangle size={18} className={logTab === 'nginx_error' ? 'text-rose-500' : 'text-slate-600'} /> Nginx Errors
                                        </Button>
                                    </CardContent>
                                    <div className="mt-auto p-6 space-y-4">
                                        <Separator className="bg-border/50" />
                                        <div className="bg-background/50 p-4 rounded-xl border border-border">
                                            <div className="flex items-center gap-2 mb-2">
                                                <CircleDot size={10} className="text-emerald-500 animate-pulse" />
                                                <span className="text-[9px] font-bold uppercase text-muted-foreground">Pipe Connectivity</span>
                                            </div>
                                            <p className="text-[9px] text-muted-foreground font-medium italic">Establishing persistent SSE tunnel to remote log buffers...</p>
                                        </div>
                                    </div>
                                </Card>

                                {/* Mobile Log Switcher */}
                                <div className="lg:hidden flex overflow-x-auto gap-2 pb-4 no-scrollbar">
                                    <Button
                                        onClick={() => setLogTab('system')}
                                        className={`rounded-xl font-bold text-[10px] uppercase h-12 px-6 whitespace-nowrap transition-all border ${logTab === 'system' ? 'bg-blue-600 border-transparent text-white' : 'bg-card border-border text-muted-foreground'}`}
                                    >
                                        System Journal
                                    </Button>
                                    <Button
                                        onClick={() => setLogTab('nginx_access')}
                                        className={`rounded-xl font-bold text-[10px] uppercase h-12 px-6 whitespace-nowrap transition-all border ${logTab === 'nginx_access' ? 'bg-blue-600 border-transparent text-white' : 'bg-card border-border text-muted-foreground'}`}
                                    >
                                        Nginx Inbound
                                    </Button>
                                    <Button
                                        onClick={() => setLogTab('nginx_error')}
                                        className={`rounded-xl font-bold text-[10px] uppercase h-12 px-6 whitespace-nowrap transition-all border ${logTab === 'nginx_error' ? 'bg-blue-600 border-transparent text-white' : 'bg-card border-border text-muted-foreground'}`}
                                    >
                                        Nginx Crits
                                    </Button>
                                </div>

                                {/* Terminal Content */}
                                <Card className="flex-1 bg-card/40 border-border shadow-xl overflow-hidden flex flex-col h-[550px] lg:h-full backdrop-blur-3xl">
                                    <CardHeader className="bg-secondary/20 border-b border-border p-4 sm:p-6 flex flex-row items-center justify-between">
                                        <div className="flex items-center gap-4">
                                            <div className="hidden xs:flex gap-2">
                                                <div className="w-3 h-3 rounded-full bg-slate-800"></div>
                                                <div className="w-3 h-3 rounded-full bg-slate-800"></div>
                                                <div className="w-3 h-3 rounded-full bg-slate-800"></div>
                                            </div>
                                            <div className="hidden xs:block h-5 w-px bg-border mx-2"></div>
                                            <div className="flex items-center gap-2 text-[10px] sm:text-[11px] font-bold text-foreground uppercase tracking-widest truncate">
                                                <FileText size={14} className="text-muted-foreground" /> {logs?.[logTab]?.path || 'Locating stream...'}
                                            </div>
                                        </div>
                                        <Badge variant="outline" className="text-[8px] font-bold border-border bg-background/50 text-muted-foreground uppercase px-3 h-7 tracking-widest">LIVE PIPE</Badge>
                                    </CardHeader>
                                    <CardContent className="p-0 flex-1 overflow-hidden">
                                        <ScrollArea className="h-full w-full p-6 sm:p-10 bg-[#00000010]">
                                            <pre className="text-blue-50/90 font-mono text-[11px] sm:text-xs md:text-sm whitespace-pre-wrap leading-relaxed tracking-tight selection:bg-blue-500/20">
                                                {/* Use muted-foreground for older lines or formatting - currently showing all white for readability */}
                                                {logs?.[logTab]?.content || 'Bufferizing incoming telemetry...'}
                                            </pre>
                                            <div className="h-20"></div>
                                        </ScrollArea>
                                    </CardContent>
                                    <div className="px-6 py-4 flex items-center justify-between bg-background/50 border-t border-border text-muted-foreground text-[10px] font-bold uppercase tracking-widest italic">
                                        <div className="flex items-center gap-3">
                                            <span className="w-2 h-2 rounded-full bg-blue-500 animate-pulse"></span>
                                            <span>Stream Connected</span>
                                        </div>
                                        <RefreshCcw size={12} className="animate-spin-slow opacity-50" />
                                    </div>
                                </Card>
                            </div>
                        </TabsContent>
                    </Tabs>
                </main>
            </div>

            <footer className="mt-auto p-10 sm:p-16 border-t border-border bg-card/30 relative overflow-hidden backdrop-blur-xl">
                <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-12">
                    <div className="flex items-center gap-12 sm:gap-16">
                        <div className="flex flex-col">
                            <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-[0.4em] mb-2">Primary Core</span>
                            <span className="text-sm font-black text-foreground italic uppercase">AcmaDash Engine v{Version}</span>
                        </div>
                        <div className="hidden xs:block h-10 w-px bg-border"></div>
                        <div className="flex flex-col">
                            <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-[0.4em] mb-2">Modern Slate UI</span>
                            <span className="text-sm font-bold text-muted-foreground uppercase tracking-widest">Premium Optimized</span>
                        </div>
                    </div>

                    <div className="flex flex-col items-center md:items-end">
                        <div className="flex items-center gap-3 mb-1">
                            <span className="text-[11px] font-bold text-foreground uppercase tracking-widest italic">Protected Ecosystem</span>
                        </div>
                        <span className="text-[10px] font-black text-muted-foreground uppercase tracking-[0.3em]">&copy; 2024 AcmaTvirus Intelligence Ops</span>
                    </div>
                </div>
            </footer>

            <style>{`
                @keyframes spin-slow {
                    from { transform: rotate(0deg); }
                    to { transform: rotate(360deg); }
                }
                .animate-spin-slow {
                    animation: spin-slow 8s linear infinite;
                }
                .no-scrollbar::-webkit-scrollbar {
                    display: none;
                }
                .no-scrollbar {
                    -ms-overflow-style: none;
                    scrollbar-width: none;
                }
            `}</style>
        </div>
    );
};

export default App;
