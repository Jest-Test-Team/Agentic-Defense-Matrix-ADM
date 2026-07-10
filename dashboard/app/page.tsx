"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  api,
  getConfig,
  setConfig,
  type ApiConfig,
  type Stats,
  type SessionRow,
  type BattleEvent,
} from "@/lib/api";

const pct = (x: number) => `${(x * 100).toFixed(0)}%`;

type Health = {
  analysis: boolean | null;
  neon: boolean | null;
  gateway: boolean | null;
};

export default function Page() {
  const [cfg, setCfg] = useState<ApiConfig | null>(null);
  const [stats, setStats] = useState<Stats | null>(null);
  const [sessions, setSessions] = useState<SessionRow[]>([]);
  const [health, setHealth] = useState<Health>({ analysis: null, neon: null, gateway: null });
  const [events, setEvents] = useState<BattleEvent[]>([]);
  const [connected, setConnected] = useState<boolean | null>(null);
  const [mixedContent, setMixedContent] = useState(false);
  const esRef = useRef<EventSource | null>(null);

  useEffect(() => {
    setCfg(getConfig());
  }, []);

  // Detect the HTTPS-page -> HTTP-api mixed-content situation up front.
  useEffect(() => {
    if (!cfg) return;
    if (typeof window !== "undefined" && window.location.protocol === "https:" && cfg.analysis.startsWith("http:")) {
      setMixedContent(true);
    }
  }, [cfg]);

  const refresh = useCallback(async () => {
    if (!cfg) return;
    try {
      const s = await api.stats(cfg);
      setStats(s);
      setConnected(true);
    } catch {
      setConnected(false);
    }
    try {
      const t = await api.timeline(cfg, 40);
      setSessions(t.sessions ?? []);
    } catch {}
  }, [cfg]);

  const refreshHealth = useCallback(async () => {
    if (!cfg) return;
    const [analysis, neon, gateway] = await Promise.all([
      api.analysisHealth(cfg),
      api.analysisReady(cfg),
      api.gatewayHealth(cfg),
    ]);
    setHealth({ analysis, neon, gateway });
  }, [cfg]);

  useEffect(() => {
    if (!cfg) return;
    refresh();
    refreshHealth();
    const a = setInterval(refresh, 3000);
    const b = setInterval(refreshHealth, 8000);
    return () => {
      clearInterval(a);
      clearInterval(b);
    };
  }, [cfg, refresh, refreshHealth]);

  // Live event stream (SSE).
  useEffect(() => {
    if (!cfg) return;
    try {
      const es = new EventSource(`${cfg.analysis}/api/stream`);
      esRef.current = es;
      es.onmessage = (e) => {
        try {
          const ev = JSON.parse(e.data) as BattleEvent;
          setEvents((prev) => [ev, ...prev].slice(0, 120));
        } catch {}
      };
      es.onerror = () => {}; // browser auto-reconnects; polling covers the data
      return () => es.close();
    } catch {
      return;
    }
  }, [cfg]);

  return (
    <>
      <header className="top">
        <div className="top-inner">
          <h1 className="brand">
            ⚔️ ADM Battle Console
            <span className="sub">Red attacks · Blue defends · Green remediates</span>
          </h1>
          <div className="conn">
            <span
              className={`dot ${connected === true ? "live" : connected === false ? "down" : ""}`}
            />
            {connected === true ? "live" : connected === false ? "unreachable" : "connecting…"}
          </div>
        </div>
      </header>

      <div className="wrap">
        {mixedContent && (
          <div className="banner">
            <strong>Live data blocked by the browser (mixed content).</strong> This page is served
            over HTTPS but the API is <code>{cfg?.analysis}</code> (HTTP). Put the API behind HTTPS
            (a domain + TLS, or a Cloudflare Tunnel) and point this dashboard at it with{" "}
            <code>?api=https://your-host</code>, or open the API host directly.
          </div>
        )}

        <h2 className="section">System status</h2>
        <div className="status-grid">
          <StatusCard label="Analysis engine" ok={health.analysis} okText="ok" />
          <StatusCard label="Neon Postgres" ok={health.neon} okText="ready" />
          <StatusCard label="Gateway (target)" ok={health.gateway} okText="up" />
          <StatusCard
            label="Elasticsearch (Bonsai)"
            ok={stats ? stats.elastic_enabled : null}
            okText="indexing"
            offText="postgres-only"
            neutralOff
          />
        </div>

        <h2 className="section">Battle scoreboard</h2>
        <div className="tiles">
          <Tile k="Attacks" v={stats ? String(stats.attacks) : "–"} cls="red" />
          <Tile k="Block rate" v={stats ? pct(stats.block_rate) : "–"} cls="blue" />
          <Tile k="Detection rate" v={stats ? pct(stats.detection_rate) : "–"} cls="blue" />
          <Tile k="Landed" v={stats ? String(stats.landed) : "–"} cls="red" />
          <Tile k="Remediations" v={stats ? String(stats.remediations) : "–"} cls="good" />
          <Tile
            k="MTTR"
            v={stats ? (stats.mttr_seconds == null ? "–" : `${stats.mttr_seconds.toFixed(1)}s`) : "–"}
            cls="good"
          />
          <Tile k="Residual risk" v={stats ? String(stats.residual_risk) : "–"} cls="warn" />
        </div>

        <div className="grid2" style={{ marginTop: 20 }}>
          <div>
            <h2 className="section">Live battle feed</h2>
            <div className="panel tall">
              {events.length === 0 && (
                <div className="feed-row muted">
                  {connected === false
                    ? "No connection to the event stream."
                    : "Waiting for events…"}
                </div>
              )}
              {events.map((ev, i) => (
                <EventRow key={ev.id ?? i} ev={ev} />
              ))}
            </div>
          </div>
          <div>
            <h2 className="section">By technique — blocked ▏landed</h2>
            <div className="panel tall">
              <div className="legend">
                <span>
                  <span className="sw" style={{ background: "var(--blue)" }} />
                  blocked (blue)
                </span>
                <span>
                  <span className="sw" style={{ background: "var(--red)" }} />
                  landed (red)
                </span>
              </div>
              {(stats?.by_technique ?? []).map((t) => (
                <TechRow key={t.technique} name={t.technique} blocked={t.blocked} landed={t.landed} />
              ))}
              {!stats && <div className="feed-row muted">Loading…</div>}
            </div>
          </div>
        </div>

        <h2 className="section">Recent sessions (attack → remediation)</h2>
        <div className="panel">
          <div className="feed-row muted" style={{ fontWeight: 600 }}>
            <span style={{ width: 90 }}>technique</span>
            <span style={{ width: 90 }}>target</span>
            <span>attack</span>
            <span className="out">remediation / MTTR</span>
          </div>
          {sessions.slice(0, 20).map((s) => (
            <div className="feed-row" key={s.session_id}>
              <span className="tech" style={{ width: 90 }}>{s.technique}</span>
              <span className="muted" style={{ width: 90 }}>{s.target || "—"}</span>
              <span className={`out ${s.attack_outcome}`}>{s.attack_outcome}</span>
              <span className="out">
                {s.remediation_outcome
                  ? `${s.remediation_outcome}${s.mttr_seconds != null ? ` · ${s.mttr_seconds.toFixed(1)}s` : ""}`
                  : "—"}
              </span>
            </div>
          ))}
          {sessions.length === 0 && <div className="feed-row muted">No sessions yet.</div>}
        </div>

        <Settings cfg={cfg} />

        <div className="foot-note">
          Polls <code>/api/stats</code> + <code>/api/timeline</code> every few seconds and streams{" "}
          <code>/api/stream</code> (SSE). Durable log in Neon Postgres; search/aggregation in
          Elasticsearch when enabled.
        </div>
      </div>
    </>
  );
}

