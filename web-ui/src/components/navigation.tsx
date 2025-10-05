'use client';

import React, { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import { Button } from '@/components/ui/button';
import {
  Home,
  Network,
  Users,
  Settings,
  Moon,
  Sun,
  Menu,
  LogOut,
  Monitor,
  Activity,
  Package,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  FileText,
  BookOpen,
  User,
  Shield,
  Key,
  FileSearch,
  Server,
  Boxes,
  Plug,
  UsersRound,
} from 'lucide-react';
import { useTheme } from 'next-themes';
import { Sheet, SheetContent, SheetTrigger } from '@/components/ui/sheet';
import { useAuth } from '@/contexts/auth-context';

interface NavSubItem {
  href: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  external?: boolean;
  adminOnly?: boolean;
}

interface NavItem {
  href?: string;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  children?: NavSubItem[];
  adminOnly?: boolean;
  external?: boolean;
}

// Normal User Navigation
const userNavigation: NavItem[] = [
  {
    href: '/dashboard',
    label: 'Dashboard',
    icon: Home,
  },
  {
    href: '/apps',
    label: 'Applications',
    icon: Package,
  },
  {
    label: 'Workflows',
    icon: Activity,
    children: [
      {
        href: '/workflows',
        label: 'Workflow Executions',
        icon: Activity,
      },
      {
        href: '/workflows/analyze',
        label: 'Workflow Analysis',
        icon: Network,
      },
    ],
  },
  {
    href: '/graph',
    label: 'Graphs',
    icon: Network,
  },
  {
    href: '/goldenpaths',
    label: 'Golden Paths',
    icon: Activity,
  },
  {
    href: '/demo',
    label: 'Demo Environment',
    icon: Monitor,
  },
  {
    href: '/profile',
    label: 'Profile',
    icon: User,
  },
  {
    href: 'http://localhost:8081/swagger-user',
    label: 'API Docs',
    icon: FileText,
    external: true,
  },
];

// Admin Navigation (extends user navigation)
const adminNavigation: NavItem[] = [
  {
    label: 'Admin',
    icon: Shield,
    adminOnly: true,
    children: [
      {
        href: '/admin/users',
        label: 'User Management',
        icon: Users,
        adminOnly: true,
      },
      {
        href: '/admin/teams',
        label: 'Team Management',
        icon: UsersRound,
        adminOnly: true,
      },
      {
        href: '/admin/secrets',
        label: 'Secrets & Vault',
        icon: Key,
        adminOnly: true,
      },
      {
        href: '/admin/audit',
        label: 'Audit Logs',
        icon: FileSearch,
        adminOnly: true,
      },
      {
        href: '/admin/settings',
        label: 'Settings',
        icon: Settings,
        adminOnly: true,
      },
      {
        href: '/admin/system',
        label: 'System Health',
        icon: Server,
        adminOnly: true,
      },
      {
        href: '/admin/graph',
        label: 'Graph Config',
        icon: Boxes,
        adminOnly: true,
      },
      {
        href: '/admin/integrations',
        label: 'Integrations',
        icon: Plug,
        adminOnly: true,
      },
      {
        href: 'http://localhost:8081/swagger-admin',
        label: 'Admin API Docs',
        icon: FileText,
        external: true,
        adminOnly: true,
      },
    ],
  },
];

export function Navigation() {
  const pathname = usePathname();
  const { theme, setTheme } = useTheme();
  const { logout } = useAuth();
  const [expandedItems, setExpandedItems] = useState<string[]>([]);

  // TODO: Get user role from API/session - for now, assume admin for demo
  const isAdmin = true;

  const navigation = isAdmin ? [...userNavigation, ...adminNavigation] : userNavigation;

  const toggleExpanded = (label: string) => {
    setExpandedItems((prev) =>
      prev.includes(label) ? prev.filter((item) => item !== label) : [...prev, label]
    );
  };

  const isExpanded = (label: string) => expandedItems.includes(label);

  const NavContent = () => (
    <div className="h-full flex flex-col bg-gray-800 text-white dark:bg-gray-900">
      <div className="p-6 border-b border-gray-700 dark:border-gray-600">
        <h1 className="text-xl font-bold">innominatus</h1>
        <p className="text-xs text-gray-400 mt-1">IDP Orchestrator</p>
      </div>

      <nav className="flex-1 p-4 overflow-y-auto">
        <ul className="space-y-2">
          {navigation.map((item) => {
            const Icon = item.icon;
            const isActive = pathname === item.href;
            const hasChildren = item.children && item.children.length > 0;
            const expanded = isExpanded(item.label);

            return (
              <li key={item.label}>
                {hasChildren ? (
                  <>
                    <button
                      onClick={() => toggleExpanded(item.label)}
                      className={cn(
                        'flex items-center justify-between w-full gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all hover:bg-gray-700 dark:hover:bg-gray-600',
                        item.adminOnly
                          ? 'text-amber-400 hover:text-amber-300'
                          : 'text-gray-100 hover:text-white dark:text-gray-200'
                      )}
                    >
                      <div className="flex items-center gap-3">
                        <Icon className="w-4 h-4" />
                        {item.label}
                      </div>
                      {expanded ? (
                        <ChevronDown className="w-4 h-4" />
                      ) : (
                        <ChevronRight className="w-4 h-4" />
                      )}
                    </button>
                    {expanded && item.children && (
                      <ul className="ml-4 mt-2 space-y-1">
                        {item.children.map((subItem) => {
                          const SubIcon = subItem.icon;
                          const isSubActive = pathname === subItem.href;

                          return (
                            <li key={subItem.href}>
                              {subItem.external ? (
                                <a
                                  href={subItem.href}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className={cn(
                                    'flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all hover:bg-gray-700 dark:hover:bg-gray-600',
                                    subItem.adminOnly
                                      ? 'text-amber-400 hover:text-amber-300'
                                      : 'text-gray-100 hover:text-white dark:text-gray-200'
                                  )}
                                >
                                  <SubIcon className="w-4 h-4" />
                                  {subItem.label}
                                  <ExternalLink className="w-3 h-3 ml-auto" />
                                </a>
                              ) : (
                                <Link
                                  href={subItem.href}
                                  className={cn(
                                    'flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all hover:bg-gray-700 dark:hover:bg-gray-600',
                                    isSubActive
                                      ? subItem.adminOnly
                                        ? 'bg-amber-700 text-white dark:bg-amber-600'
                                        : 'bg-gray-700 text-white dark:bg-gray-600'
                                      : subItem.adminOnly
                                        ? 'text-amber-400 hover:text-amber-300'
                                        : 'text-gray-100 hover:text-white dark:text-gray-200'
                                  )}
                                >
                                  <SubIcon className="w-4 h-4" />
                                  {subItem.label}
                                </Link>
                              )}
                            </li>
                          );
                        })}
                      </ul>
                    )}
                  </>
                ) : item.external ? (
                  <a
                    href={item.href!}
                    target="_blank"
                    rel="noopener noreferrer"
                    className={cn(
                      'flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all hover:bg-gray-700 dark:hover:bg-gray-600',
                      item.adminOnly
                        ? 'text-amber-400 hover:text-amber-300'
                        : 'text-gray-100 hover:text-white dark:text-gray-200'
                    )}
                  >
                    <Icon className="w-4 h-4" />
                    {item.label}
                    <ExternalLink className="w-3 h-3 ml-auto" />
                  </a>
                ) : (
                  <Link
                    href={item.href!}
                    className={cn(
                      'flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all hover:bg-gray-700 dark:hover:bg-gray-600',
                      isActive
                        ? item.adminOnly
                          ? 'bg-amber-700 text-white dark:bg-amber-600'
                          : 'bg-gray-700 text-white dark:bg-gray-600'
                        : item.adminOnly
                          ? 'text-amber-400 hover:text-amber-300'
                          : 'text-gray-100 hover:text-white dark:text-gray-200'
                    )}
                  >
                    <Icon className="w-4 h-4" />
                    {item.label}
                  </Link>
                )}
              </li>
            );
          })}
        </ul>
      </nav>

      <div className="p-4 border-t border-gray-700 dark:border-gray-600 space-y-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
          className="w-full justify-start text-gray-100 hover:text-white hover:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600"
        >
          {theme === 'dark' ? <Sun className="w-4 h-4 mr-2" /> : <Moon className="w-4 h-4 mr-2" />}
          Toggle theme
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={logout}
          className="w-full justify-start text-gray-100 hover:text-white hover:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600"
        >
          <LogOut className="w-4 h-4 mr-2" />
          Logout
        </Button>
      </div>
    </div>
  );

  return (
    <>
      {/* Desktop sidebar */}
      <div className="hidden md:block w-64 h-screen sticky top-0">
        <NavContent />
      </div>

      {/* Mobile sidebar */}
      <div className="md:hidden">
        <div className="fixed top-4 left-4 z-50">
          <Sheet>
            <SheetTrigger asChild>
              <Button variant="outline" size="icon">
                <Menu className="h-4 w-4" />
              </Button>
            </SheetTrigger>
            <SheetContent side="left" className="p-0 w-64">
              <NavContent />
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </>
  );
}
