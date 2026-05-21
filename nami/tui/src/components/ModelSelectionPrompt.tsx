import React, { type FC, useEffect, useMemo, useRef, useState } from "react";
import { Box, ModalDialog, PickerDialog, Text, useInput } from "silvery";
import type {
  UIModelSelection,
  UIModelSelectionOption,
} from "../hooks/useEvents.js";
import { stripProviderPrefix } from "../utils/formatModel.js";

interface PickerOptionItem {
  key: string;
  option: UIModelSelectionOption;
  section: string | null;
}

interface ModelSelectionPromptProps {
  selection: UIModelSelection;
  onSelect: (modelId: string, provider?: string) => void;
  onCancel: () => void;
}

const ModelSelectionPrompt: FC<ModelSelectionPromptProps> = ({
  selection,
  onSelect,
  onCancel,
}) => {
  const [terminalRows, setTerminalRows] = useState(process.stdout.rows ?? 24);
  const [searchValue, setSearchValue] = useState("");
  const [customMode, setCustomMode] = useState(false);
  const [customValue, setCustomValue] = useState("");
  const [cursorOffset, setCursorOffset] = useState(0);
  const customValueRef = useRef("");
  const cursorOffsetRef = useRef(0);
  const [cursorVisible, setCursorVisible] = useState(true);

  customValueRef.current = customValue;
  cursorOffsetRef.current = cursorOffset;

  useEffect(() => {
    setSearchValue("");
    setCustomMode(false);
    customValueRef.current = "";
    cursorOffsetRef.current = 0;
    setCustomValue("");
    setCursorOffset(0);
    setCursorVisible(true);
  }, [selection.requestId]);

  useEffect(() => {
    const handleResize = () => {
      setTerminalRows(process.stdout.rows ?? 24);
    };

    handleResize();
    process.stdout.on("resize", handleResize);

    return () => {
      process.stdout.off("resize", handleResize);
    };
  }, []);

  useEffect(() => {
    if (!customMode) {
      setCursorVisible(true);
      return;
    }

    const timer = setInterval(() => {
      setCursorVisible((current) => !current);
    }, 530);

    return () => {
      clearInterval(timer);
    };
  }, [customMode]);

  const kind: PickerKind =
    selection.title?.trim().toLowerCase() === "connect provider"
      ? "provider"
      : "model";
  const filteredOptions = useMemo(
    () => filterModelSelectionOptions(selection.options, searchValue),
    [searchValue, selection.options],
  );
  const pickerItems = useMemo<PickerOptionItem[]>(() => {
    let previousSection: string | null = null;

    return filteredOptions.map((option, index) => {
      const section = sectionForOption(option, kind);
      const showSection = section !== null && section !== previousSection;
      previousSection = section;

      return {
        key: `${option.label}-${index}`,
        option,
        section: showSection ? section : null,
      };
    });
  }, [filteredOptions, kind]);
  const dialogWidth = Math.max(
    44,
    Math.min(80, (process.stdout.columns ?? 80) - 4),
  );
  const maxVisible = Math.max(6, Math.min(18, terminalRows - 10));

  useInput(
    (input, key) => {
      if (!customMode) {
        return;
      }

      const text = key.text ?? input;
      const isEscape = key.escape || input === "\u001b" || text === "\u001b";

      if (isEscape) {
        setCustomMode(false);
        return;
      }
      if (key.return) {
        const value = customValueRef.current.trim();
        if (value.length > 0) {
          onSelect(value);
        }
        return;
      }
      if (key.leftArrow || (key.ctrl && input === "b")) {
        setCursorOffset((current) => Math.max(0, current - 1));
        return;
      }
      if (key.rightArrow || (key.ctrl && input === "f")) {
        setCursorOffset((current) =>
          Math.min(customValueRef.current.length, current + 1),
        );
        return;
      }
      if (key.home || (key.ctrl && input === "a")) {
        setCursorOffset(0);
        return;
      }
      if (key.end || (key.ctrl && input === "e")) {
        setCursorOffset(customValueRef.current.length);
        return;
      }
      if (key.backspace || (key.ctrl && input === "h")) {
        if (cursorOffsetRef.current === 0) {
          return;
        }
        setCustomValue((current) =>
          replaceRange(
            current,
            cursorOffsetRef.current - 1,
            cursorOffsetRef.current,
            "",
          ),
        );
        setCursorOffset((current) => Math.max(0, current - 1));
        return;
      }
      if (key.delete) {
        setCustomValue((current) =>
          replaceRange(
            current,
            cursorOffsetRef.current,
            cursorOffsetRef.current + 1,
            "",
          ),
        );
        return;
      }
      if (key.ctrl && input === "u") {
        setCustomValue("");
        setCursorOffset(0);
        return;
      }
      if (text && !key.ctrl && !key.meta) {
        setCustomValue((current) =>
          replaceRange(
            current,
            cursorOffsetRef.current,
            cursorOffsetRef.current,
            text,
          ),
        );
        setCursorOffset((current) => current + text.length);
      }
    },
    { isActive: customMode },
  );

  if (customMode) {
    return (
      <ModalDialog
        title="Custom model"
        width={Math.min(dialogWidth, 72)}
        footer="Enter apply · Esc return to list"
        borderColor="$inputborder"
      >
        <Box flexDirection="column" minWidth={0}>
          <Text color="$muted">
            Enter a model id or provider/model to pick a specific provider.
          </Text>
          <Box marginTop={1}>
            <Text>{renderEditableValue(customValue, cursorOffset, cursorVisible)}</Text>
          </Box>
        </Box>
      </ModalDialog>
    );
  }

  return (
    <PickerDialog
      key={selection.requestId}
      title={formatPickerTitle(selection.title)}
      items={pickerItems}
      renderItem={(item, selected) => (
        <Box
          flexDirection="column"
          minWidth={0}
          paddingX={1}
          backgroundColor={selected ? "$selectionbg" : undefined}
        >
          {item.section ? (
            <Text color="$primary" bold>
              {item.section}
            </Text>
          ) : null}
          <Text color={selected ? "$selection" : "$fg"} bold={selected}>
            {formatSelectionLine(item.option, selected, kind)}
          </Text>
        </Box>
      )}
      getKey={(item) => item.key}
      onSelect={(item) => {
        if (item.option.isCustom) {
          setCustomMode(true);
          setCursorOffset(customValueRef.current.length);
          return;
        }

        if (item.option.model) {
          onSelect(item.option.model, item.option.provider ?? undefined);
        }
      }}
      onCancel={onCancel}
      onChange={setSearchValue}
      placeholder="Search"
      emptyMessage="No options match the current filter."
      maxVisible={maxVisible}
      width={dialogWidth}
      footer="Enter choose · Esc cancel"
    />
  );
};

