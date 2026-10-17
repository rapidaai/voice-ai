// jest-dom adds custom jest matchers for asserting on DOM nodes.
// allows you to do things like:
// expect(element).toHaveTextContent(/react/i)
// learn more: https://github.com/testing-library/jest-dom
import '@testing-library/jest-dom/extend-expect';
import 'jest-styled-components';

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}

const resizeObserver =
  (globalThis as any).ResizeObserver || ResizeObserverMock;

// Carbon components may resolve ResizeObserver from either global or window.
(globalThis as any).ResizeObserver = resizeObserver;
if (typeof window !== 'undefined') {
  (window as any).ResizeObserver = resizeObserver;
}
