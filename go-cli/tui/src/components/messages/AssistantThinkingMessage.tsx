import React, { type FC, useMemo } from "react";
import { Text } from "ink";
import Spinner from "ink-spinner";
import MessageRow from "../MessageRow.js";

interface AssistantThinkingMessageProps {
  text: string;
}

function truncateThinking(text: string): string {
  const lines = text.split("\n").filter((line) => line.trim().length > 0);
  return lines.slice(-4).join("\n");
}

const AssistantThinkingMessage: FC<AssistantThinkingMessageProps> = ({
  text,
}) => {
  const preview = useMemo(() => truncateThinking(text), [text]);
  if (!preview) {
    return null;
  }

  return (
    <MessageRow markerColor="gray" markerDim>
      <Text color="gray" italic>
        <Spinner type="dots" /> Thinking
      </Text>
      <Text color="gray">{preview}</Text>
    </MessageRow>
  );
};

export default AssistantThinkingMessage;
