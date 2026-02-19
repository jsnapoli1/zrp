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
    render(<NCRDetail />);
    await waitFor(() => {
      expect(screen.getByText("Create ECO from NCR")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create ECO from NCR"));
    expect(mockNavigate).toHaveBeenCalledWith("/ecos?from_ncr=NCR-001");
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
});
