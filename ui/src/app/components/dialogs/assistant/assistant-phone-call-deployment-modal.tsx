import type { AssistantPhoneDeployment } from '@rapidaai/react';
import type { ModalProps } from '@/app/components/ui/primitives';
import { RightSideModal } from '@/app/components/dialogs/shared';
import { CONFIG } from '@/configs';
import { CopyButton } from '@/app/components/ui/primitives';
import { InputHelper } from '@/app/components/ui/primitives';
import { YellowNoticeBlock } from '@/app/components/layout/container/message/notice-block';
import { ProviderPill } from '@/app/components/domain/pills/provider-model-pill';
import type { ReactNode } from 'react';
import { useMemo, useState } from 'react';
import { createPortal } from 'react-dom';
import {
  DeploymentRow,
  DeploymentSectionHeader,
} from '@/app/components/dialogs/shared';
import { Tabs } from '@/app/components/ui/primitives';
import { DeploymentAudioPanel } from '@/app/components/dialogs/shared/deployment-audio-panel';
import { DeploymentConfigurationPanel } from '@/app/components/dialogs/shared/deployment-configuration-panel';
import { IntegrationStep } from '@/app/components/domain/integration-document/integration-primitives';
import { RapidaCredentialCard } from '@/app/components/domain/cards/rapida-credential-card';

interface AssistantPhoneCallDeploymentDialogProps extends ModalProps {
  deployment: AssistantPhoneDeployment;
}

const INTERNAL_PHONE_OPTION_KEYS = new Set(['rapida.sip_inbound']);
const PHONE_HOST_PLACEHOLDER = '<your-phone-host>';
const AUDIO_SOCKET_HOST_PLACEHOLDER = '<your-audiosocket-host>';

export function AssistantPhoneCallDeploymentDialog(
  props: AssistantPhoneCallDeploymentDialogProps,
) {
  const [selectedTab, setSelectedTab] = useState(0);
  const providerName =
    props.deployment?.getPhoneprovidername()?.toLowerCase() || '';
  const assistantId = props.deployment?.getAssistantid();
  const mediaHost = CONFIG.connection.media;
  const sipHost = CONFIG.connection.sip;
  const socketHost = CONFIG.connection.socket;
  const phoneOptions = props.deployment?.getPhoneoptionsList() || [];
  const sipDid =
    phoneOptions.find(option => option.getKey() === 'phone')?.getValue() || '';
  const sipInboundEnabled =
    phoneOptions
      .find(option => option.getKey() === 'rapida.sip_inbound')
      ?.getValue() === 'true';

  const webhookUrl = `${mediaHost}/v1/talk/${providerName || '<provider>'}/call/${assistantId}?x-api-key={{PROJECT_CREDENTIAL_KEY}}`;
  const eventUrl = `${mediaHost}/v1/talk/${providerName || '<provider>'}/event/${assistantId}?x-api-key={{PROJECT_CREDENTIAL_KEY}}`;

  const modalContent = (
    <RightSideModal
      modalOpen={props.modalOpen}
      setModalOpen={props.setModalOpen}
      className="w-[580px] max-w-[calc(100vw-2rem)]"
      label="Phone Deployment"
      title={props.deployment.getId()}
    >
      <div className="relative flex flex-col flex-1 min-h-0">
        <Tabs
          tabs={['Setup', 'Configuration', 'Audio']}
          selectedIndex={selectedTab}
          onChange={setSelectedTab}
          contained
          fullWidth
          fill
          aria-label="Phone deployment tabs"
          panelClassName="overflow-auto !p-0"
        >
          <PhoneSetup
            providerName={providerName}
            sipHost={sipHost}
            mediaHost={mediaHost}
            socketHost={socketHost}
            assistantId={assistantId}
            sipDid={sipDid}
            sipInboundEnabled={sipInboundEnabled}
            webhookUrl={webhookUrl}
            eventUrl={eventUrl}
          />
          <DeploymentConfigurationPanel
            deployment={props.deployment}
            channelDetails={<TelephonyConfig deployment={props.deployment} />}
          />
          <DeploymentAudioPanel
            inputAudio={props.deployment.getInputaudio()}
            outputAudio={props.deployment.getOutputaudio()}
          />
        </Tabs>
      </div>
    </RightSideModal>
  );

  if (typeof document === 'undefined') return modalContent;

  return createPortal(modalContent, document.body);
}

