# Transformer Package - STT & TTS Implementation Guide

The transformer package provides a unified abstraction for **Speech-to-Text (STT)** and **Text-to-Speech (TTS)** providers. It allows seamless integration with multiple AI providers while maintaining a consistent interface for developers.

## Overview

The package supports the following providers:

### Speech-to-Text (STT) Providers

- **Deepgram** - WebSocket-based streaming transcription
- **Google Cloud Speech** - Google's speech recognition service
- **Azure Speech Services** - Microsoft Azure speech recognition
- **AssemblyAI** - High-accuracy transcription API
- **RevAI** - Asynchronous speech-to-text service
- **Sarvam AI** - Indian language support
- **Cartesia** - Low-latency streaming STT

### Text-to-Speech (TTS) Providers

- **Deepgram** - High-quality voice synthesis
- **Google Cloud Text-to-Speech** - Google's TTS engine
- **Azure Speech Services** - Microsoft Azure TTS
- **Cartesia** - Real-time voice synthesis
- **RevAI** - TTS with voice customization
- **Sarvam AI** - Indian language voice synthesis
- **ElevenLabs** - AI-powered realistic voices
- **AWS Polly** - (Placeholder for future implementation)

---

## Architecture

### Factory Pattern

The package uses a factory pattern to instantiate transformers:

```go
import (
    transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
)

// Get STT transformer
sttTransformer, err := transformer.NewSpeechToText(
    transformer.WithContext(ctx),
    transformer.WithLogger(logger),
    transformer.WithProvider(transformer.DEEPGRAM.String()),
    transformer.WithCredential(credential),
    transformer.WithOnPacket(onPacket),
    transformer.WithOptions(opts),
)

// Get TTS transformer
ttsTransformer, err := transformer.GetTextToSpeechTransformer(
    transformer.GOOGLE_SPEECH_SERVICE,
    ctx,
    logger,
    credential,
    opts,
)
```

### Core Interfaces

#### Speech-to-Text Interface

```go
type SpeechToTextTransformer interface {
    // Returns human-readable name of the transformer
    Name() string

    // Initialize establishes connection to the service
    Initialize() error

    // Transform sends audio bytes for transcription
    // Calls OnPacket callback when transcript is available
    Transform(ctx context.Context, audioBytes []byte) error

    // Close cleans up resources and closes connections
    Close(ctx context.Context) error
}
```

#### Text-to-Speech Interface

```go
type TextToSpeechTransformer interface {
    // Returns human-readable name of the transformer
    Name() string

    // Initialize establishes connection to the service
    Initialize() error

    // Transform sends text for speech synthesis
    // Calls OnSpeech callback when audio is available
    Transform(ctx context.Context, packet Packet) error

    // Close cleans up resources and closes connections
    Close(ctx context.Context) error
}
```

#### Base Interface

Both STT and TTS extend the generic `Transformers[T]` interface:

```go
type Transformers[IN any] interface {
    Initialize() error
    Transform(context.Context, IN) error
    Close(context.Context) error
}
```

---

## Creating a New STT Provider

Follow these steps to add a new Speech-to-Text provider:

### Step 1: Create Provider Directory

```
transformer/
├── my-provider/
│   ├── stt.go                 # Main STT implementation
│   ├── option.go              # Configuration & credentials
│   └── internal/
│       └── type.go            # Internal type definitions
```

### Step 2: Define Configuration Structure (option.go)

```go
package internal_transformer_myprovider

import (
    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/protos"
)

type myProviderOption struct {
    apiKey       string
    language     string
    sampleRate   int
    // Add provider-specific configuration
}

func NewMyProviderOption(
    logger commons.Logger,
    credential *protos.VaultCredential,
    audioConfig *protos.AudioConfig,
    modelOpts interface{},
) (*myProviderOption, error) {
    // Extract API key from vault credential
    apiKey := credential.GetApiKey()
    if apiKey == "" {
        logger.Errorf("my-provider: API key not found in credentials")
        return nil, fmt.Errorf("missing API key")
    }

    // Extract configuration from audioConfig and modelOpts
    language := "en-US"
    sampleRate := 16000

    if audioConfig != nil {
        language = audioConfig.LanguageCode
        sampleRate = int(audioConfig.SampleRateHertz)
    }

    return &myProviderOption{
        apiKey:     apiKey,
        language:   language,
        sampleRate: sampleRate,
    }, nil
}

// Getter methods
func (m *myProviderOption) GetKey() string {
    return m.apiKey
}

func (m *myProviderOption) GetLanguage() string {
    return m.language
}
```

