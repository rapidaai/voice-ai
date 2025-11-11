import { Endpoint } from '@rapidaai/react';
import { RapidaCredentialCard } from '@/app/components/base/cards/rapida-credential-card';
import { FC } from 'react';
import { CodeHighlighting } from '@/app/components/code-highlighting';

export const Integration: FC<{
  endpoint: Endpoint;
}> = ({ endpoint }) => {
  return (
    <div className="relative  px-4 space-y-4">
      <div>
        <h1 className="inline-block text-lg font-medium dark:text-gray-100">
          Authentication
        </h1>
        <p className="">
          Setup rapidaai credentials to authenticate your request with
          publishable key and replace{' '}
          <span className="font-mono text-sm">`RAPIDA_API_KEY`</span>
        </p>
      </div>
      <RapidaCredentialCard />
      <p className="">
        Choose your prefered programming language to integrate sdk, the SDK
        provides a developer friendly way to connect to rapida.
      </p>
      <ul className="pb-px flex-none min-w-full overflow-auto border-b border-gray-200 space-x-6 flex dark:border-gray-200/10">
        <li className="cursor-pointer flex leading-6 font-semibold whitespace-nowrap -mb-px max-w-max border-b dark:text-blue-500 border-blue-500">
          Python
        </li>
        <li className="cursor-pointer flex leading-6 font-medium whitespace-nowrap -mb-px max-w-max dark:text-gray-400 ">
          Typescript
        </li>
        <li className="cursor-pointer flex leading-6 font-medium whitespace-nowrap -mb-px max-w-max dark:text-gray-400 ">
          Golang
        </li>
        <li className="cursor-pointer flex leading-6 font-medium whitespace-nowrap -mb-px max-w-max dark:text-gray-400 ">
          Java
        </li>
      </ul>
      <div className="space-y-4">
        <div>
          <h1 className="inline-block text-lg font-medium">
            Installing the RapidaAI sdk
          </h1>
          <p className="opacity-75 mt-1">Install the SDK using pip.</p>
        </div>
        <CodeHighlighting
          code={`pip install rapida-python`}
          lineNumbers={false}
          foldGutter={false}
        />
        <div>
          <h1 className="inline-block text-lg font-medium">
            Import relvant package
          </h1>
          <p className="mt-1 opacity-75">
            Import necessary classes from the Rapida package..
          </p>
        </div>
        <CodeHighlighting
          lineNumbers={false}
          foldGutter={false}
          code={`from rapida import RapidaClient, RapidaClientOptions, RapidaException, RapidaEnvironment`}
        />
        <div>
          <h1 className="inline-block text-lg font-medium">
            Create rapida client
          </h1>
          <p className="mt-1 opacity-75">
            Replace "RAPIDA_API_KEY" with your actual API key. Set the
            environment as needed (
            <span className="underline text-blue-500">PRODUCTION</span> or{' '}
            <span className="underline text-blue-500">DEVELOPMENT</span>).
          </p>
        </div>
        <CodeHighlighting
          lineNumbers={false}
          foldGutter={false}
          code={`options = RapidaClientOptions(api_key="RAPIDA_API_KEY", environment=RapidaEnvironment.PRODUCTION)
client = RapidaClient(options)`}
        />
        <div>
          <h1 className="inline-block text-lg font-medium">
            Invoke your endpoint
          </h1>
          <p className="mt-1 opacity-75">
            Set the correct endpoint and parameters to invoke.
          </p>
        </div>
        <CodeHighlighting
          lineNumbers={false}
          foldGutter={false}
          code={PythonEndpointInvokerWithArgs()}
        />
      </div>
    </div>
  );
};

export const PythonEndpointInvokerWithArgs = () => {
  return `response = await client.invoke(
endpoint=(),
print(response.to_dict())`;
};
