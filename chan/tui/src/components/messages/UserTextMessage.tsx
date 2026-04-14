import React, { type FC } from "react";
import { Text } from "silvery";
import { DEFAULT_PROMPT_MARKER } from "../../constants/prompt.js";
import type { UIUserMessage } from "../../hooks/useEvents.js";
import MessageRow from "../MessageRow.js";

interface UserTextMessageProps {
  message: UIUserMessage;
  continuation?: boolean;
}

const UserTextMessage: FC<UserTextMessageProps> = ({
  message,
  continuation = false,
}) => {
  return (
    <MessageRow
      marker={DEFAULT_PROMPT_MARKER.trimEnd()}
      markerColor="$primary"
      label={null}
    >
      <Text>{message.text}</Text>
    </MessageRow>
  );
};

export default UserTextMessage;
