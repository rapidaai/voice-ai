import type { Meta, StoryObj } from '@storybook/react-vite';
import {
  Add,
  Edit,
  Information,
  SettingsAdjust,
  TrashCan,
} from '@carbon/icons-react';
import { Button, Stack } from '@carbon/react';
import { useState } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { Breadcrumb } from './breadcrumb';
import {
  ActionCard,
  BaseCard,
  Card,
  CardDescription,
  CardTag,
  CardTitle,
  ClickableCard,
} from './card';
import CheckboxCard from './checkbox-card';
import { Disclosure } from './disclosure';
import { Dropdown, MultiSelect } from './dropdown';
import { FieldSet } from './fieldset';
import { Form, FormGroup, TextArea, TextInput } from './form';
import { Input } from './input';
import { InputCheckbox } from './input-checkbox';
import { InputGroup } from './input-group';
import { Label } from './label';
import {
  DangerButton,
  DangerGhostButton,
  GhostButton,
  IconButtonWithBadge,
  IconOnlyButton,
  PrimaryButton,
  SecondaryButton,
  TertiaryButton,
} from './button';
import { OverflowMenu, OverflowMenuItem } from './overflow-menu';
import { Pagination } from './pagination';
import { Pill, MultiplePills } from './pill';
import { Select } from './select';
import { Slider } from './slider';
import { Switch, SwitchWithLabel } from './switch';
import { Tabs } from './tabs';
import { Text } from './text';
import { Textarea } from './textarea';
import { Tile } from './tile';
import { Tooltip } from './tooltip';

const meta = {
  title: 'UI/Primitives',
  component: PrimaryButton,
  tags: ['autodocs'],
  args: {
    size: 'md',
  },
  parameters: {
    docs: {
      description: {
        component:
          'Shared UI primitives built on IBM Carbon components and app compatibility wrappers.',
      },
    },
  },
  decorators: [
    Story => (
      <MemoryRouter>
        <Story />
      </MemoryRouter>
    ),
  ],
} satisfies Meta<typeof PrimaryButton>;

export default meta;

type Story = StoryObj<typeof meta>;

const fruits = ['Apple', 'Banana', 'Orange'];

export const Buttons: Story = {
  render: () => (
    <Stack gap={5}>
      <div className="flex flex-wrap items-center gap-3">
        <PrimaryButton size="md" renderIcon={Add}>
          Create
        </PrimaryButton>
        <SecondaryButton size="md">Secondary</SecondaryButton>
        <TertiaryButton size="md">Tertiary</TertiaryButton>
        <GhostButton size="md">Ghost</GhostButton>
        <DangerButton size="md">Delete</DangerButton>
        <DangerGhostButton size="md">Remove</DangerGhostButton>
      </div>
      <div className="flex flex-wrap items-center gap-3">
        <IconOnlyButton
          kind="ghost"
          size="md"
          renderIcon={SettingsAdjust}
          iconDescription="Settings"
        />
        <IconButtonWithBadge
          kind="ghost"
          size="md"
          renderIcon={Information}
          iconDescription="Notifications"
          badgeCount={7}
        />
      </div>
    </Stack>
  ),
};

export const Forms: Story = {
  render: function Render() {
    const [enabled, setEnabled] = useState(true);
    const [sliderValue, setSliderValue] = useState(40);

    return (
      <Form className="max-w-3xl" onSubmit={event => event.preventDefault()}>
        <Stack gap={6}>
          <FormGroup legendText="Workspace">
            <FieldSet>
              <TextInput
                id="workspace-name"
                labelText="Workspace name"
                defaultValue="Support automation"
                helperText="Carbon TextInput through the shared form wrapper."
              />
              <TextArea
                id="workspace-summary"
                labelText="Summary"
                defaultValue="Routes support conversations to the right assistant."
                rows={3}
              />
            </FieldSet>
          </FormGroup>
          <div className="grid gap-4 md:grid-cols-2">
            <Input
              id="primitive-input"
              labelText="Compact input"
              hideLabel={false}
              placeholder="Assistant name"
            />
            <Textarea
              id="primitive-textarea"
              labelText="Compact textarea"
              hideLabel={false}
              placeholder="Describe behavior"
              rows={3}
            />
          </div>
          <div className="grid gap-4 md:grid-cols-2">
            <Select
              id="primitive-select"
              labelText="Environment"
              hideLabel={false}
              placeholder="Choose environment"
              options={[
                { name: 'Development', value: 'dev' },
                { name: 'Production', value: 'prod' },
              ]}
            />
            <Dropdown
              id="primitive-dropdown"
              titleText="Fruit"
              label="Choose fruit"
              items={fruits}
              selectedItem="Apple"
            />
          </div>
          <MultiSelect
            id="primitive-multiselect"
            titleText="Channels"
            label="Choose channels"
            items={['Web', 'Phone', 'API']}
            initialSelectedItems={['Web']}
          />
          <Slider
            id="primitive-slider"
            labelText="Confidence threshold"
            hideLabel={false}
            min={0}
            max={100}
            value={sliderValue}
            onSlide={setSliderValue}
          />
          <div className="flex flex-wrap items-center gap-6">
            <InputCheckbox id="primitive-checkbox">
              Enable fallback
            </InputCheckbox>
            <Switch enable={enabled} setEnable={setEnabled} label="Enabled" />
            <SwitchWithLabel
              enable={enabled}
              setEnable={setEnabled}
              label="Require approval"
              className="max-w-sm"
            />
          </div>
        </Stack>
      </Form>
    );
  },
};

