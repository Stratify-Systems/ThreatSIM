import { useEffect, useState, useRef, useMemo } from "react";
import {
  Shield,
  Activity,
  Terminal,
  Play,
  Wifi,
  WifiOff,
  AlertTriangle,
  Target,
  Zap,
} from "lucide-react";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";

interface Simulation {
  id: string;
  plugin_id: string;
  target: string;
  status: string;
}
interface Event {
  id: string;
  event_type: string;
  source_ip: string;
  target: string;
  timestamp: string;
  plugin_id: string;
}
interface Alert {
  source_ip: string;
  score: number;
  threat_level: string;
  factors: string[];
  updated_at: string;
}

const ATTACK_VECTORS = [
  { id: "brute_force", name: "Brute Force SSH" },
  { id: "port_scan", name: "Port Scan Recon" },
  { id: "ddos", name: "DDoS Flood" },
  { id: "credential_stuffing", name: "Credential Stuffing" },
  { id: "privilege_escalation", name: "Privilege Escalation" },
  { id: "scenario_account_takeover", name: "Scenario: Account Takeover ⛓️" },
];

export default function App() {
  const [events, setEvents] = useState<Event[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [simulations, setSimulations] = useState<Simulation[]>([]);
  const [isConnected, setIsConnected] = useState(false);

  const [selectedAttack, setSelectedAttack] = useState(ATTACK_VECTORS[0].id);
  const [targetIp, setTargetIp] = useState("10.0.0.100");

  const ws = useRef<WebSocket | null>(null);

  const fetchState = async () => {
    try {
      const [eRes, aRes, sRes] = await Promise.all([
        fetch("/api/v1/events").catch(() => null),
        fetch("/api/v1/alerts").catch(() => null),
        fetch("/api/v1/simulations").catch(() => null),
      ]);
      const eData = eRes && eRes.ok ? await eRes.json() : [];
      const aData = aRes && aRes.ok ? await aRes.json() : [];
      const sData = sRes && sRes.ok ? await sRes.json() : [];

      setEvents(Array.isArray(eData) ? eData : []);
      setAlerts(Array.isArray(aData) ? aData : []);
      setSimulations(Array.isArray(sData) ? sData : []);
    } catch (err) {
      console.error("Fetch error", err);
    }
  };

  useEffect(() => {
    let isMounted = true;
    let reconnectTimeout: ReturnType<typeof setTimeout>;

    const connectWs = () => {
      if (!isMounted) return;
      const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
      const socket = new WebSocket(
        `${protocol}//${window.location.host}/ws/live`,
      );
      ws.current = socket;

      socket.onopen = () => {
        if (isMounted) setIsConnected(true);
      };

      socket.onclose = () => {
        if (isMounted) {
          setIsConnected(false);
          reconnectTimeout = setTimeout(connectWs, 3000);
        }
      };

      socket.onerror = () => {
        if (!isMounted) return;
      };

      socket.onmessage = (msg) => {
        if (!isMounted) return;
        try {
          const payload = JSON.parse(msg.data);

          if (payload.type === "event" && payload.data) {
            setEvents((prev) => {
              const safePrev = Array.isArray(prev) ? prev : [];
              return [payload.data, ...safePrev].slice(0, 200);
            });
          } else if (payload.type === "alert" && payload.data) {
            setAlerts((prev) => {
              const safePrev = Array.isArray(prev) ? prev : [];
              return [
                payload.data,
                ...safePrev.filter(
                  (a) => a.source_ip !== payload.data.source_ip,
                ),
              ];
            });
          } else if (
            (payload.type === "simulation_started" ||
              payload.type === "simulation_completed") &&
            payload.data
          ) {
            setSimulations((prev) => {
              const safePrev = Array.isArray(prev) ? prev : [];
              return [
                payload.data,
                ...safePrev.filter((s) => s.id !== payload.data.id),
              ];
            });
          }
        } catch (e) {
          void e;
        }
      };
    };

    fetchState();
    const startupTimeout = setTimeout(connectWs, 50);

    return () => {
      isMounted = false;
      clearTimeout(startupTimeout);
      clearTimeout(reconnectTimeout);
      if (ws.current) {
        ws.current.onclose = null;
        if (
          ws.current.readyState === WebSocket.OPEN ||
          ws.current.readyState === WebSocket.CONNECTING
        ) {
          ws.current.close();
        }
      }
    };
  }, []);

  const launchAttack = async () => {
    if (selectedAttack.startsWith("scenario_")) {
      alert("Launching Multi-Step Scenario Action...");
      try {
        await fetch("/api/v1/simulations", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            plugin_id: "port_scan",
            target: targetIp,
            duration: "10s",
            rate: 10,
          }),
        });
      } catch (err) {
        console.error(err);
      }
    } else {
      try {
        await fetch("/api/v1/simulations", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            plugin_id: selectedAttack,
            target: targetIp,
            duration: "5s",
            rate: 10,
          }),
        });
      } catch (err) {
        console.error(err);
      }
    }
    fetchState();
  };

  const chartData = useMemo(() => {
    const buckets: Record<string, number> = {};
    (Array.isArray(events) ? events : []).forEach((e) => {
      if (!e || !e.timestamp) return;
      const time = new Date(e.timestamp).toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
        second: "2-digit",
      });
      buckets[time] = (buckets[time] || 0) + 1;
    });
    return Object.entries(buckets)
      .reverse()
      .map(([time, count]) => ({ time, events: count }))
      .slice(-20);
  }, [events]);

  const activeThreats = (Array.isArray(alerts) ? alerts : []).filter(
    (a) => a && (a.threat_level === "CRITICAL" || a.threat_level === "HIGH"),
  ).length;
  const runningSims = (Array.isArray(simulations) ? simulations : []).filter(
    (s) => s && s.status === "RUNNING",
  ).length;

  return (
    <div className="min-h-screen bg-black text-slate-200 font-sans p-4 md:p-8 overflow-x-hidden relative selection:bg-indigo-500/30">
      <div className="wrapper max-w-[1400px] mx-auto relative z-10 w-full">

        {/* HEADER */}
        <header className="glass-card mb-8 px-6 py-5 flex flex-col md:flex-row items-center justify-between gap-4">
          <div className="logo cursor-pointer shrink-0">
            <Shield className="logo-icon text-indigo-400" size={32} />
            <h1 className="text-3xl font-bold tracking-tight">Threat<span className="text-indigo-400">SIM</span></h1>
          </div>
          <div className={`px-5 py-2 rounded-full font-medium text-[10px] tracking-[0.2em] border flex items-center gap-2 ${isConnected ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20" : "bg-red-500/10 text-red-400 border-red-500/20"}`}>
            {isConnected ? <Wifi size={14} /> : <WifiOff size={14} />}
            <span>{isConnected ? "NETWORK SECURED & LIVE" : "OFFLINE OR RECONNECTING"}</span>
          </div>
        </header>

        {/* LAUNCH BAR */}
        <div className="glass-card mb-10 p-8">
          <h2 className="text-[11px] uppercase tracking-widest font-semibold text-slate-400 mb-6 flex items-center gap-2">
            <Zap size={16} className="text-indigo-400" /> Vector Configuration & Deployment
          </h2>
          
          <div className="flex flex-col lg:flex-row gap-6 items-end">
            <div className="w-full flex-1 flex flex-col gap-2">
              <label className="text-[10px] uppercase tracking-widest text-slate-500 font-medium px-2">Select Attack Vector</label>
              <div className="glass-input rounded-[18px]">
                <select
                  value={selectedAttack}
                  onChange={(e) => setSelectedAttack(e.target.value)}
                  className="w-full bg-transparent border-none px-5 py-4 text-sm text-slate-200 outline-none appearance-none"
                >
                  {ATTACK_VECTORS.map((v) => (
                    <option key={v.id} value={v.id} className="bg-slate-900">{v.name}</option>
                  ))}
                </select>
              </div>
            </div>

            <div className="w-full flex-1 flex flex-col gap-2">
              <label className="text-[10px] uppercase tracking-widest text-slate-500 font-medium px-2 flex items-center gap-2">Target Origin IP / Host</label>
              <div className="glass-input rounded-[18px]">
                <input
                  type="text"
                  value={targetIp}
                  onChange={(e) => setTargetIp(e.target.value)}
                  placeholder="e.g. 192.168.1.1"
                  className="w-full bg-transparent border-none px-5 py-4 text-sm font-mono text-slate-200 outline-none placeholder:text-slate-600"
                />
              </div>
            </div>

            <button
              onClick={launchAttack}
              className="glass-btn text-white w-full lg:w-auto h-[54px] px-8 rounded-[16px] font-semibold text-sm flex items-center justify-center gap-3 shrink-0"
            >
              <Play size={18} fill="currentColor" /> INITIATE STREAM
            </button>
          </div>
        </div>

        {/* METRICS */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-10">
          <div className="glass-card p-6 flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <p className="text-slate-400 text-[10px] uppercase tracking-widest font-semibold flex items-center gap-2">
                <Activity size={14} className="text-blue-400" /> Ingested Packets
              </p>
            </div>
            <p className="text-5xl font-light text-white tracking-tight">{(Array.isArray(events) ? events : []).length}</p>
          </div>
          
          <div className="glass-card p-6 flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <p className="text-slate-400 text-[10px] uppercase tracking-widest font-semibold flex items-center gap-2">
                <AlertTriangle size={14} className={activeThreats > 0 ? "text-red-400" : "text-emerald-400"} /> Active Warnings
              </p>
              {activeThreats > 0 && <div className="w-2 h-2 rounded-full bg-red-500 animate-pulse" />}
            </div>
            <p className={`text-5xl font-light tracking-tight ${activeThreats > 0 ? "text-red-400 drop-shadow-[0_0_15px_rgba(248,113,113,0.4)]" : "text-emerald-400"}`}>{activeThreats}</p>
          </div>

          <div className="glass-card p-6 flex flex-col gap-4">
            <div className="flex items-center justify-between">
              <p className="text-slate-400 text-[10px] uppercase tracking-widest font-semibold flex items-center gap-2">
                <Zap size={14} className="text-indigo-400" /> Active Subsystems
              </p>
            </div>
            <p className="text-5xl font-light text-white tracking-tight">{runningSims}</p>
          </div>
        </div>

        {/* MAIN DATA VIEW */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          
          {/* CHART */}
          <div className="lg:col-span-2 glass-card p-8 flex flex-col min-h-[350px]">
            <h2 className="text-[10px] uppercase tracking-widest font-semibold text-slate-400 mb-6 flex items-center gap-2">
              <Activity size={14} className="text-indigo-400" /> Saturation Graph
            </h2>
            <div className="flex-1 w-full min-h-[250px]">
              <ResponsiveContainer width="100%" height="100%" minWidth={10} minHeight={10}>
                <AreaChart data={chartData}>
                  <defs>
                    <linearGradient id="colorEvt" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#818cf8" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#818cf8" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" vertical={false} />
                  <XAxis dataKey="time" stroke="rgba(255,255,255,0.4)" fontSize={11} tickLine={false} axisLine={false} />
                  <YAxis stroke="rgba(255,255,255,0.4)" fontSize={11} tickLine={false} axisLine={false} />
                  <Tooltip contentStyle={{ backgroundColor: "rgba(10,10,10,0.85)", backdropFilter: "blur(20px)", borderColor: "rgba(255,255,255,0.1)", color: "#f8fafc", borderRadius: "12px" }} />
                  <Area type="monotone" dataKey="events" stroke="#818cf8" strokeWidth={3} fillOpacity={1} fill="url(#colorEvt)" isAnimationActive={false} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>

          {/* ALERTS */}
          <div className="glass-card p-8 flex flex-col max-h-[400px]">
            <h2 className="text-[10px] uppercase tracking-widest font-semibold text-slate-400 mb-6 flex items-center gap-2">
              <Shield size={14} className="text-red-400" /> Priority Intelligence
            </h2>
            <div className="flex-1 overflow-y-auto space-y-3 custom-scrollbar pr-2">
              {!Array.isArray(alerts) || alerts.length === 0 ? (
                <div className="h-full flex flex-col items-center justify-center text-slate-500/50 italic font-light text-sm p-4 text-center pb-10">
                  <Shield size={32} className="mb-3 opacity-20" />
                  Surveillance optimal. Zero anomalies.
                </div>
              ) : (
                alerts.map((a, i) => {
                  if (!a) return null;
                  return (
                    <div key={a.source_ip || i} className="glass-inner rounded-[16px] p-4 flex flex-col gap-2">
                      <div className="flex justify-between items-center">
                        <span className="font-mono text-[13px] tracking-wide text-slate-200">{a.source_ip || "UNKNOWN"}</span>
                        <span className={`px-2 py-1 rounded-[6px] text-[9px] uppercase tracking-widest font-bold ${a.threat_level === "CRITICAL" ? "bg-red-500/20 text-red-400" : "bg-orange-500/20 text-orange-400"}`}>
                          {a.threat_level || "UNKNOWN"} / {a.score || 0}
                        </span>
                      </div>
                      <span className="text-[11px] text-slate-400 leading-relaxed font-normal">{a.factors ? a.factors.join(" • ") : "Unidentified pattern"}</span>
                    </div>
                  );
                })
              )}
            </div>
          </div>

          {/* RAW STREAM */}
          <div className="lg:col-span-3 glass-card flex flex-col overflow-hidden">
            <div className="px-8 py-5 border-b border-white/[0.06] flex items-center justify-between bg-black/20">
              <div className="flex items-center gap-3">
                <Terminal size={16} className="text-slate-400" />
                <h3 className="text-[10px] uppercase tracking-widest font-semibold text-slate-400">Live Chronological Logs</h3>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-[9px] text-emerald-400 font-mono tracking-widest">LIVE</span>
                <div className="w-2 h-2 rounded-full border border-emerald-400/50">
                   <div className="w-full h-full bg-emerald-400 rounded-full animate-pulse-ring" />
                </div>
              </div>
            </div>
            <div className="p-4 h-[350px] overflow-y-auto font-mono text-[12px] custom-scrollbar bg-black/10">
              {(!Array.isArray(events) || events.length === 0) && (
                <div className="h-full flex items-center justify-center text-slate-500/40 italic font-light">
                  Monitoring incoming traffic anomalies...
                </div>
              )}
              <table className="w-full text-left border-collapse">
                <tbody>
                {(Array.isArray(events) ? events : []).slice(0, 50).map((evt, i) => {
                  if (!evt) return null;
                  return (
                    <tr key={evt.id || i} className="hover:bg-white/[0.03] transition-colors border-b border-white/[0.02]">
                      <td className="py-3 px-4 text-slate-500 w-28 whitespace-nowrap">
                        {evt.timestamp ? new Date(evt.timestamp).toLocaleTimeString([], { hour12: false }) : "--:--:--"}
                      </td>
                      <td className="py-3 px-4 text-slate-300 w-36 whitespace-nowrap">
                        {evt.source_ip || "0.0.0.0"}
                      </td>
                      <td className="py-3 px-4 text-indigo-400/70 w-44 whitespace-nowrap">
                        {evt.plugin_id ? `[${evt.plugin_id.toUpperCase()}]` : "[UNKNOWN]"}
                      </td>
                      <td className="py-3 px-4 text-slate-400">
                        {evt.event_type || "Unknown"}
                      </td>
                      <td className="py-3 px-4 text-emerald-400/80 text-right whitespace-nowrap">
                        {"→ "}{evt.target || "N/A"}
                      </td>
                    </tr>
                  );
                })}
                </tbody>
              </table>
            </div>
          </div>
        </div>

      </div>
    </div>
  );
}
