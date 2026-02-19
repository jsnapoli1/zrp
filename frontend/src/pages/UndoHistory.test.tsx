import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

const mockGetUndoList = vi.fn();
const mockPerformUndo = vi.fn();
const mockGetRecentChanges = vi.fn();
const mockUndoChange = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getUndoList: (...args: unknown[]) => mockGetUndoList(...args),
    performUndo: (...args: unknown[]) => mockPerformUndo(...args),
    getRecentChanges: (...args: unknown[]) => mockGetRecentChanges(...args),
    undoChange: (...args: unknown[]) => mockUndoChange(...args),
  },
}));

vi.mock("sonner", () => ({
  toast: Object.assign(vi.fn(), {
    success: vi.fn(),
    error: vi.fn(),
  }),
}));

import UndoHistory from "./UndoHistory";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("UndoHistory", () => {
  it("renders empty state when no entries", async () => {
    mockGetUndoList.mockResolvedValue([]);
    mockGetRecentChanges.mockResolvedValue([]);
    render(<UndoHistory />);
    await waitFor(() => {
      expect(screen.getByText("No changes recorded yet")).toBeInTheDocument();
    });
  });

  it("renders change history entries", async () => {
    mockGetUndoList.mockResolvedValue([]);
    mockGetRecentChanges.mockResolvedValue([
      {
        id: 1,
        table_name: "vendors",
        record_id: "V-001",
        operation: "create",
        old_data: "",
        new_data: "{}",
        user_id: "admin",
        created_at: "2026-02-18 12:00:00",
        undone: 0,
      },
    ]);
    render(<UndoHistory />);
    await waitFor(() => {
      expect(screen.getByText("V-001")).toBeInTheDocument();
    });
    expect(screen.getByText(/create/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /undo/i })).toBeInTheDocument();
  });

  it("renders undone entries with badge", async () => {
    mockGetUndoList.mockResolvedValue([]);
    mockGetRecentChanges.mockResolvedValue([
      {
        id: 2,
        table_name: "ecos",
        record_id: "ECO-001",
        operation: "update",
        old_data: "{}",
        new_data: "{}",
        user_id: "admin",
        created_at: "2026-02-18 12:00:00",
        undone: 1,
      },
    ]);
    render(<UndoHistory />);
    await waitFor(() => {
      expect(screen.getByText("Undone")).toBeInTheDocument();
    });
  });

  it("calls undoChange when Undo button clicked", async () => {
    mockGetUndoList.mockResolvedValue([]);
    mockGetRecentChanges.mockResolvedValue([
      {
        id: 5,
        table_name: "vendors",
        record_id: "V-001",
        operation: "delete",
        old_data: "{}",
        new_data: "",
        user_id: "admin",
        created_at: "2026-02-18 12:00:00",
        undone: 0,
      },
    ]);
    mockUndoChange.mockResolvedValue({ status: "undone", table_name: "vendors", record_id: "V-001", operation: "delete", redo_id: 6 });

    render(<UndoHistory />);
    await waitFor(() => {
      expect(screen.getByText("V-001")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: /undo/i }));
    await waitFor(() => {
      expect(mockUndoChange).toHaveBeenCalledWith(5);
    });
  });

  it("shows page heading with Ctrl+Z hint", async () => {
    mockGetUndoList.mockResolvedValue([]);
    mockGetRecentChanges.mockResolvedValue([]);
    render(<UndoHistory />);
    await waitFor(() => {
      expect(screen.getByText("Undo History")).toBeInTheDocument();
      expect(screen.getByText(/Ctrl\+Z/)).toBeInTheDocument();
    });
  });

  it("shows tabs for Change History and Snapshots", async () => {
    mockGetUndoList.mockResolvedValue([]);
    mockGetRecentChanges.mockResolvedValue([]);
    render(<UndoHistory />);
    await waitFor(() => {
      expect(screen.getByText("Change History")).toBeInTheDocument();
      expect(screen.getByText("Snapshots (24h)")).toBeInTheDocument();
    });
  });
});
