import { Assistant } from '@rapidaai/react';
import { Variable } from '@rapidaai/react';
import { EndpointProviderModel } from '@rapidaai/react';
import { RapidaCredentialCard } from '@/app/components/base/cards/rapida-credential-card';
import { FC } from 'react';
import { CodeHighlighting } from '@/app/components/code-highlighting';

export const AssistantApiIntegration: FC<{
  assistant: Assistant;
}> = ({ assistant }) => {
  return (
    <div className="relative space-y-4">
      <div>
        <h1 className="inline-block text-lg font-medium dark:text-gray-100">
          Getting Started
        </h1>
        <p className="mt-1 opacity-75 ">
          An introduction to using Rapida's endpoint build generative ai
          application and usecases.
        </p>
      </div>

      <div>
        <h1 className="inline-block text-lg font-medium">Authentication</h1>
        <p className="mt-1 opacity-75 ">
          Setup rapidaai credentials to authenticate your request with
          publishable key and replace{' '}
          <span className="font-mono text-sm">`RAPIDA_API_KEY`</span>
        </p>
      </div>
      <RapidaCredentialCard />
      <p className="opacity-75 ">
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
      <div className="space-y-8">
        <div>
          <h1 className="inline-block text-lg font-medium">
            Installing the RapidaAI sdk
          </h1>
          <p className="opacity-75 mt-1">Install the SDK using pip.</p>
          <CodeHighlighting
            className="mt-2"
            code={`pip install rapida-python`}
            lineNumbers={false}
            foldGutter={false}
          />
        </div>

        <div>
          <h1 className="inline-block text-lg font-medium">
            Import relvant package
          </h1>
          <p className="mt-1 opacity-75">
            Import necessary classes from the Rapida package..
          </p>
          <CodeHighlighting
            className="mt-2"
            lineNumbers={false}
            foldGutter={false}
            code={`from rapida import RapidaClient, RapidaClientOptions, RapidaException, RapidaEnvironment
from rapida.values import StringValue, AudioValue, FileValue, URLValue, ImageValue`}
          />
        </div>

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
          <CodeHighlighting
            className="mt-2"
            lineNumbers={false}
            foldGutter={false}
            code={`options = RapidaClientOptions(api_key="RAPIDA_API_KEY", environment=RapidaEnvironment.PRODUCTION)
client = RapidaClient(options)`}
          />
        </div>

        <div>
          <h1 className="inline-block text-lg font-medium">
            Invoke your endpoint
          </h1>
          <p className="mt-1 opacity-75">
            Set the correct endpoint and parameters to invoke.
          </p>
          <CodeHighlighting
            className="mt-2"
            lineNumbers={false}
            foldGutter={false}
            code={PythonEndpointInvokerWithArgs(assistant)}
          />
        </div>
      </div>
    </div>
  );
};

const PythonEndpointInvokerWithArgs = (assistant: Assistant): string => {
  const currentEndpointProviderModel = assistant.getAssistantprovidermodel();
  //   if (!currentEndpointProviderModel)
  return `response = await client.invoke()
  print(response.to_dict())`;

  //   switch (
  //     getModelModeTypeFromString(currentEndpointProviderModel.getModelmodetype())
  //   ) {
  //     case ModelModeType.complete:
  //       let prompt_c = currentEndpointProviderModel.getCompleteprompt();
  //       prompt_c?.getPromptvariablesList();
  //       return `response = await client.invoke(
  // endpoint=(${PythonEndpointDefinition(currentEndpointProviderModel)}),
  // inputs={${PythonEndpointVariable(prompt_c?.getPromptvariablesList())}})
  // for x in response.get_data():
  //     print(x.to_text())`;
  //     case ModelModeType.chat:
  //       let cc_prompt = currentEndpointProviderModel.getChatcompleteprompt();
  //       cc_prompt?.getPromptvariablesList();
  //       return `response = await client.invoke(
  // endpoint=(${PythonEndpointDefinition(currentEndpointProviderModel)}),
  // inputs={${PythonEndpointVariable(cc_prompt?.getPromptvariablesList())}})
  // for x in response.get_data():
  //     print(x.to_text())`;
  //     case ModelModeType.textToImage:
  //       let prompt_tip = currentEndpointProviderModel.getTexttoimageprompt();
  //       prompt_tip?.getPromptvariablesList();
  //       return `response = await client.invoke(
  // endpoint=(${PythonEndpointDefinition(currentEndpointProviderModel)}),
  // inputs={${PythonEndpointVariable(prompt_tip?.getPromptvariablesList())}})
  // print(response.to_dict())`;
  //     case ModelModeType.textToSpeech:
  //       let prompt_tis = currentEndpointProviderModel.getSpeechtotextprompt();
  //       prompt_tis?.getPromptvariablesList();
  //       return `response = await client.invoke(
  // endpoint=(${PythonEndpointDefinition(currentEndpointProviderModel)}),
  // inputs={${PythonEndpointVariable(prompt_tis?.getPromptvariablesList())}})
  // print(response.to_dict())`;
  //     case ModelModeType.speechToText:
  //       let prompt_stt = currentEndpointProviderModel.getSpeechtotextprompt();
  //       prompt_stt?.getPromptvariablesList();
  //       return `response = await client.invoke(
  // endpoint=(${PythonEndpointDefinition(currentEndpointProviderModel)}),
  // inputs={${PythonEndpointVariable(prompt_stt?.getPromptvariablesList())}})
  // for x in response.get_data():
  //     print(x.to_text())`;
  //     default:
  //       return `response = await client.invoke(
  // endpoint=(${PythonEndpointDefinition(currentEndpointProviderModel)}),
  // inputs={})
  // for x in response.get_data():
  //     print(x.to_text())`;
  //   }
};

const PythonEndpointVariable = (vr?: Variable[]) => {
  if (!vr || vr.length === 0) return '{}';

  // Create an array of string representations
  //   const variableStrings = vr.map(v => v.getName());

  // Join the names with commas and wrap in curly braces
  return `${vr.map((x, idx) => {
    if (x.getType() === 'audio-files') {
      return `"${x.getName()}": AudioValue('/path/to/audio/files')`;
    }
    if (x.getType() === 'files') {
      return `"${x.getName()}": FileValue('/path/to/file')`;
    }
    if (x.getType() === 'url') {
      return `"${x.getName()}": URLValue('https://an-url-to-for-somethig')`;
    }
    return `"${x.getName()}": StringValue('example-${x.getType()}')`;
  })}`;
};

const PythonEndpointDefinition = (epm: EndpointProviderModel) => {
  return `${epm.getEndpointid()}, "vrsn_${epm.getId()}"`;
};
