import React, { type FC } from "react";
import { Text } from "ink";
import Spinner from "ink-spinner";
import MessageRow from "../MessageRow.js";
import MarkdownText from "../MarkdownText.js";

interface StreamingAssistantMessageProps {
  text?: string;
  statusLabel: string;
}

const StreamingAssistantMessage: FC<StreamingAssistantMessageProps> = ({
  text,
  statusLabel,
}) => {
  return (
    <MessageRow markerColor="green" markerDim>
      <Text color="green" bold>
        Assistant
      </Text>
      <Text color="gray">
        <Spinner type="dots" /> {statusLabel}
      </Text>
      {text ? <MarkdownText text={text} streaming /> : null}
    </MessageRow>
  );
};

export default StreamingAssistantMessage;
