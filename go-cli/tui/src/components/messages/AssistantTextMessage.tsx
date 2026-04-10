import React, { type FC } from "react";
import { Text } from "ink";
import type { UIMessage } from "../../hooks/useEvents.js";
import MessageRow from "../MessageRow.js";
import MarkdownText from "../MarkdownText.js";

interface AssistantTextMessageProps {
  message: UIMessage;
}

const AssistantTextMessage: FC<AssistantTextMessageProps> = ({ message }) => {
  return (
    <MessageRow markerColor="green">
      <Text color="green" bold>
        Assistant
      </Text>
      <MarkdownText text={message.text} />
    </MessageRow>
  );
};

export default AssistantTextMessage;
