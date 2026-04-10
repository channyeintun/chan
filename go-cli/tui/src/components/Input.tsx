import React, { type FC } from "react";
import { Box, Text, useInput } from "ink";

interface InputProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  onHistoryUp: () => void;
  onHistoryDown: () => void;
  onModeToggle: () => void;
  onCancel: () => void;
  disabled?: boolean;
}

const INPUT_HINT = "Enter send | Up/Down history | Tab mode | Esc cancel";
const DISABLED_HINT = "Engine busy | Esc cancel";

const Input: FC<InputProps> = ({
  value,
  onChange,
  onSubmit,
  onHistoryUp,
  onHistoryDown,
  onModeToggle,
  onCancel,
  disabled,
}) => {
  useInput((input, key) => {
    if (key.escape) {
      onCancel();
      return;
    }
    if (disabled) return;

    if (key.tab) {
      onModeToggle();
      return;
    }
    if (key.upArrow) {
      onHistoryUp();
      return;
    }
    if (key.downArrow) {
      onHistoryDown();
      return;
    }
    if (key.return) {
      onSubmit();
      return;
    }
    if (key.backspace || key.delete) {
      onChange(value.slice(0, -1));
      return;
    }
    if (input) {
      onChange(value + input);
    }
  });

  const showPlaceholder = value.length === 0;
  const hint = disabled ? DISABLED_HINT : INPUT_HINT;

  return (
    <Box flexDirection="column">
      <Box>
        <Text color="cyan" bold>
          {"❯ "}
        </Text>
        {showPlaceholder ? (
          <Text color="gray">Ask go-cli to inspect, plan, or edit code</Text>
        ) : (
          <Text>{value}</Text>
        )}
        <Text color="gray">{"█"}</Text>
      </Box>
      <Box paddingLeft={2}>
        <Text dimColor>{hint}</Text>
      </Box>
    </Box>
  );
};

export default Input;
