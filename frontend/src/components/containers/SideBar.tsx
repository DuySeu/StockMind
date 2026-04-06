import { deleteSession, getSessions } from "@/api/sessions";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { Bolt, EllipsisVertical, LogOut, Moon, Plus, SquarePen, Sun, Trash2, TrendingUp } from "lucide-react";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuLabel,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";

type Session = {
  id: string;
  title: string;
  updated_at: string;
  [key: string]: unknown;
};

const SideBar = ({ title, setTitle }: { title: string; setTitle: (title: string) => void }) => {
  const navigate = useNavigate();
  const { id } = useParams();
  const [sessions, setSessions] = useState<Session[]>([]);

  const fetchSession = async () => {
    try {
      const response = await getSessions();
      setSessions(response as Session[]);
    } catch (error) {
      console.log(error);
    }
  };
  useEffect(() => {
    fetchSession();
  }, [title]);

  useEffect(() => {
    if (id) {
      const activeItem = sessions.find((item) => item.id === id);
      if (activeItem) {
        setTitle(activeItem.title);
      }
    }
  }, [id, sessions]);

  // Theme management
  const toggleTheme = () => {
    if (typeof document !== "undefined") {
      const isDark = document.documentElement.classList.toggle("dark");
      localStorage.setItem("theme", isDark ? "dark" : "light");
    }
  };

  // Initialize theme on mount
  useEffect(() => {
    if (typeof window !== "undefined") {
      const savedTheme = localStorage.getItem("theme");
      const prefersDark = window.matchMedia("(prefers-color-scheme: dark)").matches;

      if (savedTheme === "dark" || (!savedTheme && prefersDark)) {
        document.documentElement.classList.add("dark");
      } else {
        document.documentElement.classList.remove("dark");
      }
    }
  }, []);

  const handleDeleteSession = async (sessionId: string) => {
    try {
      await deleteSession(sessionId);
      await fetchSession();
      if (id && id === sessionId) {
        navigate("/c");
      }
    } catch (error) {
      console.log(error);
    }
  };

  return (
    <Sidebar className="border-none" collapsible="icon">
      <SidebarHeader className="p-3">
        <div className="flex items-center gap-2">
          <div className="size-10 bg-primary rounded-lg flex items-center justify-center">
            <TrendingUp className="size-5 text-primary-foreground" strokeWidth={3} />
          </div>
          <h2 className="text-2xl font-black tracking-tight">StockMind</h2>
        </div>
        <button
          className="w-full bg-primary hover:bg-primary/90 text-background-dark font-bold py-3 px-4 rounded-full flex items-center justify-center gap-2 shadow-sm transition-all"
          onClick={() => navigate("/c")}
        >
          <Plus className="h-6 w-6" /> New Chat
        </button>
      </SidebarHeader>
      <SidebarContent className="overflow-hidden">
        <SidebarGroupLabel>Chats</SidebarGroupLabel>
        <ScrollArea className="h-full overflow-y-auto">
          <SidebarGroupContent className="flex-1 min-h-0 flex flex-col px-2">
            <SidebarMenu className="flex-1 min-h-0 flex flex-col">
              {sessions.length > 0 ? (
                sessions.map((item: any) => (
                  <SidebarMenuItem key={item.id}>
                    <ContextMenu>
                      <ContextMenuTrigger asChild>
                        <SidebarMenuButton
                          className={`${id === item.id && "border-l-3 border-accent"}`}
                          onClick={() => navigate(`/c/${item.id}`)}
                        >
                          <span className="truncate max-w-56">{item.title}</span>
                        </SidebarMenuButton>
                      </ContextMenuTrigger>
                      <ContextMenuContent className="w-40 border border-border">
                        <ContextMenuLabel className="text-xs font-normal text-muted-foreground">
                          Created at {new Date(item.updated_at).toLocaleDateString()}
                        </ContextMenuLabel>
                        <ContextMenuSeparator />
                        <ContextMenuItem>
                          <SquarePen className="h-4 w-4" />
                          Rename
                        </ContextMenuItem>
                        <ContextMenuItem variant="destructive" onClick={() => handleDeleteSession(item.id)}>
                          <Trash2 className="h-4 w-4" />
                          Delete
                        </ContextMenuItem>
                      </ContextMenuContent>
                    </ContextMenu>
                  </SidebarMenuItem>
                ))
              ) : (
                <SidebarMenuItem>
                  <div className="flex w-full items-center justify-between">
                    <div className="flex flex-1 items-center justify-between gap-1 min-w-0">
                      <span className="truncate">No sessions</span>
                    </div>
                  </div>
                </SidebarMenuItem>
              )}
            </SidebarMenu>
          </SidebarGroupContent>
        </ScrollArea>
      </SidebarContent>
      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton
                  size="lg"
                  className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                >
                  <div className="grid flex-1 text-left text-sm leading-tight">
                    <span className="truncate font-medium">DuySeu</span>
                    <span className="text-muted-foreground truncate text-xs">stockmind@admin.com</span>
                  </div>
                  <EllipsisVertical />
                </SidebarMenuButton>
              </DropdownMenuTrigger>
              <DropdownMenuContent
                className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg border border-border"
                side={"right"}
                align="end"
                sideOffset={4}
              >
                <DropdownMenuLabel className="p-0 font-normal">
                  <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                    <div className="grid flex-1 text-left text-sm leading-tight">
                      <span className="truncate font-medium">DuySeu</span>
                      <span className="text-muted-foreground truncate text-xs">stockmind@admin.com</span>
                    </div>
                  </div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuGroup>
                  <DropdownMenuItem onClick={toggleTheme}>
                    <Sun className="dark:hidden" />
                    <Moon className="hidden dark:block" />
                    Theme
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => navigate("/settings")}>
                    <Bolt />
                    Settings
                  </DropdownMenuItem>
                </DropdownMenuGroup>
                <DropdownMenuSeparator />
                <DropdownMenuItem>
                  <LogOut />
                  Log out
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
};

export default SideBar;
