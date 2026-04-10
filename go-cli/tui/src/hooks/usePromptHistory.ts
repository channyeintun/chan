import { useCallback, useState } from "react";

interface PromptHistoryState {
  value: string;
  entries: string[];
  index: number;
  draft: string;
}

const MAX_HISTORY_ENTRIES = 50;

const initialState: PromptHistoryState = {
  value: "",
  entries: [],
  index: 0,
  draft: "",
};

export function usePromptHistory() {
  const [state, setState] = useState<PromptHistoryState>(initialState);

  const setValue = useCallback((value: string) => {
    setState((current) => ({
      ...current,
      value,
      index: 0,
      draft: "",
    }));
  }, []);

  const submit = useCallback((): string => {
    let submitted = "";

    setState((current) => {
      const nextValue = current.value.trim();
      if (!nextValue) {
        return current;
      }

      submitted = nextValue;
      return {
        value: "",
        entries: [
          nextValue,
          ...current.entries.filter((entry) => entry !== nextValue),
        ].slice(0, MAX_HISTORY_ENTRIES),
        index: 0,
        draft: "",
      };
    });

    return submitted;
  }, []);

  const navigateUp = useCallback(() => {
    setState((current) => {
      if (
        current.entries.length === 0 ||
        current.index >= current.entries.length
      ) {
        return current;
      }

      const nextIndex = current.index + 1;
      return {
        ...current,
        index: nextIndex,
        draft: current.index === 0 ? current.value : current.draft,
        value: current.entries[nextIndex - 1] ?? current.value,
      };
    });
  }, []);

  const navigateDown = useCallback(() => {
    setState((current) => {
      if (current.index === 0) {
        return current;
      }

      if (current.index === 1) {
        return {
          ...current,
          index: 0,
          value: current.draft,
          draft: "",
        };
      }

      const nextIndex = current.index - 1;
      return {
        ...current,
        index: nextIndex,
        value: current.entries[nextIndex - 1] ?? "",
      };
    });
  }, []);

  return {
    value: state.value,
    setValue,
    submit,
    navigateUp,
    navigateDown,
  };
}
