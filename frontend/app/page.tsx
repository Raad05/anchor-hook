"use client";

import { useState, useEffect } from "react";
import { triggerAnchorEvent } from "./actions";
import { Loader2, Activity, Play, CheckCircle2, AlertCircle } from "lucide-react";

export default function Dashboard() {
  const [webhookUrl, setWebhookUrl] = useState("http://127.0.0.1:3000/api/webhook");
  const [eventType, setEventType] = useState("transfer");
  const [regStatus, setRegStatus] = useState<"idle" | "loading" | "success" | "error">("idle");
  const [errorMsg, setErrorMsg] = useState("");

  const [triggering, setTriggering] = useState<string | null>(null);
  const [triggerTx, setTriggerTx] = useState<string | null>(null);

  const [logs, setLogs] = useState<any[]>([]);

  // Periodically fetch logs
  useEffect(() => {
    const interval = setInterval(async () => {
      try {
        const res = await fetch("/api/webhook");
        if (res.ok) {
          const data = await res.json();
          setLogs(data);
        }
      } catch (err) {
        console.error("Polling error:", err);
      }
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
        setTimeout(() => setRegStatus("idle"), 3000);
      } else {
        throw new Error(`Failed with status ${res.status}`);
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
      alert(`Error triggering event: ${result.error}`);
    }
  };

  return (
    <div className="min-h-screen bg-neutral-900 text-neutral-100 p-8">
      <div className="max-w-6xl mx-auto space-y-8">
        <header className="flex items-center space-x-3 mb-10 border-b border-neutral-800 pb-6">
          <Activity className="w-8 h-8 text-indigo-500" />
          <h1 className="text-3xl font-bold tracking-tight">Anchor Webhook Engine</h1>
        </header>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8">
          {/* Left Column: Controls */}
          <div className="space-y-8">
            
            {/* Box 1: Register */}
            <section className="bg-neutral-800/40 border border-neutral-800 rounded-2xl p-6 shadow-xl backdrop-blur-sm">
              <h2 className="text-xl font-semibold mb-4 text-white">1. Register Webhook</h2>
              <form onSubmit={handleRegister} className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-neutral-400 mb-1">Target Webhook URL</label>
                  <input
                    type="url"
                    value={webhookUrl}
                    onChange={(e) => setWebhookUrl(e.target.value)}
                    className="w-full bg-neutral-950 border border-neutral-700 rounded-lg px-4 py-2.5 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500 font-mono text-sm"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-neutral-400 mb-1">Event Type</label>
                  <select
                    value={eventType}
                    onChange={(e) => setEventType(e.target.value)}
                    className="w-full bg-neutral-950 border border-neutral-700 rounded-lg px-4 py-2.5 text-white focus:outline-none focus:ring-2 focus:ring-indigo-500"
                  >
                    <option value="*">All Events (*)</option>
                    <option value="transfer">transfer</option>
                    <option value="vote">vote</option>
                    <option value="stake">stake</option>
                    <option value="unstake">unstake</option>
                  </select>
                </div>
                <button
                  type="submit"
                  disabled={regStatus === "loading"}
                  className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2.5 rounded-lg transition-colors flex items-center justify-center space-x-2 disabled:opacity-50"
                >
                  {regStatus === "loading" ? <Loader2 className="w-5 h-5 animate-spin" /> : <span>Register Endpoint</span>}
                </button>

                {regStatus === "success" && (
                  <div className="text-green-400 text-sm flex items-center mt-2">
                    <CheckCircle2 className="w-4 h-4 mr-1" /> Registration successful!
                  </div>
                )}
                {regStatus === "error" && (
                  <div className="text-red-400 text-sm flex items-center mt-2">
                    <AlertCircle className="w-4 h-4 mr-1" /> {errorMsg}
                  </div>
                )}
              </form>
            </section>

            {/* Box 2: Trigger */}
            <section className="bg-neutral-800/40 border border-neutral-800 rounded-2xl p-6 shadow-xl backdrop-blur-sm">
              <h2 className="text-xl font-semibold mb-4 text-white">2. Trigger On-Chain Event</h2>
              <p className="text-sm text-neutral-400 mb-6">
                Fires an immediate Solana transaction to your localnet. The backend listener should catch it in &lt;1s.
              </p>
              
              <div className="grid grid-cols-2 gap-4">
                {["transfer", "vote", "stake", "withdraw"].map((action) => (
                  <button
                    key={action}
                    onClick={() => handleTrigger(action)}
                    disabled={triggering !== null}
                    className="flex flex-col items-center justify-center bg-neutral-900 border border-neutral-700 hover:border-indigo-500 hover:bg-neutral-800 rounded-xl p-4 transition-all disabled:opacity-50 group"
                  >
                    {triggering === action ? (
                      <Loader2 className="w-6 h-6 animate-spin text-indigo-400 mb-2" />
                    ) : (
                      <Play className="w-6 h-6 text-neutral-500 group-hover:text-indigo-400 mb-2 transition-colors" />
                    )}
                    <span className="font-semibold capitalize text-neutral-200">{action}</span>
                  </button>
                ))}
              </div>

              {triggerTx && (
                <div className="mt-6 p-3 bg-indigo-900/20 border border-indigo-500/30 rounded-lg">
                  <p className="text-xs text-indigo-300 font-mono truncate">
                    <span className="font-bold">TX:</span> {triggerTx}
                  </p>
                </div>
              )}
            </section>
          </div>

          {/* Right Column: Live Logs */}
          <div className="bg-neutral-950 border border-neutral-800 rounded-2xl shadow-xl flex flex-col h-[700px] overflow-hidden">
            <div className="bg-neutral-900/80 p-4 border-b border-neutral-800 flex justify-between items-center">
              <div className="flex items-center space-x-2">
                <div className="w-2.5 h-2.5 bg-green-500 rounded-full animate-pulse"></div>
                <h2 className="text-sm font-semibold uppercase tracking-wider text-neutral-400">Live Webhook Feed</h2>
              </div>
              <span className="text-xs text-neutral-500 font-mono">Count: {logs.length}</span>
            </div>
            
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {logs.length === 0 ? (
                <div className="h-full flex flex-col items-center justify-center text-neutral-600">
                  <Activity className="w-12 h-12 mb-3 opacity-20" />
                  <p>No webhooks received yet.</p>
                  <p className="text-sm mt-1">Register the callback URL and trigger an event!</p>
                </div>
              ) : (
                logs.map((log, i) => (
                  <div key={i} className="animate-in fade-in slide-in-from-bottom-2 duration-300">
                    <div className="bg-neutral-900 border border-neutral-800 rounded-xl p-4 hover:border-neutral-700 transition-colors">
                      <div className="flex justify-between items-start mb-2">
                        <span className="px-2 py-1 bg-indigo-500/10 text-indigo-400 text-xs font-bold rounded capitalize">
                          {log.event_type}
                        </span>
                        <span className="text-xs text-neutral-500 font-mono">
                          {new Date(log.received_at).toLocaleTimeString([], { hour12: false, fractionalSecondDigits: 3 })}
                        </span>
                      </div>
                      <pre className="text-xs text-neutral-300 font-mono bg-neutral-950 p-3 rounded-lg overflow-x-auto">
{JSON.stringify({
  action: log.event_type,
  user: log.user,
  amount: log.amount,
  timestamp: log.timestamp
}, null, 2)}
                      </pre>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
