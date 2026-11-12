'use client';

import { motion } from 'framer-motion';

type SummaryRow = {
  label: string;
  value: string;
  accent?: boolean;
};

type OrderSummaryProps = {
  title: string;
  rows: SummaryRow[];
  totalLabel: string;
  total: string;
  children?: React.ReactNode;
};

const revealProps = {
  initial: { opacity: 0, y: 24 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true, amount: 0.2 },
  transition: { duration: 0.5, ease: 'easeOut' as const },
};

export default function OrderSummary({
  title,
  rows,
  totalLabel,
  total,
  children,
}: OrderSummaryProps) {
  return (
    <motion.aside
      {...revealProps}
      transition={{ duration: 0.5, ease: 'easeOut', delay: 0.15 }}
      className="flex h-fit flex-col gap-4 rounded-2xl border border-line bg-paper p-6"
    >
      <h2 className="text-base font-bold text-ink">{title}</h2>

      <div className="flex flex-col gap-3">
        {rows.map(({ label, value, accent }) => (
          <div key={label} className="flex items-center justify-between gap-4">
            <span className="text-sm text-ash">{label}</span>
            <span
              className={`text-sm font-medium ${accent ? 'text-brand' : 'text-ink'}`}
            >
              {value}
            </span>
          </div>
        ))}
      </div>

      <div className="flex items-center justify-between gap-4 border-t border-line pt-4">
        <span className="text-sm font-bold text-ink">{totalLabel}</span>
        <span className="text-lg font-extrabold text-ink">{total}</span>
      </div>

      {children}
    </motion.aside>
  );
}
