import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import type { FirmwareCampaign, CampaignDevice } from "../lib/api";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const mockGetFirmwareCampaign = vi.fn();
const mockGetCampaignDevices = vi.fn();
const mockUpdateFirmwareCampaign = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getFirmwareCampaign: (...args: any[]) => mockGetFirmwareCampaign(...args),
    getCampaignDevices: (...args: any[]) => mockGetCampaignDevices(...args),
    updateFirmwareCampaign: (...args: any[]) => mockUpdateFirmwareCampaign(...args),
  },
}));

import FirmwareDetail from "./FirmwareDetail";

const runningCampaign: FirmwareCampaign = {
  id: "FW-001",
  name: "Security Update Q1",
  version: "2.0.0",
  category: "Security Update",
  status: "running",
  target_filter: "ipn:DEV-001",
  notes: "Critical security fixes",
  created_at: "2024-01-25T10:00:00Z",
  started_at: "2024-01-26T08:00:00Z",
};

const completedCampaign: FirmwareCampaign = {
  ...runningCampaign,
  id: "FW-002",
  name: "Completed Update",
  status: "completed",
  completed_at: "2024-01-30T12:00:00Z",
};

const failedCampaign: FirmwareCampaign = {
  ...runningCampaign,
  id: "FW-003",
  name: "Failed Update",
  status: "failed",
};

const pausedCampaign: FirmwareCampaign = {
  ...runningCampaign,
  id: "FW-004",
  name: "Paused Update",
  status: "paused",
};

const draftCampaign: FirmwareCampaign = {
  ...runningCampaign,
  id: "FW-005",
  name: "Draft Update",
  status: "draft",
};

const mockDevices: CampaignDevice[] = [
  { campaign_id: "FW-001", serial_number: "SN-100", status: "completed", updated_at: "2024-01-27T10:00:00Z" },
  { campaign_id: "FW-001", serial_number: "SN-101", status: "in_progress", updated_at: "2024-01-27T09:00:00Z" },
  { campaign_id: "FW-001", serial_number: "SN-102", status: "pending", updated_at: null as any },
  { campaign_id: "FW-001", serial_number: "SN-103", status: "failed", updated_at: "2024-01-27T08:00:00Z" },
];

function renderWithRoute(id = "FW-001") {
  return render(
    <MemoryRouter initialEntries={[`/firmware/${id}`]}>
      <Routes>
        <Route path="/firmware/:id" element={<FirmwareDetail />} />
      </Routes>
    </MemoryRouter>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers({ shouldAdvanceTime: true });
  mockGetFirmwareCampaign.mockResolvedValue(runningCampaign);
  mockGetCampaignDevices.mockResolvedValue(mockDevices);
});

afterEach(() => {
  vi.useRealTimers();
});

