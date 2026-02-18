import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import Audit from "./Audit";

// No api mock needed - uses internal mock data

describe("Audit", () => {
  it("renders page title after loading", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByText("Audit Log")).toBeInTheDocument();
    });
  });

  it("renders audit entries after loading", async () => {
    render(<Audit />);
    await waitFor(() => {
      // Mock data uses john.doe@example.com etc
      expect(screen.getByText(/john\.doe/i)).toBeInTheDocument();
    });
  });

  it("has search input", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText(/search/i)).toBeInTheDocument();
    });
  });

  it("shows entity types and actions", async () => {
    render(<Audit />);
    await waitFor(() => {
      expect(screen.getAllByText(/part/i).length).toBeGreaterThan(0);
    });
  });
});
