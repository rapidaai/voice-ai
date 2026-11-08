import { Tabs } from '@/app/components/ui/primitives';
import { RapidaCredentialCard } from '@/app/components/domain/cards/rapida-credential-card';
import {
  IntegrationCodeBlock,
  IntegrationStep,
} from './integration-primitives';
import type {
  Endpoint,
  EndpointProviderModel,
  Variable,
} from '@rapidaai/react';
import { useState, type ReactNode } from 'react';

interface EndpointIntegrationProps {
  endpoint: Endpoint;
  credentialCard?: ReactNode;
}

export function EndpointIntegration({
  endpoint,
  credentialCard,
}: EndpointIntegrationProps) {
  const epm = endpoint.getEndpointprovidermodel();
  const credentials = credentialCard ?? <RapidaCredentialCard />;
  const [selectedTab, setSelectedTab] = useState(0);

  const languageTabs = [
    {
      label: 'Python',
      element: (
        <div className="w-full pb-10" key="Python">
          <IntegrationStep
            number={1}
            title="Install the SDK"
            description="Install the Rapida Python SDK using pip."
          >
            <IntegrationCodeBlock
              language="bash"
              code={`pip install rapida-python`}
            />
          </IntegrationStep>

          <IntegrationStep
            number={2}
            title="Authenticate"
            description={`Copy your publishable API key and replace RAPIDA_API_KEY in the code below.`}
          >
            {credentials}
          </IntegrationStep>

          <IntegrationStep
            number={3}
            title="Import & initialise"
            description="Import the required classes and create a Rapida client."
          >
            <IntegrationCodeBlock
              language="python"
              code={`from rapida import RapidaClient, RapidaClientOptions, RapidaEnvironment
from rapida.values import StringValue, AudioValue, FileValue, URLValue

client = RapidaClient(
    RapidaClientOptions(
        api_key="RAPIDA_API_KEY",
        environment=RapidaEnvironment.PRODUCTION,
    )
)`}
            />
          </IntegrationStep>

          <IntegrationStep
            number={4}
            title="Invoke your endpoint"
            description="Call the endpoint with the required input arguments."
          >
            <IntegrationCodeBlock
              language="python"
              code={buildPythonInvoke(epm)}
            />
          </IntegrationStep>
        </div>
      ),
    },
    {
      label: 'TypeScript',
      element: (
        <div className="w-full pb-10" key="TypeScript">
          <IntegrationStep
            number={1}
            title="Install the SDK"
            description="Install the Rapida Node.js SDK using npm or yarn."
          >
            <IntegrationCodeBlock
              language="bash"
              code={`npm install @rapidaai/rapida-node\n# or\nyarn add @rapidaai/rapida-node`}
            />
          </IntegrationStep>

          <IntegrationStep
            number={2}
            title="Authenticate"
            description="Copy your publishable API key and replace RAPIDA_API_KEY in the code below."
          >
            {credentials}
          </IntegrationStep>

          <IntegrationStep
            number={3}
            title="Import & initialise"
            description="Import the required classes and create a Rapida client."
          >
            <IntegrationCodeBlock
              language="typescript"
              code={`import {
  RapidaClient,
  RapidaClientOptions,
  RapidaEnvironment,
} from '@rapidaai/rapida-node';
import {
  AudioValue,
  FileValue,
  StringValue,
  URLValue,
} from '@rapidaai/rapida-node/values';

const client = new RapidaClient(
  new RapidaClientOptions({
    apiKey: 'RAPIDA_API_KEY',
    environment: RapidaEnvironment.PRODUCTION,
  }),
);`}
            />
          </IntegrationStep>

          <IntegrationStep
            number={4}
            title="Invoke your endpoint"
            description="Call the endpoint with the required input arguments."
          >
            <IntegrationCodeBlock
              language="typescript"
              code={buildTypeScriptInvoke(epm)}
            />
          </IntegrationStep>
        </div>
      ),
    },
    {
      label: 'Golang',
      element: (
        <div className="w-full pb-10" key="Golang">
          <IntegrationStep
            number={1}
            title="Install the SDK"
            description="Add the Rapida Go module to your project."
          >
            <IntegrationCodeBlock
              language="bash"
              code={`go get github.com/rapidaai/rapida-go`}
            />
          </IntegrationStep>

          <IntegrationStep
            number={2}
            title="Authenticate"
            description="Copy your publishable API key and replace RAPIDA_API_KEY in the code below."
          >
            {credentials}
          </IntegrationStep>

          <IntegrationStep
            number={3}
            title="Import & initialise"
            description="Import the required packages and create a Rapida client."
          >
            <IntegrationCodeBlock
              language="go"
              code={`import (
    "github.com/rapidaai/rapida-go/rapida"
    "github.com/rapidaai/rapida-go/rapida_builders"
)

client, err := rapida.GetClient(
    rapida_builders.ClientOptionBuilder().
        WithApiKey("RAPIDA_API_KEY").
        Build(),
)
if err != nil {
    log.Fatalf("failed to create client: %v", err)
}`}
            />
          </IntegrationStep>

          <IntegrationStep
            number={4}
            title="Invoke your endpoint"
            description="Build the endpoint definition and invoke it with your inputs."
          >
            <IntegrationCodeBlock
              language="go"
              code={buildGolangInvoke(endpoint)}
            />
          </IntegrationStep>
        </div>
      ),
    },
  ];

  return (
    <Tabs
      tabs={languageTabs.map(tab => tab.label)}
      selectedIndex={selectedTab}
      onChange={setSelectedTab}
      contained
      fullWidth
      fill
      aria-label="Endpoint integration languages"
      className="bg-surface "
      panelClassName="overflow-auto !p-0"
    >
      {languageTabs.map(tab => tab.element)}
    </Tabs>
  );
}

