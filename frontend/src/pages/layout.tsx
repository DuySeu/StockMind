import SideBar from "@/components/containers/SideBar";
import { SidebarProvider } from "@/components/ui/sidebar";
import { ChatProvider } from "@/hooks/context";
import { Outlet } from "react-router-dom";
import { Toaster } from "@/components/ui/sonner";

const Layout = () => {
  return (
    <ChatProvider>
      <SidebarProvider className="bg-background h-svh overflow-hidden">
        <SideBar />
        <main className="w-full h-[calc(100svh-1.5rem)] flex flex-col bg-card border border-border transition-colors duration-300 rounded-2xl m-3 ml-0 overflow-hidden">
          <Outlet />
        </main>
      </SidebarProvider>
      <Toaster position="top-right" />
    </ChatProvider>
  );
};

export default Layout;
