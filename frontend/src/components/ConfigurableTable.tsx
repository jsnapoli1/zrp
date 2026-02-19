import { useState, useEffect, useCallback, useMemo, useRef, type ReactNode } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuCheckboxItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
  DropdownMenuLabel,
} from "./ui/dropdown-menu";
import { Button } from "./ui/button";
import { Settings2, ArrowUp, ArrowDown, RotateCcw, ArrowUpNarrowWide, ArrowDownWideNarrow } from "lucide-react";

export interface ColumnDef<T> {
  id: string;
  label: string;
  accessor: (row: T) => ReactNode;
  defaultVisible?: boolean;
  className?: string;
  headerClassName?: string;
  minWidth?: number;
  defaultWidth?: number;
  /** Provide a sort value extractor to enable sorting on this column */
  sortValue?: (row: T) => string | number;
}

interface ColumnState {
  id: string;
  visible: boolean;
  width?: number;
}

interface ConfigurableTableProps<T> {
  tableName: string;
  columns: ColumnDef<T>[];
  data: T[];
  rowKey: (row: T) => string;
  onRowClick?: (row: T) => void;
  rowClassName?: (row: T) => string;
  emptyMessage?: string;
  extraRowContent?: (row: T) => ReactNode;
  /** Extra column rendered before configurable columns (e.g. checkbox) */
  leadingColumn?: {
    header: ReactNode;
    cell: (row: T) => ReactNode;
    className?: string;
  };
  /** Accessible label for the table (required for screen readers) */
  ariaLabel?: string;
  /** Optional caption for the table */
  caption?: string;
}

function loadColumnState(tableName: string): ColumnState[] | null {
  try {
    const stored = localStorage.getItem(`zrp-columns-${tableName}`);
    return stored ? JSON.parse(stored) : null;
  } catch {
    return null;
  }
}

function saveColumnState(tableName: string, state: ColumnState[]) {
  try {
    localStorage.setItem(`zrp-columns-${tableName}`, JSON.stringify(state));
  } catch {
    // ignore
  }
}