const buildPythonInvoke = (epm: EndpointProviderModel | undefined): string => {
  if (!epm) {
    return `response = await client.invoke(
    endpoint=("ENDPOINT_ID", "ENDPOINT_VERSION"),
    inputs={},
)
for item in response.get_data():
    print(item.to_text())`;
  }

  const vars = getPromptVariables(epm);
  const inputs = buildPythonInputs(vars);

  return `response = await client.invoke(
    endpoint=("${epm.getEndpointid()}", "vrsn_${epm.getId()}"),
    inputs={${inputs}},
)
for item in response.get_data():
    print(item.to_text())`;
};

const buildPythonInputs = (vars: Variable[]): string => {
  if (vars.length === 0) return '';
  return vars
    .map(v => {
      if (v.getType() === 'audio-files')
        return `\n        "${v.getName()}": AudioValue("/path/to/audio")`;
      if (v.getType() === 'files')
        return `\n        "${v.getName()}": FileValue("/path/to/file")`;
      if (v.getType() === 'url')
        return `\n        "${v.getName()}": URLValue("https://example.com")`;
      return `\n        "${v.getName()}": StringValue("example-value")`;
    })
    .join(',');
};

const buildTypeScriptInvoke = (
  epm: EndpointProviderModel | undefined,
): string => {
  if (!epm) {
    return `const response = await client.invoke({
  endpoint: ['ENDPOINT_ID', 'ENDPOINT_VERSION'],
  inputs: {},
});
for (const item of await response.getData()) {
  console.log(await item.toText());
}`;
  }

  const vars = getPromptVariables(epm);
  const inputs = buildTypeScriptInputs(vars);

  return `const response = await client.invoke({
  endpoint: ['${epm.getEndpointid()}', 'vrsn_${epm.getId()}'],
  inputs: {${inputs}},
});
for (const item of await response.getData()) {
  console.log(await item.toText());
}`;
};

const buildTypeScriptInputs = (vars: Variable[]): string => {
  if (vars.length === 0) return '';
  return vars
    .map(v => {
      if (v.getType() === 'audio-files') {
        return `\n    ${v.getName()}: new AudioValue('/path/to/audio')`;
      }
      if (v.getType() === 'files') {
        return `\n    ${v.getName()}: new FileValue('/path/to/file')`;
      }
      if (v.getType() === 'url') {
        return `\n    ${v.getName()}: new URLValue('https://example.com')`;
      }
      return `\n    ${v.getName()}: new StringValue('example-value')`;
    })
    .join(',');
};

const buildGolangInvoke = (endpoint: Endpoint): string => {
  const epm = endpoint.getEndpointprovidermodel();
  const endpointId = epm ? epm.getEndpointid() : 'ENDPOINT_ID';

  return `endpoint, err := rapida_builders.NewEndpointDefinitionBuilder().
    WithEndpointId("${endpointId}").
    Build()

request := rapida_builders.NewInvokeRequestBuilder(endpoint).
    AddStringInput("variable", "value").
    Build()

res, err := client.Invoke(request)
if err == nil && res.IsSuccess() {
    data, _ := res.GetData()
    for _, item := range data {
        text, _ := item.ToText()
        println(text)
    }
}`;
};

const getPromptVariables = (epm: EndpointProviderModel): Variable[] => {
  const prompt = epm.getChatcompleteprompt();
  if (!prompt) {
    return [];
  }
  return prompt.getPromptvariablesList();
};
