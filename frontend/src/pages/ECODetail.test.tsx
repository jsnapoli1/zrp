import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockECOs } from "../test/mocks";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ id: "ECO-001" }),
  };
});

const mockGetECO = vi.fn();
const mockApproveECO = vi.fn();
const mockImplementECO = vi.fn();
const mockRejectECO = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getECO: (...args: any[]) => mockGetECO(...args),
    approveECO: (...args: any[]) => mockApproveECO(...args),
    implementECO: (...args: any[]) => mockImplementECO(...args),
    rejectECO: (...args: any[]) => mockRejectECO(...args),
  },
}));

import ECODetail from "./ECODetail";

const mockECODraft = {
  ...mockECOs[0],
  status: "draft",
  priority: "normal",
  affected_parts: [
    { ipn: "IPN-001", description: "10k Resistor" },
    { ipn: "IPN-999", error: "Part not found in system" },
  ],
};

const mockECOOpen = {
  ...mockECOs[2],
  status: "open",
  priority: "high",
  affected_parts: [],
};

const mockECOApproved = {
  ...mockECOs[1],
  status: "approved",
  priority: "normal",
  affected_parts: [{ ipn: "IPN-003", description: "MCU STM32" }],
};

const mockECOImplemented = {
  ...mockECOs[1],
  status: "implemented",
  priority: "normal",
  affected_parts: [],
};

const mockECORejected = {
  ...mockECOs[0],
  status: "rejected",
  priority: "low",
  affected_parts: [],
};

beforeEach(() => {
  vi.clearAllMocks();
  mockGetECO.mockResolvedValue(mockECODraft);
  mockApproveECO.mockResolvedValue({ ...mockECOOpen, status: "approved" });
  mockImplementECO.mockResolvedValue({ ...mockECOApproved, status: "implemented" });
  mockRejectECO.mockResolvedValue({ ...mockECODraft, status: "rejected" });
});

describe("ECODetail", () => {
  it("renders loading state initially", () => {
    mockGetECO.mockReturnValue(new Promise(() => {})); // never resolves
    render(<ECODetail />);
    // During loading, no ECO content is shown yet
    expect(screen.queryByText("ECO Details")).not.toBeInTheDocument();
  });

  it("renders ECO detail after loading", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    // Title appears in header and detail section
    expect(screen.getAllByText("Update resistor spec").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("ECO Details")).toBeInTheDocument();
  });

  it("shows back button that navigates to ECOs list", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Back to ECOs"));
    expect(mockNavigate).toHaveBeenCalledWith("/ecos");
  });

  it("displays status badge", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("Draft")).toBeInTheDocument();
    });
  });

  it("displays priority badge", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText(/normal Priority/i)).toBeInTheDocument();
    });
  });

  it("shows description and reason", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("Change tolerance")).toBeInTheDocument();
    });
    expect(screen.getByText("Quality")).toBeInTheDocument();
    expect(screen.getByText("Reason for Change")).toBeInTheDocument();
  });

  it("shows status description", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText(/ECO is being prepared and not yet submitted/)).toBeInTheDocument();
    });
  });

  // Affected parts
  it("displays affected parts section", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText(/Affected Parts/)).toBeInTheDocument();
    });
    expect(screen.getByText("IPN-001")).toBeInTheDocument();
    expect(screen.getByText("10k Resistor")).toBeInTheDocument();
  });

  it("shows part error for not-found parts", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-999")).toBeInTheDocument();
    });
    expect(screen.getByText("Part not found in system")).toBeInTheDocument();
    expect(screen.getByText("Not Found")).toBeInTheDocument();
  });

  it("shows View Part badge for valid parts", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("View Part")).toBeInTheDocument();
    });
  });

  it("navigates to part detail on valid part click", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("IPN-001"));
    expect(mockNavigate).toHaveBeenCalledWith("/parts/IPN-001");
  });

  // Sidebar metadata
  it("displays metadata sidebar", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("Information")).toBeInTheDocument();
    });
    expect(screen.getByText("Created by")).toBeInTheDocument();
    expect(screen.getByText("admin")).toBeInTheDocument();
    expect(screen.getByText("Created")).toBeInTheDocument();
    expect(screen.getByText("Last updated")).toBeInTheDocument();
  });

  it("shows approval info when approved", async () => {
    mockGetECO.mockResolvedValue(mockECOApproved);
    render(<ECODetail />);
    await waitFor(() => {
      // "Approved" appears as badge and in description
      expect(screen.getAllByText("Approved").length).toBeGreaterThanOrEqual(1);
    });
    expect(screen.getByText(/manager/)).toBeInTheDocument();
  });

  // Status actions
  it("shows reject button for draft ECO", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("Reject ECO")).toBeInTheDocument();
    });
  });

  it("does NOT show approve button for draft ECO", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    expect(screen.queryByText("Approve ECO")).not.toBeInTheDocument();
  });

  it("shows approve and reject buttons for open ECO", async () => {
    mockGetECO.mockResolvedValue(mockECOOpen);
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("Approve ECO")).toBeInTheDocument();
    });
    expect(screen.getByText("Reject ECO")).toBeInTheDocument();
  });

  it("shows implement button for approved ECO", async () => {
    mockGetECO.mockResolvedValue(mockECOApproved);
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("Implement ECO")).toBeInTheDocument();
    });
  });

  it("shows no actions message for implemented ECO", async () => {
    mockGetECO.mockResolvedValue(mockECOImplemented);
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText(/No actions available for implemented ecos/i)).toBeInTheDocument();
    });
  });

  it("shows no actions message for rejected ECO", async () => {
    mockGetECO.mockResolvedValue(mockECORejected);
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText(/No actions available for rejected ecos/i)).toBeInTheDocument();
    });
  });

  it("calls approveECO and refreshes on approve click", async () => {
    mockGetECO.mockResolvedValue(mockECOOpen);
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("Approve ECO")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Approve ECO"));
    await waitFor(() => {
      expect(mockApproveECO).toHaveBeenCalledWith("ECO-001");
    });
  });

  it("calls rejectECO on reject click", async () => {
    mockGetECO.mockResolvedValue(mockECOOpen);
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("Reject ECO")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Reject ECO"));
    await waitFor(() => {
      expect(mockRejectECO).toHaveBeenCalledWith("ECO-001");
    });
  });

  it("calls implementECO on implement click", async () => {
    mockGetECO.mockResolvedValue(mockECOApproved);
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("Implement ECO")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Implement ECO"));
    await waitFor(() => {
      expect(mockImplementECO).toHaveBeenCalledWith("ECO-001");
    });
  });

  // Not found
  it("shows not found state when ECO doesn't exist", async () => {
    mockGetECO.mockRejectedValue(new Error("Not found"));
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("ECO Not Found")).toBeInTheDocument();
    });
    expect(screen.getByText(/could not be found/)).toBeInTheDocument();
  });

  it("shows Actions card title", async () => {
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("Actions")).toBeInTheDocument();
    });
  });

  it("does not show affected parts section when empty", async () => {
    mockGetECO.mockResolvedValue({ ...mockECODraft, affected_parts: [] });
    render(<ECODetail />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    expect(screen.queryByText(/Affected Parts/)).not.toBeInTheDocument();
  });
});
