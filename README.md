<picture>
  <source media="(prefers-color-scheme: dark)" srcset="https://github.com/rapidaai/voice-ai/blob/87ff2937e086162acb5f5fad8c01e3f1f582b87b/.github/banner-01.png">
  <source media="(prefers-color-scheme: light)" srcset="https://github.com/rapidaai/voice-ai/blob/87ff2937e086162acb5f5fad8c01e3f1f582b87b/.github/banner-01.png">
  <img style="width:100%;" src="https://github.com/rapidaai/voice-ai/blob/87ff2937e086162acb5f5fad8c01e3f1f582b87b/.github/banner-01.png" alt="Banner">
</picture>

# Rapida: End-to-End Voice Orchestration Platform

[Rapida](https://rapida.ai) is an open-source platform for designing, building, and deploying voice agents at scale.  
It’s built around three core principles:

- **Reliable** — designed for production workloads, real-time audio, and fault-tolerant execution  
- **Observable** — deep visibility into calls, latency, metrics, and tool usage  
- **Customizable** — flexible architecture that adapts to any LLM, workflow, or enterprise stack  

Rapida provides both a **platform** and a **framework** for building real-world voice agents—from low-latency audio streaming to orchestration, monitoring, and integrations.

Rapida is written in **Go**, using the highly optimized [gRPC](https://github.com/grpc/grpc-go) protocol for fast, efficient, bidirectional communication.

---
## Features

- **Real-time Voice Orchestration**  
  Stream and process audio with low latency using GRPC.

- **LLM-Agnostic Architecture**  
  Bring your own model—OpenAI, Anthropic, open-source models, or custom inference.

- **Production-grade Reliability**  
  Built-in retries, error handling, call lifecycle management, and health checks.

- **Full Observability**  
  Call logs, streaming events, tool traces, latency breakdowns, metrics, and dashboards.

- **Flexible Tooling System**  
  Build custom tools and actions for your agents, or integrate with any backend.

- **Developer-friendly**  
  Clear APIs, modular components, and simple configuration.

- **Enterprise-ready**  
  Scalable design, efficient protocol, and predictable performance.

## Architecture Overview

┌─────────────────────────────────────────────────────────────┐
│                          CHANNELS                           │
│     Phone • Web • WhatsApp • SIP • WebRTC • Others          │
└───────────────────────────────┬─────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────┐
│                      RAPIDA ORCHESTRATOR                    │
│   Routing • State • Tooling • Observability • Messaging     │
└───────────────┬───────────────────────────────┬─────────────┘
                │                               │
                │                               │
                ▼                               ▼
     ┌─────────────────────┐          ┌──────────────────────┐
     │  Audio Preprocess   │          │        STT           │
     │ • VAD               │          │  Speech-to-Text      │
     │ • Noise Reduction   │<--------→│  (ASR Engine)        │
     │ • End-of-Speech     │          └──────────┬───────────┘
     └──────────┬──────────┘                     │
                │                                ▼
                │                     ┌────────────────────────┐
                │                     │          LLM           │
                │                     │ Reasoning • Tools •    │
                │                     │  Memory • Policies     │
                │                     └───────────┬────────────┘
                │                                 │
                │                                 ▼
                │                     ┌────────────────────────┐
                │                     │          TTS           │
                └────────────────────→│   Text-to-Speech       │
                                      └───────────┬────────────┘
                                                  │
                                                  ▼
                                   ┌────────────────────────────────────┐
                                   │             USER OUTPUT            │
                                   │      Audio Stream / Response       │
                                   └────────────────────────────────────┘

## Documentation & Guides

https://doc.rapida.ai

## SDKs & Tools

### Client SDKs

Client SDKs enable your frontend to include interactive, multi-user experiences.

| Language                | Repo                                                                                    | Docs                                                        |
| :---------------------- | :-------------------------------------------------------------------------------------- | :---------------------------------------------------------- |
| Web (React)                     | [rapida-react](https://github.com/rapidaai/rapida-react)                               | [docs](https://doc.rapida.ai/api-reference/installation) |
| Web Widget (react)                    | [react-widget](https://github.com/rapidaai/react-widget)                           |                                                             |

### Server SDKs

Server SDKs enable your backend to build and manage agents.

| Language                | Repo                                                                                    | Docs                                                        |
| :---------------------- | :-------------------------------------------------------------------------------------- | :---------------------------------------------------------- |
| Go                      | [rapida-go](https://github.com/rapidaai/rapida-go)                               | [docs](https://doc.rapida.ai/api-reference/installation) |
| Python| [rapida-python](https://github.com/rapidaai/rapida-python)                               | [docs](https://doc.rapida.ai/api-reference/installation/)              |

## Contributing

For those who'd like to contribute code, see our [Contribution Guide](https://github.com/rapidaai/voice-ai/blob/main/CONTRIBUTING.md).
At the same time, please consider supporting RapidaAi by sharing it on social media and at events and conferences.


## Security disclosure

To protect your privacy, please avoid posting security issues on GitHub. Instead, report issues to contact@rapida.ai, and our team will respond with detailed answer.

## License

Rapida is open-source under the GPL-2.0 license, with additional conditions:
- Open-source users must keep the Rapida logo visible in UI components.
- Future license terms may change; this does not affect released versions.

A commercial license is available for enterprise use, which allows:
- Removal of branding
- Closed-source usage
- Private modifications
Contact sales@rapida.ai for details.