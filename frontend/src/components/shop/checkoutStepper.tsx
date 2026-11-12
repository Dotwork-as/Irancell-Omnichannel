'use client';

import { useLocale, useTranslations } from 'next-intl';
import { Check } from 'lucide-react';

type CheckoutStepperProps = {
  steps: string[];
  current: number;
};

export default function CheckoutStepper({
  steps,
  current,
}: CheckoutStepperProps) {
  const t = useTranslations('checkout');
  const locale = useLocale();

  return (
    <ol
      className="flex items-center gap-2"
      aria-label={t('stepperLabel')}
    >
      {steps.map((label, index) => {
        const isDone = index < current;
        const isActive = index === current;

        return (
          <li key={label} className="flex flex-1 items-center gap-2">
            <div className="flex items-center gap-2">
              <span
                aria-hidden={!isActive}
                aria-current={isActive ? 'step' : undefined}
                className={`
                  flex h-8 w-8 shrink-0 items-center justify-center rounded-full
                  text-xs font-bold transition-colors
                  ${
                    isDone
                      ? 'bg-brand text-ink'
                      : isActive
                        ? 'bg-ink text-paper'
                        : 'bg-smoke text-ash'
                  }
                `}
              >
                {isDone ? (
                  <Check size={14} />
                ) : (
                  (index + 1).toLocaleString(locale)
                )}
              </span>
              <span
                className={`
                  hidden text-xs font-medium sm:block
                  ${isActive ? 'text-ink' : 'text-ash'}
                `}
              >
                {label}
              </span>
            </div>

            {index < steps.length - 1 && (
              <span
                className={`
                  h-0 flex-1 border-t-2 border-dashed transition-colors
                  ${isDone ? 'border-brand' : 'border-line'}
                `}
              />
            )}
          </li>
        );
      })}
    </ol>
  );
}
