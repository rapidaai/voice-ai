import { useCredential } from '@/hooks/use-credential';
import {
  AssistantConversation,
  AssistantConversationMessage,
} from '@rapidaai/react';
import { FC, useCallback, useContext, useEffect, useRef } from 'react';
import { AssistantChatContext } from '@/stores/assistant/assistant-chat.store';
import { useBoolean } from 'ahooks';
import { SectionLoader } from '@/app/components/ui/feedback';
import { Renew, Download, Chat } from '@carbon/icons-react';
import { GhostButton } from '@/app/components/ui/primitives';
import { Tag, DefinitionTooltip } from '@carbon/react';
import { EmptyState } from '@/app/components/ui/feedback';
import { getStatusMetric } from '@/utils/metadata';
import { CarbonStatusIndicator } from '@/app/components/ui/feedback';
import { toHumanReadableDateTime } from '@/utils/date';
import { AudioPlayer } from '@/app/components/ui/composites/audio-player';
import { getRoleVisual } from '@/app/pages/assistant/view/conversations/conversation-messages.helpers';

export const ConversationMessages: FC<{
  conversation: AssistantConversation;
  assistantId: string;
  conversationId: string;
}> = ({ conversation, conversationId, assistantId }) => {
  const [userId, token, projectId] = useCredential();
  const [loading, { setTrue: showLoader, setFalse: hideLoader }] =
    useBoolean(false);

  const {
    conversations,
    onGetConversationMessages,
    onChangeConversationMessages,
  } = useContext(AssistantChatContext);

  const ctrRef = useRef<HTMLDivElement>(null);

  const get = () => {
    showLoader();
    onGetConversationMessages(
      assistantId,
      conversationId,
      projectId,
      token,
      userId,
      () => hideLoader(),
      callbackOnGetConversationMessages,
    );
  };

  useEffect(() => {
    get();
  }, [assistantId, conversationId]);

  const callbackOnGetConversationMessages = useCallback(
    (msgs: Array<AssistantConversationMessage>) => {
      onChangeConversationMessages(msgs);
      scrollTo(ctrRef);
      hideLoader();
    },
    [],
  );

  const scrollTo = ref => {
    setTimeout(
      () =>
        ref.current?.scrollIntoView({ inline: 'center', behavior: 'smooth' }),
      777,
    );
  };

  function csvEscape(str: string): string {
    return `"${str.replace(/"/g, '""')}"`;
  }

  const downloadAllMessages = () => {
    const csvContent = [
      'role,message',
      ...conversations.flatMap((row: AssistantConversationMessage) => [
        `${row.getRole()},${csvEscape(row.getBody())}`,
      ]),
    ].join('\n');
    const url = URL.createObjectURL(
      new Blob([csvContent], { type: 'text/csv;charset=utf-8;' }),
    );
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', conversationId + '-messages.csv');
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  };

  if (loading) {
    return (
      <div className="h-full flex flex-col items-center justify-center">
        <SectionLoader />
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col h-full relative" ref={ctrRef}>
      {/* Recordings */}
      {conversation.getRecordingsList().map((x, idx) => (
        <div key={idx}>
          <div className="flex items-center h-8 px-4 border-b border-gray-200 dark:border-gray-800 bg-gray-50 dark:bg-gray-800/50 shrink-0">
            <p className="text-xs font-medium uppercase tracking-[0.08em] text-[var(--cds-text-secondary)]">
              Recording {idx + 1}
            </p>
          </div>
          <AudioPlayer recording={x} />
        </div>
      ))}

      <div className="flex items-center justify-between h-10 px-4 border-b border-gray-200 dark:border-gray-800 sticky top-0 z-[2] shrink-0 bg-white dark:bg-gray-900">
        <span className="text-xs font-medium uppercase tracking-wider text-[var(--cds-text-secondary)]">
          {conversations.length}{' '}
          {conversations.length === 1 ? 'message' : 'messages'}
        </span>
        <div className="flex items-center gap-2">
          <CarbonStatusIndicator
            state={getStatusMetric(conversation.getMetricsList())}
          />
          <div className="w-px h-4 bg-gray-200 dark:bg-gray-700" />
          <GhostButton
            size="sm"
            hasIconOnly
            renderIcon={Renew}
            iconDescription="Refresh"
            onClick={get}
          />
          <div className="w-px h-4 bg-gray-200 dark:bg-gray-700" />
          <GhostButton
            size="sm"
            hasIconOnly
            renderIcon={Download}
            iconDescription="Export CSV"
            onClick={downloadAllMessages}
          />
        </div>
      </div>

      {/* ── Empty state ── */}
      {conversations.length === 0 && (
        <div className="flex flex-1 items-center justify-center py-12">
          <EmptyState
            icon={Chat}
            title="No messages yet"
            subtitle="There are no messages for this session yet."
          />
        </div>
      )}

      <ul className="divide-y divide-gray-200 dark:divide-gray-800">
        {conversations.map((x, idx) => {
          const visual = getRoleVisual(x.getRole());

          return (
            <li key={idx} className="bg-white dark:bg-gray-900">
              <div className="w-full">
                <div className="bg-white dark:bg-gray-900">
                  <div className="px-4 py-2 flex items-center gap-2">
                    <p className="text-[11px] font-semibold uppercase tracking-[0.08em] text-gray-700 dark:text-gray-300">
                      {visual.label}
                    </p>
                    <p className="text-[10px] ml-auto text-[var(--cds-text-secondary)]">
                      {toHumanReadableDateTime(x.getCreateddate()!)}
                    </p>
                  </div>

                  <div className="px-4 pb-3 text-sm text-gray-900 dark:text-gray-100 leading-relaxed [&_:is([data-link],a:link,a:visited,a:hover,a:active)]:text-current [&_:is([data-link],a:link,a:visited,a:hover,a:active):hover]:underline [&_:is(code,div[data-lang])]:font-mono [&_:is(code,div[data-lang])]:bg-black/10 dark:[&_:is(code,div[data-lang])]:bg-white/10 [&_:is(code,div[data-lang])]:rounded-[2px] [&_is:(code)]:p-0.5 [&_div[data-lang]]:p-2 [&_div[data-lang]]:overflow-auto [&_:is(p,ul,ol,dl,table,blockquote,div[data-lang],h4,h5,h6,hr):not(:first-child)]:mt-2 [&_:is(p,ul,ol,dl,table,blockquote,div[data-lang],h3,h4,h5,h6,hr):not(:last-child)]:mb-2 [&_:is(ul,ol)]:pl-5 [&_ul]:list-disc [&_ol]:list-decimal [&_:is(strong,h1,h2,h3,h4,h5,h6)]:font-semibold whitespace-pre-wrap break-words">
                    {x.getBody() || '-'}
                  </div>
                  <div className="px-4 py-2 flex flex-wrap items-center gap-1.5">
                    {x
                      .getMetricsList()
                      ?.filter(m => m?.getName)
                      .map((m, mi) => {
                        const key = m.getName?.() || m.getName?.() || '';
                        const val = m.getValue?.() || '';
                        const displayVal = key.includes('latency')
                          ? `${val} ms`
                          : val;
                        return (
                          <DefinitionTooltip
                            key={mi}
                            definition={`${key}: ${val}`}
                            openOnHover
                          >
                            <Tag size="sm" type="gray" className="!cursor-help">
                              {key}: {displayVal}
                            </Tag>
                          </DefinitionTooltip>
                        );
                      })}
                    {x
                      .getMetadataList()
                      ?.filter(m => m?.getKey)
                      .map((m, mi) => {
                        const key = m.getKey();
                        const val = m.getValue();
                        const tagType =
                          key === 'language' || key === 'language_code'
                            ? 'warm-gray'
                            : 'cool-gray';
                        return (
                          <DefinitionTooltip
                            key={`md-${mi}`}
                            definition={`${key}: ${val}`}
                            openOnHover
                          >
                            <Tag
                              size="sm"
                              type={tagType}
                              className="!cursor-help"
                            >
                              {key}: {val}
                            </Tag>
                          </DefinitionTooltip>
                        );
                      })}
                  </div>
                </div>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
};
