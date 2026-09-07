import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Outlet, useNavigate, useRouterState } from "@tanstack/react-router";
import {
  ClipboardList,
  CodeXml,
  Image,
  KeyRound,
  LogOut,
  MapPin,
  Moon,
  RefreshCw,
  Sun,
  UserRound,
  Users,
} from "lucide-react";
import { useEffect, useState } from "react";

import { Link } from "@components/link";
import { Logo } from "@components/logo";
import { Button } from "@components/ui/button";
import { Separator } from "@components/ui/separator";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
  useSidebar,
} from "@components/ui/sidebar";
import { toast } from "@components/ui/toast";
import { canAccess, usePermissions } from "@features/auth/access";
import { authApi } from "@features/auth/api";
import { ApiError } from "@lib/api";
import { runtime } from "@lib/runtime";

export const navigation = [
  { resource: "locations", label: "Locations", icon: MapPin },
  { resource: "checkins", label: "Check-ins", icon: ClipboardList },
  { resource: "users", label: "Users", icon: UserRound },
  { resource: "groups", label: "Groups", icon: Users },
  { resource: "assets", label: "Assets", icon: Image },
  { resource: "api-keys", label: "API Keys", icon: KeyRound },
] as const;
function Navigation() {
  const permissions = usePermissions();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const { setOpenMobile } = useSidebar();
  return (
    <SidebarMenu>
      {navigation
        .filter((entry) => canAccess(permissions, entry.resource, "read"))
        .map(({ resource, label, icon: Icon }) => (
          <SidebarMenuItem key={resource}>
            <SidebarMenuButton
              render={<Link to={`/${resource}`} />}
              isActive={pathname.startsWith(`/${resource}`)}
              onClick={() => setOpenMobile(false)}
            >
              <Icon />
              <span>{label}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        ))}
    </SidebarMenu>
  );
}
export function AppLayout() {
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const user = useQuery({
    queryKey: ["auth", "user"],
    queryFn: ({ signal }) => authApi.getUser(signal),
    staleTime: 30_000,
  });
  const [loggingOut, setLoggingOut] = useState(false);
  const [dark, setDark] = useState(() => {
    const saved = localStorage.getItem("woodgate.theme");
    return saved ? saved === "dark" : matchMedia("(prefers-color-scheme: dark)").matches;
  });
  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark);
    localStorage.setItem("woodgate.theme", dark ? "dark" : "light");
  }, [dark]);
  const logout = async () => {
    setLoggingOut(true);
    try {
      try {
        await authApi.logout();
      } catch (error) {
        if (!(error instanceof ApiError && (error.status === 401 || error.status === 403)))
          throw error;
      }
      queryClient.clear();
      await navigate({ to: "/login", replace: true });
    } catch (error) {
      toast.add({
        title: "Unable to sign out",
        description: error instanceof Error ? error.message : undefined,
        type: "error",
      });
    } finally {
      setLoggingOut(false);
    }
  };
  return (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader>
          <Link to="/" className="flex items-center gap-3 p-2">
            <Logo className="size-8" />
            <span className="font-semibold">Woodgate</span>
          </Link>
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <Navigation />
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter>
          <span className="truncate px-2 text-sm">
            {user.data?.name || user.data?.email || "Signed In"}
          </span>
          <Button
            variant="ghost"
            className="justify-start"
            disabled={loggingOut}
            onClick={() => void logout()}
          >
            <LogOut data-icon="inline-start" />
            Sign Out
          </Button>
          {runtime.version ? (
            <span className="px-2 text-xs text-muted-foreground">{runtime.version}</span>
          ) : null}
        </SidebarFooter>
      </Sidebar>
      <SidebarInset>
        <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
          <SidebarTrigger />
          <Separator orientation="vertical" className="h-5" />
          <span className="text-sm text-muted-foreground">Administration</span>
          <div className="ml-auto flex gap-1">
            <Button
              nativeButton={false}
              variant="ghost"
              size="icon-sm"
              aria-label="View source on GitHub"
              render={
                <a
                  aria-label="View source on GitHub"
                  href="https://github.com/woodleighschool/woodgate"
                  target="_blank"
                  rel="noreferrer"
                />
              }
            >
              <CodeXml />
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label="Refresh"
              onClick={() => void queryClient.invalidateQueries()}
            >
              <RefreshCw />
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={dark ? "Use light theme" : "Use dark theme"}
              onClick={() => setDark((value) => !value)}
            >
              {dark ? <Sun /> : <Moon />}
            </Button>
          </div>
        </header>
        <Outlet />
      </SidebarInset>
    </SidebarProvider>
  );
}
