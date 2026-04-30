require("dotenv").config();
const { GoogleGenerativeAI } = require("@google/genai");

async function run() {
  try {
    const response = await fetch("https://generativelanguage.googleapis.com/v1beta/models?key=" + process.env.GOOGLE_API_KEY);
    const data = await response.json();
    console.log(data.models.map(m => m.name));
  } catch (e) { console.error(e); }
}
run();
