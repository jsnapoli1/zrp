import { Checkbox } from "./ui/checkbox";
import { useBatchSelection } from "../contexts/BatchSelectionContext";

interface BatchCheckboxProps {
  id: string;
  disabled?: boolean;
}

export function BatchCheckbox({ id, disabled }: BatchCheckboxProps) {
  const { isSelected, toggleItem } = useBatchSelection();

  return (
    <Checkbox
      checked={isSelected(id)}
      onCheckedChange={() => toggleItem(id)}
      disabled={disabled}
      aria-label={`Select item ${id}`}
    />
  );
}

interface MasterBatchCheckboxProps {
  allIds: string[];
  disabled?: boolean;
}

export function MasterBatchCheckbox({ allIds, disabled }: MasterBatchCheckboxProps) {
  const { selectedCount, toggleAll } = useBatchSelection();

  const allSelected = selectedCount === allIds.length && allIds.length > 0;
  const someSelected = selectedCount > 0 && selectedCount < allIds.length;

  return (
    <Checkbox
      checked={allSelected}
      ref={(el) => {
        if (el) {
          el.indeterminate = someSelected;
        }
      }}
      onCheckedChange={() => toggleAll(allIds)}
      disabled={disabled}
      aria-label="Select all items"
    />
  );
}
