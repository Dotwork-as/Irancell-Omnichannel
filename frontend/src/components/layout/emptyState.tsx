type Props = {
  message: string;
  description?: string;
  action?: React.ReactNode;
};

export default function EmptyState({ message, description, action }: Props) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-10 text-center">
      <p className="text-sm font-medium text-coal">{message}</p>

      {description && <p className="text-xs text-ash">{description}</p>}

      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}
