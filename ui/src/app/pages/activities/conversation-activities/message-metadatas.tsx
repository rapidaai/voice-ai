import { Metadata } from '@rapidaai/react';
import { BlueNoticeBlock } from '@/app/components/container/message/notice-block';
import { FC } from 'react';

export const MessageMetadatas: FC<{ metadata: Array<Metadata> }> = ({
  metadata,
}) => {
  if (metadata.length <= 0)
    return (
      <BlueNoticeBlock>There are no metdata for given message.</BlueNoticeBlock>
    );
  return (
    <div className="grid grid-cols-2 gap-2 m-4">
      {metadata.map((x, idx) => {
        return (
          <div
            className="flex justify-between items-center border-[0.5px] rounded-[2px]"
            key={`metadata-idx-${idx}`}
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
