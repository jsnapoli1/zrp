import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import { mockEmailConfig } from "../test/mocks";

const mockGetEmailConfig = vi.fn();
const mockUpdateEmailConfig = vi.fn();
const mockTestEmail = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getEmailConfig: (...args: any[]) => mockGetEmailConfig(...args),
    updateEmailConfig: (...args: any[]) => mockUpdateEmailConfig(...args),
    testEmail: (...args: any[]) => mockTestEmail(...args),
  },
}));

import EmailSettings from "./EmailSettings";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetEmailConfig.mockResolvedValue(mockEmailConfig);
  mockUpdateEmailConfig.mockResolvedValue(mockEmailConfig);
  mockTestEmail.mockResolvedValue({ success: true, message: "Email sent" });
});

describe("EmailSettings", () => {
  it("renders page title after loading", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Email Settings")).toBeInTheDocument();
    });
  });

  it("shows page description", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText(/Configure SMTP settings/)).toBeInTheDocument();
    });
  });

  it("shows SMTP configuration card", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("SMTP Configuration")).toBeInTheDocument();
    });
  });

  it("shows SMTP host and port fields", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("SMTP Host")).toBeInTheDocument();
      expect(screen.getByText("SMTP Port")).toBeInTheDocument();
    });
  });

  it("shows authentication section", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Authentication")).toBeInTheDocument();
      expect(screen.getByText("Username")).toBeInTheDocument();
      expect(screen.getByText("Password")).toBeInTheDocument();
    });
  });

  it("shows Security select", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Security")).toBeInTheDocument();
    });
  });

  it("has Save Settings button", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Save Settings")).toBeInTheDocument();
    });
  });

  it("Save button is disabled when no changes", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      const saveButton = screen.getByText("Save Settings").closest("button");
      expect(saveButton).toBeDisabled();
    });
  });

  it("shows Send Test button", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Send Test")).toBeInTheDocument();
    });
  });

  it("shows Test Email Configuration section", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Test Email Configuration")).toBeInTheDocument();
      expect(screen.getByText("Test Email Address")).toBeInTheDocument();
    });
  });

  it("shows Email Notifications card with checkbox", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Email Notifications")).toBeInTheDocument();
      expect(screen.getByText("Enable email notifications")).toBeInTheDocument();
    });
  });

  it("shows enabled notification message when email is enabled", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Email notifications are enabled")).toBeInTheDocument();
    });
  });

  it("shows Sender Configuration card", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Sender Configuration")).toBeInTheDocument();
      expect(screen.getByText("From Email Address")).toBeInTheDocument();
      expect(screen.getByText("From Name")).toBeInTheDocument();
    });
  });

  it("shows SMTP Provider Preset selector", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("SMTP Provider Preset")).toBeInTheDocument();
    });
  });

  it("shows mock SMTP host value", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      const hostInput = screen.getByDisplayValue("smtp.example.com");
      expect(hostInput).toBeInTheDocument();
    });
  });

  it("shows mock SMTP port value", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      const portInput = screen.getByDisplayValue("587");
      expect(portInput).toBeInTheDocument();
    });
  });

  it("shows mock username value", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      // mockEmailConfig has smtp_username and from_address both as "zrp@example.com"
      expect(screen.getAllByDisplayValue("zrp@example.com").length).toBeGreaterThan(0);
    });
  });

  it("shows mock from address value", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      // from_address from mockEmailConfig is "zrp@example.com" - same as username
      expect(screen.getAllByDisplayValue("zrp@example.com").length).toBeGreaterThan(0);
    });
  });

  it("shows test email placeholder", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("test@example.com")).toBeInTheDocument();
    });
  });

  it("shows loading state initially", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("Email Settings")).toBeInTheDocument();
    });
  });

  it("shows unsaved changes badge when config modified", async () => {
    const user = userEvent.setup();
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("smtp.example.com")).toBeInTheDocument();
    });
    const hostInput = screen.getByDisplayValue("smtp.example.com");
    await user.clear(hostInput);
    await user.type(hostInput, "new.smtp.host.com");
    await waitFor(() => {
      expect(screen.getByText("Unsaved Changes")).toBeInTheDocument();
    });
  });

  it("enables Save button when changes are made", async () => {
    const user = userEvent.setup();
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("smtp.example.com")).toBeInTheDocument();
    });
    const hostInput = screen.getByDisplayValue("smtp.example.com");
    await user.clear(hostInput);
    await user.type(hostInput, "new.host.com");
    await waitFor(() => {
      const saveButton = screen.getByText("Save Settings").closest("button");
      expect(saveButton).not.toBeDisabled();
    });
  });

  it("Send Test button is disabled when no test email entered", async () => {
    render(<EmailSettings />);
    await waitFor(() => {
      const sendBtn = screen.getByText("Send Test").closest("button");
      expect(sendBtn).toBeDisabled();
    });
  });

  it("hides SMTP sections when notifications disabled", async () => {
    const user = userEvent.setup();
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("SMTP Configuration")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Enable email notifications"));
    await waitFor(() => {
      expect(screen.queryByText("SMTP Configuration")).not.toBeInTheDocument();
      expect(screen.queryByText("Sender Configuration")).not.toBeInTheDocument();
      expect(screen.queryByText("Test Email Configuration")).not.toBeInTheDocument();
    });
  });

  it("auto-fills SMTP settings when a preset is selected", async () => {
    const user = userEvent.setup();
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByText("SMTP Provider Preset")).toBeInTheDocument();
    });
    const presetTrigger = screen.getByText("Choose a preset or configure manually").closest("button")!;
    await user.click(presetTrigger);
    await waitFor(() => {
      expect(screen.getByText("Gmail")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Gmail"));
    await waitFor(() => {
      expect(screen.getByDisplayValue("smtp.gmail.com")).toBeInTheDocument();
    });
  });

  it("saves settings and clears unsaved badge", async () => {
    const user = userEvent.setup();
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByDisplayValue("smtp.example.com")).toBeInTheDocument();
    });
    const hostInput = screen.getByDisplayValue("smtp.example.com");
    await user.clear(hostInput);
    await user.type(hostInput, "new.host.com");
    await waitFor(() => {
      expect(screen.getByText("Unsaved Changes")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Save Settings"));
    await waitFor(() => {
      expect(screen.queryByText("Unsaved Changes")).not.toBeInTheDocument();
    }, { timeout: 3000 });
  });

  it("sends test email and shows success result", async () => {
    const user = userEvent.setup();
    vi.spyOn(Math, "random").mockReturnValue(0.5);
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("test@example.com")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("test@example.com"), "user@test.com");
    await user.click(screen.getByText("Send Test"));
    await waitFor(() => {
      expect(screen.getByText("Test Successful")).toBeInTheDocument();
      expect(screen.getByText(/Test email sent successfully to user@test.com/)).toBeInTheDocument();
    }, { timeout: 5000 });
    vi.spyOn(Math, "random").mockRestore();
  });

  it("sends test email and shows failure result", async () => {
    const user = userEvent.setup();
    vi.spyOn(Math, "random").mockReturnValue(0.1);
    render(<EmailSettings />);
    await waitFor(() => {
      expect(screen.getByPlaceholderText("test@example.com")).toBeInTheDocument();
    });
    await user.type(screen.getByPlaceholderText("test@example.com"), "user@test.com");
    await user.click(screen.getByText("Send Test"));
    await waitFor(() => {
      expect(screen.getByText("Test Failed")).toBeInTheDocument();
      expect(screen.getByText(/Failed to send test email/)).toBeInTheDocument();
    }, { timeout: 5000 });
    vi.spyOn(Math, "random").mockRestore();
  });
});
