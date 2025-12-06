import { HTMLAttributes } from 'react';
import { cn } from '@/lib/utils';
import { ExecutionStatus } from '@/types';

interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  status?: ExecutionStatus;
  variant?: 'default' | 'success' | 'error' | 'warning' | 'info';
}

export default function Badge({ status, variant, children, className, ...props }: BadgeProps) {
  let badgeVariant = variant;

  if (status) {
    switch (status) {
      case 'success':
        badgeVariant = 'success';
        break;
      case 'failed':
        badgeVariant = 'error';
        break;
      case 'running':
        badgeVariant = 'info';
        break;
      case 'pending':
        badgeVariant = 'warning';
        break;
    }
  }

  const variants = {
    default: 'bg-gray-100 text-gray-800 border-gray-200',
    success: 'bg-green-100 text-green-800 border-green-200',
    error: 'bg-red-100 text-red-800 border-red-200',
    warning: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    info: 'bg-blue-100 text-blue-800 border-blue-200',
  };

  return (
    <span
      className={cn(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border',
        variants[badgeVariant || 'default'],
        className
      )}
      {...props}
    >
      {children || status}
    </span>
  );
}
