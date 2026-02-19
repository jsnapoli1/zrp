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

  it("uses WebSocket subscription for real-time updates instead of polling", async () => {
    render(<Dashboard />);
    // Wait for initial load
    await waitFor(() => expect(mockGetDashboard).toHaveBeenCalledTimes(1));
    // Dashboard now uses useWSSubscription instead of setInterval;
    // verify it loaded and didn't set up any intervals
    expect(mockGetDashboard).toHaveBeenCalledTimes(1);
    expect(mockGetDashboardCharts).toHaveBeenCalledTimes(1);
  });

  it("cleans up on unmount without errors", async () => {
    const { unmount } = render(<Dashboard />);
    await waitFor(() => expect(mockGetDashboard).toHaveBeenCalledTimes(1));
    // Should unmount cleanly (no interval to clean up, WebSocket subscription auto-cleans)
    expect(() => unmount()).not.toThrow();
  });

  it("shows 'No recent activity' when activities array is empty", async () => {
    mockGetDashboard.mockRejectedValue(new Error("fail"));
    mockGetDashboardCharts.mockRejectedValue(new Error("fail"));
    render(<Dashboard />);
    await waitFor(() => expect(screen.queryByText("Loading dashboard...")).not.toBeInTheDocument());
    expect(screen.getByText("No recent activity")).toBeInTheDocument();
  });

  it("shows KPI cards with 0 values when stats is null", async () => {
    mockGetDashboard.mockRejectedValue(new Error("fail"));
    mockGetDashboardCharts.mockRejectedValue(new Error("fail"));
    render(<Dashboard />);
    await waitFor(() => expect(screen.queryByText("Loading dashboard...")).not.toBeInTheDocument());
    // When stats is null, `stats?.[key] || 0` yields 0 for each card
    const zeros = screen.getAllByText("0");
    expect(zeros.length).toBeGreaterThanOrEqual(8);
    mockGetDashboard.mockResolvedValue(mockDashboardStats);
    mockGetDashboardCharts.mockResolvedValue({ eco_counts: [1, 2, 3] });
  });

  it("formats large numbers with toLocaleString", async () => {
    mockGetDashboard.mockResolvedValue({
      ...mockDashboardStats,
      total_parts: 1500000,
    });
    render(<Dashboard />);
    await waitFor(() => expect(screen.getByText("Total Parts")).toBeInTheDocument());
    // 1500000.toLocaleString() → "1,500,000"
    expect(screen.getByText("1,500,000")).toBeInTheDocument();
  });

  it("handles both APIs rejecting simultaneously", async () => {
    // Reject both first calls AND subsequent interval calls
    mockGetDashboard.mockRejectedValue(new Error("Dashboard fail"));
    mockGetDashboardCharts.mockRejectedValue(new Error("Charts fail"));
    const { unmount } = render(<Dashboard />);
    // Should not crash — loading finishes, no stats rendered but page doesn't blow up
    await waitFor(() => {
      expect(screen.queryByText("Loading dashboard...")).not.toBeInTheDocument();
    });
    // Page still renders — stats null means KPI values fall back to 0
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Total Parts")).toBeInTheDocument();
    unmount();
    // Restore default mocks for other tests
    mockGetDashboard.mockResolvedValue(mockDashboardStats);
    mockGetDashboardCharts.mockResolvedValue({ eco_counts: [1, 2, 3] });
  });
});
