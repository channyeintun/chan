import React, { type FC } from "react";
import { Text } from "ink";
import type { UIMessage } from "../../hooks/useEvents.js";
import MessageRow from "../MessageRow.js";
import MarkdownText from "../MarkdownText.js";

interface AssistantTextMessageProps {
  message: UIMessage;
  continuation?: boolean;
}

const AssistantTextMessage: FC<AssistantTextMessageProps> = ({
  message,
  continuation = false,
}) => {
  return (
    <MessageRow
      markerColor="green"
      label={
        continuation ? null : (
          <Text color="green" bold>
            Assistant
          </Text>
        )
      }
      meta={renderMetadata(message)}
    >
      <MarkdownText text={message.text} />
    </MessageRow>
  );
};

export default AssistantTextMessage;

function renderMetadata(message: UIMessage) {
  const parts: string[] = [];

  if (message.timestamp) {
    parts.push(
      new Date(message.timestamp).toLocaleTimeString("en-US", {
        hour: "2-digit",
        minute: "2-digit",
        hour12: true,
      }),
    );
  }

  if (message.model) {
    parts.push(message.model);
  }

  if (parts.length === 0) {
    return null;
  }

  return <Text dimColor>{parts.join("  ")}</Text>;
}
