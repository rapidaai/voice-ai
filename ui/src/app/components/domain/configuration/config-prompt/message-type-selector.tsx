import { memo } from 'react';
import { Dropdown } from '@carbon/react';
import { PromptRole } from '@/models/prompt';

type Props = {
  value?: PromptRole;
  onChange: (value: PromptRole) => void;
};

const allTypes = [PromptRole.system, PromptRole.user, PromptRole.assistant];

const getPromptRoleName = (item: PromptRole | null): string => item ?? '';

function MessageTypeSelector({ value, onChange }: Props) {
  return (
    <Dropdown
      id="prompt-message-role"
      className="w-full min-w-[140px] [&_.cds--list-box]:!w-full"
      titleText="Message role"
      hideLabel
      label="Select a role"
      items={allTypes}
      selectedItem={value ?? null}
      itemToString={getPromptRoleName}
      onChange={({ selectedItem }: { selectedItem: PromptRole | null }) => {
        if (selectedItem) onChange(selectedItem);
      }}
    />
  );
}

export default memo(MessageTypeSelector);
