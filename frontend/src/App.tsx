import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { io } from 'socket.io-client';
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
    Info,
    Globe,
    Monitor,
    Hash
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
}

const App: React.FC = () => {
    const [stats, setStats] = useState<Stats | null>(null);
    const [history, setHistory] = useState<{ time: string; cpu: number }[]>([]);
    const [logs, setLogs] = useState<string>('');
    const [logPath, setLogPath] = useState<string>('');
    const [activeTab, setActiveTab] = useState<'stats' | 'logs'>('stats');
    const [connected, setConnected] = useState(false);

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

    useEffect(() => {
        // Socket.io for LIVE stats and logs
        const socket = io({ transports: ['websocket'] });

        socket.on('connect', () => {
            setConnected(true);
            console.log('Socket.io connected');
        });

        socket.on('disconnect', () => {
            setConnected(false);
            console.log('Socket.io disconnected');
        });

        socket.on('stats', (newStat: Stats) => {
            setStats(newStat);
            const now = new Date();
            const timeStr = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });

            setHistory(prev => {
                const updated = [...prev, { time: timeStr, cpu: newStat.cpu }];
                if (updated.length > 20) return updated.slice(1);
                return updated;
            });
        });

        socket.on('logs', (data: { logs: string, path: string }) => {
            setLogs(data.logs);
            setLogPath(data.path);
        });

        // Fallback or Initial fetch
        const fetchInitial = async () => {
            try {
                const res = await axios.get('/api/stats');
                setStats(res.data);
                const lres = await axios.get('/api/logs');
                setLogs(lres.data.logs);
                setLogPath(lres.data.path);
            } catch (e) {
                console.error('Initial fetch error', e);
            }
        };
        fetchInitial();

        return () => {
            socket.disconnect();
        };
    }, []);

    return (
        <div className="min-h-screen bg-[#020617] text-slate-200 p-4 md:p-8 font-sans selection:bg-blue-500/30 overflow-x-hidden">
            {/* Background Decorative Elements */}
            <div className="fixed top-0 left-0 w-full h-full overflow-hidden pointer-events-none -z-10">
                <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/10 blur-[120px] rounded-full"></div>
                <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-emerald-600/10 blur-[120px] rounded-full"></div>
            </div>

            {/* Header */}
            <header className="flex flex-col md:flex-row items-center justify-between mb-10 max-w-7xl mx-auto gap-6 bg-slate-900/40 p-6 rounded-3xl border border-slate-800/50 backdrop-blur-xl">
                <div className="flex items-center gap-4">
                    <div className="w-14 h-14 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-2xl flex items-center justify-center shadow-2xl shadow-blue-500/20">
                        <Server size={32} className="text-white" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-black tracking-tight text-white uppercase italic">
                            Acma<span className="text-blue-500 italic not-italic">Dash</span>
                        </h1>
                        <p className="text-xs text-slate-400 flex items-center gap-1 font-medium">
                            <ShieldCheck size={14} className={connected ? "text-emerald-500" : "text-red-500"} />
                            SYSTEM STATUS: <span className={connected ? "text-emerald-400" : "text-red-400"}>
                                {connected ? "LIVE_OPERATIONAL" : "RECONNECTING..."}
                            </span>
                        </p>
                    </div>
                </div>

                <nav className="flex bg-slate-950/50 p-1.5 rounded-xl border border-slate-800">
                    <button
                        onClick={() => setActiveTab('stats')}
                        className={`px-6 py-2 rounded-lg text-sm font-bold transition-all duration-300 flex items-center gap-2 ${activeTab === 'stats' ? 'bg-blue-600 text-white shadow-lg shadow-blue-500/30' : 'text-slate-400 hover:text-white'}`}
                    >
                        <Activity size={16} /> STATISTICS
                    </button>
                    <button
                        onClick={() => setActiveTab('logs')}
                        className={`px-6 py-2 rounded-lg text-sm font-bold transition-all duration-300 flex items-center gap-2 ${activeTab === 'logs' ? 'bg-blue-600 text-white shadow-lg shadow-blue-500/30' : 'text-slate-400 hover:text-white'}`}
                    >
                        <Terminal size={16} /> SYSTEM LOGS
                    </button>
                </nav>

                <div className="flex gap-4">
                    <div className="px-5 py-2.5 bg-slate-800/30 rounded-xl border border-slate-700/30 backdrop-blur-md flex items-center gap-3">
                        <div className={`w-2 h-2 rounded-full ${connected ? "bg-emerald-500 animate-pulse" : "bg-red-500"}`}></div>
                        <span className="text-xs font-bold tracking-widest text-slate-300 uppercase">
                            {stats ? formatUptime(stats.uptime) : 'LOADING...'}
                        </span>
                    </div>
                </div>
            </header>

            <main className="max-w-7xl mx-auto">
                {activeTab === 'stats' && (
                    <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-700">
                        {/* Summary Info Bar */}
                        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                            <div className="bg-slate-900/40 border border-slate-800/50 p-4 rounded-2xl backdrop-blur-md">
                                <div className="flex items-center gap-3 mb-2">
                                    <Monitor size={16} className="text-blue-400" />
                                    <span className="text-[10px] font-bold text-slate-500 uppercase tracking-tighter">Hostname</span>
                                </div>
                                <div className="text-sm font-bold text-white truncate">{stats?.hostname || '---'}</div>
                            </div>
                            <div className="bg-slate-900/40 border border-slate-800/50 p-4 rounded-2xl backdrop-blur-md">
                                <div className="flex items-center gap-3 mb-2">
                                    <Globe size={16} className="text-emerald-400" />
                                    <span className="text-[10px] font-bold text-slate-500 uppercase tracking-tighter">OS / Platform</span>
                                </div>
                                <div className="text-sm font-bold text-white uppercase italic">{stats?.platform || '---'} ({stats?.os})</div>
                            </div>
                            <div className="bg-slate-900/40 border border-slate-800/50 p-4 rounded-2xl backdrop-blur-md">
                                <div className="flex items-center gap-3 mb-2">
                                    <Hash size={16} className="text-amber-400" />
                                    <span className="text-[10px] font-bold text-slate-500 uppercase tracking-tighter">Kernel</span>
                                </div>
                                <div className="text-sm font-bold text-white truncate">{stats?.kernel || '---'}</div>
                            </div>
                            <div className="bg-slate-900/40 border border-slate-800/50 p-4 rounded-2xl backdrop-blur-md">
                                <div className="flex items-center gap-3 mb-2">
                                    <CloudLightning size={16} className="text-purple-400" />
                                    <span className="text-[10px] font-bold text-slate-500 uppercase tracking-tighter">Network (S/R)</span>
                                </div>
                                <div className="text-sm font-bold text-white truncate">
                                    {stats ? `${formatBytes(stats.net_sent)} / ${formatBytes(stats.net_recv)}` : '---'}
                                </div>
                            </div>
                        </div>

                        {/* Stats Grid */}
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                            {/* CPU Card */}
                            <div className="bg-gradient-to-br from-slate-900/60 to-slate-900/20 border border-slate-800/50 backdrop-blur-2xl rounded-3xl p-8 hover:border-blue-500/50 transition-all duration-500 group relative overflow-hidden">
                                <div className="absolute -right-4 -top-4 w-24 h-24 bg-blue-500/5 rounded-full group-hover:bg-blue-500/10 transition-all duration-700"></div>
                                <div className="flex justify-between items-start mb-6">
                                    <div className="p-4 bg-blue-500/10 rounded-2xl group-hover:scale-110 transition-transform duration-500">
                                        <Cpu size={28} className="text-blue-400" />
                                    </div>
                                    <div className="flex flex-col items-end">
                                        <span className={`text-[10px] font-black px-3 py-1 rounded-full tracking-widest ${stats && stats.cpu > 80 ? 'bg-red-500/20 text-red-400' : 'bg-blue-500/20 text-blue-400'}`}>
                                            {stats && stats.cpu > 80 ? 'CRITICAL' : 'OPTIMAL'}
                                        </span>
                                    </div>
                                </div>
                                <h3 className="text-slate-500 text-xs font-black uppercase tracking-[0.2em] mb-2">CPU Processors</h3>
                                <div className="flex items-baseline gap-2">
                                    <span className="text-5xl font-black text-white italic">{stats?.cpu.toFixed(1) || '0.0'}%</span>
                                </div>
                                <div className="mt-8 w-full bg-slate-800/50 h-2 rounded-full overflow-hidden p-[2px]">
                                    <div
                                        className={`h-full rounded-full transition-all duration-500 ease-out shadow-[0_0_15px_rgba(59,130,246,0.5)] ${stats && stats.cpu > 80 ? 'bg-gradient-to-r from-red-500 to-orange-500' : 'bg-gradient-to-r from-blue-500 to-cyan-400'}`}
                                        style={{ width: `${stats ? stats.cpu : 0}%` }}
                                    ></div>
                                </div>
                            </div>

                            {/* RAM Card */}
                            <div className="bg-gradient-to-br from-slate-900/60 to-slate-900/20 border border-slate-800/50 backdrop-blur-2xl rounded-3xl p-8 hover:border-emerald-500/50 transition-all duration-500 group relative overflow-hidden">
                                <div className="absolute -right-4 -top-4 w-24 h-24 bg-emerald-500/5 rounded-full group-hover:bg-emerald-500/10 transition-all duration-700"></div>
                                <div className="flex justify-between items-start mb-6">
                                    <div className="p-4 bg-emerald-500/10 rounded-2xl group-hover:scale-110 transition-transform duration-500">
                                        <Database size={28} className="text-emerald-400" />
                                    </div>
                                </div>
                                <h3 className="text-slate-500 text-xs font-black uppercase tracking-[0.2em] mb-2">Memory RAM</h3>
                                <div className="flex items-baseline gap-2">
                                    <span className="text-5xl font-black text-white italic">{stats?.ram.toFixed(1) || '0.0'}%</span>
                                    <div className="flex flex-col">
                                        <span className="text-[10px] font-bold text-emerald-500/80 tracking-tighter">USED: {stats ? formatBytes(stats.ram_used) : '---'}</span>
                                        <span className="text-[10px] font-bold text-slate-600 tracking-tighter">TOTAL: {stats ? formatBytes(stats.ram_total) : '---'}</span>
                                    </div>
                                </div>
                                <div className="mt-8 w-full bg-slate-800/50 h-2 rounded-full overflow-hidden p-[2px]">
                                    <div
                                        className="h-full bg-gradient-to-r from-emerald-500 to-teal-400 rounded-full transition-all duration-500 ease-out shadow-[0_0_15px_rgba(16,185,129,0.5)]"
                                        style={{ width: `${stats ? stats.ram : 0}%` }}
                                    ></div>
                                </div>
                            </div>

                            {/* Disk Card */}
                            <div className="bg-gradient-to-br from-slate-900/60 to-slate-900/20 border border-slate-800/50 backdrop-blur-2xl rounded-3xl p-8 hover:border-amber-500/50 transition-all duration-500 group relative overflow-hidden">
                                <div className="absolute -right-4 -top-4 w-24 h-24 bg-amber-500/5 rounded-full group-hover:bg-amber-500/10 transition-all duration-700"></div>
                                <div className="flex justify-between items-start mb-6">
                                    <div className="p-4 bg-amber-500/10 rounded-2xl group-hover:scale-110 transition-transform duration-500">
                                        <HardDrive size={28} className="text-amber-400" />
                                    </div>
                                </div>
                                <h3 className="text-slate-500 text-xs font-black uppercase tracking-[0.2em] mb-2">Storage Disk</h3>
                                <div className="flex items-baseline gap-2">
                                    <span className="text-5xl font-black text-white italic">{stats?.disk.toFixed(1) || '0.0'}%</span>
                                    <div className="flex flex-col">
                                        <span className="text-[10px] font-bold text-amber-500/80 tracking-tighter">FREE: {stats ? formatBytes(stats.disk_total - stats.disk_used) : '---'}</span>
                                        <span className="text-[10px] font-bold text-slate-600 tracking-tighter">TOTAL: {stats ? formatBytes(stats.disk_total) : '---'}</span>
                                    </div>
                                </div>
                                <div className="mt-8 w-full bg-slate-800/50 h-2 rounded-full overflow-hidden p-[2px]">
                                    <div
                                        className="h-full bg-gradient-to-r from-amber-500 to-orange-400 rounded-full transition-all duration-500 ease-out shadow-[0_0_15px_rgba(245,158,11,0.5)]"
                                        style={{ width: `${stats ? stats.disk : 0}%` }}
                                    ></div>
                                </div>
                            </div>
                        </div>

                        {/* Chart Section */}
                        <div className="bg-slate-900/40 border border-slate-800/50 backdrop-blur-2xl rounded-3xl p-8 relative overflow-hidden">
                            <div className="flex items-center justify-between mb-10">
                                <div className="flex items-center gap-4">
                                    <div className="p-2 bg-blue-500/20 rounded-lg">
                                        <Activity size={20} className="text-blue-400" />
                                    </div>
                                    <div>
                                        <h2 className="text-lg font-black uppercase tracking-widest text-white italic">Real-time Performance</h2>
                                        <p className="text-[10px] text-slate-500 font-bold uppercase">CPU load percentage visualization (LIVE)</p>
                                    </div>
                                </div>
                                <div className="flex items-center gap-4 text-[10px] font-black tracking-widest text-blue-500 bg-blue-500/5 px-4 py-2 rounded-full border border-blue-500/10">
                                    <RefreshCcw size={12} className="animate-spin" /> SOCKET_STREAM
                                </div>
                            </div>

                            <div className="h-[400px] w-full">
                                <ResponsiveContainer width="100%" height="100%">
                                    <AreaChart data={history}>
                                        <defs>
                                            <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                                                <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.4} />
                                                <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                                            </linearGradient>
                                        </defs>
                                        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#1e293b" />
                                        <XAxis
                                            dataKey="time"
                                            stroke="#475569"
                                            fontSize={10}
                                            tickLine={false}
                                            axisLine={false}
                                            minTickGap={30}
                                            tick={{ fontWeight: 'bold' }}
                                        />
                                        <YAxis
                                            stroke="#475569"
                                            fontSize={10}
                                            tickLine={false}
                                            axisLine={false}
                                            domain={[0, 100]}
                                            tick={{ fontWeight: 'bold' }}
                                        />
                                        <Tooltip
                                            contentStyle={{ backgroundColor: '#0f172a', border: '1px solid #334155', borderRadius: '16px', boxShadow: '0 20px 25px -5px rgb(0 0 0 / 0.5)' }}
                                            itemStyle={{ color: '#3b82f6', fontWeight: '900', fontSize: '12px' }}
                                            cursor={{ stroke: '#3b82f6', strokeWidth: 2 }}
                                        />
                                        <Area
                                            type="monotone"
                                            dataKey="cpu"
                                            stroke="#3b82f6"
                                            strokeWidth={4}
                                            fillOpacity={1}
                                            fill="url(#colorCpu)"
                                            animationDuration={500}
                                        />
                                    </AreaChart>
                                </ResponsiveContainer>
                            </div>
                        </div>
                    </div>
                )}

                {activeTab === 'logs' && (
                    <div className="animate-in fade-in zoom-in duration-500">
                        <div className="bg-slate-950 border border-slate-800 rounded-3xl overflow-hidden shadow-2xl relative">
                            {/* Terminal Header */}
                            <div className="bg-slate-900 px-6 py-4 border-b border-slate-800 flex items-center justify-between">
                                <div className="flex items-center gap-3">
                                    <div className="flex gap-1.5">
                                        <div className="w-3 h-3 rounded-full bg-red-500/50"></div>
                                        <div className="w-3 h-3 rounded-full bg-amber-500/50"></div>
                                        <div className="w-3 h-3 rounded-full bg-emerald-500/50"></div>
                                    </div>
                                    <span className="ml-4 text-[10px] font-black text-slate-500 tracking-[0.2em] flex items-center gap-2">
                                        <Terminal size={12} /> {logPath}
                                    </span>
                                </div>
                                <div className="flex items-center gap-3">
                                    <div className="text-[10px] font-bold text-emerald-500/80 bg-emerald-500/5 px-3 py-1 rounded-md border border-emerald-500/10">
                                        {connected ? "STOCKED_CONNECTED" : "WAITING..."}
                                    </div>
                                </div>
                            </div>

                            {/* Terminal Content */}
                            <div className="p-8 h-[600px] overflow-y-auto font-mono text-sm group scrollbar-thin scrollbar-thumb-slate-800 scrollbar-track-transparent">
                                <pre className="text-slate-400 whitespace-pre-wrap leading-relaxed selection:bg-blue-500/40">
                                    {logs || 'Waiting for system logs output (Socket.io streaming)...'}
                                </pre>
                            </div>

                            {/* Terminal Footer */}
                            <div className="bg-slate-900/50 px-6 py-3 border-t border-slate-800 flex items-center justify-between">
                                <span className="text-[9px] font-bold text-slate-600 uppercase tracking-widest">
                                    Live Stream Output - Latency: ~100ms
                                </span>
                                <Info size={14} className="text-slate-700" />
                            </div>
                        </div>
                    </div>
                )}
            </main>

            <footer className="mt-16 py-12 text-center max-w-7xl mx-auto border-t border-slate-900/50">
                <div className="flex justify-center items-center gap-8 mb-6">
                    <div className="flex flex-col items-center">
                        <span className="text-[10px] font-black text-slate-600 uppercase tracking-[0.3em] mb-2">Designed by</span>
                        <span className="text-sm font-black text-white italic">AcmaTvirus</span>
                    </div>
                    <div className="border-l border-slate-800 h-8"></div>
                    <div className="flex flex-col items-center">
                        <span className="text-[10px] font-black text-slate-600 uppercase tracking-[0.3em] mb-2">Engine</span>
                        <span className="text-sm font-black text-white italic">Socket.io + Go + Gin</span>
                    </div>
                </div>
                <p className="text-[10px] font-bold text-slate-600 uppercase tracking-widest">
                    &copy; 2024 Premium LIVE Ecosystem | All Rights Reserved
                </p>
            </footer>
        </div>
    );
};

export default App;
