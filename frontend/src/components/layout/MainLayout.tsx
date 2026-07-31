import { Link, Outlet } from "react-router-dom";
import { Navbar } from "./Navbar";
import { TrendingUp } from "lucide-react";
import { Button } from "@/components/ui/button";

export function MainLayout() {
  return (
    <div className="flex min-h-screen flex-col bg-background font-sans text-foreground">
      {/* Skip link: the nav carries six items, so keyboard users need a way
          past it. Visible only when focused. */}
      <a
        href="#main"
        className="sr-only rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-[60]"
      >
        Skip to content
      </a>

      <header className="glass-chrome sticky top-0 z-50 w-full shrink-0 border-b border-border">
        <div className="mx-auto flex max-w-7xl items-center gap-4 px-4 py-3 sm:px-6 lg:px-8">
          <Link
            to="/"
            className="flex shrink-0 items-center gap-2.5 rounded-md focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          >
            <span className="flex size-9 items-center justify-center rounded-lg bg-primary shadow-xs">
              <TrendingUp className="size-5 text-primary-foreground" strokeWidth={2.5} aria-hidden="true" />
            </span>
            <span className="text-lg font-bold tracking-tight">StockMind</span>
          </Link>

          {/* Nav takes the middle and is allowed to shrink and scroll before
              the logo or the actions give up any space. */}
          <div className="min-w-0 flex-1 justify-center hidden md:flex">
            <Navbar />
          </div>

          <div className="ml-auto flex shrink-0 items-center gap-2">
            <Button asChild variant="ghost" size="sm" className="hidden sm:inline-flex">
              <Link to="/login">Login</Link>
            </Button>
            <Button asChild size="sm">
              <Link to="/c">Try StockMind</Link>
            </Button>
          </div>
        </div>

        {/* Below md the nav moves to its own row instead of vanishing. */}
        <div className="border-t border-border px-4 py-2 md:hidden">
          <Navbar />
        </div>
      </header>

      <main id="main" className="flex flex-1 flex-col">
        <Outlet />
      </main>
    </div>
  );
}
