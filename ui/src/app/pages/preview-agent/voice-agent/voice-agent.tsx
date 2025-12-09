import React, { FC, memo, useEffect, useRef, useState } from 'react';
import {
  VoiceAgent as VI,
  ConnectionConfig,
  AgentConfig,
  AgentCallback,
  Assistant,
  Variable,
} from '@rapidaai/react';
import { MessagingAction } from '@/app/pages/preview-agent/voice-agent/actions';
import { ConversationMessages } from '@/app/pages/preview-agent/voice-agent/text/conversations';
import { cn } from '@/utils';
import { QuickSuggestion } from '@/app/pages/preview-agent/voice-agent/text/suggestions';
import { Tab } from '@/app/components/tab';
import { YellowNoticeBlock } from '@/app/components/container/message/notice-block';
import {
  JsonTextarea,
  NumberTextarea,
  ParagraphTextarea,
  TextTextarea,
  UrlTextarea,
} from '@/app/components/form/textarea';
import { InputVarForm } from '@/app/pages/endpoint/view/try-playground/experiment-prompt/components/input-var-form';
import { InputVarType } from '@/models/common';
import { InputGroup } from '@/app/components/input-group';
import { Check, CheckCheck, ExternalLink, Info } from 'lucide-react';
import { useRapidaStore } from '@/hooks';

export const VoiceAgent: FC<{
  connectConfig: ConnectionConfig;
  agentConfig: AgentConfig;
  agentCallback?: AgentCallback;
}> = ({ connectConfig, agentConfig, agentCallback }) => {
  const voiceAgentContextValue = React.useMemo(() => {
    return new VI(connectConfig, agentConfig, agentCallback);
  }, [connectConfig, agentConfig, agentCallback]);
  const [assistant, setAssistant] = useState<Assistant | null>(null);
  const { showLoader, hideLoader } = useRapidaStore();

  useEffect(() => {
    showLoader('block');
    voiceAgentContextValue
      .getAssistant()
      .then(ex => {
        hideLoader();
        if (ex.getSuccess()) {
          setAssistant(ex.getData()!);
        }
      })
      .catch();
  }, [voiceAgentContextValue]);

  return (
    <div className="h-dvh flex p-8 text-sm/6 w-full">
      <div className="relative overflow-hidden h-full mx-auto w-2/3 dark:bg-gray-950/50 border rounded-[2px]">
        {!assistant?.getDebuggerdeployment()?.hasInputaudio() && (
          <div className="bg-white dark:bg-gray-950 z-10 absolute top-0 left-0 right-0 ">
            <YellowNoticeBlock className="flex items-center justify-between">
              <Info className="shrink-0 w-4 h-4" />
              <div className="ms-3 text-sm font-medium">
                Voice functionality is currently disabled. Please enable it to
                enjoy a voice experience with your assistant.
              </div>
              <a
                target="_blank"
                href={`/deployment/assistant/${assistant?.getId()}/manage/deployment/debugger`}
                className="h-7 flex items-center font-medium hover:underline ml-auto text-yellow-600"
                rel="noreferrer"
              >
                Enable voice
                <ExternalLink
                  className="shrink-0 w-4 h-4 ml-1.5"
                  strokeWidth={1.5}
                />
              </a>
            </YellowNoticeBlock>
          </div>
        )}
        <div className="h-full flex flex-row flex-nowrap items-stretch">
          <div className="flex flex-col grow min-w-0 flex-1">
            <div className="flex flex-col justify-center grow min-h-0 px-4">
              <div
                className={cn('max-h-full flex gap-2 overflow-y-auto flex-col')}
              >
                <div className="flex flex-col items-center py-20 justify-center px-4">
                  <div className="flex w-full flex-col items-start gap-1 ">
                    <span className="text-3xl font-semibold">Hello,</span>
                    <span className="text-xl font-medium opacity-80">
                      How can I help you today?
                    </span>
                  </div>
                  <div className="flex w-full flex-wrap items-center gap-2 mt-4">
                    {[
                      'What can you do?',
                      'Can you connect me to a human?',
                      'How can you help me?',
                      'Can you connect me to a human?',
                      'Do you remember our past conversations?',
                    ].map((suggestion, idx) => {
                      return (
                        <QuickSuggestion
                          key={`suggestion-${idx}`}
                          suggestion={suggestion}
                          onClick={async () => {
                            await voiceAgentContextValue?.onSendText(
                              suggestion,
                            );
                          }}
                        />
                      );
                    })}
                  </div>
                </div>
                <ConversationMessages vag={voiceAgentContextValue} />
              </div>
            </div>
            <MessagingAction
              assistant={assistant}
              placeholder="How can I help you?"
              voiceAgent={voiceAgentContextValue}
            />
          </div>
        </div>
      </div>
      <div className="shrink-0 flex flex-col overflow-auto border border-l-0 w-1/3">
        <VoiceAgentDebugger
          voiceAgent={voiceAgentContextValue}
          assistant={assistant}
        />
      </div>
    </div>
  );
};

