import { configureStore } from '@reduxjs/toolkit';

import cartReducer from './cart.slice';
import previewReducer from './preview/preview.slice';

/** Factory so tests can create isolated stores. */
export const makeStore = () =>
  configureStore({
    reducer: {
      preview: previewReducer,
      cart: cartReducer,
    },
  });

/** Application singleton. */
export const store = makeStore();

export type RootState = ReturnType<
  typeof store.getState
>;

export type AppDispatch =
  typeof store.dispatch;
