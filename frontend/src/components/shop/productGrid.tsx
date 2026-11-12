'use client';

import { motion } from 'framer-motion';

import ProductCard, { type ProductCardProps } from './productCard';

type ProductGridProps = {
  items: ProductCardProps[];
};

const revealProps = {
  initial: { opacity: 0, y: 24 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true, amount: 0.2 },
  transition: { duration: 0.5, ease: 'easeOut' as const },
};

export default function ProductGrid({ items }: ProductGridProps) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {items.map((product, index) => (
        <motion.div
          key={product.slug}
          {...revealProps}
          transition={{
            duration: 0.5,
            ease: 'easeOut',
            delay: Math.min(index * 0.08, 0.4),
          }}
        >
          <ProductCard {...product} />
        </motion.div>
      ))}
    </div>
  );
}
