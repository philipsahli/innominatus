'use client';

import { useAuth } from '@/contexts/auth-context';
import { usePathname } from 'next/navigation';
import { Navigation } from '@/components/navigation';
import { ImpersonationBanner } from '@/components/impersonation-banner';

interface LayoutContentProps {
  children: React.ReactNode;
}

export function LayoutContent({ children }: LayoutContentProps) {
  const { isAuthenticated } = useAuth();
  const pathname = usePathname();

  // Don't show navigation on login page or if not authenticated
  const shouldShowNavigation = isAuthenticated && pathname !== '/login';

  return (
    <div className="flex min-h-screen">
      {shouldShowNavigation && <Navigation />}
      <main className={shouldShowNavigation ? 'flex-1 overflow-auto' : 'w-full overflow-auto'}>
        {shouldShowNavigation && (
          <div className="p-4">
            <ImpersonationBanner />
          </div>
        )}
        {children}
      </main>
    </div>
  );
}
