import { createSlice, PayloadAction } from '@reduxjs/toolkit';

export type PreviewMode =
  | 'data'
  | 'loading'
  | 'error';

type PreviewState = {
  mode: PreviewMode;
};

const initialState: PreviewState = {
  mode: 'data',
};

const previewSlice = createSlice({
  name: 'preview',

  initialState,

  reducers: {
    setMode: (
      state,
      action: PayloadAction<PreviewMode>,
    ) => {
      state.mode = action.payload;
    },
  },
});

export const { setMode } =
  previewSlice.actions;

export default previewSlice.reducer;