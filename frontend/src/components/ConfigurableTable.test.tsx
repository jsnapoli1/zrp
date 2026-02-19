import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "../test/test-utils";
import { ConfigurableTable, type ColumnDef } from "./ConfigurableTable";

interface TestRow {
  id: string;
  name: string;
  value: number;
  extra: string;
}

const testData: TestRow[] = [
  { id: "1", name: "Alpha", value: 10, extra: "x" },
  { id: "2", name: "Beta", value: 20, extra: "y" },
];

const sortTestData: TestRow[] = [
  { id: "1", name: "Charlie", value: 30, extra: "a" },
  { id: "2", name: "Alpha", value: 10, extra: "c" },
  { id: "3", name: "Beta", value: 20, extra: "b" },
];

const testColumns: ColumnDef<TestRow>[] = [
  { id: "name", label: "Name", accessor: (r) => r.name, sortValue: (r) => r.name },
  { id: "value", label: "Value", accessor: (r) => r.value, sortValue: (r) => r.value },
  { id: "extra", label: "Extra", accessor: (r) => r.extra, defaultVisible: false, sortValue: (r) => r.extra },
];

beforeEach(() => {
  localStorage.clear();
});

describe("ConfigurableTable", () => {
  it("renders visible columns and data", () => {
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
      />
    );
    expect(screen.getByText("Name")).toBeInTheDocument();
    expect(screen.getByText("Value")).toBeInTheDocument();
    expect(screen.getByText("Alpha")).toBeInTheDocument();
    expect(screen.getByText("Beta")).toBeInTheDocument();
    // Extra column is defaultVisible=false
    expect(screen.queryByText("Extra")).not.toBeInTheDocument();
  });

  it("hides columns with defaultVisible=false", () => {
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
      />
    );
    // "x" and "y" are from the Extra column which is hidden
    expect(screen.queryByText("x")).not.toBeInTheDocument();
  });

  it("shows empty message when no data", () => {
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={[]}
        rowKey={(r) => r.id}
        emptyMessage="Nothing here"
      />
    );
    expect(screen.getByText("Nothing here")).toBeInTheDocument();
  });

  it("calls onRowClick when row is clicked", () => {
    const onClick = vi.fn();
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
        onRowClick={onClick}
      />
    );
    fireEvent.click(screen.getByText("Alpha"));
    expect(onClick).toHaveBeenCalledWith(testData[0]);
  });

  it("persists column state to localStorage", () => {
    render(
      <ConfigurableTable
        tableName="persist-test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
      />
    );
    const stored = localStorage.getItem("zrp-columns-persist-test");
    expect(stored).toBeTruthy();
    const parsed = JSON.parse(stored!);
    expect(parsed).toHaveLength(3);
    expect(parsed[0].id).toBe("name");
    expect(parsed[2].visible).toBe(false); // extra
  });

  it("restores column state from localStorage", () => {
    // Pre-set state with extra visible and name hidden
    const state = [
      { id: "name", visible: false },
      { id: "value", visible: true },
      { id: "extra", visible: true },
    ];
    localStorage.setItem("zrp-columns-restore-test", JSON.stringify(state));

    render(
      <ConfigurableTable
        tableName="restore-test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
      />
    );
    // Name column should be hidden
    // We can check that "Alpha" is not shown as table cell (but Name header is gone)
    // Extra "x" should now appear
    expect(screen.getByText("x")).toBeInTheDocument();
    expect(screen.getByText("y")).toBeInTheDocument();
  });

  it("has settings gear button for column config", () => {
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
      />
    );
    // Settings button exists (Settings2 icon)
    const settingsBtn = screen.getAllByRole("button").find(
      (btn) => btn.querySelector("svg.lucide-settings-2")
    );
    expect(settingsBtn).toBeTruthy();
  });

  it("renders leading column when provided", () => {
    render(
      <ConfigurableTable
        tableName="test"
        columns={testColumns}
        data={testData}
        rowKey={(r) => r.id}
        leadingColumn={{
          header: <span>Check</span>,
          cell: (r) => <input type="checkbox" data-testid={`check-${r.id}`} />,
        }}
      />
    );
    expect(screen.getByText("Check")).toBeInTheDocument();
    expect(screen.getByTestId("check-1")).toBeInTheDocument();
    expect(screen.getByTestId("check-2")).toBeInTheDocument();
  });

  describe("sorting", () => {
    function getCellTexts(column: number): string[] {
      const rows = screen.getAllByRole("row").slice(1); // skip header
      return rows.map((row) => {
        const cells = row.querySelectorAll("td");
        return cells[column]?.textContent || "";
      });
    }

    it("sorts ascending on first click of column header", () => {
      render(
        <ConfigurableTable
          tableName="sort-test"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Name"));
      expect(getCellTexts(0)).toEqual(["Alpha", "Beta", "Charlie"]);
    });

    it("sorts descending on second click", () => {
      render(
        <ConfigurableTable
          tableName="sort-test2"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Name"));
      fireEvent.click(screen.getByText("Name"));
      expect(getCellTexts(0)).toEqual(["Charlie", "Beta", "Alpha"]);
    });

    it("clears sort on third click (back to original order)", () => {
      render(
        <ConfigurableTable
          tableName="sort-test3"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Name"));
      fireEvent.click(screen.getByText("Name"));
      fireEvent.click(screen.getByText("Name"));
      // Original order: Charlie, Alpha, Beta
      expect(getCellTexts(0)).toEqual(["Charlie", "Alpha", "Beta"]);
    });

    it("sorts numeric values correctly", () => {
      render(
        <ConfigurableTable
          tableName="sort-num"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Value"));
      expect(getCellTexts(1)).toEqual(["10", "20", "30"]);
    });

    it("shows sort direction indicator", () => {
      const { container } = render(
        <ConfigurableTable
          tableName="sort-ind"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      // No indicator initially
      expect(container.querySelector(".lucide-arrow-up-narrow-wide")).toBeNull();
      expect(container.querySelector(".lucide-arrow-down-wide-narrow")).toBeNull();

      fireEvent.click(screen.getByText("Name"));
      expect(container.querySelector(".lucide-arrow-up-narrow-wide")).toBeTruthy();

      fireEvent.click(screen.getByText("Name"));
      expect(container.querySelector(".lucide-arrow-down-wide-narrow")).toBeTruthy();
    });

    it("switching sort column resets to ascending on new column", () => {
      render(
        <ConfigurableTable
          tableName="sort-switch"
          columns={testColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      fireEvent.click(screen.getByText("Name"));
      fireEvent.click(screen.getByText("Name")); // desc on Name
      fireEvent.click(screen.getByText("Value")); // asc on Value
      expect(getCellTexts(1)).toEqual(["10", "20", "30"]);
    });

    it("does not sort columns without sortValue", () => {
      const noSortColumns: ColumnDef<TestRow>[] = [
        { id: "name", label: "Name", accessor: (r) => r.name },
        { id: "value", label: "Value", accessor: (r) => r.value, sortValue: (r) => r.value },
      ];
      render(
        <ConfigurableTable
          tableName="no-sort"
          columns={noSortColumns}
          data={sortTestData}
          rowKey={(r) => r.id}
        />
      );
      // Click Name header - should not sort (no sortValue)
      fireEvent.click(screen.getByText("Name"));
      expect(getCellTexts(0)).toEqual(["Charlie", "Alpha", "Beta"]);
    });
  });
});
