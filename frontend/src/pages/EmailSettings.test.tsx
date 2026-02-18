import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import EmailSettings from "./EmailSettings";

// Check if this uses api or internal mock data
describe("EmailSettings", () => {
  it("renders page title after loading", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Email Settings")).toBeInTheDocument();
    });
  });

  it("shows SMTP configuration fields", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getAllByText(/smtp/i).length).toBeGreaterThan(0);
    });
  });

  it("has save button", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText(/save/i)).toBeInTheDocument();
    });
  });

  it("has test email button", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getAllByText(/send test|test email/i).length).toBeGreaterThan(0);
    });
  });
});
