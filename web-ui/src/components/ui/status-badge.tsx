import { cn } from "@/lib/utils"
import { type VariantProps, cva } from "class-variance-authority"

const statusBadgeVariants = cva(
  "inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-full border",
  {
    variants: {
      variant: {
        success: "bg-green-100 text-green-800 border border-green-200 dark:bg-green-900/30 dark:text-green-300 dark:border-green-700",
        warning: "bg-amber-100 text-amber-800 border border-amber-200 dark:bg-amber-900/30 dark:text-amber-300 dark:border-amber-700",
        error: "bg-red-100 text-red-800 border border-red-200 dark:bg-red-900/30 dark:text-red-300 dark:border-red-700",
        info: "bg-blue-100 text-blue-800 border border-blue-200 dark:bg-blue-900/30 dark:text-blue-300 dark:border-blue-700",
        pending: "bg-gray-100 text-gray-600 border border-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700",
        // Workflow-specific variants
        running: "workflow-running",
        completed: "workflow-completed",
        failed: "workflow-failed",
        // Resource-specific variants
        active: "bg-green-100 text-green-800 border border-green-200 dark:bg-green-900/30 dark:text-green-300 dark:border-green-700",
        provisioning: "bg-blue-100 text-blue-800 border border-blue-200 dark:bg-blue-900/30 dark:text-blue-300 dark:border-blue-700",
        degraded: "bg-amber-100 text-amber-800 border border-amber-200 dark:bg-amber-900/30 dark:text-amber-300 dark:border-amber-700",
        terminated: "bg-gray-100 text-gray-600 border border-gray-200 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-700",
      },
      size: {
        sm: "px-2 py-0.5 text-xs",
        md: "px-2.5 py-1 text-xs",
        lg: "px-3 py-1.5 text-sm",
      },
    },
    defaultVariants: {
      variant: "info",
      size: "md",
    },
  }
)

export interface StatusBadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof statusBadgeVariants> {
  icon?: React.ReactNode
}

export function StatusBadge({
  className,
  variant,
  size,
  icon,
  children,
  ...props
}: StatusBadgeProps) {
  return (
    <div
      className={cn(statusBadgeVariants({ variant, size }), className)}
      {...props}
    >
      {icon}
      {children}
    </div>
  )
}

// Utility component for workflow status
export function WorkflowStatusBadge({
  status,
  icon,
  size = "md"
}: {
  status: "running" | "completed" | "failed" | "pending"
  icon?: React.ReactNode
  size?: "sm" | "md" | "lg"
}) {
  const getStatusVariant = (status: string) => {
    switch (status) {
      case "running": return "running"
      case "completed": return "completed"
      case "failed": return "failed"
      case "pending": return "pending"
      default: return "pending"
    }
  }

  return (
    <StatusBadge
      variant={getStatusVariant(status) as any}
      size={size}
      icon={icon}
    >
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </StatusBadge>
  )
}

// Utility component for resource status
export function ResourceStatusBadge({
  status,
  icon,
  size = "md"
}: {
  status: "active" | "provisioning" | "degraded" | "terminated"
  icon?: React.ReactNode
  size?: "sm" | "md" | "lg"
}) {
  const getStatusVariant = (status: string) => {
    switch (status) {
      case "active": return "active"
      case "provisioning": return "provisioning"
      case "degraded": return "degraded"
      case "terminated": return "terminated"
      default: return "pending"
    }
  }

  return (
    <StatusBadge
      variant={getStatusVariant(status) as any}
      size={size}
      icon={icon}
    >
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </StatusBadge>
  )
}