// AuthProvider was consolidated into providers.tsx AuthGuard.
// This file is kept for backward compatibility but the export is a no-op.
// The actual auth logic lives in src/app/providers.tsx.

export function AuthProvider({ children }: { children: React.ReactNode }) {
  return <>{children}</>
}
