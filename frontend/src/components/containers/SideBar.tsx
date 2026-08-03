import { deleteSession, errorMessage, getSessions, updateSessionTitle } from "@/api/sessions";
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
import { useEffect, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";
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
  created_at: Date;
};

const SideBar = ({ setTitle, sessionVersion }: { setTitle: (title: string) => void; sessionVersion?: number }) => {
  const navigate = useNavigate();
  const { id } = useParams();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [renamingId, setRenamingId] = useState<string | null>(null);
  const [draft, setDraft] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const renameInputRef = useRef<HTMLInputElement>(null);

  const fetchSession = async () => {
    try {
      const response = await getSessions();
      setSessions(response as Session[]);
    } catch (error) {
      console.log(error);
    }
  };

  // Fetch on mount and when explicitly signalled via sessionVersion
  useEffect(() => {
    fetchSession();
  }, [sessionVersion]);

  useEffect(() => {
    if (id) {
      const activeItem = sessions.find((item) => item.id === id);
      if (activeItem) {
        setTitle(activeItem.title);
      }
    }
  }, [id, sessions, setTitle]);

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

  const startRename = (session: Session) => {
    setRenamingId(session.id);
    setDraft(session.title);
  };

  const cancelRename = () => {
    setRenamingId(null);
    setDraft("");
  };

  const commitRename = async (sessionId: string) => {
    if (isSaving) return;
    const next = draft.trim();
    const previous = sessions.find((s) => s.id === sessionId)?.title;
    if (!next || next === previous) {
      cancelRename();
      return;
    }

    setIsSaving(true);
    // Optimistic: the row updates immediately and rolls back if the PATCH fails.
    setSessions((prev) => prev.map((s) => (s.id === sessionId ? { ...s, title: next } : s)));
    try {
      const saved = await updateSessionTitle(sessionId, next);
      setSessions((prev) => prev.map((s) => (s.id === sessionId ? { ...s, title: saved } : s)));
      // Keep the header in step when renaming the conversation that's open.
      if (id === sessionId) setTitle(saved);
      cancelRename();
    } catch (error) {
      setSessions((prev) =>
        prev.map((s) => (s.id === sessionId && previous !== undefined ? { ...s, title: previous } : s)),
      );
      toast.error(errorMessage(error, "Could not rename this conversation"));
      renameInputRef.current?.focus();
    } finally {
      setIsSaving(false);
    }
  };

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
      <SidebarHeader className="gap-3 p-3">
        <div className="flex items-center gap-2.5">
          <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-sidebar-primary">
            <TrendingUp className="size-5 text-sidebar-primary-foreground" strokeWidth={2.5} aria-hidden="true" />
          </span>
          <span className="truncate text-lg font-bold tracking-tight group-data-[collapsible=icon]:hidden">
            StockMind
          </span>
        </div>
        {/* The one filled control on the rail. `text-background-dark` was a
            token that does not exist, so this button had no text colour at all. */}
        <button
          className="flex min-h-10 w-full items-center justify-center gap-2 rounded-lg bg-sidebar-primary px-4 text-sm font-semibold text-sidebar-primary-foreground shadow-xs transition-colors hover:bg-sidebar-primary/90 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sidebar-ring"
          onClick={() => navigate("/c")}
        >
          <Plus className="size-4 shrink-0" aria-hidden="true" />
          <span className="group-data-[collapsible=icon]:hidden">New Chat</span>
        </button>
      </SidebarHeader>
      <SidebarContent className="overflow-hidden">
        <SidebarGroupLabel>Chats</SidebarGroupLabel>
        <ScrollArea className="h-full overflow-y-auto">
          <SidebarGroupContent className="flex-1 min-h-0 flex flex-col px-2">
            <SidebarMenu className="flex-1 min-h-0 flex flex-col">
              {sessions.length > 0 ? (
                sessions.map((item: Session) =>
                  renamingId === item.id ? (
                    // Renaming in place, so the row you right-clicked is the row
                    // you edit — no dialog, no navigation.
                    <SidebarMenuItem key={item.id}>
                      <input
                        ref={renameInputRef}
                        value={draft}
                        onChange={(e) => setDraft(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter" && !e.nativeEvent.isComposing) {
                            e.preventDefault();
                            void commitRename(item.id);
                          } else if (e.key === "Escape") {
                            e.preventDefault();
                            cancelRename();
                          }
                        }}
                        onBlur={() => void commitRename(item.id)}
                        disabled={isSaving}
                        maxLength={120}
                        aria-label={`Rename ${item.title}`}
                        autoFocus
                        className="ml-2 h-8 w-[calc(100%-0.5rem)] rounded-md border border-sidebar-primary bg-sidebar px-2 text-sm text-sidebar-foreground outline-none disabled:opacity-60"
                      />
                    </SidebarMenuItem>
                  ) : (
                    <SidebarMenuItem key={item.id}>
                      <ContextMenu>
                        <ContextMenuTrigger asChild>
                          {/* Active session is marked by a solid left rule plus a
                              filled row, not colour alone. */}
                          <SidebarMenuButton
                            isActive={id === item.id}
                            aria-current={id === item.id ? "page" : undefined}
                            className={
                              id === item.id
                                ? "border-l-2 border-sidebar-primary pl-2 font-medium"
                                : "border-l-2 border-transparent pl-2"
                            }
                            onClick={() => navigate(`/c/${item.id}`)}
                          >
                            <span className="truncate">{item.title}</span>
                          </SidebarMenuButton>
                        </ContextMenuTrigger>
                        <ContextMenuContent className="w-40 border border-border">
                          <ContextMenuLabel className="text-xs font-normal text-muted-foreground">
                            Created at {new Date(item.created_at).toLocaleDateString()}
                          </ContextMenuLabel>
                          <ContextMenuSeparator />
                          <ContextMenuItem onSelect={() => startRename(item)}>
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
                  ),
                )
              ) : (
                <SidebarMenuItem>
                  {/* `text-muted-foreground` is a light-mode grey and sits at
                      roughly 3:1 on the dark rail — the sidebar has its own
                      foreground token for exactly this. */}
                  <p className="px-2 py-1.5 text-sm text-sidebar-foreground/60 group-data-[collapsible=icon]:hidden">
                    No conversations yet
                  </p>
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
                    <span className="truncate text-xs text-sidebar-foreground/60">stockmind@admin.com</span>
                  </div>
                  <EllipsisVertical aria-hidden="true" />
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
