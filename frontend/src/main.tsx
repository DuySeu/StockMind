import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "react-router-dom";
import { ThemeProvider } from "next-themes";
import "./index.css";
import { router } from "./router.tsx";

// index.css carries a full `.dark` palette but nothing ever set the class, so
// every dark token was unreachable. next-themes was already a dependency —
// only the Toaster imported it — and now puts the class on <html>.
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ThemeProvider attribute="class" defaultTheme="system" enableSystem disableTransitionOnChange>
      <RouterProvider router={router} />
    </ThemeProvider>
  </StrictMode>,
);
