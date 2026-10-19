import { FormEvent, useEffect, useMemo, useState } from "react";

type PolicyResponse = { policies: string[] };
type Endpoint = { name: string; method: string; path: string; plugin: string };
type PluginResponse = { plugins: string[] };
type Result = {
  SimulationName: string;
  Passed: boolean;
  IsError: boolean;
  Method: string;
  URL: string;
  ExpectedResult: string;
  ActualResult: string;
  Reason: string;
  Duration: number;
};
type Report = {
  TotalSimulations: number;
  Passed: number;
  Failed: number;
  Errors: number;
  ExecutionTime: number;
  SuccessRate: number;
  Results: Result[];
};
type Run = {
  id: string;
  status: "running" | "completed" | "failed";
  target_url: string;
  policy: string;
  started_at: string;
  completed_at?: string;
  error?: string;
  report?: Report;
};

const api = async <T,>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(path, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!response.ok) throw new Error(await response.text());
  return response.json() as Promise<T>;
};

function App() {
  const [policies, setPolicies] = useState<string[]>([]);
  const [plugins, setPlugins] = useState<string[]>([]);
  const [runs, setRuns] = useState<Run[]>([]);
  const [target, setTarget] = useState("http://localhost:8081");
  const [policy, setPolicy] = useState("");
  const [timeout, setTimeoutValue] = useState("15s");
  const [insecure, setInsecure] = useState(false);
  const [selectedRun, setSelectedRun] = useState<Run | null>(null);
  const [editorName, setEditorName] = useState("");
  const [editorContent, setEditorContent] = useState("");
  const [editorMessage, setEditorMessage] = useState("");
  const [editorError, setEditorError] = useState("");
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const refreshRuns = async () => {
    try {
      const response = await api<{ runs: Run[] }>("/api/runs");
      setRuns(response.runs.sort((a, b) => b.started_at.localeCompare(a.started_at)));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to load runs");
    }
  };

  useEffect(() => {
    Promise.all([api<PolicyResponse>("/api/policies"), api<PluginResponse>("/api/plugins")])
      .then(([policyResponse, pluginResponse]) => {
        setPolicies(policyResponse.policies);
        setPlugins(pluginResponse.plugins.sort());
        if (policyResponse.policies.length > 0) setPolicy(policyResponse.policies[0]);
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "API unavailable"));
    void refreshRuns();
  }, []);

  useEffect(() => {
    if (!policy) return;
    const name = policy.split("/").pop() ?? policy;
    setEditorName(name);
    setEditorMessage("");
    setEditorError("");
    api<{ content: string; endpoints: Endpoint[] }>(`/api/policies/${encodeURIComponent(name)}`)
      .then((response) => {
        setEditorContent(response.content);
        setEndpoints(response.endpoints);
      })
      .catch((err: unknown) => setEditorError(err instanceof Error ? err.message : "Unable to load policy"));
  }, [policy]);

  useEffect(() => {
    const timer = window.setInterval(() => void refreshRuns(), 1000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    if (!selectedRun) return;
    const latest = runs.find((run) => run.id === selectedRun.id);
    if (latest) setSelectedRun(latest);
  }, [runs]);

  const latestReport = selectedRun?.report;
  const activeRuns = useMemo(() => runs.filter((run) => run.status === "running").length, [runs]);

  const startRun = async (event: FormEvent) => {
    event.preventDefault();
    setBusy(true);
    setError("");
    try {
      const run = await api<Run>("/api/runs", {
        method: "POST",
        body: JSON.stringify({ target_url: target, policy, timeout, insecure }),
      });
      setSelectedRun(run);
      await refreshRuns();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to start run");
    } finally {
      setBusy(false);
    }
  };

  const savePolicy = async () => {
    setEditorMessage("");
    setEditorError("");
    try {
      await api(`/api/policies/${encodeURIComponent(editorName)}`, {
        method: "PUT",
        body: JSON.stringify({ content: editorContent }),
      });
      setPolicy(`tests/simulations/${editorName}`);
      setEditorMessage("Policy validated and saved.");
      await refreshPolicies();
    } catch (err) {
      setEditorError(err instanceof Error ? err.message : "Unable to save policy");
    }
  };

  const validatePolicy = async () => {
    setEditorMessage("");
    setEditorError("");
    try {
      await api("/api/policies/validate", {
        method: "POST",
        body: JSON.stringify({ content: editorContent }),
      });
      setEditorMessage("Policy is valid.");
    } catch (err) {
      setEditorError(err instanceof Error ? err.message : "Policy validation failed");
    }
  };

  const refreshPolicies = async () => {
    const response = await api<PolicyResponse>("/api/policies");
    setPolicies(response.policies);
  };

  return (
    <main className="shell">
      <header className="topbar">
        <div>
          <p className="eyebrow">SECURITY BEHAVIOR VALIDATION</p>
          <h1>Threat<span>Sim</span> Dashboard</h1>
        </div>
        <div className="api-status"><i /> API connected · {activeRuns} active</div>
      </header>

      <section className="hero">
        <div>
          <p className="eyebrow accent">LOCAL-FIRST SECURITY WORKSPACE</p>
          <h2>Run policies. Understand controls.</h2>
          <p className="muted">Execute existing ThreatSim simulations against a target and inspect every security control from one focused workspace.</p>
        </div>
        <div className="hero-mark">TS<span>://</span></div>
      </section>

      <section className="workspace">
        <form className="panel run-form" onSubmit={startRun}>
          <div className="panel-heading">
            <div><p className="eyebrow">01 / CONFIGURE</p><h3>Start a validation run</h3></div>
            <span className="live-dot">READY</span>
          </div>
          <label>Target URL<input value={target} onChange={(event) => setTarget(event.target.value)} placeholder="http://localhost:8081" /></label>
          <label>Policy file<select value={policy} onChange={(event) => setPolicy(event.target.value)}><option value="" disabled>Select a simulation policy</option>{policies.map((item) => <option key={item} value={item}>{item}</option>)}</select></label>
          <div className="two-col">
            <label>Timeout<input value={timeout} onChange={(event) => setTimeoutValue(event.target.value)} placeholder="15s" /></label>
            <label className="check-label"><input type="checkbox" checked={insecure} onChange={(event) => setInsecure(event.target.checked)} /> Skip TLS verification</label>
          </div>
          <button className="primary" disabled={busy || !policy}>{busy ? "Starting..." : "Run security validation"} <b>→</b></button>
          {error && <p className="error">{error}</p>}
        </form>

        <aside className="panel inventory">
          <div className="panel-heading"><div><p className="eyebrow">02 / INVENTORY</p><h3>Available plugins</h3></div><span className="count">{plugins.length}</span></div>
          <p className="muted small">Security workflows registered in this ThreatSim instance.</p>
          <div className="plugin-list">{plugins.map((plugin) => <span key={plugin}>{plugin.replace("_", " ")}</span>)}</div>
          <div className="tip"><strong>Local execution</strong><br />The API runs policies on this machine. Keep the dashboard bound to localhost unless you add authentication and network controls.</div>
        </aside>
      </section>

      <section className="panel editor-panel">
        <div className="panel-heading">
          <div><p className="eyebrow">03 / POLICY EDITOR</p><h3>Review and validate YAML</h3></div>
          <span className="count">ENDPOINT REVIEW</span>
        </div>
        <div className="editor-toolbar">
          <select value={editorName} onChange={(event) => {
            const next = event.target.value;
            setPolicy(`tests/simulations/${next}`);
          }}>
            <option value="" disabled>Select policy to edit</option>
            {policies.map((item) => <option key={item} value={item.split("/").pop()}>{item}</option>)}
          </select>
          <input value={editorName} onChange={(event) => setEditorName(event.target.value)} placeholder="policy.yaml" />
        </div>
        <textarea className="yaml-editor" value={editorContent} onChange={(event) => setEditorContent(event.target.value)} spellCheck={false} aria-label="ThreatSim policy YAML" />
        {endpoints.length > 0 && <div className="endpoint-list"><p className="eyebrow">DISCOVERED REQUESTS</p>{endpoints.map((endpoint) => <span key={`${endpoint.name}-${endpoint.path}`}><b>{endpoint.method || "PLUGIN"}</b> {endpoint.path || endpoint.plugin} <small>{endpoint.name}</small></span>)}</div>}
        <div className="editor-footer">
          <p className="muted small">Verify every endpoint, method, credential path, and expected response before saving or running.</p>
          <div className="editor-actions"><button className="quiet" onClick={() => void validatePolicy()}>Validate</button><button className="primary save" onClick={() => void savePolicy()}>Save policy →</button></div>
        </div>
        {editorMessage && <p className="success">{editorMessage}</p>}
        {editorError && <p className="error">{editorError}</p>}
      </section>

      <section className="panel runs-panel">
        <div className="panel-heading"><div><p className="eyebrow">04 / RUN HISTORY</p><h3>Validation runs</h3></div><button className="quiet" onClick={() => void refreshRuns()}>Refresh ↻</button></div>
        {runs.length === 0 ? <div className="empty">No runs yet. Configure a target and start your first validation.</div> : <div className="run-list">{runs.map((run) => <button className={`run-row ${selectedRun?.id === run.id ? "selected" : ""}`} key={run.id} onClick={() => setSelectedRun(run)}><span className={`status ${run.status}`}>{run.status}</span><span className="run-policy">{run.policy}</span><span>{run.target_url}</span><span>{run.report ? `${run.report.Passed}/${run.report.TotalSimulations} passed` : "In progress..."}</span><time>{new Date(run.started_at).toLocaleTimeString()}</time></button>)}</div>}
      </section>

      {selectedRun && <section className="panel detail-panel">
        <div className="panel-heading"><div><p className="eyebrow">05 / RUN DETAIL</p><h3>{selectedRun.policy}</h3></div><span className={`status ${selectedRun.status}`}>{selectedRun.status}</span></div>
        {selectedRun.error && <p className="error">{selectedRun.error}</p>}
        {latestReport && <><div className="metrics"><Metric label="Passed" value={latestReport.Passed} tone="good" /><Metric label="Failed" value={latestReport.Failed} tone="bad" /><Metric label="Setup errors" value={latestReport.Errors} tone="warn" /><Metric label="Success rate" value={`${latestReport.SuccessRate.toFixed(1)}%`} /></div><div className="results">{latestReport.Results.map((result, index) => <article className="result" key={`${result.SimulationName}-${index}`}><div className={`result-icon ${result.Passed ? "good" : result.IsError ? "warn" : "bad"}`}>{result.Passed ? "✓" : result.IsError ? "!" : "×"}</div><div><strong>{result.SimulationName}</strong><p>{result.Method} {result.URL}</p><small>{result.Reason || result.ActualResult}</small></div></article>)}</div></>}
        {selectedRun.status === "running" && <div className="loading">Executing simulations<span>•••</span></div>}
      </section>}
    </main>
  );
}

function Metric({ label, value, tone = "" }: { label: string; value: string | number; tone?: string }) {
  return <div className="metric"><span>{label}</span><strong className={tone}>{value}</strong></div>;
}

export default App;
