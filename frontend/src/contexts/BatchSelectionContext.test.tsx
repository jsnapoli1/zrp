import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BatchSelectionProvider, useBatchSelection } from './BatchSelectionContext';

function TestComponent() {
  const {
    selectedItems,
    selectedCount,
    toggleItem,
    toggleAll,
    clearSelection,
    isSelected,
  } = useBatchSelection();

  return (
    <div>
      <div data-testid="count">{selectedCount}</div>
      <div data-testid="selected">{Array.from(selectedItems).join(',')}</div>
      <button onClick={() => toggleItem('item-1')}>Toggle 1</button>
      <button onClick={() => toggleItem('item-2')}>Toggle 2</button>
      <button onClick={() => toggleAll(['item-1', 'item-2', 'item-3'])}>Toggle All</button>
      <button onClick={() => clearSelection()}>Clear</button>
      <div data-testid="is-1-selected">{isSelected('item-1') ? 'yes' : 'no'}</div>
    </div>
  );
}

describe('BatchSelectionContext', () => {
  it('toggles individual items', async () => {
    const user = userEvent.setup();
    render(
      <BatchSelectionProvider>
        <TestComponent />
      </BatchSelectionProvider>
    );

    expect(screen.getByTestId('count')).toHaveTextContent('0');
    expect(screen.getByTestId('is-1-selected')).toHaveTextContent('no');

    await user.click(screen.getByText('Toggle 1'));
    expect(screen.getByTestId('count')).toHaveTextContent('1');
    expect(screen.getByTestId('selected')).toHaveTextContent('item-1');
    expect(screen.getByTestId('is-1-selected')).toHaveTextContent('yes');

    await user.click(screen.getByText('Toggle 2'));
    expect(screen.getByTestId('count')).toHaveTextContent('2');

    await user.click(screen.getByText('Toggle 1'));
    expect(screen.getByTestId('count')).toHaveTextContent('1');
    expect(screen.getByTestId('selected')).toHaveTextContent('item-2');
  });

  it('toggles all items', async () => {
    const user = userEvent.setup();
    render(
      <BatchSelectionProvider>
        <TestComponent />
      </BatchSelectionProvider>
    );

    await user.click(screen.getByText('Toggle All'));
    expect(screen.getByTestId('count')).toHaveTextContent('3');

    await user.click(screen.getByText('Toggle All'));
    expect(screen.getByTestId('count')).toHaveTextContent('0');
  });

  it('clears selection', async () => {
    const user = userEvent.setup();
    render(
      <BatchSelectionProvider>
        <TestComponent />
      </BatchSelectionProvider>
    );

    await user.click(screen.getByText('Toggle 1'));
    await user.click(screen.getByText('Toggle 2'));
    expect(screen.getByTestId('count')).toHaveTextContent('2');

    await user.click(screen.getByText('Clear'));
    expect(screen.getByTestId('count')).toHaveTextContent('0');
  });
});
