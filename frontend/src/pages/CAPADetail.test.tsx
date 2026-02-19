import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockCAPAs } from "../test/mocks";

const mockNavigate = vi.fn();
vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual("react-router-dom");
  return { ...actual, useNavigate: () => mockNavigate, useParams: () => ({ id: "CAPA-2024-001" }) };
});

const mockGetCAPA = vi.fn().mockResolvedValue(mockCAPAs[0]);
const mockUpdateCAPA = vi.fn().mockResolvedValue(mockCAPAs[0]);

vi.mock("../lib/api", () => ({
  api: {
    getCAPA: (...args: any[]) => mockGetCAPA(...args),
    updateCAPA: (...args: any[]) => mockUpdateCAPA(...args),
  },
}));

import CAPADetail from "./CAPADetail";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetCAPA.mockResolvedValue(mockCAPAs[0]);
  mockUpdateCAPA.mockResolvedValue(mockCAPAs[0]);
});

describe("CAPADetail", () => {
  it("renders loading state", () => {
    render(<CAPADetail />);
    expect(document.querySelector(".animate-spin")).toBeTruthy();
  });

  it("renders CAPA details", async () => {
    render(<CAPADetail />);
    await waitFor(() => {
      expect(screen.getByText(/CAPA-2024-001/)).toBeInTheDocument();
      expect(screen.getAllByText("corrective").length).toBeGreaterThan(0);
    });
  });

  it("shows status workflow section", async () => {
    render(<CAPADetail />);
    await waitFor(() => {
      expect(screen.getByText("Status Workflow")).toBeInTheDocument();
      expect(screen.getAllByText("open").length).toBeGreaterThan(0);
    });
  });

  it("shows approval workflow", async () => {
    render(<CAPADetail />);
    await waitFor(() => {
      expect(screen.getByText("Approval Workflow")).toBeInTheDocument();
      expect(screen.getByText("Approve as QE")).toBeInTheDocument();
      expect(screen.getByText("Approve as Manager")).toBeInTheDocument();
    });
  });

  it("shows root cause and action plan", async () => {
    render(<CAPADetail />);
    await waitFor(() => {
      expect(screen.getByText("Root Cause")).toBeInTheDocument();
      expect(screen.getByText("Insufficient flux")).toBeInTheDocument();
      expect(screen.getByText("Action Plan")).toBeInTheDocument();
      expect(screen.getByText("Update solder profile")).toBeInTheDocument();
    });
  });

  it("shows effectiveness verification section", async () => {
    render(<CAPADetail />);
    await waitFor(() => {
      expect(screen.getByText("Effectiveness Verification")).toBeInTheDocument();
    });
  });

  it("enters edit mode", async () => {
    render(<CAPADetail />);
    await waitFor(() => screen.getByText("Edit"));
    fireEvent.click(screen.getByText("Edit"));
    await waitFor(() => {
      expect(screen.getByText("Save")).toBeInTheDocument();
    });
  });

  it("shows not found for missing CAPA", async () => {
    mockGetCAPA.mockRejectedValue(new Error("not found"));
    render(<CAPADetail />);
    await waitFor(() => {
      expect(screen.getByText("CAPA not found")).toBeInTheDocument();
    });
  });

  it("displays error on failed update", async () => {
    mockUpdateCAPA.mockRejectedValue(new Error("effectiveness check required before closing"));
    render(<CAPADetail />);
    await waitFor(() => screen.getByText("Edit"));
    fireEvent.click(screen.getByText("Edit"));
    await waitFor(() => screen.getByText("Save"));
    fireEvent.click(screen.getByText("Save"));
    await waitFor(() => {
      expect(screen.getByText(/effectiveness check required/)).toBeInTheDocument();
    });
  });
});
