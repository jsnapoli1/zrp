# Error Handling Best Practices Guide
**Project:** ZRP (Zero-Risk PLM)  
**Last Updated:** 2026-02-19

---

## Table of Contents
1. [Frontend Error Handling](#frontend-error-handling)
2. [Backend Error Handling](#backend-error-handling)
3. [Error Message Standards](#error-message-standards)
4. [Common Patterns](#common-patterns)
5. [Testing Error Paths](#testing-error-paths)

---

## Frontend Error Handling

### Rule 1: Always Wrap API Calls in Try/Catch

**❌ BAD:**
```typescript
const handleDelete = async (id: string) => {
  await api.deleteItem(id);
  fetchItems(); // Will never run if delete fails!
};
```

**✅ GOOD:**
```typescript
const handleDelete = async (id: string) => {
  try {
    await api.deleteItem(id);
    toast.success("Item deleted successfully");
    fetchItems();
  } catch (error) {
    toast.error("Failed to delete item");
    console.error("Delete error:", error);
  }
};
```

### Rule 2: Use Consistent Error Display Patterns

| Operation Type | Pattern | Rationale |
|----------------|---------|-----------|
| **Data mutation** (POST/PUT/DELETE) | Toast notification | User needs immediate feedback |
| **Page load errors** | Inline `<ErrorState>` with retry | Recoverable, contextual |
| **Form validation** | Inline field errors | Specific, actionable |
| **Non-critical failures** | Console.error only | Developer debugging, no user impact |

#### Pattern: Data Fetching with Error State
```typescript
const [data, setData] = useState<Item[]>([]);
const [loading, setLoading] = useState(true);
const [error, setError] = useState<string | null>(null);

useEffect(() => {
  const fetchData = async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await api.getItems();
      setData(result);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load data';
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };
  fetchData();
}, []);

// In render:
if (loading) return <LoadingState />;
if (error) return <ErrorState message={error} onRetry={fetchData} />;
```

#### Pattern: Data Mutation
```typescript
const handleSave = async () => {
  try {
    setSaving(true);
    await api.updateItem(id, formData);
    toast.success("Item updated successfully");
    navigate('/items');
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Failed to update item';
    toast.error(message);
    console.error("Update error:", err);
  } finally {
    setSaving(false);
  }
};
```

#### Pattern: Form Submission with Validation
```typescript
const [formError, setFormError] = useState<string>("");

const handleSubmit = async () => {
  // Clear previous errors
  setFormError("");
  
  // Client-side validation
  if (!formData.title.trim()) {
    setFormError("Title is required");
    return;
  }
  
  try {
    await api.createItem(formData);
    toast.success("Item created successfully");
    navigate('/items');
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Failed to create item';
    setFormError(message);
    toast.error(message);
  }
};

// In render:
{formError && <div className="text-red-600">{formError}</div>}
```

### Rule 3: Clean Up useEffect Properly

```typescript
useEffect(() => {
  let mounted = true;
  
  const fetchData = async () => {
    try {
      const data = await api.getData();
      if (mounted) {
        setData(data);
      }
    } catch (err) {
      if (mounted) {
        setError(err.message);
      }
    }
  };
  
  fetchData();
  
  return () => {
    mounted = false;
  };
}, []);
```

### Rule 4: Use ErrorBoundary for Render Errors

```typescript
// App.tsx
<ErrorBoundary fallback={<ErrorPage />}>
  <Router>
    <Routes>...</Routes>
  </Router>
</ErrorBoundary>
```

**Note:** ErrorBoundary only catches render errors, NOT async errors in useEffect/event handlers. For those, use try/catch.

---

## Backend Error Handling

### Rule 1: Never Expose Implementation Details

**❌ BAD:**
```go
if err != nil {
    jsonErr(w, err.Error(), 500) // Exposes DB errors, file paths, etc.
    return
}
```

**✅ GOOD:**
```go
if err != nil {
    jsonErr(w, "Failed to process request. Please try again.", 500)
    log.Printf("Database error: %v", err) // Log internally
    return
}
```

### Rule 2: Validate DELETE Operations

**❌ BAD:**
```go
func handleDelete(w http.ResponseWriter, r *http.Request, id string) {
    db.Exec("DELETE FROM items WHERE id = ?", id)
    jsonResp(w, map[string]string{"status": "deleted"})
}
```

**✅ GOOD:**
```go
func handleDelete(w http.ResponseWriter, r *http.Request, id string) {
    res, err := db.Exec("DELETE FROM items WHERE id = ?", id)
    if err != nil {
        jsonErr(w, "Failed to delete item. Please try again.", 500)
        return
    }
    
    rows, _ := res.RowsAffected()
    if rows == 0 {
        jsonErr(w, "Item not found", 404)
        return
    }
    
    logAudit(db, getUsername(r), "deleted", "item", id, "Deleted item")
    jsonResp(w, map[string]string{"status": "deleted"})
}
```

### Rule 3: Always Validate Input

```go
func handleCreate(w http.ResponseWriter, r *http.Request) {
    var item Item
    if err := decodeBody(r, &item); err != nil {
        jsonErr(w, "Invalid request body", 400)
        return
    }
    
    // Use validation framework
    ve := &ValidationErrors{}
    requireField(ve, "title", item.Title)
    requireField(ve, "status", item.Status)
    validateMaxLength(ve, "title", item.Title, 200)
    validateEnum(ve, "status", item.Status, []string{"draft", "active", "archived"})
    
    if ve.HasErrors() {
        jsonErr(w, ve.Error(), 400)
        return
    }
    
    // Proceed with creation...
}
```

### Rule 4: Check Foreign Key References

```go
func handleCreateOrder(w http.ResponseWriter, r *http.Request) {
    var order Order
    if err := decodeBody(r, &order); err != nil {
        jsonErr(w, "Invalid request body", 400)
        return
    }
    
    // Verify customer exists
    var customerExists int
    err := db.QueryRow("SELECT COUNT(*) FROM customers WHERE id = ?", order.CustomerID).Scan(&customerExists)
    if err != nil || customerExists == 0 {
        jsonErr(w, "Customer not found", 404)
        return
    }
    
    // Proceed with creation...
}
```

---

## Error Message Standards

### User-Facing Messages

| Situation | Message Template | Example |
|-----------|-----------------|---------|
| **Read failure** | "Failed to load {resource}. Please try again." | "Failed to load parts. Please try again." |
| **Create failure** | "Failed to create {resource}. Please try again." | "Failed to create purchase order. Please try again." |
| **Update failure** | "Failed to update {resource}. Please try again." | "Failed to update device. Please try again." |
| **Delete failure** | "Failed to delete {resource}. Please try again." | "Failed to delete vendor. Please try again." |
| **Not found** | "{Resource} not found" | "Part not found" |
| **Validation error** | "{Field} {constraint}" | "Title is required", "Email must be valid" |
| **Permission denied** | "You don't have permission to {action}" | "You don't have permission to delete this item" |
| **Network error** | "Network error. Please check your connection." | |
| **Unexpected error** | "An unexpected error occurred. Please try again." | |

### Success Messages

| Action | Message Template | Example |
|--------|-----------------|---------|
| **Create** | "{Resource} created successfully" | "Purchase order created successfully" |
| **Update** | "{Resource} updated successfully" | "Part updated successfully" |
| **Delete** | "{Resource} deleted successfully" | "Vendor deleted successfully" |
| **Status change** | "Status updated to {status}" | "Status updated to approved" |

---

## Common Patterns

### Pattern: Safe Resource Fetching

```typescript
// Frontend
const fetchResource = async (id: string) => {
  try {
    setLoading(true);
    setError(null);
    const data = await api.getResource(id);
    setResource(data);
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Failed to load resource';
    setError(message);
    
    // For critical pages, redirect on error
    if (err instanceof Error && err.message.includes('not found')) {
      navigate('/resources');
    }
  } finally {
    setLoading(false);
  }
};
```

```go
// Backend
func handleGetResource(w http.ResponseWriter, r *http.Request, id string) {
    var resource Resource
    err := db.QueryRow("SELECT * FROM resources WHERE id = ?", id).Scan(
        &resource.ID, &resource.Title, // ... all fields
    )
    if err == sql.ErrNoRows {
        jsonErr(w, "Resource not found", 404)
        return
    }
    if err != nil {
        jsonErr(w, "Failed to fetch resource. Please try again.", 500)
        return
    }
    
    jsonResp(w, resource)
}
```

### Pattern: Bulk Operations

```typescript
// Frontend
const handleBulkDelete = async (ids: string[]) => {
  try {
    const result = await api.bulkDelete(ids);
    
    if (result.failed > 0) {
      toast.warning(`Deleted ${result.success} items. Failed: ${result.failed}`);
      
      // Show detailed errors
      if (result.errors.length > 0) {
        console.error("Bulk delete errors:", result.errors);
      }
    } else {
      toast.success(`${result.success} items deleted successfully`);
    }
    
    fetchItems(); // Refresh list
  } catch (err) {
    toast.error("Failed to delete items");
    console.error("Bulk delete error:", err);
  }
};
```

```go
// Backend
func handleBulkDelete(w http.ResponseWriter, r *http.Request) {
    var req struct {
        IDs []string `json:"ids"`
    }
    if err := decodeBody(r, &req); err != nil {
        jsonErr(w, "Invalid request body", 400)
        return
    }
    
    if len(req.IDs) == 0 {
        jsonErr(w, "No IDs provided", 400)
        return
    }
    
    success := 0
    failed := 0
    var errors []string
    
    for _, id := range req.IDs {
        res, err := db.Exec("DELETE FROM items WHERE id = ?", id)
        if err != nil {
            failed++
            errors = append(errors, fmt.Sprintf("ID %s: %v", id, err))
            continue
        }
        
        rows, _ := res.RowsAffected()
        if rows > 0 {
            success++
        } else {
            failed++
            errors = append(errors, fmt.Sprintf("ID %s: not found", id))
        }
    }
    
    jsonResp(w, map[string]interface{}{
        "success": success,
        "failed":  failed,
        "errors":  errors,
    })
}
```

### Pattern: Async Operations with Progress

```typescript
// For long-running operations
const handleImport = async (file: File) => {
  const [progress, setProgress] = useState(0);
  const [status, setStatus] = useState<string>("Uploading...");
  
  try {
    setProgress(25);
    const uploadResult = await api.uploadFile(file);
    
    setStatus("Processing...");
    setProgress(50);
    const importResult = await api.processImport(uploadResult.id);
    
    setProgress(100);
    setStatus("Complete");
    
    toast.success(`Imported ${importResult.success} items`);
    
    if (importResult.errors.length > 0) {
      // Show errors in a modal or download error log
      setImportErrors(importResult.errors);
    }
  } catch (err) {
    setStatus("Failed");
    toast.error("Import failed");
    console.error("Import error:", err);
  }
};
```

---

## Testing Error Paths

### Frontend Tests

```typescript
describe("Item deletion", () => {
  it("shows error toast on API failure", async () => {
    const mockDeleteItem = vi.fn().mockRejectedValue(new Error("Network error"));
    api.deleteItem = mockDeleteItem;
    
    render(<ItemList />);
    
    const deleteButton = screen.getByRole("button", { name: /delete/i });
    await userEvent.click(deleteButton);
    
    await waitFor(() => {
      expect(screen.getByText("Failed to delete item")).toBeInTheDocument();
    });
  });
  
  it("retries on error button click", async () => {
    const mockGetItems = vi.fn()
      .mockRejectedValueOnce(new Error("Network error"))
      .mockResolvedValueOnce([{ id: "1", title: "Item" }]);
    
    api.getItems = mockGetItems;
    
    render(<ItemList />);
    
    await waitFor(() => {
      expect(screen.getByText(/failed to load/i)).toBeInTheDocument();
    });
    
    const retryButton = screen.getByRole("button", { name: /retry/i });
    await userEvent.click(retryButton);
    
    await waitFor(() => {
      expect(screen.getByText("Item")).toBeInTheDocument();
    });
    
    expect(mockGetItems).toHaveBeenCalledTimes(2);
  });
});
```

### Backend Tests

```go
func TestHandleDeleteItem_NotFound(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    req := httptest.NewRequest("DELETE", "/api/v1/items/999", nil)
    w := httptest.NewRecorder()
    
    handleDeleteItem(w, req, "999")
    
    assert.Equal(t, 404, w.Code)
    
    var resp map[string]string
    json.NewDecoder(w.Body).Decode(&resp)
    assert.Equal(t, "Item not found", resp["error"])
}

func TestHandleDeleteItem_Success(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    // Create item
    db.Exec("INSERT INTO items (id, title) VALUES (?, ?)", "1", "Test Item")
    
    req := httptest.NewRequest("DELETE", "/api/v1/items/1", nil)
    w := httptest.NewRecorder()
    
    handleDeleteItem(w, req, "1")
    
    assert.Equal(t, 200, w.Code)
    
    // Verify deleted
    var count int
    db.QueryRow("SELECT COUNT(*) FROM items WHERE id = ?", "1").Scan(&count)
    assert.Equal(t, 0, count)
}
```

---

## Checklist for New Features

### Frontend
- [ ] All API calls wrapped in try/catch
- [ ] Loading states shown during async operations
- [ ] Error states with retry button for data fetching
- [ ] Toast notifications for mutations (create/update/delete)
- [ ] Form validation errors shown inline
- [ ] Error messages are user-friendly (no stack traces)
- [ ] Console.error for debugging includes context
- [ ] Tests cover error paths

### Backend
- [ ] Input validation using ValidationErrors
- [ ] Generic errors replaced with user-friendly messages
- [ ] DELETE operations check RowsAffected()
- [ ] Foreign key references validated before INSERT
- [ ] Audit log entries created for mutations
- [ ] Error responses use correct HTTP status codes
- [ ] Tests cover validation errors, not found, and database errors

---

## Quick Reference

### Frontend Error Handling Imports
```typescript
import { toast } from "sonner";
import { LoadingState } from "../components/LoadingState";
import { ErrorState } from "../components/ErrorState";
import { EmptyState } from "../components/EmptyState";
```

### Backend Error Response Helper
```go
// Already available in main.go:
func jsonErr(w http.ResponseWriter, message string, code int) {
    w.WriteHeader(code)
    json.NewEncoder(w).Encode(map[string]string{"error": message})
}
```

### Backend Validation Helper
```go
ve := &ValidationErrors{}
requireField(ve, "title", item.Title)
validateMaxLength(ve, "title", item.Title, 200)
validateEnum(ve, "status", item.Status, validStatuses)
if ve.HasErrors() {
    jsonErr(w, ve.Error(), 400)
    return
}
```

---

## Common Mistakes to Avoid

1. **❌ Using .then().catch() instead of async/await**
   - Makes error handling harder to read and maintain
   - Use async/await with try/catch instead

2. **❌ Ignoring errors in useEffect**
   - Always handle errors, even if just logging them
   - Consider if the error should prevent component from rendering

3. **❌ Not cleaning up async operations**
   - Use cleanup functions in useEffect
   - Check if component is still mounted before setState

4. **❌ Showing technical error messages to users**
   - "SQLITE_CONSTRAINT violation" → "Failed to create item. Please try again."
   - Always translate technical errors to user-friendly messages

5. **❌ Not logging errors for debugging**
   - Always console.error or log.Printf for troubleshooting
   - Include context (operation, ID, params)

6. **❌ Silent failures (catch with no user feedback)**
   - Every error should either:
     - Show a toast/alert to the user
     - Show an inline error state
     - Or be non-critical and logged only

---

## Resources

- **Toast library:** [Sonner](https://sonner.emilkowal.ski/)
- **React error boundaries:** [React docs](https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary)
- **Go error handling:** [Effective Go](https://go.dev/doc/effective_go#errors)
- **HTTP status codes:** [MDN reference](https://developer.mozilla.org/en-US/docs/Web/HTTP/Status)

---

**Remember:** Good error handling is not optional — it's what separates a prototype from production software. Users judge reliability by how gracefully your app handles failure.
