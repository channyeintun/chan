import React, { type FC, useEffect, useMemo, useState } from "react";
import { Box, PickerDialog, Text } from "silvery";
import type { UIResumeSelection } from "../hooks/useEvents.js";
import { stripProviderPrefix } from "../utils/formatModel.js";

interface ResumeSelectionPromptProps {
  selection: UIResumeSelection;
  onSelect: (sessionId: string) => void;
  onCancel: () => void;
}

const ResumeSelectionPrompt: FC<ResumeSelectionPromptProps> = ({
  selection,
  onSelect,
  onCancel,
}) => {
  const [searchValue, setSearchValue] = useState("");
  const [terminalRows, setTerminalRows] = useState(process.stdout.rows ?? 24);
  const [terminalColumns, setTerminalColumns] = useState(
    process.stdout.columns ?? 80,
  );
  const filteredSessions = useMemo(
    () => filterResumeSessions(selection.sessions, searchValue),
    [searchValue, selection.sessions],
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
    terminalColumns > 52
      ? Math.min(96, terminalColumns - 4)
      : Math.max(24, terminalColumns - 2);
  const maxVisible =
    terminalRows > 14
      ? Math.min(12, terminalRows - 8)
      : Math.max(4, terminalRows - 4);

  return (
    <PickerDialog
      key={selection.requestId}
      title="Resume Session"
      items={filteredSessions}
      renderItem={(session, selected) => {
        const timestamp = formatUpdatedAt(session.updatedAt);

        return (
          <Box
            flexDirection="column"
            minWidth={0}
            paddingX={1}
            backgroundColor={selected ? "$selectionbg" : undefined}
          >
            <Text color={selected ? "$selection" : "$fg"} bold={selected}>
              {selected ? "›" : " "} {session.sessionId.slice(0, 8)} {timestamp}
            </Text>
            <Text color={selected ? "$selection" : "$muted"} wrap="wrap">
              {session.title}
              {session.model
                ? `  ·  ${stripProviderPrefix(session.model) ?? session.model}`
                : ""}
              {`  ·  $${session.totalCostUsd.toFixed(4)}`}
            </Text>
          </Box>
        );
      }}
      getKey={(session) => session.sessionId}
      onSelect={(session) => onSelect(session.sessionId)}
      onCancel={onCancel}
      onChange={setSearchValue}
      placeholder="Search"
      emptyMessage="No sessions match the current filter."
      maxVisible={maxVisible}
      width={dialogWidth}
      footer={`${selection.sessions.length} available · Enter resume · Esc cancel`}
    />
  );
};

export default ResumeSelectionPrompt;

function formatUpdatedAt(value: string | null): string {
  if (!value) {
    return "unknown time";
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }

  const year = parsed.getFullYear();
  const month = String(parsed.getMonth() + 1).padStart(2, "0");
  const day = String(parsed.getDate()).padStart(2, "0");
  const hours = String(parsed.getHours()).padStart(2, "0");
  const minutes = String(parsed.getMinutes()).padStart(2, "0");
  return `${year}-${month}-${day} ${hours}:${minutes}`;
}

function filterResumeSessions(
  sessions: UIResumeSelection["sessions"],
  query: string,
): UIResumeSelection["sessions"] {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) {
    return sessions;
  }

  return sessions.filter((session) => {
    const haystack = [
      session.sessionId,
      session.title,
      session.updatedAt,
      session.model,
      session.totalCostUsd.toFixed(4),
    ]
      .filter((value): value is string => typeof value === "string")
      .join(" ")
      .toLowerCase();

    return haystack.includes(normalizedQuery);
  });
}
