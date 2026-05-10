import { createRootRoute, Outlet } from "@tanstack/react-router";

export const Route = createRootRoute({
  component: () => (
    <div className="min-h-screen bg-[#0a0a0a] text-gray-100">
      <header className="sticky top-0 z-10 border-b border-white/5 bg-[#0a0a0a]/90 backdrop-blur-sm">
        <div className="max-w-7xl mx-auto px-6 h-12 flex items-center gap-3">
          <div className="flex items-center gap-2.5">
            <div className="w-5 h-5 bg-white rounded-sm flex items-center justify-center shrink-0">
              <svg width="10" height="10" viewBox="0 0 10 10" fill="none">
                <path d="M5 1L9.33 8.5H0.67L5 1Z" fill="#0a0a0a" />
              </svg>
            </div>
            <span className="text-sm font-semibold tracking-tight">
              mini-brimble
            </span>
          </div>
          <span className="text-[10px] font-medium text-gray-600 border border-white/8 rounded px-1.5 py-0.5 leading-none">
            alpha
          </span>
        </div>
      </header>
      <Outlet />
    </div>
  ),
});