function StatusCard({
  label,
  ok,
  okText,
  offText = "down",
  neutralOff = false,
}: {
  label: string;
  ok: boolean | null;
  okText: string;
  offText?: string;
  neutralOff?: boolean;
}) {
  const cls = ok === true ? "good" : ok === false ? (neutralOff ? "warn" : "crit") : "warn";
  const icon = ok === true ? "✓" : ok === false ? (neutralOff ? "○" : "✕") : "…";
  const val = ok === true ? okText : ok === false ? offText : "checking…";
  return (
    <div className="status-card">
      <div className={`pill ${cls}`}>{icon}</div>
      <div>
        <div className="label">{label}</div>
        <div className="val">{val}</div>
      </div>
    </div>
  );
}

function Tile({ k, v, cls }: { k: string; v: string; cls?: string }) {
  return (
    <div className="tile">
      <div className="k">{k}</div>
      <div className={`v ${cls ?? ""}`}>{v}</div>
    </div>
  );
}

function TechRow({ name, blocked, landed }: { name: string; blocked: number; landed: number }) {
  const total = Math.max(1, blocked + landed);
  return (
    <div className="tech-row">
      <span className="name">{name}</span>
      <span className="bar">
        <span className="blocked" style={{ width: `${(blocked / total) * 100}%` }} />
        <span className="landed" style={{ width: `${(landed / total) * 100}%` }} />
      </span>
      <span className="cnt">
        <b>{blocked}</b> ▏ {landed}
      </span>
    </div>
  );
}

function EventRow({ ev }: { ev: BattleEvent }) {
  const team = (ev.team || "?").toLowerCase();
  const tag = team === "red" ? "red" : team === "green" ? "green" : "blue";
  return (
    <div className="feed-row">
      <span className={`tag ${tag}`}>{(ev.team || "?").toUpperCase()}</span>
      <span className="tech">{ev.technique || ev.kind || ""}</span>
      <span className="detail">{ev.detail || ""}</span>
      <span className={`out ${ev.outcome || ""}`}>{ev.outcome || ""}</span>
    </div>
  );
}

function Settings({ cfg }: { cfg: ApiConfig | null }) {
  const [analysis, setAnalysis] = useState("");
  const [gateway, setGateway] = useState("");
  useEffect(() => {
    if (cfg) {
      setAnalysis(cfg.analysis);
      setGateway(cfg.gateway);
    }
  }, [cfg]);
  return (
    <>
      <h2 className="section">Endpoint</h2>
      <div className="settings">
        <input value={analysis} onChange={(e) => setAnalysis(e.target.value)} placeholder="analysis API base URL" />
        <input value={gateway} onChange={(e) => setGateway(e.target.value)} placeholder="gateway base URL" />
        <button
          onClick={() => {
            setConfig(analysis.trim(), gateway.trim());
            window.location.search = "";
          }}
        >
          Save & reload
        </button>
        <span className="muted">or use <code>?api=…&amp;gw=…</code> in the URL</span>
      </div>
    </>
  );
}
