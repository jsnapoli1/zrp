import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import { mockAPIKeys } from "../test/mocks";

const mockGetAPIKeys = vi.fn();
const mockCreateAPIKey = vi.fn();
const mockRevokeAPIKey = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getAPIKeys: (...args: any[]) => mockGetAPIKeys(...args),
    createAPIKey: (...args: any[]) => mockCreateAPIKey(...args),
    revokeAPIKey: (...args: any[]) => mockRevokeAPIKey(...args),
  },
}));

// Mock clipboard API
Object.assign(navigator, {
  clipboard: {
    writeText: vi.fn().mockResolvedValue(undefined),
  },
});

import APIKeys from "./APIKeys";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetAPIKeys.mockResolvedValue(mockAPIKeys);
  mockCreateAPIKey.mockResolvedValue({ ...mockAPIKeys[0], full_key: "zrp_prod_abc123def456" });
  mockRevokeAPIKey.mockResolvedValue(undefined);
});

describe("APIKeys", () => {
  it("renders page title after loading", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      const titles = screen.getAllByText("API Keys");
      expect(titles.length).toBeGreaterThan(0);
    });
  });

  it("shows page description", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Manage API keys for programmatic access to ZRP.")).toBeInTheDocument();
    });
  });

  it("renders API key list after loading", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Production API")).toBeInTheDocument();
      expect(screen.getByText("Test API")).toBeInTheDocument();
    });
  });

  it("shows key status badges", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      const actives = screen.getAllByText(/active/i);
      expect(actives.length).toBeGreaterThan(0);
      const revoked = screen.getAllByText(/revoked/i);
      expect(revoked.length).toBeGreaterThan(0);
    });
  });

  it("has Generate New Key button", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Generate New Key")).toBeInTheDocument();
    });
  });

  it("shows summary cards", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Total Keys")).toBeInTheDocument();
      expect(screen.getByText("Revoked")).toBeInTheDocument();
    });
  });

  it("shows correct key counts", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("2")).toBeInTheDocument(); // total keys
    });
  });

  it("shows table headers", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Name")).toBeInTheDocument();
      expect(screen.getByText("Key")).toBeInTheDocument();
      expect(screen.getByText("Created By")).toBeInTheDocument();
      expect(screen.getByText("Last Used")).toBeInTheDocument();
    });
  });

  it("shows key prefixes", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("zrp_prod_")).toBeInTheDocument();
      expect(screen.getByText("zrp_test_")).toBeInTheDocument();
    });
  });

  it("shows Revoke buttons for active keys", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      const revokeButtons = screen.getAllByText("Revoke");
      expect(revokeButtons.length).toBe(1); // 1 active key
    });
  });

  it("shows 'Never' for keys without last_used", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      const neverTexts = screen.getAllByText("Never");
      expect(neverTexts.length).toBeGreaterThan(0);
    });
  });

  it("opens generate key dialog", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Generate New Key")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Generate New Key"));
    await waitFor(() => {
      expect(screen.getByText("Generate New API Key")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Enter a descriptive name")).toBeInTheDocument();
    });
  });

  it("shows important warning in generate dialog", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Generate New Key")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Generate New Key"));
    await waitFor(() => {
      expect(screen.getByText("Important")).toBeInTheDocument();
    });
  });

  it("Generate Key button disabled when name empty", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Generate New Key")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Generate New Key"));
    await waitFor(() => {
      const generateBtn = screen.getByText("Generate Key");
      expect(generateBtn).toBeDisabled();
    });
  });

  it("generates key and shows full key", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getByText("Generate New Key")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Generate New Key"));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Enter a descriptive name")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("Enter a descriptive name"), "My Test Key");
    await user.click(screen.getByText("Generate Key"));
    await waitFor(() => {
      expect(screen.getByText("API Key Generated Successfully")).toBeInTheDocument();
    });
  });

  it("opens revoke confirmation dialog", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getAllByText("Revoke").length).toBeGreaterThan(0);
    });
    await user.click(screen.getAllByText("Revoke")[0]);
    await waitFor(() => {
      expect(screen.getByText("Revoke API Key")).toBeInTheDocument();
      expect(screen.getByText("Confirm Revocation")).toBeInTheDocument();
    });
  });

  it("confirms key revocation", async () => {
    const user = userEvent.setup();
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getAllByText("Revoke").length).toBeGreaterThan(0);
    });
    await user.click(screen.getAllByText("Revoke")[0]);
    await waitFor(() => {
      expect(screen.getByText("Yes, Revoke Key")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Yes, Revoke Key"));
    await waitFor(() => {
      expect(screen.queryByText("Yes, Revoke Key")).not.toBeInTheDocument();
    });
  });

  it("shows created_by for each key", async () => {
    render(<APIKeys />);
    await waitFor(() => {
      expect(screen.getAllByText("admin").length).toBeGreaterThan(0);
    });
  });
});
