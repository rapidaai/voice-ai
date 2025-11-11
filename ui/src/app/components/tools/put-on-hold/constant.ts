import { Metadata } from '@rapidaai/react';
import { SetMetadata } from '@/utils/metadata';

export const PutOnHoldToolDefintion = {
  name: 'put_on_hold',
  description:
    'Use this tool to temporarily pause a process or task. Specify the reason for putting it on hold along with any relevant context.',
  parameters: JSON.stringify(
    {
      properties: {
        reason: {
          description: 'The reason for putting the process or task on hold.',
          type: 'string',
        },
      },
      required: ['reason'],
      type: 'object',
    },
    null,
    2,
  ),
};

export const GetPutOnHoldDefaultOptions = (current: Metadata[]): Metadata[] => {
  const mtds: Metadata[] = [];

  const keysToKeep = ['tool.max_hold_time'];

  const addMetadata = (
    key: string,
    defaultValue?: string,
    validationFn?: (value: string) => boolean,
  ) => {
    const metadata = SetMetadata(current, key, defaultValue, validationFn);
    if (metadata) mtds.push(metadata);
  };

  addMetadata('tool.max_hold_time', '5');
  return mtds.filter(m => keysToKeep.includes(m.getKey()));
};

export const ValidatePutOnHoldDefaultOptions = (
  options: Metadata[],
): boolean => {
  const maxHoldTimeSec = options
    .find(m => m.getKey() === 'tool.max_hold_time')
    ?.getValue();

  if (maxHoldTimeSec) {
    const holdTime = parseInt(maxHoldTimeSec, 10);
    if (isNaN(holdTime) || holdTime < 1 || holdTime > 10) {
      return false; // Invalid if not a number or outside the range of 1-10 minutes
    }
  }

  return true;
};
