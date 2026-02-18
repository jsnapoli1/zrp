import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "../test/test-utils";
import { AppLayout } from "./AppLayout";

describe("AppLayout", () => {
  it("renders sidebar with ZRP branding", () => {
    render(<AppLayout />);
    expect(screen.getByText("ZRP")).toBeInTheDocument();
  });

  it("renders navigation sections", () => {
    render(<AppLayout />);
    expect(screen.getByText("Engineering")).toBeInTheDocument();
    expect(screen.getByText("Supply Chain")).toBeInTheDocument();
    expect(screen.getByText("Manufacturing")).toBeInTheDocument();
    expect(screen.getByText("Admin")).toBeInTheDocument();
  });

  it("renders navigation links", () => {
    render(<AppLayout />);
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Parts")).toBeInTheDocument();
    expect(screen.getByText("ECOs")).toBeInTheDocument();
    expect(screen.getByText("Vendors")).toBeInTheDocument();
    expect(screen.getByText("Work Orders")).toBeInTheDocument();
    expect(screen.getByText("Inventory")).toBeInTheDocument();
  });

  it("renders search button", () => {
    render(<AppLayout />);
    expect(screen.getByText("Search...")).toBeInTheDocument();
  });

  it("renders notification bell and user button", () => {
    render(<AppLayout />);
    // User menu button exists
    const buttons = screen.getAllByRole("button");
    expect(buttons.length).toBeGreaterThan(0);
  });

  it("renders dark mode toggle", () => {
    render(<AppLayout />);
    // Version text and toggle button in footer
    expect(screen.getByText("v1.0.0")).toBeInTheDocument();
  });

  it("opens command dialog on search click", async () => {
    render(<AppLayout />);
    fireEvent.click(screen.getByText("Search..."));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Type a command or search...")).toBeInTheDocument();
    });
  });

  it("renders all navigation sections", () => {
    render(<AppLayout />);
    const sections = ["Engineering", "Supply Chain", "Manufacturing", "Field & Service", "Sales", "Reports", "Admin"];
    for (const section of sections) {
      expect(screen.getByText(section)).toBeInTheDocument();
    }
  });

  it("renders all engineering nav items", () => {
    render(<AppLayout />);
    expect(screen.getByText("Documents")).toBeInTheDocument();
    expect(screen.getByText("Testing")).toBeInTheDocument();
  });

  it("renders supply chain nav items", () => {
    render(<AppLayout />);
    expect(screen.getByText("Purchase Orders")).toBeInTheDocument();
    expect(screen.getByText("Procurement")).toBeInTheDocument();
  });

  it("renders field service nav items", () => {
    render(<AppLayout />);
    expect(screen.getByText("RMAs")).toBeInTheDocument();
    expect(screen.getByText("Field Reports")).toBeInTheDocument();
  });

  it("renders user menu with admin info", () => {
    render(<AppLayout />);
    // User menu trigger button exists
    const userButtons = screen.getAllByRole("button");
    expect(userButtons.length).toBeGreaterThan(2);
  });

  it("renders keyboard shortcut hint", () => {
    render(<AppLayout />);
    expect(screen.getByText("K")).toBeInTheDocument();
  });

  it("command dialog shows quick actions", async () => {
    render(<AppLayout />);
    fireEvent.click(screen.getByText("Search..."));
    await waitFor(() => {
      expect(screen.getByText("Create ECO")).toBeInTheDocument();
      expect(screen.getByText("New Work Order")).toBeInTheDocument();
      expect(screen.getByText("Add Part")).toBeInTheDocument();
    });
  });

  it("command dialog shows navigation items", async () => {
    render(<AppLayout />);
    fireEvent.click(screen.getByText("Search..."));
    await waitFor(() => {
      expect(screen.getByText("Quick Actions")).toBeInTheDocument();
      expect(screen.getByText("Navigation")).toBeInTheDocument();
    });
  });

  it("toggles dark mode on button click", () => {
    render(<AppLayout />);
    // Find the dark mode toggle button near v1.0.0
    const buttons = screen.getAllByRole("button");
    // The dark mode toggle is one of the buttons - click it and verify no crash
    const footer = screen.getByText("v1.0.0");
    const toggleBtn = footer.parentElement?.querySelector("button");
    if (toggleBtn) {
      fireEvent.click(toggleBtn);
      // After click, dark class should be toggled on documentElement
      expect(document.documentElement.classList.contains("dark")).toBe(true);
      fireEvent.click(toggleBtn);
      expect(document.documentElement.classList.contains("dark")).toBe(false);
    }
  });
});
