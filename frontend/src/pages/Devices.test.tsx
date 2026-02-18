import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import { mockDevices } from "../test/mocks";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const mockGetDevices = vi.fn().mockResolvedValue(mockDevices);
const mockCreateDevice = vi.fn().mockResolvedValue(mockDevices[0]);
const mockImportDevices = vi.fn().mockResolvedValue({ success: 5, errors: [] });
const mockExportDevices = vi.fn().mockResolvedValue(new Blob(["csv data"]));

vi.mock("../lib/api", () => ({
  api: {
    getDevices: (...args: any[]) => mockGetDevices(...args),
    createDevice: (...args: any[]) => mockCreateDevice(...args),
    importDevices: (...args: any[]) => mockImportDevices(...args),
    exportDevices: (...args: any[]) => mockExportDevices(...args),
  },
}));

import Devices from "./Devices";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetDevices.mockResolvedValue(mockDevices);
});

describe("Devices", () => {
  it("renders loading state", () => {
    render(<Devices />);
    expect(screen.getByText("Loading devices...")).toBeInTheDocument();
  });

  it("renders device list after loading", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("SN-100")).toBeInTheDocument();
    });
    expect(screen.getByText("SN-101")).toBeInTheDocument();
  });

  it("shows page header and description", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Device Registry")).toBeInTheDocument();
    });
    expect(screen.getByText("Manage device inventory and track firmware versions")).toBeInTheDocument();
  });

  it("displays device table with correct columns", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Serial Number")).toBeInTheDocument();
    });
    expect(screen.getByText("IPN")).toBeInTheDocument();
    expect(screen.getByText("Firmware Version")).toBeInTheDocument();
    expect(screen.getByText("Customer")).toBeInTheDocument();
    expect(screen.getByText("Location")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
    expect(screen.getByText("Last Seen")).toBeInTheDocument();
    expect(screen.getByText("Actions")).toBeInTheDocument();
  });

  it("shows device details in table rows", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Acme")).toBeInTheDocument();
      expect(screen.getByText("Tech Co")).toBeInTheDocument();
    });
    expect(screen.getByText("Building A")).toBeInTheDocument();
    expect(screen.getByText("Floor 2")).toBeInTheDocument();
    expect(screen.getByText("1.0.0")).toBeInTheDocument();
    expect(screen.getByText("0.9.0")).toBeInTheDocument();
  });

  it("shows status badges", async () => {
    render(<Devices />);
    await waitFor(() => {
      const badges = screen.getAllByText("Active");
      expect(badges.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("shows empty state when no devices", async () => {
    mockGetDevices.mockResolvedValueOnce([]);
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText(/No devices found/i)).toBeInTheDocument();
    });
  });

  it("navigates to device detail on row click", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("SN-100")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("SN-100"));
    expect(mockNavigate).toHaveBeenCalledWith("/devices/SN-100");
  });

  it("navigates to device detail on View Details button", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getAllByText("View Details").length).toBeGreaterThan(0);
    });
    fireEvent.click(screen.getAllByText("View Details")[0]);
    expect(mockNavigate).toHaveBeenCalledWith("/devices/SN-100");
  });

  it("shows statistics cards", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Total Devices")).toBeInTheDocument();
    });
    expect(screen.getByText("Maintenance")).toBeInTheDocument();
    expect(screen.getByText("Inactive")).toBeInTheDocument();
  });

  it("shows correct statistics counts", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Total Devices")).toBeInTheDocument();
    });
    // 2 devices total, both active
    expect(screen.getByText("2")).toBeInTheDocument(); // total
  });

  // Export CSV
  it("has Export CSV button", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Export CSV")).toBeInTheDocument();
    });
  });

  it("calls exportDevices on Export CSV click", async () => {
    // Mock URL methods
    const mockCreateObjectURL = vi.fn().mockReturnValue("blob:test");
    const mockRevokeObjectURL = vi.fn();
    Object.defineProperty(window, "URL", {
      value: { createObjectURL: mockCreateObjectURL, revokeObjectURL: mockRevokeObjectURL },
      writable: true,
    });

    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Export CSV")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Export CSV"));
    await waitFor(() => {
      expect(mockExportDevices).toHaveBeenCalled();
    });
  });

  // Import CSV
  it("has Import CSV button", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Import CSV")).toBeInTheDocument();
    });
  });

  it("opens import dialog when Import CSV clicked", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Import CSV")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Import CSV"));
    await waitFor(() => {
      expect(screen.getByText("Import Devices from CSV")).toBeInTheDocument();
    });
  });

  it("shows file input and description in import dialog", async () => {
    render(<Devices />);
    await waitFor(() => screen.getByText("Import CSV"));
    fireEvent.click(screen.getByText("Import CSV"));
    await waitFor(() => {
      expect(screen.getByText(/CSV should include columns/)).toBeInTheDocument();
    });
    expect(screen.getByLabelText("CSV File")).toBeInTheDocument();
  });

  it("disables import button when no file selected", async () => {
    render(<Devices />);
    await waitFor(() => screen.getByText("Import CSV"));
    fireEvent.click(screen.getByText("Import CSV"));
    await waitFor(() => {
      const importBtn = screen.getByRole("button", { name: "Import" });
      expect(importBtn).toBeDisabled();
    });
  });

  it("shows import results after successful import", async () => {
    mockImportDevices.mockResolvedValueOnce({ success: 3, errors: [] });
    render(<Devices />);
    await waitFor(() => screen.getByText("Import CSV"));
    fireEvent.click(screen.getByText("Import CSV"));

    await waitFor(() => screen.getByLabelText("CSV File"));
    const fileInput = screen.getByLabelText("CSV File");
    const file = new File(["csv content"], "devices.csv", { type: "text/csv" });
    fireEvent.change(fileInput, { target: { files: [file] } });

    const importBtn = screen.getByRole("button", { name: "Import" });
    fireEvent.click(importBtn);

    await waitFor(() => {
      expect(screen.getByText(/Successfully imported: 3 devices/)).toBeInTheDocument();
    });
  });

  it("shows import errors", async () => {
    mockImportDevices.mockResolvedValueOnce({ success: 1, errors: ["Row 2: missing serial_number"] });
    render(<Devices />);
    await waitFor(() => screen.getByText("Import CSV"));
    fireEvent.click(screen.getByText("Import CSV"));

    await waitFor(() => screen.getByLabelText("CSV File"));
    const fileInput = screen.getByLabelText("CSV File");
    const file = new File(["csv"], "devices.csv", { type: "text/csv" });
    fireEvent.change(fileInput, { target: { files: [file] } });

    fireEvent.click(screen.getByRole("button", { name: "Import" }));

    await waitFor(() => {
      expect(screen.getByText(/Errors \(1\)/)).toBeInTheDocument();
      expect(screen.getByText("Row 2: missing serial_number")).toBeInTheDocument();
    });
  });

  // Create device dialog
  it("has Add Device button", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Add Device")).toBeInTheDocument();
    });
  });

  it("opens create device dialog", async () => {
    render(<Devices />);
    await waitFor(() => screen.getByText("Add Device"));
    fireEvent.click(screen.getByText("Add Device"));
    await waitFor(() => {
      expect(screen.getByText("Add New Device")).toBeInTheDocument();
    });
  });

  it("shows create device form fields", async () => {
    render(<Devices />);
    await waitFor(() => screen.getByText("Add Device"));
    fireEvent.click(screen.getByText("Add Device"));
    await waitFor(() => {
      expect(screen.getByLabelText("Serial Number *")).toBeInTheDocument();
      expect(screen.getByLabelText("IPN")).toBeInTheDocument();
      expect(screen.getByLabelText("Firmware Version")).toBeInTheDocument();
      expect(screen.getByLabelText("Customer")).toBeInTheDocument();
      expect(screen.getByLabelText("Location")).toBeInTheDocument();
      expect(screen.getByLabelText("Notes")).toBeInTheDocument();
    });
  });

  it("disables Add Device submit when serial_number empty", async () => {
    render(<Devices />);
    await waitFor(() => screen.getByText("Add Device"));
    fireEvent.click(screen.getByText("Add Device"));
    await waitFor(() => {
      // The submit button inside dialog also says "Add Device"
      const buttons = screen.getAllByRole("button", { name: "Add Device" });
      const submitBtn = buttons[buttons.length - 1];
      expect(submitBtn).toBeDisabled();
    });
  });

  it("creates device and refreshes list", async () => {
    const user = userEvent.setup();
    render(<Devices />);
    await waitFor(() => screen.getByText("Add Device"));
    await user.click(screen.getByText("Add Device"));
    await waitFor(() => screen.getByLabelText("Serial Number *"));

    await user.type(screen.getByLabelText("Serial Number *"), "SN-999");
    
    // Find the submit button (second "Add Device" button)
    const buttons = screen.getAllByRole("button", { name: "Add Device" });
    const submitBtn = buttons[buttons.length - 1];
    expect(submitBtn).not.toBeDisabled();
    await user.click(submitBtn);

    await waitFor(() => {
      expect(mockCreateDevice).toHaveBeenCalledWith(
        expect.objectContaining({ serial_number: "SN-999" })
      );
    });
    // Refreshes list after create
    expect(mockGetDevices).toHaveBeenCalledTimes(2);
  });

  it("handles fetch error gracefully", async () => {
    mockGetDevices.mockRejectedValueOnce(new Error("Network error"));
    render(<Devices />);
    await waitFor(() => {
      // Should finish loading even on error
      expect(screen.queryByText("Loading devices...")).not.toBeInTheDocument();
    });
  });

  it("renders Device Inventory card title", async () => {
    render(<Devices />);
    await waitFor(() => {
      expect(screen.getByText("Device Inventory")).toBeInTheDocument();
    });
  });
});
