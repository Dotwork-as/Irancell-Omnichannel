'use client';

import Link from 'next/link';
import { motion } from 'framer-motion';
import { ChevronLeft } from 'lucide-react';

type SectionHeadingProps = {
  title: string;
  description?: string;
  actionLabel?: string;
  actionHref?: string;
};

const revealProps = {
  initial: { opacity: 0, y: 24 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true, amount: 0.2 },
  transition: { duration: 0.5, ease: 'easeOut' as const },
};

export default function SectionHeading({
  title,
  description,
  actionLabel,
  actionHref,
}: SectionHeadingProps) {
  return (
    <motion.div {...revealProps} className="flex flex-col gap-2">
      <div className="flex items-end justify-between gap-4">
        <h2 className="text-xl font-bold text-ink sm:text-2xl">{title}</h2>

        {actionLabel && actionHref && (
          <Link
            href={actionHref}
            className="flex items-center gap-1 text-sm font-medium text-coal transition-colors hover:text-brand"
          >
            {actionLabel}
            <ChevronLeft size={16} />
          </Link>
        )}
      </div>

      {description && <p className="text-sm text-ash">{description}</p>}
    </motion.div>
  );
}
