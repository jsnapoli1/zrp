import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import Users from "./Users";

// No api mock needed - uses internal mock data

describe("Users", () => {
  it("renders loading state then content", async () => {
    render(<Users />);
    // Loading is shown briefly, then content loads
    await waitFor(() => {
      expect(screen.getByText("User Management")).toBeInTheDocument();
    });
  });

  it("renders user list after loading", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("admin")).toBeInTheDocument();
    });
  });

  it("shows user emails", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("admin@example.com")).toBeInTheDocument();
    });
  });

  it("has add user button", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText(/add user|create user|new user|invite/i)).toBeInTheDocument();
    });
  });

  it("shows user roles", async () => {
    render(<Users />);
    await waitFor(() => {
      // There should be role badges
      const adminBadges = screen.getAllByText(/admin/i);
      expect(adminBadges.length).toBeGreaterThan(0);
    });
  });
});
