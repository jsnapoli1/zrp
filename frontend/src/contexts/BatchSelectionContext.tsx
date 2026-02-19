import { createContext, useContext, useState, ReactNode } from "react";

interface BatchSelectionContextType {
  selectedItems: Set<string>;
  setSelectedItems: (items: Set<string>) => void;
  toggleItem: (id: string) => void;
  toggleAll: (ids: string[]) => void;
  clearSelection: () => void;
  isSelected: (id: string) => boolean;
  selectedCount: number;
}

const BatchSelectionContext = createContext<BatchSelectionContextType | undefined>(undefined);

export function BatchSelectionProvider({ children }: { children: ReactNode }) {
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());

  const toggleItem = (id: string) => {
    const next = new Set(selectedItems);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    setSelectedItems(next);
  };

  const toggleAll = (ids: string[]) => {
    if (selectedItems.size === ids.length) {
      // All selected, deselect all
      setSelectedItems(new Set());
    } else {
      // Select all
      setSelectedItems(new Set(ids));
    }
  };

  const clearSelection = () => {
    setSelectedItems(new Set());
  };

  const isSelected = (id: string) => {
    return selectedItems.has(id);
  };

  return (
    <BatchSelectionContext.Provider
      value={{
        selectedItems,
        setSelectedItems,
        toggleItem,
        toggleAll,
        clearSelection,
        isSelected,
        selectedCount: selectedItems.size,
      }}
    >
      {children}
    </BatchSelectionContext.Provider>
  );
}

export function useBatchSelection() {
  const context = useContext(BatchSelectionContext);
  if (!context) {
    throw new Error("useBatchSelection must be used within BatchSelectionProvider");
  }
  return context;
}
