import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockCAPAs, mockCAPADashboard } from "../test/mocks";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate };
});

const mockGetCAPAs = vi.fn().mockResolvedValue(mockCAPAs);
const mockCreateCAPA = vi.fn().mockResolvedValue(mockCAPAs[0]);
const mockGetCAPADashboard = vi.fn().mockResolvedValue(mockCAPADashboard);

vi.mock("../lib/api", () => ({
  api: {
    getCAPAs: (...args: any[]) => mockGetCAPAs(...args),
    createCAPA: (...args: any[]) => mockCreateCAPA(...args),
    getCAPADashboard: (...args: any[]) => mockGetCAPADashboard(...args),
  },
}));

import CAPAs from "./CAPAs";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetCAPAs.mockResolvedValue(mockCAPAs);
  mockGetCAPADashboard.mockResolvedValue(mockCAPADashboard);
});

describe("CAPAs", () => {
  it("renders loading state", () => {
    render(<CAPAs />);
    expect(document.querySelector(".animate-spin")).toBeTruthy();
  });

  it("renders CAPA list", async () => {
    render(<CAPAs />);
    await waitFor(() => {
      expect(screen.getByText("Fix solder defect")).toBeInTheDocument();
      expect(screen.getByText("Prevent shipping damage")).toBeInTheDocument();
    });
  });

  it("displays dashboard stats", async () => {
    render(<CAPAs />);
    await waitFor(() => {
      expect(screen.getByText("Open CAPAs")).toBeInTheDocument();
      expect(screen.getByText("Overdue")).toBeInTheDocument();
      expect(screen.getByText("By Owner")).toBeInTheDocument();
    });
  });

  it("displays CAPA type badges", async () => {
    render(<CAPAs />);
    await waitFor(() => {
      expect(screen.getByText("corrective")).toBeInTheDocument();
      expect(screen.getByText("preventive")).toBeInTheDocument();
    });
  });

  it("displays CAPA status badges", async () => {
    render(<CAPAs />);
    await waitFor(() => {
      expect(screen.getAllByText("open").length).toBeGreaterThan(0);
      expect(screen.getAllByText("in-progress").length).toBeGreaterThan(0);
    });
  });

  it("shows empty state", async () => {
    mockGetCAPAs.mockResolvedValue([]);
    render(<CAPAs />);
    await waitFor(() => {
      expect(screen.getByText("No CAPAs found")).toBeInTheDocument();
    });
  });

  it("opens create dialog", async () => {
    render(<CAPAs />);
    await waitFor(() => screen.getByText("New CAPA"));
    fireEvent.click(screen.getByText("New CAPA"));
    await waitFor(() => {
      expect(screen.getAllByText("Create CAPA").length).toBeGreaterThan(0);
    });
  });

  it("navigates to CAPA detail on row click", async () => {
    render(<CAPAs />);
    await waitFor(() => screen.getByText("CAPA-2024-001"));
    fireEvent.click(screen.getByText("CAPA-2024-001"));
    expect(mockNavigate).toHaveBeenCalledWith("/capas/CAPA-2024-001");
  });

  it("shows linked NCR/RMA info", async () => {
    render(<CAPAs />);
    await waitFor(() => {
      expect(screen.getByText(/NCR: NCR-001/)).toBeInTheDocument();
      expect(screen.getByText(/RMA: RMA-001/)).toBeInTheDocument();
    });
  });

  it("shows owner info in table and dashboard", async () => {
    render(<CAPAs />);
    await waitFor(() => {
      expect(screen.getAllByText("engineer1").length).toBeGreaterThan(0);
      expect(screen.getAllByText("logistics1").length).toBeGreaterThan(0);
    });
  });
});
