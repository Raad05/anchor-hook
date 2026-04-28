const anchor = require("@coral-xyz/anchor");
const { Connection, Keypair } = require("@solana/web3.js");
const fs = require("fs");
const os = require("os");
const IDL = require("./contracts/target/idl/contracts.json");

async function main() {
  const connection = new Connection("http://127.0.0.1:8899", "confirmed");
  const secretKeyString = fs.readFileSync(os.homedir() + "/.config/solana/id.json", "utf-8");
  const secretKeyArray = Uint8Array.from(JSON.parse(secretKeyString));
  const walletKeypair = Keypair.fromSecretKey(secretKeyArray);
  const wallet = new anchor.Wallet(walletKeypair);
  const provider = new anchor.AnchorProvider(connection, wallet, { preflightCommitment: "confirmed" });
  anchor.setProvider(provider);
  const program = new anchor.Program(IDL, provider);

  console.log("Triggering event...");
  const amount = new anchor.BN(999);
  const tx = await program.methods.triggerAction("vote", amount)
      .accounts({ user: wallet.publicKey })
      .rpc({ commitment: "confirmed", skipPreflight: true });
  console.log("Success! Tx:", tx);
}

main().catch(console.error);
