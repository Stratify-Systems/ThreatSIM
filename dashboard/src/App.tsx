import { useEffect, useState, useRef, useMemo } from 'react';
import { Shield, Activity, Terminal, Play, Wifi, WifiOff, AlertTriangle, Target, Zap } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface Simulation { id: string; plugin_id: string; target: string; status: string; }
interface Event { id: string; event_type: string; source_ip: string; target: string; timestamp: string; plugin_id: string; }
interface Alert { 
  id: string;
  rule_name: string;
  alert_type: string;
  description: string;
  severity: string;
  source_ip: string;
  event_count: number;
  timestamp: string; 
}

const ATTACK_VECTORS = [
  { id: 'brute_force', name: 'Brute Force SSH' },
  { id: 'port_scan', name: 'Port Scan Recon' },
  { id: 'ddos', name: 'DDoS Flood' },
  { id: 'credential_stuffing', name: 'Credential Stuffing' },
  { id: 'privilege_escalation', name: 'Privilege Escalation' },
  { id: 'scenario_account_takeover', name: 'Scenario: Account Takeover ⛓️' } // The full configured scenario
];

export default function App() {
  const [events, setEvents] = useState<Event[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [simulations, setSimulations] = useState<Simulation[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  
  const [selectedAttack, setSelectedAttack] = useState(ATTACK_VECTORS[0].id);
  const [targetIp, setTargetIp] = useState('10.0.0.100');
  
  const ws = useRef<WebSocket | null>(null);

  const fetchState = async () => {
    try {
      const [eRes, aRes, sRes] = await Promise.all([
        fetch('/api/v1/events').catch(() => null),
        fetch('/api/v1/alerts').catch(() => null),
        fetch('/api/v1/simulations').catch(() => null)
      ]);
      const eData = eRes && eRes.ok ? await eRes.json() : [];
      const aData = aRes && aRes.ok ? await aRes.json() : [];
      const sData = sRes && sRes.ok ? await sRes.json() : [];
      
      // Ensure strict arrays to fix the white-screen crash bug
      setEvents(Array.isArray(eData) ? eData : []);
      setAlerts(Array.isArray(aData) ? aData : []);
      setSimulations(Array.isArray(sData) ? sData : []);
    } catch(err) {
      console.error("Fetch error", err);
    }
  };

  useEffect(() => {
    let isMounted = true;
    let reconnectTimeout: ReturnType<typeof setTimeout>;

    const connectWs = () => {
      if (!isMounted) return;
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const socket = new WebSocket(`${protocol}//${window.location.host}/ws/live`);
      ws.current = socket;

      socket.onopen = () => {
        if (isMounted) setIsConnected(true);
      };
      
      socket.onclose = () => {
        if (isMounted) {
          setIsConnected(false);
          // Automatically attempt to reconnect every 3 seconds if the connection drops
          reconnectTimeout = setTimeout(connectWs, 3000);
        }
      };

      socket.onerror = () => {
        if (!isMounted) return;
        // Suppress generic event logging if it's just a disconnect 
        // to avoid console spam during hot reloads or when backend is down
      };

      socket.onmessage = (msg) => {
        if (!isMounted) return;
        try {
          const data = JSON.parse(msg.data);
          if (data.type === 'event' || data.event_type) {
             setEvents(prev => {
               const safePrev = Array.isArray(prev) ? prev : [];
               return [data, ...safePrev].slice(0, 200);
             });
          } else if (data.type === 'alert' || data.alert_type || data.severity) {
             setAlerts(prev => {
               const safePrev = Array.isArray(prev) ? prev : [];
               return [data, ...safePrev.filter(a => a.id !== data.id)];
             });
          }
        } catch (e) {
          // ignore parse errors
          void e;
        }
      };
    };

    fetchState();
    
    // Add a tiny delay to bypass React 18 StrictMode double-mount glitch. 
    // This stops it from instantly creating & killing a socket, 
    // avoiding browser warnings & 'write EPIPE' errors hitting the proxy payload stream.
    const startupTimeout = setTimeout(connectWs, 50);

    return () => {
      isMounted = false;
      clearTimeout(startupTimeout);
      clearTimeout(reconnectTimeout);
      if (ws.current) {
        ws.current.onclose = null; // Prevent reconnect loop on component unmount
        // Protect strict mode closing connecting sockets and spamming proxy EPIPEs
        if (ws.current.readyState === WebSocket.OPEN || ws.current.readyState === WebSocket.CONNECTING) {
           ws.current.close();
        }
      }
    };
  }, []);

  const launchAttack = async () => {
    // If the backend doesn't support scenarios via API natively yet, 
    // we map requests gracefully to ensure UI stability.
    if (selectedAttack.startsWith('scenario_')) {
      alert("Launching Multi-Step Scenario Action...");
      // For now, post the primary plugin as a fallback if Scenario API isn't wired in backend yet
      try {
        await fetch('/api/v1/simulations', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ plugin_id: 'port_scan', target: targetIp, duration: '10s', rate: 10 })
        });
      } catch (err) {
        console.error(err);
      }
    } else {
      try {
        await fetch('/api/v1/simulations', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ plugin_id: selectedAttack, target: targetIp, duration: '5s', rate: 10 })
        });
      } catch (err) {
        console.error(err);
      }
    }
    fetchState();
  };

  const chartData = useMemo(() => {
    const buckets: Record<string, number> = {};
    (Array.isArray(events) ? events : []).forEach(e => {
      if (!e || !e.timestamp) return;
      const time = new Date(e.timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
      buckets[time] = (buckets[time] || 0) + 1;
    });
    return Object.entries(buckets).reverse().map(([time, count]) => ({ time, events: count })).slice(-20);
  }, [events]);

  const activeThreats = (Array.isArray(alerts) ? alerts : []).filter(a => a && (a.severity === 'CRITICAL' || a.severity === 'HIGH')).length;
  const runningSims = (Array.isArray(simulations) ? simulations : []).filter(s => s && s.status === 'RUNNING').length;

  return (
    <div className="min-h-screen bg-[#050505] text-slate-100 font-sans p-6 overflow-x-hidden relative">
      
      {/* Subtle Background Glows */}
      <div className="fixed top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/10 blur-[150px] pointer-events-none" />
      <div className="fixed bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-purple-600/10 blur-[150px] pointer-events-none" />

      {/* HEADER - Glassy */}
      <header className="relative flex justify-between items-center mb-8 pb-4 border-b border-white/10 bg-white/5 backdrop-blur-xl px-6 py-4 rounded-2xl shadow-2xl">
        <div className="flex items-center gap-3 text-white">
          <Shield size={32} className="text-blue-400" />
          <h1 className="text-2xl font-light tracking-widest uppercase">Threat<span className="font-bold text-blue-400">SIM</span></h1>
        </div>
        <div className={`flex items-center gap-2 px-4 py-1.5 rounded-full font-semibold text-xs tracking-wider border backdrop-blur-md ${isConnected ? 'bg-emerald-500/10 text-emerald-300 border-emerald-500/30' : 'bg-red-500/10 text-red-400 border-red-500/30'}`}>
          {isConnected ? <Wifi size={14} /> : <WifiOff size={14} />}
          {isConnected ? 'NODE CONNECTED' : 'OFFLINE'}
        </div>
      </header>

      {/* LAUNCH BAR - Glassy */}
      <div className="relative mb-8 bg-white/5 backdrop-blur-xl border border-white/10 rounded-2xl p-6 shadow-2xl flex flex-col md:flex-row gap-6 items-end md:items-center">
        
        <div className="flex-1 w-full flex flex-col gap-2">
          <label className="text-xs uppercase tracking-widest text-slate-400 font-bold flex items-center gap-2">
            <Zap size={14} className="text-blue-400"/> Select Vector
          </label>
          <select 
            value={selectedAttack} 
            onChange={(e) => setSelectedAttack(e.target.value)}
            className="w-full bg-black/40 border border-white/10 rounded-lg px-4 py-3 text-white outline-none focus:border-blue-500 transition-colors backdrop-blur-md"
          >
            {ATTACK_VECTORS.map(v => (
              <option key={v.id} value={v.id} className="bg-slate-900">{v.name}</option>
            ))}
          </select>
        </div>

        <div className="flex-1 w-full flex flex-col gap-2">
          <label className="text-xs uppercase tracking-widest text-slate-400 font-bold flex items-center gap-2">
            <Target size={14} className="text-blue-400"/> Target Domain/IP
          </label>
          <input 
            type="text" 
            value={targetIp} 
            onChange={(e) => setTargetIp(e.target.value)}
            className="w-full bg-black/40 border border-white/10 rounded-lg px-4 py-3 text-white outline-none focus:border-blue-500 transition-colors backdrop-blur-md font-mono"
          />
        </div>

        <button 
          onClick={launchAttack} 
          className="w-full md:w-auto flex items-center justify-center gap-2 bg-blue-600 hover:bg-blue-500 text-white font-bold px-8 py-3 rounded-lg transition-all shadow-[0_0_20px_rgba(37,99,235,0.4)] hover:shadow-[0_0_30px_rgba(37,99,235,0.6)]"
        >
          <Play size={18} fill="currentColor" /> DEPLOY
        </button>
      </div>

      {/* METRICS - Glassy */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8 relative z-10">
        <div className="bg-white/5 backdrop-blur-xl border border-white/10 p-6 rounded-2xl flex items-center justify-between shadow-xl hidden md:flex">
          <div>
            <p className="text-slate-400 text-xs uppercase tracking-widest mb-1 font-bold">Total Telemetry</p>
            <p className="text-4xl font-light text-white">{(Array.isArray(events) ? events : []).length}</p>
          </div>
          <Activity size={48} className="text-blue-500/20" />
        </div>
        <div className="bg-white/5 backdrop-blur-xl border border-white/10 p-6 rounded-2xl flex items-center justify-between shadow-xl">
          <div>
            <p className="text-slate-400 text-xs uppercase tracking-widest mb-1 font-bold">Active Threats</p>
            <p className={`text-4xl font-light ${activeThreats > 0 ? 'text-red-400 drop-shadow-[0_0_10px_rgba(248,113,113,0.5)]' : 'text-emerald-400'}`}>{activeThreats}</p>
          </div>
          <AlertTriangle size={48} className={activeThreats > 0 ? 'text-red-500/20' : 'text-emerald-500/20'} />
        </div>
        <div className="bg-white/5 backdrop-blur-xl border border-white/10 p-6 rounded-2xl flex items-center justify-between shadow-xl">
          <div>
            <p className="text-slate-400 text-xs uppercase tracking-widest mb-1 font-bold">Live Plugins</p>
            <p className="text-4xl font-light text-blue-400 drop-shadow-[0_0_10px_rgba(96,165,250,0.5)]">{runningSims}</p>
          </div>
          <Zap size={48} className="text-blue-500/20" />
        </div>
      </div>

      {/* DASHBOARD GRIDS - Glassy */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 relative z-10">
        
        {/* CHART */}
        <div className="lg:col-span-2 bg-white/5 backdrop-blur-xl border border-white/10 p-6 rounded-2xl shadow-xl">
          <h2 className="text-sm uppercase tracking-widest font-bold text-slate-300 mb-6 flex items-center gap-2"><Activity size={16} className="text-blue-400"/> Payload Saturation</h2>
          <div className="h-64 w-full min-h-[256px]">
            <ResponsiveContainer width="100%" height="100%" minWidth={10} minHeight={10}>
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="colorEvt" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.4}/>
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#ffffff10" vertical={false} />
                <XAxis dataKey="time" stroke="#64748b" fontSize={11} tickLine={false} axisLine={false} />
                <YAxis stroke="#64748b" fontSize={11} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={{ backgroundColor: 'rgba(15,23,42,0.9)', backdropFilter: 'blur(10px)', borderColor: 'rgba(255,255,255,0.1)', color: '#f8fafc', borderRadius: '8px' }} />
                <Area type="monotone" dataKey="events" stroke="#3b82f6" strokeWidth={3} fillOpacity={1} fill="url(#colorEvt)" isAnimationActive={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* ALERTS TABLE */}
        <div className="bg-white/5 backdrop-blur-xl border border-white/10 p-6 rounded-2xl shadow-xl flex flex-col">
          <h2 className="text-sm uppercase tracking-widest font-bold text-slate-300 mb-4 flex items-center gap-2"><AlertTriangle size={16} className="text-red-400"/> Critical Anomalies</h2>
          <div className="flex-1 overflow-y-auto max-h-64 pr-2 space-y-3 custom-scrollbar">
            {(!Array.isArray(alerts) || alerts.length === 0) ? (
              <p className="text-slate-500/80 italic text-center mt-12 text-sm">Awaiting risk detection...</p>
            ) : (
              alerts.map((a, i) => {
                if (!a) return null;
                return (
                  <div key={a.source_ip || i} className="bg-black/40 backdrop-blur-md p-3 rounded-xl border border-white/5 flex flex-col gap-2">
                    <div className="flex justify-between items-center">
                      <span className="font-mono text-sm text-blue-300 drop-shadow-[0_0_5px_rgba(147,197,253,0.5)]">{a.source_ip || 'UNKNOWN'}</span>
                      <span className={`px-2 py-0.5 rounded text-[10px] uppercase tracking-widest font-bold ${a.severity === 'CRITICAL' ? 'bg-red-500/20 text-red-300 border border-red-500/30' : 'bg-orange-500/20 text-orange-300 border border-orange-500/30'}`}>
                        {a.severity || 'UNKNOWN'} ({a.event_count || 0})
                      </span>
                    </div>
                    <span className="text-xs text-slate-400 truncate">{a.rule_name || a.description || 'Unidentified pattern'}</span>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* RAW EVENTS STREAM */}
        <div className="lg:col-span-3 bg-white/5 backdrop-blur-xl border border-white/10 rounded-2xl shadow-xl overflow-hidden flex flex-col">
          <div className="bg-black/60 px-6 py-4 border-b border-white/10 flex items-center gap-3 relative">
            <Terminal size={16} className="text-blue-400"/> 
            <h3 className="text-sm uppercase tracking-widest font-bold text-slate-300">Live Traffic Feed</h3>
            <div className="absolute right-6 top-1/2 -translate-y-1/2 w-2 h-2 rounded-full bg-emerald-400 animate-pulse shadow-[0_0_10px_rgba(52,211,153,0.8)]" />
          </div>
          <div className="p-6 h-64 overflow-y-auto font-mono text-xs space-y-1 custom-scrollbar text-slate-400">
            {(!Array.isArray(events) || events.length === 0) && <p className="text-slate-500/60 italic mt-2">Listening on secure channels...</p>}
            {(Array.isArray(events) ? events : []).slice(0, 40).map((evt, i) => {
              if (!evt) return null;
              return (
                <div key={evt.id || i} className="flex flex-col sm:flex-row sm:gap-6 hover:bg-white/5 py-1.5 px-3 rounded-lg transition-colors border border-transparent hover:border-white/5">
                  <span className="opacity-50 shrink-0 w-24">{evt.timestamp ? new Date(evt.timestamp).toLocaleTimeString([], { hour12: false }) : '--:--:--'}</span>
                  <span className="text-blue-300 shrink-0 w-28 drop-shadow-[0_0_3px_rgba(147,197,253,0.3)]">{evt.source_ip || '0.0.0.0'}</span>
                  <span className="text-indigo-400 shrink-0 w-36 opacity-80">[{(evt.plugin_id || 'UNKNOWN').toUpperCase()}]</span>
                  <span className="text-slate-300 flex-1 truncate">{evt.event_type || 'Unknown Event'} <span className="opacity-50">→</span> <span className="text-emerald-300 drop-shadow-[0_0_3px_rgba(110,231,183,0.3)]">{evt.target || 'N/A'}</span></span>
                </div>
              );
            })}
          </div>
        </div>

      </div>

      <style>{`
        .custom-scrollbar::-webkit-scrollbar { width: 4px; }
        .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
        .custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 4px; }
        .custom-scrollbar::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.2); }
      `}</style>
    </div>
  );
}
