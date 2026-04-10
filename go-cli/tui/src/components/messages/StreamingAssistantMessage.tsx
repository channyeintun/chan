import React, { type FC } from "react";
import { Text } from "ink";
import Spinner from "ink-spinner";
import MessageRow from "../MessageRow.js";
import MarkdownText from "../MarkdownText.js";

interface StreamingAssistantMessageProps {
  text?: string;
  statusLabel: string;
  model?: string;
}

const StreamingAssistantMessage: FC<StreamingAssistantMessageProps> = ({
  text,
  statusLabel,
  model,
}) => {
  return (
    <MessageRow
      markerColor="green"
      markerDim
      label={
        <Text color="green" dimColor>
          Assistant
        </Text>
      }
      meta={model ? <Text dimColor>{model}</Text> : null}
    >
      <Text color="gray">
        <Spinner type="dots" /> {statusLabel}
      </Text>
      {text ? <MarkdownText text={text} streaming /> : null}
    </MessageRow>
  );
};

export default StreamingAssistantMessage;
