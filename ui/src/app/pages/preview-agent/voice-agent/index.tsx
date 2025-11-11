import { GreenNoticeBlock } from '@/app/components/container/message/notice-block';
import { Dropdown } from '@/app/components/Dropdown';
import { IBlueBGArrowButton } from '@/app/components/Form/Button';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import {
  JsonTextarea,
  NumberTextarea,
  ParagraphTextarea,
  TextTextarea,
  UrlTextarea,
} from '@/app/components/Form/Textarea';
import { InputGroup } from '@/app/components/input-group';
import { InputVarForm } from '@/app/pages/endpoint/view/try-playground/experiment-prompt/components/input-var-form';
import { VoiceAgent } from '@/app/pages/preview-agent/voice-agent/voice-agent';
import { useRapidaStore } from '@/hooks';
import { useCurrentCredential } from '@/hooks/use-credential';
import { InputVarType } from '@/models/common';
import { cn } from '@/styles/media';
import { getStatusMetric } from '@/utils/metadata';
import {
  AgentConfig,
  Channel,
  ConnectionConfig,
  InputOptions,
  StringToAny,
  CreatePhoneCall,
  AssistantDefinition,
  CreatePhoneCallRequest,
  Assistant,
  GetAssistant,
  GetAssistantRequest,
  Variable,
} from '@rapidaai/react';
import { useEffect, useState } from 'react';
import { Navigate, useParams, useSearchParams } from 'react-router-dom';
export const PublicPreviewVoiceAgent = () => {
  const [searchParams] = useSearchParams();
  const { assistantId } = useParams();
  const authId = searchParams.get('authId');
  const token = searchParams.get('token');
  const name = searchParams.get('name');

  if (!assistantId || !authId || !token || !name) {
    return <Navigate to="/404" replace />;
  }

  return (
    <VoiceAgent
      connectConfig={ConnectionConfig.DefaultConnectionConfig(
        ConnectionConfig.WithSDK({
          ApiKey: token,
          UserId: authId,
        }),
      )}
      agentConfig={new AgentConfig(
        assistantId,
        new InputOptions([Channel.Audio, Channel.Text], Channel.Text),
      )
        .addKeywords([name])
        .addArgument('name', name)
        .addMetadata('name', StringToAny(name))
        .addMetadata('authId', StringToAny(authId))}
    />
  );
};

