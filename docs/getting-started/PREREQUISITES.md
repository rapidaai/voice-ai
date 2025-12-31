# Prerequisites

Before installing Rapida, ensure you have the following.

---

## Required

### Docker & Docker Compose

Rapida runs as a set of containerized services. You need Docker with Compose support.

- **macOS/Windows:** [Docker Desktop](https://www.docker.com/products/docker-desktop/) (includes Compose)
- **Linux:** [Docker Engine](https://docs.docker.com/engine/install/) + [Compose Plugin](https://docs.docker.com/compose/install/linux/)

Verify installation:

```bash
docker --version        # Docker version 24.0+ recommended
docker compose version  # Docker Compose version v2.0+
```

### Git

```bash
git --version  # Any recent version
```

### System Resources

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| RAM | 8 GB | 16 GB |
| Disk | 10 GB free | 20 GB free |
| CPU | 4 cores | 8 cores |

> **Note:** OpenSearch requires significant memory. If you see OpenSearch failing to start, increase Docker's memory allocation in Docker Desktop settings.

---

## Required API Keys

You need at least one LLM provider to power your voice agents.

### LLM Provider (choose one)

| Provider | Get API Key |
|----------|-------------|
| OpenAI | [platform.openai.com/api-keys](https://platform.openai.com/api-keys) |
| Anthropic | [console.anthropic.com](https://console.anthropic.com/) |
| Azure OpenAI | [Azure Portal](https://portal.azure.com/) |
| Google AI | [aistudio.google.com](https://aistudio.google.com/) |

You'll add this key in the Rapida UI after installation.

### Speech Providers (optional, for voice)

For voice agents, you need STT (Speech-to-Text) and TTS (Text-to-Speech) providers:

| Provider | Services | Get API Key |
|----------|----------|-------------|
| Deepgram | STT, TTS | [console.deepgram.com](https://console.deepgram.com/) |
| OpenAI | STT (Whisper), TTS | Same as LLM key |
| ElevenLabs | TTS | [elevenlabs.io](https://elevenlabs.io/) |
| Azure Speech | STT, TTS | [Azure Portal](https://portal.azure.com/) |

---

## Optional (for phone integration)

To connect voice agents to phone calls:

| Provider | Purpose | Get Started |
|----------|---------|-------------|
| Twilio | Phone calls, SMS | [twilio.com/console](https://www.twilio.com/console) |
| Vonage | Phone calls | [dashboard.vonage.com](https://dashboard.vonage.com/) |
| Exotel | Phone calls (India) | [exotel.com](https://exotel.com/) |

Phone integration is covered in [doc.rapida.ai](https://doc.rapida.ai) under External Integrations.

---

## Network Requirements

Rapida uses these ports locally:

| Service | Port |
|---------|------|
| UI | 3000 |
| Web API | 9001 |
| Assistant API | 9007 |
| Integration API | 9004 |
| Endpoint API | 9005 |
| Document API | 9010 |
| PostgreSQL | 5432 |
| Redis | 6379 |
| OpenSearch | 9200 |
| NGINX (proxy) | 8080 |

Ensure these ports are available. If you have conflicts, you can modify port mappings in `docker-compose.yml`.

---

## Next Step

Ready? Continue to [Installation](./INSTALLATION.md).

