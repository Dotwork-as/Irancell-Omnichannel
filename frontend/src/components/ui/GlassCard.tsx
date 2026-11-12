import React from 'react';

type GlassCardProps = {
  title?: string;
  description?: string;
  className?: string;
  children: React.ReactNode;
};

export default function GlassCard({
  title,
  description,
  className,
  children,
}: GlassCardProps) {
  return (
    <section
      className={`rounded-2xl border border-line bg-paper shadow-lg shadow-ink/5 p-4 md:p-6 ${className ?? ''}`}
    >
      {(title || description) && (
        <header className="mb-4 flex items-start justify-between gap-2">
          <div>
            {title && (
              <h2 className="text-base md:text-lg font-semibold text-ink">
                {title}
              </h2>
            )}
            {description && (
              <p className="mt-1 text-xs md:text-sm text-ash">{description}</p>
            )}
          </div>
        </header>
      )}
      <div>{children}</div>
    </section>
  );
}
