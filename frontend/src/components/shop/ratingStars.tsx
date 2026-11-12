import { useTranslations } from 'next-intl';
import { Star } from 'lucide-react';

type RatingStarsProps = {
  value: number;
  size?: number;
  className?: string;
};

export default function RatingStars({
  value,
  size = 14,
  className = '',
}: RatingStarsProps) {
  const t = useTranslations('rating');

  return (
    <span
      className={`flex items-center gap-0.5 ${className}`}
      aria-label={t('aria', { value })}
    >
      {[1, 2, 3, 4, 5].map((star) => (
        <Star
          key={star}
          size={size}
          className={
            star <= Math.round(value)
              ? 'fill-brand text-brand'
              : 'fill-line text-line'
          }
        />
      ))}
    </span>
  );
}
