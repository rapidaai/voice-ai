import { create } from 'zustand';
import { RapidaType } from '@/types';

/**
 * Proxy method to tell proxy the loading
 * @param boolean
 */
/**
 * loading context is parent context used in flexbox to show vertical loader in case the page is loading
 */

export const useRapidaStore = create<RapidaType>((set, get) => ({
  /**
   * default value of loading
   */
  loading: false,

  /**
   * show loading type line default
   */
  loadingType: 'line',

  /**
   *
   * @param loadingType
   */
  showLoader: (loadingType?: 'line' | 'overlay' | 'block') => {
    set({
      loading: true,
      loadingType: loadingType ? loadingType : 'line',
    });
  },

  /**
   * hide and reset loading type
   */
  hideLoader: () => {
    set({
      loading: false,
      loadingType: 'line',
    });
  },

  /**
   *
   * @returns
   */
  isBlocking: (): boolean => {
    return get().loadingType === 'block';
  },
}));
