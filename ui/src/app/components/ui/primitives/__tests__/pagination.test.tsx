import { render } from '@testing-library/react';
import '@testing-library/jest-dom';

import { Pagination } from '../pagination';

const paginationProps = {
  totalItems: 30,
  page: 1,
  pageSize: 10,
  pageSizes: [10, 20, 50],
  onChange: jest.fn(),
};

describe('Pagination', () => {
  it('keeps page-level pagination sticky at the bottom', () => {
    const { container } = render(
      <Pagination {...paginationProps} sticky className="custom-pagination" />,
    );

    expect(container.querySelector('.cds--pagination')).toHaveClass(
      'sticky',
      'bottom-0',
      'z-10',
      'mt-auto!',
      'shrink-0',
      'custom-pagination',
    );
  });

  it('leaves embedded pagination in normal document flow by default', () => {
    const { container } = render(<Pagination {...paginationProps} />);

    expect(container.querySelector('.cds--pagination')).not.toHaveClass(
      'sticky',
      'bottom-0',
      'z-10',
    );
  });
});