### Step 3: Implement STT Transformer (stt.go)

```go
package internal_transformer_myprovider

import (
    "context"
    "fmt"
    "sync"

    internal_type "github.com/rapidaai/api/assistant-api/internal/type"
    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/protos"
)

type myProviderSpeechToText struct {
    *myProviderOption

    mu     sync.Mutex
    logger commons.Logger

    // Connection management
    ctx       context.Context
    ctxCancel context.CancelFunc

    // Service-specific fields
    client              interface{} // Your provider's client
    options             *internal_type.SpeechToTextInitializeOptions
}

// Name returns the provider name
func (*myProviderSpeechToText) Name() string {
    return "my-provider-speech-to-text"
}

// NewMyProviderSpeechToText creates a new STT transformer instance
func NewMyProviderSpeechToText(
    ctx context.Context,
    logger commons.Logger,
    credential *protos.VaultCredential,
    opts *internal_type.SpeechToTextInitializeOptions,
) (internal_type.SpeechToTextTransformer, error) {
    // Create provider-specific options
    providerOpts, err := NewMyProviderOption(
        logger,
        credential,
        opts.AudioConfig,
        opts.ModelOptions,
    )
    if err != nil {
        logger.Errorf("my-provider-stt: failed to create options: %v", err)
        return nil, err
    }

    // Create cancellable context
    ct, ctxCancel := context.WithCancel(ctx)

    return &myProviderSpeechToText{
        ctx:                ct,
        ctxCancel:          ctxCancel,
        logger:             logger,
        myProviderOption:   providerOpts,
        options:            opts,
    }, nil
}

// Initialize establishes connection to the provider
func (m *myProviderSpeechToText) Initialize() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.logger.Debugf("my-provider-stt: initializing connection")

    // 1. Create client (provider-specific)
    client, err := m.createClient()
    if err != nil {
        m.logger.Errorf("my-provider-stt: failed to create client: %v", err)
        return fmt.Errorf("my-provider-stt: %w", err)
    }

    // 2. Connect to service
    if err := m.connect(client); err != nil {
        m.logger.Errorf("my-provider-stt: failed to connect: %v", err)
        return fmt.Errorf("my-provider-stt: connection failed: %w", err)
    }

    // 3. Start callback handler goroutine
    go m.speechToTextCallback(client, m.ctx)

    m.client = client
    m.logger.Debugf("my-provider-stt: connection established")
    return nil
}

// Transform sends audio bytes to the provider
func (m *myProviderSpeechToText) Transform(ctx context.Context, audioBytes []byte) error {
    m.mu.Lock()
    client := m.client
    m.mu.Unlock()

    if client == nil {
        return fmt.Errorf("my-provider-stt: transformer not initialized")
    }

    // Send audio to provider (provider-specific)
    if err := m.streamAudio(client, audioBytes); err != nil {
        m.logger.Errorf("my-provider-stt: failed to stream audio: %v", err)
        return fmt.Errorf("my-provider-stt: stream error: %w", err)
    }

    return nil
}

// Close terminates the connection and cleans up resources
func (m *myProviderSpeechToText) Close(ctx context.Context) error {
    m.ctxCancel()

    m.mu.Lock()
    defer m.mu.Unlock()

    if m.client != nil {
        // Close connection (provider-specific)
        m.logger.Debugf("my-provider-stt: closing connection")
        // TODO: implement graceful close
    }

    m.logger.Debugf("my-provider-stt: connection closed")
    return nil
}

// Private helper methods

func (m *myProviderSpeechToText) createClient() (interface{}, error) {
    // TODO: Create and return provider's client instance
    return nil, nil
}

func (m *myProviderSpeechToText) connect(client interface{}) error {
    // TODO: Establish connection to provider service
    return nil
}

func (m *myProviderSpeechToText) streamAudio(client interface{}, audioBytes []byte) error {
    // TODO: Send audio bytes to provider
    return nil
}

func (m *myProviderSpeechToText) speechToTextCallback(client interface{}, ctx context.Context) {
    // TODO: Listen for responses and call OnPacket callback
    // Example:
    // for {
    //     response := <-responseChan
    //     if m.options.OnPacket != nil {
    //         pkt := &internal_type.SpeechToTextPacket{
    //             Text:       response.Text,
    //             IsFinal:    response.IsFinal,
    //             IsComplete: response.IsComplete,
    //         }
    //         m.options.OnPacket(pkt)
    //     }
    // }
}
```

---

## Creating a New TTS Provider

