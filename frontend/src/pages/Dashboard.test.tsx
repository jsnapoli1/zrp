import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import { mockDashboardStats } from "../test/mocks";

const mockGetDashboard = vi.fn().mockResolvedValue(mockDashboardStats);
const mockGetDashboardCharts = vi.fn().mockResolvedValue({ eco_counts: [1, 2, 3] });

vi.mock("../lib/api", () => ({
  api: {
    getDashboard: (...args: any[]) => mockGetDashboard(...args),
    getDashboardCharts: (...args: any[]) => mockGetDashboardCharts(...args),
  },
}));

import Dashboard from "./Dashboard";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetDashboard.mockResolvedValue(mockDashboardStats);
  mockGetDashboardCharts.mockResolvedValue({ eco_counts: [1, 2, 3] });
});

describe("Dashboard", () => {
  it("renders loading state initially", () => {
    render(<Dashboard />);
    expect(screen.getByText("Loading dashboard...")).toBeInTheDocument();
  });

  it("renders KPI cards after loading", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText("Dashboard")).toBeInTheDocument();
    });
    expect(screen.getByText("Total Parts")).toBeInTheDocument();
    expect(screen.getByText("Low Stock")).toBeInTheDocument();
    expect(screen.getByText("Active Work Orders")).toBeInTheDocument();
    expect(screen.getByText("Open ECOs")).toBeInTheDocument();
    expect(screen.getByText("Open POs")).toBeInTheDocument();
    expect(screen.getByText("Open NCRs")).toBeInTheDocument();
    expect(screen.getByText("Total Devices")).toBeInTheDocument();
    expect(screen.getByText("Open RMAs")).toBeInTheDocument();
  });

  it("displays stat values after loading", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText("Total Parts")).toBeInTheDocument();
    });
    // Stats are rendered - just verify the cards exist with numbers
    const statElements = screen.getAllByText(/^\d+$/);
    expect(statElements.length).toBeGreaterThan(0);
  });

  it("shows recent activity section", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText("Recent Activity")).toBeInTheDocument();
    });
    expect(screen.getByText(/Widget Improvement/)).toBeInTheDocument();
    expect(screen.getByText(/WO-001 completed/)).toBeInTheDocument();
  });

  it("shows ECO Status chart placeholder", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText("ECO Status")).toBeInTheDocument();
    });
  });

  it("calls API on mount", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(mockGetDashboard).toHaveBeenCalledTimes(1);
      expect(mockGetDashboardCharts).toHaveBeenCalledTimes(1);
    });
  });

  it("handles API error gracefully", async () => {
    mockGetDashboard.mockRejectedValueOnce(new Error("Network error"));
    render(<Dashboard />);
    // Should not crash - just logs error and stops loading
    await waitFor(() => {
      expect(screen.queryByText("Loading dashboard...")).not.toBeInTheDocument();
    });
  });

  it("renders welcome message", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText(/Welcome back/)).toBeInTheDocument();
    });
  });

  it("renders all 8 KPI cards", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText("Total Parts")).toBeInTheDocument();
    });
    const expectedCards = [
      "Total Parts", "Open ECOs", "Low Stock", "Active Work Orders",
      "Open POs", "Open NCRs", "Total Devices", "Open RMAs",
    ];
    for (const title of expectedCards) {
      expect(screen.getByText(title)).toBeInTheDocument();
    }
  });

  it("renders chart placeholder", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText(/Chart.js integration needed/)).toBeInTheDocument();
    });
  });

  it("renders activity feed with user and timestamp", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText(/John Doe/)).toBeInTheDocument();
    });
    expect(screen.getByText(/Jane Smith/)).toBeInTheDocument();
    expect(screen.getByText(/System/)).toBeInTheDocument();
  });

  it("renders activity type badges", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText("ECO")).toBeInTheDocument();
    });
    expect(screen.getByText("Work Order")).toBeInTheDocument();
    expect(screen.getByText("Inventory")).toBeInTheDocument();
  });

  it("displays stat values from mock data", async () => {
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.getByText("Total Parts")).toBeInTheDocument();
    });
    // Verify numeric values render - at least some stats show as numbers
    const statElements = screen.getAllByText(/^\d+$/);
    expect(statElements.length).toBeGreaterThanOrEqual(8); // 8 KPI cards
  });

  it("handles charts API error gracefully", async () => {
    mockGetDashboardCharts.mockRejectedValueOnce(new Error("Charts fail"));
    render(<Dashboard />);
    await waitFor(() => {
      expect(screen.queryByText("Loading dashboard...")).not.toBeInTheDocument();
    });
  });
});
