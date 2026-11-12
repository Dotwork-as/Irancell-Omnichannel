import { useTranslations } from 'next-intl';

type PhoneArtProps = {
  /** Device body color (product attribute, not a UI palette token). */
  body?: string;
  className?: string;
};

/**
 * Generic vector smartphone illustration used wherever a product image
 * is needed (cards, gallery, cart lines) — keeps the shop free of binary
 * assets while staying on-brand.
 */
export default function PhoneArt({
  body = '#3c3c3d',
  className = '',
}: PhoneArtProps) {
  const t = useTranslations('phoneArt');

  return (
    <svg
      viewBox="0 0 100 200"
      role="img"
      aria-label={t('aria')}
      className={className}
    >
      {/* frame */}
      <rect
        x="4"
        y="2"
        width="92"
        height="196"
        rx="16"
        fill={body}
        stroke="rgba(0,0,0,0.35)"
        strokeWidth="1.5"
      />
      {/* screen */}
      <rect
        x="9"
        y="7"
        width="82"
        height="186"
        rx="12"
        fill="#0c0c0c"
      />
      {/* screen sheen */}
      <path d="M9 100 L91 30 L91 7 a12 12 0 0 0 -12 -0 Z" fill="#ffffff" opacity="0.04" />
      <rect x="9" y="7" width="82" height="186" rx="12" fill="url(#sheen)" />
      {/* dynamic island */}
      <rect x="38" y="13" width="24" height="7" rx="3.5" fill="#000000" />
      {/* side buttons */}
      <rect x="2" y="48" width="2.5" height="18" rx="1.25" fill={body} stroke="rgba(0,0,0,0.25)" strokeWidth="0.5" />
      <rect x="2" y="72" width="2.5" height="30" rx="1.25" fill={body} stroke="rgba(0,0,0,0.25)" strokeWidth="0.5" />
      <defs>
        <linearGradient id="sheen" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#ffffff" stopOpacity="0.08" />
          <stop offset="45%" stopColor="#ffffff" stopOpacity="0" />
          <stop offset="100%" stopColor="#ffffff" stopOpacity="0.03" />
        </linearGradient>
      </defs>
    </svg>
  );
}
