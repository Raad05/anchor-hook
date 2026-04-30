require("dotenv").config();
const { ChatGoogleGenerativeAI } = require("@langchain/google-genai");
const { PromptTemplate } = require("@langchain/core/prompts");

async function run() {
  try {
    const llm = new ChatGoogleGenerativeAI({
      model: "gemini-2.5-flash", 
      temperature: 0.7,
    });
    const riskPrompt = PromptTemplate.fromTemplate(`Amount: {amount}`);
    const chain = riskPrompt.pipe(llm);
    console.log("Calling chain...");
    const res = await chain.invoke({ amount: 999 });
    console.log("Response:", res.content);
  } catch(e) {
    console.error("Error:", e);
  }
}
run();