export default ModelSelectionPrompt;

type PickerKind = "provider" | "model";

function formatSelectionLine(
  option: UIModelSelectionOption,
  isSelected: boolean,
  kind: PickerKind,
): string {
  const prefix = option.active ? "✓" : isSelected ? "›" : " ";
  if (option.isCustom) {
    return `${prefix} Custom model`;
  }
  if (kind === "provider") {
    return `${prefix} ${providerDisplayName(option.provider ?? option.label)}`;
  }

  const model = displayModelName(option.model ?? option.label);
  const displayProvider = option.displayProvider ?? option.provider;
  const provider = displayProvider ? ` ${providerDisplayName(displayProvider)}` : "";
  return `${prefix} ${model}${provider}`;
}

function formatPickerTitle(title: string | undefined): string {
  const normalized = title?.trim().toLowerCase();
  if (normalized === "connect provider") {
    return "Connect a provider";
  }
  if (normalized === "select model") {
    return "Select model";
  }
  return title?.trim() || "Select model";
}

function sectionForOption(
  option: UIModelSelectionOption,
  kind: PickerKind,
): string | null {
  if (option.isCustom) {
    return "Custom";
  }
  if (kind === "provider") {
    return isPopularProvider(option.provider) ? "Popular" : "Providers";
  }
  if (option.active) {
    return "Recent";
  }
  return option.displayProvider || option.provider
    ? providerDisplayName(option.displayProvider ?? option.provider ?? "")
    : "Models";
}

function isPopularProvider(provider: string | null): boolean {
  switch ((provider ?? "").toLowerCase()) {
    case "github-copilot":
    case "codex":
    case "openai":
    case "anthropic":
    case "gemini":
      return true;
    default:
      return false;
  }
}

function providerDisplayName(provider: string): string {
  switch (provider.toLowerCase()) {
    case "github-copilot":
      return "GitHub Copilot";
    case "openai":
      return "OpenAI";
    case "anthropic":
      return "Anthropic";
    case "gemini":
      return "Google";
    case "deepseek":
      return "DeepSeek";
    case "ollama":
      return "Ollama";
    case "codex":
      return "OpenCode Zen";
    default:
      return provider
        .split("-")
        .filter(Boolean)
        .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
        .join(" ");
  }
}

function displayModelName(model: string): string {
  const withoutProvider = stripProviderPrefix(model) ?? model;
  return withoutProvider
    .split("-")
    .map((part) => {
      if (/^gpt$/i.test(part)) {
        return "GPT";
      }
      if (/^\d/.test(part)) {
        return part;
      }
      return part.slice(0, 1).toUpperCase() + part.slice(1);
    })
    .join("-");
}

function filterModelSelectionOptions(
  options: UIModelSelectionOption[],
  query: string,
): UIModelSelectionOption[] {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) {
    return options;
  }

  return options.filter((option) => {
    const haystack = [
      option.label,
      option.model,
      option.provider,
      option.displayProvider,
      option.description,
      option.isCustom ? "custom model" : null,
    ]
      .filter((value): value is string => typeof value === "string")
      .join(" ")
      .toLowerCase();

    return haystack.includes(normalizedQuery);
  });
}

function renderEditableValue(
  value: string,
  cursorOffset: number,
  cursorVisible = true,
): string {
  const clampedOffset = Math.max(0, Math.min(value.length, cursorOffset));
  const cursor = cursorVisible ? "█" : " ";
  return value.slice(0, clampedOffset) + cursor + value.slice(clampedOffset);
}

function replaceRange(
  value: string,
  start: number,
  end: number,
  replacement: string,
): string {
  const safeStart = Math.max(0, Math.min(value.length, start));
  const safeEnd = Math.max(safeStart, Math.min(value.length, end));
  return value.slice(0, safeStart) + replacement + value.slice(safeEnd);
}
