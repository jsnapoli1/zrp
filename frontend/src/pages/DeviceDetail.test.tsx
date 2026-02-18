import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { mockDevices, mockRMAs } from "../test/mocks";
import type { Device, RMA } from "../lib/api";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const mockGetDevice = vi.fn();
const mockGetRMAs = vi.fn();
const mockUpdateDevice = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getDevice: (...args: any[]) => mockGetDevice(...args),
    getRMAs: (...args: any[]) => mockGetRMAs(...args),
    updateDevice: (...args: any[]) => mockUpdateDevice(...args),
  },
}));

import DeviceDetail from "./DeviceDetail";

const device: Device = {
  ...mockDevices[0],
  firmware_version: "1.0.0",
  customer: "Acme",
  location: "Building A",
  notes: "Test device notes",
  install_date: "2024-01-01",
  last_seen: "2024-01-20T10:00:00Z",
};

const deviceRMAs: RMA[] = [
  { ...mockRMAs[0], serial_number: "SN-100" },
];

function renderWithRoute(serialNumber = "SN-100") {
  return render(
    <MemoryRouter initialEntries={[`/devices/${serialNumber}`]}>
      <Routes>
        <Route path="/devices/:serialNumber" element={<DeviceDetail />} />
      </Routes>
    </MemoryRouter>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mockGetDevice.mockResolvedValue(device);
  mockGetRMAs.mockResolvedValue(deviceRMAs);
  mockUpdateDevice.mockResolvedValue({ ...device, customer: "Updated Corp" });
});

describe("DeviceDetail", () => {
  it("renders loading state", () => {
    renderWithRoute();
    expect(screen.getByText("Loading device...")).toBeInTheDocument();
  });

  it("renders device info after loading", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("SN-100")).toBeInTheDocument();
    });
    expect(screen.getByText("Device Information")).toBeInTheDocument();
  });

  it("shows device serial number as heading", async () => {
    renderWithRoute();
    await waitFor(() => {
      // The h1 heading shows serial number
      const heading = screen.getByRole("heading", { level: 1 });
      expect(heading).toHaveTextContent("SN-100");
    });
  });

  it("displays device details", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Acme")).toBeInTheDocument();
    });
    expect(screen.getByText("Building A")).toBeInTheDocument();
    expect(screen.getByText("Test device notes")).toBeInTheDocument();
  });

  it("shows firmware version", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("1.0.0")).toBeInTheDocument();
    });
  });

  it("shows status badge", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Active")).toBeInTheDocument();
    });
  });

  it("shows device not found when API returns no data", async () => {
    mockGetDevice.mockRejectedValueOnce(new Error("Not found"));
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Device Not Found")).toBeInTheDocument();
    });
  });

  it("shows back button", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Back to Devices")).toBeInTheDocument();
    });
  });

  it("navigates back on back button click", async () => {
    renderWithRoute();
    await waitFor(() => screen.getByText("Back to Devices"));
    fireEvent.click(screen.getByText("Back to Devices"));
    expect(mockNavigate).toHaveBeenCalledWith("/devices");
  });

  // Edit mode
  it("shows Edit Device button", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Edit Device")).toBeInTheDocument();
    });
  });

  it("enters edit mode on Edit Device click", async () => {
    renderWithRoute();
    await waitFor(() => screen.getByText("Edit Device"));
    fireEvent.click(screen.getByText("Edit Device"));
    await waitFor(() => {
      expect(screen.getByText("Save Changes")).toBeInTheDocument();
      expect(screen.getByText("Cancel")).toBeInTheDocument();
    });
  });

  it("shows input fields in edit mode", async () => {
    renderWithRoute();
    await waitFor(() => screen.getByText("Edit Device"));
    fireEvent.click(screen.getByText("Edit Device"));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Internal part number")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Customer name")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Physical location")).toBeInTheDocument();
    });
  });

  it("cancels edit mode", async () => {
    renderWithRoute();
    await waitFor(() => screen.getByText("Edit Device"));
    fireEvent.click(screen.getByText("Edit Device"));
    await waitFor(() => screen.getByText("Cancel"));
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => {
      expect(screen.getByText("Edit Device")).toBeInTheDocument();
      expect(screen.queryByText("Save Changes")).not.toBeInTheDocument();
    });
  });

  it("saves changes on Save click", async () => {
    const user = userEvent.setup();
    renderWithRoute();
    await waitFor(() => screen.getByText("Edit Device"));
    await user.click(screen.getByText("Edit Device"));

    await waitFor(() => screen.getByPlaceholderText("Customer name"));
    const customerInput = screen.getByPlaceholderText("Customer name");
    await user.clear(customerInput);
    await user.type(customerInput, "Updated Corp");

    await user.click(screen.getByText("Save Changes"));
    await waitFor(() => {
      expect(mockUpdateDevice).toHaveBeenCalledWith(
        "SN-100",
        expect.objectContaining({ customer: "Updated Corp" })
      );
    });
  });

  // Device History sidebar
  it("shows Device History section", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Device History")).toBeInTheDocument();
    });
  });

  it("shows created date", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Created")).toBeInTheDocument();
    });
  });

  it("shows last seen when available", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Last Seen")).toBeInTheDocument();
    });
  });

  it("shows Total RMAs count", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Total RMAs")).toBeInTheDocument();
    });
  });

  it("shows Active RMAs count", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Active RMAs")).toBeInTheDocument();
    });
  });

  // Related RMAs
  it("shows Related RMAs section", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Related RMAs")).toBeInTheDocument();
    });
  });

  it("shows related RMA entries", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("RMA-001")).toBeInTheDocument();
    });
  });

  it("shows no RMAs message when none exist", async () => {
    mockGetRMAs.mockResolvedValueOnce([]);
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("No RMAs found for this device.")).toBeInTheDocument();
    });
  });

  // Firmware Management sidebar
  it("shows Firmware Management card", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Firmware Management")).toBeInTheDocument();
    });
  });

  it("shows current firmware version in sidebar", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Current Version")).toBeInTheDocument();
    });
  });

  it("has View Firmware Campaigns button", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("View Firmware Campaigns")).toBeInTheDocument();
    });
  });

  it("navigates to firmware on View Firmware Campaigns click", async () => {
    renderWithRoute();
    await waitFor(() => screen.getByText("View Firmware Campaigns"));
    fireEvent.click(screen.getByText("View Firmware Campaigns"));
    expect(mockNavigate).toHaveBeenCalledWith("/firmware?device=SN-100");
  });

  // Quick Actions
  it("shows Quick Actions card", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Quick Actions")).toBeInTheDocument();
    });
  });

  it("has Create RMA button", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getAllByText("Create RMA").length).toBeGreaterThanOrEqual(1);
    });
  });

  it("has View Test History button", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("View Test History")).toBeInTheDocument();
    });
  });

  it("shows install date when available", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Install Date")).toBeInTheDocument();
    });
  });

  it("shows 'Not specified' for missing firmware version", async () => {
    mockGetDevice.mockResolvedValueOnce({ ...device, firmware_version: "" });
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Not specified")).toBeInTheDocument();
    });
  });

  it("shows 'No notes' when notes are empty", async () => {
    mockGetDevice.mockResolvedValueOnce({ ...device, notes: "" });
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("No notes")).toBeInTheDocument();
    });
  });
});
