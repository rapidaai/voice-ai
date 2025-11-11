export type ColumnarType = {
  /**
   * list of columns that will get display
   */
  columns: { name: string; key: string; visible: boolean }[];

  /**
   *
   * @param cl
   * @returns
   */
  setColumns: (cl: { name: string; key: string; visible: boolean }[]) => void;

  /**
   *
   * @param k
   * @returns
   */
  visibleColumn: (k: string) => boolean;
};
