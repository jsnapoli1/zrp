import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";

// Create events for the current month so they show up
const now = new Date();
const year = now.getFullYear();
const month = now.getMonth() + 1;
const dateStr = `${year}-${String(month).padStart(2, "0")}-15`;
const dateStr2 = `${year}-${String(month).padStart(2, "0")}-20`;

const mockCalendarEvents = [
  { date: dateStr, type: "workorder", id: "WO-001", title: "Build Widget Batch", color: "blue" },
  { date: dateStr, type: "po", id: "PO-001", title: "PO-001 Due", color: "green" },
  { date: dateStr2, type: "quote", id: "Q-001", title: "Quote Expires", color: "purple" },
];

const mockGetCalendarEvents = vi.fn().mockResolvedValue(mockCalendarEvents);

vi.mock("../lib/api", () => ({
  api: {
    getCalendarEvents: (...args: any[]) => mockGetCalendarEvents(...args),
  },
}));

import Calendar from "./Calendar";

beforeEach(() => vi.clearAllMocks());

const months = [
  'January', 'February', 'March', 'April', 'May', 'June',
  'July', 'August', 'September', 'October', 'November', 'December'
];

describe("Calendar", () => {
  it("renders loading state", () => {
    render(<Calendar />);
    expect(screen.getByText("Loading calendar...")).toBeInTheDocument();
  });

  it("renders calendar title and description", async () => {
    render(<Calendar />);
    await waitFor(() => {
      expect(screen.getByText("Calendar")).toBeInTheDocument();
    });
    expect(screen.getByText("View due dates for work orders, purchase orders, and quotes.")).toBeInTheDocument();
  });

  it("shows current month and year", async () => {
    render(<Calendar />);
    const expectedTitle = `${months[now.getMonth()]} ${year}`;
    await waitFor(() => {
      expect(screen.getByText(expectedTitle)).toBeInTheDocument();
    });
  });

  it("calls getCalendarEvents with current year and month", async () => {
    render(<Calendar />);
    await waitFor(() => {
      expect(mockGetCalendarEvents).toHaveBeenCalledWith(year, month);
    });
  });

  it("shows day-of-week headers", async () => {
    render(<Calendar />);
    await waitFor(() => {
      expect(screen.getByText("Sun")).toBeInTheDocument();
      expect(screen.getByText("Mon")).toBeInTheDocument();
      expect(screen.getByText("Tue")).toBeInTheDocument();
      expect(screen.getByText("Wed")).toBeInTheDocument();
      expect(screen.getByText("Thu")).toBeInTheDocument();
      expect(screen.getByText("Fri")).toBeInTheDocument();
      expect(screen.getByText("Sat")).toBeInTheDocument();
    });
  });

  it("renders day numbers in the grid", async () => {
    render(<Calendar />);
    await waitFor(() => {
      expect(screen.getByText("1")).toBeInTheDocument();
      expect(screen.getByText("15")).toBeInTheDocument();
    });
  });

  it("shows 'Select a date' initially", async () => {
    render(<Calendar />);
    await waitFor(() => {
      expect(screen.getByText("Select a date")).toBeInTheDocument();
    });
    expect(screen.getByText("Click on a date to view events")).toBeInTheDocument();
  });

  it("navigates to next month when next button clicked", async () => {
    render(<Calendar />);
    await waitFor(() => {
      expect(screen.getByText(`${months[now.getMonth()]} ${year}`)).toBeInTheDocument();
    });
    // Find the next button (second navigation button)
    const navButtons = screen.getAllByRole("button").filter(btn => btn.textContent === "");
    // Click next (the second chevron button)
    const buttons = screen.getAllByRole("button");
    // The next month button is the one after prev
    let nextBtn: HTMLElement | null = null;
    for (const btn of buttons) {
      if (btn.querySelector('[class*="lucide"]') || btn.innerHTML.includes("chevron")) {
        // collect navigation buttons
      }
    }
    // Simpler: just find buttons that are outline+sm size in the header
    // Use the second small outline button
    const smallButtons = buttons.filter(b => b.className.includes("outline") && b.className.includes("sm"));
    if (smallButtons.length >= 2) {
      fireEvent.click(smallButtons[1]); // next button
    }
    
    const nextMonth = new Date(year, now.getMonth() + 1, 1);
    const expectedMonth = months[nextMonth.getMonth()];
    const expectedYear = nextMonth.getFullYear();
    await waitFor(() => {
      expect(mockGetCalendarEvents).toHaveBeenCalledWith(expectedYear, nextMonth.getMonth() + 1);
    });
  });

  it("navigates to previous month when prev button clicked", async () => {
    render(<Calendar />);
    await waitFor(() => {
      expect(screen.getByText(`${months[now.getMonth()]} ${year}`)).toBeInTheDocument();
    });
    const buttons = screen.getAllByRole("button");
    const smallButtons = buttons.filter(b => b.className.includes("outline") && b.className.includes("sm"));
    if (smallButtons.length >= 1) {
      fireEvent.click(smallButtons[0]); // prev button
    }
    
    const prevMonth = new Date(year, now.getMonth() - 1, 1);
    await waitFor(() => {
      expect(mockGetCalendarEvents).toHaveBeenCalledWith(prevMonth.getFullYear(), prevMonth.getMonth() + 1);
    });
  });

  it("shows events when a date with events is clicked", async () => {
    render(<Calendar />);
    await waitFor(() => {
      expect(screen.getByText("15")).toBeInTheDocument();
    });
    // Click on day 15
    fireEvent.click(screen.getByText("15"));
    await waitFor(() => {
      expect(screen.getByText("Build Widget Batch")).toBeInTheDocument();
      expect(screen.getByText("PO-001 Due")).toBeInTheDocument();
    });
  });

  it("shows event type badges for selected date", async () => {
    render(<Calendar />);
    await waitFor(() => screen.getByText("15"));
    fireEvent.click(screen.getByText("15"));
    await waitFor(() => {
      // Legend already shows these, so there should be more after clicking a date with events
      const woLabels = screen.getAllByText("Work Order");
      expect(woLabels.length).toBeGreaterThanOrEqual(2); // 1 legend + 1 event badge
      const poLabels = screen.getAllByText("Purchase Order");
      expect(poLabels.length).toBeGreaterThanOrEqual(2);
    });
  });

  it("shows 'No events on this date' for empty dates", async () => {
    render(<Calendar />);
    await waitFor(() => screen.getByText("1"));
    // Click day 1 which has no events (unless it happens to be 15 or 20)
    fireEvent.click(screen.getByText("2"));
    await waitFor(() => {
      expect(screen.getByText("No events on this date")).toBeInTheDocument();
    });
  });

  it("shows legend with event types", async () => {
    render(<Calendar />);
    await waitFor(() => {
      expect(screen.getByText("Legend")).toBeInTheDocument();
    });
    expect(screen.getByText("Work Order")).toBeInTheDocument();
    expect(screen.getByText("Purchase Order")).toBeInTheDocument();
    expect(screen.getByText("Quote")).toBeInTheDocument();
  });

  it("handles API error gracefully", async () => {
    mockGetCalendarEvents.mockRejectedValueOnce(new Error("fail"));
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    render(<Calendar />);
    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalled();
    });
    consoleSpy.mockRestore();
  });
});
