# Plan: Modern Web UI with React + Tailwind CSS

## Current State Analysis

### Existing Architecture
- **Backend**: Go with Gin framework
- **Frontend**: Server-side rendered HTML templates with vanilla JavaScript
- **Assets**: Embedded via `go:embed` directive (72KB total)
- **API**: RESTful JSON endpoints already in place
- **Streaming**: SSE (Server-Sent Events) for real-time command output
- **Authentication**: Session-based cookies

### Current Web Pages
1. **login.html** - Authentication page
2. **dashboard.html** - Overview with recent executions
3. **apps.html** - Application listing with CRUD modals
4. **app_detail.html** - App detail with commands management
5. **execute.html** - Command execution with real-time SSE streaming
6. **executions.html** - Execution history
7. **audit_logs.html** - Audit log viewing

### Key Features to Preserve
- Real-time command output via SSE
- Session-based authentication
- Path prefix support for reverse proxy (`/devops`)
- Single binary deployment
- API token display and regeneration
- CRUD operations for apps and commands

## Implementation Approaches

### Approach 1: Full SPA with Vite + React + Tailwind (RECOMMENDED)
**Architecture:**
```
http-remote/
├── cmd/server/main.go          (unchanged)
├── internal/                    (unchanged - backend only)
├── web/                         (NEW - React SPA source)
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── pages/
│   │   │   ├── Login.tsx
│   │   │   ├── Dashboard.tsx
│   │   │   ├── Apps.tsx
│   │   │   ├── AppDetail.tsx
│   │   │   ├── Execute.tsx
│   │   │   ├── Executions.tsx
│   │   │   └── AuditLogs.tsx
│   │   ├── components/
│   │   │   ├── Layout.tsx
│   │   │   ├── Navbar.tsx
│   │   │   ├── Modal.tsx
│   │   │   └── StatusBadge.tsx
│   │   ├── hooks/
│   │   │   ├── useApi.ts
│   │   │   ├── useSSE.ts
│   │   │   └── useAuth.ts
│   │   ├── api/
│   │   │   └── client.ts
│   │   └── types/
│   │       └── index.ts
│   ├── package.json
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── tailwind.config.js
│   └── postcss.config.js
├── internal/assets/web/         (build output - embedded)
│   ├── index.html
│   └── assets/
│       ├── index-[hash].js
│       └── index-[hash].css
└── Makefile                     (updated with build-web target)
```

**Pros:**
- Modern development experience with hot reload
- Type safety with TypeScript
- Component reusability
- Better state management
- Excellent developer tooling
- No page reloads - true SPA experience
- Easy to test with React Testing Library

**Cons:**
- Larger bundle size (~200-300KB gzipped)
- Need Node.js for development
- More complex build process
- Initial learning curve for contributors

**Build Process:**
1. `cd web && npm run build` → outputs to `internal/assets/web/dist/`
2. Backend serves `index.html` for all web routes
3. React Router handles client-side routing
4. `go:embed` includes the build output

**Backend Changes Required:**
- Remove HTML template rendering handlers
- Add catch-all route: `r.NoRoute()` → serve `index.html`
- Keep all API routes unchanged
- Update assets embed path to `web/dist`

---

### Approach 2: Hybrid - Server Routes + Client Hydration
**Architecture:**
- Keep Go templates for initial page load
- Use React for interactive components (modals, forms, streaming)
- Progressive enhancement approach

**Pros:**
- Minimal backend changes
- SEO friendly
- Faster initial page load
- Works without JavaScript

**Cons:**
- Complex setup - two rendering engines
- Code duplication between template data and React state
- Harder to maintain consistency
- Still need build process

---

### Approach 3: Lightweight Modern Stack (Preact + Tailwind)
Same as Approach 1 but with Preact instead of React

**Pros:**
- Same developer experience as React
- Much smaller bundle size (~50-80KB gzipped)
- Faster performance

**Cons:**
- Smaller ecosystem than React
- Some React libraries may not work

---

## Recommended Approach: Full SPA (Approach 1)

### Reasoning
1. **API Already Exists**: All backend logic is already exposed via REST API
2. **SSE Support**: Frontend just needs EventSource client (React-friendly)
3. **Clean Separation**: Backend focuses on business logic, frontend on UX
4. **Modern DX**: Vite provides instant hot reload and optimal production builds
5. **Scalability**: Easy to add features like dark mode, preferences, etc.
6. **Maintainability**: Clear separation of concerns

### Technology Stack
- **Build Tool**: Vite 6.x (fastest, zero-config for React)
- **Framework**: React 18.x with TypeScript
- **Styling**: Tailwind CSS 3.x + shadcn/ui components
- **Routing**: React Router 7.x (client-side)
- **State Management**: Zustand (lightweight) or React Query for server state
- **Forms**: React Hook Form + Zod validation
- **HTTP Client**: Native fetch with custom wrapper
- **SSE**: EventSource with custom React hook
- **Icons**: Lucide React (modern, tree-shakeable)
- **Date Formatting**: date-fns (lightweight)

