# Rapida Documentation

Welcome to the Rapida Voice AI documentation. This guide helps you get from "I found this repo" to "I have a working voice agent."

---

## Quick Start

| Step | Time | Guide |
|------|------|-------|
| 1. Check prerequisites | 2 min | [Prerequisites](./getting-started/PREREQUISITES.md) |
| 2. Install and run | 10 min | [Installation](./getting-started/INSTALLATION.md) |
| 3. Create your first agent | 10 min | [First Voice Agent](./getting-started/FIRST_VOICE_AGENT.md) |

---

## For Service Agencies

Deploying Rapida for your clients? Start here:

- [Production Deployment](./deployment/PRODUCTION.md) — Docker production config, scaling, monitoring
- [Licensing](./licensing/README.md) — Open source vs commercial, white-labeling options

**Need to remove Rapida branding?** A commercial license is required. See [Licensing](./licensing/README.md) or contact sales@rapida.ai.

---

## For Contributors

Want to extend the platform? Understand the system first:

- [Architecture Overview](./architecture/OVERVIEW.md) — How services communicate, where code lives
- [Contributing Guide](../CONTRIBUTING.md) — PR process, issue guidelines

**Adding a new provider?** The architecture overview explains the transformer pattern used for STT/TTS/LLM providers.

---

## Product Documentation

For detailed usage guides, SDK reference, and API documentation:

**[doc.rapida.ai](https://doc.rapida.ai)**

Covers:
- Assistants, Tools, Webhooks
- Knowledge Bases
- LLM Endpoints
- Model Providers (OpenAI, Anthropic, Azure, Google, Cohere)
- External Integrations (Twilio, Vonage, Exotel)
- SDKs (React, Node.js, Go, Python)
- API Reference

---

## Directory Structure

```
docs/
├── README.md                    ← You are here
├── getting-started/
│   ├── PREREQUISITES.md         # What you need before starting
│   ├── INSTALLATION.md          # Clone to running platform
│   └── FIRST_VOICE_AGENT.md     # Create and test an agent
├── deployment/
│   └── PRODUCTION.md            # Production deployment guide
├── architecture/
│   └── OVERVIEW.md              # System architecture
├── licensing/
│   └── README.md                # License tiers and options
└── agent-context/               # Research and planning docs
```

---

## Getting Help

- **Issues:** [GitHub Issues](https://github.com/rapidaai/voice-ai/issues)
- **Security:** Report to contact@rapida.ai (not public issues)
- **Commercial inquiries:** sales@rapida.ai

