import {
  Activity,
  Ban,
  BellRing,
  Bot,
  Brain,
  Bug,
  ClipboardList,
  FolderOpen,
  FolderSync,
  LayoutDashboard,
  type LucideIcon,
  MessageSquare,
  Network,
  Plug,
  Radio,
  ScrollText,
  Settings2,
  ShieldAlert,
  Sparkles,
  Target,
  Terminal,
  Wrench,
} from "lucide-react";

export type NavBadge = "new" | "soon";

export interface NavSubItem {
  id: string;
  title: string;
  url: string;
  icon?: LucideIcon;
  badge?: NavBadge;
  disabled?: boolean;
  newTab?: boolean;
}

interface NavItemBase {
  id: string;
  title: string;
  icon?: LucideIcon;
  badge?: NavBadge;
  disabled?: boolean;
  newTab?: boolean;
}

export interface NavMainLinkItem extends NavItemBase {
  url: string;
  subItems?: never;
}

export interface NavMainParentItem extends NavItemBase {
  subItems: NavSubItem[];
}

export type NavMainItem = NavMainLinkItem | NavMainParentItem;

export interface NavGroup {
  id: number;
  label?: string;
  items: NavMainItem[];
}

export const sidebarItems: NavGroup[] = [
  {
    id: 1,
    label: "",
    items: [
      { id: "dashboard", title: "", url: "/dashboard", icon: LayoutDashboard },
      { id: "chat", title: "", url: "/chat", icon: MessageSquare },
      { id: "tasks", title: "", url: "/function/tasks", icon: Target },
      { id: "findings", title: "", url: "/function/findings", icon: Bug },
      { id: "traffic", title: "", url: "/function/traffic", icon: Activity },
      { id: "commands", title: "", url: "/function/commands", icon: Terminal },
      { id: "llm-records", title: "LLM ", url: "/function/llm-records", icon: Radio },
      { id: "assets", title: "", url: "/function/assets", icon: Network },
      { id: "sync", title: "", url: "/function/sync", icon: FolderSync },
      { id: "workspace", title: "", url: "/function/workspace", icon: FolderOpen },
    ],
  },
  {
    id: 2,
    label: "",
    items: [
      { id: "llm", title: "LLM", url: "/system/llm", icon: Brain },
      { id: "agents", title: "Agent", url: "/system/agents", icon: Bot },
      { id: "mcp", title: "MCP", url: "/system/mcp", icon: Plug },
      { id: "skills", title: "Skill", url: "/system/skills", icon: Sparkles },
      { id: "tools", title: "", url: "/system/tools", icon: Wrench },
      { id: "notify", title: "", url: "/system/notify", icon: BellRing },
      { id: "intercept", title: "", url: "/system/intercept", icon: ShieldAlert },
      { id: "asset-intercept", title: "", url: "/system/intercept/assets", icon: Ban },
      { id: "approvals", title: "", url: "/system/intercept/approvals", icon: ClipboardList },
      { id: "logs", title: "", url: "/system/logs", icon: ScrollText },
      { id: "settings", title: "", url: "/system/settings", icon: Settings2 },
    ],
  },
];