function PhoneSetup({
  providerName,
  sipHost,
  mediaHost,
  socketHost,
  assistantId,
  sipDid,
  sipInboundEnabled,
  webhookUrl,
  eventUrl,
}: {
  providerName: string;
  sipHost?: string;
  mediaHost: string;
  socketHost?: string;
  assistantId: string;
  sipDid: string;
  sipInboundEnabled: boolean;
  webhookUrl: string;
  eventUrl: string;
}) {
  const instructionStep = providerName === 'sip' ? 1 : 2;
  const instructionTitle =
    providerName === 'sip'
      ? 'Route calls to the SIP endpoint'
      : providerName === 'asterisk'
        ? 'Connect your Asterisk server'
        : 'Register the provider webhooks';

  return (
    <div className="w-full pb-10">
      {providerName !== 'sip' && (
        <IntegrationStep
          number={1}
          title="Choose a project credential"
          description="Use a publishable project key to authenticate inbound call and event requests."
        >
          <RapidaCredentialCard />
        </IntegrationStep>
      )}
      <IntegrationStep
        number={instructionStep}
        title={instructionTitle}
        description="Apply the connection values below in your telephony provider or PBX."
      >
        <div className="-mx-4 -mb-5 divide-y divide-gray-200 dark:divide-gray-800">
          {providerName === 'sip' && (
            <SipIntegrationInstructions
              sipHost={sipHost}
              assistantId={assistantId}
              did={sipDid}
              inboundRegistrationEnabled={sipInboundEnabled}
            />
          )}
          {providerName === 'asterisk' && (
            <AsteriskIntegrationInstructions
              mediaHost={mediaHost}
              audioSocketHost={socketHost}
              assistantId={assistantId}
            />
          )}
          {providerName !== 'sip' && providerName !== 'asterisk' && (
            <>
              <CodeRow label="Inbound webhook URL" value={webhookUrl}>
                <InputHelper>
                  You can add additional assistant arguments as query
                  parameters, for example{' '}
                  <code className="text-red-600">?name=your-name</code>.
                </InputHelper>
              </CodeRow>
              <CodeRow label="Call status event URL" value={eventUrl} />
            </>
          )}
        </div>
      </IntegrationStep>
    </div>
  );
}

/* -------------------------------------------------------------------------- */
/*  Shared row primitives                                                      */
/* -------------------------------------------------------------------------- */

const Row = DeploymentRow;
const SectionHeader = DeploymentSectionHeader;

function TelephonyConfig({
  deployment,
}: {
  deployment: AssistantPhoneDeployment;
}) {
  const options = (deployment.getPhoneoptionsList() || []).filter(
    detail =>
      detail.getKey() &&
      detail.getValue() &&
      !INTERNAL_PHONE_OPTION_KEYS.has(detail.getKey()),
  );

  return (
    <>
      <SectionHeader label="Telephony" />
      <Row label="Provider">
        <ProviderPill provider={deployment.getPhoneprovidername()} />
      </Row>
      {options.length > 0 ? (
        options.map((detail, index) => (
          <Row key={`phone-option-${index}`} label={detail.getKey()}>
            <span className="text-sm font-mono text-gray-900 dark:text-gray-100 truncate max-w-[200px] text-right">
              {detail.getValue()}
            </span>
            <CopyButton className="h-6 w-6 shrink-0">
              {detail.getValue()}
            </CopyButton>
          </Row>
        ))
      ) : (
        <div className="px-4 py-3">
          <YellowNoticeBlock>
            No telephony options configured.
          </YellowNoticeBlock>
        </div>
      )}
    </>
  );
}

function CodeRow({
  label,
  value,
  children,
}: {
  label: string;
  value: string;
  children?: ReactNode;
}) {
  return (
    <div>
      <SectionHeader label={label} />
      <div className="px-4 py-3 space-y-2">
        <div className="flex items-center gap-2">
          <code className="flex-1 dark:bg-gray-950 bg-gray-100 px-3 py-2 font-mono text-xs min-w-0 overflow-hidden break-all">
            {value}
          </code>
          <CopyButton className="h-7 w-7 shrink-0 border border-gray-200 dark:border-gray-800">
            {value}
          </CopyButton>
        </div>
        {children}
      </div>
    </div>
  );
}

