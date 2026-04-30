require("dotenv").config();
const { ChatGoogleGenerativeAI } = require("@langchain/google-genai");
const { PromptTemplate } = require("@langchain/core/prompts");

async function run() {
  try {
    const llm = new ChatGoogleGenerativeAI({
      model: "gemini-2.5-flash", 
      temperature: 0.7,
    });
    console.log("Calling LLM...");
    const res = await llm.invoke("Hello world");
    console.log("Response:", res.content);
  } catch(e) {
    console.error("Error:", e);
  }
}
run();
