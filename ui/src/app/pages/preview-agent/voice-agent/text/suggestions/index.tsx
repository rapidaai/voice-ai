import { FC } from 'react';

export const QuickSuggestion: FC<{
  suggestion: string;
  onClick: () => void;
}> = ({ onClick, suggestion }) => {
  return (
    <button
      type="button"
      className="w-fit focus:[outline-style:none] focus-visible:[outline-style:none] data-focus-visible:outline-solid data-focus-visible:outline-3 data-[focus-visible]:outline-focus disabled:cursor-default! overflow-hidden font-medium transition-colors relative before:absolute before:inset-0 before:pointer-events-none cursor-pointer bg-primary/10  group inline-flex justify-between items-center [--icon-size:var(--line-height)] [--avatar-size:var(--line-height)] flex-none text-sm h-9 px-3 [--avatar-font-size:var(--text-xs)] rounded-[2px] before:rounded-[2px] chat-question max-w-full *:data-label:text-start hover:bg-primary/20 text-primary shrink-0"
      onClick={onClick}
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        aria-hidden="true"
        viewBox="0 0 24 24"
        style={{
          width: 'var(--icon-size, 24px)',
          height: 'var(--icon-size, 24px)',
        }}
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          d="M12 18.25c3.866 0 7.25-2.095 7.25-6.75 0-4.655-3.384-6.75-7.25-6.75S4.75 6.845 4.75 11.5c0 1.768.488 3.166 1.305 4.22.239.31.334.72.168 1.073-.1.215-.207.42-.315.615-.454.816.172 2.005 1.087 1.822 1.016-.204 2.153-.508 3.1-.956a1.15 1.15 0 0 1 .635-.103c.415.053.84.079 1.27.079Z"
        />
      </svg>
      <span className="flex truncate only:mx-auto px-1">{suggestion}</span>
    </button>
  );
};
