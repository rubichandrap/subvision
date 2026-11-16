import { cn } from '@/lib/utils';

// An endless ticker. The list is rendered twice so the -50% loop is seamless.
export function Marquee({
  items,
  className,
}: {
  items: string[];
  className?: string;
}) {
  const row = (hidden: boolean) => (
    <ul
      aria-hidden={hidden || undefined}
      className="flex shrink-0 items-center gap-10 pr-10"
    >
      {items.map((item, index) => (
        <li key={`${item}-${index}`} className="flex items-center gap-10">
          <span
            className={cn(
              'font-semibold tracking-[-0.03em]',
              index % 2
                ? 'text-transparent [-webkit-text-stroke:1.5px_var(--foreground)]'
                : 'text-foreground'
            )}
          >
            {item}
          </span>
          <span className="h-2 w-2 rotate-45 bg-primary" />
        </li>
      ))}
    </ul>
  );
  return (
    <div
      className={cn(
        'flex w-full overflow-hidden [mask-image:linear-gradient(90deg,transparent,#000_12%,#000_88%,transparent)]',
        className
      )}
    >
      <div className="flex w-max motion-safe:animate-[marquee_38s_linear_infinite]">
        {row(false)}
        {row(true)}
      </div>
    </div>
  );
}
