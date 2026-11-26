import { ErrorContainer } from '@/app/components/ui/feedback/error-container';

import { ProviderAzureSpeechServiceModelInformationPage } from '@/app/pages/external-integration/provider-models/information/azure-speech-service';
import { CartesiaModelInformationPage } from '@/app/pages/external-integration/provider-models/information/cartesia';
import { DeepgramModelInformationPage } from '@/app/pages/external-integration/provider-models/information/deepgram';
import { ElevanlabModelInformationPage } from '@/app/pages/external-integration/provider-models/information/elevenlab';
import { GoogleSpeechServiceModelInformationPage } from '@/app/pages/external-integration/provider-models/information/google-speech-service';
import { useGlobalNavigation } from '@/hooks/use-global-navigator';
import { useParams } from 'react-router-dom';

export const ProviderModelInformationPage = () => {
  const { provider } = useParams();
  const { goToDashboard } = useGlobalNavigation();
  if (provider === 'elevenlabs') return <ElevanlabModelInformationPage />;
  if (provider === 'azure-speech-service')
    return <ProviderAzureSpeechServiceModelInformationPage />;
  if (provider === 'cartesia') return <CartesiaModelInformationPage />;
  if (provider === 'google-speech-service')
    return <GoogleSpeechServiceModelInformationPage />;
  if (provider === 'deepgram') return <DeepgramModelInformationPage />;
  return (
    <div className="flex flex-1 items-center justify-center">
      <ErrorContainer
        onAction={goToDashboard}
        code="404"
        actionLabel="Go back"
        title={"Sorry we couldn't find this page."}
        description="But dont worry, you can find plenty of other things on our homepage."
      />
    </div>
  );
};
