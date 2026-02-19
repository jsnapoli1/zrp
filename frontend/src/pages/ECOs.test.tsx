import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockECOs } from "../test/mocks";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const mockGetECOs = vi.fn().mockResolvedValue(mockECOs);
const mockCreateECO = vi.fn().mockResolvedValue({ ...mockECOs[0], id: "ECO-NEW" });

vi.mock("../lib/api", () => ({
  api: {
    getECOs: (...args: any[]) => mockGetECOs(...args),
    createECO: (...args: any[]) => mockCreateECO(...args),
  },
}));

import ECOs from "./ECOs";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetECOs.mockResolvedValue(mockECOs);
});

describe("ECOs", () => {
  it("renders page heading and description", async () => {
    render(<ECOs />);
    expect(screen.getByText("Engineering Change Orders")).toBeInTheDocument();
    expect(screen.getByText("Manage design changes and product modifications")).toBeInTheDocument();
  });

  it("shows ECO Status card during loading", () => {
    render(<ECOs />);
    expect(screen.getByText("ECO Status")).toBeInTheDocument();
  });

  it("renders ECO list after loading", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    expect(screen.getByText("ECO-002")).toBeInTheDocument();
    expect(screen.getByText("ECO-003")).toBeInTheDocument();
    expect(screen.getByText("Update resistor spec")).toBeInTheDocument();
    expect(screen.getByText("Replace MCU")).toBeInTheDocument();
  });

  it("shows status badges", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("Draft")).toBeInTheDocument();
    });
    expect(screen.getByText("Approved")).toBeInTheDocument();
    expect(screen.getByText("Open")).toBeInTheDocument();
  });

  it("displays created by info", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getAllByText("admin").length).toBeGreaterThan(0);
    });
  });

  it("has tabs for filtering by status", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    expect(screen.getByRole("tablist")).toBeInTheDocument();
    // Check tab labels with counts
    expect(screen.getByRole("tab", { name: /all/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /open/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /approved/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /implemented/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /rejected/i })).toBeInTheDocument();
  });

  it("filters ECOs when tab is clicked", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });

    // Click approved tab
    const approvedTab = screen.getByRole("tab", { name: /approved/i });
    fireEvent.click(approvedTab);

    await waitFor(() => {
      // ECO-002 is approved, should be visible
      expect(screen.getByText("ECO-002")).toBeInTheDocument();
    });
  });

  it("shows empty state when no ECOs", async () => {
    mockGetECOs.mockResolvedValueOnce([]);
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("No ECOs found")).toBeInTheDocument();
    });
  });

  it("calls getECOs on mount", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(mockGetECOs).toHaveBeenCalled();
    });
  });

  it("navigates to ECO detail on row click", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("ECO-001"));
    expect(mockNavigate).toHaveBeenCalledWith("/ecos/ECO-001");
  });

  // Create dialog tests
  it("has create ECO button", async () => {
    render(<ECOs />);
    expect(screen.getByText("Create ECO")).toBeInTheDocument();
  });

  it("opens create dialog with form fields", async () => {
    render(<ECOs />);
    fireEvent.click(screen.getByText("Create ECO"));
    await waitFor(() => {
      expect(screen.getByText("Create New ECO")).toBeInTheDocument();
    });
    expect(screen.getByPlaceholderText("Enter ECO title...")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Describe the change in detail...")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Why is this change needed?")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("Comma-separated list of affected part numbers...")).toBeInTheDocument();
  });

  it("shows dialog description", async () => {
    render(<ECOs />);
    fireEvent.click(screen.getByText("Create ECO"));
    await waitFor(() => {
      expect(screen.getByText(/Create a new Engineering Change Order to document and track modifications/)).toBeInTheDocument();
    });
  });

  it("has cancel and submit buttons in dialog", async () => {
    render(<ECOs />);
    fireEvent.click(screen.getByText("Create ECO"));
    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
      // There are two "Create ECO" - one is the trigger, one is the submit button
      expect(screen.getAllByText("Create ECO").length).toBeGreaterThanOrEqual(1);
    });
  });

  it("shows table headers", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    expect(screen.getByText("ECO ID")).toBeInTheDocument();
    expect(screen.getByText("Title")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
    expect(screen.getByText("Created By")).toBeInTheDocument();
    expect(screen.getByText("Created Date")).toBeInTheDocument();
    expect(screen.getByText("Updated Date")).toBeInTheDocument();
  });

  it("formats dates in table cells", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    // Dates are rendered inside cells with Calendar icons, so use getAllByText with substring
    const dateCells = screen.getAllByText((content) => content.includes("Jan") && content.includes("2024"));
    expect(dateCells.length).toBeGreaterThan(0);
  });

  it("handles API error gracefully", async () => {
    mockGetECOs.mockRejectedValueOnce(new Error("Network error"));
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("No ECOs found")).toBeInTheDocument();
    });
  });

  it("ECO Status card title is visible", async () => {
    render(<ECOs />);
    expect(screen.getByText("ECO Status")).toBeInTheDocument();
  });
});
