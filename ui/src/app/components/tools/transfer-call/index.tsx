import { FC, useState, useCallback } from 'react';
import { FormGroup, Stack, TextArea } from '@/app/components/carbon/form';
import { Select, SelectItem, Slider, Tooltip } from '@carbon/react';
import {
  ConfigureToolProps,
  ToolDefinitionForm,
  useParameterManager,
} from '../common';
import { InputGroup } from '../../input-group';
import ConfigSelect from '@/app/components/configuration/config-var/config-select';
import { SEPARATOR } from './constant';
import { Information } from '@carbon/icons-react';

export const ConfigureTransferCall: FC<ConfigureToolProps> = ({
  toolDefinition,
  onChangeToolDefinition,
  inputClass,
  parameters,
  onParameterChange,
}) => {
  const { getParamValue, updateParameter } = useParameterManager(
    parameters,
    onParameterChange,
  );

  const [transferToList, setTransferToList] = useState<string[]>(() => {
    const raw = getParamValue('tool.transfer_to');
    return raw ? raw.split(SEPARATOR) : [''];
  });
  const transferDelayRaw = getParamValue('tool.transfer_delay');
  const transferDelayParsed = Number.parseInt(transferDelayRaw, 10);
  const transferDelayValue =
    transferDelayRaw === '' || Number.isNaN(transferDelayParsed)
      ? 500
      : Math.min(Math.max(transferDelayParsed, 500), 3000);

  const handleTransferToChange = useCallback(
    (options: string[]) => {
      const normalizedOptions = options.length > 0 ? options : [''];
      setTransferToList(normalizedOptions);
      updateParameter(
        'tool.transfer_to',
        normalizedOptions.filter(Boolean).join(SEPARATOR),
      );
    },
    [updateParameter],
  );

  return (
    <>
      <InputGroup title="Action Definition">
        <Stack gap={6}>
          <FormGroup
            legendText={
              <span className="inline-flex items-center gap-1">
                Transfer Destinations
                <Tooltip
                  align="right"
                  label="Phone numbers or SIP URIs to transfer calls to. Drag to reorder."
                >
                  <Information size={14} />
                </Tooltip>
              </span>
            }
          >
            <ConfigSelect
              options={transferToList}
              label="Add transfer destination"
              placeholder="+14155551234 or sip:agent@example.com"
              helperText="Phone numbers or SIP URIs to transfer calls to. Drag to reorder."
              onChange={handleTransferToChange}
            />
          </FormGroup>
          <TextArea
            id="transfer-message"
            labelText="Transfer Message"
            helperText="The message to be played when transferring the call."
            value={getParamValue('tool.transfer_message')}
            onChange={e =>
              updateParameter('tool.transfer_message', e.target.value)
            }
            placeholder="Your transfer message"
          />
          <Stack gap={6} orientation="horizontal">
            <Select
              id="post-transfer-action"
              labelText={
                <span className="inline-flex items-center gap-1">
                  Post Transfer Action
                  <Tooltip
                    align="right"
                    label="Behavior after transfer completes or fails."
                  >
                    <Information size={14} />
                  </Tooltip>
                </span>
              }
              value={getParamValue('tool.post_transfer_action') || 'end_call'}
              onChange={e =>
                updateParameter('tool.post_transfer_action', e.target.value)
              }
            >
              <SelectItem value="end_call" text="Disconnect the call" />
              <SelectItem value="resume_ai" text="Hand over to AI" />
            </Select>
            <Select
              id="transfer-ringtone"
              labelText={
                <span className="inline-flex items-center gap-1">
                  Ringtone
                  <Tooltip
                    align="right"
                    label="Ringtone played during the transfer flow."
                  >
                    <Information size={14} />
                  </Tooltip>
                </span>
              }
              value={getParamValue('tool.ringtone') || 'default'}
              onChange={e => updateParameter('tool.ringtone', e.target.value)}
            >
              <SelectItem value="default" text="default" />
              <SelectItem value="dial-and-ring" text="dial-and-ring" />
              <SelectItem value="ring-ring" text="ring-ring" />
              <SelectItem value="transfer-music" text="transfer-music" />
            </Select>
            <Slider
              id="transfer-delay"
              labelText={
                <span className="inline-flex items-center gap-1">
                  Transfer Delay (ms)
                  <Tooltip
                    align="right"
                    label="Wait time before starting the transfer flow."
                  >
                    <Information size={14} />
                  </Tooltip>
                </span>
              }
              min={500}
              max={3000}
              step={50}
              value={transferDelayValue}
              onChange={({ value }: { value: number }) =>
                updateParameter('tool.transfer_delay', value.toString())
              }
            />
          </Stack>
        </Stack>
      </InputGroup>

      {toolDefinition && onChangeToolDefinition && (
        <ToolDefinitionForm
          toolDefinition={toolDefinition}
          onChangeToolDefinition={onChangeToolDefinition}
          inputClass={inputClass}
          documentationPath="/assistants/tools/add-transfer-call-tool"
          documentationTitle="Know more about supported Transfer Call behavior"
        />
      )}
    </>
  );
};
