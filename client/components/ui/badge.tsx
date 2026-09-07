import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';

import { cn } from '@/lib/utils';

const badgeVariants = cva(
  'inline-flex items-center gap-1 rounded-none border-2 border-border px-2.5 py-0.5 font-mono text-xs font-bold uppercase tracking-wide shadow-brutal-sm transition-all duration-100 focus:outline-none focus:ring-[3px] focus:ring-ring focus:ring-offset-0',
  {
    variants: {
      variant: {
        default:
          'bg-primary text-primary-foreground hover:bg-primary',
        secondary:
          'bg-secondary text-secondary-foreground hover:bg-secondary',
        destructive:
          'bg-destructive text-destructive-foreground hover:bg-destructive',
        outline: 'bg-background text-foreground',
      },
    },
    defaultVariants: {
      variant: 'default',
    },
  }
);

export interface BadgeProps
  extends React.HTMLAttributes<HTMLDivElement>,
    VariantProps<typeof badgeVariants> {}

function Badge({ className, variant, ...props }: BadgeProps) {
  return (
    <div className={cn(badgeVariants({ variant }), className)} {...props} />
  );
}

export { Badge, badgeVariants };
