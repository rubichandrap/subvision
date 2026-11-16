import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';

import { cn } from '@/lib/utils';

const buttonVariants = cva(
  'relative inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-full text-sm font-medium tracking-tight transition-all duration-200 ease-out focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background active:scale-[0.97] disabled:pointer-events-none disabled:opacity-45 [&_svg]:pointer-events-none [&_svg]:size-4 [&_svg]:shrink-0',
  {
    variants: {
      variant: {
        default:
          'bg-brand text-primary-foreground shadow-[0_8px_24px_-8px_rgb(124_108_255/0.7),inset_0_1px_0_rgb(255_255_255/0.28)] hover:-translate-y-px hover:shadow-[0_14px_34px_-8px_rgb(124_108_255/0.85),inset_0_1px_0_rgb(255_255_255/0.32)] hover:brightness-110',
        destructive:
          'bg-destructive text-destructive-foreground hover:brightness-110',
        outline:
          'border border-input bg-foreground/[0.03] text-foreground backdrop-blur hover:bg-foreground/[0.08] hover:border-foreground/25',
        secondary:
          'bg-secondary text-secondary-foreground hover:bg-accent',
        ghost:
          'text-muted-foreground hover:bg-foreground/[0.07] hover:text-foreground',
        link: 'rounded-none text-foreground underline decoration-primary decoration-2 underline-offset-4 hover:decoration-4 active:scale-100',
      },
      size: {
        default: 'h-10 px-5 py-2',
        sm: 'h-9 px-4',
        lg: 'h-12 px-8 text-base',
        icon: 'h-10 w-10',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button';
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    );
  }
);
Button.displayName = 'Button';

export { Button, buttonVariants };
