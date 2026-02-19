import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BatchCheckbox, MasterBatchCheckbox } from './BatchCheckbox';
import { BatchSelectionProvider } from '../contexts/BatchSelectionContext';

describe('BatchCheckbox', () => {
  it('renders and can be toggled', async () => {
    const user = userEvent.setup();
    render(
      <BatchSelectionProvider>
        <BatchCheckbox id="item-1" />
      </BatchSelectionProvider>
    );

    const checkbox = screen.getByRole('checkbox');
    expect(checkbox).not.toBeChecked();

    await user.click(checkbox);
    expect(checkbox).toBeChecked();

    await user.click(checkbox);
    expect(checkbox).not.toBeChecked();
  });

  it('respects disabled state', () => {
    render(
      <BatchSelectionProvider>
        <BatchCheckbox id="item-1" disabled />
      </BatchSelectionProvider>
    );

    const checkbox = screen.getByRole('checkbox');
    expect(checkbox).toBeDisabled();
  });
});

describe('MasterBatchCheckbox', () => {
  it('selects all items when clicked', async () => {
    const user = userEvent.setup();
    const allIds = ['item-1', 'item-2', 'item-3'];

    render(
      <BatchSelectionProvider>
        <MasterBatchCheckbox allIds={allIds} />
        {allIds.map(id => <BatchCheckbox key={id} id={id} />)}
      </BatchSelectionProvider>
    );

    const masterCheckbox = screen.getAllByRole('checkbox')[0];
    const itemCheckboxes = screen.getAllByRole('checkbox').slice(1);

    // Initially unchecked
    expect(masterCheckbox).not.toBeChecked();
    itemCheckboxes.forEach(cb => expect(cb).not.toBeChecked());

    // Click master to select all
    await user.click(masterCheckbox);
    expect(masterCheckbox).toBeChecked();
    itemCheckboxes.forEach(cb => expect(cb).toBeChecked());

    // Click master again to deselect all
    await user.click(masterCheckbox);
    expect(masterCheckbox).not.toBeChecked();
    itemCheckboxes.forEach(cb => expect(cb).not.toBeChecked());
  });

  it('shows indeterminate state when some items selected', async () => {
    const user = userEvent.setup();
    const allIds = ['item-1', 'item-2', 'item-3'];

    render(
      <BatchSelectionProvider>
        <MasterBatchCheckbox allIds={allIds} />
        {allIds.map(id => <BatchCheckbox key={id} id={id} />)}
      </BatchSelectionProvider>
    );

    const masterCheckbox = screen.getAllByRole('checkbox')[0] as HTMLInputElement;
    const firstItemCheckbox = screen.getAllByRole('checkbox')[1];

    // Select one item
    await user.click(firstItemCheckbox);

    // Master should be indeterminate
    expect(masterCheckbox.indeterminate).toBe(true);
  });
});
