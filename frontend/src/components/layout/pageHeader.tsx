type Props = {
  title: string;
  description?: string;
  action?: React.ReactNode;
};

export default function PageHeader({ title, description, action }: Props) {
  return (
    <header className="flex items-center justify-between gap-4">
      <div>
        <h1 className="text-2xl font-bold text-ink">{title}</h1>

        {description && <p className="mt-1 text-sm text-ash">{description}</p>}
      </div>

      {action}
    </header>
  );
}
