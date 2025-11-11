import { cn } from '@/lib/utils';

export function DataTable({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        'overflow-hidden rounded-lg border border-zinc-200 dark:border-zinc-800',
        className
      )}
    >
      <div className="overflow-x-auto">
        <table className="w-full">{children}</table>
      </div>
    </div>
  );
}

export function DataTableHeader({ children }: { children: React.ReactNode }) {
  return (
    <thead className="border-b border-zinc-200 bg-zinc-50 dark:border-zinc-800 dark:bg-zinc-900">
      <tr>{children}</tr>
    </thead>
  );
}

export function DataTableHeaderCell({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <th
      className={cn(
        'px-4 py-3 text-left text-xs font-medium uppercase tracking-wide text-zinc-600 dark:text-zinc-400',
        className
      )}
    >
      {children}
    </th>
  );
}

export function DataTableBody({ children }: { children: React.ReactNode }) {
  return <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800">{children}</tbody>;
}

export function DataTableRow({
  children,
  onClick,
  className,
}: {
  children: React.ReactNode;
  onClick?: () => void;
  className?: string;
}) {
  return (
    <tr
      onClick={onClick}
      className={cn(onClick && 'cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-900', className)}
    >
      {children}
    </tr>
  );
}

export function DataTableCell({
  children,
  className,
  mono = false,
}: {
  children: React.ReactNode;
  className?: string;
  mono?: boolean;
}) {
  return (
    <td
      className={cn(
        'px-4 py-3 text-sm text-zinc-800 dark:text-zinc-200',
        mono && 'font-mono text-xs',
        className
      )}
    >
      {children}
    </td>
  );
}

// Empty state for tables
export function DataTableEmpty({ message }: { message?: string }) {
  return (
    <tr>
      <td colSpan={100} className="px-4 py-12 text-center text-sm text-zinc-500">
        {message || 'No data available'}
      </td>
    </tr>
  );
}

// Loading state for tables
export function DataTableLoading() {
  return (
    <tr>
      <td colSpan={100} className="px-4 py-12 text-center text-sm text-zinc-500">
        <div className="flex items-center justify-center gap-2">
          <div className="h-4 w-4 animate-spin rounded-full border-2 border-zinc-300 border-t-zinc-600 dark:border-zinc-700 dark:border-t-zinc-400" />
          Loading...
        </div>
      </td>
    </tr>
  );
}
