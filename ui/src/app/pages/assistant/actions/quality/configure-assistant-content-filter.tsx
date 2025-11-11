import { PageActionButtonBlock } from '@/app/components/blocks/page-action-button-block';
import { PageHeaderBlock } from '@/app/components/blocks/page-header-block';
import { PageTitleBlock } from '@/app/components/blocks/page-title-block';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { Select } from '@/app/components/Form/Select';
import { Slider } from '@/app/components/Form/Slider';
import { InputGroup } from '@/app/components/input-group';
import { useState } from 'react';

const harmfulCategories = [
  {
    key: 'hate',
    label: 'Hate',
  },
  {
    key: 'insults',
    label: 'Insults',
  },
  {
    key: 'sexual',
    label: 'Sexual',
  },
  {
    key: 'violence',
    label: 'Violence',
  },
  {
    key: 'misconduct',
    label: 'Misconduct',
  },
];

function HarmfulCategoriesTable() {
  return (
    <div>
      {harmfulCategories.map(cat => (
        <div
          key={cat.key}
          className="flex items-center border-b border-muted last:border-b-0 py-2"
        >
          <div className="flex items-center w-[120px]">
            <input type="checkbox" checked={true} className="mr-2" />
            <span className="font-medium">{cat.label}</span>
          </div>
          <div className="ml-4 w-[120px]">
            <Select
              placeholder={'Action'}
              onChange={e => {
                //   setGroundingAction(e.target.value);
              }}
              className="text-sm! h-9 pl-3 bg-light-background"
              // value={groundingAction}
              options={[
                { name: 'Block', value: 'block' },
                { name: 'Allow', value: 'allow' },
              ]}
            />
          </div>
          <div className="flex-1 ml-4">
            <Slider onSlide={e => {}} min={0.0} max={1.0} step={0.01} />
            <div className="flex text-xs text-muted-foreground justify-between mt-1 px-1">
              <span>0.0</span>
              <span>0.25</span>
              <span>0.5</span>
              <span>0.75</span>
              <span>1.0</span>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

const HarmfulCategoriesBlock = () => {
  return (
    <>
      <div>
        <h3 className="text-md font-semibold mb-2">Input</h3>
        <HarmfulCategoriesTable />
      </div>
      <div>
        <h3 className="text-md font-semibold mb-2">Output</h3>
        <HarmfulCategoriesTable />
      </div>
    </>
  );
};

function PromptAttacksBlock() {
  // You can further split to another component if this section grows.
  return (
    <>
      {/* Placeholder table with two example attacks */}

      <div className="flex items-center border-b border-muted py-1 px-1">
        <div className="w-[160px] font-medium">Jailbreak</div>
        <div className="ml-4 w-[120px]">
          <Select
            placeholder={'Action'}
            onChange={e => {
              //   setGroundingAction(e.target.value);
            }}
            className="text-sm! h-9 pl-3 bg-light-background"
            // value={groundingAction}
            options={[
              { name: 'Block', value: 'block' },
              { name: 'Allow', value: 'allow' },
            ]}
          />
        </div>
        <div className="flex-1 ml-4">
          <Slider onSlide={e => {}} min={0.0} max={1.0} step={0.01} />
          <div className="flex text-xs text-muted-foreground justify-between mt-1 px-1">
            <span>0.0</span>
            <span>0.25</span>
            <span>0.5</span>
            <span>0.75</span>
            <span>1.0</span>
          </div>
        </div>
      </div>
      <div className="flex items-center py-1 px-1">
        <div className="w-[160px] font-medium">Prompt injection</div>
        <div className="ml-4 w-[120px]">
          <Select
            placeholder={'Action'}
            onChange={e => {
              //   setGroundingAction(e.target.value);
            }}
            className="text-sm! h-9 pl-3"
            // value={groundingAction}
            options={[
              { name: 'Block', value: 'block' },
              { name: 'Allow', value: 'allow' },
            ]}
          />
        </div>
        <div className="flex-1 ml-4">
          <Slider onSlide={e => {}} min={0.0} max={1.0} step={0.01} />
          <div className="flex text-xs text-muted-foreground justify-between mt-1 px-1">
            <span>0.0</span>
            <span>0.25</span>
            <span>0.5</span>
            <span>0.75</span>
            <span>1.0</span>
          </div>
        </div>
      </div>
    </>
  );
}

export const ConfigureAssistantContentFilterPage = () => {
  const [errorMessage, setErrorMessage] = useState('');
  return (
    <div className="relative flex flex-col flex-1">
      <PageHeaderBlock>
        <PageTitleBlock>Content moderation & filter</PageTitleBlock>
      </PageHeaderBlock>
      <div className="overflow-auto flex flex-col flex-1 pb-20">
        <div className="bg-white dark:bg-gray-900">
          <InputGroup title="Harmful categories">
            <div className="p-6 space-y-6">
              <HarmfulCategoriesBlock />
            </div>
          </InputGroup>
          <InputGroup title="Prompt attacks">
            <div className="p-6 space-y-6">
              <PromptAttacksBlock />
            </div>
          </InputGroup>
        </div>
      </div>
      <PageActionButtonBlock errorMessage={errorMessage}>
        <ICancelButton
          className="px-4 rounded-[2px]"
          onClick={() => {
            //   goToAssistant(assistantId);
          }}
        >
          Cancel
        </ICancelButton>
        <IBlueBGButton type="submit" className="px-4 rounded-[2px]">
          Configure filter
        </IBlueBGButton>
      </PageActionButtonBlock>
    </div>
  );
};