### Implementation Plan

#### Phase 1: Setup & Infrastructure (Est: 2-3 hours)
**Tasks:**
1. Initialize Vite + React + TypeScript project in `web/`
2. Configure Tailwind CSS with custom theme matching current colors
3. Setup TypeScript types from Go models
4. Create API client with path prefix support
5. Setup React Router with all routes
6. Update Makefile with `build-web` and `dev-web` targets
7. Update `internal/assets/assets.go` embed paths
8. Add catch-all route in Go router to serve SPA

**Files to Create:**
- `web/package.json` - Dependencies
- `web/vite.config.ts` - Build config with `base: '/devops'`
- `web/tailwind.config.js` - Theme config
- `web/tsconfig.json` - TypeScript config
- `web/src/main.tsx` - Entry point
- `web/src/App.tsx` - Root component with router
- `web/src/types/index.ts` - TypeScript types
- `web/src/api/client.ts` - API wrapper

**Backend Changes:**
```go
// internal/router/router.go
// Remove template loading
// Add catch-all route
r.NoRoute(func(c *gin.Context) {
    // Serve index.html for all non-API routes
    if !strings.HasPrefix(c.Request.URL.Path, cfg.Server.PathPrefix+"/api") &&
       !strings.HasPrefix(c.Request.URL.Path, cfg.Server.PathPrefix+"/deploy") {
        c.FileFromFS("index.html", http.FS(staticFS))
    } else {
        c.JSON(404, gin.H{"error": "not found"})
    }
})
```

#### Phase 2: Core Components (Est: 3-4 hours)
**Tasks:**
1. Create Layout component with Navbar
2. Create reusable UI components (Button, Modal, Card, Badge, Input)
3. Implement authentication context and protected routes
4. Create custom hooks: `useApi`, `useAuth`, `useSSE`

**Components:**
- `Layout.tsx` - Main layout with navbar and sidebar
- `Navbar.tsx` - Navigation with active state
- `Modal.tsx` - Reusable modal component
- `StatusBadge.tsx` - Execution status indicator
- `Button.tsx` - Styled button variants
- `Input.tsx` - Form input with validation
- `Card.tsx` - Content container

**Hooks:**
- `useApi.ts` - Fetch wrapper with error handling
- `useAuth.ts` - Authentication state and methods
- `useSSE.ts` - Server-Sent Events subscription
- `usePathPrefix.ts` - Path prefix context

#### Phase 3: Authentication Pages (Est: 2 hours)
**Tasks:**
1. Login page with form validation
2. Protected route wrapper
3. Session management
4. Auto-redirect on unauthorized

**Pages:**
- `Login.tsx` - Login form with validation
- `ProtectedRoute.tsx` - Auth guard component

#### Phase 4: Dashboard & Apps Pages (Est: 4-5 hours)
**Tasks:**
1. Dashboard with stats cards and recent executions table
2. Apps listing page with grid layout
3. Create/Edit/Delete app modals with forms
4. App detail page with commands list
5. Token display and regeneration

**Pages:**
- `Dashboard.tsx` - Overview with metrics
- `Apps.tsx` - Apps grid with CRUD modals
- `AppDetail.tsx` - Single app with commands
- `CreateAppModal.tsx` - App creation form
- `CreateCommandModal.tsx` - Command creation form

#### Phase 5: Execution Features (Est: 4-5 hours)
**Tasks:**
1. Execute page with real-time output streaming
2. SSE integration with React state
3. Execution history page with filtering
4. Output display with syntax highlighting
5. Status indicators and progress

**Pages:**
- `Execute.tsx` - Command execution with SSE stream
- `Executions.tsx` - Execution history table
- `OutputViewer.tsx` - Terminal-like output display

**Features:**
- Real-time streaming via EventSource
- Auto-scroll to bottom
- Status updates
- Exit code display
- Clear output button

#### Phase 6: Audit Logs & Polish (Est: 2-3 hours)
**Tasks:**
1. Audit logs page with table and filters
2. Loading states and skeletons
3. Error boundaries
4. Toast notifications
5. Responsive design refinement
6. Dark mode toggle (optional)

**Pages:**
- `AuditLogs.tsx` - Audit log viewer with filters

**Features:**
- Loading skeletons
- Error boundaries
- Toast notifications (sonner)
- Responsive mobile layout

#### Phase 7: Testing & Documentation (Est: 2-3 hours)
**Tasks:**
1. Update CLAUDE.md with new frontend architecture
2. Add README in `web/` directory
3. Test all CRUD operations
4. Test SSE streaming
5. Test in different browsers
6. Verify embedded assets work
7. Update Makefile help text

### Build Integration

**Makefile Updates:**
```makefile
# Frontend build
build-web:
	@echo "Building web UI..."
	cd web && npm ci && npm run build
	@echo "Web UI built: internal/assets/web/dist/"

# Development with hot reload
dev-web:
	cd web && npm run dev

# Build everything
build-all: build-web build

# Development mode (requires two terminals)
dev:
	@echo "Run these commands in separate terminals:"
	@echo "  Terminal 1: make dev-web"
	@echo "  Terminal 2: make run"
```