Follow these steps to add a new Text-to-Speech provider:

### Step 1: Create Provider Directory

```
transformer/
├── my-provider/
│   ├── tts.go                 # Main TTS implementation
│   ├── option.go              # Configuration & credentials
│   └── internal/
│       └── type.go            # Internal type definitions
```

### Step 2: Define Configuration Structure (option.go)

Similar to STT, define your provider-specific options:

```go
type myProviderOption struct {
    apiKey   string
    voiceId  string
    language string
    // Add provider-specific configuration
}

func NewMyProviderOption(...) (*myProviderOption, error) {
    // Extract configuration from credentials and options
}
```

### Step 3: Implement TTS Transformer (tts.go)

```go
package internal_transformer_myprovider

import (
    "context"
    "fmt"
    "sync"

    internal_type "github.com/rapidaai/api/assistant-api/internal/type"
    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/protos"
)

type myProviderTextToSpeech struct {
    *myProviderOption

    mu     sync.Mutex
    logger commons.Logger

    // Connection management
    ctx       context.Context
    ctxCancel context.CancelFunc

    // Service-specific fields
    client              interface{} // Your provider's client
    options             *internal_type.TextToSpeechInitializeOptions
}

// Name returns the provider name
func (*myProviderTextToSpeech) Name() string {
    return "my-provider-text-to-speech"
}

// NewMyProviderTextToSpeech creates a new TTS transformer instance
func NewMyProviderTextToSpeech(
    ctx context.Context,
    logger commons.Logger,
    credential *protos.VaultCredential,
    opts *internal_type.TextToSpeechInitializeOptions,
) (internal_type.TextToSpeechTransformer, error) {
    // Create provider-specific options
    providerOpts, err := NewMyProviderOption(logger, credential, opts.AudioConfig, opts.ModelOptions)
    if err != nil {
        logger.Errorf("my-provider-tts: failed to create options: %v", err)
        return nil, err
    }

    ct, ctxCancel := context.WithCancel(ctx)

    return &myProviderTextToSpeech{
        ctx:                ct,
        ctxCancel:          ctxCancel,
        logger:             logger,
        myProviderOption:   providerOpts,
        options:            opts,
    }, nil
}

// Initialize establishes connection to the provider
func (m *myProviderTextToSpeech) Initialize() error {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.logger.Debugf("my-provider-tts: initializing connection")

    // 1. Create client (provider-specific)
    client, err := m.createClient()
    if err != nil {
        m.logger.Errorf("my-provider-tts: failed to create client: %v", err)
        return fmt.Errorf("my-provider-tts: %w", err)
    }

    // 2. Connect to service
    if err := m.connect(client); err != nil {
        m.logger.Errorf("my-provider-tts: failed to connect: %v", err)
        return fmt.Errorf("my-provider-tts: connection failed: %w", err)
    }

    // 3. Start callback handler goroutine
    go m.textToSpeechCallback(client, m.ctx)

    m.client = client
    m.logger.Debugf("my-provider-tts: connection established")
    return nil
}

// Transform sends text to the provider for synthesis
func (m *myProviderTextToSpeech) Transform(ctx context.Context, packet internal_type.Packet) error {
    m.mu.Lock()
    client := m.client
    m.mu.Unlock()

    if client == nil {
        return fmt.Errorf("my-provider-tts: transformer not initialized")
    }

    // Send text to provider (provider-specific)
    if err := m.synthesize(client, packet); err != nil {
        m.logger.Errorf("my-provider-tts: failed to synthesize: %v", err)
        return fmt.Errorf("my-provider-tts: synthesis error: %w", err)
    }

    return nil
}

// Close terminates the connection and cleans up resources
func (m *myProviderTextToSpeech) Close(ctx context.Context) error {
    m.ctxCancel()

    m.mu.Lock()
    defer m.mu.Unlock()

    if m.client != nil {
        m.logger.Debugf("my-provider-tts: closing connection")
        // TODO: implement graceful close
    }

    m.logger.Debugf("my-provider-tts: connection closed")
    return nil
}

// Private helper methods

func (m *myProviderTextToSpeech) createClient() (interface{}, error) {
    // TODO: Create and return provider's client instance
    return nil, nil
}

func (m *myProviderTextToSpeech) connect(client interface{}) error {
    // TODO: Establish connection to provider service
    return nil
}

func (m *myProviderTextToSpeech) synthesize(client interface{}, packet internal_type.Packet) error {
    // TODO: Send text to provider
    return nil
}

func (m *myProviderTextToSpeech) textToSpeechCallback(client interface{}, ctx context.Context) {
    // TODO: Listen for audio responses and call OnSpeech callback
    // Example:
    // for {
    //     audioData := <-audioChan
    //     if m.options.OnSpeech != nil {
    //         pkt := &internal_type.AudioPacket{
    //             Audio: audioData,
    //         }
    //         m.options.OnSpeech(pkt)
    //     }
    // }
}
```

