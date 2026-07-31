import { Link, useLocation } from "react-router-dom";
import { Home, Library, MessageSquareText, Newspaper, Star, Telescope } from "lucide-react";

const navItems = [
  { label: "Home", path: "/", icon: Home },
  { label: "Chatbot", path: "/c", icon: MessageSquareText },
  { label: "Watchlist", path: "/watchlist", icon: Star },
  { label: "Research", path: "/research", icon: Telescope },
  { label: "Knowledge", path: "/documents", icon: Library },
  { label: "News", path: "/news", icon: Newspaper },
];

export function Navbar() {
  const location = useLocation();

  return (
    // Scrolls rather than disappearing below md — the previous `hidden md:flex`
    // left small screens with no navigation at all.
    <nav
      aria-label="Main"
      className="flex items-center gap-1 overflow-x-auto rounded-lg border border-border bg-secondary/50 p-1 backdrop-blur-sm [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
    >
      {navItems.map((item) => {
        const isActive =
          location.pathname === item.path || (item.path !== "/" && location.pathname.startsWith(item.path));
        const Icon = item.icon;
        return (
          <Link
            key={item.label}
            to={item.path}
            aria-current={isActive ? "page" : undefined}
            className={`flex min-h-9 shrink-0 items-center gap-2 rounded-md px-3 text-sm font-medium transition-colors ${
              isActive
                ? "bg-card-solid text-foreground shadow-xs"
                : "text-muted-foreground hover:bg-card/60 hover:text-foreground"
            }`}
          >
            <Icon className="size-4 shrink-0" aria-hidden="true" />
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
