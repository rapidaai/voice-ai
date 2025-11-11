import React, { useEffect, useState } from 'react';
import { Switch } from '@/app/components/Form/Switch';
import { Label } from '@/app/components/Form/Label';
import { IBlueBGButton, ICancelButton } from '@/app/components/Form/Button';
import { ErrorMessage } from '@/app/components/Form/error-message';
import { ModalFooter } from '@/app/components/base/modal/modal-footer';
import { ModalBody } from '@/app/components/base/modal/modal-body';
import { GenericModal } from '@/app/components/base/modal';
import { ModalFitHeightBlock } from '@/app/components/blocks/modal-fit-height-block';
import { ModalHeader } from '@/app/components/base/modal/modal-header';
import { ModalTitleBlock } from '@/app/components/blocks/modal-title-block';

interface TablePreferenceModalProps {
  /**
   * show and hide modal
   */
  open: boolean;

  /**
   *
   * @param boolean
   * @returns
   */
  setOpen: (boolean) => void;

  /**
   * default page size
   */
  defaultPageSize: number[];
  /**
   *
   * The columns which is shown currently
   */
  columns: { name: string; key: string; visible: boolean }[];

  /**
   *
   * @param clmns
   * @returns
   */
  onChangeColumns: (
    clmns: { name: string; key: string; visible: boolean }[],
  ) => void;

  /**
   * Item per page
   */
  pageSize: number;

  /**
   * onChange of page
   */
  onChangePageSize: (number) => void;
}

export function ColumnPreferencesDialog(props: TablePreferenceModalProps) {
  /**
   * to control locally page size
   */
  const [pgs, setPgs] = useState(props.pageSize);

  useEffect(() => {
    setPgs(props.pageSize);
  }, [props.pageSize]);

  /**
   *
   * to control locally columns
   */

  const [clmns, setClmns] = useState<
    { name: string; key: string; visible: boolean }[]
  >([]);

  useEffect(() => {
    setClmns(props.columns);
  }, [props.columns]);

  /**
   *
   * @param k
   */
  const changeVisibility = (k: string) => {
    setClmns(prevClmns =>
      prevClmns.map(column =>
        column.key === k ? { ...column, visible: !column.visible } : column,
      ),
    );
  };

  /**
   *
   * @param e
   */
  const [error, setError] = useState('');

  const onAction = e => {
    let cnt = clmns.filter(x => {
      return x.visible;
    });
    if (cnt.length < 1 && clmns.length > 0) {
      setError('Please have 2 or more column visibility selected');
      return;
    }

    props.onChangePageSize(pgs);
    props.onChangeColumns(clmns);
    props.setOpen(false);
  };
  return (
    <GenericModal
      className="flex"
      modalOpen={props.open}
      setModalOpen={props.setOpen}
    >
      <ModalFitHeightBlock>
        <ModalHeader
          onClose={() => {
            props.setOpen(false);
          }}
          title={'Column Preferences'}
        >
          <ModalTitleBlock>Column Preferences</ModalTitleBlock>
        </ModalHeader>
        <ModalBody>
          {clmns.length > 0 && (
            <div>
              <h4 className="text-sm">Column preferences</h4>
              <ul className="my-2">
                {clmns.map((cl, idx) => {
                  return (
                    <li
                      className="flex cursor-pointer items-center justify-between border-b py-3 dark:border-gray-800"
                      key={idx}
                    >
                      <span
                        className="text-sm opacity-80 font-medium flex-1"
                        onClick={() => {
                          changeVisibility(cl.key);
                        }}
                      >
                        {cl.name}
                      </span>
                      <Switch
                        enable={cl.visible}
                        setEnable={() => {
                          changeVisibility(cl.key);
                        }}
                      />
                    </li>
                  );
                })}
              </ul>
            </div>
          )}
          {/*  */}
          <div>
            <h1 className="text-sm">Page Size</h1>
            <ul className="my-2">
              {props.defaultPageSize.map((sz, idx) => {
                return (
                  <li
                    className="flex items-center space-x-2 py-1"
                    key={`page-size-${idx}`}
                  >
                    <input
                      type="radio"
                      value={sz}
                      name="page-size"
                      id={`page-size-${idx}`}
                      checked={sz === pgs}
                      onChange={t => {
                        setPgs(sz);
                      }}
                      className="w-4 h-4 text-blue-600 bg-gray-100 border-gray-300 focus:ring-blue-500 dark:focus:ring-blue-600 dark:ring-offset-gray-800 dark:bg-gray-700 dark:border-gray-600"
                    />
                    <Label
                      for={`page-size-${idx}`}
                      text={`${sz} Items`}
                    ></Label>
                  </li>
                );
              })}
            </ul>
          </div>
          <ErrorMessage message={error} />
        </ModalBody>
        <ModalFooter>
          <ICancelButton
            className="px-4 rounded-[2px]"
            onClick={() => {
              props.setOpen(false);
            }}
          >
            Cancel
          </ICancelButton>
          <IBlueBGButton
            className="px-4 rounded-[2px]"
            type="button"
            onClick={onAction}
          >
            Save Preference
          </IBlueBGButton>
        </ModalFooter>
      </ModalFitHeightBlock>
    </GenericModal>
  );
}