---

## Step 4: Register Provider in Factory

Update [transformer.go](transformer.go) to register your new provider:

### For STT:

```go
const (
    // ... existing providers
    MYPROVIDER AudioTransformer = "my-provider"
)

func NewSpeechToText(opts ...Option) (internal_type.SpeechToTextTransformer, error) {
    // ... apply options
    switch AudioTransformer(options.provider) {
    // ... existing cases
    case MYPROVIDER:
        return internal_transformer_myprovider.NewSpeechToText(
            internal_transformer_myprovider.WithContext(options.ctx),
            internal_transformer_myprovider.WithLogger(options.logger),
            internal_transformer_myprovider.WithCredential(options.credential),
            internal_transformer_myprovider.WithOnPacket(options.onPacket),
            internal_transformer_myprovider.WithOptions(options.sttOptions),
        )
    default:
        return nil, fmt.Errorf("illegal speech to text identifier")
    }
}
```

### For TTS:

```go
func GetTextToSpeechTransformer(
    at AudioTransformer,
    ctx context.Context,
    logger commons.Logger,
    credential *protos.VaultCredential,
    opts *internal_type.TextToSpeechInitializeOptions,
) (internal_type.TextToSpeechTransformer, error) {
    switch at {
    // ... existing cases
    case MYPROVIDER:
        return internal_transformer_myprovider.NewMyProviderTextToSpeech(ctx, logger, credential, opts)
    default:
        return nil, fmt.Errorf("illegal text to speech identifier")
    }
}
```

---

## Best Practices

### 1. **Thread Safety**

- Use `sync.Mutex` to protect shared state
- Always lock before accessing/modifying the client
- Example pattern:

```go
m.mu.Lock()
client := m.client
m.mu.Unlock()
// Use client without holding lock
```

### 2. **Error Handling**

- Wrap errors with context: `fmt.Errorf("provider-stt: %w", err)`
- Always log errors with logger before returning
- Prefix log messages with provider name for debugging

### 3. **Context Management**

- Use `context.WithCancel()` for graceful shutdown
- Pass context to all async operations
- Cancel context in `Close()` method

### 4. **Callback Execution**

- Never call callbacks while holding locks
- Check if callback is nil before calling
- Handle callback errors appropriately
- STT calls `OnPacket` with `SpeechToTextPacket`
- TTS calls `OnSpeech` with audio `Packet`

### 5. **Connection Lifecycle**

- Initialize: Create client and establish connection
- Transform: Send data (can be called multiple times)
- Close: Clean up resources and close connection
- Handle edge case: Transform before Initialize

### 6. **Concurrency Patterns**

- **WebSocket-based** (Deepgram, Cartesia, Sarvam): Send data through persistent connection
- **HTTP-based** (Google Cloud, Azure): Create new requests for each Transform
- **Async/Polling** (RevAI): Submit job and poll for results
- **Streaming** (AssemblyAI): Handle streaming responses

---

## Usage Example

```go
import (
    "context"
    transformer "github.com/rapidaai/api/assistant-api/internal/transformer"
    "github.com/rapidaai/pkg/commons"
)

func main() {
    logger, _ := commons.NewApplicationLogger()
    ctx := context.Background()

    // Create STT transformer
    sttOpts := &internal_type.SpeechToTextInitializeOptions{
        AudioConfig: &protos.AudioConfig{
            LanguageCode:      "en-US",
            SampleRateHertz:   16000,
        },
        OnPacket: func(pkts ...internal_type.Packet) error {
            for _, pkt := range pkts {
                logger.Infof("Transcript: %s (final: %v)", pkt.Text, pkt.IsFinal)
            }
            return nil
        },
    }

    stt, err := transformer.NewSpeechToText(
        transformer.WithContext(ctx),
        transformer.WithLogger(logger),
        transformer.WithProvider(transformer.DEEPGRAM.String()),
        transformer.WithCredential(credential),
        transformer.WithOnPacket(sttOpts.OnPacket),
        transformer.WithOptions(options),
    )
    if err != nil {
        logger.Fatalf("Failed to create STT transformer: %v", err)
    }

    // Initialize
    if err := stt.Initialize(); err != nil {
        logger.Fatalf("Failed to initialize: %v", err)
    }

    // Stream audio
    audioData := []byte{/* ... */}
    if err := stt.Transform(ctx, audioData); err != nil {
        logger.Errorf("Transform failed: %v", err)
    }

    // Close
    defer stt.Close(ctx)
}
```

