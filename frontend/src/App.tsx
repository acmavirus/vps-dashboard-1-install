import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
    Activity,
    Cpu,
    Database,
    HardDrive,
    RefreshCcw,
    ShieldCheck,
    Server,
    CloudLightning
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
    disk: number;
    uptime: number;
}

const App: React.FC = () => {
    const [stats, setStats] = useState<Stats>({ cpu: 0, ram: 0, disk: 0, uptime: 0 });
    const [history, setHistory] = useState<{ time: string; cpu: number }[]>([]);

    const fetchStats = async () => {
        try {
            const res = await axios.get('/api/stats');
            const newStat = res.data;
            setStats(newStat);

            const now = new Date();
            const timeStr = now.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });

            setHistory(prev => {
                const updated = [...prev, { time: timeStr, cpu: newStat.cpu }];
                if (updated.length > 20) return updated.slice(1);
                return updated;
            });
        } catch (error) {
            console.error("Lỗi lấy thông số:", error);
        }
    };

    useEffect(() => {
        fetchStats();
        const interval = setInterval(fetchStats, 3000);
        return () => clearInterval(interval);
    }, []);

    return (
        <div className="min-h-screen bg-[#0f172a] text-white p-4 md:p-8 font-sans selection:bg-blue-500/30">
            {/* Header */}
            <header className="flex items-center justify-between mb-10 max-w-7xl mx-auto">
                <div className="flex items-center gap-3">
                    <div className="w-12 h-12 bg-blue-600 rounded-xl flex items-center justify-center shadow-lg shadow-blue-500/20">
                        <Server size={28} className="text-white" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold tracking-tight">VPS Premium <span className="text-blue-500">Dashboard</span></h1>
                        <p className="text-sm text-slate-400 flex items-center gap-1">
                            <ShieldCheck size={14} className="text-emerald-500" /> Hệ thống đang an toàn và ổn định
                        </p>
                    </div>
                </div>

                <div className="hidden md:flex gap-4">
                    <div className="px-4 py-2 bg-slate-800/50 rounded-lg border border-slate-700/50 backdrop-blur-md flex items-center gap-2">
                        <Activity size={18} className="text-emerald-400" />
                        <span className="text-sm font-medium">Uptime: {(stats.uptime / (1024 * 1024 * 1024)).toFixed(0)} GB RAM Total</span>
                    </div>
                </div>
            </header>

            <main className="max-w-7xl mx-auto space-y-8">
                {/* Stats Grid */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                    {/* CPU Card */}
                    <div className="bg-slate-800/40 border border-slate-700/50 backdrop-blur-xl rounded-2xl p-6 hover:border-blue-500/50 transition-all duration-300 group">
                        <div className="flex justify-between items-start mb-4">
                            <div className="p-3 bg-blue-500/10 rounded-lg group-hover:bg-blue-500/20 transition-colors">
                                <Cpu size={24} className="text-blue-400" />
                            </div>
                            <span className={`text-xs font-bold px-2 py-1 rounded-full ${stats.cpu > 80 ? 'bg-red-500/10 text-red-500' : 'bg-emerald-500/10 text-emerald-500'}`}>
                                {stats.cpu > 80 ? 'HIGH LOAD' : 'NORMAL'}
                            </span>
                        </div>
                        <h3 className="text-slate-400 text-sm font-medium mb-1">CPU Usage</h3>
                        <div className="flex items-baseline gap-2">
                            <span className="text-3xl font-bold">{stats.cpu.toFixed(1)}%</span>
                            <span className="text-xs text-slate-500 tracking-wider">PROCESSOR LOAD</span>
                        </div>
                        <div className="mt-4 w-full bg-slate-700 h-1.5 rounded-full overflow-hidden">
                            <div
                                className={`h-full transition-all duration-500 ease-out ${stats.cpu > 80 ? 'bg-red-500' : 'bg-blue-500'}`}
                                style={{ width: `${stats.cpu}%` }}
                            ></div>
                        </div>
                    </div>

                    {/* RAM Card */}
                    <div className="bg-slate-800/40 border border-slate-700/50 backdrop-blur-xl rounded-2xl p-6 hover:border-emerald-500/50 transition-all duration-300 group">
                        <div className="flex justify-between items-start mb-4">
                            <div className="p-3 bg-emerald-500/10 rounded-lg group-hover:bg-emerald-500/20 transition-colors">
                                <Database size={24} className="text-emerald-400" />
                            </div>
                        </div>
                        <h3 className="text-slate-400 text-sm font-medium mb-1">RAM Memory</h3>
                        <div className="flex items-baseline gap-2">
                            <span className="text-3xl font-bold">{stats.ram.toFixed(1)}%</span>
                            <span className="text-xs text-slate-500 tracking-wider">AVAILABLE MEMORY</span>
                        </div>
                        <div className="mt-4 w-full bg-slate-700 h-1.5 rounded-full overflow-hidden">
                            <div
                                className="h-full bg-emerald-500 transition-all duration-500 ease-out"
                                style={{ width: `${stats.ram}%` }}
                            ></div>
                        </div>
                    </div>

                    {/* Disk Card */}
                    <div className="bg-slate-800/40 border border-slate-700/50 backdrop-blur-xl rounded-2xl p-6 hover:border-amber-500/50 transition-all duration-300 group">
                        <div className="flex justify-between items-start mb-4">
                            <div className="p-3 bg-amber-500/10 rounded-lg group-hover:bg-amber-500/20 transition-colors">
                                <HardDrive size={24} className="text-amber-400" />
                            </div>
                        </div>
                        <h3 className="text-slate-400 text-sm font-medium mb-1">Disk Storage</h3>
                        <div className="flex items-baseline gap-2">
                            <span className="text-3xl font-bold">{stats.disk.toFixed(1)}%</span>
                            <span className="text-xs text-slate-500 tracking-wider">REMAINING STORAGE</span>
                        </div>
                        <div className="mt-4 w-full bg-slate-700 h-1.5 rounded-full overflow-hidden">
                            <div
                                className="h-full bg-amber-500 transition-all duration-500 ease-out"
                                style={{ width: `${stats.disk}%` }}
                            ></div>
                        </div>
                    </div>
                </div>

                {/* Chart Section */}
                <div className="bg-slate-800/40 border border-slate-700/50 backdrop-blur-xl rounded-2xl p-6">
                    <div className="flex items-center justify-between mb-8">
                        <div className="flex items-center gap-3">
                            <CloudLightning size={20} className="text-blue-400" />
                            <h2 className="text-lg font-bold">Lịch sử CPU thời gian thực</h2>
                        </div>
                        <div className="flex items-center gap-4 text-xs font-mono text-slate-500">
                            <span className="flex items-center gap-1.5"><RefreshCcw size={12} className="animate-spin" /> Live Updates Every 3s</span>
                        </div>
                    </div>

                    <div className="h-[350px] w-full">
                        <ResponsiveContainer width="100%" height="100%">
                            <AreaChart data={history}>
                                <defs>
                                    <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                                    </linearGradient>
                                </defs>
                                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#334155" />
                                <XAxis
                                    dataKey="time"
                                    stroke="#64748b"
                                    fontSize={12}
                                    tickLine={false}
                                    axisLine={false}
                                    minTickGap={30}
                                />
                                <YAxis
                                    stroke="#64748b"
                                    fontSize={12}
                                    tickLine={false}
                                    axisLine={false}
                                    domain={[0, 100]}
                                />
                                <Tooltip
                                    contentStyle={{ backgroundColor: '#1e293b', border: '1px solid #334155', borderRadius: '8px' }}
                                    itemStyle={{ color: '#3b82f6', fontWeight: 'bold' }}
                                />
                                <Area
                                    type="monotone"
                                    dataKey="cpu"
                                    stroke="#3b82f6"
                                    strokeWidth={3}
                                    fillOpacity={1}
                                    fill="url(#colorCpu)"
                                    animationDuration={1500}
                                />
                            </AreaChart>
                        </ResponsiveContainer>
                    </div>
                </div>
            </main>

            <footer className="mt-12 text-center text-slate-500 text-sm max-w-7xl mx-auto border-t border-slate-800 pt-8">
                <p>&copy; 2024 VPS Dashboard | Thiết kế chuẩn Premium cho quản lý hệ thống chuyên nghiệp</p>
            </footer>
        </div>
    );
};

export default App;
