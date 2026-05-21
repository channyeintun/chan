import React, { type FC, useEffect, useMemo, useState } from "react";
import { Box, PickerDialog, Text } from "silvery";
import type { UIReasoningSelection } from "../hooks/useEvents.js";

interface ReasoningSelectionPromptProps {
  selection: UIReasoningSelection;
  onSelect: (effort: string) => void;
  onCancel: () => void;
}

const ReasoningSelectionPrompt: FC<ReasoningSelectionPromptProps> = ({
  selection,
  onSelect,
  onCancel,
}) => {
  const [searchValue, setSearchValue] = useState("");
  const [terminalRows, setTerminalRows] = useState(process.stdout.rows ?? 24);
  const [terminalColumns, setTerminalColumns] = useState(
    process.stdout.columns ?? 80,
  );
  const filteredOptions = useMemo(
    () => filterReasoningSelectionOptions(selection.options, searchValue),
    [searchValue, selection.options],
  );

  useEffect(() => {
    setSearchValue("");
  }, [selection.requestId]);

  useEffect(() => {
    const handleResize = () => {
      setTerminalRows(process.stdout.rows ?? 24);
      setTerminalColumns(process.stdout.columns ?? 80);
    };

    handleResize();
    process.stdout.on("resize", handleResize);

    return () => {
      process.stdout.off("resize", handleResize);
    };
  }, []);

  const dialogWidth =
    terminalColumns > 48
      ? Math.min(76, terminalColumns - 4)
      : Math.max(24, terminalColumns - 2);
  const maxVisible =
    terminalRows > 14
      ? Math.min(8, terminalRows - 8)
      : Math.max(4, terminalRows - 4);
  const footer = [selection.description, "Enter apply", "Esc cancel"]
    .filter((value): value is string => typeof value === "string" && value.length > 0)
    .join(" · ");

  return (
    <PickerDialog
      key={selection.requestId}
      title={selection.title?.trim() || "Select reasoning"}
      items={filteredOptions}
      renderItem={(option, selected) => (
        <Box
          flexDirection="column"
          minWidth={0}
          paddingX={1}
          backgroundColor={selected ? "$selectionbg" : undefined}
        >
          <Text color={selected ? "$selection" : "$fg"} bold={selected}>
            {option.active ? "✓" : selected ? "›" : " "} {option.label}
          </Text>
          {option.description ? (
            <Text color={selected ? "$selection" : "$muted"} wrap="wrap">
              {option.description}
            </Text>
          ) : null}
        </Box>
      )}
      getKey={(option) => option.value}
      onSelect={(option) => onSelect(option.value)}
      onCancel={onCancel}
      onChange={setSearchValue}
      placeholder="Search"
      emptyMessage="No reasoning options match the current filter."
      maxVisible={maxVisible}
      width={dialogWidth}
      footer={footer}
    />
  );
};

export default ReasoningSelectionPrompt;

function filterReasoningSelectionOptions(
  options: UIReasoningSelection["options"],
  query: string,
): UIReasoningSelection["options"] {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) {
    return options;
  }

  return options.filter((option) => {
    const haystack = [option.value, option.label, option.description]
      .filter((value): value is string => typeof value === "string")
      .join(" ")
      .toLowerCase();

    return haystack.includes(normalizedQuery);
  });
}