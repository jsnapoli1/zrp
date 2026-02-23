# UI Patterns Guide

**Version**: 1.0  
**Last Updated**: Feb 19, 2026  
**Design System**: shadcn/ui (new-york style, zinc theme)

This guide documents standardized UI patterns for ZRP to ensure consistency, accessibility, and polish across all pages.

---

## 🎨 Reusable Components

All reusable UI state components are located in `frontend/src/components/`:

### 1. LoadingState

**Location**: `components/LoadingState.tsx`  
**Purpose**: Standardized loading indicators with consistent styling

**Variants**:
- `spinner` - Centered spinner with optional message (default)
- `skeleton` - List skeleton rows
- `table` - Table skeleton rows

**Usage**:
```tsx
import { LoadingState } from "../components/LoadingState";

// Centered spinner (most common)
if (loading) {
  return <LoadingState variant="spinner" message="Loading parts..." />;
}

// Skeleton list (5 rows by default)
<LoadingState variant="skeleton" rows={5} />

// Table skeleton (10 rows)
<LoadingState variant="table" rows={10} />
```

**✅ Do**:
- Use `LoadingState` for all loading scenarios
- Show immediately when data fetch begins
- Include descriptive messages ("Loading parts..." not "Loading...")
- Use `variant="table"` for table data, `variant="skeleton"` for lists

**❌ Don't**:
- Use inline `{loading && <div>Loading...</div>}`
- Leave loading states without messages
- Mix different loading styles in the same app

---

### 2. EmptyState

**Location**: `components/EmptyState.tsx`  
**Purpose**: Helpful feedback when no data exists

**Props**:
- `icon` - Lucide icon component (defaults to Inbox)
- `title` - Main message (required)
- `description` - Optional explanation
- `action` - Optional CTA button/element

**Usage**:
```tsx
import { EmptyState } from "../components/EmptyState";
import { Package, Plus } from "lucide-react";

// Basic empty state
{data.length === 0 && (
  <EmptyState 
    icon={Package}
    title="No parts found"
    description="Get started by adding your first part"
  />
)}

// With action button
<EmptyState 
  icon={Package}
  title="No parts found"
  description="Try adjusting your search or filters"
  action={
    <Button onClick={handleCreate}>
      <Plus className="h-4 w-4 mr-2" />
      Add Part
    </Button>
  }
/>

// Search/filter context
{filteredData.length === 0 && searchQuery && (
  <EmptyState 
    title="No results found"
    description={`No items matching "${searchQuery}"`}
  />
)}
```

**✅ Do**:
- Use icons that match the content type
- Provide helpful descriptions
- Include CTAs for creating first item
- Differentiate between "no data" and "no search results"

**❌ Don't**:
- Show empty state while loading
- Use generic "No items" without context
- Forget to handle empty search results separately

---

### 3. ErrorState

**Location**: `components/ErrorState.tsx`  
**Purpose**: User-friendly error messaging with retry actions

**Variants**:
- `full` - Centered error page (default)
- `inline` - Compact error banner

**Props**:
- `title` - Error title (default: "Something went wrong")
- `message` - Error description (default: "An error occurred...")
- `onRetry` - Optional retry callback
- `variant` - "full" or "inline"

**Usage**:
```tsx
import { ErrorState } from "../components/ErrorState";

// Full page error
if (error) {
  return (
    <ErrorState 
      title="Failed to load parts"
      message={error.message}
      onRetry={fetchParts}
    />
  );
}

// Inline error (in a card)
{error && (
  <ErrorState 
    variant="inline"
    message="Failed to save changes"
    onRetry={handleSave}
  />
)}

// Without retry
<ErrorState 
  variant="inline"
  title="Validation error"
  message="Please fill in all required fields"
/>
```

**✅ Do**:
- Always provide retry actions when applicable
- Show user-friendly messages (not stack traces)
- Use `variant="inline"` for form/section errors
- Log technical errors to console separately

**❌ Don't**:
- Show raw API error messages
- Leave errors without retry options
- Use error states for validation (use form field errors instead)

---

### 4. FormField

**Location**: `components/FormField.tsx`  
**Purpose**: Standardized form field wrapper with label, validation, and accessibility

