import { NextResponse } from "next/server";

export const dynamic = "force-dynamic";

// Store webhooks in a simple in-memory queue for the hackathon demo.
// In a real app, you'd insert these into a database.
const MAX_LOGS = 50;
let webhookLogs: any[] = [];

/**
 * Handle incoming webhook POST requests from the Go dispatcher.
 */
export async function POST(req: Request) {
  try {
    const payload = await req.json();

    // Give it a timestamp if it doesn't have one
    const logEntry = {
      ...payload,
      received_at: new Date().toISOString(),
    };

    // Prepend to array
    webhookLogs.unshift(logEntry);

    // Keep memory bounded
    if (webhookLogs.length > MAX_LOGS) {
      webhookLogs = webhookLogs.slice(0, MAX_LOGS);
    }

    return NextResponse.json({ status: "ok" });
  } catch (error) {
    console.error("Webhook route error:", error);
    return NextResponse.json({ error: "Invalid payload" }, { status: 400 });
  }
}

/**
 * Handle GET requests so the frontend can poll the logs.
 */
export async function GET() {
  return NextResponse.json(webhookLogs);
}