export function ConfigurableTable<T>({
  tableName,
  columns,
  data,
  rowKey,
  onRowClick,
  rowClassName,
  emptyMessage = "No data found",
  leadingColumn,
  ariaLabel,
  caption,
}: ConfigurableTableProps<T>) {
  const [columnOrder, setColumnOrder] = useState<ColumnState[]>(() => {
    const saved = loadColumnState(tableName);
    if (saved) {
      // Merge saved state with current columns (handle new/removed columns)
      const savedMap = new Map(saved.map((s) => [s.id, s]));
      const merged: ColumnState[] = [];
      // Keep saved order for existing columns
      for (const s of saved) {
        if (columns.find((c) => c.id === s.id)) {
          merged.push(s);
        }
      }
      // Add new columns not in saved state
      for (const c of columns) {
        if (!savedMap.has(c.id)) {
          merged.push({
            id: c.id,
            visible: c.defaultVisible !== false,
            width: c.defaultWidth,
          });
        }
      }
      return merged;
    }
    return columns.map((c) => ({
      id: c.id,
      visible: c.defaultVisible !== false,
      width: c.defaultWidth,
    }));
  });

  // Persist on change
  useEffect(() => {
    saveColumnState(tableName, columnOrder);
  }, [tableName, columnOrder]);

  const colMap = useMemo(() => new Map(columns.map((c) => [c.id, c])), [columns]);

  const visibleColumns = columnOrder
    .filter((cs) => cs.visible && colMap.has(cs.id))
    .map((cs) => ({ ...colMap.get(cs.id)!, width: cs.width }));

  const toggleVisibility = (id: string) => {
    setColumnOrder((prev) =>
      prev.map((cs) => (cs.id === id ? { ...cs, visible: !cs.visible } : cs))
    );
  };

  const moveColumn = (id: string, direction: "up" | "down") => {
    setColumnOrder((prev) => {
      const idx = prev.findIndex((cs) => cs.id === id);
      if (idx < 0) return prev;
      const swapIdx = direction === "up" ? idx - 1 : idx + 1;
      if (swapIdx < 0 || swapIdx >= prev.length) return prev;
      const next = [...prev];
      [next[idx], next[swapIdx]] = [next[swapIdx], next[idx]];
      return next;
    });
  };

  const resetColumns = () => {
    const defaultState = columns.map((c) => ({
      id: c.id,
      visible: c.defaultVisible !== false,
      width: c.defaultWidth,
    }));
    setColumnOrder(defaultState);
  };

  // Sorting state: cycles none → asc → desc → none
  const [sortState, setSortState] = useState<{ colId: string; direction: "asc" | "desc" } | null>(null);

  const handleSort = useCallback((colId: string) => {
    const col = colMap.get(colId);
    if (!col?.sortValue) return;
    setSortState((prev) => {
      if (!prev || prev.colId !== colId) return { colId, direction: "asc" };
      if (prev.direction === "asc") return { colId, direction: "desc" };
      return null;
    });
  }, [colMap]);

  const sortedData = useMemo(() => {
    if (!sortState) return data;
    const col = colMap.get(sortState.colId);
    if (!col?.sortValue) return data;
    const getValue = col.sortValue;
    const mult = sortState.direction === "asc" ? 1 : -1;
    return [...data].sort((a, b) => {
      const va = getValue(a);
      const vb = getValue(b);
      // Null/undefined/empty values sort to end regardless of direction
      const aEmpty = va == null || va === "";
      const bEmpty = vb == null || vb === "";
      if (aEmpty && bEmpty) return 0;
      if (aEmpty) return 1;
      if (bEmpty) return -1;
      if (typeof va === "number" && typeof vb === "number") return (va - vb) * mult;
      return String(va).localeCompare(String(vb)) * mult;
    });
  }, [data, sortState, colMap]);

  // Column resize
  const resizingRef = useRef<{
    colId: string;
    startX: number;
    startWidth: number;
  } | null>(null);

  const handleResizeStart = useCallback(
    (e: React.MouseEvent, colId: string, currentWidth: number) => {
      e.preventDefault();
      e.stopPropagation();
      resizingRef.current = {
        colId,
        startX: e.clientX,
        startWidth: currentWidth,
      };

      const handleMouseMove = (ev: MouseEvent) => {
        if (!resizingRef.current) return;
        const diff = ev.clientX - resizingRef.current.startX;
        const newWidth = Math.max(50, resizingRef.current.startWidth + diff);
        setColumnOrder((prev) =>
          prev.map((cs) =>
            cs.id === resizingRef.current!.colId
              ? { ...cs, width: newWidth }
              : cs
          )
        );
      };

      const handleMouseUp = () => {
        resizingRef.current = null;
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    []
  );

  return (
    <div>
      <Table aria-label={ariaLabel || `${tableName} table`}>
        {caption && <caption className="sr-only">{caption}</caption>}
        <TableHeader>
          <TableRow>
            {leadingColumn && (
              <TableHead className={leadingColumn.className} scope="col">
                {leadingColumn.header}
              </TableHead>
            )}
            {visibleColumns.map((col) => {
              const isSorted = sortState?.colId === col.id;
              const sortDirection = isSorted ? sortState.direction : undefined;
              const ariaSortValue = sortDirection ? (sortDirection === "asc" ? "ascending" : "descending") : col.sortValue ? "none" : undefined;
              
              return (
                <TableHead
                  key={col.id}
                  className={col.headerClassName}
                  style={col.width ? { width: col.width, minWidth: col.minWidth || 50 } : { minWidth: col.minWidth || 50 }}
                  scope="col"
                  aria-sort={ariaSortValue}
                >
                  <div
                    className={`flex items-center gap-1 relative pr-2${col.sortValue ? " cursor-pointer select-none" : ""}`}
                    onClick={() => handleSort(col.id)}
                    role={col.sortValue ? "button" : undefined}
                    aria-label={col.sortValue ? `Sort by ${col.label}${sortDirection ? ` (currently sorted ${sortDirection === "asc" ? "ascending" : "descending"})` : ""}` : undefined}
                    tabIndex={col.sortValue ? 0 : undefined}
                    onKeyDown={col.sortValue ? (e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); handleSort(col.id); } } : undefined}
                  >
                    <span>{col.label}</span>
                    {isSorted && sortDirection === "asc" && (
                    <ArrowUpNarrowWide className="h-3.5 w-3.5 text-muted-foreground" aria-hidden="true" />
                  )}
                  {sortState?.colId === col.id && sortState.direction === "desc" && (
                    <ArrowDownWideNarrow className="h-3.5 w-3.5 text-muted-foreground" aria-hidden="true" />
                  )}
                  {/* Resize handle */}
                  <div
                    className="absolute right-0 top-0 bottom-0 w-1 cursor-col-resize hover:bg-primary/30"
                    onMouseDown={(e) =>
                      handleResizeStart(e, col.id, col.width || 150)
                    }
                  />
                </div>
              </TableHead>
            )})}
            <TableHead className="w-10 print:hidden">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="sm" className="h-6 w-6 p-0">
                    <Settings2 className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-64">
                  <DropdownMenuLabel>Configure Columns</DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  {columnOrder.map((cs, idx) => {
                    const col = colMap.get(cs.id);
                    if (!col) return null;
                    return (
                      <div key={cs.id} className="flex items-center">
                        <DropdownMenuCheckboxItem
                          checked={cs.visible}
                          onCheckedChange={() => toggleVisibility(cs.id)}
                          onSelect={(e) => e.preventDefault()}
                          className="flex-1"
                        >
                          {col.label}
                        </DropdownMenuCheckboxItem>
                        <div className="flex gap-0.5 pr-2">
                          <button
                            className="p-0.5 hover:bg-muted rounded disabled:opacity-30"
                            disabled={idx === 0}
                            onClick={(e) => {
                              e.stopPropagation();
                              moveColumn(cs.id, "up");
                            }}
                          >
                            <ArrowUp className="h-3 w-3" />
                          </button>
                          <button
                            className="p-0.5 hover:bg-muted rounded disabled:opacity-30"
                            disabled={idx === columnOrder.length - 1}
                            onClick={(e) => {
                              e.stopPropagation();
                              moveColumn(cs.id, "down");
                            }}
                          >
                            <ArrowDown className="h-3 w-3" />
                          </button>
                        </div>
                      </div>
                    );
                  })}
                  <DropdownMenuSeparator />
                  <div className="p-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      className="w-full justify-start"
                      onClick={resetColumns}
                    >
                      <RotateCcw className="h-3 w-3 mr-2" />
                      Reset to Default
                    </Button>
                  </div>
                </DropdownMenuContent>
              </DropdownMenu>
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sortedData.length === 0 ? (
            <TableRow>
              <TableCell
                colSpan={visibleColumns.length + (leadingColumn ? 2 : 1)}
                className="text-center py-8 text-muted-foreground"
              >
                {emptyMessage}
              </TableCell>
            </TableRow>
          ) : (
            sortedData.map((row) => (
              <TableRow
                key={rowKey(row)}
                className={`${onRowClick ? "cursor-pointer hover:bg-muted/50" : ""} ${rowClassName?.(row) || ""}`}
                onClick={() => onRowClick?.(row)}
              >
                {leadingColumn && (
                  <TableCell className={leadingColumn.className}>
                    {leadingColumn.cell(row)}
                  </TableCell>
                )}
                {visibleColumns.map((col) => (
                  <TableCell
                    key={col.id}
                    className={col.className}
                    style={col.width ? { width: col.width } : undefined}
                  >
                    {col.accessor(row)}
                  </TableCell>
                ))}
                <TableCell className="print:hidden" />
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>
    </div>
  );
}

export default ConfigurableTable;
