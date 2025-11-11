import Link from 'next/link';
import { User, Home, Package, Database, GitBranch, Network, MessageSquare } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu';

export default function DevLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-white dark:bg-zinc-950">
      {/* Top Navigation */}
      <nav className="border-b border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-950">
        <div className="mx-auto max-w-screen-2xl px-6">
          <div className="flex h-14 items-center justify-between">
            {/* Logo + Nav Links */}
            <div className="flex items-center gap-8">
              <Link href="/dev" className="text-sm font-semibold text-zinc-900 dark:text-white">
                innominatus<span className="ml-2 text-xs text-zinc-500">/dev</span>
              </Link>

              <div className="flex items-center gap-6">
                <NavLink href="/dev" icon={<Home size={14} />}>
                  Home
                </NavLink>
                <NavLink href="/dev/applications" icon={<Package size={14} />}>
                  Applications
                </NavLink>
                <NavLink href="/dev/resources" icon={<Database size={14} />}>
                  Resources
                </NavLink>
                <NavLink href="/dev/workflows" icon={<GitBranch size={14} />}>
                  Workflows
                </NavLink>
                <NavLink href="/dev/graph" icon={<Network size={14} />}>
                  Graph
                </NavLink>
                <NavLink href="/dev/assistant" icon={<MessageSquare size={14} />}>
                  Assistant
                </NavLink>
              </div>
            </div>

            {/* Right Side: Profile Dropdown */}
            <div className="flex items-center gap-4">
              <Link
                href="/"
                className="text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
              >
                Full UI →
              </Link>
              <DropdownMenu>
                <DropdownMenuTrigger className="flex h-8 w-8 items-center justify-center rounded-full bg-zinc-100 hover:bg-zinc-200 dark:bg-zinc-800 dark:hover:bg-zinc-700">
                  <User size={16} className="text-zinc-600 dark:text-zinc-400" />
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-48">
                  <DropdownMenuItem asChild>
                    <Link href="/profile">Profile & API Keys</Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem asChild>
                    <Link href="/api/auth/logout">Logout</Link>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </div>
      </nav>

      {/* Main Content */}
      <main className="mx-auto max-w-screen-2xl px-6 py-8">{children}</main>
    </div>
  );
}

function NavLink({
  href,
  icon,
  children,
}: {
  href: string;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      className="flex items-center gap-1.5 text-sm text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-white transition-colors"
    >
      {icon}
      {children}
    </Link>
  );
}
