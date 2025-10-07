'use client';

import { useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { useAuth } from '@/contexts/auth-context';

export default function OIDCCallbackPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { login } = useAuth();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Get token from query parameters
    const token = searchParams.get('token');
    const errorParam = searchParams.get('error');

    if (errorParam) {
      // Handle error from backend
      setError(`Authentication failed: ${errorParam}`);
      setTimeout(() => {
        router.push('/login');
      }, 3000);
      return;
    }

    if (!token) {
      setError('No authentication token received');
      setTimeout(() => {
        router.push('/login');
      }, 3000);
      return;
    }

    // Store token and redirect to dashboard
    try {
      login(token);
      router.push('/dashboard');
    } catch (err) {
      setError('Failed to complete authentication');
      setTimeout(() => {
        router.push('/login');
      }, 3000);
    }
  }, [searchParams, login, router]);

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 via-blue-50 to-slate-50 dark:from-slate-900 dark:via-slate-800 dark:to-slate-900">
        <div className="text-center">
          <div className="w-16 h-16 border-4 border-red-500/30 border-t-red-500 rounded-full animate-spin mx-auto mb-4"></div>
          <p className="text-red-600 dark:text-red-400 text-lg">{error}</p>
          <p className="text-sm text-muted-foreground mt-2">Redirecting to login...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-slate-50 via-blue-50 to-slate-50 dark:from-slate-900 dark:via-slate-800 dark:to-slate-900">
      <div className="text-center">
        <div className="w-16 h-16 border-4 border-blue-500/30 border-t-blue-500 rounded-full animate-spin mx-auto mb-4"></div>
        <p className="text-lg font-medium">Completing authentication...</p>
        <p className="text-sm text-muted-foreground mt-2">Please wait</p>
      </div>
    </div>
  );
}
