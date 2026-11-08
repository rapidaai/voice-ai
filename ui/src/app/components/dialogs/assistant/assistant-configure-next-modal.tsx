import { ModalProps } from '@/app/components/ui/primitives';
import {
  Modal,
  ModalHeader,
  ModalBody,
  ModalFooter,
} from '@/app/components/ui/primitives';
import { PrimaryButton, SecondaryButton } from '@/app/components/ui/primitives';
import {
  Phone,
  Code,
  Globe,
  Debug,
  ChartLine,
  Webhook,
  ArrowRight,
  Launch,
} from '@carbon/icons-react';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { Assistant } from '@rapidaai/react';
import { ClickableTile } from '@carbon/react';

interface ConfigureAssistantNextDialogProps extends ModalProps {
  assistant: Assistant;
}

const makeDeploymentOptions = (
  assistantId: string,
  nav: ReturnType<typeof useGlobalNavigation>,
) => [
  {
    icon: Phone,
    title: 'Phone call',
    description:
      'Enable voice conversations over inbound and outbound phone calls.',
    onClick: () => nav.goToConfigureCall(assistantId),
  },
  {
    icon: Code,
    title: 'API',
    description: 'Integrate into your application using our REST API or SDKs.',
    onClick: () => nav.goToConfigureApi(assistantId),
  },
  {
    icon: Globe,
    title: 'Web Widget',
    description:
      'Embed on your website to handle text and voice customer queries.',
    onClick: () => nav.goToConfigureWeb(assistantId),
  },
  {
    icon: Debug,
    title: 'Debugger',
    description: 'Deploy to a sandbox environment for testing and debugging.',
    onClick: () => nav.goToConfigureDebugger(assistantId),
  },
];

const makeAutomationOptions = (
  assistantId: string,
  nav: ReturnType<typeof useGlobalNavigation>,
) => [
  {
    icon: ChartLine,
    title: 'Post-conversation analysis',
    description:
      'Gain insights from every interaction — transcripts, sentiment, quality analysis.',
    onClick: () => nav.goToCreateAssistantAnalysis(assistantId),
  },
  {
    icon: Webhook,
    title: 'Webhooks',
    description:
      'Trigger external events on key actions: conversation start/end, human escalation.',
    onClick: () => nav.goToCreateAssistantWebhook(assistantId),
  },
];

export function ConfigureAssistantNextDialog({
  assistant,
  modalOpen,
  setModalOpen,
}: ConfigureAssistantNextDialogProps) {
  const nav = useGlobalNavigation();
  const assistantId = assistant.getId();

  const deploymentOptions = makeDeploymentOptions(assistantId, nav);
  const automationOptions = makeAutomationOptions(assistantId, nav);

  return (
    <Modal
      open={modalOpen}
      onClose={() => setModalOpen(false)}
      size="lg"
      containerClassName="!w-[800px] !max-w-[800px]"
    >
      <ModalHeader
        label="Next steps"
        title="Assistant created"
        onClose={() => setModalOpen(false)}
      />

      <ModalBody hasScrollingContent>
        {/* Deployment channels */}
        <h2 className="text-sm font-semibold uppercase tracking-wider text-muted mb-3">
          Deployment Channels
        </h2>
        <div className="grid grid-cols-2 gap-4 mb-6">
          {deploymentOptions.map(option => {
            const Icon = option.icon;
            return (
              <ClickableTile
                key={option.title}
                className="!rounded-none !p-4 !flex !flex-col"
                onClick={option.onClick}
              >
                <Icon size={24} className="text-muted mb-3" />
                <p className="text-sm font-semibold mb-1">{option.title}</p>
                <p className="text-xs text-muted leading-relaxed flex-1">
                  {option.description}
                </p>
                <span className="inline-flex items-center gap-1 text-xs font-medium text-primary mt-3">
                  Configure <ArrowRight size={12} />
                </span>
              </ClickableTile>
            );
          })}
        </div>

        {/* Automation */}
        <h2 className="text-sm font-semibold uppercase tracking-wider text-muted mb-3">
          Automation & Integrations
        </h2>
        <div className="flex flex-col gap-2">
          {automationOptions.map(option => {
            const Icon = option.icon;
            return (
              <ClickableTile
                key={option.title}
                className="!rounded-none !p-4 !flex !flex-row !items-center !gap-4"
                onClick={option.onClick}
              >
                <Icon size={24} className="text-muted shrink-0" />
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-semibold">{option.title}</p>
                  <p className="text-xs text-muted mt-0.5">
                    {option.description}
                  </p>
                </div>
                <ArrowRight size={16} className="text-primary shrink-0" />
              </ClickableTile>
            );
          })}
        </div>
      </ModalBody>

      <ModalFooter>
        <SecondaryButton size="lg" onClick={() => setModalOpen(false)}>
          Do this later
        </SecondaryButton>
        <PrimaryButton
          size="lg"
          renderIcon={Launch}
          onClick={() => nav.goToAssistantPreview(assistantId)}
        >
          Preview assistant
        </PrimaryButton>
      </ModalFooter>
    </Modal>
  );
}
