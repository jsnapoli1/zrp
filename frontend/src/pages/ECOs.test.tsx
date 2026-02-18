import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockECOs } from "../test/mocks";

const mockGetECOs = vi.fn().mockResolvedValue(mockECOs);
const mockCreateECO = vi.fn().mockResolvedValue(mockECOs[0]);

vi.mock("../lib/api", () => ({
  api: {
    getECOs: (...args: any[]) => mockGetECOs(...args),
    createECO: (...args: any[]) => mockCreateECO(...args),
  },
}));

import ECOs from "./ECOs";

beforeEach(() => vi.clearAllMocks());

describe("ECOs", () => {
  it("renders loading state", () => {
    render(<ECOs />);
    expect(screen.getByText(/engineering change orders|ecos/i)).toBeInTheDocument();
  });

  it("renders ECO list after loading", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    expect(screen.getByText("ECO-002")).toBeInTheDocument();
    expect(screen.getByText("Update resistor spec")).toBeInTheDocument();
  });

  it("shows status badges", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("Draft")).toBeInTheDocument();
      expect(screen.getByText("Approved")).toBeInTheDocument();
    });
  });

  it("has tabs for filtering by status", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    // Tabs should exist
    expect(screen.getByRole("tablist")).toBeInTheDocument();
  });

  it("has create ECO button", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText(/create eco|new eco/i)).toBeInTheDocument();
    });
  });

  it("opens create dialog", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
    const btn = screen.getByText(/create eco|new eco/i);
    fireEvent.click(btn);
    await waitFor(() => {
      expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    mockGetECOs.mockResolvedValueOnce([]);
    render(<ECOs />);
    await waitFor(() => {
      expect(screen.getByText(/no ecos found|no engineering change/i)).toBeInTheDocument();
    });
  });

  it("calls getECOs on mount", async () => {
    render(<ECOs />);
    await waitFor(() => {
      expect(mockGetECOs).toHaveBeenCalled();
    });
  });
});
