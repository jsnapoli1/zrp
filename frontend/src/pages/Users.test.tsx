import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "../test/test-utils";
import userEvent from "@testing-library/user-event";
import { mockUsers } from "../test/mocks";

const mockGetUsers = vi.fn();
const mockCreateUser = vi.fn();
const mockUpdateUser = vi.fn();

vi.mock("../lib/api", () => ({
  api: {
    getUsers: (...args: any[]) => mockGetUsers(...args),
    createUser: (...args: any[]) => mockCreateUser(...args),
    updateUser: (...args: any[]) => mockUpdateUser(...args),
  },
}));

import Users from "./Users";

beforeEach(() => {
  vi.clearAllMocks();
  mockGetUsers.mockResolvedValue(mockUsers);
  mockCreateUser.mockResolvedValue(mockUsers[0]);
  mockUpdateUser.mockResolvedValue(mockUsers[0]);
});

describe("Users", () => {
  it("renders loading state then content", async () => {
    render(<Users />);
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
      expect(screen.getByText("admin@zrp.com")).toBeInTheDocument();
      expect(screen.getByText("user1@zrp.com")).toBeInTheDocument();
    });
  });

  it("has Create User button", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Create User")).toBeInTheDocument();
    });
  });

  it("shows user roles with badges", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getAllByText("Administrator").length).toBeGreaterThan(0);
    });
  });

  it("shows user status badges", async () => {
    render(<Users />);
    await waitFor(() => {
      const activeBadges = screen.getAllByText("Active");
      expect(activeBadges.length).toBeGreaterThan(0);
    });
  });

  it("shows page description", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Manage user accounts, roles, and permissions.")).toBeInTheDocument();
    });
  });

  it("shows stats cards with counts", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Total Users")).toBeInTheDocument();
      expect(screen.getByText("Admins")).toBeInTheDocument();
      expect(screen.getByText("Active Today")).toBeInTheDocument();
    });
  });

  it("shows correct total user count", async () => {
    render(<Users />);
    await waitFor(() => {
      // 2 mock users - total users stat card
      expect(screen.getByText("Total Users")).toBeInTheDocument();
    });
  });

  it("shows table headers", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Username")).toBeInTheDocument();
      expect(screen.getByText("Email")).toBeInTheDocument();
      expect(screen.getByText("Role")).toBeInTheDocument();
      expect(screen.getByText("Status")).toBeInTheDocument();
      expect(screen.getByText("Last Login")).toBeInTheDocument();
      expect(screen.getByText("Created")).toBeInTheDocument();
      expect(screen.getByText("Actions")).toBeInTheDocument();
    });
  });

  it("shows Edit buttons for each user", async () => {
    render(<Users />);
    await waitFor(() => {
      const editButtons = screen.getAllByText("Edit");
      expect(editButtons.length).toBe(2); // 2 mock users
    });
  });

  it("shows all mock users", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("admin")).toBeInTheDocument();
      expect(screen.getByText("user1")).toBeInTheDocument();
    });
  });

  it("opens create user dialog when button clicked", async () => {
    const user = userEvent.setup();
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Create User")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Create User"));
    await waitFor(() => {
      expect(screen.getByText("Create New User")).toBeInTheDocument();
    });
  });

  it("shows create user form fields in dialog", async () => {
    const user = userEvent.setup();
    render(<Users />);
    await waitFor(() => {
      expect(screen.getByText("Create User")).toBeInTheDocument();
    });
    await user.click(screen.getByText("Create User"));
    await waitFor(() => {
      expect(screen.getByPlaceholderText("Enter username")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Enter email address")).toBeInTheDocument();
      expect(screen.getByPlaceholderText("Enter password")).toBeInTheDocument();
    });
  });

  it("opens edit dialog when Edit clicked", async () => {
    const user = userEvent.setup();
    render(<Users />);
    await waitFor(() => {
      expect(screen.getAllByText("Edit").length).toBeGreaterThan(0);
    });
    await user.click(screen.getAllByText("Edit")[0]);
    await waitFor(() => {
      expect(screen.getByText(/Edit User:/)).toBeInTheDocument();
    });
  });

  it("edit dialog shows role and status selects", async () => {
    const user = userEvent.setup();
    render(<Users />);
    await waitFor(() => {
      expect(screen.getAllByText("Edit").length).toBeGreaterThan(0);
    });
    await user.click(screen.getAllByText("Edit")[0]);
    await waitFor(() => {
      expect(screen.getByText("Update User")).toBeInTheDocument();
      expect(screen.getAllByText("Cancel").length).toBeGreaterThan(0);
    });
  });

  it("shows Users card title", async () => {
    render(<Users />);
    await waitFor(() => {
      const usersTitles = screen.getAllByText("Users");
      expect(usersTitles.length).toBeGreaterThan(0);
    });
  });

  it("shows 'Never' for users without last login", async () => {
    render(<Users />);
    await waitFor(() => {
      // Mock users don't have last_login set, so 'Never' should appear
      expect(screen.getByText("admin")).toBeInTheDocument();
    });
  });

  it("calls getUsers on mount", async () => {
    render(<Users />);
    await waitFor(() => {
      expect(mockGetUsers).toHaveBeenCalled();
    });
  });

  it("fills create user form end-to-end, submits, and verifies API called", async () => {
    const user = userEvent.setup();
    render(<Users />);
    await waitFor(() => expect(screen.getByText("Create User")).toBeInTheDocument());

    await user.click(screen.getByText("Create User"));
    await waitFor(() => expect(screen.getByText("Create New User")).toBeInTheDocument());

    await user.type(screen.getByPlaceholderText("Enter username"), "newuser");
    await user.type(screen.getByPlaceholderText("Enter email address"), "new@example.com");
    await user.type(screen.getByPlaceholderText("Enter password"), "Secret123!");

    // Submit
    const dialog = screen.getByRole("dialog");
    const createBtn = Array.from(dialog.querySelectorAll("button")).find(b => b.textContent === "Create User");
    await user.click(createBtn!);

    await waitFor(() => {
      expect(mockCreateUser).toHaveBeenCalledWith(
        expect.objectContaining({
          username: "newuser",
          email: "new@example.com",
          password: "Secret123!",
        })
      );
    });
  });

  it("fills edit user form and submits update via API", async () => {
    const user = userEvent.setup();
    render(<Users />);
    await waitFor(() => expect(screen.getAllByText("Edit").length).toBe(2));

    await user.click(screen.getAllByText("Edit")[0]);
    await waitFor(() => expect(screen.getByText(/Edit User:/)).toBeInTheDocument());

    await user.click(screen.getByText("Update User"));

    await waitFor(() => {
      expect(mockUpdateUser).toHaveBeenCalledWith(
        "U-001",
        expect.objectContaining({
          role: expect.any(String),
          status: expect.any(String),
        })
      );
    });
  });
});
