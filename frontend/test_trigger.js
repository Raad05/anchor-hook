const anchor = require("@coral-xyz/anchor");
const { Connection, Keypair } = require("@solana/web3.js");
const fs = require("fs");
const os = require("os");
const path = require("path");

async function main() {
  const idlPath = path.join(process.cwd(), "..", "contracts", "target", "idl", "contracts.json");
  const IDL = JSON.parse(fs.readFileSync(idlPath, "utf-8"));
  
  const connection = new Connection("http://127.0.0.1:8899", "confirmed");
  const secretKeyString = fs.readFileSync(os.homedir() + "/.config/solana/id.json", "utf-8");
  const walletKeypair = Keypair.fromSecretKey(Uint8Array.from(JSON.parse(secretKeyString)));
  
  const wallet = {
      publicKey: walletKeypair.publicKey,
      signTransaction: async (tx) => { tx.partialSign(walletKeypair); return tx; },
      signAllTransactions: async (txs) => { txs.forEach((tx) => tx.partialSign(walletKeypair)); return txs; },
  };

  const provider = new anchor.AnchorProvider(connection, wallet, { preflightCommitment: "confirmed" });
  anchor.setProvider(provider);
  const program = new anchor.Program(IDL, provider);

  console.log("Triggering event on program:", IDL.address);
  const amount = new anchor.BN(999);
  const tx = await program.methods.triggerAction("vote", amount)
      .accounts({ user: wallet.publicKey })
      .rpc({ commitment: "confirmed", skipPreflight: true });
  console.log("Success! Tx:", tx);
}
main().catch(console.error);