**Props**:
- `label` - Field label (required)
- `htmlFor` - Input ID for label association
- `required` - Show required indicator
- `error` - Validation error message
- `description` - Helper text
- `children` - Form input element

**Usage**:
```tsx
import { FormField } from "../components/FormField";
import { Input } from "../components/ui/input";

<FormField 
  label="Part Number"
  htmlFor="ipn"
  required
  error={errors.ipn?.message}
  description="Use format: XXX-NNNNN"
>
  <Input 
    id="ipn"
    placeholder="ABC-12345"
    {...register("ipn")}
  />
</FormField>

// With Select
<FormField 
  label="Category"
  htmlFor="category"
  required
>
  <Select onValueChange={handleChange}>
    <SelectTrigger id="category">
      <SelectValue placeholder="Select..." />
    </SelectTrigger>
    <SelectContent>
      <SelectItem value="1">Hardware</SelectItem>
    </SelectContent>
  </Select>
</FormField>
```

**✅ Do**:
- Always use `htmlFor` with matching input `id`
- Mark required fields with `required` prop
- Show validation errors inline
- Include helpful descriptions for complex fields

**❌ Don't**:
- Manually create label/error markup
- Forget to associate labels with inputs
- Show errors and descriptions at the same time

---

## 📐 Responsive Design Patterns

Use Tailwind's responsive prefixes consistently:

### Breakpoints
- `sm:` - 640px (small tablets)
- `md:` - 768px (tablets)
- `lg:` - 1024px (laptops)
- `xl:` - 1280px (desktops)
- `2xl:` - 1536px (large desktops)

### Common Patterns

**Grid Layouts**:
```tsx
// 1 col mobile, 2 col tablet, 4 col desktop
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
  {/* cards */}
</div>
```

**Flex Layouts**:
```tsx
// Stack on mobile, row on desktop
<div className="flex flex-col sm:flex-row gap-4">
  {/* content */}
</div>
```

**Table Visibility**:
```tsx
// Hide less important columns on mobile
<th className="hidden md:table-cell">Description</th>
<td className="hidden md:table-cell">{item.description}</td>
```

**Button Groups**:
```tsx
// Stack on mobile, inline on desktop
<div className="flex flex-col sm:flex-row gap-2">
  <Button>Primary</Button>
  <Button variant="outline">Secondary</Button>
</div>
```

---

## ♿ Accessibility Patterns

### Form Labels
**Always** associate labels with inputs:
```tsx
// ✅ Good
<Label htmlFor="email">Email</Label>
<Input id="email" type="email" />

// ❌ Bad
<Label>Email</Label>
<Input type="email" />
```

### Button Accessibility
```tsx
// Icon-only buttons need aria-label
<Button 
  variant="ghost" 
  size="icon"
  aria-label="Delete item"
>
  <Trash2 className="h-4 w-4" />
</Button>

// Buttons with text don't need aria-label
<Button>
  <Plus className="h-4 w-4 mr-2" />
  Add Item
</Button>
```

### Loading States
```tsx
// Add role and aria-label
<div role="status" aria-label="Loading data">
  <Loader2 className="h-6 w-6 animate-spin" />
</div>
```

### Focus Management
- Ensure all interactive elements are keyboard-accessible
- Maintain logical tab order
- Use visible focus indicators (default shadcn/ui styles)

---

## 🎨 Theme & Spacing

### Colors (Zinc Theme)
Use semantic color classes from shadcn/ui:

- `text-foreground` - Primary text
- `text-muted-foreground` - Secondary text
- `text-destructive` - Errors
- `bg-background` - Page background
- `bg-muted` - Subtle backgrounds
- `border` - Default borders

### Spacing Scale
Use Tailwind's spacing scale consistently:
- `gap-2` (8px) - Tight spacing (icon + text)
- `gap-4` (16px) - Default spacing (form fields, cards)
- `gap-6` (24px) - Section spacing
- `space-y-6` (24px) - Vertical rhythm for page sections

### Typography
```tsx
// Page title
<h1 className="text-3xl font-bold tracking-tight">Title</h1>

// Card title
<CardTitle className="text-base font-medium">Card Title</CardTitle>

// Muted text
<p className="text-sm text-muted-foreground">Helper text</p>
```

---

## 🧪 Testing Guidelines