function CodeBlock({
  label,
  code,
  helper,
}: {
  label: string;
  code: string;
  helper?: ReactNode;
}) {
  return (
    <div>
      <SectionHeader label={label} />
      <div className="px-4 py-3 space-y-2">
        <div className="relative">
          <pre className="dark:bg-gray-950 bg-gray-100 px-3 py-2 font-mono text-xs overflow-auto">
            {code}
          </pre>
          <div className="absolute top-1 right-1">
            <CopyButton className="h-6 w-6 bg-gray-200 dark:bg-gray-800">
              {code}
            </CopyButton>
          </div>
        </div>
        {helper}
      </div>
    </div>
  );
}

function SipRouteRow({
  route,
  uri,
  description,
}: {
  route: string;
  uri: string;
  description: string;
}) {
  return (
    <div className="grid grid-cols-[150px_minmax(0,1fr)_auto] gap-2 px-4 py-3 items-start">
      <span className="font-mono text-xs text-gray-700 dark:text-gray-300 pt-2">
        {route}
      </span>
      <div className="min-w-0 space-y-1">
        <code className="block dark:bg-gray-950 bg-gray-100 px-3 py-2 font-mono text-xs break-all">
          {uri}
        </code>
        <InputHelper>{description}</InputHelper>
      </div>
      <CopyButton className="h-7 w-7 shrink-0 border border-gray-200 dark:border-gray-800">
        {uri}
      </CopyButton>
    </div>
  );
}

/* -------------------------------------------------------------------------- */
/*  SIP Provider Integration Instructions                                      */
/* -------------------------------------------------------------------------- */

function SipIntegrationInstructions({
  sipHost,
  assistantId,
  did,
  inboundRegistrationEnabled,
}: {
  sipHost?: string;
  assistantId: string;
  did?: string;
  inboundRegistrationEnabled?: boolean;
}) {
  const routeHost = sipHost || `${PHONE_HOST_PLACEHOLDER}:5060`;
  const sipEndpoint = `sip:${routeHost}`;
  const sipPort = routeHost.includes(':')
    ? routeHost.split(':').pop() || '5060'
    : '5060';
  const assistantRoute = `sip:agent-${assistantId || '{ASSISTANT_ID}'}@${routeHost}`;
  const didRoute = `sip:did-${did || '{DID}'}@${routeHost}`;
  const rawDidRoute = `sip:${did || '{DID}'}@${routeHost}`;

  return (
    <>
      <CodeRow
        label={
          inboundRegistrationEnabled ? 'Default SIP DNS' : 'SIP Server Endpoint'
        }
        value={inboundRegistrationEnabled ? routeHost : sipEndpoint}
      >
        <InputHelper>
          {inboundRegistrationEnabled
            ? 'Inbound registration is enabled. Use this DNS as the default SIP target.'
            : 'Point your SIP trunk / PBX outbound proxy to this address. The service accepts SIP INVITE and establishes an RTP media session directly.'}
        </InputHelper>
      </CodeRow>

      <SectionHeader label="Inbound SIP Routes" />
      <SipRouteRow
        route="agent-{assistantID}"
        uri={assistantRoute}
        description="Route by numeric assistant ID."
      />
      <SipRouteRow
        route="did-{did}"
        uri={didRoute}
        description="Route by active SIP phone deployment phone value."
      />
      <SipRouteRow
        route="{did}"
        uri={rawDidRoute}
        description="Same DID lookup without the did- prefix."
      />

      <SectionHeader label="SIP Configuration" />
      <Row label="Transport">
        <span className="text-sm text-gray-900 dark:text-gray-100">
          UDP, TCP, or TLS
        </span>
      </Row>
      <Row label="Port">
        <span className="text-sm font-mono text-gray-900 dark:text-gray-100">
          {sipPort}
        </span>
      </Row>
      <Row label="Codec">
        <span className="text-sm text-gray-900 dark:text-gray-100 text-right">
          G.711 μ-law / A-law + DTMF
        </span>
      </Row>
      <Row label="Routing">
        <span className="text-sm text-gray-900 dark:text-gray-100 text-right">
          {'Route user: agent-{assistantID}, did-{did}, or raw DID'}
        </span>
      </Row>
      <Row label="Media">
        <span className="text-sm text-gray-900 dark:text-gray-100">
          RTP (direct)
        </span>
      </Row>

      <CodeBlock
        label="PBX Dial Plan — FreeSWITCH"
        code={`<extension name="voice-ai">
  <condition field="destination_number" expression="^(\\d+)$">
    <action application="bridge"
            data="sofia/external/sip:did-+\${destination_number}@${routeHost}"/>
  </condition>
</extension>`}
      />

      <CodeBlock
        label="PBX Dial Plan — Asterisk (pjsip.conf + extensions.conf)"
        code={`; pjsip.conf
[voice-ai-trunk]
type = endpoint
transport = transport-udp
context = from-voice-ai
aors = voice-ai-trunk-aor

[voice-ai-trunk-aor]
type = aor
contact = ${sipEndpoint}

; extensions.conf
[voice-ai-outbound]
exten => _X.,1,Dial(PJSIP/did-+\${EXTEN}@voice-ai-trunk)`}
      />
    </>
  );
}

