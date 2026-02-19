import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { toast } from "sonner";
import { useUndo } from "./useUndo";

vi.mock("sonner", () => ({
  toast: Object.assign(vi.fn(), {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
  }),
}));

vi.mock("../lib/api", () => ({
  api: {
    performUndo: vi.fn().mockResolvedValue({ status: "restored" }),
    undoChange: vi.fn().mockResolvedValue({ status: "undone", table_name: "vendors", record_id: "V-001", operation: "delete", redo_id: 2 }),
    getRecentChanges: vi.fn().mockResolvedValue([]),
  },
}));

beforeEach(() => {
  vi.clearAllMocks();
});

describe("useUndo", () => {
  it("showUndoToast calls toast with action button", () => {
    const { result } = renderHook(() => useUndo());
    act(() => {
      result.current.showUndoToast(42, {
        entityType: "vendor",
        entityId: "V-001",
        action: "Deleted",
      });
    });
    expect(toast).toHaveBeenCalledWith(
      "Deleted vendor V-001",
      expect.objectContaining({
        duration: 5000,
        action: expect.objectContaining({ label: "Undo" }),
      })
    );
  });

  it("showChangeUndoToast calls toast with Redo button", () => {
    const { result } = renderHook(() => useUndo());
    act(() => {
      result.current.showChangeUndoToast(1, "vendors", "V-001", "delete");
    });
    expect(toast).toHaveBeenCalledWith(
      "Undone: delete vendors V-001",
      expect.objectContaining({
        duration: 5000,
        action: expect.objectContaining({ label: "Redo" }),
      })
    );
  });

  it("withUndo calls API and shows toast when undo_id present", async () => {
    const { result } = renderHook(() => useUndo());
    const mockApiCall = vi.fn().mockResolvedValue({ deleted: "V-001", undo_id: 7 });

    let response: Record<string, unknown> | undefined;
    await act(async () => {
      response = await result.current.withUndo(mockApiCall, {
        entityType: "vendor",
        entityId: "V-001",
      });
    });

    expect(mockApiCall).toHaveBeenCalled();
    expect(response).toEqual({ deleted: "V-001", undo_id: 7 });
    expect(toast).toHaveBeenCalledWith(
      expect.stringContaining("vendor V-001"),
      expect.objectContaining({ action: expect.objectContaining({ label: "Undo" }) })
    );
  });

  it("withUndo does not show toast when no undo_id", async () => {
    const { result } = renderHook(() => useUndo());
    const mockApiCall = vi.fn().mockResolvedValue({ status: "ok" });

    await act(async () => {
      await result.current.withUndo(mockApiCall, {
        entityType: "vendor",
        entityId: "V-001",
      });
    });

    expect(toast).not.toHaveBeenCalled();
  });
});
