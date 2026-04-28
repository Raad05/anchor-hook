import * as anchor from "@coral-xyz/anchor";
import { Program } from "@coral-xyz/anchor";
import { Contracts } from "../target/types/contracts";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const ACTION_TYPES = ["transfer", "stake", "vote", "unstake", "withdraw"];

function randomActionType(): string {
  return ACTION_TYPES[Math.floor(Math.random() * ACTION_TYPES.length)];
}

function randomAmount(): anchor.BN {
  // Random amount between 1 and 1_000_000_000 (lamports)
  return new anchor.BN(Math.floor(Math.random() * 1_000_000_000) + 1);
}

// ---------------------------------------------------------------------------
// Test suite
// ---------------------------------------------------------------------------

describe("contracts – UserAction event emitter", () => {
  // Use the local validator configured via ANCHOR_PROVIDER_URL or Anchor.toml.
  const provider = anchor.AnchorProvider.env();
  anchor.setProvider(provider);

  const program = anchor.workspace.contracts as Program<Contracts>;
  const user = provider.wallet;

  // ------------------------------------------------------------------
  // Smoke test: single invocation
  // ------------------------------------------------------------------
  it("emits a UserAction event on trigger_action", async () => {
    const actionType = "transfer";
    const amount = new anchor.BN(500_000_000);

    const tx = await program.methods
      .triggerAction(actionType, amount)
      .accounts({ user: user.publicKey })
      .rpc();

    console.log(`[smoke] tx: ${tx}`);
  });

  // ------------------------------------------------------------------
  // Stress test: spam N events to generate a rich log stream for the
  // Go listener to consume.
  // ------------------------------------------------------------------
  it("spams 10 random UserAction events", async () => {
    const ITERATIONS = 10;

    for (let i = 0; i < ITERATIONS; i++) {
      const actionType = randomActionType();
      const amount = randomAmount();

      const tx = await program.methods
        .triggerAction(actionType, amount)
        .accounts({ user: user.publicKey })
        .rpc();

      console.log(
        `[spam][${i + 1}/${ITERATIONS}] action=${actionType} amount=${amount.toString()} tx=${tx}`
      );
    }
  });

  // ------------------------------------------------------------------
  // Event listener test: verify the emitted data can be decoded via the
  // Anchor client (mirrors what the Go backend must replicate natively).
  // ------------------------------------------------------------------
  it("decodes UserAction event from transaction logs", async () => {
    const expectedAction = "vote";
    const expectedAmount = new anchor.BN(42_000_000);

    // Subscribe to the event before triggering.
    let receivedEvent: {
      user: anchor.web3.PublicKey;
      actionType: string;
      amount: anchor.BN;
    } | null = null;

    const listener = program.addEventListener(
      "userAction",
      (event: {
        user: anchor.web3.PublicKey;
        actionType: string;
        amount: anchor.BN;
      }) => {
        receivedEvent = event;
      }
    );

    try {
      const tx = await program.methods
        .triggerAction(expectedAction, expectedAmount)
        .accounts({ user: user.publicKey })
        .rpc({ commitment: "confirmed", skipPreflight: true });

      console.log(`[event-listener] tx: ${tx}`);

      // Give the validator a moment to propagate the confirmed log.
      await new Promise((resolve) => setTimeout(resolve, 1_500));

      if (receivedEvent === null) {
        throw new Error("No UserAction event received via Anchor listener");
      }

      const ev = receivedEvent as {
        user: anchor.web3.PublicKey;
        actionType: string;
        amount: anchor.BN;
      };
      console.log(
        `[event-listener] decoded: user=${ev.user.toBase58()} action=${ev.actionType} amount=${ev.amount.toString()}`
      );

      // Assertions
      if (ev.actionType !== expectedAction) {
        throw new Error(
          `Expected action_type="${expectedAction}" but got "${ev.actionType}"`
        );
      }
      if (!ev.amount.eq(expectedAmount)) {
        throw new Error(
          `Expected amount=${expectedAmount.toString()} but got ${ev.amount.toString()}`
        );
      }
      if (!ev.user.equals(user.publicKey)) {
        throw new Error(
          `Expected user=${user.publicKey.toBase58()} but got ${ev.user.toBase58()}`
        );
      }

      console.log("[event-listener] ✅ all assertions passed");
    } finally {
      await program.removeEventListener(listener);
    }
  });
});
