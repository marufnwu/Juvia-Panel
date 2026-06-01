# Panel UI

Next.js 14 frontend for Server Panel PaaS.

## Tech Stack

- **Framework:** Next.js 14 (App Router)
- **Language:** TypeScript 5.3+
- **Styling:** Tailwind CSS 3.4+
- **Components:** shadcn/ui
- **State:** Zustand (global), TanStack Query (server)
- **Icons:** Lucide React
- **Charts:** Recharts

## Development

```bash
npm install
npm run dev
```

## Build

```bash
npm run build
```

Output is static export to `/out` directory.

## Design Tokens

See `ui-ux-specification.md` for full design system.

| Token | Value |
|-------|-------|
| Primary | `#2563EB` |
| Success | `#16A34A` |
| Warning | `#D97706` |
| Danger | `#DC2626` |
| Background | `#0F172A` |
| Surface | `#1E293B` |
| Border | `#334155` |
| Text | `#F8FAFC` |

## Directory Structure

```
src/
├── app/          # Next.js App Router pages
├── components/   # React components
├── lib/          # Utilities and API client
├── stores/       # Zustand state stores
└── types/        # TypeScript definitions