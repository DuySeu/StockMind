import { Link, Outlet } from "react-router-dom";
import { Navbar } from "./Navbar";
import { TrendingUp } from "lucide-react";
import { Button } from "@/components/ui/button";

export function MainLayout() {
  return (
    <div className="min-h-screen flex flex-col bg-background text-foreground font-sans transition-colors">
      <header className="sticky top-0 z-50 w-full border-b border-primary/20 bg-background/80 backdrop-blur-md px-6 lg:px-20 py-4 shrink-0">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          {/* Logo */}
          <Link to="/" className="flex items-center gap-2">
            <div className="size-10 bg-primary rounded-lg flex items-center justify-center">
              <TrendingUp className="size-5 text-primary-foreground" strokeWidth={3} />
            </div>
            <h2 className="text-2xl font-black tracking-tight text-foreground">StockMind</h2>
          </Link>

          {/* Nav */}
          <Navbar />

          {/* Actions */}
          <div className="flex items-center gap-4">
            <Link to="/login" className="text-sm font-semibold hover:text-primary transition-colors hidden sm:block">
              Login
            </Link>
            <Button asChild className="rounded-xl px-6 py-2.5 font-bold text-sm shadow-lg shadow-primary/20">
              <Link to="/c">Try StockMind</Link>
            </Button>
          </div>
        </div>
      </header>
      <main className="flex-1 flex flex-col">
        <Outlet />
      </main>
    </div>
  );
}
