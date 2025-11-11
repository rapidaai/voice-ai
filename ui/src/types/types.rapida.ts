export type RapidaType = {
  /**
   * show the loading or not
   */
  loading: boolean;

  /**
   * type of loading
   * line -> show the line after the status bar
   * overlay -> block the complete view by overlay and loading
   */
  loadingType: 'line' | 'overlay' | 'block';

  /**
   * show loading
   * @param loadingType
   * @returns
   */
  showLoader: (loadingType?: 'line' | 'overlay' | 'block') => void;

  /**
   * hide any loader that is showing
   * @returns
   */
  hideLoader: () => void;

  /**
   *
   * @returns boolean
   */
  isBlocking: () => boolean;
};
