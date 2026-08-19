# CONTRIBUTING

So you're looking to contribute to **Rapida** that’s awesome.  
We’re building an open, reliable, end-to-end voice orchestration stack, and community contributions matter a lot. As a young project with a big vision, every bit of help truly counts.

We want to keep development fast and nimble, while also giving contributors a smooth path into the codebase. This contribution guide is meant to help you understand how we work, what to expect, and how to get started quickly.

Just like Rapida itself, this guide is a **work in progress**. Some sections may lag behind development, and we will continue improving the contributor documentation as the project grows.

---

## Before You Dive In

Looking for something to work on?  
- Check out issues labeled **good first issue**.  
- Want to add a new STT/TTS/LLM provider? Open a PR and show us what you’ve built.  
- Want to contribute new tools, telephony integrations, or audio-processing components? You’re welcome these areas are wide open.

### Detailed contribution docs coming soon

We will publish full contributor documentation for the following areas:

- Adding a new **speech-to-text** provider  
- Adding a new **text-to-speech** provider  
- Adding new **LLM engines**  
- Adding new **telephony channels** and RTC integrations  
- Adding **audio components** (noise reduction, VAD, echo cancellation, AGC, etc.)  
- Adding **LLM tools** for agent actions  
- Improving orchestrator logic, observability, and performance  

These guides are **not written yet** they will be added as the project stabilizes.  
For now, feel free to explore the codebase or open an issue if you need guidance.

---

## Bug Reports

Please include:

- A clear title  
- A detailed description  
- Steps to reproduce  
- Expected behavior  
- Logs (critical for voice pipelines & telephony flows)  
- Screenshots or call traces, if applicable  

### How we prioritize:

| Issue Type | Priority |
|------------|----------|
| Core failures (call crashes, media pipeline breaks, STT/LLM/TTS blocking, security) | **Critical** |
| Non-critical bugs, performance issues | **Medium** |
| Minor fixes (typos, cosmetic issues) | **Low** |

---

## Feature Requests

Please include:

- A clear, descriptive title  
- What the feature is  
- Why it’s needed  
- The use case / scenario  
- Any context, examples, or screenshots  

### How we prioritize:

| Feature Type | Priority |
|--------------|----------|
| High-priority features labeled by maintainers | **High** |
| Popular community requests | **Medium** |
| Non-core or small improvements | **Low** |
| Useful but not urgent | **Future** |

---

## Submitting a Pull Request

1. **Fork** the repository  
2. **Open an issue first** to discuss your proposed change  
3. Create a new branch  
4. Add tests where applicable  
5. Ensure all tests pass  
6. Link the issue in the PR description  
7. Submit the PR  
8. Get merged! 🎉

## Local Quality Gates

Install the repository hooks once after cloning:

```sh
./bin/git-commit-hook-setup.sh
```

The pre-commit hook checks merge-conflict markers, private keys, common secret tokens, JSON syntax, shell syntax, Go formatting, and `go mod tidy`.

The pre-push hook runs CI-safe Go gates that currently pass in this repository: `go vet` and Linux builds for `web-api`, `integration-api`, and `endpoint-api`. The stricter `golangci-lint`, import-order formatter, and full Go test hooks are available as manual checks while the existing lint/test baseline is cleaned up. The `assistant-api` package is intentionally excluded from these local pre-push checks because it needs external CGO libraries; Docker/CI validation covers that surface.

Useful manual commands:

```sh
./bin/pre-commit run --all-files
./bin/pre-commit run --hook-stage push --all-files
./bin/pre-commit run --hook-stage manual --all-files
```

---

## Setting Up Rapida

Full setup guides will be added soon for:

- **Backend orchestrator**
- **Voice pipelines (STT/LLM/TTS)**
- **Channels / telephony adapters**
- **Monitoring & local development tools**

Until then:

- Refer to the root `README.md`  
- Open an issue if you need help with setup  

---

## Getting Help

If you get stuck or have questions:

- Open an issue in the repo  
- Join our community discussions (Discord/Slack coming soon)

We're here to help.  
Let’s build the future of voice AI together. 🔊✨
