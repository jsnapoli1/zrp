import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import { mockFirmwareCampaigns } from "../test/mocks";
import type { FirmwareCampaign } from "../lib/api";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const mockGetFirmwareCampaigns = vi.fn().mockResolvedValue(mockFirmwareCampaigns);
const mockCreateFirmwareCampaign = vi.fn().mockResolvedValue(mockFirmwareCampaigns[0]);
const mockUpdateFirmwareCampaign = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getFirmwareCampaigns: (...args: any[]) => mockGetFirmwareCampaigns(...args),
    createFirmwareCampaign: (...args: any[]) => mockCreateFirmwareCampaign(...args),
    updateFirmwareCampaign: (...args: any[]) => mockUpdateFirmwareCampaign(...args),
  },
}));

import Firmware from "./Firmware";

const allStatusCampaigns: FirmwareCampaign[] = [
  { id: "FW-R", name: "Running Campaign", version: "2.0.0", category: "Security Update", status: "running", target_filter: "", notes: "", created_at: "2024-01-25", started_at: "2024-01-26" },
  { id: "FW-C", name: "Completed Campaign", version: "1.5.0", category: "Bug Fix", status: "completed", target_filter: "", notes: "", created_at: "2024-01-10", completed_at: "2024-01-20" },
  { id: "FW-P", name: "Paused Campaign", version: "1.3.0", category: "", status: "paused", target_filter: "", notes: "", created_at: "2024-01-15" },
  { id: "FW-D", name: "Draft Campaign", version: "3.0.0", category: "", status: "draft", target_filter: "", notes: "", created_at: "2024-01-20" },
  { id: "FW-F", name: "Failed Campaign", version: "1.1.0", category: "", status: "failed", target_filter: "", notes: "", created_at: "2024-01-05" },
];

beforeEach(() => {
  vi.clearAllMocks();
  mockGetFirmwareCampaigns.mockResolvedValue(mockFirmwareCampaigns);
});

