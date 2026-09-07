'use client';

import * as React from 'react';
import { Moon, Sun } from 'lucide-react';
import { useTheme } from 'next-themes';

import { Button } from '@/components/ui/button';

export function ThemeToggle() {
  const { resolvedTheme, setTheme } = useTheme();
  const [mounted, setMounted] = React.useState(false);

  React.useEffect(() => {
    setMounted(true);
  }, []);

  return (
    <Button
      variant="ghost"
      size="icon"
      aria-label="Toggle theme"
      className="h-9 w-9 rounded-none border-2 border-border bg-background text-foreground shadow-brutal-sm transition-all duration-100 hover:-translate-x-0.5 hover:-translate-y-0.5 hover:bg-accent hover:text-accent-foreground hover:shadow-brutal focus-visible:ring-[3px] active:translate-x-0.5 active:translate-y-0.5 active:shadow-none"
      onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}
    >
      {mounted && resolvedTheme === 'dark' ? (
        <Sun className="h-4 w-4 transition-transform duration-200 rotate-0 hover:rotate-45" />
      ) : (
        <Moon className="h-4 w-4 transition-transform duration-200 rotate-0 hover:-rotate-12" />
      )}
    </Button>
  );
}
