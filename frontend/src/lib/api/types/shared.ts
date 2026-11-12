import { z } from 'zod';

export const ApiListResponseSchema = <T extends z.ZodTypeAny>(itemSchema: T) =>
  z.object({
    items: z.array(itemSchema),
  });

export type ApiListResponse<T> = {
  items: T[];
};

export type ApiError = {
  status: number;
  message: string;
  data?: unknown;
};
