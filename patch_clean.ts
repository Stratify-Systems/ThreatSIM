import fs from 'fs';
const text = fs.readFileSync('dashboard/src/App.tsx', 'utf-8');

const returnObjStart = text.indexOf('  return (');
if (returnObjStart === -1) {
  console.error("Could not find return statement");
  process.exit(1);
}

const prefix = text.substring(0, returnObjStart);

const newJSX = `  return (
    <div className="min-h-screen bg-[#070709] text-slate-200 font-sans p-4 md:p-8 overflow-x-hidden relative selection:bg-blue-500/30">
      {/* Subtle Background Glows - Deep and Refined */}
      <div className="fixed top-[-10%] left-[-10%] w-[60%] h-[60%] bg-blue-500/5 blur-[150px] pointer-events-none rounded-full" />
      <div className="fixed bottom-[-10%] right-[-10%] w-[60%] h-[60%] bg-emerald-500/5 blur-[150px] pointer-events-none rounded-full" />

      {/* HEADER - Clean Glass */}
      <header className="relative flex justify-between items-center mb-8 bg-white/5 backdrop-blur-3xl border border-white/10 px-8 py-5 rounded-[2rem] shadow-2xl">
        <div className="flex items-center gap-4 text-white">
          <Shield size={28} className="text-blue-400" />
          <h1 className="text-xl font-light tracking-[0.2em] uppercase">
            Threat<span className="font-medium text-blue-400">SIM</span>
          </h1>
        </div>
        <div className={\`flex items-center gap-2 px-5 py-2 rounded-full font-medium text-[10px] tracking-widest border backdrop-blur-md \${isConnected ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20" : "bg-red-500/10 text-red-400 border-red-500/20"}\`}>
          {isConnected ? <Wifi size={14} /> : <WifiOff size={14} />}
          <span>{isConnected ? "NETWORK SECURED" : "OFFLINE UNKNOWN"}</span>
        </div>
      </header>

      {/* LAUNCH BAR - Sleek */}
      <div className="relative mb-8 bg-white/5 backdrop-blur-3xl border border-white/10 rounded-[2rem] shadow-2xl p-8 flex flex-col md:flex-row gap-6 items-center">
        <div className="flex-1 w-full flex flex-col gap-3">
          <label className="text-[10px] uppercase tracking-widest text-slate-400 font-medium flex items-center gap-2 px-1">
            <Zap size={14} className="text-blue-400" /> Vector Configuration
          </label>
          <select
            value={selectedAttack}
            onChange={(e) => setSelectedAttack(e.target.value)}
            className="w-full bg-black/20 border border-white/10 rounded-2xl px-5 py-4 text-sm text-slate-200 outline-none focus:border-blue-500/50 transition-colors backdrop-blur-xl appearance-none hover:bg-white/5"
          >
            {ATTACK_VECTORS.map((v) => (
              <option key={v.id} value={v.id} className="bg-slate-900">{v.name}</option>
            ))}
          </select>
        </div>

        <div className="flex-1 w-full flex flex-col gap-3">
          <label className="text-[10px] uppercase tracking-widest text-slate-400 font-medium flex items-center gap-2 px-1">
            <Target size={14} className="text-blue-400" /> Target Origin
          </label>
          <input
            type="text"
            value={targetIp}
            onChange={(e) => setTargetIp(e.target.value)}
            className="w-full bg-black/20 border border-white/10 rounded-2xl px-5 py-4 text-sm text-slate-200 outline-none focus:border-blue-500/50 transition-colors backdrop-blur-xl font-mono hover:bg-white/5"
          />
        </div>

        <button
          onClick={launchAttack}
          className="w-full md:w-auto self-end flex items-center justify-center gap-3 bg-white text-black hover:bg-slate-200 font-semibold px-10 py-4 lg:ml-4 rounded-2xl transition-all shadow-[0_4px_20px_rgba(255,255,255,0.1)] hover:shadow-[0_8px_30px_rgba(255,255,255,0.2)] active:scale-95 text-sm"
        >
          <Play size={16} fill="currentColor" /> INITIATE STREAM
        </button>
      </div>

      {/* METRICS - Minimal */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8 relative z-10">
        <div className="bg-white/5 backdrop-blur-3xl border border-white/10 rounded-[2rem] p-8 flex items-center justify-between shadow-2xl transition-colors hover:bg-white/10">
          <div>
            <p className="text-slate-400 text-[10px] uppercase tracking-widest mb-3 font-medium">Ingested Packets</p>
            <p className="text-4xl font-light text-white">{(Array.isArray(events) ? events : []).length}</p>
          </div>
          <div className="bg-blue-500/10 p-4 rounded-full border border-blue-500/20"><Activity size={24} className="text-blue-400" /></div>
        </div>
        <div className="bg-white/5 backdrop-blur-3xl border border-white/10 rounded-[2rem] p-8 flex items-center justify-between shadow-2xl transition-colors hover:bg-white/10">
          <div>
            <p className="text-slate-400 text-[10px] uppercase tracking-widest mb-3 font-medium">Active Warnings</p>
            <p className={\`text-4xl font-light \${activeThreats > 0 ? "text-red-400" : "text-emerald-400"}\`}>{activeThreats}</p>
          </div>
          <div className={activeThreats > 0 ? "bg-red-500/10 p-4 rounded-full border border-red-500/20" : "bg-emerald-500/10 p-4 rounded-full border border-emerald-500/20"}>
            <AlertTriangle size={24} className={activeThreats > 0 ? "text-red-400" : "text-emerald-400"} />
          </div>
        </div>
        <div className="bg-white/5 backdrop-blur-3xl border border-white/10 rounded-[2rem] p-8 flex items-center justify-between shadow-2xl transition-colors hover:bg-white/10">
          <div>
            <p className="text-slate-400 text-[10px] uppercase tracking-widest mb-3 font-medium">Active Subsystems</p>
            <p className="text-4xl font-light text-white">{runningSims}</p>
          </div>
          <div className="bg-indigo-500/10 p-4 rounded-full border border-indigo-500/20"><Zap size={24} className="text-indigo-400" /></div>
        </div>
      </div>

      {/* MAIN GRIDS */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 relative z-10">
        {/* CHART */}
        <div className="lg:col-span-2 bg-white/5 backdrop-blur-3xl border border-white/10 p-8 rounded-[2rem] shadow-2xl">
          <h2 className="text-[10px] uppercase tracking-widest font-medium text-slate-400 mb-8 flex items-center gap-2">
            <Activity size={14} className="text-blue-400" /> Network Saturation Graph
          </h2>
          <div className="h-64 w-full min-h-[256px]">
            <ResponsiveContainer width="100%" height="100%" minWidth={10} minHeight={10}>
              <AreaChart data={chartData}>
                <defs>
                  <linearGradient id="colorEvt" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#ffffff" stopOpacity={0.15} />
                    <stop offset="95%" stopColor="#ffffff" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#ffffff10" vertical={false} />
                <XAxis dataKey="time" stroke="#64748b" fontSize={11} tickLine={false} axisLine={false} />
                <YAxis stroke="#64748b" fontSize={11} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={{ backgroundColor: "rgba(10,10,10,0.8)", backdropFilter: "blur(20px)", borderColor: "rgba(255,255,255,0.1)", color: "#f8fafc", borderRadius: "16px" }} />
                <Area type="monotone" dataKey="events" stroke="#ffffff" strokeWidth={2} fillOpacity={1} fill="url(#colorEvt)" isAnimationActive={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* ALERTS TABLE */}
        <div className="bg-white/5 backdrop-blur-3xl border border-white/10 p-8 rounded-[2rem] shadow-2xl flex flex-col">
          <h2 className="text-[10px] uppercase tracking-widest font-medium text-slate-400 mb-6 flex items-center gap-2">
            <AlertTriangle size={14} className="text-red-400" /> Priority Intelligence
          </h2>
          <div className="flex-1 overflow-y-auto max-h-64 pr-2 space-y-4 custom-scrollbar">
            {!Array.isArray(alerts) || alerts.length === 0 ? (
              <p className="text-slate-500/70 italic text-center mt-12 text-sm font-light">Surveillance optimal. Zero anomalies.</p>
            ) : (
              alerts.map((a, i) => {
                if (!a) return null;
                return (
                  <div key={a.source_ip || i} className="bg-white/5 p-5 rounded-2xl border border-white/5 flex flex-col gap-3 transition-colors hover:bg-white/10">
                    <div className="flex justify-between items-center">
                      <span className="font-mono text-sm tracking-wide text-slate-200">{a.source_ip || "UNKNOWN"}</span>
                      <span className={\`px-3 py-1 rounded-full text-[9px] uppercase tracking-widest font-bold \${a.threat_level === "CRITICAL" ? "bg-red-500/20 text-red-400 border border-red-500/20" : "bg-orange-500/20 text-orange-400 border border-orange-500/20"}\`}>
                        {a.threat_level || "UNKNOWN"} : {a.score || 0}
                      </span>
                    </div>
                    <span className="text-xs text-slate-400 leading-relaxed font-light">{a.factors ? a.factors.join(" • ") : "Unidentified pattern"}</span>
                  </div>
                );
              })
            )}
          </div>
        </div>

        {/* RAW EVENTS STREAM */}
        <div className="lg:col-span-3 bg-white/5 backdrop-blur-3xl border border-white/10 rounded-[2rem] shadow-2xl overflow-hidden flex flex-col mt-4">
          <div className="px-8 py-6 border-b border-white/10 flex items-center justify-between bg-white/[0.02]">
            <div className="flex items-center gap-3">
              <Terminal size={14} className="text-slate-400" />
              <h3 className="text-[10px] uppercase tracking-widest font-medium text-slate-400">Chronological Event Logs</h3>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[9px] text-emerald-500 font-mono tracking-widest">LIVE</span>
              <div className="w-2 h-2 rounded-full bg-emerald-500 shadow-[0_0_10px_rgba(16,185,129,0.8)] animate-pulse" />
            </div>
          </div>
          <div className="p-8 h-[22rem] overflow-y-auto font-mono text-[11px] space-y-2 custom-scrollbar text-slate-400">
            {(!Array.isArray(events) || events.length === 0) && (
              <p className="text-slate-500/50 italic mt-2 font-light">Monitoring traffic anomalies...</p>
            )}
            {(Array.isArray(events) ? events : []).slice(0, 40).map((evt, i) => {
              if (!evt) return null;
              return (
                <div key={evt.id || i} className="flex flex-col sm:flex-row sm:gap-6 hover:bg-white/5 py-3 px-5 rounded-xl transition-colors border border-transparent hover:border-white/5">
                  <span className="text-slate-500 shrink-0 w-20">{evt.timestamp ? new Date(evt.timestamp).toLocaleTimeString([], { hour12: false }) : "--:--:--"}</span>
                  <span className="text-slate-200 shrink-0 w-32">{evt.source_ip || "0.0.0.0"}</span>
                  <span className="text-indigo-400/80 shrink-0 w-40 opacity-80">::{(evt.plugin_id || "UNKNOWN").toUpperCase()}</span>
                  <span className="text-slate-400 flex-1 truncate font-light">
                    {evt.event_type || "Unknown Event"} <span className="text-slate-600 mx-2">→</span> <span className="text-emerald-400/80">{evt.target || "N/A"}</span>
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      <style>{\`
        .custom-scrollbar::-webkit-scrollbar { width: 4px; }
        .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
        .custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 10px; }
        .custom-scrollbar::-webkit-scrollbar-thumb:hover { background: rgba(255,255,255,0.2); }
      \`}</style>
    </div>
  );
}
`;

fs.writeFileSync('dashboard/src/App.tsx', prefix + newJSX);