export const PreviewVoiceAgent = () => {
  const { user, authId, token, projectId } = useCurrentCredential();
  const { assistantId } = useParams();

  if (!assistantId || !user?.name) {
    return <Navigate to="/404" replace />;
  }

  return (
    <VoiceAgent
      //   agentCallback={agentCallback}
      connectConfig={
        ConnectionConfig.DefaultConnectionConfig(
          ConnectionConfig.WithDebugger({
            authorization: token,
            userId: authId,
            projectId: projectId,
          }),
        ).withLocal()
        // .withCustomEndpoint({
        //   assistant: 'https://integral-presently-cub.ngrok-free.app',
        // })
      }
      agentConfig={new AgentConfig(
        assistantId,
        new InputOptions([Channel.Audio, Channel.Text], Channel.Text),
      )
        .addKeywords([user.name])
        .addMetadata('authId', StringToAny(authId))
        .addMetadata('projectId', StringToAny(projectId))}
      // .addCustomOption('listen.language', StringToAny('en'))
      // .addCustomOption('speak.language', StringToAny('en'))
      // .addCustomOption('listen.model', StringToAny('nova-3'))}
    />
  );
};
export const PreviewPhoneAgent = () => {
  const { authId, token, projectId } = useCurrentCredential();
  let connectionCfg = ConnectionConfig.DefaultConnectionConfig(
    ConnectionConfig.WithPersonalToken({
      Authorization: token,
      AuthId: authId,
      ProjectId: projectId,
    }),
  );
  //   .withLocal();
  //   .withLocal();

  const { showLoader, hideLoader, loading } = useRapidaStore();
  const { assistantId } = useParams();
  const [assistant, setAssistant] = useState<Assistant | null>(null);

  const countries = [
    { name: 'Afghanistan', value: '+93', code: 'AF' },
    { name: 'Albania', value: '+355', code: 'AL' },
    { name: 'Algeria', value: '+213', code: 'DZ' },
    { name: 'Andorra', value: '+376', code: 'AD' },
    { name: 'Angola', value: '+244', code: 'AO' },
    { name: 'Argentina', value: '+54', code: 'AR' },
    { name: 'Armenia', value: '+374', code: 'AM' },
    { name: 'Australia', value: '+61', code: 'AU' },
    { name: 'Austria', value: '+43', code: 'AT' },
    { name: 'Azerbaijan', value: '+994', code: 'AZ' },
    { name: 'Bahrain', value: '+973', code: 'BH' },
    { name: 'Bangladesh', value: '+880', code: 'BD' },
    { name: 'Belgium', value: '+32', code: 'BE' },
    { name: 'Brazil', value: '+55', code: 'BR' },
    { name: 'Bulgaria', value: '+359', code: 'BG' },
    { name: 'Cambodia', value: '+855', code: 'KH' },
    { name: 'Cameroon', value: '+237', code: 'CM' },
    { name: 'Canada', value: '+1', code: 'CA' },
    { name: 'Chile', value: '+56', code: 'CL' },
    { name: 'China', value: '+86', code: 'CN' },
    { name: 'Colombia', value: '+57', code: 'CO' },
    { name: 'Costa Rica', value: '+506', code: 'CR' },
    { name: 'Croatia', value: '+385', code: 'HR' },
    { name: 'Czech Republic', value: '+420', code: 'CZ' },
    { name: 'Denmark', value: '+45', code: 'DK' },
    { name: 'Egypt', value: '+20', code: 'EG' },
    { name: 'Estonia', value: '+372', code: 'EE' },
    { name: 'Finland', value: '+358', code: 'FI' },
    { name: 'France', value: '+33', code: 'FR' },
    { name: 'Germany', value: '+49', code: 'DE' },
    { name: 'Greece', value: '+30', code: 'GR' },
    { name: 'Hong Kong', value: '+852', code: 'HK' },
    { name: 'Hungary', value: '+36', code: 'HU' },
    { name: 'Iceland', value: '+354', code: 'IS' },
    { name: 'India', value: '+91', code: 'IN' },
    { name: 'Indonesia', value: '+62', code: 'ID' },
    { name: 'Iran', value: '+98', code: 'IR' },
    { name: 'Iraq', value: '+964', code: 'IQ' },
    { name: 'Ireland', value: '+353', code: 'IE' },
    { name: 'Israel', value: '+972', code: 'IL' },
    { name: 'Italy', value: '+39', code: 'IT' },
    { name: 'Japan', value: '+81', code: 'JP' },
    { name: 'Jordan', value: '+962', code: 'JO' },
    { name: 'Kazakhstan', value: '+7', code: 'KZ' },
    { name: 'Kenya', value: '+254', code: 'KE' },
    { name: 'Kuwait', value: '+965', code: 'KW' },
    { name: 'Latvia', value: '+371', code: 'LV' },
    { name: 'Lebanon', value: '+961', code: 'LB' },
    { name: 'Lithuania', value: '+370', code: 'LT' },
    { name: 'Luxembourg', value: '+352', code: 'LU' },
    { name: 'Malaysia', value: '+60', code: 'MY' },
    { name: 'Maldives', value: '+960', code: 'MV' },
    { name: 'Malta', value: '+356', code: 'MT' },
    { name: 'Mexico', value: '+52', code: 'MX' },
    { name: 'Monaco', value: '+377', code: 'MC' },
    { name: 'Morocco', value: '+212', code: 'MA' },
    { name: 'Nepal', value: '+977', code: 'NP' },
    { name: 'Netherlands', value: '+31', code: 'NL' },
    { name: 'New Zealand', value: '+64', code: 'NZ' },
    { name: 'Nigeria', value: '+234', code: 'NG' },
    { name: 'Norway', value: '+47', code: 'NO' },
    { name: 'Oman', value: '+968', code: 'OM' },
    { name: 'Pakistan', value: '+92', code: 'PK' },
    { name: 'Peru', value: '+51', code: 'PE' },
    { name: 'Philippines', value: '+63', code: 'PH' },
    { name: 'Poland', value: '+48', code: 'PL' },
    { name: 'Portugal', value: '+351', code: 'PT' },
    { name: 'Qatar', value: '+974', code: 'QA' },
    { name: 'Romania', value: '+40', code: 'RO' },
    { name: 'Russia', value: '+7', code: 'RU' },
    { name: 'Saudi Arabia', value: '+966', code: 'SA' },
    { name: 'Serbia', value: '+381', code: 'RS' },
    { name: 'Singapore', value: '+65', code: 'SG' },
    { name: 'Slovakia', value: '+421', code: 'SK' },
    { name: 'Slovenia', value: '+386', code: 'SI' },
    { name: 'South Africa', value: '+27', code: 'ZA' },
    { name: 'South Korea', value: '+82', code: 'KR' },
    { name: 'Spain', value: '+34', code: 'ES' },
    { name: 'Sri Lanka', value: '+94', code: 'LK' },
    { name: 'Sweden', value: '+46', code: 'SE' },
    { name: 'Switzerland', value: '+41', code: 'CH' },
    { name: 'Syria', value: '+963', code: 'SY' },
    { name: 'Taiwan', value: '+886', code: 'TW' },
    { name: 'Thailand', value: '+66', code: 'TH' },
    { name: 'Turkey', value: '+90', code: 'TR' },
    { name: 'Ukraine', value: '+380', code: 'UA' },
    { name: 'United Arab Emirates', value: '+971', code: 'AE' },
    { name: 'United Kingdom', value: '+44', code: 'GB' },
    { name: 'United States', value: '+1', code: 'US' },
    { name: 'Uruguay', value: '+598', code: 'UY' },
    { name: 'Uzbekistan', value: '+998', code: 'UZ' },
    { name: 'Venezuela', value: '+58', code: 'VE' },
    { name: 'Vietnam', value: '+84', code: 'VN' },
    { name: 'Yemen', value: '+967', code: 'YE' },
    { name: 'Zimbabwe', value: '+263', code: 'ZW' },
  ];
  const [country, setCountry] = useState({
    name: 'Singapore',
    value: '+65',
  });
  const [variables, setVaribales] = useState<Variable[]>([]);
  const [phoneNumber, setPhoneNumber] = useState('');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [argumentMap, setArgumentMap] = useState<Map<string, string>>(
    new Map(),
  );

  useEffect(() => {
    if (assistantId) {
      const request = new GetAssistantRequest();
      const assistantDef = new AssistantDefinition();
      assistantDef.setAssistantid(assistantId);
      request.setAssistantdefinition(assistantDef);
      GetAssistant(connectionCfg, request)
        .then(response => {
          hideLoader();
          if (response?.getSuccess()) {
            setAssistant(response.getData()!);
            const assistantProvider = response
              .getData()
              ?.getAssistantprovidermodel();
            if (assistantProvider?.getTemplate()?.getPromptvariablesList()) {
              setVaribales(
                assistantProvider?.getTemplate()?.getPromptvariablesList()!,
              );
              assistantProvider
                ?.getTemplate()
                ?.getPromptvariablesList()
                .forEach(v => {
                  if (v.getDefaultvalue()) {
                    onChangeArgument(v.getName(), v.getDefaultvalue());
                  }
                });
              return;
            }
          }
        })
        .catch(err => {
          hideLoader();
        });
    }
  }, []);
  if (!assistantId) {
    return <Navigate to="/404" replace />;
  }

  const handlePhoneNumberChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value.replace(/\D/g, ''); // Remove non-digit characters
    setPhoneNumber(value);
    setError('');
  };

  const validatePhoneNumber = () => {
    if (!country.value) {
      setError('Please select a country');
      return false;
    }
    if (phoneNumber.length < 7 || phoneNumber.length > 15) {
      setError('Please enter a valid phone number for call.');
      return false;
    }
    return true;
  };

  const onChangeArgument = (k: string, vl: string) => {
    setArgumentMap(prev => {
      const updatedMap = new Map(prev);
      updatedMap.set(k, vl);
      return updatedMap;
    });
  };

  const handleSubmit = () => {
    if (validatePhoneNumber()) {
      setError('');
      showLoader();
      const phoneCallRequest = new CreatePhoneCallRequest();
      const assistant = new AssistantDefinition();
      assistant.setAssistantid(assistantId);
      assistant.setVersion('latest');
      phoneCallRequest.setAssistant(assistant);
      argumentMap.forEach((value, key) => {
        phoneCallRequest.getArgsMap().set(key, StringToAny(value));
      });

      //   phoneCallRequest.setFromnumber('FROM_NUMBER');
      phoneCallRequest.setTonumber(country.value + phoneNumber);
      CreatePhoneCall(connectionCfg, phoneCallRequest)
        .then(x => {
          hideLoader();
          if (x.getSuccess()) {
            const status = getStatusMetric(x.getData()?.getMetricsList());
            if (status === 'FAILED') {
              setError('Unable to start the call, please try again.');
              return;
            }
            setSuccess('Call has been create successfully.');
            setTimeout(() => setSuccess(''), 60000);
            return;
          }
          let err = x.getError();
          if (err?.getHumanmessage()) setError(err?.getHumanmessage());
        })
        .catch(x => {
          hideLoader();
          setError('Unable to start the call, please try again.');
        });
    }
  };

  return (
    <div className="h-dvh flex justify-center">
      <div className="bg-light-background dark:bg-gray-950/50 w-[700px]! mx-auto my-auto shadow-sm">
        <div className="space-y-6 m-10">
          <div className="space-y-2">
            <h1 className="text-3xl font-semibold">Hello,</h1>
            <h3 className="text-xl font-medium opacity-80">
              How can I help you with your call today?
            </h3>
          </div>
          <FieldSet className="mt-8">
            <div
              className={cn(
                'p-px',
                'text-sm!',
                'outline-solid outline-transparent',
                'focus-within:outline-blue-600 focus:outline-blue-600 -outline-offset-1',
                'border-b border-gray-400 dark:border-gray-600',
                'dark:focus-within:border-blue-600 focus-within:border-blue-600',
                'transition-all duration-200 ease-in-out',
                'flex relative',
                'divide-x',
              )}
            >
              <div className="w-44 relative">
                <Dropdown
                  className="bg-white max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none! outline-hidden"
                  currentValue={country}
                  setValue={v => {
                    setCountry(v);
                  }}
                  allValue={countries}
                  placeholder="Select country"
                  option={c => (
                    <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                      <span className="truncate capitalize">{c.name}</span>
                    </span>
                  )}
                  label={c => (
                    <span className="inline-flex items-center gap-2 sm:gap-2.5 max-w-full text-sm font-medium">
                      <span className="truncate capitalize">{c.name}</span>
                    </span>
                  )}
                />
              </div>
              <Input
                type="tel"
                placeholder="Enter your phone number"
                className="bg-white max-w-full dark:bg-gray-950 focus-within:border-none! focus-within:outline-hidden! border-none!"
                value={phoneNumber}
                onChange={handlePhoneNumberChange}
              />
            </div>
            <ErrorMessage message={error}></ErrorMessage>
            {success && <GreenNoticeBlock>{success}</GreenNoticeBlock>}
          </FieldSet>
          <div className="flex justify-end">
            <IBlueBGArrowButton onClick={handleSubmit} isLoading={loading}>
              Start Call
            </IBlueBGArrowButton>
          </div>
        </div>
      </div>
      {assistant && (
        <div className="w-96 border-l h-dvh overflow-auto">
          <div className="px-4 py-4 text-sm leading-normal ">
            <div className="flex flex-row justify-between items-center text-sm tracking-wider">
              <h3>Name</h3>
            </div>
            <div className="py-2 text-sm leading-normal">
              {assistant.getName()}
            </div>
            <div className="flex mt-4 flex-row  justify-between items-center text-sm tracking-wider">
              <h3>Description</h3>
            </div>
            <div className="py-2 text-sm leading-normal">
              {assistant.getDescription()}
            </div>
          </div>
          <InputGroup title="Arguments">
            <div className="text-sm leading-normal ">
              {variables.map((x, idx) => {
                return (
                  <InputVarForm
                    key={idx}
                    var={x}
                    className="bg-light-background"
                  >
                    {x.getType() === InputVarType.textInput && (
                      <TextTextarea
                        id={x.getName()}
                        defaultValue={x.getDefaultvalue()}
                        onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
                          onChangeArgument(x.getName(), e.target.value)
                        }
                      />
                    )}
                    {x.getType() === InputVarType.paragraph && (
                      <ParagraphTextarea
                        id={x.getName()}
                        defaultValue={x.getDefaultvalue()}
                        onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
                          onChangeArgument(x.getName(), e.target.value)
                        }
                      />
                    )}
                    {x.getType() === InputVarType.number && (
                      <NumberTextarea
                        id={x.getName()}
                        defaultValue={x.getDefaultvalue()}
                        onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
                          onChangeArgument(x.getName(), e.target.value)
                        }
                      />
                    )}
                    {x.getType() === InputVarType.json && (
                      <JsonTextarea
                        id={x.getName()}
                        defaultValue={x.getDefaultvalue()}
                        onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
                          onChangeArgument(x.getName(), e.target.value)
                        }
                      />
                    )}
                    {x.getType() === InputVarType.url && (
                      <UrlTextarea
                        id={x.getName()}
                        defaultValue={x.getDefaultvalue()}
                        onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) =>
                          onChangeArgument(x.getName(), e.target.value)
                        }
                      />
                    )}
                  </InputVarForm>
                );
              })}
            </div>
          </InputGroup>
          <InputGroup title="Deployment">
            <div className="px-4 py-4 space-y-px text-sm leading-normal ">
              <div className="space-y-4">
                <div className="flex justify-between">
                  <div className="-foreground">Input Mode</div>
                  <div className="font-medium">
                    Text
                    {assistant?.getPhonedeployment()?.getInputaudio() &&
                      ', Audio'}
                  </div>
                </div>

                {/*  */}
                {assistant
                  .getPhonedeployment()
                  ?.getInputaudio()
                  ?.getAudiooptionsList() &&
                  assistant
                    .getPhonedeployment()
                    ?.getInputaudio()
                    ?.getAudiooptionsList().length! > 0 && (
                    <div className="space-y-4">
                      <div className="flex justify-between">
                        <div className="-foreground">Listen.Provider</div>
                        <div className="font-medium mt-1 ">
                          {assistant
                            .getPhonedeployment()
                            ?.getInputaudio()
                            ?.getAudioprovider()}
                        </div>
                      </div>
                      {assistant
                        .getPhonedeployment()
                        ?.getInputaudio()
                        ?.getAudiooptionsList()
                        .filter(d => d.getValue())
                        .filter(d => d.getKey().startsWith('listen.'))
                        .map((detail, index) => (
                          <div className="flex justify-between" key={index}>
                            <div className="-foreground capitalize">
                              {detail.getKey()}
                            </div>
                            <div className="font-medium">
                              {detail.getValue()}
                            </div>
                          </div>
                        ))}
                    </div>
                  )}
                <div className="flex justify-between">
                  <div className="-foreground">Output Mode</div>
                  <div className="font-medium">
                    Text
                    {assistant?.getPhonedeployment()?.getOutputaudio() &&
                      ', Audio'}
                  </div>
                </div>
                {assistant
                  .getPhonedeployment()
                  ?.getOutputaudio()
                  ?.getAudiooptionsList() &&
                  assistant
                    .getPhonedeployment()
                    ?.getOutputaudio()
                    ?.getAudiooptionsList().length! > 0 && (
                    <div className="space-y-4">
                      <div className="flex justify-between">
                        <div className="-foreground">Speak.Provider</div>
                        <div className="font-medium mt-1 ">
                          {assistant
                            .getPhonedeployment()
                            ?.getOutputaudio()
                            ?.getAudioprovider()}
                        </div>
                      </div>
                      {assistant
                        .getPhonedeployment()
                        ?.getOutputaudio()
                        ?.getAudiooptionsList()
                        .filter(d => d.getValue())
                        .filter(d => d.getKey().startsWith('speak.'))
                        .map((detail, index) => (
                          <div
                            key={index}
                            className="flex justify-between gap-8"
                          >
                            <div className="-foreground capitalize">
                              {detail.getKey()}
                            </div>
                            <div className="font-medium truncate">
                              {detail.getValue()}
                            </div>
                          </div>
                        ))}
                    </div>
                  )}

                <div className="flex justify-between">
                  <div className="-foreground">Telephony</div>
                  <div className="font-medium">
                    {assistant?.getPhonedeployment()?.getPhoneprovidername()}
                  </div>
                </div>
              </div>
            </div>
          </InputGroup>
        </div>
      )}
    </div>
  );
};
