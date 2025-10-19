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
  Bot,
  Database,
  GitBranch,
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

interface NavSection {
  title?: string;
  items: NavItem[];
}

// User Navigation organized in sections
const userNavigationSections: NavSection[] = [
  {
    title: 'Overview',
    items: [
      {
        href: '/dashboard',
        label: 'Dashboard',
        icon: Home,
      },
    ],
  },
  {
    title: 'Platform',
    items: [
      {
        href: '/apps',
        label: 'Applications',
        icon: Package,
      },
      {
        href: '/resources',
        label: 'Resources',
        icon: Database,
      },
      {
        href: '/graph',
        label: 'Dependency Graph',
        icon: Network,
      },
    ],
  },
  {
    title: 'Workflows',
    items: [
      {
        href: '/workflows',
        label: 'Executions',
        icon: Activity,
      },
      {
        href: '/workflows/analyze',
        label: 'Analysis',
        icon: FileSearch,
      },
      {
        href: '/goldenpaths',
        label: 'Golden Paths',
        icon: GitBranch,
      },
    ],
  },
  {
    title: 'Tools',
    items: [
      {
        href: '/ai-assistant',
        label: 'AI Assistant',
        icon: Bot,
      },
      {
        href: '/demo',
        label: 'Demo Environment',
        icon: Monitor,
      },
    ],
  },
  {
    title: 'Help',
    items: [
      {
        href: '/docs',
        label: 'Documentation',
        icon: BookOpen,
      },
      {
        href: 'http://localhost:8081/swagger-user',
        label: 'API Docs',
        icon: FileText,
        external: true,
      },
    ],
  },
];

// Admin Navigation sections
const adminNavigationSections: NavSection[] = [
  {
    title: 'Administration',
    items: [
      {
        href: '/admin/users',
        label: 'Users',
        icon: Users,
        adminOnly: true,
      },
      {
        href: '/admin/teams',
        label: 'Teams',
        icon: UsersRound,
        adminOnly: true,
      },
      {
        href: '/admin/secrets',
        label: 'Secrets',
        icon: Key,
        adminOnly: true,
      },
      {
        href: '/admin/audit',
        label: 'Audit Logs',
        icon: FileSearch,
        adminOnly: true,
      },
    ],
  },
  {
    title: 'System',
    items: [
      {
        href: '/admin/system',
        label: 'Health',
        icon: Server,
        adminOnly: true,
      },
      {
        href: '/admin/settings',
        label: 'Settings',
        icon: Settings,
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
        label: 'Admin API',
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
  const { logout, isAdmin } = useAuth();

  const navigationSections = isAdmin
    ? [...userNavigationSections, ...adminNavigationSections]
    : userNavigationSections;

  const NavContent = () => (
    <div className="h-full flex flex-col bg-gray-800 text-white dark:bg-gray-900">
      <div className="p-6 border-b border-gray-700 dark:border-gray-600">
        <h1 className="text-xl font-bold">innominatus</h1>
        <p className="text-xs text-gray-400 mt-1">IDP Orchestrator</p>
      </div>

      <nav className="flex-1 p-4 overflow-y-auto">
        <div className="space-y-6">
          {navigationSections.map((section, sectionIndex) => (
            <div key={section.title || `section-${sectionIndex}`}>
              {section.title && (
                <h3 className="px-3 mb-2 text-xs font-semibold text-gray-400 uppercase tracking-wider">
                  {section.title}
                </h3>
              )}
              <ul className="space-y-1">
                {section.items.map((item) => {
                  const Icon = item.icon;
                  const isActive = pathname === item.href;

                  return (
                    <li key={item.label}>
                      {item.external ? (
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
            </div>
          ))}
        </div>
      </nav>

      <div className="p-4 border-t border-gray-700 dark:border-gray-600 space-y-2">
        <Link href="/profile">
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              'w-full justify-start text-gray-100 hover:text-white hover:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600',
              pathname === '/profile' && 'bg-gray-700 text-white dark:bg-gray-600'
            )}
          >
            <User className="w-4 h-4 mr-2" />
            Profile
          </Button>
        </Link>
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
