export function BorderedButton(props: {
  onClick: () => void;
  type: 'submit' | 'button';
  size?: 'xs' | 'sm' | 'lg' | 'xl';
  width?: 'w-full' | 'w-fit';
  color?: 'gray' | 'red' | 'blue';
  height?: 'h-10' | 'h-10' | 'h-9' | 'h-8';
  children?: any;
}) {
  const sizeClz = (size: 'xs' | 'sm' | 'lg' | 'xl' | undefined) => {
    if (size === 'xs') return 'px-3 py-[0.2rem] text-xs rounded-sm font-medium';
    if (size === 'sm') return 'px-4 rounded-[2px] text-sm font-medium';
    if (size === 'lg') return 'px-4 rounded-[2px] text-base font-medium';
    return 'px-3 py-2 rounded-[2px]';
  };

  const clrClz = (color: 'gray' | 'red' | 'blue' | undefined) => {
    if (color === 'red')
      return 'focus-visible:outline-red-500 border-red-500 hover:border-red-600  text-red-600';
    return 'focus-visible:outline-gray-700 border-gray-300 hover:border-gray-400 dark:border-gray-600 ';
  };

  const widthClz = (width: 'w-full' | 'w-fit' | undefined) => {
    if (!width) return 'w-full';
    return width;
  };

  const heightClz = (height?: 'h-10' | 'h-10' | 'h-9' | 'h-8') => {
    if (!height) return 'h-10';
    return height;
  };
  return (
    <button
      type={props.type}
      onClick={props.onClick}
      className={`flex ${
        props.width
      } justify-center items-center border ${widthClz(props.width)} ${clrClz(
        props.color,
      )} leading-6 ${heightClz(props.height)} ${sizeClz(props.size)}`}
    >
      {props.children}
    </button>
  );
}
