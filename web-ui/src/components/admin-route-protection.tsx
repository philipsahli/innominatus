'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useAuth } from '@/contexts/auth-context';
import { Loader2 } from 'lucide-react';

interface AdminRouteProtectionProps {
  children: React.ReactNode;
}

export function AdminRouteProtection({ children }: AdminRouteProtectionProps) {
  const { isAuthenticated, isAdmin, user } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isAuthenticated && user && !isAdmin) {
      // User is logged in but not an admin, redirect to dashboard
      router.push('/dashboard');
    }
  }, [isAuthenticated, isAdmin, user, router]);

  // Show loading while checking authentication
  if (!user) {
    return (
      <div className="p-6 flex items-center justify-center min-h-[400px]">
        <div className="flex flex-col items-center gap-2">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Verifying access...</p>
        </div>
      </div>
    );
  }

  // User is not an admin, show access denied (will redirect shortly)
  if (!isAdmin) {
    return (
      <div className="p-6">
        <div className="rounded-md bg-destructive/15 p-4 text-destructive">
          <h3 className="font-semibold">Access Denied</h3>
          <p className="text-sm mt-1">
            You do not have permission to access this page. Redirecting...
          </p>
        </div>
      </div>
    );
  }

  // User is an admin, render the protected content
  return <>{children}</>;
}
