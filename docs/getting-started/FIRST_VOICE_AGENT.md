# Your First Voice Agent

Create and test a voice agent in about 10 minutes.

**Prerequisites:** Rapida is running locally. See [Installation](./INSTALLATION.md).

---

## Overview

You'll:
1. Create an account
2. Set up a project
3. Add LLM provider credentials
4. Create an assistant
5. Test it in your browser

---

## 1. Open the UI

Navigate to [http://localhost:3000](http://localhost:3000)

You should see a login/signup page.

---

## 2. Create an Account

Click **Sign Up** and create your account:
- Email address
- Password
- Name

After signup, you'll be prompted to create your first organization and project.

---

## 3. Create a Project

Projects contain your assistants, knowledge bases, and configurations.

- **Organization:** Your company or team name
- **Project:** A logical grouping (e.g., "Customer Support", "Sales Demo")

Click through the onboarding flow to create these.

---

## 4. Add LLM Provider Credentials

Before creating an assistant, you need to configure at least one LLM provider.

1. Go to **Settings** (gear icon in sidebar)
2. Navigate to **Model Providers**
3. Click **Add Provider**
4. Select your provider (e.g., OpenAI, Anthropic)
5. Enter your API key
6. Save

Your credentials are encrypted and stored securely.

---

## 5. Create an Assistant

Now create your first voice agent:

1. Go to **Assistants** in the sidebar
2. Click **Create Assistant**
3. Choose **Rapida Assistant** (the default option)

### Step 1: Choose Model

- Select your LLM provider (the one you just configured)
- Choose a model (e.g., `gpt-4o`, `claude-3-sonnet`)
- Adjust parameters if needed:
  - **Temperature:** Lower = more deterministic, Higher = more creative
  - **Max tokens:** Maximum response length

### Step 2: Define Assistant

- **Name:** Give your assistant a memorable name
- **Description:** Optional, for your reference
- **System Prompt:** This defines your agent's personality and behavior

Example system prompt for a customer support agent:

```
You are a helpful customer support agent for Acme Corp.

Your role:
- Answer questions about our products and services
- Help customers troubleshoot issues
- Be friendly, professional, and concise

Guidelines:
- If you don't know something, say so honestly
- For billing issues, offer to connect them with a human agent
- Keep responses under 3 sentences when possible
```

### Step 3: Tools (Optional)

Tools let your agent take actions (call APIs, look up data). Skip this for now—you can add them later.

### Create

Click **Create Assistant**. You'll see a success message.

---

## 6. Configure Voice (Optional)

To enable voice interaction, configure STT and TTS providers:

1. Open your new assistant
2. Go to the **Voice** or **Providers** tab
3. Configure:
   - **Speech-to-Text (STT):** How your agent hears (e.g., Deepgram, OpenAI Whisper)
   - **Text-to-Speech (TTS):** How your agent speaks (e.g., Deepgram, ElevenLabs)

If you don't configure these, you can still test with text chat.

---

## 7. Test Your Agent

### In Browser (Text + Voice)

1. From your assistant page, click **Preview** or **Test**
2. You'll see a chat interface
3. Type a message or click the microphone to speak

The agent will respond based on your system prompt. If you configured TTS, you'll hear the response.

### What You'll See

- **Conversation:** Your messages and agent responses
- **Events:** Real-time debugging info (transcripts, LLM calls, tool usage)
- **Latency:** Time breakdown for each step (STT → LLM → TTS)

---

## 8. View Conversation Logs

After testing:

1. Go to **Activity** in the sidebar
2. Select **Conversation Logs**
3. You'll see all conversations with your agent

Click a conversation to see:
- Full transcript
- Timing breakdown
- Token usage
- Any errors

---

## What's Next?

### Connect to Phone

Integrate with Twilio, Vonage, or Exotel to let users call your agent. See [External Integrations](https://doc.rapida.ai) in the product docs.

### Add Knowledge

Upload documents to give your agent domain-specific knowledge. Go to **Knowledge** in the sidebar.

### Add Tools

Let your agent call APIs, check databases, or take actions. Configure in the **Tools** tab of your assistant.

### Deploy for Users

Embed the agent in your website with the React widget, or integrate via API. See the [SDK documentation](https://doc.rapida.ai/api-reference/installation).

---

## Troubleshooting

### Agent not responding

- Check that your LLM provider credentials are valid
- View logs: `make logs-assistant`
- Ensure the Assistant API is healthy: [http://localhost:9007/readiness/](http://localhost:9007/readiness/)

### Voice not working

- Verify STT/TTS provider credentials
- Check browser microphone permissions
- Try a different browser (Chrome works best)

### Slow responses

- Check LLM latency in conversation logs
- Consider using a faster model (e.g., `gpt-4o-mini`)
- Reduce max tokens if responses are too long

---

## Summary

You now have:
- ✅ A running Rapida instance
- ✅ An LLM provider configured
- ✅ Your first voice agent
- ✅ Tested it in the browser

For production deployment, see [Production Deployment](../deployment/PRODUCTION.md).

