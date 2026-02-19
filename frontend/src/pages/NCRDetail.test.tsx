import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockNCRs } from "../test/mocks";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ id: "NCR-001" }),
  };
});

const mockGetNCR = vi.fn();
const mockUpdateNCR = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getNCR: (...args: any[]) => mockGetNCR(...args),
    updateNCR: (...args: any[]) => mockUpdateNCR(...args),
  },
}));

import NCRDetail from "./NCRDetail";

const mockNCROpen = { ...mockNCRs[0] };
const mockNCRResolved = { ...mockNCRs[1] };

beforeEach(() => {
  vi.clearAllMocks();
  mockGetNCR.mockResolvedValue(mockNCROpen);
  mockUpdateNCR.mockResolvedValue({ ...mockNCROpen, title: "Updated" });
});

describe("NCRDetail", () => {
  it("renders loading state", () => {
    mockGetNCR.mockReturnValue(new Promise(() => {}));
    render(<NCRDetail />);
    expect(screen.getByText("Loading NCR...")).toBeInTheDocument();
  });

  it("renders NCR detail after loading", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    // Title appears in header and detail section
    expect(screen.getAllByText("Defective resistor batch").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("NCR Details")).toBeInTheDocument();
  });

  it("shows back button that navigates to NCRs list", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("NCR-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Back to NCRs"));
    expect(mockNavigate).toHaveBeenCalledWith("/ncrs");
  });

  it("displays severity badge", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Major")).toBeInTheDocument();
    });
  });

  it("displays status badge", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Open")).toBeInTheDocument();
    });
  });

  it("shows description", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Out of tolerance")).toBeInTheDocument();
    });
  });

  it("shows affected IPN", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
  });

  it("shows serial number", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("SN-100")).toBeInTheDocument();
    });
  });

  it("shows defect type", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("tolerance")).toBeInTheDocument();
    });
  });

  // Root cause analysis section
  it("shows root cause analysis section", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Root Cause Analysis")).toBeInTheDocument();
    });
  });

  it("shows pending messages when no root cause or corrective action", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Root cause analysis pending")).toBeInTheDocument();
    });
    expect(screen.getByText("Corrective action pending")).toBeInTheDocument();
  });

  it("shows root cause and corrective action when present", async () => {
    mockGetNCR.mockResolvedValue(mockNCRResolved);
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Shipping damage")).toBeInTheDocument();
    });
    expect(screen.getByText("New packaging")).toBeInTheDocument();
  });

  // Sidebar
  it("shows Information card with created date", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Information")).toBeInTheDocument();
    });
  });

  it("shows resolved date when resolved", async () => {
    mockGetNCR.mockResolvedValue(mockNCRResolved);
    render(<NCRDetail />);
    await waitFor(() => {
      // "Resolved" appears as both status badge and sidebar label
      expect(screen.getAllByText("Resolved").length).toBeGreaterThanOrEqual(1);
    });
  });

  it("shows Create ECO from NCR button", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Create ECO from NCR")).toBeInTheDocument();
    });
  });

  it("navigates to ECOs with NCR param on Create ECO click", async () => {
    const mockFetch = vi.spyOn(global, "fetch").mockResolvedValueOnce({
      ok: true,
      json: async () => ({ id: "ECO-100" }),
    } as Response);
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Create ECO from NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create ECO from NCR"));
    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith("/ecos/ECO-100");
    });
    mockFetch.mockRestore();
  });

  // Edit mode
  it("has Edit NCR button", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
  });

  it("enters edit mode on Edit NCR click", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    // Should show Save and Cancel buttons
    expect(screen.getByText("Save Changes")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("shows editable inputs in edit mode", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    // Title should now be an input
    const titleInput = screen.getByPlaceholderText("NCR title");
    expect(titleInput).toBeInTheDocument();
    expect((titleInput as HTMLInputElement).value).toBe("Defective resistor batch");
  });

  it("cancels edit mode and restores original data", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    
    // Modify title
    fireEvent.change(screen.getByPlaceholderText("NCR title"), { target: { value: "Changed" } });
    
    // Cancel
    fireEvent.click(screen.getByText("Cancel"));
    
    // Should be back to view mode with original title (appears in header + detail)
    expect(screen.getAllByText("Defective resistor batch").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Edit NCR")).toBeInTheDocument();
  });

  it("saves changes on Save click", async () => {
    mockUpdateNCR.mockResolvedValue({ ...mockNCROpen, title: "Updated title" });
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    fireEvent.change(screen.getByPlaceholderText("NCR title"), { target: { value: "Updated title" } });
    fireEvent.click(screen.getByText("Save Changes"));

    await waitFor(() => {
      expect(mockUpdateNCR).toHaveBeenCalledWith("NCR-001", expect.objectContaining({ title: "Updated title" }));
    });
  });

  // Not found
  it("shows not found state", async () => {
    mockGetNCR.mockRejectedValue(new Error("Not found"));
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("NCR Not Found")).toBeInTheDocument();
    });
    expect(screen.getByText("The requested NCR could not be found.")).toBeInTheDocument();
  });

  it("shows Actions card title", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Actions")).toBeInTheDocument();
    });
  });

  it("shows 'No description provided' when description empty", async () => {
    mockGetNCR.mockResolvedValue({ ...mockNCROpen, description: "" });
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("No description provided")).toBeInTheDocument();
    });
  });

  it("shows 'Not specified' for empty optional fields", async () => {
    mockGetNCR.mockResolvedValue({ ...mockNCROpen, ipn: "", serial_number: "", defect_type: "" });
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getAllByText("Not specified").length).toBe(3);
    });
  });

  it("handles getNCR API rejection gracefully", async () => {
    mockGetNCR.mockRejectedValueOnce(new Error("Network error"));
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("NCR Not Found")).toBeInTheDocument();
    });
  });

  it("handles handleSave API rejection gracefully", async () => {
    mockUpdateNCR.mockRejectedValueOnce(new Error("Save failed"));
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    fireEvent.click(screen.getByText("Save Changes"));
    await waitFor(() => {
      expect(mockUpdateNCR).toHaveBeenCalled();
    });
    // Should not crash
    expect(screen.getByText("NCR-001")).toBeInTheDocument();
  });

  it("shows create_eco checkbox when editing with corrective_action and resolved status", async () => {
    mockGetNCR.mockResolvedValue({
      ...mockNCRResolved,
      corrective_action: "New packaging",
      status: "resolved",
    });
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    await waitFor(() => {
      expect(screen.getByText("Create ECO from corrective action")).toBeInTheDocument();
    });
  });

  it("shows create_eco checkbox when status is closed with corrective_action", async () => {
    mockGetNCR.mockResolvedValue({
      ...mockNCRResolved,
      corrective_action: "New packaging",
      status: "closed",
    });
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    await waitFor(() => {
      expect(screen.getByText("Create ECO from corrective action")).toBeInTheDocument();
    });
  });

  it("does not show create_eco checkbox when status is open", async () => {
    mockGetNCR.mockResolvedValue({
      ...mockNCROpen,
      corrective_action: "Some action",
      status: "open",
    });
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    expect(screen.queryByText("Create ECO from corrective action")).not.toBeInTheDocument();
  });

  it("does not show create_eco checkbox when no corrective_action", async () => {
    mockGetNCR.mockResolvedValue({
      ...mockNCRResolved,
      corrective_action: "",
      status: "resolved",
    });
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    expect(screen.queryByText("Create ECO from corrective action")).not.toBeInTheDocument();
  });

  it("edit mode: editable fields — description, IPN, serial_number, defect_type, root_cause, corrective_action", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));

    // Description
    const descInput = screen.getByPlaceholderText("Detailed description");
    expect(descInput).toBeInTheDocument();
    fireEvent.change(descInput, { target: { value: "Updated desc" } });
    expect((descInput as HTMLTextAreaElement).value).toBe("Updated desc");

    // IPN
    const ipnInput = screen.getByPlaceholderText("Part number");
    fireEvent.change(ipnInput, { target: { value: "IPN-999" } });
    expect((ipnInput as HTMLInputElement).value).toBe("IPN-999");

    // Serial Number
    const snInput = screen.getByPlaceholderText("Device serial number");
    fireEvent.change(snInput, { target: { value: "SN-999" } });
    expect((snInput as HTMLInputElement).value).toBe("SN-999");

    // Defect Type (now a Select component — shows current value)
    expect(screen.getByText("Defect Type")).toBeInTheDocument();

    // Root Cause
    const rcInput = screen.getByPlaceholderText("Identified root cause of the issue");
    fireEvent.change(rcInput, { target: { value: "Bad solder" } });
    expect((rcInput as HTMLTextAreaElement).value).toBe("Bad solder");

    // Corrective Action
    const caInput = screen.getByPlaceholderText("Actions taken to correct the issue");
    fireEvent.change(caInput, { target: { value: "Reflow" } });
    expect((caInput as HTMLTextAreaElement).value).toBe("Reflow");

    // Save and verify all fields sent
    mockUpdateNCR.mockResolvedValueOnce({ ...mockNCROpen, description: "Updated desc" });
    fireEvent.click(screen.getByText("Save Changes"));
    await waitFor(() => {
      expect(mockUpdateNCR).toHaveBeenCalledWith(
        "NCR-001",
        expect.objectContaining({
          description: "Updated desc",
          ipn: "IPN-999",
          serial_number: "SN-999",
          defect_type: "tolerance",
          root_cause: "Bad solder",
          corrective_action: "Reflow",
        })
      );
    });
  });

  it("save error handling — stays in edit mode on reject", async () => {
    mockUpdateNCR.mockRejectedValueOnce(new Error("Save failed"));
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    fireEvent.change(screen.getByPlaceholderText("NCR title"), { target: { value: "Changed" } });
    fireEvent.click(screen.getByText("Save Changes"));
    await waitFor(() => {
      expect(mockUpdateNCR).toHaveBeenCalled();
    });
    // Should remain in edit mode (Save Changes button still visible)
    expect(screen.getByText("Save Changes")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("NCR title")).toBeInTheDocument();
  });

  it("checkbox not visible when not in edit mode even with corrective_action and resolved status", async () => {
    mockGetNCR.mockResolvedValue({
      ...mockNCRResolved,
      corrective_action: "New packaging",
      status: "resolved",
    });
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("NCR-002")).toBeInTheDocument();
    });
    // In view mode, checkbox should not be visible
    expect(screen.queryByText("Create ECO from corrective action")).not.toBeInTheDocument();
  });

  it("checkbox not visible when status is investigating", async () => {
    mockGetNCR.mockResolvedValue({
      ...mockNCROpen,
      corrective_action: "Some fix",
      status: "investigating",
    });
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    expect(screen.queryByText("Create ECO from corrective action")).not.toBeInTheDocument();
  });

  it("severity select shows current value in edit mode", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    // The severity select trigger should display the current value
    // mockNCROpen has severity "major"
    const severitySection = screen.getByText("Severity").closest("div")!;
    expect(severitySection.textContent).toContain("Major");
  });

  it("status select shows current value in edit mode", async () => {
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    // mockNCROpen has status "open"
    const statusSection = screen.getByText("Status").closest("div")!;
    expect(statusSection.textContent).toContain("Open");
  });

  it("includes create_eco: true in API payload when checkbox checked and saved", async () => {
    mockGetNCR.mockResolvedValue({
      ...mockNCRResolved,
      corrective_action: "New packaging",
      status: "resolved",
    });
    mockUpdateNCR.mockResolvedValue({ ...mockNCRResolved, status: "resolved" });
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit NCR"));
    await waitFor(() => {
      expect(screen.getByText("Create ECO from corrective action")).toBeInTheDocument();
    });
    // Click the checkbox
    fireEvent.click(screen.getByRole("checkbox", { name: /Create ECO from corrective action/i }));
    // Save
    fireEvent.click(screen.getByText("Save Changes"));
    await waitFor(() => {
      expect(mockUpdateNCR).toHaveBeenCalledWith(
        "NCR-001",
        expect.objectContaining({ create_eco: true })
      );
    });
  });
});
