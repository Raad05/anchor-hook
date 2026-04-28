"use server";

import * as anchor from "@coral-xyz/anchor";
import { Connection, Keypair } from "@solana/web3.js";
import fs from "fs";
import os from "os";
import path from "path";

export async function triggerAnchorEvent(actionType: string) {
  try {
    // 1. Read the IDL raw at runtime to prevent Next.js Webpack from restricting out-of-root imports.
    const idlPath = path.join(process.cwd(), "..", "contracts", "target", "idl", "contracts.json");
    const idlRaw = fs.readFileSync(idlPath, "utf-8");
    const IDL = JSON.parse(idlRaw);

    // 2. Setup connection & wallet layer
    const connection = new Connection("http://127.0.0.1:8899", "confirmed");
    const keypairPath = `${os.homedir()}/.config/solana/id.json`;
    const secretKeyString = fs.readFileSync(keypairPath, "utf-8");
    const secretKeyArray = Uint8Array.from(JSON.parse(secretKeyString));
    const walletKeypair = Keypair.fromSecretKey(secretKeyArray);

    // Provide the Wallet interface directly so we don't rely on the Anchor Wallet class
    // which has ESM export resolution issues in Next.js Server Actions.
    const wallet = {
      publicKey: walletKeypair.publicKey,
      signTransaction: async (tx: any) => {
        tx.partialSign(walletKeypair);
        return tx;
      },
      signAllTransactions: async (txs: any[]) => {
        txs.forEach((tx) => tx.partialSign(walletKeypair));
        return txs;
      },
    };

    // 3. Setup Provider
    const provider = new anchor.AnchorProvider(connection, wallet, {
      preflightCommitment: "confirmed",
    });
    anchor.setProvider(provider);

    // 4. Initialize Program
    const programId = new anchor.web3.PublicKey(IDL.address);
    const program = new anchor.Program(IDL, provider);

    // 5. Generate random amount
    const amount = new anchor.BN(Math.floor(Math.random() * 1_000_000_000) + 1);

    // 6. Fire transaction!
    const tx = await program.methods
      .triggerAction(actionType, amount)
      .accounts({ user: wallet.publicKey })
      .rpc({ commitment: "confirmed", skipPreflight: true });

    console.log(`[actions] Triggered ${actionType} event on-chain! Tx: ${tx}`);

    return { success: true, tx, actionType, amount: amount.toString() };
  } catch (error: any) {
    console.error("Action error:", error);
    return { success: false, error: error.message };
  }
}

