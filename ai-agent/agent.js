require("dotenv").config();
const express = require("express");
const { ChatGoogleGenerativeAI } = require("@langchain/google-genai");
const { PromptTemplate } = require("@langchain/core/prompts");

const app = express();
app.use(express.json());

// Initialize the Gemini LLM Model
const llm = new ChatGoogleGenerativeAI({
  model: "gemini-2.5-flash", 
  temperature: 0.7,
});

// Create the LangChain prompt template
const riskPrompt = PromptTemplate.fromTemplate(`
You are the lead hype-man and autonomous community manager for a decentralized Solana protocol.

You receive raw on-chain data from our custom webhook engine. Your job is to convert this boring JSON data into a highly engaging, viral Discord message celebrating the protocol's activity.

Rules:
- Always mention the specific Action Type.
- Format the Amount so it is readable (e.g., if it's a huge number, format it with commas).
- Include a shortened version of the Transaction Signature (e.g., '4PGeZn...UDuQH') so the community knows it's real.

Use 2-3 emojis. Keep it under 3 sentences. Bring the energy!
`);

// The webhook receiver endpoint
app.post("/webhook", async (req, res) => {
  try {
    const { event_type, user, amount, timestamp } = req.body;
    
    // Quick validation to ensure it's from the Go indexer
    if (!event_type || !user) {
      return res.status(400).json({ error: "Invalid payload format" });
    }

    console.log(`\n======================================================`);
    console.log(`⚡ [AI AGENT] Received On-Chain Event: ${event_type} from ${user}`);
    console.log(`🧠 [AI AGENT] Analyzing Risk Context with LangChain...`);

    // Chain the prompt and the LLM
    const chain = riskPrompt.pipe(llm);
    
    // Execute the chain
    const response = await chain.invoke({
      event_type,
      user,
      amount,
      timestamp,
    });

    console.log(`\n📄 --- ON-CHAIN RISK REPORT ---`);
    console.log(`\x1b[36m${response.content}\x1b[0m`); // Print in cyan
    console.log(`======================================================\n`);

    res.status(200).json({ status: "analyzed", insight: response.content });
  } catch (error) {
    console.error("AI Error:", error.message);
    res.status(500).json({ error: "AI Processing Failed" });
  }
});

const PORT = 4000;
app.listen(PORT, () => {
  console.log(`🤖 AI Agent running on http://localhost:${PORT}`);
  console.log(`Register 'http://127.0.0.1:${PORT}/webhook' in your Colosseum dashboard!`);
});
