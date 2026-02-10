# Telnyx Telephony Provider Integration

This directory contains the Telnyx telephony provider implementation for the Rapida voice orchestration platform.

## Overview

Telnyx provides programmable voice and messaging services with the following capabilities:

- **SIP Trunking**: Enterprise-grade voice connectivity
- **Programmable Voice**: API-driven call control
- **TeXML**: XML-based instruction set for call handling
- **WebRTC Support**: Real-time WebSocket audio streaming

## Configuration

The Telnyx provider requires the following credentials stored in the vault:

```yaml
credentials:
  api_key: "your-telnyx-api-key"
  api_secret: "your-telnyx-api-secret"
```

### Environment Variables

Configure the following environment variables for the Telnyx provider:

| Variable | Description | Required |
|----------|-------------|----------|
| `TELNYX_API_KEY` | Telnyx API Key | Yes |
| `TELNYX_API_SECRET` | Telnyx API Secret | Yes |
| `PUBLIC_ASSISTANT_HOST` | Public hostname for WebSocket callbacks | Yes |

## Architecture

### Directory Structure

```
telnyx/
├── telephony.go           # Main provider implementation
├── websocket.go           # WebSocket streamer for audio
└── internal/
    └── type.go            # Telnyx-specific types
```

### Components

#### Telephony Interface

The `telnyxTelephony` struct implements the `Telephony` interface with the following methods:

| Method | Description |
|--------|-------------|
| `Streamer()` | Creates WebSocket streamer for real-time audio |
| `StatusCallback()` | Handles call status events from Telnyx |
| `CatchAllStatusCallback()` | Handles unexpected callbacks |
| `ReceiveCall()` | Extracts caller information from incoming requests |
| `OutboundCall()` | Initiates outbound calls |
| `InboundCall()` | Answers incoming calls with TeXML |

#### WebSocket Streamer

The `telnyxWebsocketStreamer` handles real-time audio exchange:

| Method | Description |
|--------|-------------|
| `Context()` | Returns the stream context |
| `Recv()` | Receives audio/messages from Telnyx |
| `Send()` | Sends audio/responses to Telnyx |

## Call Flow

```
1. Incoming Call
   ↓
   [Telnyx] → [InboundCall()] → Return TeXML with WebSocket URL
   ↓
2. WebSocket Connection
   ↓
   [Streamer()] → Create media handler
   ↓
3. Real-time Audio Exchange
   ↓
   [Recv()] → Audio from caller (PCM 16kHz)
   [Send()] → Audio to caller (PCM 16kHz)
   ↓
4. Call Events
   ↓
   [StatusCallback()] → Handle status updates
   ↓
5. Call Termination
   ↓
   [Streamer.Close()] → Clean up WebSocket
```

## TeXML Reference

Telnyx uses TeXML for call control. The provider generates TeXML like this:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Response>
    <Connect>
        <Stream url="wss://your-host/stream" name="call-name" statusCallback="https://your-host/callback">
            <Parameter name="assistant_id" value="123"/>
            <Parameter name="client_number" value="+1234567890"/>
        </Stream>
    </Connect>
</Response>
```

## Audio Format

| Property | Value |
|----------|-------|
| Encoding | PCM 16-bit signed |
| Sample Rate | 16000 Hz |
| Channels | Mono |
| WebSocket Message Type | Binary |

## Usage

### Inbound Call Handling

```go
provider, _ := telephony.GetTelephony(
    telephony.Telnyx,
    assistantConfig,
    logger,
)

// Telnyx sends webhook to InboundCall endpoint
err := provider.InboundCall(ctx, auth, assistantId, clientNumber, conversationId)
```

### Outbound Call

```go
metadata, metrics, events, err := provider.OutboundCall(
    auth,              // Authentication context
    "+14155551234",    // To phone number
    "+14155559876",    // From phone number
    assistantId,       // Assistant ID
    conversationId,    // Conversation ID
    vaultCredential,    // API credentials
    nil,               // Options
)
```

### WebSocket Streaming

```go
// Upgrade HTTP to WebSocket
conn, _ := upgrader.Upgrade(w, r, nil)

// Create streamer
streamer := provider.Streamer(ctx, conn, assistant, conversation, credential)

// Receive audio
for {
    req, err := streamer.Recv()
    if err == io.EOF {
        break
    }
    // Process audio...
}

// Send audio response
err = streamer.Send(response)
```

## Events

The Telnyx provider emits the following event types:

| Event | Description |
|-------|-------------|
| `connected` | WebSocket connection established |
| `start` | Media stream started, contains stream ID |
| `media` | Audio data message |
| `stop` | Media stream stopped |
| `status` | Call status update (ringing, answered, completed) |

## Error Handling

### Common Errors

| Error | Cause | Resolution |
|-------|-------|------------|
| `illegal vault config api_key is not found` | Missing credentials | Configure Telnyx API credentials in vault |
| `illegal vault config api_secret not found` | Missing credentials | Configure Telnyx API secret in vault |
| `WebSocket connection is nil` | Connection dropped | Check network connectivity |
| `failed to unmarshal Telnyx media event` | Invalid JSON | Verify Telnyx sends valid JSON |

### Logging

The provider logs at the following levels:

- `Debug`: Detailed event processing
- `Info`: Call lifecycle events
- `Warn`: Unexpected event types
- `Error`: Operation failures

## Testing

Run the provider tests:

```bash
go test -v ./api/assistant-api/internal/telephony/internal/telnyx/...
```

## Integration Checklist

- [ ] Configure Telnyx API credentials
- [ ] Set public assistant host for callbacks
- [ ] Register Telnyx in the telephony factory
- [ ] Configure Telnyx webhook URLs
- [ ] Test inbound call flow
- [ ] Test outbound call flow
- [ ] Verify WebSocket audio streaming
- [ ] Validate event callback handling

## References

- [Telnyx API Documentation](https://developers.telnyx.com/)
- [Telnyx Programmable Voice](https://developers.telnyx.com/api/v2/voice)
- [TeXML Reference](https://developers.telnyx.com/api/v2/texml)
- [Rapida Telephony Interface](https://github.com/rapidaai/voice-ai/blob/main/api/assistant-api/internal/type/telephony.go)