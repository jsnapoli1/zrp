import { describe, it, expect } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import APIKeys from "./APIKeys";

describe("APIKeys", () => {
  it("renders page title after loading", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      const titles = screen.getAllByText("API Keys");
      expect(titles.length).toBeGreaterThan(0);
    });
  });

  it("renders API key list after loading", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Production Integration")).toBeInTheDocument();
    });
  });

  it("shows key status badges", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      const actives = screen.getAllByText(/active/i);
      expect(actives.length).toBeGreaterThan(0);
    });
  });

  it("has generate key button", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      const btns = screen.getAllByText(/generate/i);
      expect(btns.length).toBeGreaterThan(0);
    });
  });

  it("shows summary cards", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Total Keys")).toBeInTheDocument();
    });
  });
});
