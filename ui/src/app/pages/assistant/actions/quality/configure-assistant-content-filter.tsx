import { PageActionButtonBlock } from '@/app/components/layout/blocks/page-action-button-block';
import { CompactPageHeader } from '@/app/components/layout/compact-page-header';
import { PageTitleBlock } from '@/app/components/layout/blocks/page-title-block';
import {
  GhostButton,
  InputCheckbox,
  PrimaryButton,
} from '@/app/components/ui/primitives';
import { Select } from '@/app/components/ui/primitives';
import { Slider } from '@/app/components/ui/primitives';
import { InputGroup } from '@/app/components/ui/primitives';

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
            <InputCheckbox
              checked
              readOnly
              aria-label={`${cat.label} category enabled`}
              className="mr-2"
            />
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
  return (
    <div className="relative flex flex-col flex-1">
      <CompactPageHeader>
        <PageTitleBlock>Content moderation & filter</PageTitleBlock>
      </CompactPageHeader>
      <div className="overflow-auto flex flex-col flex-1 pb-20">
        <div className="bg-surface ">
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
        <GhostButton
          size="md"
          onClick={() => {
            //   goToAssistant(assistantId);
          }}
        >
          Cancel
        </GhostButton>
        <PrimaryButton size="md" type="submit">
          Configure filter
        </PrimaryButton>
      </PageActionButtonBlock>
    </div>
  );
};
