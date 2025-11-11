import type { FC } from 'react';
// import React, { useRef } from 'react';
// import { useBoolean } from 'ahooks';
// import { IButton } from '@/app/components/Form/Button';
// import { CloseIcon } from '@/app/components/Icon/Close';
// import { ExpandIcon } from '@/app/components/Icon/Expand';
// import { TickIcon } from '@/app/components/Icon/Tick';
// import { CopyIcon } from '@/app/components/Icon/Copy';
// import { cn } from '@/styles/media';
// import { useToggleExpend } from '@/hooks/use-toggle-expend';
// import PromptEditor from '@/app/components/prompt-editor';

// // Prommpt editor //
// type SimplePromptInputProps = {
//   placeholder: string;
//   value: string;
//   onChange: (value: string) => void;
//   className?: string;
//   editorClass?: string;
//   instanceId?: string;
// };

// const SimplePromptInput: FC<SimplePromptInputProps> = ({
//   placeholder,
//   value,
//   onChange,
//   editorClass,
//   instanceId,
// }) => {
//   // expand feature
//   const ref = useRef<HTMLDivElement>(null);
//   const { isExpand, setIsExpand } = useToggleExpend(ref);
//   const [isFocus, { setTrue: setFocus, setFalse: setBlur }] = useBoolean(false);
//   const [isChecked, { setTrue: setChecked, setFalse: setUnCheck }] =
//     useBoolean(false);
//   const handlePromptChange = (newValue: string) => {
//     if (value === newValue) return;
//     onChange(newValue);
//   };

//   const copyItem = (item: string) => {
//     setChecked();
//     navigator.clipboard.writeText(item);
//     setTimeout(() => {
//       setUnCheck();
//     }, 4000); // Reset back after 2 seconds
//   };

//   return (
//     <div
//       ref={ref}
//       className={cn(
//         'group',
//         'outline-solid outline-[1.5px] outline-transparent',
//         'focus-within:outline-blue-600 focus:outline-blue-600 outline-offset-[-1.5px]',
//         'border-b border-gray-400 dark:border-gray-600',
//         'dark:focus:border-blue-600 focus:border-blue-600',
//         'transition-all duration-200 ease-in-out',
//         'relative',
//         'bg-white dark:bg-gray-950',
//         isFocus && 'border-blue-600! outline-blue-600! ',
//         isExpand
//           ? 'fixed top-0 bottom-0 right-0 left-0 h-full z-50 m-0! p-0!'
//           : '',
//       )}
//     >
//       <div
//         className={cn(
//           'flex items-center absolute right-1 top-1 z-20 invisible group-hover:visible bg-light-background dark:bg-gray-900',
//         )}
//       >
//         <IButton
//           className="h-8"
//           tabIndex={-1}
//           onClick={() => {
//             copyItem(value);
//           }}
//         >
//           {isChecked ? (
//             <TickIcon className="h-3.5 w-3.5 text-green-600" />
//           ) : (
//             <CopyIcon className="h-3.5 w-3.5" />
//           )}
//         </IButton>
//         <IButton
//           className="h-8"
//           tabIndex={-1}
//           onClick={() => {
//             setIsExpand(!isExpand);
//           }}
//         >
//           {isExpand ? (
//             <CloseIcon className="h-3.5 w-3.5" />
//           ) : (
//             <ExpandIcon className="h-3.5 w-3.5" />
//           )}
//         </IButton>
//       </div>

//       <PromptEditor
//         className={cn(
//           'min-h-52 overflow-auto p-2',
//           editorClass,
//           isExpand && 'h-screen p-4',
//         )}
//         placeholder={placeholder}
//         value={value}
//         onFocus={setFocus}
//         onChange={handlePromptChange}
//         onBlur={setBlur}
//       />
//     </div>
//   );
// };
// export default SimplePromptInput;