describe("FirmwareDetail", () => {
  it("renders loading state", () => {
    renderWithRoute();
    expect(screen.getByText("Loading campaign...")).toBeInTheDocument();
  });

  it("renders campaign name after loading", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Security Update Q1");
    });
  });

  it("shows target version", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("2.0.0")).toBeInTheDocument();
    });
  });

  it("shows back button", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Back to Firmware")).toBeInTheDocument();
    });
  });

  it("navigates back on back button click", async () => {
    renderWithRoute();
    await waitFor(() => screen.getByText("Back to Firmware"));
    fireEvent.click(screen.getByText("Back to Firmware"));
    expect(mockNavigate).toHaveBeenCalledWith("/firmware");
  });

  it("shows Campaign Not Found when no data", async () => {
    mockGetFirmwareCampaign.mockRejectedValueOnce(new Error("Not found"));
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Campaign Not Found")).toBeInTheDocument();
    });
  });

  // Campaign Progress
  it("shows Campaign Progress section", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Campaign Progress")).toBeInTheDocument();
    });
  });

  it("shows overall progress percentage", async () => {
    renderWithRoute();
    await waitFor(() => {
      // 1 of 4 completed = 25%
      expect(screen.getByText(/25%/)).toBeInTheDocument();
    });
  });

  it("shows device stats breakdown", async () => {
    renderWithRoute();
    await waitFor(() => {
      // These labels appear in the stats section
      expect(screen.getAllByText("Completed").length).toBeGreaterThanOrEqual(1);
      expect(screen.getByText("In Progress")).toBeInTheDocument();
      expect(screen.getAllByText("Pending").length).toBeGreaterThanOrEqual(1);
      expect(screen.getAllByText("Failed").length).toBeGreaterThanOrEqual(1);
    });
  });

  it("shows correct stat counts", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Campaign Progress")).toBeInTheDocument();
    });
    // Stats: 1 completed, 1 in_progress, 1 pending, 1 failed
    const statValues = screen.getAllByText("1");
    expect(statValues.length).toBeGreaterThanOrEqual(4);
  });

  // Device Update Status table
  it("shows Device Update Status section", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Device Update Status")).toBeInTheDocument();
    });
  });

  it("shows device serial numbers in table", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("SN-100")).toBeInTheDocument();
      expect(screen.getByText("SN-101")).toBeInTheDocument();
      expect(screen.getByText("SN-102")).toBeInTheDocument();
      expect(screen.getByText("SN-103")).toBeInTheDocument();
    });
  });

  it("shows no devices message when empty", async () => {
    mockGetCampaignDevices.mockResolvedValueOnce([]);
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("No devices found for this campaign.")).toBeInTheDocument();
    });
  });

  it("navigates to device on serial number click", async () => {
    renderWithRoute();
    await waitFor(() => screen.getByText("SN-100"));
    fireEvent.click(screen.getByText("SN-100"));
    expect(mockNavigate).toHaveBeenCalledWith("/devices/SN-100");
  });

  it("navigates to device on View Device button", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getAllByText("View Device").length).toBeGreaterThan(0);
    });
    fireEvent.click(screen.getAllByText("View Device")[0]);
    expect(mockNavigate).toHaveBeenCalledWith("/devices/SN-100");
  });

  // Campaign Details sidebar
  it("shows Campaign Details card", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Campaign Details")).toBeInTheDocument();
    });
  });

  it("shows category", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Category")).toBeInTheDocument();
      expect(screen.getByText("Security Update")).toBeInTheDocument();
    });
  });

  it("shows target filter", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Target Filter")).toBeInTheDocument();
      expect(screen.getByText("ipn:DEV-001")).toBeInTheDocument();
    });
  });

  it("shows 'All devices' when no target filter", async () => {
    mockGetFirmwareCampaign.mockResolvedValue({ ...runningCampaign, target_filter: "" });
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("All devices")).toBeInTheDocument();
    });
  });

  it("shows created date", async () => {
    renderWithRoute();
    await waitFor(() => {
      const labels = screen.getAllByText("Created");
      expect(labels.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("shows started date for running campaign", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Started")).toBeInTheDocument();
    });
  });

  it("shows completed date for completed campaign", async () => {
    mockGetFirmwareCampaign.mockResolvedValue(completedCampaign);
    renderWithRoute("FW-002");
    await waitFor(() => {
      // "Completed" appears both as status label and date label
      const completedTexts = screen.getAllByText("Completed");
      expect(completedTexts.length).toBeGreaterThanOrEqual(1);
    });
  });

  // Release Notes
  it("shows Release Notes section", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Release Notes")).toBeInTheDocument();
    });
  });

  it("shows release notes content", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Critical security fixes")).toBeInTheDocument();
    });
  });

  it("shows default when no notes", async () => {
    mockGetFirmwareCampaign.mockResolvedValue({ ...runningCampaign, notes: "" });
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("No release notes available.")).toBeInTheDocument();
    });
  });

  // Action buttons based on status
  it("shows Pause Campaign button for running campaign", async () => {
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Pause Campaign")).toBeInTheDocument();
    });
  });

  it("shows Start Campaign button for paused campaign", async () => {
    mockGetFirmwareCampaign.mockResolvedValue(pausedCampaign);
    renderWithRoute("FW-004");
    await waitFor(() => {
      expect(screen.getByText("Start Campaign")).toBeInTheDocument();
    });
  });

  it("shows Start Campaign button for draft campaign", async () => {
    mockGetFirmwareCampaign.mockResolvedValue(draftCampaign);
    renderWithRoute("FW-005");
    await waitFor(() => {
      expect(screen.getByText("Start Campaign")).toBeInTheDocument();
    });
  });

  it("shows Retry Failed button for failed campaign", async () => {
    mockGetFirmwareCampaign.mockResolvedValue(failedCampaign);
    renderWithRoute("FW-003");
    await waitFor(() => {
      expect(screen.getByText("Retry Failed")).toBeInTheDocument();
    });
  });

  it("shows no action buttons for completed campaign", async () => {
    mockGetFirmwareCampaign.mockResolvedValue(completedCampaign);
    mockGetCampaignDevices.mockResolvedValue([]);
    renderWithRoute("FW-002");
    await waitFor(() => {
      expect(screen.getByText("Completed Update")).toBeInTheDocument();
    });
    expect(screen.queryByText("Pause Campaign")).not.toBeInTheDocument();
    expect(screen.queryByText("Start Campaign")).not.toBeInTheDocument();
    expect(screen.queryByText("Retry Failed")).not.toBeInTheDocument();
  });

  // Progress with 0 devices
  it("shows 0% progress when no devices", async () => {
    mockGetCampaignDevices.mockResolvedValueOnce([]);
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText(/0%/)).toBeInTheDocument();
    });
  });

  it("shows 'Not specified' for empty category", async () => {
    mockGetFirmwareCampaign.mockResolvedValue({ ...runningCampaign, category: "" });
    renderWithRoute();
    await waitFor(() => {
      expect(screen.getByText("Not specified")).toBeInTheDocument();
    });
  });

  // Bug 2: Action button handlers
  it("calls API to pause a running campaign", async () => {
    mockUpdateFirmwareCampaign.mockResolvedValueOnce({ ...runningCampaign, status: "paused" });
    renderWithRoute();
    await waitFor(() => screen.getByText("Pause Campaign"));
    fireEvent.click(screen.getByText("Pause Campaign"));
    await waitFor(() => {
      expect(mockUpdateFirmwareCampaign).toHaveBeenCalledWith("FW-001", { status: "paused" });
    });
  });

  it("calls API to start a paused campaign", async () => {
    mockGetFirmwareCampaign.mockResolvedValue(pausedCampaign);
    mockUpdateFirmwareCampaign.mockResolvedValueOnce({ ...pausedCampaign, status: "running" });
    renderWithRoute("FW-004");
    await waitFor(() => screen.getByText("Start Campaign"));
    fireEvent.click(screen.getByText("Start Campaign"));
    await waitFor(() => {
      expect(mockUpdateFirmwareCampaign).toHaveBeenCalledWith("FW-004", { status: "running" });
    });
  });

  it("calls API to start a draft campaign", async () => {
    mockGetFirmwareCampaign.mockResolvedValue(draftCampaign);
    mockUpdateFirmwareCampaign.mockResolvedValueOnce({ ...draftCampaign, status: "running" });
    renderWithRoute("FW-005");
    await waitFor(() => screen.getByText("Start Campaign"));
    fireEvent.click(screen.getByText("Start Campaign"));
    await waitFor(() => {
      expect(mockUpdateFirmwareCampaign).toHaveBeenCalledWith("FW-005", { status: "running" });
    });
  });

  it("calls API to retry a failed campaign", async () => {
    mockGetFirmwareCampaign.mockResolvedValue(failedCampaign);
    mockUpdateFirmwareCampaign.mockResolvedValueOnce({ ...failedCampaign, status: "running" });
    renderWithRoute("FW-003");
    await waitFor(() => screen.getByText("Retry Failed"));
    fireEvent.click(screen.getByText("Retry Failed"));
    await waitFor(() => {
      expect(mockUpdateFirmwareCampaign).toHaveBeenCalledWith("FW-003", { status: "running" });
    });
  });

  // Bug 3: Polling uses ref, so it polls when status is running
  it("polls for updates when campaign is running", async () => {
    renderWithRoute();
    await waitFor(() => screen.getByText("Security Update Q1"));
    // Initial fetch: 1 call each
    expect(mockGetFirmwareCampaign).toHaveBeenCalledTimes(1);
    // Advance 5 seconds to trigger poll
    await vi.advanceTimersByTimeAsync(5000);
    await waitFor(() => {
      expect(mockGetFirmwareCampaign).toHaveBeenCalledTimes(2);
    });
  });

  it("does not poll when campaign is not running", async () => {
    mockGetFirmwareCampaign.mockResolvedValue(completedCampaign);
    mockGetCampaignDevices.mockResolvedValue([]);
    renderWithRoute("FW-002");
    await waitFor(() => screen.getByText("Completed Update"));
    const callCount = mockGetFirmwareCampaign.mock.calls.length;
    await vi.advanceTimersByTimeAsync(5000);
    // Should not have made additional calls
    expect(mockGetFirmwareCampaign).toHaveBeenCalledTimes(callCount);
  });
});