export const VoiceAgentDebugger: FC<{
  voiceAgent: VI;
  assistant: Assistant | null;
}> = memo(({ voiceAgent, assistant }) => {
  type EventData = {
    type: string;
    payload: any; // Adjust `any` to a more specific type if possible
  };
  const ctrRef = useRef<HTMLDivElement>(null);

  const [variables, setVaribales] = useState<Variable[]>([]);
  const [events, setEvents] = useState<EventData[]>([]);
  const callbackRegisteredRef = useRef(false); // Persistent across renders

  /**
   *
   * @param ref
   */
  const scrollTo = ref => {
    setTimeout(
      () =>
        ref.current?.scrollIntoView({ inline: 'center', behavior: 'smooth' }),
      777,
    );
  };

  //   on change of message to scroll
  useEffect(() => {
    scrollTo(ctrRef);
  }, [JSON.stringify(events)]);

  const onChangeArgument = (k: string, vl: string) => {
    voiceAgent.agentConfiguration.addArgument(k, vl);
  };

  useEffect(() => {
    if (assistant) {
      let pmtVar = assistant
        ?.getAssistantprovidermodel()
        ?.getTemplate()
        ?.getPromptvariablesList();
      if (pmtVar) {
        pmtVar.forEach(v => {
          if (v.getDefaultvalue()) {
            voiceAgent.agentConfiguration.addArgument(
              v.getName(),
              v.getDefaultvalue(),
            );
          }
        });
        setVaribales(pmtVar);
      }
    }

    if (!callbackRegisteredRef.current) {
      callbackRegisteredRef.current = true; // Mark the callback as registered
      voiceAgent.registerCallback({
        onAction(arg) {
          setEvents(prevEvents => [
            ...prevEvents,
            { type: 'action', payload: arg },
          ]);
        },
        onConfiguration: args => {
          setEvents(prevEvents => [
            ...prevEvents,
            { type: 'configuration', payload: args },
          ]);
        },
        onUserMessage: args => {
          setEvents(prevEvents => [
            ...prevEvents,
            { type: 'userMessage', payload: args },
          ]);
        },
        onAssistantMessage: args => {
          if (args?.messageText)
            setEvents(prevEvents => [
              ...prevEvents,
              { type: 'assistantMessage', payload: args },
            ]);
        },
        onInterrupt: args => {
          setEvents(prevEvents => [
            ...prevEvents,
            { type: 'interrupt', payload: args },
          ]);
        },
        onMessage: args => {
          setEvents(prevEvents => [
            ...prevEvents,
            { type: 'message', payload: args },
          ]);
        },
      });
    }
  }, [voiceAgent, JSON.stringify(assistant)]);

  return (
    <Tab
      strict
      active="assistant"
      className={cn(
        'sticky top-0 z-1',
        'border-b dark:bg-gray-900 bg-white dark:border-gray-800',
      )}
      tabs={[
        {
          label: 'assistant',
          element: (
            <>
              {assistant && (
                <div className="flex flex-col w-full h-full flex-1 grow">
                  <div className="p-4 text-sm leading-normal">
                    <div className="flex flex-row justify-between items-center text-sm uppercase tracking-wider">
                      <h3>Name</h3>
                    </div>
                    <div className="py-2 text-sm leading-normal">
                      {assistant.getName()}
                    </div>
                    {assistant.getDescription() && (
                      <>
                        <div className="flex mt-4 flex-row justify-between items-center text-sm uppercase tracking-wider">
                          <h3>Description</h3>
                        </div>
                        <div className="py-2 text-sm leading-normal">
                          {assistant.getDescription()}
                        </div>
                      </>
                    )}
                  </div>
                  <InputGroup title="Arguments" childClass="!p-0">
                    {variables.length > 0 ? (
                      <div className="text-sm leading-normal">
                        {variables.map((x, idx) => {
                          return (
                            <InputVarForm
                              key={idx}
                              var={x}
                              className="bg-light-background"
                            >
                              {x.getType() === InputVarType.textInput && (
                                <TextTextarea
                                  readOnly={voiceAgent.isConnected}
                                  id={x.getName()}
                                  defaultValue={x.getDefaultvalue()}
                                  onChange={(
                                    e: React.ChangeEvent<HTMLTextAreaElement>,
                                  ) =>
                                    onChangeArgument(
                                      x.getName(),
                                      e.target.value,
                                    )
                                  }
                                />
                              )}
                              {x.getType() === InputVarType.paragraph && (
                                <ParagraphTextarea
                                  id={x.getName()}
                                  readOnly={voiceAgent.isConnected}
                                  defaultValue={x.getDefaultvalue()}
                                  onChange={(
                                    e: React.ChangeEvent<HTMLTextAreaElement>,
                                  ) =>
                                    onChangeArgument(
                                      x.getName(),
                                      e.target.value,
                                    )
                                  }
                                />
                              )}
                              {x.getType() === InputVarType.number && (
                                <NumberTextarea
                                  readOnly={voiceAgent.isConnected}
                                  id={x.getName()}
                                  defaultValue={x.getDefaultvalue()}
                                  onChange={(
                                    e: React.ChangeEvent<HTMLTextAreaElement>,
                                  ) =>
                                    onChangeArgument(
                                      x.getName(),
                                      e.target.value,
                                    )
                                  }
                                />
                              )}
                              {x.getType() === InputVarType.json && (
                                <JsonTextarea
                                  readOnly={voiceAgent.isConnected}
                                  id={x.getName()}
                                  defaultValue={x.getDefaultvalue()}
                                  onChange={(
                                    e: React.ChangeEvent<HTMLTextAreaElement>,
                                  ) =>
                                    onChangeArgument(
                                      x.getName(),
                                      e.target.value,
                                    )
                                  }
                                />
                              )}
                              {x.getType() === InputVarType.url && (
                                <UrlTextarea
                                  readOnly={voiceAgent.isConnected}
                                  id={x.getName()}
                                  defaultValue={x.getDefaultvalue()}
                                  onChange={(
                                    e: React.ChangeEvent<HTMLTextAreaElement>,
                                  ) =>
                                    onChangeArgument(
                                      x.getName(),
                                      e.target.value,
                                    )
                                  }
                                />
                              )}
                            </InputVarForm>
                          );
                        })}
                      </div>
                    ) : (
                      <YellowNoticeBlock>
                        Assistant do not accept any arguments.
                      </YellowNoticeBlock>
                    )}
                  </InputGroup>
                  <InputGroup title="Deployment" childClass="p-3 text-muted">
                    <div className="space-y-4">
                      <div className="flex justify-between">
                        <div className="text-sm uppercase tracking-wider">
                          Input Mode
                        </div>
                        <div className="font-medium">
                          Text
                          {assistant
                            ?.getDebuggerdeployment()
                            ?.getInputaudio() && ', Audio'}
                        </div>
                      </div>
                      <div className="flex justify-between">
                        <div className="text-sm uppercase tracking-wider">
                          Output Mode
                        </div>
                        <div className="font-medium">
                          Text
                          {assistant
                            ?.getDebuggerdeployment()
                            ?.getOutputaudio() && ', Audio'}
                        </div>
                      </div>
                      {/*  */}
                      {assistant
                        .getDebuggerdeployment()
                        ?.getInputaudio()
                        ?.getAudiooptionsList() &&
                        assistant
                          .getDebuggerdeployment()
                          ?.getInputaudio()
                          ?.getAudiooptionsList().length! > 0 && (
                          <div className="space-y-4">
                            <div className="flex justify-between">
                              <div className="text-muted uppercase">
                                Listen.Provider
                              </div>
                              <div className="font-medium mt-1 underline underline-offset-4">
                                {assistant
                                  .getDebuggerdeployment()
                                  ?.getInputaudio()
                                  ?.getAudioprovider()}
                              </div>
                            </div>
                            {assistant
                              .getDebuggerdeployment()
                              ?.getInputaudio()
                              ?.getAudiooptionsList()
                              .filter(d => d.getValue())
                              .filter(d => d.getKey().startsWith('listen.'))
                              .map((detail, index) => (
                                <div
                                  className="flex justify-between"
                                  key={index}
                                >
                                  <div className="text-sm uppercase tracking-wider">
                                    {detail.getKey()}
                                  </div>
                                  <div className="font-medium">
                                    {detail.getValue()}
                                  </div>
                                </div>
                              ))}
                          </div>
                        )}
                      {assistant
                        .getDebuggerdeployment()
                        ?.getInputaudio()
                        ?.getAudiooptionsList() &&
                        assistant
                          .getDebuggerdeployment()
                          ?.getOutputaudio()
                          ?.getAudiooptionsList().length! > 0 && (
                          <div className="space-y-4">
                            <div className="flex justify-between">
                              <div className="text-sm uppercase tracking-wider">
                                Listen.Provider
                              </div>
                              <div className="font-medium mt-1 underline underline-offset-4">
                                {assistant
                                  .getDebuggerdeployment()
                                  ?.getOutputaudio()
                                  ?.getAudioprovider()}
                              </div>
                            </div>
                            {assistant
                              .getDebuggerdeployment()
                              ?.getOutputaudio()
                              ?.getAudiooptionsList()
                              .filter(d => d.getValue())
                              .filter(d => d.getKey().startsWith('speak.'))
                              .map((detail, index) => (
                                <div
                                  key={index}
                                  className="flex justify-between"
                                >
                                  <div className="text-sm uppercase tracking-wider">
                                    {detail.getKey()}
                                  </div>
                                  <div className="font-medium">
                                    {detail.getValue()}
                                  </div>
                                </div>
                              ))}
                          </div>
                        )}
                    </div>
                  </InputGroup>
                </div>
              )}
            </>
          ),
        },
        {
          label: 'events',
          element: (
            <div className="flex flex-col flex-1 divide-y w-full ">
              {events.length === 0 ? (
                <YellowNoticeBlock>
                  No events have been recorded yet. Start interacting to see
                  updates here.
                </YellowNoticeBlock>
              ) : (
                events.map((event, idx) => {
                  if (event.type === 'action') {
                    return (
                      <div
                        key={idx}
                        className="p-2 text-xs flex flex-col space-y-1"
                      >
                        <span>
                          <strong>Rapida</strong>
                        </span>
                        <div>
                          Action Triggered with data{' '}
                          {JSON.stringify(event.payload)}
                        </div>
                      </div>
                    );
                  } else if (event.type === 'configuration') {
                    const { assistantconversationid, assistant } =
                      event.payload;
                    return (
                      <div
                        key={idx}
                        className="p-2 text-xs flex flex-col space-y-1"
                      >
                        <span>
                          <strong>Rapida</strong>
                        </span>
                        <div>
                          Connected with Assistant {assistant.assistantid}{' '}
                          version {assistant.version}
                        </div>
                        <div>
                          Conversation created with ID {assistantconversationid}
                        </div>
                      </div>
                    );
                  } else if (event.type === 'userMessage') {
                    return (
                      <div
                        key={idx}
                        className="p-2 text-xs flex flex-col space-y-1"
                      >
                        <div className="flex space-x-1 items-center">
                          <strong>User</strong>
                          <span className="inline-block">
                            {event.payload.completed ? (
                              <CheckCheck
                                className="w-3 h-3 text-green-600"
                                strokeWidth={1.5}
                              />
                            ) : (
                              <Check className="w-3 h-3" strokeWidth={1.5} />
                            )}
                          </span>
                        </div>
                        <div>{JSON.stringify(event.payload.messageText)}</div>{' '}
                      </div>
                    );
                  } else if (event.type === 'assistantMessage') {
                    return (
                      <div
                        key={idx}
                        className="p-2 text-xs flex flex-col space-y-1"
                      >
                        <div className="flex space-x-1 items-center">
                          <strong>Assistant</strong>
                          <span className="inline-block">
                            {event.payload.completed ? (
                              <CheckCheck
                                className="w-3 h-3 text-green-600"
                                strokeWidth={1.5}
                              />
                            ) : (
                              <Check className="w-3 h-3" strokeWidth={1.5} />
                            )}
                          </span>
                        </div>
                        <div>{JSON.stringify(event.payload.messageText)}</div>{' '}
                        {/* Show specific field */}
                      </div>
                    );
                  } else if (event.type === 'message') {
                    return (
                      <div
                        key={idx}
                        className="p-2 text-xs flex flex-col space-y-1"
                      >
                        <span>
                          <strong>Rapida</strong>
                        </span>
                        <div>
                          Trun completed for message{' '}
                          {JSON.stringify(event.payload.messageid)}
                        </div>
                      </div>
                    );
                  } else if (event.type === 'interrupt') {
                    return (
                      <div
                        key={idx}
                        className="p-2 text-xs flex flex-col space-y-1"
                      >
                        <span>
                          <strong>Rapida</strong>
                        </span>
                        <div>
                          {event.payload.type === 1
                            ? 'Interruption detected from client audio [vad]'
                            : 'Interruption from STT provider [word]'}
                        </div>
                      </div>
                    );
                  } else {
                    // Default case for other event types
                    return (
                      <div
                        key={idx}
                        className="p-2 text-xs flex flex-col space-y-1"
                      >
                        <span>
                          <strong>System</strong>
                        </span>
                        <code>{JSON.stringify(event.payload)}</code>
                      </div>
                    );
                  }
                })
              )}
              <div ref={ctrRef} />
            </div>
          ),
        },
      ]}
    />
  );
});