**Vite Config:**
```typescript
// web/vite.config.ts
export default defineConfig({
  base: '/devops', // Path prefix
  build: {
    outDir: '../internal/assets/web/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/devops/api': 'http://localhost:8080',
      '/devops/deploy': 'http://localhost:8080',
    }
  }
})
```

### Migration Strategy

**Option A: Big Bang (Recommended for small team)**
1. Keep current templates in `internal/assets/web/templates-old/`
2. Build entire React app
3. Test thoroughly
4. Deploy and switch
5. Remove old templates after successful deployment

**Option B: Gradual (Recommended for production)**
1. Add feature flag in config: `enable_spa: false`
2. Build React app alongside templates
3. Route based on feature flag
4. Test in staging environment
5. Enable gradually (canary deployment)
6. Remove templates after 100% migration

### Dependencies Size Estimate
```json
{
  "dependencies": {
    "react": "^18.3.1",           // ~8KB gzipped
    "react-dom": "^18.3.1",       // ~40KB gzipped
    "react-router-dom": "^7.0.1", // ~12KB gzipped
    "zustand": "^5.0.1",          // ~1KB gzipped
    "@tanstack/react-query": "^5.60.0", // ~15KB gzipped
    "date-fns": "^4.1.0",         // ~10KB gzipped (tree-shakeable)
    "lucide-react": "^0.460.0"    // ~5KB gzipped (tree-shakeable)
  },
  "devDependencies": {
    "vite": "^6.0.0",
    "typescript": "^5.7.0",
    "tailwindcss": "^3.4.0",
    "@vitejs/plugin-react": "^4.3.0"
  }
}
```

**Total Production Bundle:** ~150-200KB gzipped (still smaller than most images!)

### Environment Variables

```bash
# .env for development
VITE_API_BASE_URL=http://localhost:8080
VITE_PATH_PREFIX=/devops
```

### CI/CD Integration

Update GitHub Actions workflow:
```yaml
# .github/workflows/release.yml
- name: Install Node.js
  uses: actions/setup-node@v4
  with:
    node-version: '20'

- name: Build Web UI
  run: make build-web

- name: Build Binary
  run: make build
```

### Compatibility Considerations

1. **Path Prefix**: All routes must respect `pathPrefix` from config
2. **Session Cookies**: React app must send cookies with `credentials: 'include'`
3. **SSE**: EventSource works in all modern browsers
4. **Browser Support**: Target ES2020+ (95%+ browser support)
5. **Single Binary**: All assets embedded via `go:embed`

### Testing Strategy

1. **Unit Tests**: React Testing Library for components
2. **Integration Tests**: Test API calls with MSW (Mock Service Worker)
3. **E2E Tests**: Optional - Playwright for critical flows
4. **Manual Testing**: All CRUD operations and SSE streaming

### Rollback Plan

1. Keep old templates in codebase but commented out
2. Feature flag to switch between old/new UI
3. Quick rollback via config change if issues found
4. No database changes needed (pure frontend change)

### Success Metrics

1. Bundle size < 250KB gzipped ✓
2. Initial page load < 2s ✓
3. All existing features working ✓
4. SSE streaming works flawlessly ✓
5. Mobile responsive ✓
6. No breaking changes to API ✓

## Questions for User

Before proceeding with implementation, I need clarification on:

1. **Bundle Size Priority**: Are you OK with ~200KB gzipped bundle, or do you prefer smaller bundle with Preact?
2. **UI Library**: Do you want to use shadcn/ui components (modern, accessible) or build custom components?
3. **Dark Mode**: Should we include dark mode toggle from the start?
4. **State Management**: Prefer Zustand (simple) or React Query (powerful server state)?
5. **Migration Strategy**: Big bang switch or gradual with feature flag?
6. **Testing**: Should we include unit tests from the start, or add later?
7. **Additional Features**: Any new features you want to add during the rewrite?
   - Bulk operations
   - Execution scheduling
   - Favorites/bookmarks
   - Keyboard shortcuts
   - Command history search

## Estimated Timeline

- **Phase 1**: Setup & Infrastructure - 2-3 hours
- **Phase 2**: Core Components - 3-4 hours
- **Phase 3**: Authentication - 2 hours
- **Phase 4**: Dashboard & Apps - 4-5 hours
- **Phase 5**: Execution Features - 4-5 hours
- **Phase 6**: Audit Logs & Polish - 2-3 hours
- **Phase 7**: Testing & Documentation - 2-3 hours

**Total: 19-25 hours** for complete implementation

## Next Steps

Once you approve this plan and answer the questions, I will:

1. Create detailed file-by-file implementation checklist
2. Set up the React project with all configurations
3. Implement phase by phase with testing at each step
4. Ensure zero breaking changes to backend API
5. Provide migration guide for deployment
