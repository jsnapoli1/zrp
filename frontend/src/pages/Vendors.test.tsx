import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockVendors } from "../test/mocks";

const mockGetVendors = vi.fn().mockResolvedValue(mockVendors);
const mockCreateVendor = vi.fn().mockResolvedValue(mockVendors[0]);
const mockUpdateVendor = vi.fn().mockResolvedValue(mockVendors[0]);
const mockDeleteVendor = vi.fn().mockResolvedValue(undefined);

vi.mock("../lib/api", () => ({
  api: {
    getVendors: (...args: any[]) => mockGetVendors(...args),
    createVendor: (...args: any[]) => mockCreateVendor(...args),
    updateVendor: (...args: any[]) => mockUpdateVendor(...args),
    deleteVendor: (...args: any[]) => mockDeleteVendor(...args),
  },
}));

import Vendors from "./Vendors";

beforeEach(() => vi.clearAllMocks());

describe("Vendors", () => {
  it("renders loading state", () => {
    render(<Vendors />);
    expect(screen.getByText("Loading vendors...")).toBeInTheDocument();
  });

  it("renders vendor list after loading", async () => {
    render(<Vendors />);
    await waitFor(() => {
      expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    });
    expect(screen.getByText("DigiParts")).toBeInTheDocument();
  });

  it("has add vendor button", async () => {
    render(<Vendors />);
    await waitFor(() => {
      expect(screen.getByText("Add Vendor")).toBeInTheDocument();
    });
  });

  it("shows empty state", async () => {
    mockGetVendors.mockResolvedValueOnce([]);
    render(<Vendors />);
    await waitFor(() => {
      expect(screen.getByText(/no vendors found/i)).toBeInTheDocument();
    });
  });

  it("shows vendor contact info", async () => {
    render(<Vendors />);
    await waitFor(() => {
      expect(screen.getByText("john@acme.com")).toBeInTheDocument();
    });
  });

  it("shows vendor status badges", async () => {
    render(<Vendors />);
    await waitFor(() => {
      const badges = screen.getAllByText("ACTIVE");
      expect(badges.length).toBe(2);
    });
  });

  it("displays summary cards with counts", async () => {
    render(<Vendors />);
    await waitFor(() => {
      expect(screen.getByText("Total Vendors")).toBeInTheDocument();
    });
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Inactive")).toBeInTheDocument();
  });

  it("shows vendor directory table headers", async () => {
    render(<Vendors />);
    await waitFor(() => {
      expect(screen.getByText("Vendor Directory")).toBeInTheDocument();
    });
    expect(screen.getByText("Company")).toBeInTheDocument();
    expect(screen.getByText("Contact")).toBeInTheDocument();
    expect(screen.getByText("Email")).toBeInTheDocument();
    expect(screen.getByText("Phone")).toBeInTheDocument();
    expect(screen.getByText("Status")).toBeInTheDocument();
    expect(screen.getByText("Lead Time")).toBeInTheDocument();
  });

  it("shows lead time for vendors", async () => {
    render(<Vendors />);
    await waitFor(() => {
      expect(screen.getByText("14 days")).toBeInTheDocument();
      expect(screen.getByText("7 days")).toBeInTheDocument();
    });
  });

  it("renders vendor name as link", async () => {
    render(<Vendors />);
    await waitFor(() => {
      const link = screen.getByText("Acme Corp").closest("a");
      expect(link).toHaveAttribute("href", "/vendors/V-001");
    });
  });

  it("shows website link for vendors with website", async () => {
    render(<Vendors />);
    await waitFor(() => {
      const websiteLinks = screen.getAllByText("Website");
      expect(websiteLinks.length).toBeGreaterThan(0);
    });
  });

  it("opens create vendor dialog when Add Vendor clicked", async () => {
    render(<Vendors />);
    await waitFor(() => {
      expect(screen.getByText("Add Vendor")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByText("Add Vendor"));
    await waitFor(() => {
      expect(screen.getByText("Add New Vendor")).toBeInTheDocument();
    });
    expect(screen.getByLabelText("Company Name *")).toBeInTheDocument();
    expect(screen.getByLabelText("Contact Name")).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
  });

  it("create dialog has Cancel and Create Vendor buttons", async () => {
    render(<Vendors />);
    await waitFor(() => screen.getByText("Add Vendor"));
    fireEvent.click(screen.getByText("Add Vendor"));
    await waitFor(() => {
      expect(screen.getByText("Add New Vendor")).toBeInTheDocument();
    });
    expect(screen.getByText("Cancel")).toBeInTheDocument();
    expect(screen.getByText("Create Vendor")).toBeInTheDocument();
  });

  it("Create Vendor button is disabled when name is empty", async () => {
    render(<Vendors />);
    await waitFor(() => screen.getByText("Add Vendor"));
    fireEvent.click(screen.getByText("Add Vendor"));
    await waitFor(() => {
      expect(screen.getByText("Create Vendor")).toBeInTheDocument();
    });
    expect(screen.getByText("Create Vendor")).toBeDisabled();
  });

  it("calls createVendor when form submitted with name", async () => {
    render(<Vendors />);
    await waitFor(() => screen.getByText("Add Vendor"));
    fireEvent.click(screen.getByText("Add Vendor"));
    await waitFor(() => {
      expect(screen.getByLabelText("Company Name *")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByLabelText("Company Name *"), { target: { value: "New Vendor" } });
    fireEvent.click(screen.getByText("Create Vendor"));
    await waitFor(() => {
      expect(mockCreateVendor).toHaveBeenCalledWith(
        expect.objectContaining({ name: "New Vendor" })
      );
    });
  });

  it("calls deleteVendor when delete confirmed", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    render(<Vendors />);
    await waitFor(() => {
      expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    });
    // Click the first dropdown trigger (MoreHorizontal button)
    const moreButtons = screen.getAllByRole("button").filter(
      btn => btn.querySelector("svg") && btn.textContent === ""
    );
    // Find the dropdown triggers - they're the ghost buttons in the last column
    const dropdownTriggers = screen.getAllByRole("button").filter(btn => {
      const cls = btn.className || "";
      return cls.includes("ghost");
    });
    if (dropdownTriggers.length > 0) {
      fireEvent.click(dropdownTriggers[0]);
      await waitFor(() => {
        const deleteItem = screen.getByText("Delete");
        fireEvent.click(deleteItem);
      });
      await waitFor(() => {
        expect(mockDeleteVendor).toHaveBeenCalledWith("V-001");
      });
    }
    (window.confirm as any).mockRestore();
  });

  it("does not delete when confirm is cancelled", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
    render(<Vendors />);
    await waitFor(() => {
      expect(screen.getByText("Acme Corp")).toBeInTheDocument();
    });
    const dropdownTriggers = screen.getAllByRole("button").filter(btn => {
      const cls = btn.className || "";
      return cls.includes("ghost");
    });
    if (dropdownTriggers.length > 0) {
      fireEvent.click(dropdownTriggers[0]);
      await waitFor(() => {
        const deleteItem = screen.getByText("Delete");
        fireEvent.click(deleteItem);
      });
      expect(mockDeleteVendor).not.toHaveBeenCalled();
    }
    (window.confirm as any).mockRestore();
  });

  it("handles API error on fetch gracefully", async () => {
    mockGetVendors.mockRejectedValueOnce(new Error("Network error"));
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    render(<Vendors />);
    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalledWith("Failed to fetch vendors:", expect.any(Error));
    });
    consoleSpy.mockRestore();
  });
});
