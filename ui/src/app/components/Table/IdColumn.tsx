import { CustomLink, CustomLinkProps } from '@/app/components/custom-link';
import { TD } from '@/app/components/Table/Body';

/**
 *
 * @param props
 * @returns
 */
export function IdColumn(props: CustomLinkProps) {
  /**
   *
   */
  return (
    <TD className="underline underline-offset-2 hover:text-blue-600 text-blue-500 text-[15px]">
      <CustomLink to={props.to} className={props.className} {...props}>
        {props.children}
      </CustomLink>
    </TD>
  );
}
