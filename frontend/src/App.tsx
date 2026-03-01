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

const Version = "1.1.2-Premium";

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
        <div className="min-h-screen bg-slate-950 text-slate-50 flex flex-col font-sans selection:bg-blue-500/30">
            {/* Background Glow */}
            <div className="fixed inset-0 overflow-hidden pointer-events-none -z-10">
                <div className="absolute top-[-10%] left-[-10%] w-[50%] h-[50%] bg-blue-600/5 blur-[120px] rounded-full animate-pulse"></div>
                <div className="absolute bottom-[-10%] right-[-10%] w-[50%] h-[50%] bg-indigo-600/5 blur-[120px] rounded-full animate-pulse"></div>
            </div>

            <div className="flex flex-1 overflow-hidden relative">
                {/* Desktop Sidebar */}
                <aside className="hidden lg:flex flex-col w-20 bg-slate-900/50 border-r border-slate-800/60 items-center py-8 gap-10">
                    <div className="w-12 h-12 bg-blue-600 rounded-2xl flex items-center justify-center shadow-lg shadow-blue-500/20">
                        <Server size={24} className="text-white" />
                    </div>
                    <nav className="flex flex-col gap-6">
                        <Button variant="ghost" size="icon" className="text-blue-500 bg-blue-500/10 rounded-xl transition-all hover:scale-110">
                            <LayoutDashboard size={24} />
                        </Button>
                        <Button variant="ghost" size="icon" className="text-slate-500 hover:text-white rounded-xl transition-all hover:scale-110">
                            <Activity size={24} />
                        </Button>
                        <Button variant="ghost" size="icon" className="text-slate-500 hover:text-white rounded-xl transition-all hover:scale-110">
                            <Terminal size={24} />
                        </Button>
                    </nav>
                </aside>

                {/* Mobile Menu Backdrop */}
                {isMobileMenuOpen && (
                    <div className="lg:hidden fixed inset-0 bg-black/60 backdrop-blur-sm z-40 transition-all duration-300" onClick={() => setIsMobileMenuOpen(false)}></div>
                )}

                {/* Mobile Sidebar */}
                <aside className={`lg:hidden fixed left-0 top-0 h-full w-64 bg-slate-900 z-50 transform transition-transform duration-300 border-r border-slate-800 ${isMobileMenuOpen ? 'translate-x-0' : '-translate-x-full'}`}>
                    <div className="p-6 flex flex-col h-full">
                        <div className="flex items-center justify-between mb-10">
                            <div className="flex items-center gap-2">
                                <Server className="text-blue-500" size={24} />
                                <span className="font-black italic uppercase text-xl">Acma<span className="text-blue-500">Dash</span></span>
                            </div>
                            <Button variant="ghost" size="icon" onClick={() => setIsMobileMenuOpen(false)}>
                                <X size={24} />
                            </Button>
                        </div>
                        <nav className="flex flex-col gap-2 flex-1">
                            <Button variant="ghost" className="justify-start gap-4 text-xs font-black uppercase text-blue-400 bg-blue-400/5 border border-blue-400/20 rounded-xl h-12">
                                <LayoutDashboard size={18} /> Overview
                            </Button>
                            <Button variant="ghost" className="justify-start gap-4 text-xs font-black uppercase text-slate-500 hover:text-white h-12 rounded-xl">
                                <Activity size={18} /> Performance
                            </Button>
                            <Button variant="ghost" className="justify-start gap-4 text-xs font-black uppercase text-slate-500 hover:text-white h-12 rounded-xl">
                                <Terminal size={18} /> Terminal
                            </Button>
                        </nav>
                        <Separator className="my-6 bg-slate-800" />
                        <div className="flex items-center gap-2 text-[10px] font-black text-slate-500 uppercase tracking-widest pb-6">
                            <Badge variant="outline" className="h-5 text-[8px] border-emerald-500/50 text-emerald-500">LIVE</Badge> Secure Engine Active
                        </div>
                    </div>
                </aside>

                <main className="flex-1 overflow-y-auto p-3 sm:p-4 md:p-8">
                    <header className="flex flex-col md:flex-row items-center justify-between mb-6 sm:mb-8 gap-4">
                        <div className="flex items-center justify-between w-full md:w-auto">
                            <div className="flex items-center gap-3 sm:gap-4">
                                <Button variant="ghost" size="icon" className="lg:hidden border border-slate-800 rounded-xl" onClick={() => setIsMobileMenuOpen(true)}>
                                    <Menu size={24} />
                                </Button>
                                <div className="hidden sm:flex w-10 h-10 bg-blue-600 rounded-xl items-center justify-center">
                                    <Server size={20} className="text-white" />
                                </div>
                                <div>
                                    <h1 className="text-xl sm:text-2xl font-black tracking-tight flex items-center gap-2 italic uppercase">
                                        Acma<span className="text-blue-500 not-italic">Dash</span>
                                        <Badge variant="outline" className="hidden xs:inline-flex ml-2 border-blue-500/30 text-blue-400 bg-blue-500/5 h-5 px-1.5 text-[9px] font-bold">PREMIUM</Badge>
                                    </h1>
                                    <p className="text-[10px] sm:text-xs text-slate-500 font-medium whitespace-nowrap">Real-time Orchestration Hub</p>
                                </div>
                            </div>

                            <div className="md:hidden flex items-center gap-2">
                                <div className={`w-2 h-2 rounded-full ${connected ? "bg-emerald-500 animate-pulse" : "bg-amber-500"}`}></div>
                                <span className="text-[9px] font-black uppercase tracking-widest text-slate-400">LIVE</span>
                            </div>
                        </div>

                        <div className="flex items-center gap-2 sm:gap-3 w-full md:w-auto overflow-x-auto pb-2 md:pb-0">
                            <div className="hidden md:flex items-center gap-2 bg-slate-900/80 px-4 py-2 rounded-2xl border border-slate-800">
                                <div className={`w-2 h-2 rounded-full ${connected ? "bg-emerald-500 animate-pulse" : "bg-amber-500"}`}></div>
                                <span className="text-[10px] font-black uppercase tracking-widest text-slate-400 whitespace-nowrap">
                                    {connected ? "Live Operation" : "Connecting..."}
                                </span>
                            </div>
                            <Button size="sm" variant="outline" className="rounded-xl border-slate-800 bg-slate-900/50 hover:bg-slate-800 flex-1 md:flex-initial text-[10px] sm:text-xs">
                                <RefreshCcw size={14} className={`mr-2 ${connected ? "animate-spin-slow" : ""}`} /> Up: {stats ? formatUptime(stats.uptime) : '---'}
                            </Button>
                        </div>
                    </header>

                    <Tabs defaultValue="overview" className="space-y-6 sm:space-y-8 animate-in fade-in duration-500">
                        <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
                            <TabsList className="bg-slate-900/50 border border-slate-800 p-1 rounded-xl w-full sm:w-auto">
                                <TabsTrigger value="overview" className="flex-1 sm:flex-initial rounded-lg px-4 sm:px-6 font-bold text-[10px] sm:text-xs tracking-wider uppercase">Overview</TabsTrigger>
                                <TabsTrigger value="logs" className="flex-1 sm:flex-initial rounded-lg px-4 sm:px-6 font-bold text-[10px] sm:text-xs tracking-wider uppercase">Terminal</TabsTrigger>
                            </TabsList>
                            <div className="hidden sm:flex items-center gap-2 text-slate-500 text-xs font-bold italic truncate max-w-[200px]">
                                <Monitor size={14} /> {stats?.hostname || 'Connecting...'}
                            </div>
                        </div>

                        <TabsContent value="overview" className="space-y-6 sm:space-y-8">
                            {/* Grid adjust for mobile */}
                            <div className="grid grid-cols-1 xs:grid-cols-2 lg:grid-cols-4 gap-4 sm:gap-6">
                                <Card className="bg-slate-900/40 border-slate-800/60 backdrop-blur-md overflow-hidden relative group hover:border-blue-500/40 transition-all duration-300">
                                    <CardHeader className="pb-2 p-4">
                                        <div className="flex justify-between items-center text-[10px] font-black text-slate-500 tracking-[0.2em] uppercase">
                                            <span>CPU</span>
                                            <Cpu size={14} className="text-blue-500" />
                                        </div>
                                    </CardHeader>
                                    <CardContent className="p-4 pt-0">
                                        <div className="text-2xl sm:text-3xl font-black italic mb-2 tracking-tight">{stats?.cpu.toFixed(1) || '0.0'}%</div>
                                        <Progress value={stats?.cpu || 0} className="h-1.5 bg-slate-800 overflow-hidden"
                                            indicatorClassName={stats && stats.cpu > 85 ? "bg-red-500" : "bg-blue-500"}
                                        />
                                    </CardContent>
                                    <CardFooter className="p-4 pt-0 text-[8px] text-slate-600 font-black uppercase tracking-widest">
                                        Processor load
                                    </CardFooter>
                                </Card>

                                <Card className="bg-slate-900/40 border-slate-800/60 backdrop-blur-md overflow-hidden relative group hover:border-emerald-500/40 transition-all duration-300">
                                    <CardHeader className="pb-2 p-4">
                                        <div className="flex justify-between items-center text-[10px] font-black text-slate-500 tracking-[0.2em] uppercase">
                                            <span>RAM</span>
                                            <Database size={14} className="text-emerald-500" />
                                        </div>
                                    </CardHeader>
                                    <CardContent className="p-4 pt-0">
                                        <div className="text-2xl sm:text-3xl font-black italic mb-2 tracking-tight">{stats?.ram.toFixed(1) || '0.0'}%</div>
                                        <Progress value={stats?.ram || 0} className="h-1.5 bg-slate-800 overflow-hidden"
                                            indicatorClassName="bg-emerald-500"
                                        />
                                    </CardContent>
                                    <CardFooter className="p-4 pt-0 text-[8px] text-slate-600 font-black uppercase tracking-widest flex justify-between w-full">
                                        <span>{stats ? formatBytes(stats.ram_used) : '---'}</span>
                                        <span>OF {stats ? formatBytes(stats.ram_total).split(' ')[0] : '---'}</span>
                                    </CardFooter>
                                </Card>

                                <Card className="bg-slate-900/40 border-slate-800/60 backdrop-blur-md overflow-hidden relative group hover:border-amber-500/40 transition-all duration-300">
                                    <CardHeader className="pb-2 p-4">
                                        <div className="flex justify-between items-center text-[10px] font-black text-slate-500 tracking-[0.2em] uppercase">
                                            <span>DISK</span>
                                            <HardDrive size={14} className="text-amber-500" />
                                        </div>
                                    </CardHeader>
                                    <CardContent className="p-4 pt-0">
                                        <div className="text-2xl sm:text-3xl font-black italic mb-2 tracking-tight">{stats?.disk.toFixed(1) || '0.0'}%</div>
                                        <Progress value={stats?.disk || 0} className="h-1.5 bg-slate-800 overflow-hidden"
                                            indicatorClassName="bg-amber-500"
                                        />
                                    </CardContent>
                                    <CardFooter className="p-4 pt-0 text-[8px] text-slate-600 font-black uppercase tracking-widest flex justify-between w-full">
                                        <span>FREE: {stats ? formatBytes(stats.disk_total - stats.disk_used) : '---'}</span>
                                    </CardFooter>
                                </Card>

                                <Card className="bg-slate-900/40 border-slate-800/60 backdrop-blur-md overflow-hidden relative group hover:border-indigo-500/40 transition-all duration-300">
                                    <CardHeader className="pb-2 p-4">
                                        <div className="flex justify-between items-center text-[10px] font-black text-slate-500 tracking-[0.2em] uppercase">
                                            <span>NET</span>
                                            <CloudLightning size={14} className="text-indigo-500" />
                                        </div>
                                    </CardHeader>
                                    <CardContent className="p-4 pt-0">
                                        <div className="text-2xl sm:text-3xl font-black italic mb-0 tracking-tight leading-none truncate overflow-hidden">
                                            {stats ? formatBytes(stats.net_recv) : '0.0'}
                                        </div>
                                        <div className="text-[10px] text-slate-500 font-black mt-1 uppercase tracking-tighter">Inbound data</div>
                                    </CardContent>
                                    <CardFooter className="p-4 pt-0 text-[8px] text-slate-600 font-black uppercase tracking-widest flex justify-between w-full">
                                        <span>OUT: {stats ? formatBytes(stats.net_sent).split(' ')[0] : '---'}</span>
                                        <span className="flex items-center gap-1.5"><LinkIcon size={10} /> {stats?.connections || 0}</span>
                                    </CardFooter>
                                </Card>
                            </div>

                            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 sm:gap-8">
                                <Card className="lg:col-span-2 bg-slate-900/30 border-slate-800/60 backdrop-blur-md">
                                    <CardHeader className="flex flex-row items-center justify-between p-4 sm:p-6 pb-0">
                                        <div>
                                            <CardTitle className="text-xs sm:text-sm font-black italic tracking-widest uppercase">Performance Stream</CardTitle>
                                            <CardDescription className="text-[9px] sm:text-[10px] uppercase font-bold text-slate-600 tracking-tighter">Real-time CPU track via SSE</CardDescription>
                                        </div>
                                        <Badge variant="secondary" className="bg-blue-500/10 text-blue-400 border-none text-[8px] sm:text-[9px] font-black h-5">LIVE</Badge>
                                    </CardHeader>
                                    <CardContent className="h-[250px] sm:h-[350px] p-0 pt-4 overflow-hidden">
                                        <ResponsiveContainer width="100%" height="100%">
                                            <AreaChart data={history}>
                                                <defs>
                                                    <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                                                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                                                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                                                    </linearGradient>
                                                </defs>
                                                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#1e293b" />
                                                <XAxis dataKey="time" stroke="#475569" fontSize={9} tickLine={false} axisLine={false} minTickGap={30} />
                                                <YAxis stroke="#475569" fontSize={9} tickLine={false} axisLine={false} domain={[0, 100]} />
                                                <Tooltip
                                                    contentStyle={{ backgroundColor: '#020617', border: '1px solid #1e293b', borderRadius: '12px', fontSize: '10px' }}
                                                />
                                                <Area type="monotone" dataKey="cpu" stroke="#3b82f6" strokeWidth={3} fillOpacity={1} fill="url(#colorCpu)" animationDuration={400} isAnimationActive={history.length < 2} />
                                            </AreaChart>
                                        </ResponsiveContainer>
                                    </CardContent>
                                </Card>

                                <div className="space-y-6">
                                    <Card className="bg-slate-900/30 border-slate-800/60 backdrop-blur-md">
                                        <CardHeader className="pb-3 p-4">
                                            <CardTitle className="text-[10px] font-black italic tracking-widest uppercase text-slate-400">Environment</CardTitle>
                                        </CardHeader>
                                        <CardContent className="space-y-4 p-4 pt-0">
                                            <div className="flex items-center justify-between group">
                                                <div className="flex items-center gap-3">
                                                    <div className="p-2 bg-slate-800/50 rounded-lg"><Globe size={14} className="text-blue-400" /></div>
                                                    <span className="text-[9px] font-black text-slate-500 uppercase tracking-tighter">OS Platform</span>
                                                </div>
                                                <span className="text-[10px] font-bold text-slate-200 uppercase italic truncate max-w-[100px]">{stats?.platform || 'Linux'}</span>
                                            </div>
                                            <Separator className="bg-slate-800/50" />
                                            <div className="flex items-center justify-between group">
                                                <div className="flex items-center gap-3">
                                                    <div className="p-2 bg-slate-800/50 rounded-lg"><Hash size={14} className="text-amber-400" /></div>
                                                    <span className="text-[9px] font-black text-slate-500 uppercase tracking-tighter">Kernel</span>
                                                </div>
                                                <span className="text-[10px] font-bold text-slate-200 truncate max-w-[100px]">{stats?.kernel || '---'}</span>
                                            </div>
                                            <Separator className="bg-slate-800/50" />
                                            <div className="flex items-center justify-between group">
                                                <div className="flex items-center gap-3">
                                                    <div className="p-2 bg-slate-800/50 rounded-lg"><ShieldCheck size={14} className="text-emerald-400" /></div>
                                                    <span className="text-[9px] font-black text-slate-500 uppercase tracking-tighter">State</span>
                                                </div>
                                                <Badge className="bg-emerald-500/10 text-emerald-500 border-none text-[8px] font-black uppercase px-2 h-5">ACTIVE</Badge>
                                            </div>
                                        </CardContent>
                                    </Card>

                                    <Card className="bg-gradient-to-br from-indigo-600/10 to-transparent border-indigo-500/20 backdrop-blur-md border-dashed">
                                        <CardContent className="p-5 flex flex-col items-center text-center gap-2">
                                            <div className="p-3 bg-indigo-500/10 rounded-full">
                                                <ShieldCheck size={24} className="text-indigo-400" />
                                            </div>
                                            <h4 className="text-[10px] font-black uppercase tracking-widest italic text-indigo-300">Security Active</h4>
                                            <p className="text-[8px] text-slate-500 font-bold leading-relaxed uppercase">Threat protection and AI monitor initialized.</p>
                                        </CardContent>
                                    </Card>
                                </div>
                            </div>
                        </TabsContent>

                        <TabsContent value="logs" className="space-y-4 sm:space-y-6">
                            <div className="flex flex-col lg:flex-row gap-4 sm:gap-6 h-auto lg:h-[700px]">
                                {/* Mobile Log Navigation (Horizontal Scroll) */}
                                <div className="lg:hidden flex overflow-x-auto gap-2 pb-2 no-scrollbar">
                                    <Button
                                        onClick={() => setLogTab('system')}
                                        variant={logTab === 'system' ? 'secondary' : 'outline'}
                                        className={`rounded-xl font-bold text-[10px] uppercase italic tracking-tighter h-10 px-4 whitespace-nowrap transition-all ${logTab === 'system' ? 'bg-blue-600/20 text-blue-400 border-blue-500/30' : 'text-slate-500 border-slate-800'}`}
                                    >
                                        <Terminal size={12} className="mr-2" /> System Journal
                                    </Button>
                                    <Button
                                        onClick={() => setLogTab('nginx_access')}
                                        variant={logTab === 'nginx_access' ? 'secondary' : 'outline'}
                                        className={`rounded-xl font-bold text-[10px] uppercase italic tracking-tighter h-10 px-4 whitespace-nowrap transition-all ${logTab === 'nginx_access' ? 'bg-emerald-600/20 text-emerald-400 border-emerald-500/30' : 'text-slate-500 border-slate-800'}`}
                                    >
                                        <Globe size={12} className="mr-2" /> Nginx Access
                                    </Button>
                                    <Button
                                        onClick={() => setLogTab('nginx_error')}
                                        variant={logTab === 'nginx_error' ? 'secondary' : 'outline'}
                                        className={`rounded-xl font-bold text-[10px] uppercase italic tracking-tighter h-10 px-4 whitespace-nowrap transition-all ${logTab === 'nginx_error' ? 'bg-red-600/20 text-red-400 border-red-500/30' : 'text-slate-500 border-slate-800'}`}
                                    >
                                        <AlertTriangle size={12} className="mr-2 text-red-500" /> Nginx Error
                                    </Button>
                                </div>

                                {/* Desktop Log Navigation */}
                                <Card className="hidden lg:flex flex-col w-64 bg-slate-900/30 border-slate-800 backdrop-blur-xl h-fit">
                                    <CardHeader className="p-4 py-3">
                                        <CardTitle className="text-[10px] font-black uppercase tracking-widest text-slate-500 italic">Select Stream</CardTitle>
                                    </CardHeader>
                                    <CardContent className="p-2 space-y-1">
                                        <Button
                                            onClick={() => setLogTab('system')}
                                            variant={logTab === 'system' ? 'secondary' : 'ghost'}
                                            className={`w-full justify-start rounded-xl font-bold text-xs uppercase italic tracking-tighter transition-all ${logTab === 'system' ? 'bg-blue-600/20 text-blue-400 border border-blue-500/30' : 'text-slate-500'}`}
                                        >
                                            <Terminal size={14} className="mr-3" /> System Journal
                                        </Button>
                                        <Button
                                            onClick={() => setLogTab('nginx_access')}
                                            variant={logTab === 'nginx_access' ? 'secondary' : 'ghost'}
                                            className={`w-full justify-start rounded-xl font-bold text-xs uppercase italic tracking-tighter transition-all ${logTab === 'nginx_access' ? 'bg-emerald-600/20 text-emerald-400 border border-emerald-500/30' : 'text-slate-500'}`}
                                        >
                                            <Globe size={14} className="mr-3" /> Nginx Access
                                        </Button>
                                        <Button
                                            onClick={() => setLogTab('nginx_error')}
                                            variant={logTab === 'nginx_error' ? 'secondary' : 'ghost'}
                                            className={`w-full justify-start rounded-xl font-bold text-xs uppercase italic tracking-tighter transition-all ${logTab === 'nginx_error' ? 'bg-red-600/20 text-red-400 border border-red-500/30' : 'text-slate-500'}`}
                                        >
                                            <AlertTriangle size={14} className="mr-3 text-red-500" /> Nginx Errors
                                        </Button>
                                    </CardContent>
                                </Card>

                                {/* Terminal Content */}
                                <Card className="flex-1 bg-slate-950/80 border-slate-800 backdrop-blur-xl overflow-hidden flex flex-col shadow-2xl h-[500px] lg:h-full">
                                    <CardHeader className="bg-slate-900/80 border-b border-slate-800/60 p-3 sm:p-4 pb-2 sm:pb-3 flex flex-row items-center justify-between">
                                        <div className="flex items-center gap-2 sm:gap-3">
                                            <div className="hidden xs:flex gap-1.5">
                                                <div className="w-2 h-2 rounded-full bg-red-500/40"></div>
                                                <div className="w-2 h-2 rounded-full bg-amber-500/40"></div>
                                                <div className="w-2 h-2 rounded-full bg-emerald-500/40"></div>
                                            </div>
                                            <div className="hidden xs:block h-4 w-px bg-slate-800 mx-1 sm:mx-2"></div>
                                            <div className="flex items-center gap-2 text-[8px] sm:text-[10px] font-black text-slate-500 uppercase tracking-widest italic truncate max-w-[150px] sm:max-w-none">
                                                <FileText size={12} /> {logs?.[logTab]?.path || 'Loading path...'}
                                            </div>
                                        </div>
                                        <Badge variant="outline" className="text-[7px] sm:text-[8px] font-black border-blue-500/30 text-blue-500 uppercase px-1.5 sm:px-2">STREAMING</Badge>
                                    </CardHeader>
                                    <CardContent className="p-0 flex-1 overflow-hidden">
                                        <ScrollArea className="h-full w-full p-4 sm:p-6 bg-slate-950/40">
                                            <pre className="text-slate-400 font-mono text-[10px] sm:text-xs md:text-sm whitespace-pre-wrap leading-relaxed tracking-tight selection:bg-blue-500/20">
                                                {logs?.[logTab]?.content || 'Bufferizing log data stream...'}
                                            </pre>
                                        </ScrollArea>
                                    </CardContent>
                                    <Separator className="bg-slate-800/60" />
                                    <div className="px-4 sm:px-6 py-2 sm:py-3 flex items-center justify-between bg-slate-900/40 text-slate-600 text-[8px] sm:text-[9px] font-black uppercase tracking-[0.2em] italic">
                                        <span className="hidden xs:inline">Payload Sync Active</span>
                                        <span className="xs:hidden">Live Sync</span>
                                        <div className="flex items-center gap-4">
                                            <RefreshCcw size={10} className="animate-spin-slow text-blue-500" />
                                        </div>
                                    </div>
                                </Card>
                            </div>
                        </TabsContent>
                    </Tabs>
                </main>
            </div>

            <footer className="mt-auto p-4 sm:px-4 py-6 sm:py-8 border-t border-slate-900/80 bg-slate-950/50 backdrop-blur-lg">
                <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-6">
                    <div className="flex items-center gap-6 sm:gap-10">
                        <div className="flex flex-col">
                            <span className="text-[8px] sm:text-[9px] font-black text-slate-600 uppercase tracking-[0.3em] mb-1">Architect</span>
                            <span className="text-[10px] sm:text-xs font-black text-slate-300 italic uppercase">AcmaDash Engine v{Version}</span>
                        </div>
                        <div className="hidden xs:block h-6 w-px bg-slate-900"></div>
                        <div className="flex flex-col">
                            <span className="text-[8px] sm:text-[9px] font-black text-slate-600 uppercase tracking-[0.3em] mb-1">Framework</span>
                            <span className="text-[10px] sm:text-xs font-black text-slate-300 italic uppercase">React Premium v2024</span>
                        </div>
                    </div>

                    <div className="flex flex-col items-center md:items-end">
                        <span className="text-[8px] sm:text-[10px] font-black text-slate-600 uppercase tracking-widest">&copy; 2024 Premium Ecosystem</span>
                        <div className="flex items-center gap-2 sm:gap-4 mt-2">
                            <div className="h-1 w-1 sm:h-1.5 sm:w-1.5 rounded-full bg-blue-500 shadow-[0_0_8px_rgba(59,130,246,0.6)] animate-pulse"></div>
                            <span className="text-[8px] font-black text-slate-500 uppercase tracking-widest">Security Monitoring Global</span>
                        </div>
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
