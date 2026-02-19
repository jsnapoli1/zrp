import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockRMAs } from "../test/mocks";
import type { RMA } from "../lib/api";

const mockRMADetail: RMA = {
  ...mockRMAs[0],
  defect_description: "Device has no power when plugged in",
  resolution: "",
};

const mockGetRMA = vi.fn().mockResolvedValue(mockRMADetail);
const mockUpdateRMA = vi.fn().mockResolvedValue({ ...mockRMADetail, status: "investigating" });

vi.mock("../lib/api", () => ({
  api: {
    getRMA: (...args: any[]) => mockGetRMA(...args),
    updateRMA: (...args: any[]) => mockUpdateRMA(...args),
  },
}));

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return {
    ...actual,
    useParams: () => ({ id: "RMA-001" }),
  };
});

import RMADetail from "./RMADetail";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetRMA.mockResolvedValue(mockRMADetail);
});

describe("RMADetail", () => {
  it("renders loading state", () => {
    render(<RMADetail />);
    expect(screen.getByText("Loading RMA...")).toBeInTheDocument();
  });

  it("renders RMA details after loading", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("RMA-001")).toBeInTheDocument();
    });
  });

  it("shows customer and serial number in subtitle", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Acme Inc - SN-500")).toBeInTheDocument();
    });
  });

  it("shows RMA Details card", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("RMA Details")).toBeInTheDocument();
    });
  });

  it("shows customer name", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Acme Inc")).toBeInTheDocument();
    });
  });

  it("shows serial number", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      // Serial number appears in multiple places
      const snElements = screen.getAllByText("SN-500");
      expect(snElements.length).toBeGreaterThan(0);
    });
  });

  it("shows reason for return", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Device not working")).toBeInTheDocument();
    });
  });

  it("shows defect description", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Device has no power when plugged in")).toBeInTheDocument();
    });
  });

  it("shows status badge", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      // "Received" appears as both badge and workflow label
      const receivedElements = screen.getAllByText("Received");
      expect(receivedElements.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("shows resolution pending when no resolution", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Resolution pending")).toBeInTheDocument();
    });
  });

  it("shows resolution when set", async () => {
    mockGetRMA.mockResolvedValueOnce({ ...mockRMADetail, resolution: "Replaced power supply" });
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Replaced power supply")).toBeInTheDocument();
    });
  });

  it("shows Status Workflow card", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Status Workflow")).toBeInTheDocument();
    });
  });

  it("shows all workflow statuses", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Open")).toBeInTheDocument();
      // "Received" is both badge and workflow label
      expect(screen.getByText("Investigating")).toBeInTheDocument();
      expect(screen.getByText("Resolved")).toBeInTheDocument();
      expect(screen.getByText("Shipped")).toBeInTheDocument();
      expect(screen.getByText("Closed")).toBeInTheDocument();
    });
  });

  it("shows workflow descriptions", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("RMA request created")).toBeInTheDocument();
      expect(screen.getByText("Device received for inspection")).toBeInTheDocument();
      expect(screen.getByText("Analyzing the defect")).toBeInTheDocument();
    });
  });

  it("shows Timeline card", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Timeline")).toBeInTheDocument();
      expect(screen.getByText("Created")).toBeInTheDocument();
    });
  });

  it("shows Device Information card", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Device Information")).toBeInTheDocument();
      expect(screen.getByText("Serial Number:")).toBeInTheDocument();
    });
  });

  it("has View Device Details button", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("View Device Details")).toBeInTheDocument();
    });
  });

  it("has Back to RMAs button", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Back to RMAs")).toBeInTheDocument();
    });
  });

  it("has Edit RMA button", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit RMA")).toBeInTheDocument();
    });
  });

  it("enters edit mode when Edit RMA clicked", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit RMA")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit RMA"));
    await waitFor(() => {
      expect(screen.getByText("Save Changes")).toBeInTheDocument();
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
  });

  it("shows editable fields in edit mode", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit RMA")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit RMA"));
    await waitFor(() => {
      expect(screen.getByDisplayValue("Acme Inc")).toBeInTheDocument();
      expect(screen.getByDisplayValue("SN-500")).toBeInTheDocument();
      expect(screen.getByDisplayValue("Device not working")).toBeInTheDocument();
    });
  });

  it("shows resolution textarea in edit mode", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit RMA")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit RMA"));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Resolution taken for this RMA")).toBeInTheDocument();
    });
  });

  it("saves changes when Save Changes clicked", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit RMA")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit RMA"));
    await waitFor(() => {
      expect(screen.getByText("Save Changes")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Save Changes"));
    await waitFor(() => {
      expect(mockUpdateRMA).toHaveBeenCalledWith("RMA-001", expect.any(Object));
    });
  });

  it("cancels edit mode", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Edit RMA")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Edit RMA"));
    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.getByText("Edit RMA")).toBeInTheDocument();
    });
  });

  it("shows RMA Not Found when rma is null", async () => {
    mockGetRMA.mockResolvedValueOnce(null);
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("RMA Not Found")).toBeInTheDocument();
    });
  });

  it("shows Back to RMAs on not found page", async () => {
    mockGetRMA.mockResolvedValueOnce(null);
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Back to RMAs")).toBeInTheDocument();
    });
  });

  it("shows labels for fields", async () => {
    render(<RMADetail />);
    await waitFor(() => {
      expect(screen.getByText("Customer")).toBeInTheDocument();
      expect(screen.getByText("Device Serial Number")).toBeInTheDocument();
      expect(screen.getByText("Reason for Return")).toBeInTheDocument();
      expect(screen.getByText("Defect Description")).toBeInTheDocument();
      expect(screen.getByText("Resolution")).toBeInTheDocument();
    });
  });
});