/* -------------------------------------------------------------------------- */
/*  Asterisk Provider Integration Instructions                                 */
/* -------------------------------------------------------------------------- */

function AsteriskIntegrationInstructions({
  mediaHost,
  audioSocketHost,
  assistantId,
}: {
  mediaHost: string;
  audioSocketHost?: string;
  assistantId: string;
}) {
  const mediaHostname = useMemo(() => {
    try {
      return new URL(mediaHost).hostname;
    } catch {
      return PHONE_HOST_PLACEHOLDER;
    }
  }, [mediaHost]);

  const audioSocketHostPart = useMemo(() => {
    if (!audioSocketHost) return AUDIO_SOCKET_HOST_PLACEHOLDER;
    return audioSocketHost.split(':')[0] || AUDIO_SOCKET_HOST_PLACEHOLDER;
  }, [audioSocketHost]);

  const audioSocketPort = useMemo(() => {
    if (!audioSocketHost) return '4573';
    const parts = audioSocketHost.split(':');
    return parts.length > 1 ? parts[1] : '4573';
  }, [audioSocketHost]);

  const webhookUrl = `https://${mediaHostname}/v1/talk/asterisk/call/${assistantId || '{ASSISTANT_ID}'}?from=\${CALLERID(num)}&x-api-key={{PROJECT_CREDENTIAL_KEY}}`;

  return (
    <>
      <CodeRow
        label="WebSocket — Endpoint"
        value={`wss://${mediaHostname}/v1/talk/asterisk/ctx/{contextId}`}
      />

      <CodeBlock
        label="WebSocket — Dialplan (extensions.conf)"
        code={`[voice-ai-inbound-ws]
exten => _X.,1,Answer()
 same => n,Set(CTX=\${CURL(${webhookUrl})})
 same => n,GotoIf($["\${CTX}" = ""]?error)
 same => n,WebSocket(wss://${mediaHostname}/v1/talk/asterisk/ctx/\${CTX})
 same => n,Hangup()
 same => n(error),Playback(an-error-has-occurred)
 same => n,Hangup()`}
        helper={
          <InputHelper>
            Requires <code>chan_websocket.so</code> (Asterisk 20+). WSS port 443
            — ideal for cloud / NAT traversal.
          </InputHelper>
        }
      />

      <CodeRow label="AudioSocket — Endpoint" value={audioSocketHost ?? ''} />

      <CodeBlock
        label="AudioSocket — Dialplan (extensions.conf)"
        code={`[voice-ai-inbound]
exten => _X.,1,Answer()
 same => n,Set(CHANNEL(audioreadformat)=slin)
 same => n,Set(CHANNEL(audiowriteformat)=slin)
 same => n,Set(CTX=\${CURL(${webhookUrl})})
 same => n,GotoIf($["\${CTX}" = ""]?error)
 same => n,AudioSocket(\${CTX},${audioSocketHostPart}:${audioSocketPort})
 same => n,Hangup()
 same => n(error),Playback(an-error-has-occurred)
 same => n,Hangup()`}
        helper={
          <InputHelper>
            Requires <code>res_audiosocket.so</code> (Asterisk 16+). Raw TCP
            port {audioSocketPort} — SLIN 16-bit 8 kHz. Best for LAN / private
            network.
          </InputHelper>
        }
      />
    </>
  );
}