describe("Firmware", () => {
  it("renders loading state", () => {
    render(<Firmware />);
    expect(screen.getByText("Loading firmware campaigns...")).toBeInTheDocument();
  });

  it("renders page title and description", async () => {
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.getByText("Firmware Management")).toBeInTheDocument();
    });
    expect(screen.getByText("Manage firmware update campaigns across device fleet")).toBeInTheDocument();
  });

  it("renders campaign list after loading", async () => {
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.getByText("FW-001")).toBeInTheDocument();
    });
    expect(screen.getByText("FW-002")).toBeInTheDocument();
  });

  it("shows campaign names and versions", async () => {
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.getByText("Update to v1.1")).toBeInTheDocument();
      expect(screen.getByText("Security patch")).toBeInTheDocument();
    });
    expect(screen.getByText("1.1.0")).toBeInTheDocument();
    expect(screen.getByText("1.0.1")).toBeInTheDocument();
  });

  it("displays table headers", async () => {
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.getByText("Campaign ID")).toBeInTheDocument();
    });
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Version")).toBeInTheDocument();
    expect(screen.getByText("Progress")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
    expect(screen.getByText("Created")).toBeInTheDocument();
  });

  it("shows empty state", async () => {
    mockGetFirmwareCampaigns.mockResolvedValueOnce([]);
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.getByText(/No firmware campaigns found/i)).toBeInTheDocument();
    });
  });

  it("navigates to campaign detail on row click", async () => {
    render(<Firmware />);
    await waitFor(() => screen.getByText("FW-001"));
    fireEvent.click(screen.getByText("FW-001"));
    expect(mockNavigate).toHaveBeenCalledWith("/firmware/FW-001");
  });

  it("navigates to detail on View button", async () => {
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.getAllByText("View").length).toBeGreaterThan(0);
    });
    fireEvent.click(screen.getAllByText("View")[0]);
    expect(mockNavigate).toHaveBeenCalledWith("/firmware/FW-001");
  });

  // Statistics cards
  it("shows statistics cards", async () => {
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.getByText("Total Campaigns")).toBeInTheDocument();
    });
    // "Running" may also appear as a badge in the table
    expect(screen.getAllByText("Running").length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText("Completed").length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("Paused/Draft")).toBeInTheDocument();
  });

  it("shows correct statistics for various statuses", async () => {
    mockGetFirmwareCampaigns.mockResolvedValueOnce(allStatusCampaigns);
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.getByText("Total Campaigns")).toBeInTheDocument();
    });
    expect(screen.getByText("5")).toBeInTheDocument(); // total
  });

  // Progress display
  it("shows progress bars for campaigns", async () => {
    render(<Firmware />);
    await waitFor(() => {
      // Progress percentages are shown
      expect(screen.getAllByText(/%/).length).toBeGreaterThan(0);
    });
  });

  // Create campaign dialog
  it("has Create Campaign button", async () => {
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.getByText("Create Campaign")).toBeInTheDocument();
    });
  });

  it("opens create campaign dialog", async () => {
    render(<Firmware />);
    await waitFor(() => screen.getByText("Create Campaign"));
    fireEvent.click(screen.getByText("Create Campaign"));
    await waitFor(() => {
      expect(screen.getByText("Create Firmware Campaign")).toBeInTheDocument();
    });
  });

  it("shows form fields in create dialog", async () => {
    render(<Firmware />);
    await waitFor(() => screen.getByText("Create Campaign"));
    fireEvent.click(screen.getByText("Create Campaign"));
    await waitFor(() => {
      expect(screen.getByLabelText("Campaign Name *")).toBeInTheDocument();
      expect(screen.getByLabelText("Target Version *")).toBeInTheDocument();
      expect(screen.getByLabelText("Target Filter")).toBeInTheDocument();
      expect(screen.getByLabelText("Notes")).toBeInTheDocument();
    });
  });

  it("creates campaign on form submit", async () => {
    const user = userEvent.setup();
    render(<Firmware />);
    await waitFor(() => screen.getByText("Create Campaign"));
    await user.click(screen.getByText("Create Campaign"));

    await waitFor(() => screen.getByLabelText("Campaign Name *"));
    await user.type(screen.getByLabelText("Campaign Name *"), "New Security Update");
    await user.type(screen.getByLabelText("Target Version *"), "v3.0.0");
    await user.type(screen.getByLabelText("Notes"), "Important update");

    // Submit the form
    const submitBtn = screen.getByRole("button", { name: "Create Campaign" });
    await user.click(submitBtn);

    await waitFor(() => {
      expect(mockCreateFirmwareCampaign).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "New Security Update",
          version: "v3.0.0",
          notes: "Important update",
        })
      );
    });
  });

  it("closes create dialog after successful creation", async () => {
    const user = userEvent.setup();
    render(<Firmware />);
    await waitFor(() => screen.getByText("Create Campaign"));
    await user.click(screen.getByText("Create Campaign"));

    await waitFor(() => screen.getByLabelText("Campaign Name *"));
    await user.type(screen.getByLabelText("Campaign Name *"), "Test");
    await user.type(screen.getByLabelText("Target Version *"), "v1.0");

    await user.click(screen.getByRole("button", { name: "Create Campaign" }));

    await waitFor(() => {
      expect(screen.queryByText("Create Firmware Campaign")).not.toBeInTheDocument();
    });
  });

  it("has cancel button in create dialog", async () => {
    render(<Firmware />);
    await waitFor(() => screen.getByText("Create Campaign"));
    fireEvent.click(screen.getByText("Create Campaign"));
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    });
  });

  it("closes dialog on cancel", async () => {
    render(<Firmware />);
    await waitFor(() => screen.getByText("Create Campaign"));
    fireEvent.click(screen.getByText("Create Campaign"));
    await waitFor(() => screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => {
      expect(screen.queryByText("Create Firmware Campaign")).not.toBeInTheDocument();
    });
  });

  it("shows Firmware Campaigns card title", async () => {
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.getByText("Firmware Campaigns")).toBeInTheDocument();
    });
  });

  it("handles fetch error gracefully", async () => {
    mockGetFirmwareCampaigns.mockRejectedValueOnce(new Error("Network error"));
    render(<Firmware />);
    await waitFor(() => {
      expect(screen.queryByText("Loading firmware campaigns...")).not.toBeInTheDocument();
    });
  });

  it("calls API to pause a running campaign and does not navigate", async () => {
    const runningCampaigns: FirmwareCampaign[] = [
      { id: "FW-R", name: "Running", version: "2.0.0", category: "", status: "running", target_filter: "", notes: "", created_at: "2024-01-25" },
    ];
    mockGetFirmwareCampaigns.mockResolvedValueOnce(runningCampaigns);
    mockUpdateFirmwareCampaign.mockResolvedValueOnce({ ...runningCampaigns[0], status: "paused" });
    render(<Firmware />);
    await waitFor(() => screen.getByText("FW-R"));
    mockNavigate.mockClear();
    // The pause button is the first button in the actions cell (has Pause icon)
    const pauseBtn = screen.getByRole("row", { name: /FW-R/ }).querySelector("button");
    fireEvent.click(pauseBtn!);
    await waitFor(() => {
      expect(mockUpdateFirmwareCampaign).toHaveBeenCalledWith("FW-R", { status: "paused" });
    });
    // Should NOT navigate (stopPropagation)
    expect(mockNavigate).not.toHaveBeenCalledWith("/firmware/FW-R");
  });

  it("calls API to start a paused campaign and does not navigate", async () => {
    const pausedCampaigns: FirmwareCampaign[] = [
      { id: "FW-P", name: "Paused", version: "1.0.0", category: "", status: "paused", target_filter: "", notes: "", created_at: "2024-01-25" },
    ];
    mockGetFirmwareCampaigns.mockResolvedValueOnce(pausedCampaigns);
    mockUpdateFirmwareCampaign.mockResolvedValueOnce({ ...pausedCampaigns[0], status: "running" });
    render(<Firmware />);
    await waitFor(() => screen.getByText("FW-P"));
    mockNavigate.mockClear();
    const playBtn = screen.getByRole("row", { name: /FW-P/ }).querySelector("button");
    fireEvent.click(playBtn!);
    await waitFor(() => {
      expect(mockUpdateFirmwareCampaign).toHaveBeenCalledWith("FW-P", { status: "running" });
    });
    expect(mockNavigate).not.toHaveBeenCalledWith("/firmware/FW-P");
  });
});
