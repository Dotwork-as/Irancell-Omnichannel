import SiteFooter from '@/src/components/layout/siteFooter';
import SiteHeader from '@/src/components/layout/siteHeader';

import StoreProvider from '@/src/store/provider';

export default function MainLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <StoreProvider>
      <div className="flex min-h-screen flex-col bg-paper text-ink">
        <SiteHeader />
        <div className="flex-1">{children}</div>
        <SiteFooter />
      </div>
    </StoreProvider>
  );
}
