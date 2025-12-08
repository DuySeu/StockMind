import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
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
import { CreditCard, EllipsisVertical, LogOut, SquarePen, Trash2, UserCircle } from "lucide-react";

const SideBar = ({ items }: { items: any[] }) => {
  return (
    <Sidebar>
      <SidebarHeader>StockMind</SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Session History</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {items.length > 0 ? (
                items.map((item: any) => (
                  <SidebarMenuItem key={item.id}>
                    <SidebarMenuButton asChild className="flex items-center">
                      <div className="flex w-full items-center justify-between">
                        <div className="flex flex-1 items-center justify-between gap-1 min-w-0">
                          <span className="truncate">{item.title}</span>
                          <span className="text-xs text-muted-foreground flex-shrink-0 group-hover/menu-item:hidden">
                            {new Date(item.updated_at).toLocaleDateString()}
                          </span>
                        </div>
                        <div className="hidden group-hover/menu-item:flex items-center gap-1 px-2 flex-shrink-0">
                          <button
                            className="text-blue-500 cursor-pointer hover:text-blue-600 p-1"
                            // onclick={() => startEditingTitle(thread.id, thread.name)}
                            aria-label="Edit"
                            title="Edit"
                          >
                            <SquarePen className="w-4 h-4" />
                          </button>
                          <button
                            className="text-destructive cursor-pointer hover:text-destructive/80 p-1"
                            // onclick={() => handleDeleteSession(thread.id)}
                            aria-label="Delete"
                            title="Delete"
                          >
                            <Trash2 className="w-4 h-4" />
                          </button>
                        </div>
                      </div>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))
              ) : (
                <SidebarMenuItem>
                  <SidebarMenuButton asChild>
                    <div className="flex w-full items-center justify-between">
                      <div className="flex flex-1 items-center justify-between gap-1 min-w-0">
                        <span className="truncate">No sessions</span>
                      </div>
                    </div>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              )}
            </SidebarMenu>
          </SidebarGroupContent>
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
                className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
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
                  <DropdownMenuItem>
                    <UserCircle />
                    Account
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
