'use client';

import Link from 'next/link';
import { useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { AnimatePresence, motion } from 'framer-motion';
import { useLocale, useTranslations } from 'next-intl';

import CheckoutStepper from '@/src/components/shop/checkoutStepper';
import EmptyState from '@/src/components/layout/emptyState';
import OrderSummary from '@/src/components/shop/orderSummary';
import { SkeletonBlock } from '@/src/components/skeleton/skeletonBlock';

import Button from '@/src/components/ui/Button';
import Input from '@/src/components/ui/Field';
import { useCart } from '@/src/hooks/useCart';
import { useCheckout } from '@/src/hooks/useCheckout';
import { useShippingMethods } from '@/src/hooks/useShippingMethods';
import { formatToman } from '@/src/lib/format';

const revealProps = {
  initial: { opacity: 0, y: 24 },
  whileInView: { opacity: 1, y: 0 },
  viewport: { once: true, amount: 0.2 },
  transition: { duration: 0.5, ease: 'easeOut' as const },
};

type FieldErrors = Record<string, string>;

const PHONE_PATTERN = /^09\d{9}$/;

/** Client-side mirror of the §4.4 rules — the backend remains the
 * source of truth and server details are mapped onto fields too. */
function validateRecipient(
  recipient: {
    firstName: string;
    lastName: string;
    phone: string;
    address: string;
    shippingMethodId?: string;
  },
  t: ReturnType<typeof useTranslations>,
): FieldErrors {
  const errors: FieldErrors = {};
  const first = recipient.firstName.trim();
  const last = recipient.lastName.trim();
  const phone = recipient.phone.trim();
  const address = recipient.address.trim();

  if (first === '') errors['recipient.firstName'] = t('validation.required');
  else if (first.length < 2 || first.length > 60) errors['recipient.firstName'] = t('validation.nameLength');

  if (last === '') errors['recipient.lastName'] = t('validation.required');
  else if (last.length < 2 || last.length > 60) errors['recipient.lastName'] = t('validation.nameLength');

  if (phone === '') errors['recipient.phone'] = t('validation.required');
  else if (!PHONE_PATTERN.test(phone)) errors['recipient.phone'] = t('validation.invalidPhone');

  if (address === '') errors['recipient.address'] = t('validation.required');
  else if (address.length < 10 || address.length > 500) errors['recipient.address'] = t('validation.addressLength');

  if ((recipient.shippingMethodId ?? '').trim() === '') {
    errors['shippingMethodId'] = t('validation.required');
  }
  return errors;
}

export default function CheckoutFlow() {
  const t = useTranslations('checkout');
  const tc = useTranslations('common');
  const locale = useLocale() as 'fa' | 'en';
  const router = useRouter();

  const { cart, loading: cartLoading } = useCart();
  const {
    methods,
    loading: shippingLoading,
    error: shippingError,
    refetch: refetchShipping,
  } = useShippingMethods();
  const { submitCheckout, pending: checkoutPending, error: checkoutError } = useCheckout();

  const [step, setStep] = useState(0);
  const [recipient, setRecipient] = useState({
    firstName: '',
    lastName: '',
    phone: '',
    address: '',
  });
  const [shippingMethodId, setShippingMethodId] = useState('');
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({});

  const steps = t.raw('steps') as string[];

  const selectedShipping = useMemo(
    () => methods.find((m) => m.id === shippingMethodId) ?? null,
    [methods, shippingMethodId],
  );

  const setField = (field: keyof typeof recipient) => (value: string) => {
    setRecipient((prev) => ({ ...prev, [field]: value }));
    setFieldErrors((prev) => {
      if (!prev[`recipient.${field}`]) return prev;
      const next = { ...prev };
      delete next[`recipient.${field}`];
      return next;
    });
  };

  const isEmpty = !cartLoading && (cart?.items.length ?? 0) === 0;

  const goNext = () => {
    if (step === 0) {
      const errors = validateRecipient({ ...recipient, shippingMethodId }, t);
      const recipientErrors = Object.fromEntries(
        Object.entries(errors).filter(([field]) => field.startsWith('recipient.')),
      );
      setFieldErrors(recipientErrors);
      if (Object.keys(recipientErrors).length === 0) setStep(1);
      return;
    }
    if (step === 1) {
      if (shippingMethodId === '') {
        setFieldErrors({ shippingMethodId: t('validation.required') });
        return;
      }
      setFieldErrors({});
      setStep(2);
    }
  };

  const submit = async () => {
    const receipt = await submitCheckout({
      recipient,
      shippingMethodId,
    });
    if (receipt) {
      router.push('/checkout/success');
    }
  };

  // Server-side VALIDATION_ERROR details are derived (not stored):
  // they render on their fields and never produce an empty banner.
  const serverFieldErrors = useMemo(() => {
    const out: FieldErrors = {};
    if (checkoutError) {
      for (const d of checkoutError.validationDetails()) out[d.field] = d.message;
    }
    return out;
  }, [checkoutError]);

  const fieldError = (field: string) =>
    fieldErrors[field] ?? serverFieldErrors[field] ?? null;

  if (cartLoading) {
    return (
      <main className="mx-auto w-full max-w-6xl px-4 pt-8 pb-10 sm:px-6">
        <SkeletonBlock className="h-8 w-40" />
        <SkeletonBlock className="mt-8 h-96 rounded-2xl" />
      </main>
    );
  }

  if (isEmpty) {
    return (
      <main className="mx-auto w-full max-w-6xl px-4 pt-16 pb-10 sm:px-6">
        <EmptyState
          message={t('emptyCart')}
          description={t('emptyCartDescription')}
          action={
            <Link href="/products" className="inline-block">
              <Button variant="brand">{t('browse')}</Button>
            </Link>
          }
        />
      </main>
    );
  }

  const subtotal = cart?.subtotalIrr ?? 0;
  const shippingIrr = step >= 1 ? (selectedShipping?.priceIrr ?? 0) : 0;

  return (
    <main className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-4 pt-8 pb-10 sm:px-6">
      <motion.div {...revealProps}>
        <h1 className="text-2xl font-bold text-ink">{t('title')}</h1>
      </motion.div>

      <motion.div
        {...revealProps}
        transition={{ duration: 0.5, ease: 'easeOut', delay: 0.08 }}
      >
        <CheckoutStepper steps={steps} current={step} />
      </motion.div>

      <div className="grid gap-8 lg:grid-cols-[1fr_340px]">
        <motion.div {...revealProps}>
          <section className="min-h-[340px] rounded-2xl border border-line p-6">
            <AnimatePresence mode="wait">
              {/* Step 1 — Recipient */}
              {step === 0 && (
                <motion.div
                  key="recipient"
                  initial={{ opacity: 0, x: -24 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 24 }}
                  transition={{ duration: 0.3, ease: 'easeOut' }}
                  className="flex flex-col gap-5"
                >
                  <h2 className="text-base font-bold text-ink">
                    {t('address.title')}
                  </h2>

                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="flex flex-col gap-1">
                      <Input
                        label={t('address.firstName')}
                        value={recipient.firstName}
                        onChange={setField('firstName')}
                        placeholder={t('address.firstNamePlaceholder')}
                      />
                      {fieldError('recipient.firstName') && (
                        <p className="text-xs text-red-600">
                          {fieldError('recipient.firstName')}
                        </p>
                      )}
                    </div>
                    <div className="flex flex-col gap-1">
                      <Input
                        label={t('address.lastName')}
                        value={recipient.lastName}
                        onChange={setField('lastName')}
                        placeholder={t('address.lastNamePlaceholder')}
                      />
                      {fieldError('recipient.lastName') && (
                        <p className="text-xs text-red-600">
                          {fieldError('recipient.lastName')}
                        </p>
                      )}
                    </div>
                  </div>

                  <div className="flex flex-col gap-1">
                    <Input
                      label={t('address.phone')}
                      value={recipient.phone}
                      onChange={setField('phone')}
                      placeholder={t('address.phonePlaceholder')}
                    />
                    {fieldError('recipient.phone') && (
                      <p className="text-xs text-red-600">
                        {fieldError('recipient.phone')}
                      </p>
                    )}
                  </div>

                  <div className="flex flex-col gap-1">
                    <label className="text-xs font-medium text-coal">
                      {t('address.fullAddress')}
                    </label>
                    <textarea
                      value={recipient.address}
                      onChange={(event) => setField('address')(event.target.value)}
                      rows={3}
                      placeholder={t('address.fullAddressPlaceholder')}
                      className="
                        w-full resize-none rounded-lg border border-line bg-paper
                        px-3 py-3 text-sm text-ink outline-none
                        transition-all placeholder:text-ash
                        focus:border-ink focus:ring-2 focus:ring-ink/10
                      "
                    />
                    {fieldError('recipient.address') && (
                      <p className="text-xs text-red-600">
                        {fieldError('recipient.address')}
                      </p>
                    )}
                  </div>
                </motion.div>
              )}

              {/* Step 2 — Shipping */}
              {step === 1 && (
                <motion.div
                  key="shipping"
                  initial={{ opacity: 0, x: -24 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 24 }}
                  transition={{ duration: 0.3, ease: 'easeOut' }}
                  className="flex flex-col gap-5"
                >
                  <h2 className="text-base font-bold text-ink">
                    {t('shipping.title')}
                  </h2>

                  {shippingLoading && (
                    <div className="flex flex-col gap-3">
                      <SkeletonBlock className="h-20 rounded-xl" />
                      <SkeletonBlock className="h-20 rounded-xl" />
                    </div>
                  )}

                  {shippingError && (
                    <EmptyState
                      message={t('shippingError')}
                      description={shippingError.message}
                      action={
                        <button
                          type="button"
                          onClick={() => refetchShipping()}
                          className="rounded-lg border border-line px-4 py-2 text-sm font-medium text-ink transition-colors hover:border-ink/40"
                        >
                          {t('retry')}
                        </button>
                      }
                    />
                  )}

                  {!shippingLoading && !shippingError && (
                    <div className="flex flex-col gap-3">
                      {methods.map((method) => {
                        const isActive = method.id === shippingMethodId;

                        return (
                          <button
                            key={method.id}
                            type="button"
                            aria-pressed={isActive}
                            onClick={() => setShippingMethodId(method.id)}
                            className={`
                              flex items-center justify-between gap-4 rounded-xl border p-4 text-right
                              transition-all duration-200
                              ${
                                isActive
                                  ? 'border-ink bg-smoke/60'
                                  : 'border-line hover:border-ink/40'
                              }
                            `}
                          >
                            <span className="flex flex-col gap-1">
                              <span className="text-sm font-semibold text-ink">
                                {method.name}
                              </span>
                              <span className="text-xs text-ash">
                                {method.description}
                              </span>
                            </span>

                            <span className="flex items-center gap-3">
                              <span className="text-sm font-bold text-ink">
                                {method.priceIrr === 0
                                  ? tc('free')
                                  : `${formatToman(method.priceIrr, locale)} ${tc('toman')}`}
                              </span>
                              <span
                                className={`
                                  flex h-5 w-5 items-center justify-center rounded-full border-2
                                  ${isActive ? 'border-ink bg-ink' : 'border-line bg-paper'}
                                `}
                              >
                                {isActive && (
                                  <span className="h-2 w-2 rounded-full bg-brand" />
                                )}
                              </span>
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  )}

                  {fieldError('shippingMethodId') && (
                    <p className="text-xs text-red-600">
                      {fieldError('shippingMethodId')}
                    </p>
                  )}
                </motion.div>
              )}

              {/* Step 3 — Final confirmation */}
              {step === 2 && (
                <motion.div
                  key="confirmation"
                  initial={{ opacity: 0, x: -24 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 24 }}
                  transition={{ duration: 0.3, ease: 'easeOut' }}
                  className="flex flex-col gap-5"
                >
                  <h2 className="text-base font-bold text-ink">
                    {t('confirm.title')}
                  </h2>

                  <dl className="flex flex-col gap-3 rounded-xl bg-smoke/60 p-4 text-sm">
                    <div className="flex justify-between gap-4">
                      <dt className="text-ash">{t('confirm.recipient')}</dt>
                      <dd className="font-medium text-ink">
                        {recipient.firstName} {recipient.lastName}
                      </dd>
                    </div>
                    <div className="flex justify-between gap-4">
                      <dt className="text-ash">{t('address.phone')}</dt>
                      <dd dir="ltr" className="font-medium text-ink">
                        {recipient.phone}
                      </dd>
                    </div>
                    <div className="flex justify-between gap-4">
                      <dt className="text-ash">{t('address.fullAddress')}</dt>
                      <dd className="max-w-[60%] text-left font-medium text-ink" dir="auto">
                        {recipient.address}
                      </dd>
                    </div>
                    <div className="flex justify-between gap-4">
                      <dt className="text-ash">{t('shipping.title')}</dt>
                      <dd className="font-medium text-ink">
                        {selectedShipping?.name ?? '—'}
                      </dd>
                    </div>
                  </dl>

                  <p className="text-xs leading-6 text-ash">
                    {t('confirm.note')}
                  </p>
                </motion.div>
              )}
            </AnimatePresence>

            <div className="mt-8 flex items-center justify-between gap-3 border-t border-line pt-5">
              {step > 0 ? (
                <Button
                  variant="outline"
                  onClick={() => setStep((prev) => prev - 1)}
                  className="cursor-pointer"
                >
                  {t('back')}
                </Button>
              ) : (
                <span />
              )}

              {step < steps.length - 1 ? (
                <Button
                  variant="primary"
                  onClick={goNext}
                  className="cursor-pointer"
                >
                  {t('next')}
                </Button>
              ) : (
                <Button
                  variant="brand"
                  className="cursor-pointer"
                  disabled={checkoutPending}
                  onClick={() => void submit()}
                >
                  {checkoutPending ? t('payPending') : t('payAndSubmit')}
                </Button>
              )}
            </div>
          </section>
        </motion.div>

        <OrderSummary
          title={t('summaryTitle')}
          rows={[
            { label: t('itemsLabel'), value: String(cart?.itemCount ?? 0) },
            {
              label: t('subtotalLabel'),
              value: `${formatToman(subtotal, locale)} ${tc('toman')}`,
            },
            {
              label: t('shippingLabel'),
              value:
                step === 0
                  ? t('shippingAtNextStep')
                  : selectedShipping
                    ? selectedShipping.priceIrr === 0
                      ? tc('free')
                      : `${formatToman(selectedShipping.priceIrr, locale)} ${tc('toman')}`
                    : '—',
            },
          ]}
          totalLabel={t('totalLabel')}
          total={`${formatToman(subtotal + shippingIrr, locale)} ${tc('toman')}`}
        />
      </div>

      {checkoutError && checkoutError.validationDetails().length === 0 && (
        <div className="mx-auto w-full max-w-6xl">
          <p className="rounded-lg bg-smoke px-3 py-2 text-xs text-coal" role="alert">
            {checkoutError.message}
          </p>
        </div>
      )}
    </main>
  );
}
