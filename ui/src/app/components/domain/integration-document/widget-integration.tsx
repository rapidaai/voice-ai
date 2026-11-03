import { Button } from '@carbon/react';
import { Launch } from '@carbon/icons-react';
import type { ReactNode } from 'react';
import { RapidaCredentialCard } from '@/app/components/domain/cards/rapida-credential-card';
import {
  IntegrationCodeBlock,
  IntegrationStep,
} from './integration-primitives';
import { useDocumentationUrl } from '@/theme/documentation-url';
import { CONFIG } from '@/configs';

interface WidgetIntegrationProps {
  assistantId: string;
  credentialCard?: ReactNode;
  showDocumentationStep?: boolean;
}

export function WidgetIntegration({
  assistantId,
  credentialCard,
  showDocumentationStep = true,
}: WidgetIntegrationProps) {
  const documentationUrl = useDocumentationUrl();
  const resolvedAssistantId = assistantId || 'YOUR_ASSISTANT_ID';
  const credentials = credentialCard ?? <RapidaCredentialCard />;

  return (
    <div className="w-full pb-10">
      <IntegrationStep
        number={1}
        title="Add the widget script"
        description="Load the widget bundle once on every page where the assistant should be available."
      >
        <IntegrationCodeBlock
          language="html"
          code={`<script src="${CONFIG.deployment.webWidgetScriptUrl}" defer></script>`}
        />
      </IntegrationStep>

      <IntegrationStep
        number={2}
        title="Choose a project credential"
        description="Copy a publishable project key and replace PROJECT_KEY in the initialization snippet."
      >
        {credentials}
      </IntegrationStep>

      <IntegrationStep
        number={3}
        title="Initialize the assistant"
        description="Add the widget configuration after the script and customize the user and presentation values."
      >
        <IntegrationCodeBlock
          language="html"
          code={`<script>
window.chatbotConfig = {
  assistant_id: "${resolvedAssistantId}",
  token: "{PROJECT_KEY}",
  user: {
    id: "{UNIQUE_IDENTIFIER}",
    name: "{NAME}",
  },
  layout: "docked-right",
  position: "bottom-right",
  showLauncher: true,
  name: "Assistant",
  theme: {
    mode: "light",
  },
};
</script>`}
        />
      </IntegrationStep>

      {showDocumentationStep && (
        <IntegrationStep
          number={4}
          title="Customize the experience"
          description="Review the widget reference for supported layouts, themes, and launcher options."
        >
          <Button
            kind="tertiary"
            renderIcon={Launch}
            onClick={() => window.open(documentationUrl, '_blank')}
          >
            View documentation
          </Button>
        </IntegrationStep>
      )}
    </div>
  );
}
