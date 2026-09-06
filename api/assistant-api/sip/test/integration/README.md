# SIP integration tests

Provider-specific SIP interoperability suites live in separate packages under
this directory. Each package owns its harness, scenarios, documentation, and
configuration fixtures so additional implementations such as Asterisk can be
added without mixing provider behavior.

Current packages:

- `freeswitch`: FreeSWITCH registration, inbound, outbound, routing, headers,
  media setup, failure mapping, and teardown coverage.
