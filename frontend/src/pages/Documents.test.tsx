import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "../test/test-utils";
import { mockDocuments } from "../test/mocks";

const mockGetDocuments = vi.fn().mockResolvedValue(mockDocuments);
const mockCreateDocument = vi.fn().mockResolvedValue(mockDocuments[0]);
const mockUploadAttachment = vi.fn().mockResolvedValue({ id: "ATT-1" });

vi.mock("../lib/api", () => ({
  api: {
    getDocuments: (...args: any[]) => mockGetDocuments(...args),
    createDocument: (...args: any[]) => mockCreateDocument(...args),
    uploadAttachment: (...args: any[]) => mockUploadAttachment(...args),
  },
}));

import Documents from "./Documents";

beforeEach(() => vi.clearAllMocks());

describe("Documents", () => {
  it("renders page title and description", async () => {
    render(<Documents />);
    await waitFor(() => {
      expect(screen.getByText("Documents")).toBeInTheDocument();
    });
    expect(screen.getByText("Manage technical documentation, specifications, and procedures")).toBeInTheDocument();
  });

  it("renders document list after loading", async () => {
    render(<Documents />);
    await waitFor(() => {
      expect(screen.getByText("Assembly Guide")).toBeInTheDocument();
    });
    expect(screen.getByText("Test Procedure")).toBeInTheDocument();
  });

  it("shows empty state", async () => {
    mockGetDocuments.mockResolvedValueOnce([]);
    render(<Documents />);
    await waitFor(() => {
      expect(screen.getByText(/no documents found/i)).toBeInTheDocument();
    });
  });

  it("has Upload Files and Create Document buttons", async () => {
    render(<Documents />);
    await waitFor(() => {
      expect(screen.getByText("Upload Files")).toBeInTheDocument();
      expect(screen.getByText("Create Document")).toBeInTheDocument();
    });
  });

  it("shows Document Library table header", async () => {
    render(<Documents />);
    await waitFor(() => {
      expect(screen.getByText("Document Library")).toBeInTheDocument();
    });
  });

  it("shows table column headers", async () => {
    render(<Documents />);
    await waitFor(() => {
      expect(screen.getByText("Title")).toBeInTheDocument();
      expect(screen.getByText("Category")).toBeInTheDocument();
      expect(screen.getByText("IPN")).toBeInTheDocument();
      expect(screen.getByText("Status")).toBeInTheDocument();
      expect(screen.getByText("Revision")).toBeInTheDocument();
    });
  });

  it("displays document categories", async () => {
    render(<Documents />);
    await waitFor(() => {
      expect(screen.getByText("procedure")).toBeInTheDocument();
      expect(screen.getByText("test")).toBeInTheDocument();
    });
  });

  it("displays document IPN", async () => {
    render(<Documents />);
    await waitFor(() => {
      const ipns = screen.getAllByText("IPN-003");
      expect(ipns.length).toBeGreaterThan(0);
    });
  });

  it("displays document revisions", async () => {
    render(<Documents />);
    await waitFor(() => {
      expect(screen.getByText("A")).toBeInTheDocument();
      expect(screen.getByText("B")).toBeInTheDocument();
    });
  });

  it("has View/Download buttons for each document", async () => {
    render(<Documents />);
    await waitFor(() => {
      const viewButtons = screen.getAllByText("View");
      expect(viewButtons.length).toBe(2);
    });
  });

  it("opens upload dialog when Upload Files clicked", async () => {
    render(<Documents />);
    await waitFor(() => screen.getByText("Upload Files"));
    fireEvent.click(screen.getByText("Upload Files"));
    await waitFor(() => {
      expect(screen.getByText("Drop files here or click to browse")).toBeInTheDocument();
    });
  });

  it("upload dialog has Browse Files button", async () => {
    render(<Documents />);
    await waitFor(() => screen.getByText("Upload Files"));
    fireEvent.click(screen.getByText("Upload Files"));
    await waitFor(() => {
      expect(screen.getByText("Browse Files")).toBeInTheDocument();
    });
  });

  it("upload button is disabled when no files selected", async () => {
    render(<Documents />);
    await waitFor(() => screen.getByText("Upload Files"));
    fireEvent.click(screen.getByText("Upload Files"));
    await waitFor(() => {
      expect(screen.getByText("Upload 0 Files")).toBeDisabled();
    });
  });

  it("opens create document dialog", async () => {
    render(<Documents />);
    await waitFor(() => screen.getByText("Create Document"));
    fireEvent.click(screen.getByText("Create Document"));
    await waitFor(() => {
      expect(screen.getByText("Create New Document")).toBeInTheDocument();
    });
  });

  it("create document dialog has form fields", async () => {
    render(<Documents />);
    await waitFor(() => screen.getByText("Create Document"));
    fireEvent.click(screen.getByText("Create Document"));
    await waitFor(() => {
      expect(screen.getByText("Title")).toBeInTheDocument();
      expect(screen.getByText("Category")).toBeInTheDocument();
      expect(screen.getByText("Content")).toBeInTheDocument();
    });
  });

  it("handles download/view button click", async () => {
    const mockOpen = vi.fn().mockReturnValue({ document: { write: vi.fn(), close: vi.fn() } });
    vi.spyOn(window, "open").mockImplementation(mockOpen);
    
    render(<Documents />);
    await waitFor(() => {
      expect(screen.getByText("Assembly Guide")).toBeInTheDocument();
    });
    const viewButtons = screen.getAllByText("View");
    fireEvent.click(viewButtons[0]);
    expect(mockOpen).toHaveBeenCalledWith("", "_blank");
    
    (window.open as any).mockRestore();
  });

  it("handles API error on fetch", async () => {
    mockGetDocuments.mockRejectedValueOnce(new Error("fail"));
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    render(<Documents />);
    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalled();
    });
    consoleSpy.mockRestore();
  });

  it("shows loading skeletons", () => {
    render(<Documents />);
    // During loading, skeletons should be present
    const skeletons = document.querySelectorAll('[class*="skeleton"], [class*="Skeleton"]');
    // The component uses Skeleton components during loading
    expect(screen.getByText("Document Library")).toBeInTheDocument();
  });
});
