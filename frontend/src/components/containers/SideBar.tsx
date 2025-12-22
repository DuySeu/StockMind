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
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { useChatContext } from "@/hooks/context";
import {
  Bolt,
  Compass,
  CreditCard,
  EllipsisVertical,
  Folder,
  LayoutGrid,
  LogOut,
  Moon,
  Plus,
  SquarePen,
  Sun,
  Trash2,
} from "lucide-react";
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

// Menu items
const menuItems = [
  {
    title: "Explore",
    url: "/explore",
    icon: Compass,
  },
  {
    title: "Categories",
    url: "/categories",
    icon: LayoutGrid,
  },
  {
    title: "Library",
    url: "/library",
    icon: Folder,
  },
  {
    title: "Settings",
    url: "/setting",
    icon: Bolt,
  },
];

const SideBar = () => {
  const navigate = useNavigate();
  const { id } = useParams();
  const { title, setTitle } = useChatContext();
  const [theme, setTheme] = useState<"light" | "dark">("light");
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
    // Check system theme preference
    if (window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches) {
      setTheme("dark");
      document.documentElement.classList.add("dark");
    }
  }, []);

  useEffect(() => {
    if (id) {
      const activeItem = sessions.find((item) => item.id === id);
      if (activeItem) {
        setTitle(activeItem.title);
      }
    }
  }, [id, sessions]);

  const toggleTheme = () => {
    if (theme === "light") {
      setTheme("dark");
      document.documentElement.classList.add("dark");
    } else {
      setTheme("light");
      document.documentElement.classList.remove("dark");
    }
  };

  const handleCreateSession = () => {
    console.log("Create session");
    navigate("/");
  };

  const handleLoadSession = (id: string) => {
    navigate("/c/" + id);
  };

  const handleDeleteSession = async (sessionId: string) => {
    try {
      await deleteSession(sessionId);
      await fetchSession();
      if (id && id === sessionId) {
        navigate("/");
      }
    } catch (error) {
      console.log(error);
    }
  };

  return (
    <Sidebar className="border-none" collapsible="icon">
      <SidebarHeader className="p-3">
        <span>StockMind</span>
        <button
          className="bg-accent text-accent-foreground flex items-center justify-center gap-2 p-3 rounded-xl mx-2 cursor-pointer"
          onClick={handleCreateSession}
        >
          <Plus className="h-6 w-6 rounded-full bg-background text-accent p-1" /> New Chat
        </button>
      </SidebarHeader>
      <SidebarContent className="overflow-hidden">
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu>
              {menuItems.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild>
                    <a href={item.url}>
                      <item.icon />
                      <span>{item.title}</span>
                    </a>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
        <SidebarGroup className="flex-1 min-h-0 p-0">
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
                            onClick={() => handleLoadSession(item.id)}
                          >
                            <span className="truncate">{item.title}</span>
                          </SidebarMenuButton>
                        </ContextMenuTrigger>
                        <ContextMenuContent className="w-56 border border-border">
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
        </SidebarGroup>
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
                  {/* <Avatar className="h-8 w-8 rounded-lg grayscale">
                      <AvatarImage src={user.avatar} alt={user.name} />
                      <AvatarFallback className="rounded-lg">CN</AvatarFallback>
                    </Avatar> */}
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
                    {/* <Avatar className="h-8 w-8 rounded-lg">
                        <AvatarImage src={user.avatar} alt={user.name} />
                        <AvatarFallback className="rounded-lg">CN</AvatarFallback>
                      </Avatar> */}
                    <div className="grid flex-1 text-left text-sm leading-tight">
                      <span className="truncate font-medium">DuySeu</span>
                      <span className="text-muted-foreground truncate text-xs">stockmind@admin.com</span>
                    </div>
                  </div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuGroup>
                  <DropdownMenuItem onClick={toggleTheme}>
                    {theme === "light" ? <Sun /> : <Moon />}
                    Theme
                  </DropdownMenuItem>
                  <DropdownMenuItem>
                    <CreditCard />
                    Billing
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
