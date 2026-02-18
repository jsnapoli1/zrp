import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import { mockTestRecords } from "../test/mocks";

const mockGetTestRecords = vi.fn().mockResolvedValue(mockTestRecords);
const mockCreateTestRecord = vi.fn().mockResolvedValue(mockTestRecords[0]);

vi.mock("../lib/api", () => ({
  api: {
    getTestRecords: (...args: any[]) => mockGetTestRecords(...args),
    createTestRecord: (...args: any[]) => mockCreateTestRecord(...args),
  },
}));

import Testing from "./Testing";

beforeEach(() => vi.clearAllMocks());

describe("Testing", () => {
  it("renders loading state", () => {
    render(<Testing />);
    expect(screen.getByText("Loading test records...")).toBeInTheDocument();
  });

  it("renders test records after loading", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText("SN-100")).toBeInTheDocument();
    });
    expect(screen.getByText("SN-101")).toBeInTheDocument();
  });

  it("shows test results", async () => {
    render(<Testing />);
    await waitFor(() => {
      // Internal mock data uses "pass" results 
      expect(screen.getByText("Pass")).toBeInTheDocument();
    });
  });

  it("has create test record button", async () => {
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText(/new test|create test|record test/i)).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    mockGetTestRecords.mockResolvedValueOnce([]);
    render(<Testing />);
    await waitFor(() => {
      expect(screen.getByText(/no test records found/i)).toBeInTheDocument();
    });
  });
});