### Component Tests
Test reusable components in isolation:
```tsx
// FormField.test.tsx
it("associates label with input", () => {
  render(
    <FormField label="Email" htmlFor="email">
      <input id="email" />
    </FormField>
  );
  const label = screen.getByText("Email");
  expect(label).toHaveAttribute("for", "email");
});
```

### Page Tests
Focus on user interactions and state:
```tsx
// Parts.test.tsx
it("shows loading state initially", () => {
  render(<Parts />);
  expect(screen.getByText(/loading/i)).toBeInTheDocument();
});

it("shows empty state when no parts", async () => {
  mockAPI.getParts.mockResolvedValue([]);
  render(<Parts />);
  await waitFor(() => {
    expect(screen.getByText(/no parts found/i)).toBeInTheDocument();
  });
});
```

---

## 📋 Page Checklist

Use this checklist when creating or reviewing pages:

### Loading States
- [ ] Uses `<LoadingState />` component
- [ ] Shows immediately on data fetch
- [ ] Includes descriptive message
- [ ] Blocks UI during loading

### Empty States
- [ ] Uses `<EmptyState />` component
- [ ] Provides helpful message
- [ ] Includes CTA when applicable
- [ ] Differentiates between "no data" and "no search results"

### Error States
- [ ] Uses `<ErrorState />` component
- [ ] Shows user-friendly messages
- [ ] Provides retry action
- [ ] Logs technical errors to console

### Form Validation
- [ ] Uses `<FormField />` for all fields
- [ ] Marks required fields with `required` prop
- [ ] Shows inline validation errors
- [ ] Disables submit during save

### Responsive Design
- [ ] Grid/flex layouts adapt to mobile
- [ ] Tables hide less important columns on small screens
- [ ] Buttons/inputs sized for touch (min 44px)
- [ ] Tested at 320px, 768px, 1024px widths

### Accessibility
- [ ] All form labels use `htmlFor` + `id`
- [ ] Icon-only buttons have `aria-label`
- [ ] Focus order is logical
- [ ] Keyboard navigation works

### Consistency
- [ ] Uses shadcn/ui components
- [ ] Follows zinc theme colors
- [ ] Spacing uses 4/8/16/24px scale
- [ ] Typography matches style guide

---

## 🚀 Migration Strategy

### For Existing Pages
1. Identify missing patterns using the checklist
2. Replace inline loading with `<LoadingState />`
3. Replace inline empty states with `<EmptyState />`
4. Add error handling with `<ErrorState />`
5. Wrap form fields with `<FormField />`
6. Add responsive classes where needed
7. Run tests to ensure no breaks

### For New Pages
1. Start with page template (see below)
2. Use reusable components from the start
3. Follow checklist during development
4. Add tests before merging

---

## 📄 Page Template

```tsx
import { useState, useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { LoadingState } from "../components/LoadingState";
import { EmptyState } from "../components/EmptyState";
import { ErrorState } from "../components/ErrorState";
import { Package, Plus } from "lucide-react";
import { api } from "../lib/api";

export default function MyPage() {
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await api.getData();
      setData(result);
    } catch (err: any) {
      setError(err.message || "Failed to load data");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  if (loading) {
    return <LoadingState variant="spinner" message="Loading data..." />;
  }

  if (error) {
    return (
      <ErrorState 
        title="Failed to load data"
        message={error}
        onRetry={fetchData}
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">My Page</h1>
          <p className="text-muted-foreground">Page description</p>
        </div>
        <Button>
          <Plus className="h-4 w-4 mr-2" />
          Add Item
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Items ({data.length})</CardTitle>
        </CardHeader>
        <CardContent>
          {data.length === 0 ? (
            <EmptyState 
              icon={Package}
              title="No items yet"
              description="Get started by creating your first item"
              action={<Button>Create Item</Button>}
            />
          ) : (
            // Render data
            <div>{/* table or list */}</div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
```

---

## 📚 References

- [shadcn/ui Documentation](https://ui.shadcn.com/)
- [Tailwind CSS Responsive Design](https://tailwindcss.com/docs/responsive-design)
- [WAI-ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
- [Lucide Icons](https://lucide.dev/)

---

**Questions?** Check existing high-scoring pages (RFQs.tsx, Parts.tsx) for examples.