export const CardsAndContent: Story = {
  render: function Render() {
    const [selected, setSelected] = useState(true);

    return (
      <div className="grid max-w-5xl gap-4 md:grid-cols-3">
        <BaseCard className="p-4">
          <CardTitle title="Static card" status="active" />
          <CardDescription description="A shared non-interactive card container." />
          <CardTag tags={['assistant', 'production']} />
        </BaseCard>
        <ActionCard className="p-4" onClick={() => setSelected(!selected)}>
          <CardTitle title="Action card" />
          <CardDescription>
            Click or press Enter to toggle selection state.
          </CardDescription>
        </ActionCard>
        <ClickableCard to="/components" className="p-4">
          <CardTitle title="Link card" />
          <CardDescription description="Uses react-router navigation." />
        </ClickableCard>
        <Card className="gap-3">
          <Text as="h3" className="text-base font-semibold">
            Inline content
          </Text>
          <div className="flex items-center gap-2">
            <Pill>stable</Pill>
            <MultiplePills
              tags={['telephony', 'webhook', 'analytics']}
              items={1}
            />
          </div>
        </Card>
        <Tile>
          <Text as="p">
            Carbon tile wrapper for simple content blocks and loading states.
          </Text>
        </Tile>
        <CheckboxCard
          id="primitive-checkbox-card"
          label="Selectable tile"
          checked={selected}
          onChange={() => setSelected(!selected)}
        />
      </div>
    );
  },
};

export const NavigationAndActions: Story = {
  render: function Render() {
    const [page, setPage] = useState(1);

    return (
      <Stack gap={6} className="max-w-4xl">
        <Breadcrumb
          items={[
            { label: 'Components', href: '#' },
            { label: 'UI', href: '#' },
            { label: 'Primitives' },
          ]}
        />
        <div className="flex flex-wrap items-center gap-4">
          <OverflowMenu size="md" iconDescription="Open actions">
            <OverflowMenuItem itemText="Edit" />
            <OverflowMenuItem itemText="Delete" isDelete hasDivider />
          </OverflowMenu>
          <Tooltip icon={<Button renderIcon={Information}>Details</Button>}>
            Primitive tooltip content
          </Tooltip>
        </div>
        <Pagination
          totalItems={52}
          page={page}
          pageSize={10}
          pageSizes={[10, 20, 50]}
          onChange={({ page: nextPage }) => setPage(nextPage)}
        />
      </Stack>
    );
  },
};

export const DisclosureAndTabs: Story = {
  render: function Render() {
    const [open, setOpen] = useState(true);

    return (
      <Stack gap={5} className="max-w-3xl">
        <InputGroup title="Advanced settings" initiallyExpanded>
          <Label text="Routing policy" />
          <p className="text-sm text-[var(--cds-text-secondary)]">
            Grouped inputs can be collapsed while preserving form structure.
          </p>
        </InputGroup>
        <Button
          kind="tertiary"
          size="sm"
          renderIcon={open ? TrashCan : Edit}
          onClick={() => setOpen(current => !current)}
        >
          Toggle disclosure
        </Button>
        <Disclosure open={open}>
          <Tile>Disclosure renders children only when open.</Tile>
        </Disclosure>
        <Tabs
          tabs={['Overview', 'Metrics', 'Logs']}
          aria-label="Primitive tabs"
        >
          <div className="p-4">Overview panel</div>
          <div className="p-4">Metrics panel</div>
          <div className="p-4">Logs panel</div>
        </Tabs>
      </Stack>
    );
  },
};
