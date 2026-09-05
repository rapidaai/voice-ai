# FreeSWITCH SIP Integration Fixtures

This package contains the gated SIP interoperability tests for FreeSWITCH and
the FreeSWITCH configuration fragments required to run them.

The tests are opt-in and are not part of the default unit suite.

Required environment:

```bash
RAPIDA_SIP_FREESWITCH_ENABLE=1
RAPIDA_SIP_REDIS_ADDR=127.0.0.1:6379
FREESWITCH_SIP_USERNAME=rapida
FREESWITCH_SIP_PASSWORD=secret
FREESWITCH_OUTBOUND_ANSWER_USER=9197001
FREESWITCH_OUTBOUND_HEADER_ASSERT_USER=9197002
FREESWITCH_OUTBOUND_RING_ONLY_USER=9197003
FREESWITCH_REGISTER_DID=rapida
FREESWITCH_INBOUND_CALLER_USER=9197000
FREESWITCH_TWILIO_OUTBOUND_USER=9197010
FREESWITCH_TWILIO_INBOUND_DID=+15551230000
FREESWITCH_OUTBOUND_BUSY_USER=9197020
FREESWITCH_OUTBOUND_REJECTED_USER=9197021
FREESWITCH_OUTBOUND_NO_ANSWER_USER=9197022
FREESWITCH_OUTBOUND_UNAVAILABLE_USER=9197023
FREESWITCH_OUTBOUND_MEDIA_REJECT_USER=9197024
```

Useful defaults:

```bash
FREESWITCH_CLI=/opt/homebrew/opt/freeswitch/bin/fs_cli
FREESWITCH_HOST=127.0.0.1
FREESWITCH_CLI_PORT=8022
FREESWITCH_PASSWORD=ClueCon
FREESWITCH_SIP_HOST=127.0.0.1
FREESWITCH_SIP_PORT=5060
RAPIDA_SIP_REDIS_DB=15
RAPIDA_SIP_LISTEN_ADDRESS=127.0.0.1
RAPIDA_SIP_EXTERNAL_IP=127.0.0.1
```

The FreeSWITCH dialplan should provide:

- `9197001`: answer the call and keep it alive long enough for Rapida to send
  BYE.
- `9197002`: answer only when `X-Rapida-Integration-Test` equals `headers-ok`.
- `9197003`: return ringing/progress without answer so Rapida can send CANCEL.
- `FREESWITCH_REGISTER_DID`: a FreeSWITCH directory user that accepts REGISTER
  with `FREESWITCH_SIP_USERNAME` / `FREESWITCH_SIP_PASSWORD`.
- `user/${FREESWITCH_REGISTER_DID}`: routes to the registered Rapida contact so
  FreeSWITCH can originate an inbound call through the registration binding.
- `9197010`: emulates Twilio Elastic SIP Trunk outbound by requiring
  `X-Twilio-AccountSid`, `X-Twilio-Elastic-Trunk-SID`, and
  `X-Rapida-Twilio-Trunk-Profile` headers before answering.
- `FREESWITCH_TWILIO_INBOUND_DID`: direct SIP destination used by FreeSWITCH to
  send a Twilio-style inbound INVITE into Rapida.
- `9197020`: responds with `486 Busy Here`.
- `9197021`: responds with `603 Decline`.
- `9197022`: rings without answer so Rapida reaches outbound no-answer timeout.
- `9197023`: responds with `480 Temporarily Unavailable`.
- `9197024`: responds with `488 Not Acceptable Here`.

The route integration tests also originate direct INVITEs to the dynamically
allocated Rapida SIP listener. They exercise the production route and vault
middleware with both `agent-<assistant-id>` and `did-<phone>` Request-URIs,
plain DID Request-URIs, custom SIP headers, immutable From/To URI capture,
authentication context, answer, RTP startup, and BYE teardown. Registration
coverage includes both valid digest authentication and invalid credentials. No
additional dialplan entry is needed for the direct calls.

Run:

```bash
go test -tags='sipintegration freeswitch' -count=1 ./api/assistant-api/sip/test/integration/freeswitch
```
