import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';

import {
  addToCart,
  getCart,
  removeCartItem,
  updateCartItem,
} from '@/src/lib/api/services/cart';
import { normalizeApiError } from '@/src/lib/api/error';
import type { Cart } from '@/src/lib/api/types/schemas';

/** Plain-object mutation failure (serializable in Redux). */
export interface CartMutationError {
  status: number;
  code: string;
  message: string;
  details?: unknown;
}

interface CartState {
  cart: Cart | null;
  /** fetch lifecycle; mutations track mutationPending separately. */
  status: 'idle' | 'loading' | 'ready';
  mutationPending: boolean;
  /** GET /cart failure — page-level error. */
  error: CartMutationError | null;
  /** add/update/remove failure — shown inline, cart stays visible. */
  mutationError: CartMutationError | null;
}

const initialState: CartState = {
  cart: null,
  status: 'idle',
  mutationPending: false,
  error: null,
  mutationError: null,
};

/**
 * Fetches the guest cart (empty cart when no valid session — the
 * backend never 401s, §4.5).
 */
export const fetchCart = createAsyncThunk<Cart>(
  'cart/fetch',
  async () => getCart(),
);

/**
 * Every mutation returns the backend's recomputed cart and stores it
 * verbatim — totals, stock bounds and quantities are always
 * server-driven (§4.4), never merged client-side.
 */
export const submitAddToCart = createAsyncThunk<
  Cart,
  { variantId: number; quantity: number },
  { rejectValue: CartMutationError }
>('cart/add', async ({ variantId, quantity }, { rejectWithValue }) => {
  try {
    return await addToCart({ variantId, quantity });
  } catch (error) {
    return rejectWithValue(mutationErrorOf(error));
  }
});

export const submitUpdateCartItem = createAsyncThunk<
  Cart,
  { lineId: number; quantity: number },
  { rejectValue: CartMutationError }
>('cart/update', async ({ lineId, quantity }, { rejectWithValue }) => {
  try {
    return await updateCartItem(lineId, { quantity });
  } catch (error) {
    return rejectWithValue(mutationErrorOf(error));
  }
});

export const submitRemoveCartItem = createAsyncThunk<
  Cart,
  { lineId: number },
  { rejectValue: CartMutationError }
>('cart/remove', async ({ lineId }, { rejectWithValue }) => {
  try {
    return await removeCartItem(lineId);
  } catch (error) {
    return rejectWithValue(mutationErrorOf(error));
  }
})

const cartSlice = createSlice({
  name: 'cart',

  initialState,

  reducers: {
    /** Resets to a pristine empty state (e.g. after a paid checkout). */
    clearCart(state) {
      state.cart = { items: [], subtotalIrr: 0, itemCount: 0, currency: 'IRR' };
      state.error = null;
    },
  },

  extraReducers: (builder) => {
    builder
      .addCase(fetchCart.pending, (state) => {
        state.status = 'loading';
        state.error = null;
      })
      .addCase(fetchCart.fulfilled, (state, action) => {
        state.status = 'ready';
        state.cart = action.payload;
      })
      .addCase(fetchCart.rejected, (state, action) => {
        state.status = 'ready';
        state.error = action.error
          ? { status: 0, code: 'FETCH_FAILED', message: action.error.message ?? '' }
          : null;
      })
      .addCase(submitAddToCart.pending, (state) => {
        state.mutationPending = true;
        state.mutationError = null;
      })
      .addCase(submitAddToCart.fulfilled, (state, action) => {
        state.mutationPending = false;
        state.status = 'ready';
        state.cart = action.payload;
      })
      .addCase(submitAddToCart.rejected, (state, action) => {
        state.mutationPending = false;
        if (action.payload) state.mutationError = action.payload;
      })
      .addCase(submitUpdateCartItem.pending, (state) => {
        state.mutationPending = true;
        state.mutationError = null;
      })
      .addCase(submitUpdateCartItem.fulfilled, (state, action) => {
        state.mutationPending = false;
        state.cart = action.payload;
      })
      .addCase(submitUpdateCartItem.rejected, (state, action) => {
        state.mutationPending = false;
        if (action.payload) state.mutationError = action.payload;
      })
      .addCase(submitRemoveCartItem.pending, (state) => {
        state.mutationPending = true;
        state.mutationError = null;
      })
      .addCase(submitRemoveCartItem.fulfilled, (state, action) => {
        state.mutationPending = false;
        state.cart = action.payload;
      })
      .addCase(submitRemoveCartItem.rejected, (state, action) => {
        state.mutationPending = false;
        if (action.payload) state.mutationError = action.payload;
      });
  },
});

function mutationErrorOf(error: unknown): CartMutationError {
  // Services already normalize, but this guards any raw rejection too.
  const apiError = normalizeApiError(error);
  return {
    status: apiError.status,
    code: apiError.code,
    message: apiError.message,
    details: apiError.details,
  };
}

export const { clearCart } = cartSlice.actions;
export default cartSlice.reducer;
