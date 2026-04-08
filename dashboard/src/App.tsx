import { useEffect, useState, useRef } from 'react';

// Core data types mirroring our backend
interface Simulation {
  id: string; plugin_id: string; target: string; status: string;
}
interface Event {
  id: string; event_type: string; source_ip: string; target: string; timestamp: string; plugin_id: string;
}
interface Alert {
  source_ip: string; score: number; threat_level: string; factors: string[]; updated_at: string;
}

function App() {
  const [events, setEvents] = useState<Event[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [simulations, setSimulations] = useState<Simulation[]>([]);
  const [connectionStatus, setConnectionStatus] = useState<string>('Disconnected');
  const ws = useRef<WebSocket | null>(null);

  const fetchState = async () => {
    try {
      const [eRes, aRes, sRes] = await Promise.all([
        fetch('/api/v1/events'),
        fetch('/api/v1/alerts'),
        fetch('/api/v1/simulations')
      ]);
      setEvents(await eRes.json() || []);
      setAlerts(await aRes.json() || []);
      setSimulations(await sRes.json() || []);
    } catch(err) {
      console.error("Failed to fetch initial state:", err);
    }
  };

  useEffect(() => {
    fetchState();

    // Connect to WebSocket using same hostname for Proxy
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    ws.current = new WebSocket(`${protocol}//${window.location.host}/ws/live`);
    
    ws.current.onopen = () => setConnectionStatus('Connected');
    ws.current.onclose = () => setConnectionStatus('Disconnected');

    ws.current.onmessage = (msg) => {
      try {
        const data = JSON.parse(msg.data);
        if (data.type === 'event' || data.event_type) {
           // Append new event to top of stack
           setEvents(prev => [data, ...prev]);
        } else if (data.type === 'alert' || data.threat_level) {
           // Replace the existing alert for this IP with the new updated score
           setAlerts(prev => {
             const clean = prev.filter(a => a.source_ip !== data.source_ip);
             return [data, ...clean];
           });
        }
      } catch (e) {
        console.error('WS Parse Error', e);
      }
    };
    return () => ws.current?.close();
  }, []);

  const launchAttack = async () => {
    await fetch('/api/v1/simulations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ plugin_id: 'brute_force', target: '10.0.0.100', duration: '5s', rate: 5 })
    });
    fetchState(); // Refresh simulation table
  };

  const wipeData = async () => {
     // A handy button if you decide to implement a DELETE HTTP route later!
     alert("Feature coming soon! Use CLI `psql` TRUNCATE for now.");
     setEvents([]); setAlerts([]); setSimulations([]);
  }

  return (
    <div style={{ padding: '2rem', fontFamily: 'system-ui, sans-serif', maxWidth: '1200px', margin: '0 auto', background: '#f9f9f9', minHeight:'100vh' }}>
      <header style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '2px solid #ccc', paddingBottom: '1rem', marginBottom: '2rem' }}>
        <h1>ThreatSIM Control Center</h1>
        <div style={{ padding: '0.5rem 1rem', background: connectionStatus === 'Connected' ? '#d4edda' : '#ffebee', borderRadius: '4px', fontWeight: 'bold' }}>
           Websocket: {connectionStatus}
        </div>
      </header>

      <section style={{ display: 'flex', gap: '1rem', marginBottom: '2rem' }}>
        <button onClick={launchAttack} style={{ padding: '1rem 2rem', fontSize: '1.2rem', background: '#007bff', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer' }}>
          🚀 Launch Brute Force Attack
        </button>
        <button onClick={wipeData} style={{ padding: '1rem 2rem', fontSize: '1.2rem', background: '#dc3545', color: 'white', border: 'none', borderRadius: '4px', cursor: 'pointer' }}>
          🗑️ Clear Dashboard
        </button>
      </section>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '2rem' }}>
        
        {/* SIMULATIONS */}
        <div style={{ background: 'white', padding: '1rem', borderRadius: '8px', boxShadow: '0 2px 4px rgba(0,0,0,0.1)' }}>
          <h2 style={{marginTop:0}}>Active Simulations</h2>
          <pre style={{ overflow: 'auto', background: '#f4f4f4', padding: '1rem', borderRadius: '4px', maxHeight: '400px' }}>
            {JSON.stringify(simulations, null, 2)}
          </pre>
        </div>

        {/* ALERTS */}
        <div style={{ background: 'white', padding: '1rem', borderRadius: '8px', boxShadow: '0 2px 4px rgba(0,0,0,0.1)', border: '2px solid #ff9800' }}>
          <h2 style={{marginTop:0}}>Triggered Alerts</h2>
          <pre style={{ overflow: 'auto', background: '#fff3e0', padding: '1rem', borderRadius: '4px', maxHeight: '400px' }}>
            {JSON.stringify(alerts, null, 2)}
          </pre>
        </div>

        {/* EVENTS */}
        <div style={{ background: 'white', padding: '1rem', borderRadius: '8px', boxShadow: '0 2px 4px rgba(0,0,0,0.1)' }}>
          <h2 style={{marginTop:0}}>Raw Event Stream</h2>
          <pre style={{ overflow: 'auto', background: '#e3f2fd', padding: '1rem', borderRadius: '4px', maxHeight: '400px' }}>
            {JSON.stringify(events.slice(0, 20), null, 2)} 
            // Currently showing only 20 latest events...
          </pre>
        </div>

      </div>
    </div>
  );
}

export default App;
