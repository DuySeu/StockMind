import { getSession } from "@/api/sessions";
import SideBar from "@/components/containers/SideBar";
import { SidebarProvider } from "@/components/ui/sidebar";
import { useEffect, useState } from "react";
import { Outlet } from "react-router-dom";

type Session = {
  id: string;
  title: string;
  updated_at: string;
  [key: string]: unknown;
};

const Layout = () => {
  const [sessions, setSessions] = useState<Session[]>([]);

  useEffect(() => {
    getSession().then((data) => {
      setSessions(data as Session[]);
    });
  }, []);

  return (
    <SidebarProvider className="bg-background">
      <SideBar items={sessions} />
      <main className="w-full h-[calc(screen-6rem)] flex flex-col bg-card border border-border transition-colors duration-300 rounded-2xl m-3 ml-0">
        <Outlet />
      </main>
    </SidebarProvider>
  );
};

export default Layout;
