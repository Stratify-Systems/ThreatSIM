import { useEffect, useState, useRef, useMemo } from 'react';
import { ShieldAlert, Activity, Cpu, Server, Play, Trash2, Wifi, WifiOff, AlertTriangle } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface Simulation { id: string; plugin_id: string; target: string; status: string; }
interface Event { id: string; event_type: string; source_ip: string; target: string; timestamp: string; plugin_id: string; }
interface Alert { source_ip: string; score: number; threat_level: string; factors: string[]; updated_at: string; }

export default function App() {
  const [events, setEvents] = useState<Event[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [simulations, setSimulations] = useState<Simulation[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const ws = useRef<WebSocket | null>(null);

  const fetchState = async () => {
    try {
      const [eRes, aRes, sRes] = await Promise.all([
        fetch('/api/v1/events').catch(() => ({ json: () => [] as Event[] })),
        fetch('/api/v1/alerts').catch(() => ({ json: () => [] as Alert[] })),
        fetch('/api/v1/simulations').catch(() => ({ json: () => [] as Simulation[] }))
      ]);
      setEvents((await eRes.json()) || []);
      setAlerts((await aRes.json()) || []);
      setSimulations((await sRes.json()) || []);
    } catch(err) {
      console.error("Fetch error", err);
    }
  };

  useEffect(() => {
    fetchState();
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws.current = new WebSocket(`${protocol}//${window.location.host}/ws/live`);
    
    ws.current.onopen = () => setIsConnected(true);
    ws.current.onclose = () => setIsConnected(false);

    ws.current.onmessage = (msg) => {
      try {
        const data = JSON.parse(msg.data);
        if (data.type === 'event' || data.event_type) {
           setEvents(prev => [data, ...prev].slice(0, 100)); // keep last 100 to prevent lag
        } else if (data.type === 'alert' || data.threat_level) {
           setAlerts(prev => [data, ...prev.filter(a => a.source_ip !== data.source_ip)]);
        }
      } catch (e) {}
    };
    return () => ws.current?.close();
  }, []);

  const launchAttack = async () => {
    await fetch('/api/v1/simulations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ plugin_id: 'brute_force', target: '10.0.0.100', duration: '5s', rate: 10 })
    });
    fetchState();
  };

  const chartData = useMemo(() => {
    // Generate a simple time-series distribution from the last N events
    const buckets: Record<string, number> = {};
    events.forEach(e => {
      const time = new Date(e.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
      buckets[time] = (buckets[time] || 0) + 1;
    });
    return Object.entries(buckets).reverse().map(([time, count]) => ({ time, events: count })).slice(-20);
  }, [events]);

  const activeThreats = alerts.filter(a => a.threat_level === 'CRITICAL' || a.threat_level === 'HIGH').length;

  return (
    <div className="min-h-screen bg-slate-950 text-slate-300 font-sans p-6">
      
      {/* HEADER */}
      <header className="flex justify-between items-center mb-8 border-b border-slate-800 pb-4">
        <div className="flex items-center gap-3 text-cyan-400">
          <ShieldAlert size={32} />
          <h1 className="text-3xl font-bold tracking-tight text-white">ThreatSIM</h1>
        </div>
        <div className={`flex items-center gap-2 px-4 py-2 rounded-full font-semibold text-sm ${isConnected ? 'bg-emerald-900/40 text-emerald-400 border border-emerald-800' : 'bg-rose-900/40 text-rose-400 border border-rose-800'}`}>
          {isConnected ? <Wifi size={18} /> : <WifiOff size={18} />}
          {isConnected ? 'LIVE TELEMETRY' : 'DISCONNECTED'}
        </div>
      </header>

      {/* KPI CARDS */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
        <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-4 opacity-10"><Activity size={64} /></div>
          <p className="text-slate-400 text-sm font-medium mb-1">Total Events Captured</p>
          <p className="text-4xl font-bold text-white">{events.length}</p>
        </div>
        <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-4 opacity-10"><AlertTriangle size={64} /></div>
          <p className="text-slate-400 text-sm font-medium mb-1">Active Alerts</p>
          <p className={`text-4xl font-bold ${activeThreats > 0 ? 'text-rose-500' : 'text-emerald-400'}`}>{alerts.length}</p>
        </div>
        <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-4 opacity-10"><Server size={64} /></div>
          <p className="text-slate-400 text-sm font-medium mb-1">Simulations Running</p>
          <p className="text-4xl font-bold text-cyan-400">{simulations.filter(s => s.status === 'RUNNING').length}</p>
        </div>
        <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl flex flex-col justify-center gap-3">
          <button onClick={launchAttack} className="flex items-center justify-center gap-2 w-full bg-cyan-600 hover:bg-cyan-500 text-white font-bold py-3 rounded-lg transition-colors">
            <Play size={20} fill="currentColor" /> LAUNCH ATTACK
          </button>
        </div>
      </div>

      {/* DASHBOARD GRIDS */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        
        {/* CHART SPANNING 2 COLUMNS */}
        <div className="lg:col-span-2 bg-slate-900 border border-slate-800 p-6 rounded-xl">
          <h2 className="text-lg font-bold text-white mb-6 flex items-center gap-2"><Activity size={20} className="text-cyan-400"/> Network Activity</h2>
          <div className="h-72">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="colorEvt" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#22d3ee" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#22d3ee" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#334155" vertical={false} />
                <XAxis dataKey="time" stroke="#94a3b8" fontSize={12} tickLine={false} axisLine={false} />
                <YAxis stroke="#94a3b8" fontSize={12} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={{ backgroundColor: '#0f172a', borderColor: '#1e293b', color: '#f8fafc' }} />
                <Area type="monotone" dataKey="events" stroke="#22d3ee" strokeWidth={2} fillOpacity={1} fill="url(#colorEvt)" isAnimationActive={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* ALERTS TABLE */}
        <div className="bg-slate-900 border border-slate-800 p-6 rounded-xl flex flex-col">
          <h2 className="text-lg font-bold text-white mb-4 flex items-center gap-2"><ShieldAlert size={20} className="text-rose-400"/> Threat Intelligence</h2>
          <div className="flex-1 overflow-y-auto max-h-72 pr-2 space-y-3">
            {alerts.length === 0 ? (
              <p className="text-slate-500 italic text-center mt-10">No active threats detected.</p>
            ) : (
              alerts.map(a => (
                <div key={a.source_ip} className="bg-slate-950 p-3 rounded border border-slate-800 flex justify-between items-center">
                  <div>
                    <p className="font-mono text-sm text-cyan-300">{a.source_ip}</p>
                    <p className="text-xs text-slate-500 mt-1">{a.factors[0] || 'Unknown anomaly'}</p>
                  </div>
                  <div className={`px-2 py-1 rounded text-xs font-bold ${a.threat_level === 'CRITICAL' ? 'bg-rose-900/50 text-rose-400' : 'bg-orange-900/50 text-orange-400'}`}>
                    {a.threat_level} ({a.score})
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* RAW EVENTS STREAM */}
        <div className="lg:col-span-3 bg-slate-950 border border-slate-800 rounded-xl overflow-hidden shadow-inner">
          <div className="bg-slate-900 px-4 py-3 border-b border-slate-800 flex items-center gap-2">
            <Cpu size={18} className="text-emerald-400"/> 
            <h3 className="font-bold text-white">Live Event Stream</h3>
          </div>
          <div className="p-4 h-64 overflow-y-auto font-mono text-sm space-y-1">
            {events.length === 0 && <p className="text-slate-600">Waiting for telemetry data...</p>}
            {events.slice(0, 30).map((evt, i) => (
              <div key={evt.id || i} className="flex gap-4 hover:bg-slate-800/50 py-1 px-2 rounded">
                <span className="text-slate-500 shrink-0 w-24">{new Date(evt.timestamp).toLocaleTimeString()}</span>
                <span className="text-cyan-400 shrink-0 w-24">{evt.source_ip}</span>
                <span className="text-emerald-400 shrink-0 w-28">[{evt.plugin_id}]</span>
                <span className="text-slate-300 flex-1 truncate">{evt.event_type} at {evt.target}</span>
              </div>
            ))}
          </div>
        </div>

      </div>
    </div>
  );
}
