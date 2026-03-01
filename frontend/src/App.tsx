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

const Version = "1.1.5-Slate-Final";

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
        <div className="min-h-screen bg-background text-foreground flex flex-col font-sans selection:bg-primary/20 antialiased">
            {/* Ambient Overlays */}
            <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10 opacity-30">
                <div className="absolute top-0 left-0 w-full h-full bg-[radial-gradient(circle_at_50%_50%,rgba(59,130,246,0.05),transparent_50%)]"></div>
            </div>

            <div className="flex flex-1 overflow-hidden relative">
                {/* Desktop Sidebar */}
                <aside className="hidden lg:flex flex-col w-20 bg-card border-r border-border items-center py-8 gap-12 backdrop-blur-md">
                    <div className="w-12 h-12 bg-primary/5 border border-primary/20 rounded-2xl flex items-center justify-center shadow-sm">
                        <Server size={22} className="text-primary" />
                    </div>
                    <nav className="flex flex-col gap-8">
                        <Button variant="ghost" size="icon" className="hover:bg-secondary rounded-2xl transition-all">
                            <LayoutDashboard size={20} className="text-primary" />
                        </Button>
                        <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-foreground rounded-2xl transition-all">
                            <Activity size={20} />
                        </Button>
                        <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-foreground rounded-2xl transition-all">
                            <Terminal size={20} />
                        </Button>
                    </nav>
                </aside>

                {/* Mobile Menu Backdrop */}
                {isMobileMenuOpen && (
                    <div className="lg:hidden fixed inset-0 bg-background/80 backdrop-blur-sm z-40 transition-all duration-300" onClick={() => setIsMobileMenuOpen(false)}></div>
                )}

                {/* Mobile Sidebar */}
                <aside className={`lg:hidden fixed left-0 top-0 h-full w-72 bg-card z-50 transform transition-transform duration-500 border-r border-border ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}`}>
                    <div className="p-8 flex flex-col h-full">
                        <div className="flex items-center justify-between mb-12">
                            <div className="flex items-center gap-3">
                                <div className="w-9 h-9 bg-primary/10 border border-primary/20 rounded-lg flex items-center justify-center">
                                    <Server className="text-primary" size={18} />
                                </div>
                                <span className="font-bold uppercase tracking-tight text-xl">Acma<span className="text-primary font-black">Dash</span></span>
                            </div>
                            <Button variant="ghost" size="icon" className="text-muted-foreground rounded-xl" onClick={() => setIsMobileMenuOpen(false)}>
                                <X size={20} />
                            </Button>
                        </div>
                        <nav className="flex flex-col gap-3 flex-1">
                            <Button variant="secondary" className="justify-start gap-4 text-xs font-bold uppercase tracking-widest h-14 rounded-2xl px-6 border border-border/50">
                                <LayoutDashboard size={18} className="text-primary" /> Dashboard
                            </Button>
                            <Button variant="ghost" className="justify-start gap-4 text-xs font-bold uppercase tracking-widest text-muted-foreground hover:text-foreground h-14 rounded-2xl px-6">
                                <Activity size={18} /> Monitoring
                            </Button>
                            <Button variant="ghost" className="justify-start gap-4 text-xs font-bold uppercase tracking-widest text-muted-foreground hover:text-foreground h-14 rounded-2xl px-6">
                                <Terminal size={18} /> Logs Terminal
                            </Button>
                        </nav>
                        <div className="mt-auto">
                            <div className="bg-secondary/50 border border-border p-4 rounded-2xl">
                                <Badge variant="outline" className="text-[8px] font-black tracking-widest mb-2 border-primary/50 text-primary">ENCRYPTED</Badge>
                                <p className="text-[10px] text-muted-foreground uppercase leading-tight font-bold">SSE Tunnel Active</p>
                            </div>
                        </div>
                    </div>
                </aside>

                <main className="flex-1 overflow-y-auto p-4 sm:p-6 md:p-10 space-y-10 bg-background">
                    <header className="flex flex-col md:flex-row items-center justify-between gap-6">
                        <div className="flex items-center justify-between w-full md:w-auto">
                            <div className="flex items-center gap-4 sm:gap-6">
                                <Button variant="outline" size="icon" className="lg:hidden bg-card border-border rounded-xl w-10 h-10 shadow-sm" onClick={() => setIsMobileMenuOpen(true)}>
                                    <Menu size={20} />
                                </Button>
                                <div>
                                    <h1 className="text-2xl sm:text-3xl font-black tracking-tight flex items-center gap-3 uppercase text-foreground">
                                        Acma<span className="text-primary">Dash</span>
                                        <Badge variant="outline" className="hidden xs:inline-flex border-primary/20 text-primary bg-primary/5 h-6 px-2 text-[10px] font-bold">SLATE V2</Badge>
                                    </h1>
                                    <p className="text-[10px] sm:text-xs text-muted-foreground font-black uppercase tracking-[0.2em] mt-1">Infrastructure Control Center</p>
                                </div>
                            </div>
                        </div>

                        <div className="flex items-center gap-3 w-full md:w-auto">
                            <div className="hidden md:flex items-center gap-3 bg-secondary px-5 py-2.5 rounded-2xl border border-border">
                                <div className={`w-2 h-2 rounded-full ${connected ? "bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.2)]" : "bg-amber-500"}`}></div>
                                <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">
                                    {connected ? "Stream Established" : "Connecting..."}
                                </span>
                            </div>
                            <Button size="sm" variant="outline" className="rounded-2xl border-border bg-secondary/50 hover:bg-secondary flex-1 md:flex-initial text-[10px] sm:text-xs h-10 px-6 font-bold uppercase shadow-sm">
                                <RefreshCcw size={14} className={`mr-2 ${connected ? "animate-spin-slow text-primary" : ""}`} /> Uptime: {stats ? formatUptime(stats.uptime) : 'LOADING'}
                            </Button>
                        </div>
                    </header>

                    <Tabs defaultValue="overview" className="space-y-8 animate-in fade-in duration-500">
                        <div className="flex flex-col sm:flex-row items-center justify-between gap-6">
                            <TabsList className="bg-secondary border border-border p-1 rounded-2xl w-full sm:w-auto shadow-sm">
                                <TabsTrigger value="overview" className="flex-1 sm:flex-initial data-[state=active]:bg-card data-[state=active]:shadow-sm rounded-xl px-8 font-bold text-[10px] tracking-widest uppercase h-9 transition-all">Overview</TabsTrigger>
                                <TabsTrigger value="logs" className="flex-1 sm:flex-initial data-[state=active]:bg-card data-[state=active]:shadow-sm rounded-xl px-8 font-bold text-[10px] tracking-widest uppercase h-9 transition-all">Terminal</TabsTrigger>
                            </TabsList>
                            <div className="flex items-center gap-3 bg-secondary/30 px-4 py-2 rounded-xl text-muted-foreground text-[10px] font-black uppercase border border-border">
                                <Monitor size={14} className="text-primary/70" /> {stats?.hostname || 'Resolving...'}
                            </div>
                        </div>

                        <TabsContent value="overview" className="space-y-8">
                            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
                                {/* Dashboard Cards - Standardized to bg-card */}
                                <Card className="bg-card border-border hover:border-primary/30 transition-all duration-300 shadow-sm relative overflow-hidden group">
                                    <CardHeader className="pb-3 p-6 pt-5 flex flex-row items-center justify-between">
                                        <CardTitle className="text-[10px] font-black text-muted-foreground uppercase tracking-[0.3em]">C-Processor</CardTitle>
                                        <Cpu size={16} className="text-primary/70" />
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0 space-y-4">
                                        <div className="text-4xl font-black italic tracking-tighter text-foreground">{stats?.cpu.toFixed(1) || '0.0'}<span className="text-sm text-muted-foreground ml-1 uppercase">%</span></div>
                                        <Progress value={stats?.cpu || 0} className="h-2 bg-secondary rounded-full"
                                            indicatorClassName={stats && stats.cpu > 80 ? "bg-destructive shadow-[0_0_8px_rgba(239,68,68,0.2)]" : "bg-primary shadow-[0_0_8px_rgba(255,255,255,0.1)]"}
                                        />
                                    </CardContent>
                                    <CardFooter className="p-4 py-3 bg-secondary/20 text-[9px] text-muted-foreground font-black uppercase tracking-[0.2em] border-t border-border/20">
                                        Core Load Signal
                                    </CardFooter>
                                </Card>

                                <Card className="bg-card border-border hover:border-emerald-500/30 transition-all duration-300 shadow-sm relative overflow-hidden group">
                                    <CardHeader className="pb-3 p-6 pt-5 flex flex-row items-center justify-between">
                                        <CardTitle className="text-[10px] font-black text-muted-foreground uppercase tracking-[0.3em]">M-Storage</CardTitle>
                                        <Database size={16} className="text-emerald-500/70" />
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0 space-y-4">
                                        <div className="text-4xl font-black italic tracking-tighter text-foreground">{stats?.ram.toFixed(1) || '0.0'}<span className="text-sm text-muted-foreground ml-1 uppercase">%</span></div>
                                        <Progress value={stats?.ram || 0} className="h-2 bg-secondary rounded-full"
                                            indicatorClassName="bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.1)]"
                                        />
                                    </CardContent>
                                    <CardFooter className="p-4 py-3 bg-secondary/20 text-[9px] text-muted-foreground font-black uppercase tracking-[0.2em] flex justify-between w-full border-t border-border/20">
                                        <span>USED: {stats ? formatBytes(stats.ram_used) : '0.00'}</span>
                                        <span className="opacity-40">OF {stats ? formatBytes(stats.ram_total).split(' ')[0] : '0.00'}</span>
                                    </CardFooter>
                                </Card>

                                <Card className="bg-card border-border hover:border-amber-500/30 transition-all duration-300 shadow-sm relative overflow-hidden group">
                                    <CardHeader className="pb-3 p-6 pt-5 flex flex-row items-center justify-between">
                                        <CardTitle className="text-[10px] font-black text-muted-foreground uppercase tracking-[0.3em]">H-Storage</CardTitle>
                                        <HardDrive size={16} className="text-amber-500/70" />
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0 space-y-4">
                                        <div className="text-4xl font-black italic tracking-tighter text-foreground">{stats?.disk.toFixed(1) || '0.0'}<span className="text-sm text-muted-foreground ml-1 uppercase">%</span></div>
                                        <Progress value={stats?.disk || 0} className="h-2 bg-secondary rounded-full"
                                            indicatorClassName="bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.1)]"
                                        />
                                    </CardContent>
                                    <CardFooter className="p-4 py-3 bg-secondary/20 text-[9px] text-muted-foreground font-black uppercase tracking-[0.2em] border-t border-border/20">
                                        REMAIN: {stats ? formatBytes(stats.disk_total - stats.disk_used) : '---'}
                                    </CardFooter>
                                </Card>

                                <Card className="bg-card border-border hover:border-indigo-500/30 transition-all duration-300 shadow-sm relative overflow-hidden group">
                                    <CardHeader className="pb-3 p-6 pt-5 flex flex-row items-center justify-between">
                                        <CardTitle className="text-[10px] font-black text-muted-foreground uppercase tracking-[0.3em]">N-Interface</CardTitle>
                                        <CloudLightning size={16} className="text-indigo-500/70" />
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0 space-y-4">
                                        <div className="text-3xl font-black italic tracking-tighter text-foreground truncate drop-shadow-sm">
                                            {stats ? formatBytes(stats.net_recv) : '0.00 GB'}
                                        </div>
                                        <div className="text-[10px] text-muted-foreground font-black uppercase tracking-widest opacity-60">Inbound Load</div>
                                    </CardContent>
                                    <CardFooter className="p-4 py-3 bg-secondary/20 text-[9px] text-muted-foreground font-black uppercase tracking-[0.2em] flex justify-between w-full border-t border-border/20">
                                        <span>TX: {stats ? formatBytes(stats.net_sent).split(' ')[0] : '0.00'}</span>
                                        <span className="flex items-center gap-2 text-primary/70">{stats?.connections || 0} CONNS</span>
                                    </CardFooter>
                                </Card>
                            </div>

                            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                                <Card className="lg:col-span-2 bg-card border-border shadow-sm">
                                    <CardHeader className="flex flex-row items-center justify-between p-6 sm:p-8 pb-4">
                                        <div>
                                            <CardTitle className="text-lg font-black uppercase tracking-[0.2em] text-foreground italic">Spectral Telemetry</CardTitle>
                                            <CardDescription className="text-[10px] uppercase font-black text-muted-foreground mt-1">CPU Oscillation / 60s Buffer</CardDescription>
                                        </div>
                                        <Badge variant="secondary" className="bg-primary/5 text-primary border border-primary/20 text-[9px] font-black px-3 h-7 tracking-widest">LIVE DATA</Badge>
                                    </CardHeader>
                                    <CardContent className="h-[300px] sm:h-[450px] p-0 overflow-hidden pt-6">
                                        <ResponsiveContainer width="100%" height="100%">
                                            <AreaChart data={history} margin={{ left: -10, right: 10, top: 0, bottom: 0 }}>
                                                <defs>
                                                    <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                                                        <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.15} />
                                                        <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0} />
                                                    </linearGradient>
                                                </defs>
                                                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="hsl(var(--border) / 0.1)" />
                                                <XAxis dataKey="time" stroke="hsl(var(--muted-foreground))" fontSize={9} tickLine={false} axisLine={false} minTickGap={50} tick={{ fontWeight: '900' }} />
                                                <YAxis stroke="hsl(var(--muted-foreground))" fontSize={9} tickLine={false} axisLine={false} domain={[0, 100]} tick={{ fontWeight: '900' }} />
                                                <Tooltip
                                                    contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '12px', boxShadow: '0 10px 15px -3px rgba(0,0,0,0.1)' }}
                                                    itemStyle={{ color: 'hsl(var(--primary))', fontWeight: '900', fontSize: '13px' }}
                                                    labelStyle={{ color: 'hsl(var(--muted-foreground))', fontSize: '10px', fontWeight: '900', textTransform: 'uppercase' }}
                                                />
                                                <Area type="monotone" dataKey="cpu" stroke="hsl(var(--primary))" strokeWidth={3} fillOpacity={1} fill="url(#colorCpu)" animationDuration={300} isAnimationActive={history.length < 2} />
                                            </AreaChart>
                                        </ResponsiveContainer>
                                    </CardContent>
                                </Card>

                                <div className="space-y-6">
                                    <Card className="bg-card border-border shadow-sm">
                                        <CardHeader className="pb-4 p-8">
                                            <CardTitle className="text-[10px] font-black italic tracking-[0.4em] uppercase text-muted-foreground">Local Context</CardTitle>
                                        </CardHeader>
                                        <CardContent className="space-y-6 p-8 pt-0">
                                            <div className="flex items-center justify-between">
                                                <span className="text-[9px] font-black text-muted-foreground uppercase tracking-widest">OS Base</span>
                                                <span className="text-[11px] font-black text-foreground uppercase italic px-3 py-1 bg-secondary rounded-lg border border-border/50">{stats?.platform || 'LINUX'}</span>
                                            </div>
                                            <Separator className="bg-border/30" />
                                            <div className="flex items-center justify-between">
                                                <span className="text-[9px] font-black text-muted-foreground uppercase tracking-widest">Build ID</span>
                                                <span className="text-[10px] font-black text-muted-foreground truncate max-w-[130px] font-mono">{stats?.kernel || '---'}</span>
                                            </div>
                                            <Separator className="bg-border/30" />
                                            <div className="flex items-center justify-between">
                                                <span className="text-[9px] font-black text-muted-foreground uppercase tracking-widest">Sentinel</span>
                                                <Badge className="bg-emerald-500/10 text-emerald-500 border border-emerald-500/20 text-[9px] font-black uppercase px-3 h-6">Operational</Badge>
                                            </div>
                                        </CardContent>
                                    </Card>

                                    <Card className="bg-primary/5 border-border border-dashed shadow-sm">
                                        <CardContent className="p-10 flex flex-col items-center text-center gap-6">
                                            <div className="p-5 bg-card border border-border rounded-full shadow-inner ring-4 ring-primary/5">
                                                <ShieldCheck size={36} className="text-primary/70" />
                                            </div>
                                            <div>
                                                <h4 className="text-[11px] font-black uppercase tracking-[0.3em] text-foreground mb-3 italic underline decoration-primary/30 decoration-2 underline-offset-4">Security Protocol</h4>
                                                <p className="text-[10px] text-muted-foreground font-bold leading-relaxed uppercase tracking-tighter">Automated DDoS filtration and global alerts initialized in secondary thread.</p>
                                            </div>
                                        </CardContent>
                                    </Card>
                                </div>
                            </div>
                        </TabsContent>

                        <TabsContent value="logs" className="space-y-6">
                            <div className="flex flex-col lg:flex-row gap-8 h-auto lg:h-[750px] items-stretch">
                                {/* Side Nav - Consistent Sidebar UI */}
                                <Card className="hidden lg:flex flex-col w-72 bg-card border-border shadow-sm h-full overflow-hidden">
                                    <CardHeader className="p-8 pb-4">
                                        <CardTitle className="text-[10px] font-black uppercase tracking-[0.5em] text-muted-foreground opacity-60">Source Pipelines</CardTitle>
                                    </CardHeader>
                                    <CardContent className="p-4 space-y-2">
                                        <Button
                                            onClick={() => setLogTab('system')}
                                            variant={logTab === 'system' ? 'secondary' : 'ghost'}
                                            className={`w-full justify-start rounded-2xl font-black text-[11px] uppercase italic tracking-widest h-14 transition-all gap-4 border ${logTab === 'system' ? 'bg-primary/10 border-primary/20 text-foreground' : 'text-muted-foreground border-transparent hover:bg-secondary'}`}
                                        >
                                            <Terminal size={18} className={logTab === 'system' ? 'text-primary' : 'text-muted-foreground'} /> Sys Journal
                                        </Button>
                                        <Button
                                            onClick={() => setLogTab('nginx_access')}
                                            variant={logTab === 'nginx_access' ? 'secondary' : 'ghost'}
                                            className={`w-full justify-start rounded-2xl font-black text-[11px] uppercase italic tracking-widest h-14 transition-all gap-4 border ${logTab === 'nginx_access' ? 'bg-primary/10 border-primary/20 text-foreground' : 'text-muted-foreground border-transparent hover:bg-secondary'}`}
                                        >
                                            <Globe size={18} className={logTab === 'nginx_access' ? 'text-emerald-500' : 'text-muted-foreground'} /> Web Inbound
                                        </Button>
                                        <Button
                                            onClick={() => setLogTab('nginx_error')}
                                            variant={logTab === 'nginx_error' ? 'secondary' : 'ghost'}
                                            className={`w-full justify-start rounded-2xl font-black text-[11px] uppercase italic tracking-widest h-14 transition-all gap-4 border ${logTab === 'nginx_error' ? 'bg-primary/10 border-primary/20 text-foreground' : 'text-muted-foreground border-transparent hover:bg-secondary'}`}
                                        >
                                            <AlertTriangle size={18} className={logTab === 'nginx_error' ? 'text-rose-500' : 'text-muted-foreground'} /> Web Crits
                                        </Button>
                                    </CardContent>
                                    <div className="mt-auto p-8 border-t border-border/10 bg-secondary/10">
                                        <div className="flex items-center gap-3 mb-3">
                                            <CircleDot size={12} className="text-emerald-500 animate-pulse" />
                                            <span className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Buffer Health</span>
                                        </div>
                                        <div className="h-1 w-full bg-border/20 rounded-full overflow-hidden">
                                            <div className="h-full w-[85%] bg-primary/40"></div>
                                        </div>
                                    </div>
                                </Card>

                                {/* Mobile Log Switcher */}
                                <div className="lg:hidden flex overflow-x-auto gap-3 pb-4 no-scrollbar">
                                    {(['system', 'nginx_access', 'nginx_error'] as const).map((tab) => (
                                        <Button
                                            key={tab}
                                            onClick={() => setLogTab(tab)}
                                            className={`rounded-2xl font-black text-[10px] uppercase h-12 px-8 whitespace-nowrap transition-all border ${logTab === tab ? 'bg-primary border-transparent text-primary-foreground shadow-lg' : 'bg-card border-border text-muted-foreground'}`}
                                        >
                                            {tab === 'system' ? 'System' : tab === 'nginx_access' ? 'Web Access' : 'Web Errors'}
                                        </Button>
                                    ))}
                                </div>

                                {/* Terminal Content - Refined Slate Background, No Pitch Black */}
                                <Card className="flex-1 bg-secondary/10 border-border shadow-2xl overflow-hidden flex flex-col h-[550px] lg:h-full backdrop-blur-xl group">
                                    <CardHeader className="bg-card border-b border-border p-4 sm:p-8 flex flex-row items-center justify-between shadow-sm">
                                        <div className="flex items-center gap-4">
                                            <div className="hidden xs:flex gap-2">
                                                <div className="w-3 h-3 rounded-full bg-border/40"></div>
                                                <div className="w-3 h-3 rounded-full bg-border/40"></div>
                                                <div className="w-3 h-3 rounded-full bg-border/40"></div>
                                            </div>
                                            <div className="hidden xs:block h-6 w-px bg-border/50 mx-2"></div>
                                            <div className="flex flex-col">
                                                <span className="text-[8px] font-black text-muted-foreground uppercase tracking-[0.3em] mb-1">Tailstream Output</span>
                                                <div className="flex items-center gap-3 text-[10px] sm:text-[13px] font-black text-foreground uppercase tracking-tight truncate">
                                                    <FileText size={16} className="text-primary/60" /> {logs?.[logTab]?.path || 'Loading stream...'}
                                                </div>
                                            </div>
                                        </div>
                                        <Badge variant="outline" className="text-[8px] font-black border-border bg-secondary/50 text-muted-foreground uppercase px-4 h-8 tracking-widest hidden sm:flex">SSE TUNNEL ONLINE</Badge>
                                    </CardHeader>
                                    <CardContent className="p-0 flex-1 overflow-hidden relative">
                                        <ScrollArea className="h-full w-full p-6 sm:p-12">
                                            <pre className="text-foreground/90 font-mono text-[11px] sm:text-xs md:text-sm whitespace-pre-wrap leading-relaxed tracking-tight selection:bg-primary/20 drop-shadow-sm">
                                                {logs?.[logTab]?.content || 'Bufferizing encrypted telemetric stream...'}
                                            </pre>
                                            <div className="h-20"></div>
                                        </ScrollArea>
                                    </CardContent>
                                    <Separator className="bg-border/30" />
                                    <div className="px-8 py-5 flex items-center justify-between bg-card text-muted-foreground text-[10px] font-black uppercase tracking-[0.3em] italic shadow-inner">
                                        <div className="flex items-center gap-4">
                                            <span className="w-2 h-2 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.4)]"></span>
                                            <span className="hidden xs:inline">Persistent Link Active</span>
                                            <span className="xs:hidden">Live</span>
                                        </div>
                                        <div className="flex items-center gap-6">
                                            <span className="hidden sm:inline opacity-30">Auto-tail: 30L</span>
                                            <RefreshCcw size={14} className="animate-spin-slow text-primary/60" />
                                        </div>
                                    </div>
                                </Card>
                            </div>
                        </TabsContent>
                    </Tabs>
                </main>
            </div>

            <footer className="mt-auto p-12 sm:p-20 border-t border-border bg-card/60 backdrop-blur-3xl relative overflow-hidden">
                <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-12">
                    <div className="flex items-center gap-12 sm:gap-20">
                        <div className="flex flex-col">
                            <span className="text-[10px] font-black text-muted-foreground uppercase tracking-[0.6em] mb-3">Project Engine</span>
                            <span className="text-sm font-black text-foreground italic uppercase tracking-widest">AcmaDash Core <span className="text-primary ml-1 pr-2 border-r-2 border-primary/20">v{Version}</span></span>
                        </div>
                        <div className="hidden sm:flex flex-col">
                            <span className="text-[10px] font-black text-muted-foreground uppercase tracking-[0.6em] mb-3">UI Manifest</span>
                            <span className="text-sm font-black text-muted-foreground uppercase tracking-widest italic">Slate Elite v2024</span>
                        </div>
                    </div>

                    <div className="flex flex-col items-center md:items-end">
                        <div className="flex items-center gap-4 mb-3">
                            <div className="h-1.5 w-1.5 rounded-full bg-primary shadow-[0_0_12px_rgba(255,255,255,0.2)] animate-pulse"></div>
                            <span className="text-[11px] font-black text-foreground uppercase tracking-[0.4em] italic leading-none">Security Node Live</span>
                        </div>
                        <span className="text-[9px] font-black text-muted-foreground uppercase tracking-[0.4em] mt-3">&copy; 2024 AcmaTvirus Intelligence Systems</span>
                    </div>
                </div>
            </footer>

            <style>{`
                @keyframes spin-slow {
                    from { transform: rotate(0deg); }
                    to { transform: rotate(360deg); }
                }
                .animate-spin-slow {
                    animation: spin-slow 10s linear infinite;
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
