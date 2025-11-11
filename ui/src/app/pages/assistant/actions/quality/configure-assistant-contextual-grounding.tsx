import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { FormLabel } from '@/app/components/form-label';
import {
  IBlueBGButton,
  IBlueBorderButton,
  ICancelButton,
} from '@/app/components/Form/Button';
import { FieldSet } from '@/app/components/Form/Fieldset';
import { Input } from '@/app/components/Form/Input';
import { Select } from '@/app/components/Form/Select';
import { Slider } from '@/app/components/Form/Slider';
import { SwitchWithLabel } from '@/app/components/Form/Switch';
import { InputGroup } from '@/app/components/input-group';
import { InputHelper } from '@/app/components/input-helper';
import { useState } from 'react';

export const ConfigureAssistantContextualGroundingPage = () => {
  // State management for Grounding
  const [groundingEnabled, setGroundingEnabled] = useState(true);
  const [groundingThreshold, setGroundingThreshold] = useState(0.7);
  const [groundingAction, setGroundingAction] = useState('block');

  // State management for Relevance
  const [relevanceEnabled, setRelevanceEnabled] = useState(true);
  const [relevanceThreshold, setRelevanceThreshold] = useState(0.7);
  const [relevanceAction, setRelevanceAction] = useState('block');

  // Reset functions
  const resetGrounding = () => setGroundingThreshold(0.7);
  const resetRelevance = () => setRelevanceThreshold(0.7);

  return (
    <div className="relative flex flex-col flex-1">
      <PageHeaderBlock>
        <PageTitleBlock>Contextual grounding</PageTitleBlock>
      </PageHeaderBlock>

      <div className="overflow-auto flex flex-col flex-1 pb-20">
        <div className=" bg-white dark:bg-gray-900">
          <InputGroup title="Grounding">
            <div className="p-6 space-y-6">
              <FieldSet>
                <SwitchWithLabel
                  enable={groundingEnabled}
                  setEnable={setGroundingEnabled}
                  label="Enable grounding check"
                  className="bg-light-background"
                ></SwitchWithLabel>
                <InputHelper>
                  Grounding score represents the confidence that the model
                  response is factually correct and grounded in the source. If
                  the model response has a lower score than the defined
                  threshold, the response will be blocked and the configured
                  blocked message will be returned to the user. A higher
                  threshold level blocks more responses.
                </InputHelper>
              </FieldSet>
              <FieldSet className="flex flex-row items-center gap-3">
                <Slider
                  min={0}
                  max={0.99}
                  step={0.01}
                  value={groundingThreshold}
                  onSlide={v => setGroundingThreshold(Number(v.toFixed(2)))}
                  className="flex-1"
                />
                <Input
                  type="number"
                  min={0}
                  max={0.99}
                  step={0.01}
                  value={groundingThreshold}
                  onChange={e => {
                    let v = Math.max(0, Math.min(0.99, Number(e.target.value)));
                    setGroundingThreshold(Number(v.toFixed(2)));
                  }}
                  className="w-16 ml-2 h-9 bg-light-background"
                />
                <IBlueBorderButton
                  className="w-fit shrink-0 pe-4"
                  onClick={resetGrounding}
                >
                  Reset
                </IBlueBorderButton>
              </FieldSet>
              <FieldSet className="flex justify-between">
                <FormLabel>Contextual grounding action</FormLabel>
                <div className="w-40">
                  <Select
                    placeholder={'Action'}
                    onChange={e => {
                      setGroundingAction(e.target.value);
                    }}
                    className="text-sm! bg-light-background"
                    value={groundingAction}
                    options={[
                      { name: 'Block', value: 'block' },
                      { name: 'Allow', value: 'allow' },
                    ]}
                  ></Select>
                </div>
                <InputHelper>
                  Choose what action the guardrail should take on contextual
                  grounding check.
                </InputHelper>
              </FieldSet>
            </div>
          </InputGroup>

          <InputGroup title="Relevance">
            <div className="p-6 space-y-6">
              <FieldSet>
                <SwitchWithLabel
                  className="bg-light-background"
                  enable={relevanceEnabled}
                  setEnable={setRelevanceEnabled}
                  label="Enable relevance check"
                ></SwitchWithLabel>
                <InputHelper>
                  Relevance score represents the confidence that the model
                  response is relevant to the user's query. If the model
                  response has a lower score than the defined threshold, the
                  response will be blocked and the configured blocked message
                  will be returned to the user. A higher threshold level blocks
                  more responses.
                </InputHelper>
              </FieldSet>
              <FieldSet className="flex flex-row items-center gap-3">
                <Slider
                  min={0}
                  max={0.99}
                  step={0.01}
                  value={groundingThreshold}
                  onSlide={v => setRelevanceThreshold(Number(v.toFixed(2)))}
                  className="flex-1"
                />
                <Input
                  type="number"
                  min={0}
                  max={0.99}
                  step={0.01}
                  value={relevanceThreshold}
                  onChange={e => {
                    let v = Math.max(0, Math.min(0.99, Number(e.target.value)));
                    setRelevanceThreshold(Number(v.toFixed(2)));
                  }}
                  className="w-16 ml-2 h-9 bg-light-background"
                />
                <IBlueBorderButton
                  className="w-fit shrink-0 pe-4"
                  onClick={resetRelevance}
                >
                  Reset
                </IBlueBorderButton>
              </FieldSet>
              <FieldSet className="flex justify-between">
                <FormLabel>Relevance action</FormLabel>
                <div className="w-40">
                  <Select
                    placeholder={'Action'}
                    onChange={e => {
                      setRelevanceAction(e.target.value);
                    }}
                    className="text-sm! bg-light-background"
                    value={relevanceAction}
                    options={[
                      { name: 'Block', value: 'block' },
                      { name: 'Allow', value: 'allow' },
                    ]}
                  ></Select>
                </div>
                <InputHelper>
                  Choose what action the guardrail should take on relevance
                  check.
                </InputHelper>
              </FieldSet>
            </div>
          </InputGroup>
        </div>
      </div>

      <PageActionButtonBlock errorMessage={''}>
        <ICancelButton
          className="px-4 rounded-[2px]"
          onClick={() => {
            //   goToAssistant(assistantId);
          }}
        >
          Cancel
        </ICancelButton>
        <IBlueBGButton type="submit" className="px-4 rounded-[2px]">
          Configure grounding
        </IBlueBGButton>
      </PageActionButtonBlock>
    </div>
  );
};
