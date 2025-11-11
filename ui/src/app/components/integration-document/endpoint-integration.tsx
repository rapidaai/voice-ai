import { Variable } from '@rapidaai/react';
import { Endpoint, EndpointProviderModel } from '@rapidaai/react';
import { Tab } from '@/app/components/Tab';
import { FC } from 'react';
import { CodeHighlighting } from '@/app/components/code-highlighting';

export const EndpointIntegration: FC<{
  endpoint: Endpoint;
}> = ({ endpoint }) => {
  return (
    <Tab
      active="Python"
      tabs={[
        {
          label: 'Python',
          element: (
            <CodeHighlighting
              className="h-[400px]"
              lineNumbers={false}
              foldGutter={false}
              lang="python"
              code={`client = RapidaClient(
    RapidaClientOptions(
        api_key="RAPIDA_API_KEY", 
        environment=RapidaEnvironment.PRODUCTION
    ),
)
${PythonEndpointInvokerWithArgs(endpoint)}
`}
            />
          ),
        },
        {
          label: 'Golang',
          element: (
            <CodeHighlighting
              lang="go"
              className="h-[400px]"
              lineNumbers={false}
              foldGutter={false}
              code={`client, err := rapida.GetClient(rapida_builders.
ClientOptionBuilder().
WithApiKey(RAPIDA_API_KEY).
Build())

if err != nil {
	fmt.Println("Getclient error with %+v", err)
	return
}
${GolangEndpointInvokerWithArgs(endpoint)}    
`}
            />
          ),
        },
        {
          label: 'React',
          element: (
            <CodeHighlighting
              lineNumbers={false}
              className="h-[400px]"
              lang="typescript"
              foldGutter={false}
              code={`options = RapidaClientOptions(
    api_key="RAPIDA_API_KEY", 
    environment=RapidaEnvironment.PRODUCTION,
)
client = RapidaClient(options)
${PythonEndpointInvokerWithArgs(endpoint)}}`}
            />
          ),
        },
        {
          label: 'NodeJs',
          element: (
            <CodeHighlighting
              className="h-[400px]"
              lang="typescript"
              lineNumbers={false}
              foldGutter={false}
              code={`options = RapidaClientOptions(
    api_key="RAPIDA_API_KEY", 
    environment=RapidaEnvironment.PRODUCTION
)
client = RapidaClient(options)
${PythonEndpointInvokerWithArgs(endpoint)}
                `}
            />
          ),
        },
      ]}
    />
  );
};

const PythonEndpointInvokerWithArgs = (endpoint: Endpoint): string => {
  const currentEndpointProviderModel = endpoint.getEndpointprovidermodel();
  if (!currentEndpointProviderModel)
    return `response = await client.invoke()
  print(response.to_dict())`;

  let cc_prompt = currentEndpointProviderModel.getChatcompleteprompt();
  cc_prompt?.getPromptvariablesList();
  return `response = await client.invoke(
endpoint=(${PythonEndpointDefinition(currentEndpointProviderModel)}),
inputs={${PythonEndpointVariable(cc_prompt?.getPromptvariablesList())}})
for x in response.get_data():
    print(x.to_text())`;
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

const GolangEndpointInvokerWithArgs = (endpoint: Endpoint): string => {
  return `${GolangEndpointDefinition(endpoint)}
requestBuilder := rapida_builders.NewInvokeRequestBuilder(endpoint)
requestBuilder.AddStringInput("{variable}", "value")
res, err := client.Invoke(requestBuilder.
	Build())
if err == nil {
	if res.IsSuccess() {
		data, _ := res.GetData()
		for _, c := range data {
			cnt, _ := c.ToText()
			println(cnt)
		}
	}
}`;
};

const GolangEndpointDefinition = (epm: Endpoint) => {
  return `endpoint, err := rapida_builders.NewEndpointDefinitionBuilder().
		WithEndpointId(${epm.getId()}).
		Build()`;
};
