'use client';

import { AdminRouteProtection } from '@/components/admin-route-protection';
import { AdminImpersonation } from '@/components/admin-impersonation';

export default function AdminImpersonatePage() {
  return (
    <AdminRouteProtection>
      <div className="container mx-auto p-6">
        <div className="mb-6">
          <h1 className="text-3xl font-bold">User Impersonation</h1>
          <p className="text-muted-foreground mt-2">
            Impersonate users to troubleshoot issues or view their experience
          </p>
        </div>

        <div className="max-w-2xl">
          <AdminImpersonation />
        </div>
      </div>
    </AdminRouteProtection>
  );
}
