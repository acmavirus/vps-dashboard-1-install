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
    X
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

const Version = "1.1.3-Elite";

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
        <div className="min-h-screen bg-[#0a0c10] text-[#f0f2f5] flex flex-col font-sans selection:bg-blue-500/30">
            {/* Background Texture & Glow */}
            <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10 bg-[radial-gradient(#1c1c1c_1px,transparent_1px)] [background-size:40px_40px] [opacity:0.2]"></div>
            <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10">
                <div className="absolute top-[-10%] left-[-10%] w-[50%] h-[50%] bg-blue-600/10 blur-[150px] rounded-full animate-pulse"></div>
                <div className="absolute bottom-[-10%] right-[-10%] w-[50%] h-[50%] bg-indigo-600/10 blur-[150px] rounded-full animate-pulse"></div>
            </div>

            <div className="flex flex-1 overflow-hidden relative">
                {/* Desktop Sidebar - Darker than background */}
                <aside className="hidden lg:flex flex-col w-20 bg-[#050608] border-r border-white/5 items-center py-8 gap-10">
                    <div className="w-12 h-12 bg-blue-600 rounded-2xl flex items-center justify-center shadow-lg shadow-blue-500/20">
                        <Server size={24} className="text-white" />
                    </div>
                    <nav className="flex flex-col gap-6">
                        <Button variant="ghost" size="icon" className="text-blue-500 bg-blue-500/10 rounded-xl transition-all hover:scale-110">
                            <LayoutDashboard size={24} />
                        </Button>
                        <Button variant="ghost" size="icon" className="text-slate-600 hover:text-white rounded-xl transition-all hover:scale-110">
                            <Activity size={24} />
                        </Button>
                        <Button variant="ghost" size="icon" className="text-slate-600 hover:text-white rounded-xl transition-all hover:scale-110">
                            <Terminal size={24} />
                        </Button>
                    </nav>
                </aside>

                {/* Mobile Menu Backdrop */}
                {isMobileMenuOpen && (
                    <div className="lg:hidden fixed inset-0 bg-black/80 backdrop-blur-md z-40 transition-all duration-300" onClick={() => setIsMobileMenuOpen(false)}></div>
                )}

                {/* Mobile Sidebar - Darker than background */}
                <aside className={`lg:hidden fixed left-0 top-0 h-full w-72 bg-[#050608] z-50 transform transition-transform duration-500 border-r border-white/5 ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}`}>
                    <div className="p-8 flex flex-col h-full">
                        <div className="flex items-center justify-between mb-12">
                            <div className="flex items-center gap-3">
                                <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
                                    <Server className="text-white" size={18} />
                                </div>
                                <span className="font-black italic uppercase text-2xl tracking-tighter">Acma<span className="text-blue-500 not-italic">Dash</span></span>
                            </div>
                            <Button variant="ghost" size="icon" className="text-slate-400" onClick={() => setIsMobileMenuOpen(false)}>
                                <X size={24} />
                            </Button>
                        </div>
                        <nav className="flex flex-col gap-3 flex-1">
                            <Button variant="ghost" className="justify-start gap-4 text-[11px] font-black uppercase text-blue-400 bg-blue-500/10 border border-blue-500/20 rounded-2xl h-14">
                                <LayoutDashboard size={20} /> Overview Hub
                            </Button>
                            <Button variant="ghost" className="justify-start gap-4 text-[11px] font-black uppercase text-slate-500 hover:text-white h-14 rounded-2xl hover:bg-white/5">
                                <Activity size={20} /> Performance Data
                            </Button>
                            <Button variant="ghost" className="justify-start gap-4 text-[11px] font-black uppercase text-slate-500 hover:text-white h-14 rounded-2xl hover:bg-white/5">
                                <Terminal size={20} /> Remote Terminal
                            </Button>
                        </nav>
                        <div className="mt-auto flex flex-col gap-4">
                            <Separator className="bg-white/5" />
                            <div className="bg-blue-500/5 border border-blue-500/10 rounded-2xl p-4">
                                <p className="text-[10px] font-black text-blue-400/80 uppercase tracking-widest mb-1">Encrypted Line</p>
                                <p className="text-[9px] text-slate-500 font-bold uppercase leading-relaxed">Secure SSE Tunnel established to VPS core.</p>
                            </div>
                        </div>
                    </div>
                </aside>

                <main className="flex-1 overflow-y-auto p-4 sm:p-6 md:p-10 space-y-10">
                    <header className="flex flex-col md:flex-row items-center justify-between gap-6">
                        <div className="flex items-center justify-between w-full md:w-auto">
                            <div className="flex items-center gap-4 sm:gap-6">
                                <Button variant="ghost" size="icon" className="lg:hidden bg-black/40 border border-white/5 rounded-2xl w-12 h-12" onClick={() => setIsMobileMenuOpen(true)}>
                                    <Menu size={24} />
                                </Button>
                                <div className="hidden sm:flex w-14 h-14 bg-blue-600 rounded-2xl items-center justify-center shadow-2xl shadow-blue-500/20">
                                    <Server size={28} className="text-white" />
                                </div>
                                <div>
                                    <h1 className="text-2xl sm:text-3xl font-black tracking-tight flex items-center gap-3 italic uppercase text-white">
                                        Acma<span className="text-blue-500 not-italic">Dash</span>
                                        <Badge variant="outline" className="hidden xs:inline-flex border-blue-500/40 text-blue-400 bg-blue-500/10 h-6 px-2 text-[10px] font-black">ELITE</Badge>
                                    </h1>
                                    <p className="text-[10px] sm:text-xs text-slate-500 font-bold uppercase tracking-widest">Autonomous VPS Orchestrator</p>
                                </div>
                            </div>

                            <div className="md:hidden flex items-center gap-2 bg-[#050608] px-3 py-1.5 rounded-full border border-white/5">
                                <div className={`w-2 h-2 rounded-full ${connected ? "bg-emerald-500 shadow-[0_0_8px_#10b981]" : "bg-amber-500"}`}></div>
                                <span className="text-[10px] font-black uppercase text-slate-400">Live</span>
                            </div>
                        </div>

                        <div className="flex items-center gap-3 w-full md:w-auto">
                            <div className="hidden md:flex items-center gap-3 bg-[#050608] px-5 py-2.5 rounded-2xl border border-white/5 shadow-inner">
                                <div className={`w-2 h-2 rounded-full ${connected ? "bg-emerald-500 shadow-[0_0_8px_#10b981]" : "bg-amber-500"}`}></div>
                                <span className="text-[11px] font-black uppercase tracking-[0.1em] text-slate-300">
                                    {connected ? "Stream Established" : "Connecting..."}
                                </span>
                            </div>
                            <Button size="sm" variant="outline" className="rounded-2xl border-white/5 bg-[#050608] hover:bg-[#111] flex-1 md:flex-initial text-[10px] sm:text-xs h-11 px-6 font-black uppercase tracking-widest italic">
                                <RefreshCcw size={16} className={`mr-2 ${connected ? "animate-spin-slow text-blue-500" : ""}`} /> Up: {stats ? formatUptime(stats.uptime) : 'LODING'}
                            </Button>
                        </div>
                    </header>

                    <Tabs defaultValue="overview" className="space-y-8 animate-in fade-in duration-700 slide-in-from-bottom-4">
                        <div className="flex flex-col sm:flex-row items-center justify-between gap-6">
                            <TabsList className="bg-[#050608] border border-white/5 p-1.5 rounded-2xl w-full sm:w-auto shadow-2xl">
                                <TabsTrigger value="overview" className="flex-1 sm:flex-initial data-[state=active]:bg-blue-600 data-[state=active]:text-white rounded-xl px-8 font-black text-[11px] tracking-widest uppercase h-10 transition-all">Overview</TabsTrigger>
                                <TabsTrigger value="logs" className="flex-1 sm:flex-initial data-[state=active]:bg-blue-600 data-[state=active]:text-white rounded-xl px-8 font-black text-[11px] tracking-widest uppercase h-10 transition-all">Deep Logs</TabsTrigger>
                            </TabsList>
                            <div className="flex items-center gap-3 bg-white/5 px-4 py-2 rounded-xl text-slate-400 text-xs font-black uppercase italic border border-white/5">
                                <Monitor size={16} className="text-blue-500" /> {stats?.hostname || 'Resolving...'}
                            </div>
                        </div>

                        <TabsContent value="overview" className="space-y-8">
                            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6">
                                {/* Dashboard Cards - Darker than background for recession depth */}
                                <Card className="bg-[#05080c] border-white/5 shadow-[0_10px_30px_-15px_rgba(0,0,0,0.5)] overflow-hidden relative group hover:border-blue-500/20 transition-all duration-500 transform hover:-translate-y-1">
                                    <div className="absolute top-0 right-0 w-24 h-24 bg-blue-600/5 blur-3xl -z-10 group-hover:bg-blue-600/10 transition-colors"></div>
                                    <CardHeader className="pb-3 p-6 pt-5">
                                        <div className="flex justify-between items-center text-[10px] font-black text-slate-500 tracking-[0.3em] uppercase">
                                            <span>CORE LOAD</span>
                                            <Cpu size={18} className="text-blue-500 transform group-hover:rotate-12 transition-transform" />
                                        </div>
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0">
                                        <div className="text-4xl font-black italic mb-3 tracking-tighter text-white drop-shadow-lg">{stats?.cpu.toFixed(1) || '0.0'}<span className="text-lg text-slate-600 ml-1">%</span></div>
                                        <Progress value={stats?.cpu || 0} className="h-2 bg-white/5 overflow-hidden rounded-full"
                                            indicatorClassName={stats && stats.cpu > 80 ? "bg-red-500 shadow-[0_0_10px_#ef4444]" : "bg-blue-500 shadow-[0_0_10px_#3b82f6]"}
                                        />
                                    </CardContent>
                                    <CardFooter className="p-6 py-4 bg-white/[0.02] text-[9px] text-slate-600 font-bold uppercase tracking-[0.2em]">
                                        Processing Utilization
                                    </CardFooter>
                                </Card>

                                <Card className="bg-[#05080c] border-white/5 shadow-[0_10px_30px_-15px_rgba(0,0,0,0.5)] overflow-hidden relative group hover:border-emerald-500/20 transition-all duration-500 transform hover:-translate-y-1">
                                    <div className="absolute top-0 right-0 w-24 h-24 bg-emerald-600/5 blur-3xl -z-10 group-hover:bg-emerald-600/10 transition-colors"></div>
                                    <CardHeader className="pb-3 p-6 pt-5">
                                        <div className="flex justify-between items-center text-[10px] font-black text-slate-500 tracking-[0.3em] uppercase">
                                            <span>MEM ALLOC</span>
                                            <Database size={18} className="text-emerald-500 transform group-hover:scale-110 transition-transform" />
                                        </div>
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0">
                                        <div className="text-4xl font-black italic mb-3 tracking-tighter text-white drop-shadow-lg">{stats?.ram.toFixed(1) || '0.0'}<span className="text-lg text-slate-600 ml-1">%</span></div>
                                        <Progress value={stats?.ram || 0} className="h-2 bg-white/5 overflow-hidden rounded-full"
                                            indicatorClassName="bg-emerald-500 shadow-[0_0_10px_#10b981]"
                                        />
                                    </CardContent>
                                    <CardFooter className="p-6 py-4 bg-white/[0.02] text-[9px] text-slate-600 font-bold uppercase tracking-[0.2em] flex justify-between w-full">
                                        <span>{stats ? formatBytes(stats.ram_used) : '0.00 GB'}</span>
                                        <span className="opacity-40">OF {stats ? formatBytes(stats.ram_total).split(' ')[0] : '0.00'}</span>
                                    </CardFooter>
                                </Card>

                                <Card className="bg-[#05080c] border-white/5 shadow-[0_10px_30px_-15px_rgba(0,0,0,0.5)] overflow-hidden relative group hover:border-amber-500/20 transition-all duration-500 transform hover:-translate-y-1">
                                    <div className="absolute top-0 right-0 w-24 h-24 bg-amber-600/5 blur-3xl -z-10 group-hover:bg-amber-600/10 transition-colors"></div>
                                    <CardHeader className="pb-3 p-6 pt-5">
                                        <div className="flex justify-between items-center text-[10px] font-black text-slate-500 tracking-[0.3em] uppercase">
                                            <span>STORAGE</span>
                                            <HardDrive size={18} className="text-amber-500 transform group-hover:-translate-y-1 transition-transform" />
                                        </div>
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0">
                                        <div className="text-4xl font-black italic mb-3 tracking-tighter text-white drop-shadow-lg">{stats?.disk.toFixed(1) || '0.0'}<span className="text-lg text-slate-600 ml-1">%</span></div>
                                        <Progress value={stats?.disk || 0} className="h-2 bg-white/5 overflow-hidden rounded-full"
                                            indicatorClassName="bg-amber-500 shadow-[0_0_10px_#f59e0b]"
                                        />
                                    </CardContent>
                                    <CardFooter className="p-6 py-4 bg-white/[0.02] text-[9px] text-slate-600 font-bold uppercase tracking-[0.2em] flex justify-between w-full">
                                        <span>REMAIN: {stats ? formatBytes(stats.disk_total - stats.disk_used) : '---'}</span>
                                    </CardFooter>
                                </Card>

                                <Card className="bg-[#05080c] border-white/5 shadow-[0_10px_30px_-15px_rgba(0,0,0,0.5)] overflow-hidden relative group hover:border-indigo-500/20 transition-all duration-500 transform hover:-translate-y-1">
                                    <div className="absolute top-0 right-0 w-24 h-24 bg-indigo-600/5 blur-3xl -z-10 group-hover:bg-indigo-600/10 transition-colors"></div>
                                    <CardHeader className="pb-3 p-6 pt-5">
                                        <div className="flex justify-between items-center text-[10px] font-black text-slate-500 tracking-[0.3em] uppercase">
                                            <span>TRAFFIC</span>
                                            <CloudLightning size={18} className="text-indigo-500 animate-pulse" />
                                        </div>
                                    </CardHeader>
                                    <CardContent className="p-6 pt-0">
                                        <div className="text-3xl sm:text-4xl font-black italic mb-1 tracking-tighter text-white overflow-hidden text-ellipsis whitespace-nowrap">
                                            {stats ? formatBytes(stats.net_recv) : '0.00 GB'}
                                        </div>
                                        <div className="text-[10px] text-slate-500 font-black uppercase tracking-widest mt-1 opacity-60">Inbound Cumulative</div>
                                    </CardContent>
                                    <CardFooter className="p-6 py-4 bg-white/[0.02] text-[9px] text-slate-600 font-bold uppercase tracking-[0.2em] flex justify-between w-full">
                                        <span>OUT: {stats ? formatBytes(stats.net_sent).split(' ')[0] : '0.00'}</span>
                                        <span className="flex items-center gap-2 text-indigo-400"><Hash size={12} /> {stats?.connections || 0} CONNS</span>
                                    </CardFooter>
                                </Card>
                            </div>

                            <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
                                <Card className="lg:col-span-2 bg-[#05080c] border-white/5 shadow-[0_20px_50px_-20px_rgba(0,0,0,0.5)] relative overflow-hidden group">
                                    <CardHeader className="flex flex-row items-center justify-between p-6 sm:p-8 pb-0">
                                        <div>
                                            <CardTitle className="text-sm sm:text-lg font-black italic tracking-[0.2em] uppercase text-white">Live Telemetry</CardTitle>
                                            <CardDescription className="text-[10px] sm:text-xs uppercase font-black text-slate-600 tracking-wider mt-1">Processor oscillation delta • 1.0s resolution</CardDescription>
                                        </div>
                                        <Badge variant="secondary" className="bg-blue-600/20 text-blue-400 border border-blue-500/30 text-[9px] sm:text-[10px] font-black px-3 h-7 tracking-widest">LIVE STREAM</Badge>
                                    </CardHeader>
                                    <CardContent className="h-[300px] sm:h-[450px] p-0 pt-10 overflow-hidden">
                                        <ResponsiveContainer width="100%" height="100%">
                                            <AreaChart data={history} margin={{ left: -20, right: 20, top: 0, bottom: 0 }}>
                                                <defs>
                                                    <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                                                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.4} />
                                                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                                                    </linearGradient>
                                                </defs>
                                                <CartesianGrid strokeDasharray="5 5" vertical={false} stroke="#ffffff05" />
                                                <XAxis dataKey="time" stroke="#475569" fontSize={10} tickLine={false} axisLine={false} minTickGap={40} tick={{ fontWeight: '900' }} />
                                                <YAxis stroke="#475569" fontSize={10} tickLine={false} axisLine={false} domain={[0, 100]} tick={{ fontWeight: '900' }} />
                                                <Tooltip
                                                    contentStyle={{ backgroundColor: '#000000', border: '1px solid #ffffff10', borderRadius: '16px', backdropFilter: 'blur(20px)', boxShadow: '0 20px 40px -10px rgba(0,0,0,0.8)' }}
                                                    itemStyle={{ color: '#3b82f6', fontWeight: '900', fontSize: '14px' }}
                                                    labelStyle={{ color: '#64748b', fontSize: '10px', fontWeight: '900', marginBottom: '4px', textTransform: 'uppercase', letterSpacing: '0.1em' }}
                                                />
                                                <Area type="monotone" dataKey="cpu" stroke="#3b82f6" strokeWidth={4} fillOpacity={1} fill="url(#colorCpu)" animationDuration={600} isAnimationActive={history.length < 2} strokeLinecap="round" />
                                            </AreaChart>
                                        </ResponsiveContainer>
                                    </CardContent>
                                    <div className="absolute inset-0 pointer-events-none border border-white/5 rounded-xl"></div>
                                </Card>

                                <div className="space-y-6">
                                    <Card className="bg-[#05080c] border-white/5 shadow-2xl relative group">
                                        <CardHeader className="pb-4 p-8">
                                            <CardTitle className="text-xs font-black italic tracking-[0.3em] uppercase text-slate-500">Node Specs</CardTitle>
                                        </CardHeader>
                                        <CardContent className="space-y-6 p-8 pt-0">
                                            <div className="flex items-center justify-between">
                                                <div className="flex items-center gap-4">
                                                    <div className="p-2.5 bg-blue-500/10 rounded-2xl border border-blue-500/20"><Globe size={18} className="text-blue-500" /></div>
                                                    <div>
                                                        <p className="text-[9px] font-black text-slate-600 uppercase tracking-widest">Platform</p>
                                                        <p className="text-[11px] font-black text-white uppercase italic">{stats?.platform || 'Detecting...'}</p>
                                                    </div>
                                                </div>
                                            </div>
                                            <Separator className="bg-white/5" />
                                            <div className="flex items-center justify-between">
                                                <div className="flex items-center gap-4">
                                                    <div className="p-2.5 bg-amber-500/10 rounded-2xl border border-amber-500/20"><Hash size={18} className="text-amber-500" /></div>
                                                    <div>
                                                        <p className="text-[9px] font-black text-slate-600 uppercase tracking-widest">Build</p>
                                                        <p className="text-[11px] font-black text-white truncate max-w-[140px]">{stats?.kernel || '---'}</p>
                                                    </div>
                                                </div>
                                            </div>
                                            <Separator className="bg-white/5" />
                                            <div className="flex items-center justify-between">
                                                <div className="flex items-center gap-4">
                                                    <div className="p-2.5 bg-emerald-500/10 rounded-2xl border border-emerald-500/20"><ShieldCheck size={18} className="text-emerald-500" /></div>
                                                    <div>
                                                        <p className="text-[9px] font-black text-slate-600 uppercase tracking-widest">Protection</p>
                                                        <Badge className="bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 text-[9px] font-black uppercase px-3 h-6 mt-1 tracking-widest">GOD MODE</Badge>
                                                    </div>
                                                </div>
                                            </div>
                                        </CardContent>
                                    </Card>

                                    <div className="bg-gradient-to-br from-blue-600/20 to-indigo-600/5 border border-white/10 p-1 rounded-3xl group cursor-help transition-all duration-500 hover:border-blue-500/40">
                                        <div className="bg-[#05080c] p-10 rounded-[22px] flex flex-col items-center text-center gap-6 shadow-3xl">
                                            <div className="p-6 bg-blue-500/10 rounded-full shadow-[0_0_30px_rgba(59,130,246,0.1)] group-hover:scale-110 transition-transform duration-500">
                                                <ShieldCheck size={48} className="text-blue-500 drop-shadow-[0_0_10px_#3b82f6]" />
                                            </div>
                                            <div className="space-y-3">
                                                <h4 className="text-sm font-black uppercase tracking-[0.3em] italic text-white underline decoration-blue-500 decoration-4 underline-offset-8">Sentinel Active</h4>
                                                <p className="text-[10px] text-slate-500 font-bold leading-relaxed uppercase tracking-tighter px-2">Quantum DDoS filtration and telegram-alert protocols are running at zero-latency in the background thread.</p>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </TabsContent>

                        <TabsContent value="logs" className="space-y-8 animate-in fade-in duration-500">
                            <div className="flex flex-col lg:flex-row gap-8 h-auto lg:h-[750px]">
                                {/* Mobile Log Navigation (Horizontal Scroll) - Darker navigation */}
                                <div className="lg:hidden flex overflow-x-auto gap-3 pb-4 no-scrollbar">
                                    <Button
                                        onClick={() => setLogTab('system')}
                                        variant="default"
                                        className={`rounded-2xl font-black text-[10px] uppercase italic tracking-widest h-14 px-8 whitespace-nowrap transition-all flex-1 ${logTab === 'system' ? 'bg-blue-600 shadow-xl shadow-blue-600/20 text-white' : 'bg-[#050608] text-slate-500 border border-white/5'}`}
                                    >
                                        <Terminal size={18} className="mr-3" /> Sys/Node
                                    </Button>
                                    <Button
                                        onClick={() => setLogTab('nginx_access')}
                                        variant="default"
                                        className={`rounded-2xl font-black text-[10px] uppercase italic tracking-widest h-14 px-8 whitespace-nowrap transition-all flex-1 ${logTab === 'nginx_access' ? 'bg-emerald-600 shadow-xl shadow-emerald-600/20 text-white' : 'bg-[#050608] text-slate-500 border border-white/5'}`}
                                    >
                                        <Globe size={18} className="mr-3" /> Nginx Inbound
                                    </Button>
                                    <Button
                                        onClick={() => setLogTab('nginx_error')}
                                        variant="default"
                                        className={`rounded-2xl font-black text-[10px] uppercase italic tracking-widest h-14 px-8 whitespace-nowrap transition-all flex-1 ${logTab === 'nginx_error' ? 'bg-red-600 shadow-xl shadow-red-600/20 text-white' : 'bg-[#050608] text-slate-500 border border-white/5'}`}
                                    >
                                        <AlertTriangle size={18} className="mr-3" /> Nginx Crits
                                    </Button>
                                </div>

                                {/* Desktop Log Navigation - Pitch Black Recession */}
                                <Card className="hidden lg:flex flex-col w-72 bg-[#050608] border-white/5 shadow-3xl h-fit sticky top-10">
                                    <CardHeader className="p-8 pb-4">
                                        <CardTitle className="text-[10px] font-black uppercase tracking-[0.4em] text-slate-600 italic">Data Pipeline</CardTitle>
                                    </CardHeader>
                                    <CardContent className="p-4 space-y-2">
                                        <Button
                                            onClick={() => setLogTab('system')}
                                            variant="ghost"
                                            className={`w-full justify-start rounded-2xl font-black text-xs uppercase italic tracking-widest h-14 transition-all ${logTab === 'system' ? 'bg-blue-600 text-white shadow-xl shadow-blue-600/10' : 'text-slate-500 hover:text-white hover:bg-white/5'}`}
                                        >
                                            <Terminal size={20} className="mr-4" /> System Core
                                        </Button>
                                        <Button
                                            onClick={() => setLogTab('nginx_access')}
                                            variant="ghost"
                                            className={`w-full justify-start rounded-2xl font-black text-xs uppercase italic tracking-widest h-14 transition-all ${logTab === 'nginx_access' ? 'bg-emerald-600 text-white shadow-xl shadow-emerald-600/10' : 'text-slate-500 hover:text-white hover:bg-white/5'}`}
                                        >
                                            <Globe size={20} className="mr-4" /> Nginx Access
                                        </Button>
                                        <Button
                                            onClick={() => setLogTab('nginx_error')}
                                            variant="ghost"
                                            className={`w-full justify-start rounded-2xl font-black text-xs uppercase italic tracking-widest h-14 transition-all ${logTab === 'nginx_error' ? 'bg-red-600 text-white shadow-xl shadow-red-600/10' : 'text-slate-500 hover:text-white hover:bg-white/5'}`}
                                        >
                                            <AlertTriangle size={20} className="mr-4" /> Nginx Errors
                                        </Button>
                                    </CardContent>
                                    <div className="p-8 pt-0">
                                        <div className="bg-white/5 p-4 rounded-2xl border border-white/5">
                                            <div className="flex items-center gap-2 mb-2">
                                                <Activity size={12} className="text-blue-500" />
                                                <span className="text-[9px] font-black uppercase text-slate-400">Stream Status</span>
                                            </div>
                                            <div className="h-1 w-full bg-white/5 rounded-full overflow-hidden">
                                                <div className="h-full w-2/3 bg-blue-500 animate-pulse"></div>
                                            </div>
                                        </div>
                                    </div>
                                </Card>

                                {/* Terminal Content - Absolute Black */}
                                <Card className="flex-1 bg-[#020305] border-white/5 shadow-[0_30px_100px_-20px_rgba(0,0,0,0.8)] overflow-hidden flex flex-col h-[550px] lg:h-full relative group">
                                    <div className="absolute top-0 inset-x-0 h-px bg-gradient-to-r from-transparent via-blue-500/50 to-transparent opacity-50"></div>
                                    <CardHeader className="bg-[#050608]/80 border-b border-white/5 p-6 sm:p-8 flex flex-row items-center justify-between backdrop-blur-3xl">
                                        <div className="flex items-center gap-4 sm:gap-6">
                                            <div className="hidden xs:flex gap-2.5">
                                                <div className="w-3 h-3 rounded-full bg-[#ff5f56]"></div>
                                                <div className="w-3 h-3 rounded-full bg-[#ffbd2e]"></div>
                                                <div className="w-3 h-3 rounded-full bg-[#27c93f]"></div>
                                            </div>
                                            <div className="hidden xs:block h-6 w-px bg-white/5 mx-2"></div>
                                            <div className="flex flex-col">
                                                <span className="text-[9px] font-black text-slate-600 uppercase tracking-[0.4em] mb-1">Active File Path</span>
                                                <div className="flex items-center gap-2 text-[10px] sm:text-[12px] font-black text-blue-400 uppercase tracking-tighter truncate max-w-[180px] sm:max-w-none">
                                                    <FileText size={16} className="text-slate-600" /> {logs?.[logTab]?.path || 'Locating stream...'}
                                                </div>
                                            </div>
                                        </div>
                                        <Badge variant="outline" className="text-[8px] sm:text-[10px] font-black border-blue-500/40 text-blue-400 bg-blue-500/10 uppercase px-4 h-8 tracking-widest hidden sm:flex">ENCRYPTED STREAM</Badge>
                                    </CardHeader>
                                    <CardContent className="p-0 flex-1 overflow-hidden relative">
                                        <div className="absolute inset-0 bg-blue-600/[0.01] pointer-events-none"></div>
                                        <ScrollArea className="h-full w-full p-6 sm:p-10">
                                            <pre className="text-slate-400 font-mono text-[11px] sm:text-sm md:text-md whitespace-pre-wrap leading-relaxed tracking-tight selection:bg-blue-600/30">
                                                {logs?.[logTab]?.content || 'Initializing buffer system...'}
                                            </pre>
                                            <div className="h-20"></div> {/* Gap for end of scroll */}
                                        </ScrollArea>
                                    </CardContent>
                                    <Separator className="bg-white/5" />
                                    <div className="px-8 py-5 flex items-center justify-between bg-[#050608]/90 text-slate-500 text-[10px] font-black uppercase tracking-[0.3em] italic backdrop-blur-3xl">
                                        <div className="flex items-center gap-4">
                                            <span className="text-emerald-500 animate-pulse">●</span>
                                            <span className="hidden xs:inline">Bilateral Sync Active</span>
                                            <span className="xs:hidden">Live Link</span>
                                        </div>
                                        <div className="flex items-center gap-6">
                                            <span className="hidden sm:inline opacity-30">Auto-tail: enabled</span>
                                            <RefreshCcw size={14} className="animate-spin-slow text-blue-500" />
                                        </div>
                                    </div>
                                </Card>
                            </div>
                        </TabsContent>
                    </Tabs>
                </main>
            </div>

            <footer className="mt-auto p-8 sm:p-12 border-t border-white/5 bg-[#050608] relative overflow-hidden">
                <div className="absolute top-0 right-1/4 w-[400px] h-[400px] bg-blue-600/[0.02] blur-[150px] -z-10 rounded-full"></div>
                <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-12">
                    <div className="flex items-center gap-12 sm:gap-16">
                        <div className="flex flex-col">
                            <span className="text-[10px] font-black text-slate-600 uppercase tracking-[0.5em] mb-2">Original Architect</span>
                            <span className="text-sm font-black text-white italic uppercase tracking-[0.1em]">AcmaDash <span className="text-blue-500">Engine</span> Core v{Version}</span>
                        </div>
                        <div className="hidden xs:block h-10 w-px bg-white/5"></div>
                        <div className="flex flex-col">
                            <span className="text-[10px] font-black text-slate-600 uppercase tracking-[0.5em] mb-2">Framework Elite</span>
                            <span className="text-sm font-black text-slate-400 italic uppercase">Premium React 2024 Stack</span>
                        </div>
                    </div>

                    <div className="flex flex-col items-center md:items-end">
                        <div className="flex items-center gap-3 mb-2">
                            <div className="h-1.5 w-1.5 rounded-full bg-blue-500 shadow-[0_0_15px_#3b82f6] animate-pulse"></div>
                            <span className="text-[11px] font-black text-slate-300 uppercase tracking-[0.3em]">Sentinel Guard <span className="text-blue-500 ml-1">Live</span></span>
                        </div>
                        <span className="text-[10px] font-black text-slate-700 uppercase tracking-[0.4em] mt-2">&copy; 2024 All Rights Reserved. AcmaTvirus Ecosystem</span>
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
                @media (max-width: 400px) {
                    .xs\\:hidden { display: none; }
                    .xs\\:inline-flex { display: none; }
                }
            `}</style>
        </div>
    );
};

export default App;
