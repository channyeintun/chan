import React, { type FC, useMemo } from "react";
import { Text } from "ink";
import Spinner from "ink-spinner";
import MessageRow from "../MessageRow.js";

interface AssistantThinkingMessageProps {
  text: string;
  model?: string;
}

function truncateThinking(text: string): string {
  const lines = text.split("\n").filter((line) => line.trim().length > 0);
  return lines.slice(-4).join("\n");
}

const AssistantThinkingMessage: FC<AssistantThinkingMessageProps> = ({
  text,
  model,
}) => {
  const preview = useMemo(() => truncateThinking(text), [text]);
  if (!preview) {
    return null;
  }

  return (
    <MessageRow
      markerColor="gray"
      markerDim
      label={
        <Text color="gray" dimColor>
          Assistant
        </Text>
      }
      meta={model ? <Text dimColor>{model}</Text> : null}
    >
      <Text color="gray" italic>
        <Spinner type="dots" /> Thinking
      </Text>
      <Text color="gray">{preview}</Text>
    </MessageRow>
  );
};

export default AssistantThinkingMessage;
