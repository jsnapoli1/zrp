# ZRP React Frontend

Modern React frontend for ZRP (Zero Research Platform) built with TypeScript, Vite, and shadcn/ui.

## Tech Stack

- **React 19** with TypeScript
- **Vite 7** for build tooling
- **shadcn/ui** for UI components
- **Tailwind CSS v3** for styling  
- **React Router** for client-side routing
- **Lucide React** for icons

## Development

```bash
# Install dependencies
npm install

# Start development server (with API proxy)
npm run dev

# Build for production
npm run build

# Preview production build
npm run preview
```

## Architecture

- `src/components/ui/` - shadcn/ui components
- `src/layouts/` - App layout with sidebar navigation
- `src/pages/` - Page components (one per module)
- `src/lib/api.ts` - Typed API client for ZRP backend
- `src/lib/utils.ts` - Utility functions

## API Integration

The frontend automatically proxies `/api/*` requests to the Go backend on `http://localhost:9000` during development. In production, the Go server serves the React build.

## Module Structure

The app is organized around ZRP's main functional areas:

- **Engineering**: Parts, ECOs, Documents, Testing
- **Supply Chain**: Vendors, Purchase Orders, Procurement  
- **Manufacturing**: Work Orders, Inventory, NCRs
- **Field & Service**: RMAs, Field Reports
- **Sales**: Quotes, Pricing
- **Reports**: Analytics, Calendar
- **Admin**: Users, Settings

## Dark Mode

Dark mode is supported out of the box via shadcn/ui's built-in theme system.

## Next Steps

This is the foundation. Individual module pages can now be built on top of this structure, using the typed API client and shadcn/ui components for consistency.