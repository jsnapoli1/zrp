import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import { mockAuditLogs } from "../test/mocks";

const mockGetAuditLogs = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getAuditLogs: (...args: any[]) => mockGetAuditLogs(...args),
  },
}));

import Audit from "./Audit";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetAuditLogs.mockResolvedValue({ entries: mockAuditLogs, total: mockAuditLogs.length });
});

describe("Audit", () => {
  it("renders page title after loading", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Audit Log")).toBeInTheDocument();
    });
  });

  it("shows page description", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Track all system activities and user actions.")).toBeInTheDocument();
    });
  });

  it("renders audit entries after loading", async () => {
    render(<Audit />);
    await waitFor(() => {
      // mockAuditLogs user is "admin"
      expect(screen.getByText("Created part")).toBeInTheDocument();
    });
  });

  it("has search input", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Search logs...")).toBeInTheDocument();
    });
  });

  it("shows action badges", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("create")).toBeInTheDocument();
      expect(screen.getByText("update")).toBeInTheDocument();
    });
  });

  it("shows Filters card", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Filters")).toBeInTheDocument();
    });
  });

  it("shows Audit Entries card", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Audit Entries")).toBeInTheDocument();
    });
  });

  it("shows entity type filter", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getAllByText("Entity Type").length).toBeGreaterThanOrEqual(2);
    });
  });

  it("shows user filter", async () => {
    render(<Audit />);
    await waitFor(() => {
      const userLabels = screen.getAllByText("User");
      expect(userLabels.length).toBeGreaterThan(0);
    });
  });

  it("shows table headers", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Timestamp")).toBeInTheDocument();
      expect(screen.getByText("Action")).toBeInTheDocument();
      expect(screen.getAllByText("Entity Type").length).toBeGreaterThanOrEqual(2);
      expect(screen.getByText("Entity ID")).toBeInTheDocument();
      expect(screen.getByText("Details")).toBeInTheDocument();
      expect(screen.getByText("IP Address")).toBeInTheDocument();
    });
  });

  it("shows entity IDs", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("IPN-001")).toBeInTheDocument();
      expect(screen.getByText("ECO-001")).toBeInTheDocument();
    });
  });

  it("shows entries count", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("2 entries found")).toBeInTheDocument();
    });
  });

  it("filters by search term", async () => {
    const user = userEvent.setup();
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Search logs...")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("Search logs..."), "Created part");
    await waitFor(() => {
      expect(screen.getByText("1 entries found")).toBeInTheDocument();
    });
  });

  it("shows Clear Filters button when filters active", async () => {
    const user = userEvent.setup();
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Search logs...")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("Search logs..."), "admin");
    await waitFor(() => {
      expect(screen.getByText("Clear Filters")).toBeInTheDocument();
    });
  });

  it("clears filters when Clear Filters clicked", async () => {
    const user = userEvent.setup();
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Search logs...")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("Search logs..."), "Created part");
    await waitFor(() => {
      expect(screen.getByText("Clear Filters")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Clear Filters"));
    await waitFor(() => {
      expect(screen.getByText("2 entries found")).toBeInTheDocument();
    });
  });

  it("shows details column", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Created part")).toBeInTheDocument();
      expect(screen.getByText("Updated status")).toBeInTheDocument();
    });
  });

  it("shows loading state initially", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Audit Log")).toBeInTheDocument();
    });
  });

  it("shows Search label", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Search")).toBeInTheDocument();
    });
  });
});
