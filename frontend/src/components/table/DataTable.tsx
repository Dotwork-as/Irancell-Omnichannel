import { ReactNode } from 'react';

type Column<T> = {
  key: keyof T | string;
  title: string;
  render?: (item: T) => ReactNode;
  mobileHidden?: boolean;
};

type Props<T> = {
  data: T[];
  columns: Column<T>[];

  page: number;
  totalPages: number;

  onPrev: () => void;
  onNext: () => void;

  emptyMessage?: string;
};

export default function DataTable<T>({
  data,
  columns,

  page,
  totalPages,

  onPrev,
  onNext,

  emptyMessage = 'No data found',
}: Props<T>) {
  if (data.length === 0) {
    return (
      <div className="py-8 text-center text-sm text-ash">
        {emptyMessage}
      </div>
    );
  }

  return (
    <div className="w-full">
      <div className="overflow-x-auto custom-scrollbar">
        <table className="min-w-[700px] w-full border-collapse text-sm">
          <thead>
            <tr className="border-b border-line text-start text-ash">
              {columns.map((column) => (
                <th
                  key={String(column.key)}
                  className={`
                    py-3 font-medium whitespace-nowrap
                    ${column.mobileHidden ? 'hidden md:table-cell' : ''}
                  `}
                >
                  {column.title}
                </th>
              ))}
            </tr>
          </thead>

          <tbody>
            {data.map((item, index) => (
              <tr key={index} className="border-b border-line/60">
                {columns.map((column) => (
                  <td
                    key={String(column.key)}
                    className={`
                      py-4 pr-4 whitespace-nowrap
                      ${column.mobileHidden ? 'hidden md:table-cell' : ''}
                    `}
                  >
                    {column.render
                      ? column.render(item)
                      : String(item[column.key as keyof T])}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>

        {/* PAGINATION */}
      </div>
      <div className="mt-6 flex items-center justify-end gap-2">
        <button
          onClick={onPrev}
          disabled={page === 1}
          className="rounded-lg border border-line px-3 py-1 text-sm text-coal disabled:opacity-40"
        >
          Prev
        </button>

        <span className="text-sm text-ash">
          {page} / {totalPages}
        </span>

        <button
          onClick={onNext}
          disabled={page === totalPages}
          className="rounded-lg border border-line px-3 py-1 text-sm text-coal disabled:opacity-40"
        >
          Next
        </button>
      </div>
    </div>
  );
}
