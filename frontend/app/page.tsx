"use client";

import { useState, useEffect } from "react";
import { triggerAnchorEvent } from "./actions";
import { Loader2, Activity, CheckCircle2, AlertCircle, Zap } from "lucide-react";

// Action type → colour (mirroring Go dispatcher)
const ACTION_COLORS: Record<string, string> = {
  transfer: "border-l-indigo-500 bg-indigo-500/5",
  vote:     "border-l-green-500 bg-green-500/5",
  stake:    "border-l-yellow-500 bg-yellow-500/5",
  unstake:  "border-l-red-500 bg-red-500/5",
  withdraw: "border-l-pink-500 bg-pink-500/5",
};

const ACTION_BADGE: Record<string, string> = {
  transfer: "bg-indigo-500/20 text-indigo-300",
  vote:     "bg-green-500/20 text-green-300",
  stake:    "bg-yellow-500/20 text-yellow-300",
  unstake:  "bg-red-500/20 text-red-300",
  withdraw: "bg-pink-500/20 text-pink-300",
};

const ACTION_EMOJI: Record<string, string> = {
  transfer: "💸", vote: "🗳️", stake: "🔒", unstake: "🔓", withdraw: "📤",
};

export default function Dashboard() {
  const [webhookUrl, setWebhookUrl] = useState("");
  const [eventType, setEventType] = useState("vote");
  const [regStatus, setRegStatus] = useState<"idle" | "loading" | "success" | "error">("idle");
  const [errorMsg, setErrorMsg] = useState("");

  const [triggering, setTriggering] = useState<string | null>(null);
  const [triggerTx, setTriggerTx] = useState<string | null>(null);

  const [logs, setLogs] = useState<any[]>([]);

  const isDiscordUrl = webhookUrl.includes("discord.com/api/webhooks");

  // Poll for logs every second
  useEffect(() => {
    const interval = setInterval(async () => {
      try {
        const res = await fetch("/api/webhook");
        if (res.ok) setLogs(await res.json());
      } catch {}
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setRegStatus("loading");
    try {
      const res = await fetch("http://localhost:8080/register-webhook", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ webhook_url: webhookUrl, event_type: eventType }),
      });
      if (res.ok) {
        setRegStatus("success");
        setTimeout(() => setRegStatus("idle"), 4000);
      } else {
        throw new Error(`Backend returned ${res.status}`);
      }
    } catch (err: any) {
      setRegStatus("error");
      setErrorMsg(err.message);
      setTimeout(() => setRegStatus("idle"), 5000);
    }
  };

  const handleTrigger = async (action: string) => {
    setTriggering(action);
    setTriggerTx(null);
    const result = await triggerAnchorEvent(action);
    setTriggering(null);
    if (result.success) {
      setTriggerTx(result.tx!);
    } else {
      alert(`Error: ${result.error}`);
    }
  };

  return (
    <div className="min-h-screen bg-[#0f1117] text-neutral-100 p-6 md:p-10">
      <div className="max-w-6xl mx-auto space-y-8">

        {/* Header */}
        <header className="flex items-center justify-between border-b border-white/10 pb-6">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-indigo-600 flex items-center justify-center">
              <Zap className="w-5 h-5 text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold tracking-tight">Anchor Hook</h1>
              <p className="text-xs text-neutral-500">On-chain Solana events → Discord / Webhooks</p>
            </div>
          </div>
          <span className="hidden md:flex items-center gap-1.5 text-xs text-neutral-500 bg-neutral-800 px-3 py-1.5 rounded-full border border-neutral-700">
            <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse inline-block" />
            Localnet connected
          </span>
        </header>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">

          {/* ── Left: Controls ────────────────────────────────────── */}
          <div className="space-y-6">

            {/* Step 1: Register */}
            <section className="bg-neutral-800/30 border border-white/10 rounded-2xl p-6 shadow-lg">
              <div className="flex items-center gap-2 mb-5">
                <span className="w-6 h-6 rounded-full bg-indigo-600 text-white text-xs font-bold flex items-center justify-center">1</span>
                <h2 className="text-lg font-semibold">Register a Webhook</h2>
              </div>

              <form onSubmit={handleRegister} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-neutral-400 mb-1.5">
                    Destination URL
                  </label>
                  <div className="relative">
                    <input
                      type="url"
                      value={webhookUrl}
                      onChange={(e) => setWebhookUrl(e.target.value)}
                      placeholder="https://discord.com/api/webhooks/..."
                      className="w-full bg-neutral-950 border border-neutral-700 rounded-lg px-4 py-2.5 pr-24 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono text-sm placeholder:text-neutral-600"
                      required
                    />
                    {isDiscordUrl && (
                      <span className="absolute right-3 top-1/2 -translate-y-1/2 flex items-center gap-1 bg-indigo-500/20 text-indigo-300 text-xs font-bold px-2 py-1 rounded">
                        <svg className="w-3 h-3" viewBox="0 0 24 24" fill="currentColor">
                          <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028 14.09 14.09 0 0 0 1.226-1.994.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03z" />
                        </svg>
                        Discord
                      </span>
                    )}
                  </div>
                  <p className="mt-1.5 text-xs text-neutral-500">
                    {isDiscordUrl
                      ? "✅ Discord URL detected — the Go backend will send a rich embed."
                      : "Paste a Discord Webhook URL for rich embeds, or any HTTPS endpoint for raw JSON."}
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-neutral-400 mb-1.5">Listen for event type</label>
                  <select
                    value={eventType}
                    onChange={(e) => setEventType(e.target.value)}
                    className="w-full bg-neutral-950 border border-neutral-700 rounded-lg px-4 py-2.5 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  >
                    <option value="*">All Events (*)</option>
                    <option value="vote">🗳️  vote</option>
                    <option value="transfer">💸  transfer</option>
                    <option value="stake">🔒  stake</option>
                    <option value="unstake">🔓  unstake</option>
                    <option value="withdraw">📤  withdraw</option>
                  </select>
                </div>

                <button
                  type="submit"
                  disabled={regStatus === "loading"}
                  className="w-full bg-indigo-600 hover:bg-indigo-700 active:scale-95 text-white font-semibold py-2.5 rounded-lg transition-all flex items-center justify-center gap-2 disabled:opacity-50"
                >
                  {regStatus === "loading"
                    ? <Loader2 className="w-5 h-5 animate-spin" />
                    : "Register Endpoint"}
                </button>

                {regStatus === "success" && (
                  <div className="text-green-400 text-sm flex items-center gap-1.5 bg-green-500/10 border border-green-500/20 rounded-lg px-3 py-2">
                    <CheckCircle2 className="w-4 h-4 flex-shrink-0" />
                    Registered! Trigger an event to test the full loop.
                  </div>
                )}
                {regStatus === "error" && (
                  <div className="text-red-400 text-sm flex items-center gap-1.5 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
                    <AlertCircle className="w-4 h-4 flex-shrink-0" /> {errorMsg}
                  </div>
                )}
              </form>
            </section>

            {/* Step 2: Trigger */}
            <section className="bg-neutral-800/30 border border-white/10 rounded-2xl p-6 shadow-lg">
              <div className="flex items-center gap-2 mb-2">
                <span className="w-6 h-6 rounded-full bg-indigo-600 text-white text-xs font-bold flex items-center justify-center">2</span>
                <h2 className="text-lg font-semibold">Trigger On-Chain Event</h2>
              </div>
              <p className="text-sm text-neutral-500 mb-5 ml-8">
                Fires a real Solana transaction on localnet. The Go indexer catches it in &lt;1s and routes it to your registered URL.
              </p>

              <div className="grid grid-cols-2 gap-3">
                {[
                  { action: "vote",     label: "Vote",     emoji: "🗳️" },
                  { action: "transfer", label: "Transfer", emoji: "💸" },
                  { action: "stake",    label: "Stake",    emoji: "🔒" },
                  { action: "withdraw", label: "Withdraw", emoji: "📤" },
                ].map(({ action, label, emoji }) => (
                  <button
                    key={action}
                    onClick={() => handleTrigger(action)}
                    disabled={triggering !== null}
                    className="flex items-center justify-center gap-2 bg-neutral-900 border border-neutral-700 hover:border-indigo-500 hover:bg-neutral-800 active:scale-95 rounded-xl px-4 py-3.5 transition-all disabled:opacity-40 group font-medium"
                  >
                    {triggering === action
                      ? <Loader2 className="w-4 h-4 animate-spin text-indigo-400" />
                      : <span className="text-lg">{emoji}</span>}
                    <span className="text-neutral-200 group-hover:text-white transition-colors">{label}</span>
                  </button>
                ))}
              </div>

              {triggerTx && (
                <div className="mt-4 p-3 bg-indigo-900/20 border border-indigo-500/20 rounded-lg flex items-start gap-2">
                  <CheckCircle2 className="w-4 h-4 text-indigo-400 mt-0.5 flex-shrink-0" />
                  <div className="min-w-0">
                    <p className="text-xs text-indigo-400 font-semibold mb-0.5">Transaction confirmed!</p>
                    <p className="text-xs text-indigo-300/70 font-mono truncate">{triggerTx}</p>
                  </div>
                </div>
              )}
            </section>
          </div>

          {/* ── Right: Live Feed ──────────────────────────────────── */}
          <div className="bg-neutral-950 border border-white/10 rounded-2xl shadow-xl flex flex-col h-[680px] overflow-hidden">
            <div className="bg-neutral-900/60 px-5 py-3.5 border-b border-white/10 flex justify-between items-center">
              <div className="flex items-center gap-2">
                <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
                <h2 className="text-sm font-semibold text-neutral-300 tracking-wide">Live Webhook Feed</h2>
              </div>
              <span className="text-xs text-neutral-500 font-mono bg-neutral-800 px-2 py-0.5 rounded">{logs.length} received</span>
            </div>

            <div className="flex-1 overflow-y-auto p-4 space-y-3">
              {logs.length === 0 ? (
                <div className="h-full flex flex-col items-center justify-center text-neutral-600 gap-3">
                  <Activity className="w-10 h-10 opacity-20" />
                  <div className="text-center">
                    <p className="font-medium">Waiting for events…</p>
                    <p className="text-sm mt-1">Register your Discord URL above, then fire an event!</p>
                  </div>
                </div>
              ) : (
                logs.map((log, i) => {
                  const colorClass = ACTION_COLORS[log.event_type] ?? "border-l-neutral-600 bg-neutral-900/50";
                  const badgeClass = ACTION_BADGE[log.event_type] ?? "bg-neutral-700/50 text-neutral-300";
                  const emoji = ACTION_EMOJI[log.event_type] ?? "⚡";
                  return (
                    <div key={i} className={`border-l-4 rounded-r-xl p-4 ${colorClass}`}>
                      <div className="flex justify-between items-center mb-2">
                        <span className={`text-xs font-bold px-2 py-0.5 rounded-full ${badgeClass}`}>
                          {emoji} {log.event_type}
                        </span>
                        <span className="text-xs text-neutral-500 font-mono">
                          {new Date(log.received_at).toLocaleTimeString([], { hour12: false })}
                        </span>
                      </div>
                      <div className="grid grid-cols-2 gap-x-4 gap-y-1.5 mt-2 text-xs">
                        <div>
                          <p className="text-neutral-500 uppercase tracking-wider text-[10px] font-semibold">Action</p>
                          <p className="text-neutral-200 font-mono">{log.event_type}</p>
                        </div>
                        <div>
                          <p className="text-neutral-500 uppercase tracking-wider text-[10px] font-semibold">Amount</p>
                          <p className="text-neutral-200 font-mono">{log.amount?.toLocaleString()}</p>
                        </div>
                        <div className="col-span-2">
                          <p className="text-neutral-500 uppercase tracking-wider text-[10px] font-semibold">Wallet</p>
                          <p className="text-neutral-400 font-mono truncate">{log.user}</p>
                        </div>
                      </div>
                      <p className="text-[10px] text-neutral-600 mt-3 font-medium">Anchor Hook • Colosseum</p>
                    </div>
                  );
                })
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
