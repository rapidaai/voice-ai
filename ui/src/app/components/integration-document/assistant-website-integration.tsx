import { Assistant } from '@rapidaai/react';
import { RapidaCredentialCard } from '@/app/components/base/cards/rapida-credential-card';
import { FC } from 'react';
import { CodeHighlighting } from '@/app/components/code-highlighting';

export const AssistantWebsiteIntegration: FC<{
  assistant: Assistant;
}> = ({ assistant }) => {
  return (
    <div className="relative space-y-4">
      {/* <div>
        <h1 className="inline-block text-lg font-medium dark:text-gray-100">
          Getting Started
        </h1>
        <p className="mt-1 opacity-75 ">
          An introduction to using Rapida's endpoint build generative ai
          application and usecases.
        </p>
      </div> */}

      <div>
        <h1 className="inline-block text-lg font-medium">Authentication</h1>
        <p className="mt-1 opacity-75 ">
          Setup rapidaai credentials to authenticate your request with
          publishable key and replace{' '}
          <span className="font-mono text-sm">`RAPIDA_API_KEY`</span>
        </p>
      </div>
      <RapidaCredentialCard />

      <div className="space-y-8">
        <div>
          <h1 className="inline-block text-lg font-medium">
            Place the code into html
          </h1>
          <p className="opacity-75 mt-1">
            To add a chat app to the bottom right of your website add this code
            to your html.
          </p>
          <CodeHighlighting
            lang="html"
            className="mt-2"
            code={`<script>
window.chatbotConfig = {
    assistant_id: "${assistant.getId()}",
    token:
        "{RAPIDA_API_KEY}",
    user: {
        name: "<User>",
    },
};
</script>
<script src="https://cdn-01.rapida.ai/public/scripts/app.min.js" defer></script>`}
            lineNumbers={false}
            foldGutter={false}
          />
        </div>
      </div>
    </div>
  );
};
