import { Metadata } from '@rapidaai/react';
import { BlueNoticeBlock } from '@/app/components/container/message/notice-block';
import { FC } from 'react';

export const EndpointOptions: FC<{ options: Array<Metadata> }> = ({
  options,
}) => {
  if (options.length <= 0)
    return (
      <BlueNoticeBlock>
        There are no options for given endpoint execution.
      </BlueNoticeBlock>
    );
  return (
    <div className="grid grid-cols-2 gap-2 m-4">
      {options.map((x, idx) => {
        return (
          <div
            className="flex justify-between items-center border-[0.5px] rounded-[2px]"
            key={`options-idx-${idx}`}
          >
            <div className="py-3 px-4 flex items-center gap-2">
              <span className="">{x.getKey()}</span>
            </div>
            <div className="py-3 px-4 ">
              <div className="flex items-center">{x.getValue()}</div>
            </div>
          </div>
        );
      })}
    </div>
  );
};
