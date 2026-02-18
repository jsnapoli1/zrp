import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockRMAs } from "../test/mocks";

const mockGetRMAs = vi.fn().mockResolvedValue(mockRMAs);
const mockCreateRMA = vi.fn().mockResolvedValue(mockRMAs[0]);

vi.mock("../lib/api", () => ({
  api: {
    getRMAs: (...args: any[]) => mockGetRMAs(...args),
    createRMA: (...args: any[]) => mockCreateRMA(...args),
  },
}));

import RMAs from "./RMAs";

beforeEach(() => vi.clearAllMocks());

describe("RMAs", () => {
  it("renders loading state", () => {
    render(<RMAs />);
    expect(screen.getByText("Loading RMAs...")).toBeInTheDocument();
  });

  it("renders RMA list after loading", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("RMA-001")).toBeInTheDocument();
    });
    expect(screen.getByText("RMA-002")).toBeInTheDocument();
  });

  it("has create RMA button", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("Create RMA")).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    mockGetRMAs.mockResolvedValueOnce([]);
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText(/No RMAs found/)).toBeInTheDocument();
    });
  });

  it("shows customer info", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("Acme Inc")).toBeInTheDocument();
      expect(screen.getByText("Tech Co")).toBeInTheDocument();
    });
  });

  it("shows page title and description", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("Return Merchandise Authorization")).toBeInTheDocument();
      expect(screen.getByText("Manage device returns and warranty claims")).toBeInTheDocument();
    });
  });

  it("shows RMA Records card title", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("RMA Records")).toBeInTheDocument();
    });
  });

  it("displays table headers", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("RMA ID")).toBeInTheDocument();
      expect(screen.getByText("Customer")).toBeInTheDocument();
      expect(screen.getByText("Device S/N")).toBeInTheDocument();
      expect(screen.getByText("Reason")).toBeInTheDocument();
      expect(screen.getByText("Status")).toBeInTheDocument();
    });
  });

  it("shows serial numbers", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("SN-500")).toBeInTheDocument();
      expect(screen.getByText("SN-600")).toBeInTheDocument();
    });
  });

  it("shows reasons", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("Device not working")).toBeInTheDocument();
      expect(screen.getByText("Wrong firmware")).toBeInTheDocument();
    });
  });

  it("shows status badges", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("Received")).toBeInTheDocument();
      expect(screen.getByText("Resolved")).toBeInTheDocument();
    });
  });

  it("shows view details buttons", async () => {
    render(<RMAs />);
    await waitFor(() => {
      const viewButtons = screen.getAllByText("View Details");
      expect(viewButtons.length).toBe(2);
    });
  });

  it("opens create RMA dialog", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("RMA-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create RMA"));
    await waitFor(() => {
      expect(screen.getByText("Create New RMA")).toBeInTheDocument();
    });
  });

  it("shows form fields in create dialog", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("RMA-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create RMA"));
    await waitFor(() => {
      expect(screen.getByLabelText("Device Serial Number *")).toBeInTheDocument();
      expect(screen.getByLabelText("Customer *")).toBeInTheDocument();
      expect(screen.getByLabelText("Reason for Return *")).toBeInTheDocument();
      expect(screen.getByLabelText("Defect Description")).toBeInTheDocument();
    });
  });

  it("submits create RMA form", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("RMA-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create RMA"));
    await waitFor(() => {
      expect(screen.getByLabelText("Device Serial Number *")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText("Device Serial Number *"), { target: { value: "SN-999" } });
    fireEvent.change(screen.getByLabelText("Customer *"), { target: { value: "New Corp" } });
    fireEvent.change(screen.getByLabelText("Reason for Return *"), { target: { value: "Broken" } });

    // Submit button is the second "Create RMA"
    const submitButtons = screen.getAllByText("Create RMA");
    fireEvent.click(submitButtons[submitButtons.length - 1]);
    await waitFor(() => {
      expect(mockCreateRMA).toHaveBeenCalledWith(expect.objectContaining({
        serial_number: "SN-999",
        customer: "New Corp",
        reason: "Broken",
      }));
    });
  });

  it("shows cancel button in create dialog", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("RMA-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create RMA"));
    await waitFor(() => {
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
  });

  it("fills defect description in create dialog", async () => {
    render(<RMAs />);
    await waitFor(() => {
      expect(screen.getByText("RMA-001")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Create RMA"));
    await waitFor(() => {
      expect(screen.getByLabelText("Defect Description")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText("Defect Description"), { target: { value: "Screen broken" } });
    expect(screen.getByLabelText("Defect Description")).toHaveValue("Screen broken");
  });
});