---

## Testing Your Implementation

Create `provider_test.go` in your provider directory:

```go
package internal_transformer_myprovider

import (
    "context"
    "testing"

    internal_type "github.com/rapidaai/api/assistant-api/internal/type"
    "github.com/rapidaai/pkg/commons"
    "github.com/rapidaai/protos"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewMyProviderSpeechToText(t *testing.T) {
    logger, _ := commons.NewApplicationLogger()
    ctx := context.Background()

    credential := &protos.VaultCredential{
        ApiKey: "test-key",
    }
    opts := &internal_type.SpeechToTextInitializeOptions{
        AudioConfig: &protos.AudioConfig{
            LanguageCode: "en-US",
        },
    }

    stt, err := NewMyProviderSpeechToText(ctx, logger, credential, opts)
    require.NoError(t, err)
    assert.NotNil(t, stt)
    assert.Equal(t, "my-provider-speech-to-text", stt.Name())
}

func TestInitialize(t *testing.T) {
    logger, _ := commons.NewApplicationLogger()
    ctx := context.Background()

    credential := &protos.VaultCredential{
        ApiKey: "test-key",
    }
    opts := &internal_type.SpeechToTextInitializeOptions{}

    stt, _ := NewMyProviderSpeechToText(ctx, logger, credential, opts)

    // Should fail with test credentials
    err := stt.Initialize()
    assert.Error(t, err) // Expected with invalid credentials

    defer stt.Close(ctx)
}
```

---

## Common Patterns by Provider Type

### WebSocket-Based (Deepgram, Cartesia, Sarvam)

```go
// 1. Create WebSocket client in Initialize()
conn, err := websocket.Dial(url, headers)

// 2. Send configuration on connection
conn.WriteJSON(config)

// 3. Stream audio in Transform()
conn.WriteMessage(websocket.BinaryMessage, audioBytes)

// 4. Read responses in callback goroutine
for {
    response := &Response{}
    conn.ReadJSON(response)
    // Call OnPacket callback
}

// 5. Close in Close()
conn.Close()
```

### REST API-Based (Google Cloud, Azure)

```go
// 1. Create HTTP client in Initialize()
client := &http.Client{}

// 2. Each Transform() makes a new request
response, err := client.Do(createRequest(audioBytes))

// 3. Parse response and call callback
pkt := parseResponse(response)
m.options.OnPacket(pkt)

// 4. No special Close() needed for stateless HTTP
```

### Async/Polling-Based (RevAI)

```go
// 1. Initialize: Prepare client
client := revai.NewClient(apiKey)

// 2. Transform: Submit audio job
jobId, err := client.SubmitJob(audioBytes)

// 3. Polling: Check status in callback goroutine
for {
    status := client.GetJobStatus(jobId)
    if status.IsComplete {
        pkt := parseTranscript(status)
        m.options.OnPacket(pkt)
        break
    }
}

// 4. Close: Clean up ongoing jobs
client.DeleteJob(jobId)
```

---

## Troubleshooting

| Issue                                | Solution                                                     |
| ------------------------------------ | ------------------------------------------------------------ |
| Calls to Transform before Initialize | Check for nil client before use, return clear error          |
| Race conditions on shared state      | Always use mutex when accessing shared fields                |
| Deadlocks                            | Don't call callbacks while holding locks                     |
| Memory leaks                         | Ensure Close() is called and cancels goroutines              |
| Connection timeouts                  | Set appropriate context timeouts in Initialize()             |
| Lost callbacks                       | Store references to options, don't rely on closure variables |

---

## Summary

To implement a new STT or TTS provider:

1. **Create directory**: `transformer/{provider}/`
2. **Implement `option.go`**: Configuration extraction from credentials
3. **Implement `{stt|tts}.go`**: Core transformer with interface methods
4. **Register constructor**: Add STT provider cases to `NewSpeechToText()` and TTS provider cases to `GetTextToSpeechTransformer()`
5. **Test thoroughly**: Unit tests + integration tests with real credentials
6. **Follow patterns**: Use mutex for thread safety, context for cancellation

The transformer package provides a clean abstraction for voice AI integration, enabling developers to add new providers with minimal boilerplate.
