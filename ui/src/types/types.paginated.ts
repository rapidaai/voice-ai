export const initialPaginatedState = {
  /**
   *
   */
  page: 1,

  /**
   *
   */
  pageSize: 20,

  /**
   *
   */
  totalCount: 0,

  /**
   *
   */
  criteria: [],
};

export const initialPaginated = {
  ...initialPaginatedState,

  /**
   *
   */
  columns: [],

  addCriteria: function (key: string, value: string, logic: string): void {
    throw new Error('Function not implemented.');
  },

  addCriterias: function (v: { k: string; v: string; logic: string }[]): void {
    throw new Error('Function not implemented.');
  },

  clearCriteria: function (): void {
    throw new Error('Function not implemented.');
  },

  setCriterias: function (v: { k: string; v: string; logic: string }[]): void {
    throw new Error('Function not implemented.');
  },

  setTotalCount: function (number: any): void {
    throw new Error('Function not implemented.');
  },
  setPage: function (number: any): void {
    throw new Error('Function not implemented.');
  },
  setPageSize: function (number: any): void {
    throw new Error('Function not implemented.');
  },
  removeCriteria: function (number: any): void {
    throw new Error('Function not implemented.');
  },
  setColumns: function (
    cl: { name: string; key: string; visible: boolean }[],
  ): void {
    throw new Error('Function not implemented.');
  },
  visibleColumn: function (k: string): boolean {
    throw new Error('Function not implemented.');
  },
};

/**
 *
 */
export type PaginatedType = {
  /**
   * page
   */
  page: number;

  /**
   * page size
   */
  pageSize: number;

  /**
   * total count
   */
  totalCount: number;

  /**
   *
   */
  criteria: { key: string; value: string; logic: string }[];

  /**
   *
   * @param key
   * @param value
   * @returns
   */
  addCriteria: (key: string, value: string, logic: string) => void;
  /**
   *
   * @param v
   * @returns
   */
  setCriterias: (v: { k: string; v: string; logic: string }[]) => void;
  /**
   *
   */
  addCriterias: (v: { k: string; v: string; logic: string }[]) => void;

  /**
   *
   */
  removeCriteria: (key: string) => void;
  /**
   *
   * @returns
   */
  clearCriteria: () => void;

  /**
   *
   * @param number
   * @returns
   */
  setTotalCount: (number) => void;

  /**
   *
   * @param number
   * @returns
   */
  setPage: (number) => void;

  /**
   *
   * @param number
   * @returns
   */
  setPageSize: (number) => void;
};
