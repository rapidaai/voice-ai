import { Button } from '@carbon/react';
import { Launch } from '@carbon/icons-react';
import { RapidaCredentialCard } from '@/app/components/domain/cards/rapida-credential-card';
import { useDocumentationUrl } from '@/theme/documentation-url';
import { IntegrationStep } from './integration-primitives';

export function ApiDeploymentSetup() {
  const documentationUrl = useDocumentationUrl();

  return (
    <div className="w-full pb-10">
      <IntegrationStep
        number={1}
        title="Choose a project credential"
        description="Use a publishable project key to authenticate the client that connects to this deployment."
      >
        <RapidaCredentialCard />
      </IntegrationStep>
      <IntegrationStep
        number={2}
        title="Connect your client"
        description="Follow the SDK and API reference to start a session with your assistant and handle its events."
      >
        <Button
          kind="tertiary"
          renderIcon={Launch}
          onClick={() => window.open(documentationUrl, '_blank')}
        >
          View integration documentation
        </Button>
      </IntegrationStep>
    </div>
  );
}
