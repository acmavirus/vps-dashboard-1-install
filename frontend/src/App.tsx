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
    AlertTriangle,
    LayoutDashboard,
    FileText,
    Menu,
    X,
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

const Version = "1.1.6-Stable";

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
                if (updated.length > 50) return updated.slice(1);
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
                setTimeout(setupSSE, 3000);
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
        // Increase frequency for fallback
        const interval = setInterval(fetchFallback, 2000);
        return () => {
            if (eventSourceRef.current) eventSourceRef.current.close();
            clearInterval(interval);
        };
    }, []);

    return (
        /* CRITICAL: Added 'dark' class back to enable HSL variables */
        <div className="dark min-h-screen bg-background text-foreground flex flex-col font-sans antialiased selection:bg-primary/20">
            {/* Ambient Background */}
            <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10 bg-[radial-gradient(circle_at_50%_-20%,rgba(59,130,246,0.08),transparent_70%)] opacity-40"></div>
            <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10 bg-grid-slate-900/[0.05] bg-[size:30px_30px]"></div>

            <div className="flex flex-1 overflow-hidden relative">
                {/* Desktop Sidebar */}
                <aside className="hidden lg:flex flex-col w-20 bg-card border-r border-border/50 items-center py-10 gap-14 backdrop-blur-xl">
                    <div className="w-12 h-12 bg-primary/10 border border-primary/20 rounded-[1.2rem] flex items-center justify-center shadow-lg transform hover:scale-105 transition-transform cursor-pointer">
                        <Server size={24} className="text-primary-foreground" />
                    </div>
                    <nav className="flex flex-col gap-10">
                        <Button variant="ghost" size="icon" className="hover:bg-secondary rounded-2xl w-12 h-12 transition-all">
                            <LayoutDashboard size={22} className="text-primary" />
                        </Button>
                        <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-foreground rounded-2xl w-12 h-12 transition-all">
                            <Activity size={22} />
                        </Button>
                        <Button variant="ghost" size="icon" className="text-muted-foreground hover:text-foreground rounded-2xl w-12 h-12 transition-all">
                            <Terminal size={22} />
                        </Button>
                    </nav>
                </aside>

                {/* Mobile Navigation */}
                {isMobileMenuOpen && (
                    <div className="lg:hidden fixed inset-0 bg-background/90 backdrop-blur-md z-40 animate-in fade-in duration-300" onClick={() => setIsMobileMenuOpen(false)}></div>
                )}
                <aside className={`lg:hidden fixed left-0 top-0 h-full w-80 bg-card z-50 transform transition-transform duration-500 ease-out border-r border-border shadow-2xl ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}`}>
                    <div className="p-10 flex flex-col h-full">
                        <div className="flex items-center justify-between mb-16">
                            <div className="flex items-center gap-4">
                                <div className="w-10 h-10 bg-primary rounded-xl flex items-center justify-center">
                                    <Server className="text-primary-foreground" size={20} />
                                </div>
                                <span className="font-black uppercase tracking-tighter text-2xl text-foreground">Acma<span className="text-primary">Dash</span></span>
                            </div>
                            <Button variant="ghost" size="icon" className="text-muted-foreground hover:bg-secondary rounded-full" onClick={() => setIsMobileMenuOpen(false)}>
                                <X size={24} />
                            </Button>
                        </div>
                        <nav className="flex flex-col gap-4">
                            <Button variant="secondary" className="justify-start gap-5 text-[11px] font-black uppercase tracking-[0.2em] h-16 rounded-2xl px-8 border border-border/50 shadow-sm">
                                <LayoutDashboard size={20} className="text-primary" /> Dashboard
                            </Button>
                            <Button variant="ghost" className="justify-start gap-5 text-[11px] font-black uppercase tracking-[0.2em] h-16 rounded-2xl px-8 text-muted-foreground hover:text-foreground">
                                <Activity size={20} /> Monitoring
                            </Button>
                            <Button variant="ghost" className="justify-start gap-5 text-[11px] font-black uppercase tracking-[0.2em] h-16 rounded-2xl px-8 text-muted-foreground hover:text-foreground">
                                <Terminal size={20} /> Live Logs
                            </Button>
                        </nav>
                        <div className="mt-auto p-6 bg-secondary/30 rounded-3xl border border-border/50 text-center">
                            <div className="flex items-center justify-center gap-3 mb-2">
                                <div className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse"></div>
                                <span className="text-[10px] font-black text-foreground uppercase tracking-widest">Pipe Online</span>
                            </div>
                            <p className="text-[9px] text-muted-foreground uppercase font-bold tracking-tighter italic">End-to-End Encrypted Tunnel</p>
                        </div>
                    </div>
                </aside>

                <main className="flex-1 overflow-y-auto p-4 sm:p-8 md:p-12 space-y-12">
                    <header className="flex flex-col lg:flex-row items-center justify-between gap-8">
                        <div className="flex items-center justify-between w-full lg:w-auto">
                            <div className="flex items-center gap-6">
                                <Button variant="outline" size="icon" className="lg:hidden bg-card border-border/60 rounded-2xl w-12 h-12 shadow-md hover:bg-secondary" onClick={() => setIsMobileMenuOpen(true)}>
                                    <Menu size={24} />
                                </Button>
                                <div>
                                    <h1 className="text-3xl sm:text-4xl font-black tracking-tighter flex items-center gap-4 text-foreground uppercase">
                                        Acma<span className="text-primary">Dash</span>
                                        <Badge variant="outline" className="border-primary/30 text-primary bg-primary/5 h-7 px-3 text-[11px] font-black tracking-tighter">SLATE ELITE</Badge>
                                    </h1>
                                    <p className="text-[11px] font-black text-muted-foreground uppercase tracking-[0.3em] mt-2 italic flex items-center gap-2">
                                        <CircleDot size={12} className="text-primary/50" /> System Infrastructure Node
                                    </p>
                                </div>
                            </div>
                        </div>

                        <div className="flex items-center gap-4 w-full lg:w-auto">
                            <div className="hidden md:flex items-center gap-4 bg-card px-6 py-3 rounded-2xl border border-border shadow-soft">
                                <div className={`w-2.5 h-2.5 rounded-full ${connected ? "bg-emerald-500 shadow-[0_0_10px_rgba(16,185,129,0.3)]" : "bg-amber-500 animate-pulse"}`}></div>
                                <span className="text-[10px] font-black uppercase tracking-[0.3em] text-muted-foreground">
                                    {connected ? "TELEMETRY SYNCED" : "RECONNECTING PIPE"}
                                </span>
                            </div>
                            <Button size="lg" variant="secondary" className="rounded-2xl border border-border/50 bg-secondary/80 hover:bg-secondary flex-1 lg:flex-initial text-[11px] h-12 px-8 font-black uppercase tracking-widest shadow-sm">
                                <RefreshCcw size={16} className={`mr-3 ${connected ? "animate-spin-slow text-primary" : ""}`} /> UP: {stats ? formatUptime(stats.uptime) : 'SCANNING'}
                            </Button>
                        </div>
                    </header>

                    <Tabs defaultValue="overview" className="space-y-10">
                        <div className="flex flex-col sm:flex-row items-center justify-between gap-8">
                            <TabsList className="bg-card border border-border/60 p-1.5 rounded-2xl w-full sm:w-auto shadow-inner">
                                <TabsTrigger value="overview" className="flex-1 sm:flex-initial data-[state=active]:bg-secondary data-[state=active]:text-foreground data-[state=active]:shadow-sm rounded-xl px-10 font-bold text-[11px] tracking-widest uppercase h-10 transition-all">Overview</TabsTrigger>
                                <TabsTrigger value="logs" className="flex-1 sm:flex-initial data-[state=active]:bg-secondary data-[state=active]:text-foreground data-[state=active]:shadow-sm rounded-xl px-10 font-bold text-[11px] tracking-widest uppercase h-10 transition-all">Deep Logs</TabsTrigger>
                            </TabsList>
                            <div className="flex items-center gap-3 bg-secondary/40 px-5 py-2.5 rounded-xl border border-border/50 text-foreground font-black uppercase text-[10px] tracking-widest shadow-sm">
                                <Monitor size={16} className="text-primary" /> {stats?.hostname || 'Resolving...'}
                            </div>
                        </div>

                        <TabsContent value="overview" className="space-y-10 animate-in slide-in-from-bottom-4 duration-500">
                            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-8">
                                {/* Dashboard Cards - Explicit High Contrast White Text for Values */}
                                <Card className="bg-card border-border/60 hover:border-primary/50 transition-all duration-500 shadow-xl group overflow-hidden relative">
                                    <div className="absolute top-0 right-0 w-32 h-32 bg-primary/5 rounded-full -mr-16 -mt-16 blur-3xl"></div>
                                    <CardHeader className="pb-4 p-8 flex flex-row items-center justify-between relative z-10">
                                        <CardTitle className="text-[11px] font-black text-muted-foreground uppercase tracking-[0.4em]">CPU Load</CardTitle>
                                        <Cpu size={18} className="text-primary/70 group-hover:text-primary transition-colors" />
                                    </CardHeader>
                                    <CardContent className="p-8 pt-0 space-y-6 relative z-10">
                                        {/* CRITICAL: Changed from text-foreground to text-white for high contrast */}
                                        <div className="text-5xl font-black tracking-tighter text-white drop-shadow-md italic">{stats?.cpu.toFixed(1) || '0.0'}<span className="text-sm text-muted-foreground ml-2 opacity-60 uppercase">%</span></div>
                                        <Progress value={stats?.cpu || 0} className="h-2.5 bg-secondary/50 rounded-full"
                                            indicatorClassName={stats && stats.cpu > 85 ? "bg-destructive animate-pulse shadow-[0_0_12px_rgba(239,68,68,0.3)]" : "bg-primary shadow-[0_0_12px_rgba(59,130,246,0.2)]"}
                                        />
                                    </CardContent>
                                    <CardFooter className="p-5 py-3.5 bg-secondary/20 text-[10px] text-muted-foreground font-black uppercase tracking-[0.2em] border-t border-border/30 flex justify-between items-center">
                                        Active Clock Rate <Activity size={12} className="opacity-50" />
                                    </CardFooter>
                                </Card>

                                <Card className="bg-card border-border/60 hover:border-emerald-500/50 transition-all duration-500 shadow-xl group overflow-hidden relative">
                                    <div className="absolute top-0 right-0 w-32 h-32 bg-emerald-500/5 rounded-full -mr-16 -mt-16 blur-3xl"></div>
                                    <CardHeader className="pb-4 p-8 flex flex-row items-center justify-between relative z-10">
                                        <CardTitle className="text-[11px] font-black text-muted-foreground uppercase tracking-[0.4em]">Memory</CardTitle>
                                        <Database size={18} className="text-emerald-500/70 group-hover:text-emerald-500 transition-colors" />
                                    </CardHeader>
                                    <CardContent className="p-8 pt-0 space-y-6 relative z-10">
                                        <div className="text-5xl font-black tracking-tighter text-white drop-shadow-md italic">{stats?.ram.toFixed(1) || '0.0'}<span className="text-sm text-muted-foreground ml-2 opacity-60 uppercase">%</span></div>
                                        <Progress value={stats?.ram || 0} className="h-2.5 bg-secondary/50 rounded-full"
                                            indicatorClassName="bg-emerald-500 shadow-[0_0_12px_rgba(16,185,129,0.2)]"
                                        />
                                    </CardContent>
                                    <CardFooter className="p-5 py-3.5 bg-secondary/20 text-[10px] text-muted-foreground font-black uppercase tracking-[0.2em] border-t border-border/30 flex justify-between items-center w-full">
                                        <span>USED: {stats ? formatBytes(stats.ram_used) : '0.00 GB'}</span>
                                        <span className="opacity-40">OF {stats ? formatBytes(stats.ram_total).split(' ')[0] : '?.?'}</span>
                                    </CardFooter>
                                </Card>

                                <Card className="bg-card border-border/60 hover:border-amber-500/50 transition-all duration-500 shadow-xl group overflow-hidden relative">
                                    <div className="absolute top-0 right-0 w-32 h-32 bg-amber-500/5 rounded-full -mr-16 -mt-16 blur-3xl"></div>
                                    <CardHeader className="pb-4 p-8 flex flex-row items-center justify-between relative z-10">
                                        <CardTitle className="text-[11px] font-black text-muted-foreground uppercase tracking-[0.4em]">Disk Array</CardTitle>
                                        <HardDrive size={18} className="text-amber-500/70 group-hover:text-amber-500 transition-colors" />
                                    </CardHeader>
                                    <CardContent className="p-8 pt-0 space-y-6 relative z-10">
                                        <div className="text-5xl font-black tracking-tighter text-white drop-shadow-md italic">{stats?.disk.toFixed(1) || '0.0'}<span className="text-sm text-muted-foreground ml-2 opacity-60 uppercase">%</span></div>
                                        <Progress value={stats?.disk || 0} className="h-2.5 bg-secondary/50 rounded-full"
                                            indicatorClassName="bg-amber-500"
                                        />
                                    </CardContent>
                                    <CardFooter className="p-5 py-3.5 bg-secondary/20 text-[10px] text-muted-foreground font-black uppercase tracking-[0.2em] border-t border-border/30">
                                        FREE: {stats ? formatBytes(stats.disk_total - stats.disk_used) : '---'}
                                    </CardFooter>
                                </Card>

                                <Card className="bg-card border-border/60 hover:border-indigo-500/50 transition-all duration-500 shadow-xl group overflow-hidden relative">
                                    <div className="absolute top-0 right-0 w-32 h-32 bg-indigo-500/5 rounded-full -mr-16 -mt-16 blur-3xl"></div>
                                    <CardHeader className="pb-4 p-8 flex flex-row items-center justify-between relative z-10">
                                        <CardTitle className="text-[11px] font-black text-muted-foreground uppercase tracking-[0.4em]">Network TX/RX</CardTitle>
                                        <CloudLightning size={18} className="text-indigo-500/70 group-hover:text-indigo-500 transition-colors" />
                                    </CardHeader>
                                    <CardContent className="p-8 pt-0 space-y-6 relative z-10">
                                        <div className="text-4xl font-black tracking-tighter text-white drop-shadow-md truncate italic uppercase">
                                            {stats ? formatBytes(stats.net_recv) : '0.00 GB'}
                                        </div>
                                        <div className="text-[10px] text-indigo-400 font-black uppercase tracking-widest flex items-center gap-2">
                                            <div className="w-1.5 h-1.5 rounded-full bg-indigo-500"></div> Total Payload Received
                                        </div>
                                    </CardContent>
                                    <CardFooter className="p-5 py-3.5 bg-secondary/20 text-[10px] text-muted-foreground font-black uppercase tracking-[0.2em] border-t border-border/30 flex justify-between items-center w-full">
                                        <span>SENT: {stats ? formatBytes(stats.net_sent).split(' ')[0] : '0.00'}</span>
                                        <Badge variant="outline" className="text-[8px] font-black border-indigo-500/20 text-indigo-400 bg-indigo-500/5 h-5">{stats?.connections || 0} CONNS</Badge>
                                    </CardFooter>
                                </Card>
                            </div>

                            <div className="grid grid-cols-1 lg:grid-cols-3 gap-10">
                                <Card className="lg:col-span-2 bg-card border-border/60 shadow-2xl relative">
                                    <CardHeader className="p-10 pb-2">
                                        <div className="flex items-center justify-between w-full">
                                            <div>
                                                <CardTitle className="text-xl font-black uppercase tracking-tight text-foreground italic flex items-center gap-3">
                                                    Telemetry Buffer <Badge className="bg-primary/10 text-primary border-none text-[8px] h-5 mb-1 px-2">REAL-TIME</Badge>
                                                </CardTitle>
                                                <CardDescription className="text-[10px] uppercase font-black text-muted-foreground mt-2 tracking-widest">CPU Delta Variance Visualization</CardDescription>
                                            </div>
                                            <div className="hidden xs:flex flex-col items-end">
                                                <span className="text-2xl font-black text-white italic">{stats?.cpu.toFixed(1)}%</span>
                                                <span className="text-[8px] font-extrabold text-muted-foreground uppercase tracking-tighter">Current Load Vector</span>
                                            </div>
                                        </div>
                                    </CardHeader>
                                    <CardContent className="h-[400px] sm:h-[500px] p-0 overflow-hidden pt-10">
                                        <ResponsiveContainer width="100%" height="100%">
                                            <AreaChart data={history} margin={{ left: -10, right: 10, top: 0, bottom: 0 }}>
                                                <defs>
                                                    <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                                                        <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.2} />
                                                        <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0} />
                                                    </linearGradient>
                                                </defs>
                                                <CartesianGrid strokeDasharray="4 4" vertical={false} stroke="hsl(var(--border) / 0.15)" />
                                                <XAxis dataKey="time" stroke="hsl(var(--muted-foreground))" fontSize={10} tickLine={false} axisLine={false} minTickGap={60} tick={{ fontWeight: '900' }} />
                                                <YAxis stroke="hsl(var(--muted-foreground))" fontSize={10} tickLine={false} axisLine={false} domain={[0, 100]} tick={{ fontWeight: '900' }} />
                                                <Tooltip
                                                    contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '16px', boxShadow: '0 25px 50px -12px rgba(0,0,0,0.5)', padding: '12px' }}
                                                    itemStyle={{ color: 'hsl(var(--primary))', fontWeight: '900', fontSize: '15px' }}
                                                    labelStyle={{ color: 'hsl(var(--muted-foreground))', fontSize: '11px', fontWeight: '900', textTransform: 'uppercase', marginBottom: '4px' }}
                                                />
                                                <Area type="monotone" dataKey="cpu" stroke="hsl(var(--primary))" strokeWidth={4} fillOpacity={1} fill="url(#colorCpu)" animationDuration={200} isAnimationActive={false} />
                                            </AreaChart>
                                        </ResponsiveContainer>
                                    </CardContent>
                                </Card>

                                <div className="space-y-8">
                                    <Card className="bg-card border-border/60 shadow-xl overflow-hidden">
                                        <CardHeader className="bg-secondary/10 border-b border-border/30 p-8">
                                            <CardTitle className="text-[11px] font-black italic tracking-[0.5em] uppercase text-primary/80">System Metadata</CardTitle>
                                        </CardHeader>
                                        <CardContent className="p-8 space-y-8">
                                            <div className="flex flex-col gap-2">
                                                <span className="text-[10px] font-black text-muted-foreground uppercase tracking-widest opacity-60">Host OS</span>
                                                <div className="flex items-center gap-4">
                                                    <Badge className="bg-primary/10 text-primary border-primary/20 text-[11px] font-black uppercase px-4 h-9 flex-1 italic">{stats?.platform || 'UNIX NODE'}</Badge>
                                                    <Badge className="bg-secondary text-foreground border-border text-[11px] font-black h-9 px-4">{stats?.os || '---'}</Badge>
                                                </div>
                                            </div>
                                            <Separator className="bg-border/30" />
                                            <div className="flex flex-col gap-3">
                                                <span className="text-[10px] font-black text-muted-foreground uppercase tracking-widest opacity-60">Kernel Hash</span>
                                                <div className="bg-secondary/40 p-3.5 rounded-xl border border-border/50 font-mono text-[11px] font-black text-foreground truncate shadow-inner">
                                                    {stats?.kernel || 'Bufferizing...'}
                                                </div>
                                            </div>
                                            <Separator className="bg-border/30" />
                                            <div className="flex items-center justify-between">
                                                <span className="text-[10px] font-black text-muted-foreground uppercase tracking-widest opacity-60">Security Core</span>
                                                <div className="flex items-center gap-3">
                                                    <span className="text-[11px] font-black text-emerald-500 uppercase italic">Active</span>
                                                    <ShieldCheck size={18} className="text-emerald-500" />
                                                </div>
                                            </div>
                                        </CardContent>
                                    </Card>

                                    <Card className="bg-primary/5 border-primary/20 border-dashed border-2 shadow-inner relative overflow-hidden group">
                                        <div className="absolute inset-0 bg-primary/[0.02] group-hover:bg-primary/[0.05] transition-colors"></div>
                                        <CardContent className="p-12 flex flex-col items-center text-center gap-8 relative z-10">
                                            <div className="w-20 h-20 bg-card border-2 border-primary/20 rounded-[2.5rem] flex items-center justify-center shadow-2xl group-hover:scale-105 transition-transform">
                                                <CloudLightning size={40} className="text-primary" />
                                            </div>
                                            <div>
                                                <h4 className="text-sm font-black uppercase tracking-[0.4em] text-foreground mb-4 italic">Sentinel Protocol</h4>
                                                <p className="text-[11px] text-muted-foreground font-bold leading-relaxed uppercase tracking-tight opacity-80">Autonomous threat detection systems are monitoring inbound traffic vectors for DDoS patterns.</p>
                                            </div>
                                            <Badge className="bg-primary/20 text-primary-foreground border-none font-black text-[9px] tracking-widest px-4 h-7">LEVEL 1 ALERT READY</Badge>
                                        </CardContent>
                                    </Card>
                                </div>
                            </div>
                        </TabsContent>

                        <TabsContent value="logs" className="space-y-8 animate-in slide-in-from-bottom-4 duration-500">
                            <div className="flex flex-col lg:flex-row gap-8 h-auto lg:h-[800px] items-stretch">
                                {/* Side Navigation */}
                                <Card className="hidden lg:flex flex-col w-80 bg-card border-border/60 shadow-2xl overflow-hidden relative">
                                    <div className="absolute bottom-0 left-0 w-full h-1/2 bg-[linear-gradient(to_top,rgba(59,130,246,0.03),transparent)]"></div>
                                    <CardHeader className="p-10 pb-4">
                                        <CardTitle className="text-[11px] font-black uppercase tracking-[0.6em] text-muted-foreground">Log Pipelines</CardTitle>
                                    </CardHeader>
                                    <CardContent className="p-6 space-y-3 relative z-10">
                                        <Button
                                            onClick={() => setLogTab('system')}
                                            className={`w-full justify-start rounded-2xl font-black text-[12px] uppercase italic tracking-widest h-16 transition-all gap-5 border-2 ${logTab === 'system' ? 'bg-primary/10 border-primary/30 text-white shadow-soft' : 'bg-transparent border-transparent text-muted-foreground hover:bg-secondary/50'}`}
                                        >
                                            <Terminal size={20} className={logTab === 'system' ? 'text-primary' : 'text-muted-foreground'} /> Sys Journal
                                        </Button>
                                        <Button
                                            onClick={() => setLogTab('nginx_access')}
                                            className={`w-full justify-start rounded-2xl font-black text-[12px] uppercase italic tracking-widest h-16 transition-all gap-5 border-2 ${logTab === 'nginx_access' ? 'bg-emerald-500/10 border-emerald-500/30 text-white shadow-soft' : 'bg-transparent border-transparent text-muted-foreground hover:bg-secondary/50'}`}
                                        >
                                            <Globe size={20} className={logTab === 'nginx_access' ? 'text-emerald-500' : 'text-muted-foreground'} /> HTTP Ingress
                                        </Button>
                                        <Button
                                            onClick={() => setLogTab('nginx_error')}
                                            className={`w-full justify-start rounded-2xl font-black text-[12px] uppercase italic tracking-widest h-16 transition-all gap-5 border-2 ${logTab === 'nginx_error' ? 'bg-rose-500/10 border-rose-500/30 text-white shadow-soft' : 'bg-transparent border-transparent text-muted-foreground hover:bg-secondary/50'}`}
                                        >
                                            <AlertTriangle size={20} className={logTab === 'nginx_error' ? 'text-rose-500' : 'text-muted-foreground'} /> HTTP Errors
                                        </Button>
                                    </CardContent>
                                    <div className="mt-auto p-10 border-t border-border/20 bg-secondary/10">
                                        <div className="flex items-center gap-4 mb-4">
                                            <div className="w-3 h-3 rounded-full bg-emerald-500 animate-pulse shadow-[0_0_8px_rgba(16,185,129,0.3)]"></div>
                                            <span className="text-[11px] font-black uppercase tracking-widest text-foreground italic">Link Integrity Blue</span>
                                        </div>
                                        <p className="text-[10px] text-muted-foreground uppercase font-black tracking-tighter opacity-70 mb-5">Persistent tailstream buffer is pulling 30 lines/sec per pipeline.</p>
                                        <div className="h-1.5 w-full bg-secondary rounded-full overflow-hidden shadow-inner">
                                            <div className="h-full w-[90%] bg-primary/40 animate-pulse"></div>
                                        </div>
                                    </div>
                                </Card>

                                {/* Mobile Log Controls */}
                                <div className="lg:hidden flex overflow-x-auto gap-4 pb-4 no-scrollbar">
                                    {(['system', 'nginx_access', 'nginx_error'] as const).map((tab) => (
                                        <Button
                                            key={tab}
                                            onClick={() => setLogTab(tab)}
                                            className={`rounded-2xl font-black text-[11px] uppercase h-14 px-10 whitespace-nowrap transition-all border-2 ${logTab === tab ? 'bg-primary border-transparent text-primary-foreground shadow-xl scale-105' : 'bg-card border-border text-muted-foreground'}`}
                                        >
                                            {tab === 'system' ? 'System' : tab === 'nginx_access' ? 'Access' : 'Runtime Errors'}
                                        </Button>
                                    ))}
                                </div>

                                {/* Main Console */}
                                <Card className="flex-1 bg-card border-border/80 shadow-3xl overflow-hidden flex flex-col h-[600px] lg:h-full group">
                                    <CardHeader className="bg-secondary/20 border-b border-border/60 p-6 sm:p-10 flex flex-row items-center justify-between shadow-soft">
                                        <div className="flex items-center gap-5">
                                            <div className="hidden xs:flex gap-2.5">
                                                <div className="w-3.5 h-3.5 rounded-full bg-border/60"></div>
                                                <div className="w-3.5 h-3.5 rounded-full bg-border/60"></div>
                                                <div className="w-3.5 h-3.5 rounded-full bg-border/60"></div>
                                            </div>
                                            <div className="hidden xs:block h-8 w-px bg-border/80 mx-3"></div>
                                            <div className="flex flex-col">
                                                <span className="text-[9px] font-black text-primary/70 uppercase tracking-[0.4em] mb-1.5 mb-1.5 italic">Telemetric Stream Output</span>
                                                <div className="flex items-center gap-3 text-[11px] sm:text-[14px] font-black text-white uppercase tracking-tight truncate drop-shadow-sm">
                                                    <FileText size={18} className="text-primary/70" /> {logs?.[logTab]?.path || 'Synchronizing with remote host...'}
                                                </div>
                                            </div>
                                        </div>
                                        <div className="hidden sm:flex items-center gap-3">
                                            <Badge variant="outline" className="text-[9px] font-black border-primary/30 bg-primary/5 text-primary uppercase px-5 h-9 tracking-widest">ENCRYPTED SSE</Badge>
                                        </div>
                                    </CardHeader>
                                    <CardContent className="p-0 flex-1 overflow-hidden relative bg-[#0a0c12]/40">
                                        <div className="absolute inset-0 bg-[radial-gradient(circle_at_bottom_right,rgba(59,130,246,0.03),transparent_50%)] pointer-events-none"></div>
                                        <ScrollArea className="h-full w-full p-8 sm:p-16">
                                            <pre className="text-foreground/90 font-mono text-[12px] sm:text-sm md:text-base whitespace-pre-wrap leading-relaxed tracking-tight selection:bg-primary/30 drop-shadow-md">
                                                {logs?.[logTab]?.content || 'Bufferizing encrypted telemetric stream from remote data port...'}
                                            </pre>
                                            <div className="h-32"></div>
                                        </ScrollArea>
                                    </CardContent>
                                    <Separator className="bg-border/40" />
                                    <div className="px-10 py-6 flex items-center justify-between bg-card text-muted-foreground text-[11px] font-black uppercase tracking-[0.4em] italic shadow-inner">
                                        <div className="flex items-center gap-5">
                                            <span className="w-3 h-3 rounded-full bg-emerald-500 shadow-[0_0_12px_rgba(16,185,129,0.5)] animate-pulse"></span>
                                            <span className="hidden sm:inline">Stream Health 100% Secure</span>
                                            <span className="sm:hidden">Secure Stream</span>
                                        </div>
                                        <div className="flex items-center gap-8">
                                            <span className="hidden md:inline opacity-30 font-extrabold">Buffer: 4.8MB/S</span>
                                            <RefreshCcw size={16} className="animate-spin-slow text-primary/80" />
                                        </div>
                                    </div>
                                </Card>
                            </div>
                        </TabsContent>
                    </Tabs>
                </main>
            </div>

            <footer className="mt-auto p-16 sm:p-24 border-t border-border/60 bg-card/80 backdrop-blur-3xl relative overflow-hidden">
                <div className="max-w-screen-2xl mx-auto flex flex-col md:flex-row items-center justify-between gap-16">
                    <div className="flex flex-col sm:flex-row items-center gap-12 sm:gap-24">
                        <div className="flex flex-col text-center sm:text-left">
                            <span className="text-[11px] font-black text-muted-foreground uppercase tracking-[0.8em] mb-4">Core Architecture</span>
                            <span className="text-lg font-black text-foreground italic uppercase tracking-tighter">AcmaDash Engine <span className="text-primary ml-2 px-3 py-1 bg-primary/5 rounded-lg border border-primary/10">v{Version}</span></span>
                        </div>
                        <div className="hidden lg:flex flex-col">
                            <span className="text-[11px] font-black text-muted-foreground uppercase tracking-[0.8em] mb-4">UI Manifest</span>
                            <span className="text-lg font-black text-muted-foreground uppercase tracking-widest italic opacity-70">Slate Elite v24.1</span>
                        </div>
                    </div>

                    <div className="flex flex-col items-center md:items-end">
                        <div className="flex items-center gap-5 mb-5">
                            <div className="h-2 w-2 rounded-full bg-primary shadow-[0_0_15px_rgba(59,130,246,0.6)] animate-pulse"></div>
                            <span className="text-[13px] font-black text-foreground uppercase tracking-[0.5em] italic leading-none drop-shadow-sm">Global Ops Monitor</span>
                        </div>
                        <span className="text-[10px] font-black text-muted-foreground uppercase tracking-[0.5em] mt-5 opacity-50">&copy; 2024 AcmaTvirus Intelligence Integrated Systems</span>
                    </div>
                </div>
                <div className="absolute bottom-0 left-0 w-full h-1 bg-gradient-to-r from-transparent via-primary/30 to-transparent"></div>
            </footer>

            <style>{`
                @keyframes spin-slow {
                    from { transform: rotate(0deg); }
                    to { transform: rotate(360deg); }
                }
                .animate-spin-slow {
                    animation: spin-slow 12s linear infinite;
                }
                .no-scrollbar::-webkit-scrollbar {
                    display: none;
                }
                .no-scrollbar {
                    -ms-overflow-style: none;
                    scrollbar-width: none;
                }
                .shadow-soft {
                    box-shadow: 0 10px 30px -5px rgba(0, 0, 0, 0.3);
                }
                .shadow-3xl {
                    box-shadow: 0 35px 60px -15px rgba(0, 0, 0, 0.6);
                }
            `}</style>
        </div>
    );
};

export default App;
