import { Link, useLocation } from "react-router-dom";

export function Navbar() {
  const location = useLocation();
  const navItems = [
    { label: "Home", path: "/" },
    { label: "Chatbot", path: "/c" },
    { label: "Watchlist", path: "/watchlist" },
    { label: "Research", path: "/research" },
    { label: "News", path: "/news" },
  ];

  return (
    <nav className="hidden md:flex items-center gap-8">
      {navItems.map((item) => {
        const isActive =
          location.pathname === item.path || (item.path !== "/" && location.pathname.startsWith(item.path));
        return (
          <Link
            key={item.label}
            to={item.path}
            className={`text-sm font-semibold transition-colors ${
              isActive ? "text-primary" : "text-muted-foreground hover:text-primary"
            }`}
          >
            {item.label}
          </Link>
        );
      })}
    </nav>
  );
}
