type Props = {
  children: React.ReactNode;
};

export default function PageContainer({
  children,
}: Props) {
  return (
    <main className="min-h-screen bg-paper text-ink">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 pt-8 pb-10 sm:px-6">
        {children}
      </div>
    </main>
  );
}
